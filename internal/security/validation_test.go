package security

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	assert.NotNil(t, v)
	assert.Equal(t, 10000, v.MaxInputLength)
	assert.Equal(t, 2048, v.MaxURLLength)
	assert.Contains(t, v.AllowedSchemes, "http")
	assert.Contains(t, v.AllowedSchemes, "https")
}

func TestValidator_ValidateUsername(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		username  string
		wantError bool
	}{
		{"valid username", "john_doe", false},
		{"valid with numbers", "user123", false},
		{"valid with hyphen", "john-doe", false},
		{"empty username", "", true},
		{"too short", "ab", true},
		{"too long", "this_username_is_way_too_long_for_validation", true},
		{"invalid characters", "user@name", true},
		{"with spaces", "user name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateUsername(tt.username)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateEmail(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		email     string
		wantError bool
	}{
		{"valid email", "user@example.com", false},
		{"valid with subdomain", "user@mail.example.com", false},
		{"valid with plus", "user+tag@example.com", false},
		{"empty email", "", true},
		{"missing @", "userexample.com", true},
		{"missing domain", "user@", true},
		{"missing username", "@example.com", true},
		{"invalid format", "not an email", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateEmail(tt.email)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateURL(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{"valid http", "http://example.com", false},
		{"valid https", "https://example.com", false},
		{"valid with path", "https://example.com/path/to/resource", false},
		{"empty URL", "", true},
		{"invalid scheme", "ftp://example.com", true},
		{"javascript scheme", "javascript:alert(1)", true},
		{"private IP (localhost)", "http://127.0.0.1", true},
		{"private IP (LAN)", "http://192.168.1.1", true},
		{"private IP (10.x)", "http://10.0.0.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateURL(tt.url)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateTextInput(t *testing.T) {
	v := NewValidator()

	t.Run("valid text", func(t *testing.T) {
		err := v.ValidateTextInput("This is valid text", "message")
		assert.NoError(t, err)
	})

	t.Run("contains null byte", func(t *testing.T) {
		err := v.ValidateTextInput("text\x00with null", "message")
		assert.Error(t, err)
	})

	t.Run("too long", func(t *testing.T) {
		longText := make([]byte, v.MaxInputLength+1)
		for i := range longText {
			longText[i] = 'a'
		}
		err := v.ValidateTextInput(string(longText), "message")
		assert.Error(t, err)
	})
}

func TestValidator_ValidateChannelName(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		channel   string
		wantError bool
	}{
		{"valid channel", "general", false},
		{"with spaces", "general chat", false},
		{"with hyphen", "off-topic", false},
		{"with underscore", "dev_team", false},
		{"empty", "", true},
		{"too short", "a", true},
		{"too long", "this_channel_name_is_way_too_long_and_should_fail_validation_test", true},
		{"special chars", "channel#1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateChannelName(tt.channel)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateServerName(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		server    string
		wantError bool
	}{
		{"valid server", "My Game Server", false},
		{"with emoji", "Gaming 🎮", false},
		{"empty", "", true},
		{"too short", "X", true},
		{"control characters", "server\x01name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateServerName(tt.server)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateFileExtension(t *testing.T) {
	v := NewValidator()
	allowed := []string{"jpg", "png", "pdf"}

	tests := []struct {
		name      string
		filename  string
		wantError bool
	}{
		{"allowed jpg", "photo.jpg", false},
		{"allowed PNG uppercase", "photo.PNG", false},
		{"not allowed", "file.exe", true},
		{"no extension", "filename", true},
		{"empty filename", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateFileExtension(tt.filename, allowed)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip        string
		isPrivate bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := parseIP(tt.ip)
			result := isPrivateIP(ip)
			assert.Equal(t, tt.isPrivate, result)
		})
	}
}

func TestSanitizeSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"escapes single quotes", "O'Brien", "O''Brien"},
		{"removes null bytes", "text\x00here", "texthere"},
		{"normal text", "normal", "normal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSQL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsSQLKeywords(t *testing.T) {
	tests := []struct {
		input    string
		contains bool
	}{
		{"SELECT * FROM users", true},
		{"DROP TABLE users", true},
		{"Hello world", false},
		{"insert into", true},
		{"normal text", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ContainsSQLKeywords(tt.input)
			assert.Equal(t, tt.contains, result)
		})
	}
}

// Helper function for IP parsing
func parseIP(ip string) net.IP {
	return net.ParseIP(ip)
}
