package model

import (
	"database/sql"
	"time"
)

// Message представляет запись из таблицы messages.
type Message struct {
	ID         int64         `json:"id"`
	FromUserID int64         `json:"from_user_id"`
	ToUserID   int64         `json:"to_user_id"`
	ThreadID   sql.NullInt64 `json:"thread_id,omitempty"`
	Body       string        `json:"body"`
	ReadAt     sql.NullTime  `json:"read_at,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}

// ConversationSummary — сводка диалога для списка.
type ConversationSummary struct {
	WithUser    User    `json:"with_user"`
	LastMessage Message `json:"last_message"`
	UnreadCount int     `json:"unread_count"`
}
