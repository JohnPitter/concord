package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

// Config represents the complete application configuration
type Config struct {
	// Application settings
	App AppConfig `json:"app"`

	// Database configuration
	Database DatabaseConfig `json:"database"`

	// Server configuration (for central server mode)
	Server ServerConfig `json:"server"`

	// P2P networking configuration
	P2P P2PConfig `json:"p2p"`

	// Voice configuration
	Voice VoiceConfig `json:"voice"`

	// Translation configuration
	Translation TranslationConfig `json:"translation"`

	// Security configuration
	Security SecurityConfig `json:"security"`

	// Logging configuration
	Logging LoggingConfig `json:"logging"`

	// Cache configuration
	Cache CacheConfig `json:"cache"`
}

// AppConfig contains general application settings
type AppConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"` // dev, staging, production
	DataDir     string `json:"data_dir"`    // Directory for user data
	ConfigDir   string `json:"config_dir"`  // Directory for config files
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	// SQLite configuration (client-side)
	SQLite SQLiteConfig `json:"sqlite"`

	// PostgreSQL configuration (server-side)
	Postgres PostgresConfig `json:"postgres"`
}

// SQLiteConfig contains SQLite-specific settings
type SQLiteConfig struct {
	Path            string        `json:"path"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	WALMode         bool          `json:"wal_mode"` // Write-Ahead Logging
	ForeignKeys     bool          `json:"foreign_keys"`
	BusyTimeout     time.Duration `json:"busy_timeout"`
}

// PostgresConfig contains PostgreSQL-specific settings
type PostgresConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Database        string        `json:"database"`
	User            string        `json:"user"`
	Password        string        `json:"password"`
	SSLMode         string        `json:"ssl_mode"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// ServerConfig contains central server settings
type ServerConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	TLSEnabled      bool          `json:"tls_enabled"`
	TLSCertFile     string        `json:"tls_cert_file"`
	TLSKeyFile      string        `json:"tls_key_file"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	CORS            CORSConfig    `json:"cors"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	Enabled        bool     `json:"enabled"`
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers"`
}

// P2PConfig contains P2P networking settings
type P2PConfig struct {
	Enabled           bool          `json:"enabled"`
	ListenAddresses   []string      `json:"listen_addresses"`
	BootstrapPeers    []string      `json:"bootstrap_peers"`
	RendezvousString  string        `json:"rendezvous_string"`
	EnableRelay       bool          `json:"enable_relay"`
	EnableHolePunch   bool          `json:"enable_hole_punch"`
	EnableMDNS        bool          `json:"enable_mdns"` // Local network discovery
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	MaxPeers          int           `json:"max_peers"`
}

// VoiceConfig contains voice chat settings
type VoiceConfig struct {
	SampleRate        int           `json:"sample_rate"`        // Hz (48000)
	Channels          int           `json:"channels"`           // 1 = mono, 2 = stereo
	FrameSize         int           `json:"frame_size"`         // Samples per frame (960 for 20ms @ 48kHz)
	Bitrate           int           `json:"bitrate"`            // bps (64000)
	JitterBufferSize  time.Duration `json:"jitter_buffer_size"` // Default jitter buffer (50ms)
	MaxJitterBuffer   time.Duration `json:"max_jitter_buffer"`  // Max jitter buffer (200ms)
	EnableVAD         bool          `json:"enable_vad"`         // Voice Activity Detection
	VADThreshold      float32       `json:"vad_threshold"`      // 0.0 - 1.0
	EnableNoiseSuppression bool     `json:"enable_noise_suppression"`
	MaxChannelUsers   int           `json:"max_channel_users"` // Max users per voice channel
}

