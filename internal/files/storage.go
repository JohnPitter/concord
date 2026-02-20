package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// Storage abstracts file storage backends.
// Currently supports local filesystem; S3 can be added as a future implementation.
type Storage interface {
	Save(filename string, r io.Reader) (path string, err error)
	Load(path string) (io.ReadCloser, error)
	Delete(path string) error
	Exists(path string) bool
}

// LocalStorage implements Storage using the local filesystem.
type LocalStorage struct {
	baseDir string
	logger  zerolog.Logger
}

// NewLocalStorage creates a local file storage rooted at baseDir.
func NewLocalStorage(baseDir string, logger zerolog.Logger) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0750); err != nil {
		return nil, fmt.Errorf("files: create storage dir: %w", err)
	}
	return &LocalStorage{
		baseDir: baseDir,
		logger:  logger.With().Str("component", "file_storage").Logger(),
	}, nil
}

// Save writes a file to local storage. Returns the full path.
func (s *LocalStorage) Save(filename string, r io.Reader) (string, error) {
	// Sanitize filename to prevent path traversal
	clean := filepath.Base(filename)
	dest := filepath.Join(s.baseDir, clean)

	// Avoid overwriting: append suffix if exists
	dest = s.uniquePath(dest)

	f, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("files: create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, io.LimitReader(r, MaxFileSize+1))
	if err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("files: write file: %w", err)
	}
	if written > MaxFileSize {
		os.Remove(dest)
		return "", fmt.Errorf("files: file exceeds maximum size of %d bytes", MaxFileSize)
	}

	s.logger.Info().
		Str("path", dest).
		Int64("size", written).
		Msg("file saved")

	return dest, nil
}

// Load opens a file for reading.
func (s *LocalStorage) Load(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("files: open file: %w", err)
	}
	return f, nil
}

// Delete removes a file from storage.
func (s *LocalStorage) Delete(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("files: delete file: %w", err)
	}
	return nil
}

// Exists checks if a file exists.
func (s *LocalStorage) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// uniquePath appends a numeric suffix if the file already exists.
func (s *LocalStorage) uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return path
}
