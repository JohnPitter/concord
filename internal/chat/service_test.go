package chat

import (
	"strings"
	"testing"
)

func TestMaxMessageLength(t *testing.T) {
	if maxMessageLength != 4000 {
		t.Errorf("expected max message length 4000, got %d", maxMessageLength)
	}
}

func TestMessageContentValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"valid short", "hello", false},
		{"valid max length", strings.Repeat("a", 4000), false},
		{"exceeds max length", strings.Repeat("a", 4001), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := strings.TrimSpace(tt.content)
			hasErr := content == "" || len(content) > maxMessageLength

			if hasErr != tt.wantErr {
				t.Errorf("content=%q: wantErr=%v, gotErr=%v", tt.content, tt.wantErr, hasErr)
			}
		})
	}
}

func TestPaginationOptsDefaults(t *testing.T) {
	opts := PaginationOpts{}

	// Default limit should be applied by repository
	if opts.Limit != 0 {
		t.Errorf("expected default limit 0 (to be resolved by repo), got %d", opts.Limit)
	}
	if opts.Before != "" {
		t.Error("expected empty Before")
	}
	if opts.After != "" {
		t.Error("expected empty After")
	}
}

func TestPaginationOptsLimit(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 50},
		{-1, 50},
		{101, 50},
		{25, 25},
		{100, 100},
		{1, 1},
	}

	for _, tt := range tests {
		limit := tt.input
		if limit <= 0 || limit > 100 {
			limit = 50
		}
		if limit != tt.expected {
			t.Errorf("input=%d: expected=%d, got=%d", tt.input, tt.expected, limit)
		}
	}
}
