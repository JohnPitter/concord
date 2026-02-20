package files

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Scanner validates files before upload/download.
// Checks MIME type, file extension, and size constraints.
type Scanner struct {
	maxSize      int64
	allowedMIMEs map[string]bool
	blockedExts  map[string]bool
}

// NewScanner creates a file scanner with default security policies.
func NewScanner() *Scanner {
	return &Scanner{
		maxSize: MaxFileSize,
		allowedMIMEs: map[string]bool{
			// Images
			"image/jpeg":    true,
			"image/png":     true,
			"image/gif":     true,
			"image/webp":    true,
			"image/svg+xml": true,
			// Documents
			"application/pdf":  true,
			"text/plain":       true,
			"text/csv":         true,
			"text/markdown":    true,
			"application/json": true,
			// Archives
			"application/zip":    true,
			"application/gzip":   true,
			"application/x-tar":  true,
			"application/x-7z-compressed": true,
			// Audio
			"audio/mpeg":  true,
			"audio/ogg":   true,
			"audio/wav":   true,
			"audio/webm":  true,
			"audio/flac":  true,
			// Video
			"video/mp4":  true,
			"video/webm": true,
			"video/ogg":  true,
			// Code / text
			"text/html":                true,
			"text/css":                 true,
			"text/javascript":          true,
			"application/javascript":   true,
			"application/octet-stream": true, // generic binary
		},
		blockedExts: map[string]bool{
			".exe":  true,
			".bat":  true,
			".cmd":  true,
			".com":  true,
			".msi":  true,
			".scr":  true,
			".pif":  true,
			".vbs":  true,
			".js":   false, // allowed in files context
			".ps1":  true,
			".sh":   false, // allowed
			".dll":  true,
			".sys":  true,
			".drv":  true,
			".cpl":  true,
			".inf":  true,
			".reg":  true,
		},
	}
}

// ScanResult holds the result of a file validation.
type ScanResult struct {
	Valid    bool   `json:"valid"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	Error    string `json:"error,omitempty"`
}

// ScanFile validates a file at the given path.
func (s *Scanner) ScanFile(path string) ScanResult {
	info, err := os.Stat(path)
	if err != nil {
		return ScanResult{Valid: false, Error: fmt.Sprintf("cannot stat file: %v", err)}
	}

	// Check size
	if info.Size() > s.maxSize {
		return ScanResult{
			Valid: false,
			Size:  info.Size(),
			Error: fmt.Sprintf("file exceeds maximum size of %d MB", s.maxSize>>20),
		}
	}

	if info.Size() == 0 {
		return ScanResult{Valid: false, Size: 0, Error: "file is empty"}
	}

	// Check blocked extensions
	ext := strings.ToLower(filepath.Ext(path))
	if blocked, exists := s.blockedExts[ext]; exists && blocked {
		return ScanResult{
			Valid: false,
			Size:  info.Size(),
			Error: fmt.Sprintf("file extension %s is not allowed", ext),
		}
	}

	// Detect MIME type from file content (first 512 bytes)
	mime, err := detectMIME(path)
	if err != nil {
		return ScanResult{Valid: false, Size: info.Size(), Error: fmt.Sprintf("cannot detect MIME type: %v", err)}
	}

	// Normalize MIME type (strip parameters like "; charset=utf-8")
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}

	// Check MIME whitelist
	if !s.allowedMIMEs[mime] {
		return ScanResult{
			Valid:    false,
			MimeType: mime,
			Size:     info.Size(),
			Error:    fmt.Sprintf("MIME type %s is not allowed", mime),
		}
	}

	return ScanResult{
		Valid:    true,
		MimeType: mime,
		Size:     info.Size(),
	}
}

// ScanBytes validates raw bytes (for inline validation without disk).
func (s *Scanner) ScanBytes(data []byte, filename string) ScanResult {
	size := int64(len(data))

	if size > s.maxSize {
		return ScanResult{Valid: false, Size: size, Error: fmt.Sprintf("file exceeds maximum size of %d MB", s.maxSize>>20)}
	}
	if size == 0 {
		return ScanResult{Valid: false, Size: 0, Error: "file is empty"}
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if blocked, exists := s.blockedExts[ext]; exists && blocked {
		return ScanResult{Valid: false, Size: size, Error: fmt.Sprintf("file extension %s is not allowed", ext)}
	}

	mime := http.DetectContentType(data)
	// Normalize MIME type (strip parameters like "; charset=utf-8")
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	if !s.allowedMIMEs[mime] {
		return ScanResult{Valid: false, MimeType: mime, Size: size, Error: fmt.Sprintf("MIME type %s is not allowed", mime)}
	}

	return ScanResult{Valid: true, MimeType: mime, Size: size}
}

// detectMIME reads the first 512 bytes to detect the MIME type.
func detectMIME(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return "", err
	}

	return http.DetectContentType(buf[:n]), nil
}
