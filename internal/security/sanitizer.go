package security

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Sanitizer provides functions to sanitize user input
// Prevents XSS, HTML injection, and other content-based attacks
type Sanitizer struct {
	// AllowedTags contains HTML tags that are allowed (for rich text)
	AllowedTags map[string]bool
	// MaxLength is the maximum allowed length after sanitization
	MaxLength int
}

// NewSanitizer creates a new sanitizer with secure defaults
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		AllowedTags: map[string]bool{
			// Allow only safe formatting tags
			"b":      true,
			"i":      true,
			"u":      true,
			"em":     true,
			"strong": true,
			"code":   true,
			"pre":    true,
		},
		MaxLength: 50000, // 50KB
	}
}

// SanitizeHTML removes potentially dangerous HTML tags and attributes
// Complexity: O(n) where n is the length of input
func (s *Sanitizer) SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}

	// First, escape all HTML
	sanitized := html.EscapeString(input)

	// Then, selectively unescape allowed tags
	for tag := range s.AllowedTags {
		openTag := "&lt;" + tag + "&gt;"
		closeTag := "&lt;/" + tag + "&gt;"

		sanitized = strings.ReplaceAll(sanitized, openTag, "<"+tag+">")
		sanitized = strings.ReplaceAll(sanitized, closeTag, "</"+tag+">")
	}

	// Remove any remaining HTML attributes from allowed tags
	sanitized = s.removeHTMLAttributes(sanitized)

	// Truncate if too long
	if len(sanitized) > s.MaxLength {
		sanitized = sanitized[:s.MaxLength]
	}

	return sanitized
}

// StripHTML removes all HTML tags from input
// Complexity: O(n) where n is the length of input
func (s *Sanitizer) StripHTML(input string) string {
	// Remove all HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	stripped := re.ReplaceAllString(input, "")

	// Decode HTML entities
	stripped = html.UnescapeString(stripped)

	return stripped
}

// removeHTMLAttributes removes attributes from HTML tags
func (s *Sanitizer) removeHTMLAttributes(input string) string {
	// Pattern to match tags with attributes: <tag attr="value">
	re := regexp.MustCompile(`<(\w+)[^>]*>`)

	// Replace with tag without attributes: <tag>
	return re.ReplaceAllString(input, "<$1>")
}

// SanitizeMarkdown sanitizes markdown to prevent XSS through markdown injection
// Complexity: O(n) where n is the length of input
func (s *Sanitizer) SanitizeMarkdown(input string) string {
	if input == "" {
		return ""
	}

	// Remove javascript: and data: URLs from links
	// Pattern: [text](javascript:...)
	jsLinkRe := regexp.MustCompile(`\[([^\]]+)\]\s*\(\s*(?:javascript|data|vbscript):([^)]+)\)`)
	sanitized := jsLinkRe.ReplaceAllString(input, "[$1](#)")

	// Remove on* event handlers from inline HTML
	eventRe := regexp.MustCompile(`on\w+\s*=\s*["'][^"']*["']`)
	sanitized = eventRe.ReplaceAllString(sanitized, "")

	// Truncate if too long
	if len(sanitized) > s.MaxLength {
		sanitized = sanitized[:s.MaxLength]
	}

	return sanitized
}

// SanitizeFilename sanitizes a filename to prevent path traversal
// Complexity: O(n) where n is the length of filename
func (s *Sanitizer) SanitizeFilename(filename string) string {
	// Remove path separators
	sanitized := strings.ReplaceAll(filename, "/", "")
	sanitized = strings.ReplaceAll(sanitized, "\\", "")
	sanitized = strings.ReplaceAll(sanitized, "..", "")

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Remove control characters
	sanitized = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, sanitized)

	// Limit length
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	// If empty after sanitization, use a default
	if sanitized == "" {
		sanitized = "untitled"
	}

	return sanitized
}

// SanitizePath sanitizes a file path to prevent directory traversal
// Complexity: O(n) where n is the length of path
func (s *Sanitizer) SanitizePath(path string) string {
	// Remove any ../ or ..\\ sequences
	sanitized := strings.ReplaceAll(path, "../", "")
	sanitized = strings.ReplaceAll(sanitized, "..\\", "")

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Normalize slashes
	sanitized = strings.ReplaceAll(sanitized, "\\", "/")

	// Remove leading slashes
	sanitized = strings.TrimLeft(sanitized, "/")

	return sanitized
}

// RemoveControlCharacters removes control characters from input
// Complexity: O(n) where n is the length of input
func RemoveControlCharacters(input string) string {
	return strings.Map(func(r rune) rune {
		// Keep newlines, tabs, and carriage returns
		if r == '\n' || r == '\t' || r == '\r' {
			return r
		}
		// Remove other control characters
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, input)
}

// RemoveNullBytes removes null bytes from input
// Null bytes can cause issues in C-based systems
// Complexity: O(n) where n is the length of input
func RemoveNullBytes(input string) string {
	return strings.ReplaceAll(input, "\x00", "")
}

// TruncateString truncates a string to a maximum length
// Adds ellipsis if truncated
// Complexity: O(n) where n is maxLength
func TruncateString(input string, maxLength int) string {
	if len(input) <= maxLength {
		return input
	}

	if maxLength <= 3 {
		return input[:maxLength]
	}

	return input[:maxLength-3] + "..."
}

// SanitizeUsername sanitizes a username for display
// Complexity: O(n) where n is the length of username
func (s *Sanitizer) SanitizeUsername(username string) string {
	// HTML escape
	sanitized := html.EscapeString(username)

	// Remove control characters
	sanitized = RemoveControlCharacters(sanitized)

	// Truncate to reasonable length
	if len(sanitized) > 32 {
		sanitized = sanitized[:32]
	}

	return sanitized
}

// SanitizeMessage sanitizes a chat message
// Complexity: O(n) where n is the length of message
func (s *Sanitizer) SanitizeMessage(message string) string {
	// Remove null bytes
	sanitized := RemoveNullBytes(message)

	// Remove excessive whitespace
	sanitized = regexp.MustCompile(`\s+`).ReplaceAllString(sanitized, " ")

	// Trim
	sanitized = strings.TrimSpace(sanitized)

	// HTML escape to prevent XSS
	sanitized = html.EscapeString(sanitized)

	// Truncate if too long
	if len(sanitized) > 5000 {
		sanitized = TruncateString(sanitized, 5000)
	}

	return sanitized
}

// NormalizeWhitespace normalizes whitespace in input
// Replaces multiple spaces with single space
// Complexity: O(n) where n is the length of input
func NormalizeWhitespace(input string) string {
	// Replace multiple spaces with single space
	normalized := regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")

	// Trim leading and trailing whitespace
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// IsAlphanumeric checks if a string contains only alphanumeric characters
// Complexity: O(n) where n is the length of input
func IsAlphanumeric(input string) bool {
	for _, r := range input {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ContainsOnlyASCII checks if a string contains only ASCII characters
// Complexity: O(n) where n is the length of input
func ContainsOnlyASCII(input string) bool {
	for _, r := range input {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// EscapeShellArg escapes a string for safe use in shell commands
// NOTE: Prefer using exec.Command with separate arguments instead
// Complexity: O(n) where n is the length of input
func EscapeShellArg(arg string) string {
	// Surround with single quotes and escape any single quotes
	escaped := strings.ReplaceAll(arg, "'", "'\"'\"'")
	return "'" + escaped + "'"
}
