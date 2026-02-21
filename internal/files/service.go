package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service orchestrates file upload, download, validation, and chunking.
type Service struct {
	repo      *Repository
	storage   Storage
	scanner   *Scanner
	chunker   *Chunker
	transfers sync.Map // transferID -> *TransferState
	logger    zerolog.Logger
}

// NewService creates a new file service.
func NewService(repo *Repository, storage Storage, logger zerolog.Logger) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		scanner: NewScanner(),
		chunker: NewChunker(DefaultChunkSize),
		logger:  logger.With().Str("component", "file_service").Logger(),
	}
}

// Upload validates and stores a file, creating an attachment record.
func (s *Service) Upload(ctx context.Context, messageID, filename string, data []byte) (*Attachment, error) {
	// Validate file
	result := s.scanner.ScanBytes(data, filename)
	if !result.Valid {
		return nil, fmt.Errorf("files: validation failed: %s", result.Error)
	}

	// Write to temp file for hashing
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, "concord_upload_"+uuid.NewString())
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return nil, fmt.Errorf("files: write temp: %w", err)
	}
	defer os.Remove(tmpPath)

	// Hash the file
	hash, err := HashFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("files: hash: %w", err)
	}

	// Check for deduplication — if we already have a file with this hash, reuse it
	existing, err := s.repo.GetByHash(ctx, hash)
	if err == nil && existing != nil && s.storage.Exists(existing.LocalPath) {
		// Reuse existing file, create new attachment record
		att := &Attachment{
			ID:        uuid.NewString(),
			MessageID: messageID,
			Filename:  filepath.Base(filename),
			SizeBytes: int64(len(data)),
			MimeType:  result.MimeType,
			Hash:      hash,
			LocalPath: existing.LocalPath,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if err := s.repo.Save(ctx, att); err != nil {
			return nil, err
		}
		s.logger.Info().
			Str("attachment_id", att.ID).
			Str("hash", hash).
			Msg("file deduplicated")
		return att, nil
	}

	// Save to storage
	f, err := os.Open(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("files: open temp: %w", err)
	}
	defer f.Close()

	localPath, err := s.storage.Save(filepath.Base(filename), f)
	if err != nil {
		return nil, err
	}

	att := &Attachment{
		ID:        uuid.NewString(),
		MessageID: messageID,
		Filename:  filepath.Base(filename),
		SizeBytes: int64(len(data)),
		MimeType:  result.MimeType,
		Hash:      hash,
		LocalPath: localPath,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.repo.Save(ctx, att); err != nil {
		s.storage.Delete(localPath)
		return nil, err
	}

	s.logger.Info().
		Str("attachment_id", att.ID).
		Str("filename", att.Filename).
		Int64("size", att.SizeBytes).
		Str("mime", att.MimeType).
		Msg("file uploaded")

	return att, nil
}

// Download returns the file data for an attachment.
func (s *Service) Download(ctx context.Context, attachmentID string) ([]byte, *Attachment, error) {
	att, err := s.repo.GetByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("files: attachment not found: %w", err)
	}

	rc, err := s.storage.Load(att.LocalPath)
	if err != nil {
		return nil, nil, fmt.Errorf("files: load file: %w", err)
	}
	defer rc.Close()

	data, err := os.ReadFile(att.LocalPath)
	if err != nil {
		return nil, nil, fmt.Errorf("files: read file: %w", err)
	}

	return data, att, nil
}

// GetAttachments returns all attachments for a message.
func (s *Service) GetAttachments(ctx context.Context, messageID string) ([]*Attachment, error) {
	return s.repo.GetByMessageID(ctx, messageID)
}

// DeleteAttachment removes an attachment and its file from storage.
func (s *Service) DeleteAttachment(ctx context.Context, attachmentID string) error {
	att, err := s.repo.GetByID(ctx, attachmentID)
	if err != nil {
		return fmt.Errorf("files: attachment not found: %w", err)
	}

	// Check if other attachments reference the same file
	other, _ := s.repo.GetByHash(ctx, att.Hash)
	if other != nil && other.ID != att.ID {
		// Another attachment uses this file — only delete the record
		return s.repo.Delete(ctx, attachmentID)
	}

	// Delete file from storage
	if err := s.storage.Delete(att.LocalPath); err != nil {
		s.logger.Warn().Err(err).Str("path", att.LocalPath).Msg("failed to delete file from storage")
	}

	return s.repo.Delete(ctx, attachmentID)
}

