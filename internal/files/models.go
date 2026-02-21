package files

// Attachment represents a file attached to a chat message.
type Attachment struct {
	ID        string `json:"id"`
	MessageID string `json:"message_id"`
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
	Hash      string `json:"hash"`
	LocalPath string `json:"local_path,omitempty"`
	CreatedAt string `json:"created_at"` // ISO 8601
}

// FileOffer is sent to a peer to initiate a file transfer.
type FileOffer struct {
	TransferID string `msgpack:"transfer_id" json:"transfer_id"`
	Filename   string `msgpack:"filename"    json:"filename"`
	SizeBytes  int64  `msgpack:"size_bytes"  json:"size_bytes"`
	MimeType   string `msgpack:"mime_type"   json:"mime_type"`
	Hash       string `msgpack:"hash"        json:"hash"`
	ChunkSize  int    `msgpack:"chunk_size"  json:"chunk_size"`
	ChunkCount int    `msgpack:"chunk_count" json:"chunk_count"`
	ChannelID  string `msgpack:"channel_id"  json:"channel_id"`
	SenderID   string `msgpack:"sender_id"   json:"sender_id"`
}

// FileAccept is sent to accept a file offer.
type FileAccept struct {
	TransferID string `msgpack:"transfer_id" json:"transfer_id"`
}

// FileChunk carries one chunk of file data.
type FileChunk struct {
	TransferID string `msgpack:"transfer_id" json:"transfer_id"`
	Index      int    `msgpack:"index"       json:"index"`
	Data       []byte `msgpack:"data"        json:"data"`
	Hash       string `msgpack:"hash"        json:"hash"` // SHA-256 of this chunk
}

// FileComplete signals all chunks have been sent.
type FileComplete struct {
	TransferID string `msgpack:"transfer_id" json:"transfer_id"`
	Hash       string `msgpack:"hash"        json:"hash"` // SHA-256 of entire file
}

// TransferState tracks an in-progress file transfer.
type TransferState struct {
	Offer          FileOffer
	ChunksReceived map[int]bool
	LocalPath      string
	Done           bool
	Error          error
}

// MaxFileSize is the maximum allowed file size (50 MB).
const MaxFileSize = 50 << 20

// DefaultChunkSize is the default chunk size for P2P transfer (256 KB).
const DefaultChunkSize = 256 << 10
