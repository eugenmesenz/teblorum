package model

import "time"

// Session представляет запись из таблицы sessions.
type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IsExpired возвращает true, если сессия истекла.
func (s Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
