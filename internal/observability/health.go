package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthCheck represents a single health check function
type HealthCheck func(ctx context.Context) error

// ComponentHealth represents the health status of a single component
type ComponentHealth struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration_ms"`
}

// Health represents the overall health status of the application
type Health struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
	Version    string                     `json:"version"`
	Uptime     time.Duration              `json:"uptime_seconds"`
}

// HealthChecker manages health checks for various components
type HealthChecker struct {
	mu        sync.RWMutex
	checks    map[string]HealthCheck
	cache     map[string]ComponentHealth
	cacheTTL  time.Duration
	logger    zerolog.Logger
	startTime time.Time
	version   string
}

// NewHealthChecker creates a new health checker
// Complexity: O(1)
func NewHealthChecker(logger zerolog.Logger, version string) *HealthChecker {
	return &HealthChecker{
		checks:    make(map[string]HealthCheck),
		cache:     make(map[string]ComponentHealth),
		cacheTTL:  10 * time.Second,
		logger:    logger,
		startTime: time.Now(),
		version:   version,
	}
}

// RegisterCheck registers a health check for a component
// Complexity: O(1)
func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = check
	hc.logger.Info().
		Str("component", name).
		Msg("health check registered")
}

// UnregisterCheck removes a health check
// Complexity: O(1)
func (hc *HealthChecker) UnregisterCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.checks, name)
	delete(hc.cache, name)
	hc.logger.Info().
		Str("component", name).
		Msg("health check unregistered")
}

