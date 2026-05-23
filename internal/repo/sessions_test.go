package repo

import (
	"errors"
	"testing"

	"github.com/teblorum/teblorum/internal/model"
)

func TestSessionRepo_Create(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSessionRepo(db)
	user := SeedUser(t, db, nil)

	session, err := repo.Create(user.ID, 30)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
	if len(session.ID) != 64 { // 32 bytes → hex = 64 chars
		t.Errorf("session ID length = %d, want 64", len(session.ID))
	}
	if session.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", session.UserID, user.ID)
	}
	if session.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}
}

func TestSessionRepo_GetByID(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSessionRepo(db)
	user := SeedUser(t, db, nil)

	t.Run("found", func(t *testing.T) {
		session, _ := repo.Create(user.ID, 30)
		got, err := repo.GetByID(session.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.ID != session.ID {
			t.Errorf("ID = %q, want %q", got.ID, session.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByID("nonexistent")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("expected %v, got %v", model.ErrNotFound, err)
		}
	})
}

func TestSessionRepo_DeleteByID(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSessionRepo(db)
	user := SeedUser(t, db, nil)
	session, _ := repo.Create(user.ID, 30)

	err := repo.DeleteByID(session.ID)
	if err != nil {
		t.Fatalf("DeleteByID() error = %v", err)
	}

	_, err = repo.GetByID(session.ID)
	if !errors.Is(err, model.ErrNotFound) {
		t.Error("session should be deleted")
	}
}

func TestSessionRepo_CleanExpired(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSessionRepo(db)
	user := SeedUser(t, db, nil)

	// Создаём сессию с истекшим сроком напрямую (TTL = 0 — не помогает, т.к. код добавляет даты)
	// Вставим просроченную сессию руками
	_, err := db.Exec(`
		INSERT INTO sessions (id, user_id, created_at, expires_at)
		VALUES ('expired-session', ?, datetime('now', '-2 days'), datetime('now', '-1 day'))
	`, user.ID)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	// Создаём нормальную сессию
	repo.Create(user.ID, 30)

	err = repo.CleanExpired()
	if err != nil {
		t.Fatalf("CleanExpired() error = %v", err)
	}

	// Просроченная должна быть удалена
	_, err = repo.GetByID("expired-session")
	if !errors.Is(err, model.ErrNotFound) {
		t.Error("expired session should be cleaned")
	}
}
