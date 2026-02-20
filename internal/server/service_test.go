package server

import (
	"testing"
)

func TestHasPermission_Owner(t *testing.T) {
	perms := []Permission{PermManageServer, PermManageChannels, PermManageMembers, PermCreateInvite, PermSendMessages, PermManageMessages}
	for _, p := range perms {
		if !HasPermission(RoleOwner, p) {
			t.Errorf("owner should have permission %d", p)
		}
	}
}

func TestHasPermission_Member(t *testing.T) {
	if !HasPermission(RoleMember, PermSendMessages) {
		t.Error("member should have PermSendMessages")
	}
	if !HasPermission(RoleMember, PermCreateInvite) {
		t.Error("member should have PermCreateInvite")
	}
	if HasPermission(RoleMember, PermManageServer) {
		t.Error("member should NOT have PermManageServer")
	}
	if HasPermission(RoleMember, PermManageChannels) {
		t.Error("member should NOT have PermManageChannels")
	}
	if HasPermission(RoleMember, PermManageMembers) {
		t.Error("member should NOT have PermManageMembers")
	}
}

func TestHasPermission_Admin(t *testing.T) {
	if HasPermission(RoleAdmin, PermManageServer) {
		t.Error("admin should NOT have PermManageServer")
	}
	if !HasPermission(RoleAdmin, PermManageChannels) {
		t.Error("admin should have PermManageChannels")
	}
	if !HasPermission(RoleAdmin, PermManageMembers) {
		t.Error("admin should have PermManageMembers")
	}
}

func TestHasPermission_InvalidRole(t *testing.T) {
	if HasPermission(Role("unknown"), PermSendMessages) {
		t.Error("unknown role should have no permissions")
	}
}

func TestRoleHierarchy(t *testing.T) {
	if RoleHierarchy(RoleOwner) <= RoleHierarchy(RoleAdmin) {
		t.Error("owner should outrank admin")
	}
	if RoleHierarchy(RoleAdmin) <= RoleHierarchy(RoleModerator) {
		t.Error("admin should outrank moderator")
	}
	if RoleHierarchy(RoleModerator) <= RoleHierarchy(RoleMember) {
		t.Error("moderator should outrank member")
	}
	if RoleHierarchy(RoleMember) <= RoleHierarchy(Role("unknown")) {
		t.Error("member should outrank unknown")
	}
}

func TestGenerateInviteCode(t *testing.T) {
	code1, err := generateInviteCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code1) != 8 {
		t.Errorf("expected 8-char code, got %d: %s", len(code1), code1)
	}

	code2, err := generateInviteCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code1 == code2 {
		t.Error("expected unique codes")
	}
}