// Check runs all registered health checks and returns the overall health status
// Complexity: O(n) where n is the number of registered checks
func (hc *HealthChecker) Check(ctx context.Context) *Health {
	hc.mu.RLock()
	checks := make(map[string]HealthCheck, len(hc.checks))
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.mu.RUnlock()

	components := make(map[string]ComponentHealth)
	overallStatus := HealthStatusHealthy

	// Run all health checks concurrently
	var wg sync.WaitGroup
	resultsChan := make(chan ComponentHealth, len(checks))

	for name, check := range checks {
		wg.Add(1)
		go func(name string, check HealthCheck) {
			defer wg.Done()
			health := hc.runCheck(ctx, name, check)
			resultsChan <- health
		}(name, check)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for health := range resultsChan {
		components[health.Name] = health

		// Determine overall status (worst case wins)
		if health.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if health.Status == HealthStatusDegraded && overallStatus != HealthStatusUnhealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	// If no checks are registered, status is unknown
	if len(components) == 0 {
		overallStatus = HealthStatusUnknown
	}

	return &Health{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Components: components,
		Version:    hc.version,
		Uptime:     time.Since(hc.startTime),
	}
}

// runCheck executes a single health check with timeout and error handling
func (hc *HealthChecker) runCheck(ctx context.Context, name string, check HealthCheck) ComponentHealth {
	startTime := time.Now()

	// Check cache first
	if cached, ok := hc.getCachedHealth(name); ok {
		return cached
	}

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Run the check
	err := check(checkCtx)
	duration := time.Since(startTime)

	health := ComponentHealth{
		Name:      name,
		Timestamp: time.Now(),
		Duration:  duration,
	}

	if err != nil {
		health.Status = HealthStatusUnhealthy
		health.Error = err.Error()
		hc.logger.Warn().
			Str("component", name).
			Err(err).
			Dur("duration_ms", duration).
			Msg("health check failed")
	} else {
		health.Status = HealthStatusHealthy
		health.Message = "OK"
		hc.logger.Debug().
			Str("component", name).
			Dur("duration_ms", duration).
			Msg("health check passed")
	}

	// Cache the result
	hc.cacheHealth(name, health)

	return health
}

// getCachedHealth retrieves cached health status if still valid
func (hc *HealthChecker) getCachedHealth(name string) (ComponentHealth, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	cached, exists := hc.cache[name]
	if !exists {
		return ComponentHealth{}, false
	}

	// Check if cache is still valid
	if time.Since(cached.Timestamp) > hc.cacheTTL {
		return ComponentHealth{}, false
	}

	return cached, true
}

// cacheHealth stores health status in cache
func (hc *HealthChecker) cacheHealth(name string, health ComponentHealth) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.cache[name] = health
}

// IsHealthy returns true if the overall status is healthy
func (h *Health) IsHealthy() bool {
	return h.Status == HealthStatusHealthy
}

// IsDegraded returns true if the overall status is degraded
func (h *Health) IsDegraded() bool {
	return h.Status == HealthStatusDegraded
}

// IsUnhealthy returns true if the overall status is unhealthy
func (h *Health) IsUnhealthy() bool {
	return h.Status == HealthStatusUnhealthy
}

// GetUnhealthyComponents returns a list of unhealthy components
func (h *Health) GetUnhealthyComponents() []string {
	var unhealthy []string
	for name, component := range h.Components {
		if component.Status == HealthStatusUnhealthy {
			unhealthy = append(unhealthy, name)
		}
	}
	return unhealthy
}

// GetDegradedComponents returns a list of degraded components
func (h *Health) GetDegradedComponents() []string {
	var degraded []string
	for name, component := range h.Components {
		if component.Status == HealthStatusDegraded {
			degraded = append(degraded, name)
		}
	}
	return degraded
}

// Common health check functions

// DatabaseHealthCheck creates a health check for a database connection
func DatabaseHealthCheck(pingFunc func(ctx context.Context) error) HealthCheck {
	return func(ctx context.Context) error {
		if err := pingFunc(ctx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}
		return nil
	}
}

// RedisHealthCheck creates a health check for a Redis connection
func RedisHealthCheck(pingFunc func(ctx context.Context) error) HealthCheck {
	return func(ctx context.Context) error {
		if err := pingFunc(ctx); err != nil {
			return fmt.Errorf("redis ping failed: %w", err)
		}
		return nil
	}
}

// P2PHostHealthCheck creates a health check for P2P host status
func P2PHostHealthCheck(statusFunc func() error) HealthCheck {
	return func(ctx context.Context) error {
		if err := statusFunc(); err != nil {
			return fmt.Errorf("p2p host unhealthy: %w", err)
		}
		return nil
	}
}

// WebSocketHealthCheck creates a health check for WebSocket connection
func WebSocketHealthCheck(isConnectedFunc func() bool) HealthCheck {
	return func(ctx context.Context) error {
		if !isConnectedFunc() {
			return fmt.Errorf("websocket not connected")
		}
		return nil
	}
}

// VoiceEngineHealthCheck creates a health check for voice engine
func VoiceEngineHealthCheck(statusFunc func() error) HealthCheck {
	return func(ctx context.Context) error {
		if err := statusFunc(); err != nil {
			return fmt.Errorf("voice engine unhealthy: %w", err)
		}
		return nil
	}
}

// DiskSpaceHealthCheck creates a health check for available disk space
func DiskSpaceHealthCheck(path string, minFreeBytes int64) HealthCheck {
	return func(ctx context.Context) error {
		available, err := getDiskSpace(path)
		if err != nil {
			return fmt.Errorf("failed to check disk space: %w", err)
		}

		if available < minFreeBytes {
			return fmt.Errorf("insufficient disk space: %d bytes available, %d bytes required", available, minFreeBytes)
		}

		return nil
	}
}

// MemoryHealthCheck creates a health check for memory usage
func MemoryHealthCheck(maxMemoryBytes uint64) HealthCheck {
	return func(ctx context.Context) error {
		memStats := getMemoryStats()

		// Check both heap allocation and system memory
		if memStats.Alloc > maxMemoryBytes {
			return fmt.Errorf("memory usage exceeded: %d bytes allocated, %d bytes max", memStats.Alloc, maxMemoryBytes)
		}

		// Also check if we're using too much system memory
		if memStats.Sys > maxMemoryBytes*2 {
			return fmt.Errorf("system memory usage high: %d bytes reserved", memStats.Sys)
		}

		return nil
	}
}
