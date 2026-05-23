package repo

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/teblorum/teblorum/internal/model"
)

// SessionRepo — репозиторий сессий.
type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

// GenerateSessionID создаёт случайный 32-байтный hex-ключ сессии.
func GenerateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (r *SessionRepo) Create(userID int64, ttlDays int) (*model.Session, error) {
	id := GenerateSessionID()
	expiresAt := time.Now().AddDate(0, 0, ttlDays)
	_, err := r.db.Exec(`
		INSERT INTO sessions (id, user_id, created_at, expires_at)
		VALUES (?, ?, datetime('now'), ?)
	`, id, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return r.GetByID(id)
}

func (r *SessionRepo) GetByID(id string) (*model.Session, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, created_at, expires_at FROM sessions WHERE id = ?
	`, id)
	s := &model.Session{}
	err := row.Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session: %w", model.ErrNotFound)
		}
		return nil, err
	}
	return s, nil
}

func (r *SessionRepo) DeleteByID(id string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (r *SessionRepo) DeleteByUserID(userID int64) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// RefreshSession обновляет expires_at (скользящее окно).
func (r *SessionRepo) RefreshSession(id string, ttlDays int) error {
	expiresAt := time.Now().AddDate(0, 0, ttlDays)
	_, err := r.db.Exec("UPDATE sessions SET expires_at = ? WHERE id = ?", expiresAt, id)
	return err
}

// CleanExpired удаляет просроченные сессии.
func (r *SessionRepo) CleanExpired() error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE expires_at < datetime('now')")
	return err
}
