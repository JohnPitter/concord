package friends

// RequestStatus represents the state of a friend request.
type RequestStatus string

const (
	StatusPending  RequestStatus = "pending"
	StatusAccepted RequestStatus = "accepted"
	StatusRejected RequestStatus = "rejected"
	StatusBlocked  RequestStatus = "blocked"
)

// FriendRequest represents a pending, accepted, rejected or blocked friend request.
type FriendRequest struct {
	ID         string        `json:"id"`
	SenderID   string        `json:"sender_id"`
	ReceiverID string        `json:"receiver_id"`
	Status     RequestStatus `json:"status"`
	CreatedAt  string        `json:"created_at"`
	UpdatedAt  string        `json:"updated_at"`
}

// FriendRequestView is the enriched view sent to the frontend.
type FriendRequestView struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Direction   string `json:"direction"` // "incoming" or "outgoing"
	CreatedAt   string `json:"createdAt"`
}

// FriendView represents a friend with profile info for the frontend.
type FriendView struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Status      string `json:"status"` // "online" | "offline"
}

// DMPaginationOpts controls cursor-based pagination for direct messages.
type DMPaginationOpts struct {
	After string `json:"after"` // Message ID to load messages after (newer)
	Limit int    `json:"limit"` // Max messages to return (default 50, max 100)
}

// DirectMessage represents a direct message between two friends.
type DirectMessage struct {
	ID         string `json:"id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
}
