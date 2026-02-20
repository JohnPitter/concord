package config

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Default returns a Config with sensible default values
func Default() *Config {
	dataDir := getDefaultDataDir()
	configDir := getDefaultConfigDir()

	return &Config{
		App: AppConfig{
			Name:        "Concord",
			Version:     "0.1.0",
			Environment: "dev",
			DataDir:     dataDir,
			ConfigDir:   configDir,
		},

		Database: DatabaseConfig{
			SQLite: SQLiteConfig{
				Path:            filepath.Join(dataDir, "concord.db"),
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				WALMode:         true,
				ForeignKeys:     true,
				BusyTimeout:     5 * time.Second,
			},
			Postgres: PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				Database:        "concord",
				User:            "concord",
				Password:        "",
				SSLMode:         "prefer",
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
			},
		},

		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			TLSEnabled:      false,
			TLSCertFile:     "",
			TLSKeyFile:      "",
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			CORS: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:5173"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Authorization", "Content-Type"},
			},
		},

		P2P: P2PConfig{
			Enabled: true,
			ListenAddresses: []string{
				"/ip4/0.0.0.0/tcp/0",
				"/ip4/0.0.0.0/udp/0/quic-v1",
			},
			BootstrapPeers:    []string{},
			RendezvousString:  "concord-v1",
			EnableRelay:       true,
			EnableHolePunch:   true,
			EnableMDNS:        true,
			ConnectionTimeout: 30 * time.Second,
			MaxPeers:          100,
		},

		Voice: VoiceConfig{
			SampleRate:             48000,
			Channels:               1,
			FrameSize:              960, // 20ms at 48kHz
			Bitrate:                64000,
			JitterBufferSize:       50 * time.Millisecond,
			MaxJitterBuffer:        200 * time.Millisecond,
			EnableVAD:              true,
			VADThreshold:           0.3,
			EnableNoiseSuppression: true,
			MaxChannelUsers:        25,
		},

		Translation: TranslationConfig{
			Enabled:          false,
			PersonaPlexURL:   "https://personaplex.nvidia.com/api/v1",
			APIKey:           "",
			DefaultLang:      "en",
			CacheEnabled:     true,
			CacheSize:        1000,
			Timeout:          10 * time.Second,
			MaxLatency:       500 * time.Millisecond,
			CircuitBreaker:   true,
			FailureThreshold: 5,
		},

		Auth: AuthConfig{
			GitHubClientID: "", // Set via CONCORD_GITHUB_CLIENT_ID env var
		},

		Security: SecurityConfig{
			JWTSecret:        generateDefaultJWTSecret(),
			JWTAccessExpiry:  15 * time.Minute,
			JWTRefreshExpiry: 30 * 24 * time.Hour,

			RateLimitEnabled:  true,
			RateLimitMessages: 10, // 10 messages per second
			RateLimitFiles:    5,  // 5 files per minute
			RateLimitAPI:      60, // 60 requests per minute

			MaxFileSize: 50 * 1024 * 1024, // 50MB
			AllowedFileTypes: []string{
				"image/jpeg", "image/png", "image/gif", "image/webp",
				"video/mp4", "video/webm",
				"audio/mpeg", "audio/ogg", "audio/wav",
				"application/pdf",
				"text/plain",
				"application/zip",
			},

			EncryptLocalDB: false, // Enable in production
			E2EEEnabled:    true,
		},

		Logging: LoggingConfig{
			Level:        "info",
			Format:       "json",
			OutputPath:   "stdout",
			ErrorPath:    "stderr",
			EnableCaller: false,
			EnableStack:  true,
		},

		Cache: CacheConfig{
			LRU: LRUConfig{
				Enabled:    true,
				MaxEntries: 10000,
				TTL:        5 * time.Minute,
			},
			Redis: RedisConfig{
				Enabled:      false,
				Host:         "localhost",
				Port:         6379,
				Password:     "",
				DB:           0,
				MaxRetries:   3,
				PoolSize:     10,
				MinIdleConns: 5,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
		},
	}
}

// getDefaultDataDir returns the default data directory based on OS
func getDefaultDataDir() string {
	var baseDir string

	switch runtime.GOOS {
	case "windows":
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		baseDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	default: // linux and others
		baseDir = os.Getenv("XDG_DATA_HOME")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("HOME"), ".local", "share")
		}
	}

	return filepath.Join(baseDir, "Concord")
}

// getDefaultConfigDir returns the default config directory based on OS
func getDefaultConfigDir() string {
	var baseDir string

	switch runtime.GOOS {
	case "windows":
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		baseDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	default: // linux and others
		baseDir = os.Getenv("XDG_CONFIG_HOME")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("HOME"), ".config")
		}
	}

	return filepath.Join(baseDir, "Concord")
}

// generateDefaultJWTSecret generates a default JWT secret for development
// WARNING: In production, this MUST be overridden with a secure random secret
func generateDefaultJWTSecret() string {
	// For development only - NOT secure for production
	return "dev-secret-change-me-in-production-min-32-chars-required"
}
