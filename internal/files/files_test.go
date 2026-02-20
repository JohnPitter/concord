package files

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// --- Chunker Tests ---

func TestChunkerChunkFile(t *testing.T) {
	// Create a temp file with known content
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	data := bytes.Repeat([]byte("A"), 1024) // 1 KB
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	chunker := NewChunker(512) // 512-byte chunks
	chunks, fullHash, err := chunker.ChunkFile(path)
	if err != nil {
		t.Fatalf("chunk file: %v", err)
	}

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}

	// Verify chunk indices
	for i, c := range chunks {
		if c.Index != i {
			t.Errorf("chunk %d: expected index %d, got %d", i, i, c.Index)
		}
		if len(c.Data) != 512 {
			t.Errorf("chunk %d: expected 512 bytes, got %d", i, len(c.Data))
		}
		if c.Hash == "" {
			t.Errorf("chunk %d: empty hash", i)
		}
	}

	// Verify full hash
	h := sha256.Sum256(data)
	expectedHash := hex.EncodeToString(h[:])
	if fullHash != expectedHash {
		t.Errorf("full hash mismatch: expected %s, got %s", expectedHash, fullHash)
	}
}

func TestChunkerChunkCount(t *testing.T) {
	chunker := NewChunker(256 * 1024)

	tests := []struct {
		size     int64
		expected int
	}{
		{0, 0},
		{1, 1},
		{256 * 1024, 1},
		{256*1024 + 1, 2},
		{512 * 1024, 2},
		{1024 * 1024, 4},
	}

	for _, tt := range tests {
		got := chunker.ChunkCount(tt.size)
		if got != tt.expected {
			t.Errorf("ChunkCount(%d): expected %d, got %d", tt.size, tt.expected, got)
		}
	}
}

func TestChunkerReassemble(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.bin")
	data := bytes.Repeat([]byte("HELLO"), 200) // 1000 bytes
	if err := os.WriteFile(srcPath, data, 0600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	chunker := NewChunker(400) // 3 chunks: 400 + 400 + 200
	chunks, originalHash, err := chunker.ChunkFile(srcPath)
	if err != nil {
		t.Fatalf("chunk: %v", err)
	}

	// Reassemble into new file
	destPath := filepath.Join(tmpDir, "reassembled.bin")
	reassembledHash, err := chunker.Reassemble(chunks, destPath)
	if err != nil {
		t.Fatalf("reassemble: %v", err)
	}

	if originalHash != reassembledHash {
		t.Errorf("hash mismatch after reassembly: %s != %s", originalHash, reassembledHash)
	}

	// Verify content matches
	reassembled, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read reassembled: %v", err)
	}
	if !bytes.Equal(data, reassembled) {
		t.Error("reassembled content does not match original")
	}
}

func TestChunkerReassembleOutOfOrder(t *testing.T) {
	chunks := []FileChunk{
		{Index: 1, Data: []byte("b")},
		{Index: 0, Data: []byte("a")}, // wrong order
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "out.bin")
	chunker := NewChunker(1)

	_, err := chunker.Reassemble(chunks, destPath)
	if err == nil {
		t.Error("expected error for out-of-order chunks")
	}
}

func TestChunkerReassembleHashMismatch(t *testing.T) {
	chunks := []FileChunk{
		{Index: 0, Data: []byte("hello"), Hash: "badhash"},
	}

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "out.bin")
	chunker := NewChunker(1024)

	_, err := chunker.Reassemble(chunks, destPath)
	if err == nil {
		t.Error("expected error for hash mismatch")
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hashme.txt")
	data := []byte("hello world")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	h, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	expected := sha256.Sum256(data)
	if h != hex.EncodeToString(expected[:]) {
		t.Errorf("hash mismatch: got %s", h)
	}
}

// --- Scanner Tests ---

func TestScannerBlockedExtensions(t *testing.T) {
	scanner := NewScanner()

	blocked := []string{".exe", ".bat", ".cmd", ".com", ".msi", ".dll", ".ps1"}
	for _, ext := range blocked {
		result := scanner.ScanBytes([]byte("content"), "file"+ext)
		if result.Valid {
			t.Errorf("extension %s should be blocked", ext)
		}
	}
}

func TestScannerAllowedExtensions(t *testing.T) {
	scanner := NewScanner()

	// .js and .sh are explicitly allowed (false in blockedExts)
	allowed := []string{".js", ".sh", ".txt", ".go", ".png"}
	for _, ext := range allowed {
		// Use valid content that produces an allowed MIME type
		result := scanner.ScanBytes([]byte("plain text content"), "file"+ext)
		if !result.Valid {
			t.Errorf("extension %s should be allowed, error: %s", ext, result.Error)
		}
	}
}

func TestScannerEmptyFile(t *testing.T) {
	scanner := NewScanner()
	result := scanner.ScanBytes([]byte{}, "empty.txt")
	if result.Valid {
		t.Error("empty file should be invalid")
	}
	if result.Error != "file is empty" {
		t.Errorf("unexpected error: %s", result.Error)
	}
}

