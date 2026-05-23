package repo

import (
	"errors"
	"testing"
	"time"

	"github.com/teblorum/teblorum/internal/model"
)

func TestUserRepo_Create(t *testing.T) {
	db := NewTestDB(t)
	repo := NewUserRepo(db)

	t.Run("success", func(t *testing.T) {
		user, err := repo.Create(model.CreateUserRequest{
			Username: "alice",
			Email:    "alice@example.com",
			Password: "hashedpassword",
			Bio:      "Hello!",
			Role:     model.RoleUser,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if user.Username != "alice" {
			t.Errorf("Username = %q, want %q", user.Username, "alice")
		}
		if user.Email != "alice@example.com" {
			t.Errorf("Email = %q, want %q", user.Email, "alice@example.com")
		}
		if user.Role != model.RoleUser {
			t.Errorf("Role = %q, want %q", user.Role, model.RoleUser)
		}
		if user.ID == 0 {
			t.Error("ID should not be zero")
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		_, err := repo.Create(model.CreateUserRequest{
			Username: "alice2",
			Email:    "alice@example.com",
			Password: "pw",
			Role:     model.RoleUser,
		})
		if err == nil {
			t.Fatal("expected error for duplicate email")
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		_, err := repo.Create(model.CreateUserRequest{
			Username: "alice",
			Email:    "alice2@example.com",
			Password: "pw",
			Role:     model.RoleUser,
		})
		if err == nil {
			t.Fatal("expected error for duplicate username")
		}
	})
}

func TestUserRepo_GetByID(t *testing.T) {
	db := NewTestDB(t)
	repo := NewUserRepo(db)
	user := SeedUser(t, db, nil)

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.Username != user.Username {
			t.Errorf("Username = %q, want %q", got.Username, user.Username)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByID(99999)
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("expected %v, got %v", model.ErrNotFound, err)
		}
	})
}

func TestUserRepo_GetByEmail(t *testing.T) {
	db := NewTestDB(t)
	repo := NewUserRepo(db)
	user := SeedUser(t, db, map[string]interface{}{"email": "findme@example.com"})

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetByEmail("findme@example.com")
		if err != nil {
			t.Fatalf("GetByEmail() error = %v", err)
		}
		if got.ID != user.ID {
			t.Errorf("ID = %d, want %d", got.ID, user.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByEmail("nobody@example.com")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("expected %v, got %v", model.ErrNotFound, err)
		}
	})
}

func TestUserRepo_GetByUsername(t *testing.T) {
	db := NewTestDB(t)
	repo := NewUserRepo(db)
	user := SeedUser(t, db, map[string]interface{}{"username": "uniqueuser"})

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetByUsername("uniqueuser")
		if err != nil {
			t.Fatalf("GetByUsername() error = %v", err)
		}
		if got.ID != user.ID {
			t.Errorf("ID = %d, want %d", got.ID, user.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByUsername("nonexistent")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("expected %v, got %v", model.ErrNotFound, err)
		}
	})
}

func TestUserRepo_GetByGoogleID(t *testing.T) {
	db := NewTestDB(t)
	repo := NewUserRepo(db)
	user := SeedUser(t, db, map[string]interface{}{
		"google_id": "google-123",
		"email":     "google@example.com",
	})

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetByGoogleID("google-123")
		if err != nil {
			t.Fatalf("GetByGoogleID() error = %v", err)
		}
		if got.ID != user.ID {
			t.Errorf("ID = %d, want %d", got.ID, user.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByGoogleID("nonexistent")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("expected %v, got %v", model.ErrNotFound, err)
		}
	})
}

func TestUserRepo_Update(t *testing.T) {
	db := NewTestDB(t)
	repo := NewUserRepo(db)
	user := SeedUser(t, db, nil)

	err := repo.Update(user.ID, model.UpdateUserRequest{
		Username: "newname",
		Bio:      "updated bio",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, _ := repo.GetByID(user.ID)
	if updated.Username != "newname" {
		t.Errorf("Username = %q, want %q", updated.Username, "newname")
	}
	if updated.Bio != "updated bio" {
		t.Errorf("Bio = %q, want %q", updated.Bio, "updated bio")
	}
}

func TestUserRepo_Ban(t *testing.T) {
	db := NewTestDB(t)
	repo := NewUserRepo(db)
	user := SeedUser(t, db, nil)
	admin := SeedUser(t, db, map[string]interface{}{
		"username": "admin",
		"email":    "admin@example.com",
		"role":     model.RoleAdmin,
	})

	t.Run("set ban", func(t *testing.T) {
		until := time.Now().Add(24 * time.Hour)
		err := repo.SetBan(user.ID, until, admin.ID)
		if err != nil {
			t.Fatalf("SetBan() error = %v", err)
		}
		banned, _ := repo.GetByID(user.ID)
		if !banned.BannedUntil.Valid {
			t.Fatal("BannedUntil should be set")
		}
		if banned.BannedBy.Int64 != admin.ID {
			t.Errorf("BannedBy = %d, want %d", banned.BannedBy.Int64, admin.ID)
		}
	})

	t.Run("remove ban", func(t *testing.T) {
		err := repo.RemoveBan(user.ID)
		if err != nil {
			t.Fatalf("RemoveBan() error = %v", err)
		}
		unbanned, _ := repo.GetByID(user.ID)
		if unbanned.BannedUntil.Valid {
			t.Error("BannedUntil should be null after unban")
		}
	})
}

func TestUserRepo_Exists(t *testing.T) {
	db := NewTestDB(t)
	repo := NewUserRepo(db)
	SeedUser(t, db, map[string]interface{}{"email": "exists@example.com", "username": "exists"})

	t.Run("email exists", func(t *testing.T) {
		ok, err := repo.ExistsByEmail("exists@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected email to exist")
		}
	})

	t.Run("email not exists", func(t *testing.T) {
		ok, err := repo.ExistsByEmail("no@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected email not to exist")
		}
	})

	t.Run("username exists", func(t *testing.T) {
		ok, err := repo.ExistsByUsername("exists")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected username to exist")
		}
	})

	t.Run("username not exists", func(t *testing.T) {
		ok, err := repo.ExistsByUsername("noexists")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected username not to exist")
		}
	})
}