// PrepareOffer creates a FileOffer for P2P transfer of an attachment.
func (s *Service) PrepareOffer(ctx context.Context, attachmentID, channelID, senderID string) (*FileOffer, error) {
	att, err := s.repo.GetByID(ctx, attachmentID)
	if err != nil {
		return nil, fmt.Errorf("files: attachment not found: %w", err)
	}

	chunkCount := s.chunker.ChunkCount(att.SizeBytes)

	return &FileOffer{
		TransferID: uuid.NewString(),
		Filename:   att.Filename,
		SizeBytes:  att.SizeBytes,
		MimeType:   att.MimeType,
		Hash:       att.Hash,
		ChunkSize:  DefaultChunkSize,
		ChunkCount: chunkCount,
		ChannelID:  channelID,
		SenderID:   senderID,
	}, nil
}

// ChunkAttachment splits an attachment into chunks for P2P transfer.
func (s *Service) ChunkAttachment(ctx context.Context, attachmentID, transferID string) ([]FileChunk, error) {
	att, err := s.repo.GetByID(ctx, attachmentID)
	if err != nil {
		return nil, fmt.Errorf("files: attachment not found: %w", err)
	}

	chunks, _, err := s.chunker.ChunkFile(att.LocalPath)
	if err != nil {
		return nil, err
	}

	// Stamp transfer ID on each chunk
	for i := range chunks {
		chunks[i].TransferID = transferID
	}

	return chunks, nil
}

// StartReceive begins tracking an incoming file transfer.
func (s *Service) StartReceive(offer FileOffer) {
	state := &TransferState{
		Offer:          offer,
		ChunksReceived: make(map[int]bool),
	}
	s.transfers.Store(offer.TransferID, state)
	s.logger.Info().
		Str("transfer_id", offer.TransferID).
		Str("filename", offer.Filename).
		Int64("size", offer.SizeBytes).
		Msg("file transfer started")
}

// ReceiveChunk stores a received chunk and returns true if all chunks are received.
func (s *Service) ReceiveChunk(chunk FileChunk) (bool, error) {
	val, ok := s.transfers.Load(chunk.TransferID)
	if !ok {
		return false, fmt.Errorf("files: unknown transfer %s", chunk.TransferID)
	}
	state := val.(*TransferState)
	state.ChunksReceived[chunk.Index] = true

	allReceived := len(state.ChunksReceived) >= state.Offer.ChunkCount
	return allReceived, nil
}

// CompleteReceive finalizes a file transfer.
func (s *Service) CompleteReceive(ctx context.Context, transferID, messageID string, chunks []FileChunk) (*Attachment, error) {
	val, ok := s.transfers.Load(transferID)
	if !ok {
		return nil, fmt.Errorf("files: unknown transfer %s", transferID)
	}
	state := val.(*TransferState)
	defer s.transfers.Delete(transferID)

	// Reassemble file
	tmpPath := filepath.Join(os.TempDir(), "concord_recv_"+transferID)
	fullHash, err := s.chunker.Reassemble(chunks, tmpPath)
	if err != nil {
		return nil, fmt.Errorf("files: reassemble: %w", err)
	}
	defer os.Remove(tmpPath)

	// Verify full-file hash
	if state.Offer.Hash != "" && fullHash != state.Offer.Hash {
		return nil, fmt.Errorf("files: hash mismatch: expected %s, got %s", state.Offer.Hash, fullHash)
	}

	// Read reassembled file and store
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("files: read reassembled: %w", err)
	}

	return s.Upload(ctx, messageID, state.Offer.Filename, data)
}

// ScanFile validates a file at the given path.
func (s *Service) ScanFile(path string) ScanResult {
	return s.scanner.ScanFile(path)
}

