package auth

import (
	"testing"

	"github.com/teblorum/teblorum/internal/repo"
)

func TestGenerateSessionID(t *testing.T) {
	id := GenerateSessionID()
	if len(id) != 64 {
		t.Errorf("length = %d, want 64", len(id))
	}

	id2 := GenerateSessionID()
	if id == id2 {
		t.Error("subsequent calls should generate different IDs")
	}
}

func TestCreateAndGetSession(t *testing.T) {
	db := repo.NewTestDB(t)
	user := repo.SeedUser(t, db, nil)

	session, err := CreateSession(db, user.ID, 30)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if session.ID == "" {
		t.Fatal("session ID should not be empty")
	}
	if session.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", session.UserID, user.ID)
	}
	if session.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}

	// GetSession
	got, err := GetSession(db, session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.ID != session.ID {
		t.Errorf("ID = %q, want %q", got.ID, session.ID)
	}
}

func TestRefreshSession(t *testing.T) {
	db := repo.NewTestDB(t)
	user := repo.SeedUser(t, db, nil)

	session, _ := CreateSession(db, user.ID, 30)
	originalExpires := session.ExpiresAt

	err := RefreshSession(db, session.ID, 30)
	if err != nil {
		t.Fatalf("RefreshSession() error = %v", err)
	}

	updated, _ := GetSession(db, session.ID)
	if !updated.ExpiresAt.After(originalExpires) {
		t.Error("ExpiresAt should be extended after refresh")
	}
}

func TestDeleteSession(t *testing.T) {
	db := repo.NewTestDB(t)
	user := repo.SeedUser(t, db, nil)

	session, _ := CreateSession(db, user.ID, 30)

	err := DeleteSession(db, session.ID)
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	_, err = GetSession(db, session.ID)
	if err == nil {
		t.Error("session should be deleted")
	}
}

func TestGenerateAndValidateResetToken(t *testing.T) {
	db := repo.NewTestDB(t)

	token, err := GenerateResetToken(db, "user@example.com")
	if err != nil {
		t.Fatalf("GenerateResetToken() error = %v", err)
	}
	if len(token) != 64 {
		t.Errorf("token length = %d, want 64", len(token))
	}

	email, err := ValidateResetToken(db, token)
	if err != nil {
		t.Fatalf("ValidateResetToken() error = %v", err)
	}
	if email != "user@example.com" {
		t.Errorf("email = %q, want %q", email, "user@example.com")
	}
}

func TestValidateExpiredResetToken(t *testing.T) {
	db := repo.NewTestDB(t)

	// Создаём просроченный токен напрямую
	_, err := db.Exec(`
		INSERT INTO password_resets (email, token, expires_at)
		VALUES ('test@example.com', 'expired-token', datetime('now', '-1 hour'))
	`)
	if err != nil {
		t.Fatalf("insert expired token: %v", err)
	}

	_, err = ValidateResetToken(db, "expired-token")
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestDeleteResetToken(t *testing.T) {
	db := repo.NewTestDB(t)

	token, _ := GenerateResetToken(db, "user@example.com")

	err := DeleteResetToken(db, token)
	if err != nil {
		t.Fatalf("DeleteResetToken() error = %v", err)
	}

	_, err = ValidateResetToken(db, token)
	if err == nil {
		t.Error("token should be deleted")
	}
}

func TestCleanExpiredSessions(t *testing.T) {
	db := repo.NewTestDB(t)
	user := repo.SeedUser(t, db, nil)

	// Вставляем просроченную сессию напрямую
	_, err := db.Exec(`
		INSERT INTO sessions (id, user_id, created_at, expires_at)
		VALUES ('expired-session', ?, datetime('now', '-2 days'), datetime('now', '-1 day'))
	`, user.ID)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	// Создаём нормальную сессию
	CreateSession(db, user.ID, 30)

	err = CleanExpiredSessions(db)
	if err != nil {
		t.Fatalf("CleanExpiredSessions() error = %v", err)
	}

	// Просроченная должна быть удалена
	_, err = GetSession(db, "expired-session")
	if err == nil {
		t.Error("expired session should be cleaned")
	}
}