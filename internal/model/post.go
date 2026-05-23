package model

import (
	"database/sql"
	"time"
)

// Post представляет запись из таблицы posts.
type Post struct {
	ID              int64        `json:"id"`
	AuthorID        int64        `json:"author_id"`
	Type            string       `json:"type"`
	Title           string       `json:"title"`
	Body            string       `json:"body"`
	CommentsEnabled bool         `json:"comments_enabled"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
	DeletedAt       sql.NullTime `json:"deleted_at,omitempty"`
}

// CreatePostRequest содержит данные для создания публикации.
type CreatePostRequest struct {
	AuthorID        int64
	Type            string
	Title           string
	Body            string
	CommentsEnabled bool
}

// UpdatePostRequest содержит изменяемые поля публикации.
type UpdatePostRequest struct {
	Title           string
	Body            string
	CommentsEnabled *bool // nil = не менять
}
