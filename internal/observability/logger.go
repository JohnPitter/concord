package observability

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// LoggerConfig contains configuration for logger setup
type LoggerConfig struct {
	Level        zerolog.Level
	Format       string // "json" or "console"
	OutputPath   string // file path or "stdout"
	ErrorPath    string // error log file or "stderr"
	EnableCaller bool   // Include caller information
	EnableStack  bool   // Include stack trace for errors
	Service      string // Service name
	Version      string // Application version
}

// NewLogger creates a new zerolog logger with the given configuration
// All logs are structured and include timestamp, service name, and version
// Complexity: O(1)
func NewLogger(cfg LoggerConfig) zerolog.Logger {
	// Configure zerolog to use pkgerrors for stack traces
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Determine output writer
	var output io.Writer
	if cfg.OutputPath == "" || cfg.OutputPath == "stdout" {
		output = os.Stdout
	} else {
		file, err := openLogFile(cfg.OutputPath)
		if err != nil {
			// Fallback to stdout if file can't be opened
			output = os.Stdout
		} else {
			output = file
		}
	}

	// Apply formatting
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	}

	// Create base logger
	logger := zerolog.New(output).
		Level(cfg.Level).
		With().
		Timestamp().
		Str("service", cfg.Service).
		Str("version", cfg.Version).
		Logger()

	// Add caller information if enabled
	if cfg.EnableCaller {
		logger = logger.With().Caller().Logger()
	}

	// Add stack trace for errors if enabled
	if cfg.EnableStack {
		logger = logger.With().Stack().Logger()
	}

	return logger
}

// openLogFile opens or creates a log file with appropriate permissions
// Creates parent directories if they don't exist
func openLogFile(path string) (*os.File, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// NewNopLogger creates a no-op logger that discards all logs
// Useful for testing
func NewNopLogger() zerolog.Logger {
	return zerolog.New(io.Discard).Level(zerolog.Disabled)
}

// NewTestLogger creates a logger suitable for testing
// Outputs to a buffer that can be inspected
func NewTestLogger(output io.Writer) zerolog.Logger {
	return zerolog.New(output).
		Level(zerolog.DebugLevel).
		With().
		Timestamp().
		Logger()
}

// LoggerMiddleware is a helper to add consistent context to loggers
type LoggerMiddleware struct {
	logger zerolog.Logger
}

// NewLoggerMiddleware creates a new logger middleware
func NewLoggerMiddleware(logger zerolog.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{logger: logger}
}

// WithContext adds context fields to the logger
func (lm *LoggerMiddleware) WithContext(fields map[string]interface{}) zerolog.Logger {
	ctx := lm.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}

// WithUserID adds user_id to logger context
func (lm *LoggerMiddleware) WithUserID(userID string) zerolog.Logger {
	return lm.logger.With().Str("user_id", userID).Logger()
}

// WithChannelID adds channel_id to logger context
func (lm *LoggerMiddleware) WithChannelID(channelID string) zerolog.Logger {
	return lm.logger.With().Str("channel_id", channelID).Logger()
}

// WithServerID adds server_id to logger context
func (lm *LoggerMiddleware) WithServerID(serverID string) zerolog.Logger {
	return lm.logger.With().Str("server_id", serverID).Logger()
}

// WithPeerID adds peer_id to logger context
func (lm *LoggerMiddleware) WithPeerID(peerID string) zerolog.Logger {
	return lm.logger.With().Str("peer_id", peerID).Logger()
}

// WithAction adds action to logger context
func (lm *LoggerMiddleware) WithAction(action string) zerolog.Logger {
	return lm.logger.With().Str("action", action).Logger()
}

// LogEvent represents common log events with consistent structure
type LogEvent struct {
	Logger  zerolog.Logger
	Action  string
	Entity  string // user, channel, server, peer, etc.
	ID      string
	Context map[string]interface{}
}

// Success logs a successful operation
func (le *LogEvent) Success(msg string) {
	event := le.Logger.Info().
		Str("action", le.Action).
		Str("entity", le.Entity).
		Str("entity_id", le.ID).
		Str("status", "success")

	for k, v := range le.Context {
		event = event.Interface(k, v)
	}

	event.Msg(msg)
}

// Error logs a failed operation
func (le *LogEvent) Error(err error, msg string) {
	event := le.Logger.Error().
		Err(err).
		Str("action", le.Action).
		Str("entity", le.Entity).
		Str("entity_id", le.ID).
		Str("status", "error")

	for k, v := range le.Context {
		event = event.Interface(k, v)
	}

	event.Msg(msg)
}

// Warning logs a warning
func (le *LogEvent) Warning(msg string) {
	event := le.Logger.Warn().
		Str("action", le.Action).
		Str("entity", le.Entity).
		Str("entity_id", le.ID).
		Str("status", "warning")

	for k, v := range le.Context {
		event = event.Interface(k, v)
	}

	event.Msg(msg)
}

// PerformanceLog logs performance metrics
type PerformanceLog struct {
	Logger    zerolog.Logger
	Operation string
	StartTime time.Time
}

// NewPerformanceLog creates a new performance logger
func NewPerformanceLog(logger zerolog.Logger, operation string) *PerformanceLog {
	return &PerformanceLog{
		Logger:    logger,
		Operation: operation,
		StartTime: time.Now(),
	}
}

// End logs the completion of the operation with duration
func (pl *PerformanceLog) End() {
	duration := time.Since(pl.StartTime)
	pl.Logger.Debug().
		Str("operation", pl.Operation).
		Dur("duration_ms", duration).
		Int64("duration_ns", duration.Nanoseconds()).
		Msg("operation completed")
}

// EndWithError logs the completion with an error
func (pl *PerformanceLog) EndWithError(err error) {
	duration := time.Since(pl.StartTime)
	pl.Logger.Error().
		Err(err).
		Str("operation", pl.Operation).
		Dur("duration_ms", duration).
		Int64("duration_ns", duration.Nanoseconds()).
		Msg("operation failed")
}

// EndWithContext logs completion with additional context
func (pl *PerformanceLog) EndWithContext(ctx map[string]interface{}) {
	duration := time.Since(pl.StartTime)
	event := pl.Logger.Debug().
		Str("operation", pl.Operation).
		Dur("duration_ms", duration).
		Int64("duration_ns", duration.Nanoseconds())

	for k, v := range ctx {
		event = event.Interface(k, v)
	}

	event.Msg("operation completed")
}

// SanitizeForLog removes sensitive information from log output
// Complexity: O(n) where n is the number of fields
func SanitizeForLog(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	sensitiveKeys := map[string]bool{
		"password":     true,
		"token":        true,
		"secret":       true,
		"api_key":      true,
		"private_key":  true,
		"access_token": true,
		"refresh_token": true,
	}

	for k, v := range data {
		if sensitiveKeys[k] {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}