// TranslationConfig contains voice translation settings
type TranslationConfig struct {
	Enabled         bool          `json:"enabled"`
	PersonaPlexURL  string        `json:"personaplex_url"`
	APIKey          string        `json:"api_key"`
	DefaultLang     string        `json:"default_lang"`
	CacheEnabled    bool          `json:"cache_enabled"`
	CacheSize       int           `json:"cache_size"`       // Max cached translations
	Timeout         time.Duration `json:"timeout"`          // Request timeout
	MaxLatency      time.Duration `json:"max_latency"`      // Auto-disable if exceeded
	CircuitBreaker  bool          `json:"circuit_breaker"`  // Auto-disable on failures
	FailureThreshold int          `json:"failure_threshold"` // Failures before circuit breaks
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	// JWT settings
	JWTSecret           string        `json:"jwt_secret"`
	JWTAccessExpiry     time.Duration `json:"jwt_access_expiry"`  // 15 minutes
	JWTRefreshExpiry    time.Duration `json:"jwt_refresh_expiry"` // 30 days

	// Rate limiting
	RateLimitEnabled    bool          `json:"rate_limit_enabled"`
	RateLimitMessages   int           `json:"rate_limit_messages"`   // per minute
	RateLimitFiles      int           `json:"rate_limit_files"`      // per hour
	RateLimitAPI        int           `json:"rate_limit_api"`        // per minute

	// File upload limits
	MaxFileSize         int64         `json:"max_file_size"`         // bytes (50MB)
	AllowedFileTypes    []string      `json:"allowed_file_types"`

	// Encryption
	EncryptLocalDB      bool          `json:"encrypt_local_db"`
	E2EEEnabled         bool          `json:"e2ee_enabled"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level       string `json:"level"`        // debug, info, warn, error
	Format      string `json:"format"`       // json, console
	OutputPath  string `json:"output_path"`  // file path or stdout
	ErrorPath   string `json:"error_path"`   // error log file
	EnableCaller bool   `json:"enable_caller"` // Include caller in logs
	EnableStack  bool   `json:"enable_stack"`  // Include stack trace for errors
}

// CacheConfig contains cache settings
type CacheConfig struct {
	// In-memory LRU cache (client-side)
	LRU LRUConfig `json:"lru"`

	// Redis cache (server-side)
	Redis RedisConfig `json:"redis"`
}

// LRUConfig contains LRU cache settings
type LRUConfig struct {
	Enabled    bool          `json:"enabled"`
	MaxEntries int           `json:"max_entries"`
	TTL        time.Duration `json:"ttl"`
}

// RedisConfig contains Redis cache settings
type RedisConfig struct {
	Enabled      bool          `json:"enabled"`
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Password     string        `json:"password"`
	DB           int           `json:"db"`
	MaxRetries   int           `json:"max_retries"`
	PoolSize     int           `json:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

// Load loads configuration from file and environment variables
// Priority: env vars > config file > defaults
func Load(configPath string) (*Config, error) {
	// Start with defaults
	cfg := Default()

	// Load from config file if it exists
	if configPath != "" {
		if err := cfg.loadFromFile(configPath); err != nil {
			// If config file doesn't exist, create it with defaults
			if errors.Is(err, os.ErrNotExist) {
				if err := cfg.Save(configPath); err != nil {
					return nil, fmt.Errorf("failed to create default config: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to load config: %w", err)
			}
		}
	}

	// Override with environment variables
	cfg.loadFromEnv()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// loadFromFile loads configuration from a JSON file
func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// loadFromEnv overrides configuration with environment variables
func (c *Config) loadFromEnv() {
	// App
	if v := os.Getenv("CONCORD_ENV"); v != "" {
		c.App.Environment = v
	}
	if v := os.Getenv("CONCORD_DATA_DIR"); v != "" {
		c.App.DataDir = v
	}

	// Database
	if v := os.Getenv("CONCORD_DB_PATH"); v != "" {
		c.Database.SQLite.Path = v
	}
	if v := os.Getenv("POSTGRES_HOST"); v != "" {
		c.Database.Postgres.Host = v
	}
	if v := os.Getenv("POSTGRES_PASSWORD"); v != "" {
		c.Database.Postgres.Password = v
	}

	// Server
	if v := os.Getenv("CONCORD_SERVER_HOST"); v != "" {
		c.Server.Host = v
	}

	// Security
	if v := os.Getenv("CONCORD_JWT_SECRET"); v != "" {
		c.Security.JWTSecret = v
	}

	// Translation
	if v := os.Getenv("PERSONAPLEX_URL"); v != "" {
		c.Translation.PersonaPlexURL = v
	}
	if v := os.Getenv("PERSONAPLEX_API_KEY"); v != "" {
		c.Translation.APIKey = v
	}

	// Redis
	if v := os.Getenv("REDIS_HOST"); v != "" {
		c.Cache.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		c.Cache.Redis.Password = v
	}

	// Logging
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
}

// Save saves configuration to a JSON file
func (c *Config) Save(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate app config
	if c.App.Name == "" {
		return errors.New("app name cannot be empty")
	}
	if c.App.Environment != "dev" && c.App.Environment != "staging" && c.App.Environment != "production" {
		return fmt.Errorf("invalid environment: %s (must be dev, staging, or production)", c.App.Environment)
	}

	// Validate database paths
	if c.Database.SQLite.Path == "" {
		return errors.New("SQLite database path cannot be empty")
	}

	// Validate server config (if running in server mode)
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate voice config
	if c.Voice.SampleRate <= 0 {
		return fmt.Errorf("invalid sample rate: %d", c.Voice.SampleRate)
	}
	if c.Voice.Bitrate <= 0 {
		return fmt.Errorf("invalid bitrate: %d", c.Voice.Bitrate)
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true, "fatal": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	// Validate JWT secret in production
	if c.App.Environment == "production" && len(c.Security.JWTSecret) < 32 {
		return errors.New("JWT secret must be at least 32 characters in production")
	}

	return nil
}

// GetLogLevel returns the zerolog level based on configuration
func (c *Config) GetLogLevel() zerolog.Level {
	switch c.Logging.Level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "dev"
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Postgres.Host,
		c.Database.Postgres.Port,
		c.Database.Postgres.User,
		c.Database.Postgres.Password,
		c.Database.Postgres.Database,
		c.Database.Postgres.SSLMode,
	)
}

// GetRedisDSN returns the Redis connection string
func (c *Config) GetRedisDSN() string {
	return fmt.Sprintf("%s:%d", c.Cache.Redis.Host, c.Cache.Redis.Port)
}
