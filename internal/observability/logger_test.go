package observability

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("creates logger with default config", func(t *testing.T) {
		cfg := LoggerConfig{
			Level:        zerolog.InfoLevel,
			Format:       "json",
			OutputPath:   "stdout",
			EnableCaller: false,
			EnableStack:  false,
			Service:      "test-service",
			Version:      "1.0.0",
		}

		logger := NewLogger(cfg)
		assert.NotNil(t, logger)
	})

	t.Run("creates logger with console format", func(t *testing.T) {
		cfg := LoggerConfig{
			Level:        zerolog.DebugLevel,
			Format:       "console",
			OutputPath:   "stdout",
			EnableCaller: true,
			EnableStack:  true,
			Service:      "test-service",
			Version:      "1.0.0",
		}

		logger := NewLogger(cfg)
		assert.NotNil(t, logger)
	})

	t.Run("creates logger with file output", func(t *testing.T) {
		// Note: Using a persistent temp directory instead of t.TempDir()
		// because NewLogger keeps the file open and Windows can't clean up open files
		tmpDir, err := os.MkdirTemp("", "concord_logger_test_*")
		require.NoError(t, err)
		logFile := filepath.Join(tmpDir, "test.log")

		cfg := LoggerConfig{
			Level:        zerolog.InfoLevel,
			Format:       "json",
			OutputPath:   logFile,
			EnableCaller: false,
			EnableStack:  false,
			Service:      "test-service",
			Version:      "1.0.0",
		}

		logger := NewLogger(cfg)
		assert.NotNil(t, logger)

		// Write a log message
		logger.Info().Msg("test message")

		// Verify file was created (but don't immediately clean up due to open file handle)
		_, err = os.Stat(logFile)
		assert.NoError(t, err)

		// Clean up is deferred - the OS will clean up temp dirs eventually
		// In production, log files stay open for the lifetime of the application
		t.Cleanup(func() {
			// Best-effort cleanup - may fail on Windows if file is still open
			os.RemoveAll(tmpDir)
		})
	})
}

func TestNewNopLogger(t *testing.T) {
	logger := NewNopLogger()
	assert.NotNil(t, logger)

	// Should not panic when logging
	logger.Info().Msg("this should be discarded")
	logger.Error().Msg("this should also be discarded")
}

func TestNewTestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewTestLogger(&buf)

	logger.Info().Msg("test message")

	assert.Contains(t, buf.String(), "test message")
}

func TestLoggerMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := NewTestLogger(&buf)

	middleware := NewLoggerMiddleware(logger)
	assert.NotNil(t, middleware)

	t.Run("WithContext", func(t *testing.T) {
		contextLogger := middleware.WithContext(map[string]interface{}{
			"request_id": "123",
			"user_id":    "456",
		})

		contextLogger.Info().Msg("test with context")
		output := buf.String()
		assert.Contains(t, output, "request_id")
		assert.Contains(t, output, "123")
	})

	buf.Reset()

	t.Run("WithUserID", func(t *testing.T) {
		userLogger := middleware.WithUserID("user-789")
		userLogger.Info().Msg("user action")
		assert.Contains(t, buf.String(), "user-789")
	})

	buf.Reset()

	t.Run("WithChannelID", func(t *testing.T) {
		channelLogger := middleware.WithChannelID("channel-abc")
		channelLogger.Info().Msg("channel event")
		assert.Contains(t, buf.String(), "channel-abc")
	})

	buf.Reset()

	t.Run("WithServerID", func(t *testing.T) {
		serverLogger := middleware.WithServerID("server-xyz")
		serverLogger.Info().Msg("server event")
		assert.Contains(t, buf.String(), "server-xyz")
	})

	buf.Reset()

	t.Run("WithPeerID", func(t *testing.T) {
		peerLogger := middleware.WithPeerID("peer-def")
		peerLogger.Info().Msg("peer connection")
		assert.Contains(t, buf.String(), "peer-def")
	})

	buf.Reset()

	t.Run("WithAction", func(t *testing.T) {
		actionLogger := middleware.WithAction("send_message")
		actionLogger.Info().Msg("action performed")
		assert.Contains(t, buf.String(), "send_message")
	})
}

func TestLogEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := NewTestLogger(&buf)

	t.Run("Success", func(t *testing.T) {
		buf.Reset()
		event := &LogEvent{
			Logger: logger,
			Action: "create_user",
			Entity: "user",
			ID:     "123",
			Context: map[string]interface{}{
				"username": "testuser",
			},
		}

		event.Success("user created successfully")
		output := buf.String()
		assert.Contains(t, output, "create_user")
		assert.Contains(t, output, "user")
		assert.Contains(t, output, "123")
		assert.Contains(t, output, "success")
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		event := &LogEvent{
			Logger: logger,
			Action: "delete_user",
			Entity: "user",
			ID:     "456",
		}

		event.Error(assert.AnError, "failed to delete user")
		output := buf.String()
		assert.Contains(t, output, "delete_user")
		assert.Contains(t, output, "error")
	})

	t.Run("Warning", func(t *testing.T) {
		buf.Reset()
		event := &LogEvent{
			Logger: logger,
			Action: "update_user",
			Entity: "user",
			ID:     "789",
		}

		event.Warning("user update requires verification")
		output := buf.String()
		assert.Contains(t, output, "update_user")
		assert.Contains(t, output, "warning")
	})
}

func TestPerformanceLog(t *testing.T) {
	var buf bytes.Buffer
	logger := NewTestLogger(&buf)

	t.Run("End", func(t *testing.T) {
		buf.Reset()
		perfLog := NewPerformanceLog(logger, "database_query")
		perfLog.End()

		output := buf.String()
		assert.Contains(t, output, "database_query")
		assert.Contains(t, output, "duration")
	})

	t.Run("EndWithError", func(t *testing.T) {
		buf.Reset()
		perfLog := NewPerformanceLog(logger, "api_request")
		perfLog.EndWithError(assert.AnError)

		output := buf.String()
		assert.Contains(t, output, "api_request")
		assert.Contains(t, output, "error")
		assert.Contains(t, output, "duration")
	})

	t.Run("EndWithContext", func(t *testing.T) {
		buf.Reset()
		perfLog := NewPerformanceLog(logger, "cache_operation")
		perfLog.EndWithContext(map[string]interface{}{
			"cache_hit": true,
			"items":     42,
		})

		output := buf.String()
		assert.Contains(t, output, "cache_operation")
		assert.Contains(t, output, "cache_hit")
		assert.Contains(t, output, "duration")
	})
}

func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "sanitizes password",
			input: map[string]interface{}{
				"username": "testuser",
				"password": "secret123",
			},
			expected: map[string]interface{}{
				"username": "testuser",
				"password": "[REDACTED]",
			},
		},
		{
			name: "sanitizes multiple sensitive fields",
			input: map[string]interface{}{
				"user_id":       "123",
				"token":         "abc123",
				"api_key":       "key123",
				"refresh_token": "refresh123",
			},
			expected: map[string]interface{}{
				"user_id":       "123",
				"token":         "[REDACTED]",
				"api_key":       "[REDACTED]",
				"refresh_token": "[REDACTED]",
			},
		},
		{
			name: "preserves non-sensitive fields",
			input: map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
				"role":     "admin",
			},
			expected: map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
				"role":     "admin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForLog(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOpenLogFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates file in existing directory", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "test.log")
		file, err := openLogFile(logPath)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		// Verify file exists
		_, err = os.Stat(logPath)
		assert.NoError(t, err)
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		logPath := filepath.Join(tmpDir, "subdir", "test.log")
		file, err := openLogFile(logPath)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer file.Close()

		// Verify file exists
		_, err = os.Stat(logPath)
		assert.NoError(t, err)

		// Verify directory was created
		dirPath := filepath.Dir(logPath)
		_, err = os.Stat(dirPath)
		assert.NoError(t, err)
	})
}
