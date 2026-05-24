package service

import (
	"testing"
	"time"

	"github.com/teblorum/teblorum/internal/model"
)

func TestBanUser(t *testing.T) {
	t.Run("moderator can ban user", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})
		user := seedServiceUser(t, db, map[string]interface{}{"username": "user", "email": "u@b.com"})

		err := svc.BanUser(user.ID, mod.ID, 24*time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("moderator cannot ban admin", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})
		admin := seedServiceUser(t, db, map[string]interface{}{"username": "admin", "email": "a@b.com", "role": model.RoleAdmin})

		err := svc.BanUser(admin.ID, mod.ID, 24*time.Hour)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("cannot ban yourself", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		u := seedServiceUser(t, db, nil)

		err := svc.BanUser(u.ID, u.ID, 24*time.Hour)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("admin can ban moderator", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		admin := seedServiceUser(t, db, map[string]interface{}{"username": "admin", "email": "a@b.com", "role": model.RoleAdmin})
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})

		err := svc.BanUser(mod.ID, admin.ID, 24*time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestUnbanUser(t *testing.T) {
	t.Run("moderator can unban user", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})
		user := seedServiceUser(t, db, map[string]interface{}{"username": "user", "email": "u@b.com"})

		// Ban first
		svc.BanUser(user.ID, mod.ID, 24*time.Hour)

		// Then unban
		err := svc.UnbanUser(user.ID, mod.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("moderator cannot unban admin", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})
		admin := seedServiceUser(t, db, map[string]interface{}{"username": "admin", "email": "a@b.com", "role": model.RoleAdmin})

		err := svc.UnbanUser(admin.ID, mod.ID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPromoteToModerator(t *testing.T) {
	t.Run("root can promote user", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		root := seedServiceUser(t, db, map[string]interface{}{"username": "root", "email": "r@b.com", "role": model.RoleRoot})
		user := seedServiceUser(t, db, map[string]interface{}{"username": "user", "email": "u@b.com"})

		err := svc.PromoteToModerator(user.ID, root.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("admin cannot promote", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		admin := seedServiceUser(t, db, map[string]interface{}{"username": "admin", "email": "a@b.com", "role": model.RoleAdmin})
		user := seedServiceUser(t, db, map[string]interface{}{"username": "user", "email": "u@b.com"})

		err := svc.PromoteToModerator(user.ID, admin.ID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("cannot promote yourself", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		root := seedServiceUser(t, db, map[string]interface{}{"username": "root", "email": "r@b.com", "role": model.RoleRoot})

		err := svc.PromoteToModerator(root.ID, root.ID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDemoteFromModerator(t *testing.T) {
	t.Run("root can demote moderator", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		root := seedServiceUser(t, db, map[string]interface{}{"username": "root", "email": "r@b.com", "role": model.RoleRoot})
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})

		err := svc.DemoteFromModerator(mod.ID, root.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("cannot demote user (not a moderator)", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewModerationService(db)
		root := seedServiceUser(t, db, map[string]interface{}{"username": "root", "email": "r@b.com", "role": model.RoleRoot})
		user := seedServiceUser(t, db, map[string]interface{}{"username": "user", "email": "u@b.com"})

		err := svc.DemoteFromModerator(user.ID, root.ID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
