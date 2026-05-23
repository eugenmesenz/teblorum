package model

import (
	"testing"
)

func TestRoleLevel(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want int
	}{
		{"user", RoleUser, 0},
		{"moderator", RoleModerator, 1},
		{"admin", RoleAdmin, 2},
		{"root", RoleRoot, 3},
		{"unknown", Role("unknown"), -1},
		{"empty", Role(""), -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.Level(); got != tt.want {
				t.Errorf("Role(%q).Level() = %d, want %d", tt.role, got, tt.want)
			}
		})
	}
}

func TestCanActOn(t *testing.T) {
	tests := []struct {
		name   string
		actor  Role
		target Role
		want   bool
	}{
		// user vs all
		{"user cannot act on user", RoleUser, RoleUser, false},
		{"user cannot act on moderator", RoleUser, RoleModerator, false},
		{"user cannot act on admin", RoleUser, RoleAdmin, false},
		{"user cannot act on root", RoleUser, RoleRoot, false},

		// moderator vs all
		{"moderator can act on user", RoleModerator, RoleUser, true},
		{"moderator cannot act on moderator", RoleModerator, RoleModerator, false},
		{"moderator cannot act on admin", RoleModerator, RoleAdmin, false},
		{"moderator cannot act on root", RoleModerator, RoleRoot, false},

		// admin vs all
		{"admin can act on user", RoleAdmin, RoleUser, true},
		{"admin can act on moderator", RoleAdmin, RoleModerator, true},
		{"admin cannot act on admin", RoleAdmin, RoleAdmin, false},
		{"admin cannot act on root", RoleAdmin, RoleRoot, false},

		// root vs all
		{"root can act on user", RoleRoot, RoleUser, true},
		{"root can act on moderator", RoleRoot, RoleModerator, true},
		{"root can act on admin", RoleRoot, RoleAdmin, true},
		{"root cannot act on root", RoleRoot, RoleRoot, false},

		// unknown roles
		{"unknown actor returns false", Role("wtf"), RoleUser, false},
		{"unknown target returns false", RoleUser, Role("wtf"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanActOn(tt.actor, tt.target); got != tt.want {
				t.Errorf("CanActOn(%q, %q) = %v, want %v", tt.actor, tt.target, got, tt.want)
			}
		})
	}
}

func TestMinRole(t *testing.T) {
	tests := []struct {
		name     string
		userRole Role
		required Role
		want     bool
	}{
		{"user meets user", RoleUser, RoleUser, true},
		{"user fails moderator", RoleUser, RoleModerator, false},
		{"moderator meets user", RoleModerator, RoleUser, true},
		{"moderator meets moderator", RoleModerator, RoleModerator, true},
		{"moderator fails admin", RoleModerator, RoleAdmin, false},
		{"admin meets moderator", RoleAdmin, RoleModerator, true},
		{"admin meets admin", RoleAdmin, RoleAdmin, true},
		{"admin fails root", RoleAdmin, RoleRoot, false},
		{"root meets root", RoleRoot, RoleRoot, true},
		{"root meets user", RoleRoot, RoleUser, true},
		{"unknown role fails", Role("wtf"), RoleUser, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinRole(tt.userRole, tt.required); got != tt.want {
				t.Errorf("MinRole(%q, %q) = %v, want %v", tt.userRole, tt.required, got, tt.want)
			}
		})
	}
}
