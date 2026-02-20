package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSanitizer(t *testing.T) {
	s := NewSanitizer()
	assert.NotNil(t, s)
	assert.NotNil(t, s.AllowedTags)
	assert.Equal(t, 50000, s.MaxLength)
	assert.True(t, s.AllowedTags["b"])
	assert.True(t, s.AllowedTags["i"])
	assert.False(t, s.AllowedTags["script"])
}

func TestSanitizer_SanitizeHTML(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes script tags",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "allows safe tags",
			input:    "Hello <b>world</b>!",
			expected: "Hello <b>world</b>!",
		},
		{
			name:  "escapes tags with attributes",
			input: "<b onclick='alert(1)'>text</b>",
			// Tags with attributes are not recognized as allowed tags (which are plain <b></b>),
			// so they remain escaped for safety
			expected: "&lt;b onclick=&#39;alert(1)&#39;&gt;text</b>",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_StripHTML(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes all tags",
			input:    "<p>Hello <b>world</b>!</p>",
			expected: "Hello world!",
		},
		{
			name:     "handles script tags",
			input:    "<script>alert('xss')</script>Text",
			expected: "alert('xss')Text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.StripHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_SanitizeMarkdown(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "removes javascript URLs",
			input:    "[Click me](javascript:alert(1))",
			contains: "#",
		},
		{
			name:     "removes data URLs",
			input:    "[Image](data:text/html,<script>alert(1)</script>)",
			contains: "#",
		},
		{
			name:     "preserves safe links",
			input:    "[Link](https://example.com)",
			contains: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeMarkdown(tt.input)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestSanitizer_SanitizeFilename(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes path separators",
			input:    "../../../etc/passwd",
			expected: "etcpasswd",
		},
		{
			name:     "removes null bytes",
			input:    "file\x00.txt",
			expected: "file.txt",
		},
		{
			name:     "normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "empty becomes default",
			input:    "",
			expected: "untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_SanitizePath(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name       string
		input      string
		notContain string
	}{
		{
			name:       "removes directory traversal",
			input:      "../../../etc/passwd",
			notContain: "..",
		},
		{
			name:       "removes null bytes",
			input:      "path/\x00/file.txt",
			notContain: "\x00",
		},
		{
			name:       "normalizes slashes",
			input:      "path\\to\\file",
			notContain: "\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizePath(tt.input)
			assert.NotContains(t, result, tt.notContain)
		})
	}
}

func TestRemoveControlCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "keeps newlines and tabs",
			input:    "text\n\twith formatting",
			expected: "text\n\twith formatting",
		},
		{
			name:     "removes control chars",
			input:    "text\x01\x02here",
			expected: "texthere",
		},
		{
			name:     "normal text unchanged",
			input:    "Hello World",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveControlCharacters(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveNullBytes(t *testing.T) {
	input := "text\x00with\x00nulls"
	result := RemoveNullBytes(input)
	assert.Equal(t, "textwithnulls", result)
	assert.NotContains(t, result, "\x00")
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "no truncation needed",
			input:     "short",
			maxLength: 10,
			expected:  "short",
		},
		{
			name:      "truncates with ellipsis",
			input:     "this is a very long string",
			maxLength: 10,
			expected:  "this is...",
		},
		{
			name:      "very short max",
			input:     "test",
			maxLength: 2,
			expected:  "te",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), tt.maxLength)
		})
	}
}

func TestSanitizer_SanitizeUsername(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "escapes HTML",
			input: "<script>alert(1)</script>",
		},
		{
			name:  "normal username",
			input: "john_doe",
		},
		{
			name:  "with special chars",
			input: "user@example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeUsername(tt.input)
			assert.NotContains(t, result, "<script>")
			assert.LessOrEqual(t, len(result), 32)
		})
	}
}

func TestSanitizer_SanitizeMessage(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name       string
		input      string
		notContain string
	}{
		{
			name:       "removes null bytes",
			input:      "message\x00with null",
			notContain: "\x00",
		},
		{
			name:       "escapes HTML",
			input:      "<script>alert(1)</script>",
			notContain: "<script>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeMessage(tt.input)
			assert.NotContains(t, result, tt.notContain)
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "multiple spaces",
			input:    "text  with   spaces",
			expected: "text with spaces",
		},
		{
			name:     "leading and trailing",
			input:    "  text  ",
			expected: "text",
		},
		{
			name:     "normal text",
			input:    "normal text",
			expected: "normal text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWhitespace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc123", true},
		{"onlyletters", true},
		{"123456", true},
		{"with space", false},
		{"with-hyphen", false},
		{"with_underscore", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsAlphanumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsOnlyASCII(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"ASCII text", true},
		{"123", true},
		{"émoji 🎮", false},
		{"日本語", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ContainsOnlyASCII(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeShellArg(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"normal arg", "file.txt"},
		{"with spaces", "my file.txt"},
		{"with quotes", "file's.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeShellArg(tt.input)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "'")
		})
	}
}
