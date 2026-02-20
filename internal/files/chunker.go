package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// Chunker splits files into fixed-size chunks for P2P transfer.
// Complexity: O(n/c) where n = file size, c = chunk size.
type Chunker struct {
	chunkSize int
}

// NewChunker creates a new file chunker with the given chunk size.
func NewChunker(chunkSize int) *Chunker {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &Chunker{chunkSize: chunkSize}
}

// ChunkFile reads a file and returns chunks with their SHA-256 hashes.
// Also returns the full-file SHA-256 hash.
func (c *Chunker) ChunkFile(path string) ([]FileChunk, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("files: open: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("files: stat: %w", err)
	}
	if info.Size() > MaxFileSize {
		return nil, "", fmt.Errorf("files: file exceeds maximum size of %d bytes", MaxFileSize)
	}

	var chunks []FileChunk
	fileHasher := sha256.New()
	buf := make([]byte, c.chunkSize)
	idx := 0

	for {
		n, err := f.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			// Hash the chunk
			chunkHash := sha256.Sum256(data)
			// Feed into full-file hash
			fileHasher.Write(data)

			chunks = append(chunks, FileChunk{
				Index: idx,
				Data:  data,
				Hash:  hex.EncodeToString(chunkHash[:]),
			})
			idx++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("files: read chunk %d: %w", idx, err)
		}
	}

	fullHash := hex.EncodeToString(fileHasher.Sum(nil))
	return chunks, fullHash, nil
}

// ChunkCount returns how many chunks a file of the given size will produce.
func (c *Chunker) ChunkCount(sizeBytes int64) int {
	count := int(sizeBytes / int64(c.chunkSize))
	if sizeBytes%int64(c.chunkSize) != 0 {
		count++
	}
	return count
}

// Reassemble writes chunks back to a file in order. Verifies each chunk hash.
// Returns the full-file SHA-256 hash.
func (c *Chunker) Reassemble(chunks []FileChunk, destPath string) (string, error) {
	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("files: create dest: %w", err)
	}
	defer f.Close()

	fileHasher := sha256.New()

	for i, chunk := range chunks {
		if chunk.Index != i {
			return "", fmt.Errorf("files: expected chunk %d, got %d", i, chunk.Index)
		}

		// Verify chunk hash
		h := sha256.Sum256(chunk.Data)
		actual := hex.EncodeToString(h[:])
		if chunk.Hash != "" && actual != chunk.Hash {
			return "", fmt.Errorf("files: chunk %d hash mismatch: expected %s, got %s", i, chunk.Hash, actual)
		}

		if _, err := f.Write(chunk.Data); err != nil {
			return "", fmt.Errorf("files: write chunk %d: %w", i, err)
		}
		fileHasher.Write(chunk.Data)
	}

	return hex.EncodeToString(fileHasher.Sum(nil)), nil
}

// HashFile computes the SHA-256 hash of a file.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
