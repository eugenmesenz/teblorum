package repo

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/teblorum/teblorum/internal/model"
)

// PostRepo — репозиторий публикаций.
type PostRepo struct {
	db *sql.DB
}

func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

func (r *PostRepo) Create(req model.CreatePostRequest) (*model.Post, error) {
	now := time.Now()
	commentsEnabled := 1
	if !req.CommentsEnabled {
		commentsEnabled = 0
	}
	result, err := r.db.Exec(`
		INSERT INTO posts (author_id, type, title, body, comments_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, req.AuthorID, req.Type, req.Title, req.Body, commentsEnabled, now, now)
	if err != nil {
		return nil, fmt.Errorf("create post: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *PostRepo) GetByID(id int64) (*model.Post, error) {
	row := r.db.QueryRow(`
		SELECT id, author_id, type, title, body, comments_enabled, created_at, updated_at, deleted_at
		FROM posts WHERE id = ?
	`, id)
	return scanPost(row)
}

func (r *PostRepo) GetFeed(page model.PaginationParams) ([]*model.Post, error) {
	rows, err := r.db.Query(`
		SELECT id, author_id, type, title, body, comments_enabled, created_at, updated_at, deleted_at
		FROM posts WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, page.Limit, page.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func (r *PostRepo) GetByType(postType string, page model.PaginationParams) ([]*model.Post, error) {
	rows, err := r.db.Query(`
		SELECT id, author_id, type, title, body, comments_enabled, created_at, updated_at, deleted_at
		FROM posts WHERE type = ? AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, postType, page.Limit, page.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func (r *PostRepo) GetByAuthorID(authorID int64, page model.PaginationParams) ([]*model.Post, error) {
	rows, err := r.db.Query(`
		SELECT id, author_id, type, title, body, comments_enabled, created_at, updated_at, deleted_at
		FROM posts WHERE author_id = ? AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, authorID, page.Limit, page.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func (r *PostRepo) Update(id int64, req model.UpdatePostRequest) error {
	query := "UPDATE posts SET updated_at = datetime('now')"
	args := []interface{}{}

	if req.Title != "" {
		query += ", title = ?"
		args = append(args, req.Title)
	}
	if req.Body != "" {
		query += ", body = ?"
		args = append(args, req.Body)
	}
	if req.CommentsEnabled != nil {
		v := 0
		if *req.CommentsEnabled {
			v = 1
		}
		query += ", comments_enabled = ?"
		args = append(args, v)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *PostRepo) SoftDelete(id int64) error {
	_, err := r.db.Exec(`
		UPDATE posts SET deleted_at = datetime('now'), updated_at = datetime('now') WHERE id = ?
	`, id)
	return err
}

func (r *PostRepo) CountByType(postType string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM posts WHERE type = ? AND deleted_at IS NULL", postType).Scan(&count)
	return count, err
}

func (r *PostRepo) CountFeed() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL").Scan(&count)
	return count, err
}

func scanPost(row *sql.Row) (*model.Post, error) {
	p := &model.Post{}
	var commentsEnabled int
	err := row.Scan(&p.ID, &p.AuthorID, &p.Type, &p.Title, &p.Body, &commentsEnabled,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post: %w", model.ErrNotFound)
		}
		return nil, err
	}
	p.CommentsEnabled = commentsEnabled == 1
	return p, nil
}

func scanPosts(rows *sql.Rows) ([]*model.Post, error) {
	var posts []*model.Post
	for rows.Next() {
		p := &model.Post{}
		var commentsEnabled int
		err := rows.Scan(&p.ID, &p.AuthorID, &p.Type, &p.Title, &p.Body, &commentsEnabled,
			&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
		if err != nil {
			return nil, err
		}
		p.CommentsEnabled = commentsEnabled == 1
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// Ensure time is used (for datetime('now') compatibility in tests)
var _ = time.Now
