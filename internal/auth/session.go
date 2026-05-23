package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

// GenerateSessionID создаёт случайный 32-байтный hex-ключ сессии.
func GenerateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateSession создаёт новую сессию в БД и возвращает её.
func CreateSession(db *sql.DB, userID int64, ttlDays int) (*model.Session, error) {
	r := repo.NewSessionRepo(db)
	return r.Create(userID, ttlDays)
}

// GetSession возвращает сессию по ID.
func GetSession(db *sql.DB, sessionID string) (*model.Session, error) {
	r := repo.NewSessionRepo(db)
	return r.GetByID(sessionID)
}

// RefreshSession обновляет expires_at (скользящее окно).
func RefreshSession(db *sql.DB, sessionID string, ttlDays int) error {
	r := repo.NewSessionRepo(db)
	return r.RefreshSession(sessionID, ttlDays)
}

// DeleteSession удаляет сессию.
func DeleteSession(db *sql.DB, sessionID string) error {
	r := repo.NewSessionRepo(db)
	return r.DeleteByID(sessionID)
}

// DeleteUserSessions удаляет все сессии пользователя.
func DeleteUserSessions(db *sql.DB, userID int64) error {
	r := repo.NewSessionRepo(db)
	return r.DeleteByUserID(userID)
}

// CleanExpiredSessions удаляет просроченные сессии.
func CleanExpiredSessions(db *sql.DB) error {
	r := repo.NewSessionRepo(db)
	return r.CleanExpired()
}

// GenerateResetToken создаёт токен для сброса пароля.
func GenerateResetToken(db *sql.DB, email string) (string, error) {
	token := GenerateSessionID() // 64 hex chars

	_, err := db.Exec(`
		INSERT INTO password_resets (email, token, expires_at)
		VALUES (?, ?, datetime('now', '+1 hour'))
	`, email, token)
	if err != nil {
		return "", fmt.Errorf("create reset token: %w", err)
	}
	return token, nil
}

// ValidateResetToken проверяет токен сброса и возвращает email.
func ValidateResetToken(db *sql.DB, token string) (string, error) {
	var email string
	var expiresAt time.Time

	err := db.QueryRow(`
		SELECT email, expires_at FROM password_resets
		WHERE token = ? AND expires_at > datetime('now')
	`, token).Scan(&email, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("reset token: %w", model.ErrNotFound)
		}
		return "", fmt.Errorf("validate reset token: %w", err)
	}
	return email, nil
}

// DeleteResetToken удаляет использованный токен сброса.
func DeleteResetToken(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM password_resets WHERE token = ?", token)
	return err
}