func TestScannerMaxSize(t *testing.T) {
	scanner := NewScanner()

	// File exactly at max should be valid (MIME type is text/plain for this content)
	data := bytes.Repeat([]byte("a"), int(MaxFileSize))
	result := scanner.ScanBytes(data, "big.txt")
	if !result.Valid {
		t.Errorf("file at max size should be valid, error: %s", result.Error)
	}

	// File over max should be invalid
	data = bytes.Repeat([]byte("a"), int(MaxFileSize)+1)
	result = scanner.ScanBytes(data, "toobig.txt")
	if result.Valid {
		t.Error("file over max size should be invalid")
	}
}

func TestScannerMIMEDetection(t *testing.T) {
	scanner := NewScanner()

	// Plain text
	result := scanner.ScanBytes([]byte("Hello, World!"), "test.txt")
	if !result.Valid {
		t.Errorf("text file should be valid: %s", result.Error)
	}
	if !strings.HasPrefix(result.MimeType, "text/") {
		t.Errorf("expected text/* MIME, got %s", result.MimeType)
	}

	// PNG magic bytes
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	png = append(png, bytes.Repeat([]byte{0}, 100)...) // pad
	result = scanner.ScanBytes(png, "image.png")
	if !result.Valid {
		t.Errorf("PNG should be valid: %s", result.Error)
	}
	if result.MimeType != "image/png" {
		t.Errorf("expected image/png, got %s", result.MimeType)
	}
}

func TestScannerFileValidation(t *testing.T) {
	tmpDir := t.TempDir()
	scanner := NewScanner()

	// Valid text file
	path := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}
	result := scanner.ScanFile(path)
	if !result.Valid {
		t.Errorf("text file should be valid: %s", result.Error)
	}

	// Non-existent file
	result = scanner.ScanFile(filepath.Join(tmpDir, "nonexistent.txt"))
	if result.Valid {
		t.Error("non-existent file should be invalid")
	}

	// Empty file
	emptyPath := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}
	result = scanner.ScanFile(emptyPath)
	if result.Valid {
		t.Error("empty file should be invalid")
	}
}

// --- Storage Tests ---

func TestLocalStorageSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	logger := testLogger()
	storage, err := NewLocalStorage(tmpDir, logger)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	data := []byte("test file content")
	path, err := storage.Save("test.txt", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	if !storage.Exists(path) {
		t.Error("file should exist after save")
	}

	rc, err := storage.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	defer rc.Close()

	loaded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(data, loaded) {
		t.Error("loaded content does not match saved content")
	}
}

func TestLocalStorageDelete(t *testing.T) {
	tmpDir := t.TempDir()
	logger := testLogger()
	storage, err := NewLocalStorage(tmpDir, logger)
	if err != nil {
		t.Fatal(err)
	}

	path, err := storage.Save("delete_me.txt", bytes.NewReader([]byte("delete me")))
	if err != nil {
		t.Fatal(err)
	}

	if err := storage.Delete(path); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if storage.Exists(path) {
		t.Error("file should not exist after delete")
	}
}

func TestLocalStorageUniquePath(t *testing.T) {
	tmpDir := t.TempDir()
	logger := testLogger()
	storage, err := NewLocalStorage(tmpDir, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Save same filename twice — should get unique paths
	path1, err := storage.Save("dup.txt", bytes.NewReader([]byte("first")))
	if err != nil {
		t.Fatal(err)
	}
	path2, err := storage.Save("dup.txt", bytes.NewReader([]byte("second")))
	if err != nil {
		t.Fatal(err)
	}

	if path1 == path2 {
		t.Error("expected different paths for duplicate filenames")
	}
}

func TestLocalStoragePathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	logger := testLogger()
	storage, err := NewLocalStorage(tmpDir, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt path traversal — filepath.Base should sanitize
	path, err := storage.Save("../../etc/passwd", bytes.NewReader([]byte("nope")))
	if err != nil {
		t.Fatal(err)
	}

	// File should be saved within baseDir, not in /etc/
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("file saved outside base dir: %s", path)
	}
}

// --- Constants Tests ---

func TestConstants(t *testing.T) {
	if MaxFileSize != 50<<20 {
		t.Errorf("MaxFileSize should be 50MB, got %d", MaxFileSize)
	}
	if DefaultChunkSize != 256<<10 {
		t.Errorf("DefaultChunkSize should be 256KB, got %d", DefaultChunkSize)
	}
}

func TestNewChunkerDefaultSize(t *testing.T) {
	c := NewChunker(0)
	if c.chunkSize != DefaultChunkSize {
		t.Errorf("expected default chunk size %d, got %d", DefaultChunkSize, c.chunkSize)
	}

	c = NewChunker(-1)
	if c.chunkSize != DefaultChunkSize {
		t.Errorf("expected default chunk size for negative input, got %d", c.chunkSize)
	}
}

// --- Test helpers ---

func testLogger() zerolog.Logger {
	return zerolog.New(os.Stderr).Level(zerolog.Disabled)
}
