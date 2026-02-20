package server

// Permission represents a specific action that can be performed on a server.
type Permission int

const (
	PermManageServer   Permission = iota // Rename, delete server
	PermManageChannels                   // Create, edit, delete channels
	PermManageMembers                    // Kick, change roles
	PermCreateInvite                     // Generate invite codes
	PermSendMessages                     // Send text messages
	PermManageMessages                   // Delete others' messages
)

// rolePermissions maps each role to its allowed permissions.
// Complexity: O(1) lookup
var rolePermissions = map[Role]map[Permission]bool{
	RoleOwner: {
		PermManageServer:   true,
		PermManageChannels: true,
		PermManageMembers:  true,
		PermCreateInvite:   true,
		PermSendMessages:   true,
		PermManageMessages: true,
	},
	RoleAdmin: {
		PermManageChannels: true,
		PermManageMembers:  true,
		PermCreateInvite:   true,
		PermSendMessages:   true,
		PermManageMessages: true,
	},
	RoleModerator: {
		PermCreateInvite:   true,
		PermSendMessages:   true,
		PermManageMessages: true,
	},
	RoleMember: {
		PermCreateInvite: true,
		PermSendMessages: true,
	},
}

// HasPermission checks if a role has the given permission.
// Complexity: O(1)
func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}

// RoleHierarchy returns the rank of a role (higher = more powerful).
// Complexity: O(1)
func RoleHierarchy(role Role) int {
	switch role {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleModerator:
		return 2
	case RoleMember:
		return 1
	default:
		return 0
	}
}
