package chat

// Message represents a text message in a channel.
type Message struct {
	ID        string  `json:"id"`
	ChannelID string  `json:"channel_id"`
	AuthorID  string  `json:"author_id"`
	Content   string  `json:"content"`
	Type      string  `json:"type"` // "text", "file", "system"
	EditedAt  *string `json:"edited_at,omitempty"` // ISO 8601
	CreatedAt string  `json:"created_at"`          // ISO 8601
	// Joined fields (from users table)
	AuthorName   string `json:"author_name,omitempty"`
	AuthorAvatar string `json:"author_avatar,omitempty"`
}

// PaginationOpts controls cursor-based pagination for message listing.
type PaginationOpts struct {
	Before string `json:"before"` // Message ID to load messages before (older)
	After  string `json:"after"`  // Message ID to load messages after (newer)
	Limit  int    `json:"limit"`  // Max messages to return (default 50, max 100)
}

// SearchResult represents a message found by full-text search.
type SearchResult struct {
	Message
	Snippet string `json:"snippet"` // Highlighted matching text
}
