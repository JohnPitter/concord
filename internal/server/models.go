package server

// Role represents a member's permission level within a server.
type Role string

const (
	RoleOwner     Role = "owner"
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
	RoleMember    Role = "member"
)

// Server represents a Concord server (guild).
type Server struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IconURL    string `json:"icon_url"`
	OwnerID    string `json:"owner_id"`
	InviteCode string `json:"invite_code"`
	CreatedAt  string `json:"created_at"` // ISO 8601
}

// Channel represents a text or voice channel within a server.
type Channel struct {
	ID        string `json:"id"`
	ServerID  string `json:"server_id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "text" or "voice"
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"` // ISO 8601
}

// Member represents a user's membership in a server.
type Member struct {
	ServerID string `json:"server_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar_url"`
	Role     Role   `json:"role"`
	JoinedAt string `json:"joined_at"` // ISO 8601
}

// InviteInfo is returned when inspecting an invite code.
type InviteInfo struct {
	ServerID   string `json:"server_id"`
	ServerName string `json:"server_name"`
	InviteCode string `json:"invite_code"`
	MemberCount int   `json:"member_count"`
}
