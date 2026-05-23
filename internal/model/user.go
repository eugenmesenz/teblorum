package model

import (
	"database/sql"
	"time"
)

// User представляет запись из таблицы users.
type User struct {
	ID          int64         `json:"id"`
	Username    string        `json:"username"`
	Email       string        `json:"-"`
	Password    string        `json:"-"`
	GoogleID    string        `json:"-"`
	Bio         string        `json:"bio"`
	Role        Role          `json:"role"`
	BannedUntil sql.NullTime  `json:"banned_until,omitempty"`
	BannedBy    sql.NullInt64 `json:"banned_by,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// CreateUserRequest содержит данные для создания пользователя.
type CreateUserRequest struct {
	Username string
	Email    string
	Password string
	GoogleID string
	Bio      string
	Role     Role
}

// UpdateUserRequest содержит изменяемые поля пользователя.
type UpdateUserRequest struct {
	Username string
	Bio      string
}
