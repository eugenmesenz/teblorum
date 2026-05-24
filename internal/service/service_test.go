package service

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

// newTestDB создаёт in-memory SQLite для тестов сервисов.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			t.Fatalf("set pragma %s: %v", p, err)
		}
	}

	if err := repo.RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// seedServiceUser создаёт пользователя через репозиторий (для тестов).
func seedServiceUser(t *testing.T, db *sql.DB, overrides map[string]interface{}) *model.User {
	t.Helper()

	username := "testuser"
	email := "test@example.com"
	password := "$2a$10$dummyhashdummyhashdummyhashdummyhashdummyhashdummyha"
	role := model.RoleUser
	bio := ""

	if v, ok := overrides["username"]; ok {
		username = v.(string)
	}
	if v, ok := overrides["email"]; ok {
		email = v.(string)
	}
	if v, ok := overrides["password"]; ok {
		password = v.(string)
	}
	if v, ok := overrides["role"]; ok {
		role = v.(model.Role)
	}
	if v, ok := overrides["bio"]; ok {
		bio = v.(string)
	}

	result, err := db.Exec(`
		INSERT INTO users (username, email, password, bio, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, username, email, password, bio, string(role))
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, _ := result.LastInsertId()

	userRepo := repo.NewUserRepo(db)
	user, err := userRepo.GetByID(id)
	if err != nil {
		t.Fatalf("get seeded user: %v", err)
	}
	return user
}

// ensure fmt and time are used
var _ = fmt.Sprintf
var _ = time.Now