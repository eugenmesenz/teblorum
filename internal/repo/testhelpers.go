package repo

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/teblorum/teblorum/internal/model"
)

// NewTestDB создаёт in-memory SQLite, накатывает миграцию и регистрирует cleanup.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	// PRAGMA для тестов
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			t.Fatalf("set pragma %s: %v", p, err)
		}
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// SeedUser создаёт пользователя в тестовой БД.
func SeedUser(t *testing.T, db *sql.DB, overrides map[string]interface{}) *model.User {
	t.Helper()

	username := "testuser"
	email := "test@example.com"
	password := "$2a$10$dummyhashdummyhashdummyhashdummyhashdummyhashdummyha"
	role := model.RoleUser
	bio := ""
	googleID := ""

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
	if v, ok := overrides["google_id"]; ok {
		googleID = v.(string)
	}

	var googleIDArg interface{} = nil
	if googleID != "" {
		googleIDArg = googleID
	}

	result, err := db.Exec(`
		INSERT INTO users (username, email, password, google_id, bio, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, username, email, password, googleIDArg, bio, string(role))
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, _ := result.LastInsertId()

	repo := NewUserRepo(db)
	user, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("get seeded user: %v", err)
	}
	return user
}

// SeedPost создаёт публикацию в тестовой БД.
func SeedPost(t *testing.T, db *sql.DB, overrides map[string]interface{}) *model.Post {
	t.Helper()

	title := "Test Post"
	body := "Test body content"
	postType := "article"
	commentsEnabled := 1
	var authorID int64 = 1

	if v, ok := overrides["title"]; ok {
		title = v.(string)
	}
	if v, ok := overrides["body"]; ok {
		body = v.(string)
	}
	if v, ok := overrides["type"]; ok {
		postType = v.(string)
	}
	if v, ok := overrides["author_id"]; ok {
		authorID = v.(int64)
	}
	if v, ok := overrides["comments_enabled"]; ok {
		if v.(bool) {
			commentsEnabled = 1
		} else {
			commentsEnabled = 0
		}
	}

	result, err := db.Exec(`
		INSERT INTO posts (author_id, type, title, body, comments_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, authorID, postType, title, body, commentsEnabled)
	if err != nil {
		t.Fatalf("seed post: %v", err)
	}
	id, _ := result.LastInsertId()

	repo := NewPostRepo(db)
	post, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("get seeded post: %v", err)
	}
	return post
}

// SeedComment создаёт комментарий в тестовой БД.
func SeedComment(t *testing.T, db *sql.DB, overrides map[string]interface{}) *model.Comment {
	t.Helper()

	body := "Test comment"
	var postID int64 = 1
	var authorID int64 = 1

	if v, ok := overrides["body"]; ok {
		body = v.(string)
	}
	if v, ok := overrides["post_id"]; ok {
		postID = v.(int64)
	}
	if v, ok := overrides["author_id"]; ok {
		authorID = v.(int64)
	}

	var parentIDArg interface{} = nil
	if v, ok := overrides["parent_id"]; ok {
		parentIDArg = v.(int64)
	}

	result, err := db.Exec(`
		INSERT INTO comments (post_id, author_id, parent_id, body, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))
	`, postID, authorID, parentIDArg, body)
	if err != nil {
		t.Fatalf("seed comment: %v", err)
	}
	id, _ := result.LastInsertId()

	repo := NewCommentRepo(db)
	comment, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("get seeded comment: %v", err)
	}
	return comment
}

// SeedSession создаёт сессию в тестовой БД.
func SeedSession(t *testing.T, db *sql.DB, userID int64) *model.Session {
	t.Helper()

	repo := NewSessionRepo(db)
	session, err := repo.Create(userID, 30)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return session
}

// Ensure fmt is used
var _ = fmt.Sprintf
