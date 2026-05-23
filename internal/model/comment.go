package model

import (
	"database/sql"
	"time"
)

// Comment представляет запись из таблицы comments.
type Comment struct {
	ID        int64         `json:"id"`
	PostID    int64         `json:"post_id"`
	AuthorID  int64         `json:"author_id"`
	ParentID  sql.NullInt64 `json:"parent_id,omitempty"`
	Body      string        `json:"body"`
	CreatedAt time.Time     `json:"created_at"`
	DeletedAt sql.NullTime  `json:"deleted_at,omitempty"`
}

// CommentTreeNode — комментарий с вложенными ответами.
type CommentTreeNode struct {
	Comment
	Children []*CommentTreeNode `json:"children,omitempty"`
	Depth    int                `json:"depth"`
}

// CreateCommentRequest содержит данные для создания комментария.
type CreateCommentRequest struct {
	PostID   int64
	AuthorID int64
	ParentID *int64 // nil = корневой комментарий
	Body     string
}
