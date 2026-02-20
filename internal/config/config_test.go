package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	assert.NotNil(t, cfg)
	assert.Equal(t, "Concord", cfg.App.Name)
	assert.Equal(t, "dev", cfg.App.Environment)
	assert.True(t, cfg.Voice.SampleRate > 0)
	assert.True(t, cfg.Database.SQLite.WALMode)
	assert.True(t, cfg.Security.E2EEEnabled)
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			setup:   func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid environment",
			setup: func(c *Config) {
				c.App.Environment = "invalid"
			},
			wantErr: true,
			errMsg:  "invalid environment",
		},
		{
			name: "empty app name",
			setup: func(c *Config) {
				c.App.Name = ""
			},
			wantErr: true,
			errMsg:  "app name cannot be empty",
		},
		{
			name: "invalid port",
			setup: func(c *Config) {
				c.Server.Port = 99999
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "invalid sample rate",
			setup: func(c *Config) {
				c.Voice.SampleRate = -1
			},
			wantErr: true,
			errMsg:  "invalid sample rate",
		},
		{
			name: "invalid log level",
			setup: func(c *Config) {
				c.Logging.Level = "invalid"
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "short JWT secret in production",
			setup: func(c *Config) {
				c.App.Environment = "production"
				c.Security.JWTSecret = "short"
			},
			wantErr: true,
			errMsg:  "JWT secret must be at least 32 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.setup(cfg)

			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config with custom values
	cfg := Default()
	cfg.App.Environment = "production"
	cfg.Server.Port = 9090
	cfg.Logging.Level = "debug"

	// Save to file
	err := cfg.Save(configPath)
	require.NoError(t, err)

	// Load from file
	loaded, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "production", loaded.App.Environment)
	assert.Equal(t, 9090, loaded.Server.Port)
	assert.Equal(t, "debug", loaded.Logging.Level)
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("CONCORD_ENV", "staging")
	os.Setenv("CONCORD_SERVER_HOST", "192.168.1.100")
	os.Setenv("LOG_LEVEL", "warn")
	defer func() {
		os.Unsetenv("CONCORD_ENV")
		os.Unsetenv("CONCORD_SERVER_HOST")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg := Default()
	cfg.loadFromEnv()

	assert.Equal(t, "staging", cfg.App.Environment)
	assert.Equal(t, "192.168.1.100", cfg.Server.Host)
	assert.Equal(t, "warn", cfg.Logging.Level)
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create and save config
	original := Default()
	original.Voice.Bitrate = 128000
	original.P2P.MaxPeers = 50

	err := original.Save(configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load config
	loaded, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, 128000, loaded.Voice.Bitrate)
	assert.Equal(t, 50, loaded.P2P.MaxPeers)
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"fatal", "fatal"},
		{"invalid", "info"}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			cfg := Default()
			cfg.Logging.Level = tt.level
			level := cfg.GetLogLevel()
			assert.Equal(t, tt.expected, level.String())
		})
	}
}

func TestIsProduction(t *testing.T) {
	cfg := Default()

	cfg.App.Environment = "production"
	assert.True(t, cfg.IsProduction())
	assert.False(t, cfg.IsDevelopment())

	cfg.App.Environment = "dev"
	assert.False(t, cfg.IsProduction())
	assert.True(t, cfg.IsDevelopment())
}

func TestGetDatabaseDSN(t *testing.T) {
	cfg := Default()
	cfg.Database.Postgres.Host = "localhost"
	cfg.Database.Postgres.Port = 5432
	cfg.Database.Postgres.User = "testuser"
	cfg.Database.Postgres.Password = "testpass"
	cfg.Database.Postgres.Database = "testdb"
	cfg.Database.Postgres.SSLMode = "disable"

	dsn := cfg.GetDatabaseDSN()
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	assert.Equal(t, expected, dsn)
}

func TestGetRedisDSN(t *testing.T) {
	cfg := Default()
	cfg.Cache.Redis.Host = "localhost"
	cfg.Cache.Redis.Port = 6379

	dsn := cfg.GetRedisDSN()
	assert.Equal(t, "localhost:6379", dsn)
}

func TestConfigDefaults(t *testing.T) {
	cfg := Default()

	// Verify voice defaults follow architecture specs
	assert.Equal(t, 48000, cfg.Voice.SampleRate)
	assert.Equal(t, 1, cfg.Voice.Channels) // mono
	assert.Equal(t, 960, cfg.Voice.FrameSize) // 20ms at 48kHz
	assert.Equal(t, 64000, cfg.Voice.Bitrate)
	assert.Equal(t, 50*time.Millisecond, cfg.Voice.JitterBufferSize)
	assert.Equal(t, 200*time.Millisecond, cfg.Voice.MaxJitterBuffer)
	assert.True(t, cfg.Voice.EnableVAD)
	assert.Equal(t, 25, cfg.Voice.MaxChannelUsers)

	// Verify security defaults
	assert.Equal(t, 15*time.Minute, cfg.Security.JWTAccessExpiry)
	assert.Equal(t, 30*24*time.Hour, cfg.Security.JWTRefreshExpiry)
	assert.True(t, cfg.Security.RateLimitEnabled)
	assert.Equal(t, int64(50*1024*1024), cfg.Security.MaxFileSize)

	// Verify P2P defaults
	assert.True(t, cfg.P2P.Enabled)
	assert.True(t, cfg.P2P.EnableRelay)
	assert.True(t, cfg.P2P.EnableHolePunch)
	assert.True(t, cfg.P2P.EnableMDNS)
	assert.Equal(t, 100, cfg.P2P.MaxPeers)

	// Verify cache defaults
	assert.True(t, cfg.Cache.LRU.Enabled)
	assert.Equal(t, 10000, cfg.Cache.LRU.MaxEntries)
	assert.Equal(t, 5*time.Minute, cfg.Cache.LRU.TTL)
}

func TestLoadNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	// Should create default config if file doesn't exist
	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify file was created
	_, err = os.Stat(configPath)
	require.NoError(t, err)
}

func TestDefaultDataDirExists(t *testing.T) {
	dataDir := getDefaultDataDir()
	assert.NotEmpty(t, dataDir)
	assert.Contains(t, dataDir, "Concord")
}

func TestDefaultConfigDirExists(t *testing.T) {
	configDir := getDefaultConfigDir()
	assert.NotEmpty(t, configDir)
	assert.Contains(t, configDir, "Concord")
}
