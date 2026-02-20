package security

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validator provides input validation functions to prevent injection attacks
type Validator struct {
	// MaxInputLength is the maximum allowed length for text inputs
	MaxInputLength int
	// MaxURLLength is the maximum allowed length for URLs
	MaxURLLength int
	// AllowedSchemes contains the list of allowed URL schemes
	AllowedSchemes []string
}

// NewValidator creates a new input validator with secure defaults
// Complexity: O(1)
func NewValidator() *Validator {
	return &Validator{
		MaxInputLength: 10000,   // 10KB
		MaxURLLength:   2048,    // Standard URL max length
		AllowedSchemes: []string{"http", "https"},
	}
}

// ValidateUsername validates a username to prevent injection attacks
// Complexity: O(n) where n is the length of the username
func (v *Validator) ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}

	if len(username) > 32 {
		return fmt.Errorf("username must be at most 32 characters")
	}

	// Only allow alphanumeric characters, underscores, and hyphens
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, username)
	if err != nil {
		return fmt.Errorf("failed to validate username: %w", err)
	}

	if !matched {
		return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
	}

	return nil
}

// ValidateEmail validates an email address
// Complexity: O(n) where n is the length of the email
func (v *Validator) ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	if len(email) > 254 { // RFC 5321
		return fmt.Errorf("email is too long")
	}

	// Simple but effective email regex
	// More complex regex can lead to ReDoS attacks
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// ValidateURL validates a URL to prevent SSRF and XSS attacks
// Complexity: O(n) where n is the length of the URL
func (v *Validator) ValidateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	if len(urlStr) > v.MaxURLLength {
		return fmt.Errorf("URL is too long (max %d characters)", v.MaxURLLength)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme
	schemeAllowed := false
	for _, scheme := range v.AllowedSchemes {
		if parsedURL.Scheme == scheme {
			schemeAllowed = true
			break
		}
	}

	if !schemeAllowed {
		return fmt.Errorf("URL scheme not allowed (allowed: %v)", v.AllowedSchemes)
	}

	// Prevent SSRF by blocking private IP ranges
	if parsedURL.Hostname() != "" {
		ip := net.ParseIP(parsedURL.Hostname())
		if ip != nil {
			if isPrivateIP(ip) {
				return fmt.Errorf("URL points to private IP address")
			}
		}
	}

	return nil
}

// ValidateTextInput validates general text input
// Complexity: O(n) where n is the length of the input
func (v *Validator) ValidateTextInput(input string, fieldName string) error {
	if !utf8.ValidString(input) {
		return fmt.Errorf("%s contains invalid UTF-8 characters", fieldName)
	}

	if len(input) > v.MaxInputLength {
		return fmt.Errorf("%s is too long (max %d characters)", fieldName, v.MaxInputLength)
	}

	// Check for null bytes (can cause issues in C-based systems)
	if strings.Contains(input, "\x00") {
		return fmt.Errorf("%s contains null bytes", fieldName)
	}

	return nil
}

// ValidateChannelName validates a channel name
// Complexity: O(n) where n is the length of the channel name
func (v *Validator) ValidateChannelName(name string) error {
	if name == "" {
		return fmt.Errorf("channel name cannot be empty")
	}

	if len(name) < 2 {
		return fmt.Errorf("channel name must be at least 2 characters")
	}

	if len(name) > 64 {
		return fmt.Errorf("channel name must be at most 64 characters")
	}

	// Allow alphanumeric, spaces, underscores, and hyphens
	matched, err := regexp.MatchString(`^[a-zA-Z0-9 _-]+$`, name)
	if err != nil {
		return fmt.Errorf("failed to validate channel name: %w", err)
	}

	if !matched {
		return fmt.Errorf("channel name can only contain letters, numbers, spaces, underscores, and hyphens")
	}

	return nil
}

// ValidateServerName validates a server name
// Complexity: O(n) where n is the length of the server name
func (v *Validator) ValidateServerName(name string) error {
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if len(name) < 2 {
		return fmt.Errorf("server name must be at least 2 characters")
	}

	if len(name) > 100 {
		return fmt.Errorf("server name must be at most 100 characters")
	}

	if !utf8.ValidString(name) {
		return fmt.Errorf("server name contains invalid UTF-8 characters")
	}

	// Check for control characters
	for _, r := range name {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return fmt.Errorf("server name contains control characters")
		}
	}

	return nil
}

// ValidateFileExtension validates a file extension against a whitelist
// Complexity: O(n) where n is the number of allowed extensions
func (v *Validator) ValidateFileExtension(filename string, allowedExtensions []string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Extract extension
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return fmt.Errorf("filename has no extension")
	}

	ext := strings.ToLower(parts[len(parts)-1])

	// Check against whitelist
	for _, allowed := range allowedExtensions {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}

	return fmt.Errorf("file extension .%s is not allowed", ext)
}

// isPrivateIP checks if an IP address is in a private range
// Prevents SSRF attacks by blocking requests to internal services
func isPrivateIP(ip net.IP) bool {
	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for private ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // Link-local
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
	}

	for _, cidr := range privateRanges {
		_, subnet, _ := net.ParseCIDR(cidr)
		if subnet != nil && subnet.Contains(ip) {
			return true
		}
	}

	return false
}

// SanitizeSQL prevents SQL injection by escaping dangerous characters
// NOTE: This should NOT be used as a replacement for parameterized queries
// Use this only for logging or displaying SQL, never for actual queries
// Complexity: O(n) where n is the length of the input
func SanitizeSQL(input string) string {
	// Replace single quotes with two single quotes (SQL escaping)
	sanitized := strings.ReplaceAll(input, "'", "''")
	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")
	return sanitized
}

// ContainsSQLKeywords checks if input contains common SQL keywords
// This is a basic defense-in-depth measure, not a primary security control
// Complexity: O(n*m) where n is input length and m is number of keywords
func ContainsSQLKeywords(input string) bool {
	sqlKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE",
		"ALTER", "EXEC", "EXECUTE", "UNION", "DECLARE", "CAST",
		"SCRIPT", "JAVASCRIPT", "ONERROR", "ONLOAD",
	}

	upperInput := strings.ToUpper(input)

	for _, keyword := range sqlKeywords {
		if strings.Contains(upperInput, keyword) {
			return true
		}
	}

	return false
}
