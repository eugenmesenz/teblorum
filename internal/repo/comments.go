package repo

import (
	"database/sql"
	"fmt"

	"github.com/teblorum/teblorum/internal/model"
)

// CommentRepo — репозиторий комментариев.
type CommentRepo struct {
	db *sql.DB
}

func NewCommentRepo(db *sql.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

func (r *CommentRepo) Create(req model.CreateCommentRequest) (*model.Comment, error) {
	var parentID interface{}
	if req.ParentID != nil {
		parentID = *req.ParentID
	}
	result, err := r.db.Exec(`
		INSERT INTO comments (post_id, author_id, parent_id, body, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))
	`, req.PostID, req.AuthorID, parentID, req.Body)
	if err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *CommentRepo) GetByID(id int64) (*model.Comment, error) {
	row := r.db.QueryRow(`
		SELECT id, post_id, author_id, parent_id, body, created_at, deleted_at
		FROM comments WHERE id = ?
	`, id)
	return scanComment(row)
}

// GetByPostID возвращает дерево комментариев для поста через рекурсивный CTE.
func (r *CommentRepo) GetByPostID(postID int64) ([]*model.CommentTreeNode, error) {
	rows, err := r.db.Query(`
		WITH RECURSIVE tree AS (
			SELECT id, post_id, author_id, parent_id, body, created_at, deleted_at, 0 AS depth
			FROM comments
			WHERE post_id = ? AND parent_id IS NULL AND deleted_at IS NULL
			UNION ALL
			SELECT c.id, c.post_id, c.author_id, c.parent_id, c.body, c.created_at, c.deleted_at, t.depth + 1
			FROM comments c
			JOIN tree t ON c.parent_id = t.id
			WHERE c.deleted_at IS NULL
		)
		SELECT id, post_id, author_id, parent_id, body, created_at, deleted_at, depth
		FROM tree ORDER BY created_at
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return buildCommentTree(rows)
}

// GetChildren возвращает дочерние комментарии для HTMX-подгрузки.
func (r *CommentRepo) GetChildren(parentID int64) ([]*model.CommentTreeNode, error) {
	rows, err := r.db.Query(`
		WITH RECURSIVE tree AS (
			SELECT id, post_id, author_id, parent_id, body, created_at, deleted_at, 0 AS depth
			FROM comments
			WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT c.id, c.post_id, c.author_id, c.parent_id, c.body, c.created_at, c.deleted_at, t.depth + 1
			FROM comments c
			JOIN tree t ON c.parent_id = t.id
			WHERE c.deleted_at IS NULL
		)
		SELECT id, post_id, author_id, parent_id, body, created_at, deleted_at, depth
		FROM tree WHERE depth > 0 ORDER BY created_at
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return buildCommentTree(rows)
}

func (r *CommentRepo) SoftDelete(id int64) error {
	_, err := r.db.Exec(`
		UPDATE comments SET deleted_at = datetime('now') WHERE id = ?
	`, id)
	return err
}

func buildCommentTree(rows *sql.Rows) ([]*model.CommentTreeNode, error) {
	// Карта id → узел
	nodes := make(map[int64]*model.CommentTreeNode)
	var roots []*model.CommentTreeNode

	for rows.Next() {
		node := &model.CommentTreeNode{}
		var parentID sql.NullInt64
		err := rows.Scan(&node.ID, &node.PostID, &node.AuthorID, &parentID,
			&node.Body, &node.CreatedAt, &node.DeletedAt, &node.Depth)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			node.ParentID = parentID
		}
		node.Children = []*model.CommentTreeNode{}
		nodes[node.ID] = node
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Строим дерево
	for _, node := range nodes {
		if node.ParentID.Valid {
			parent, ok := nodes[node.ParentID.Int64]
			if ok {
				parent.Children = append(parent.Children, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	return roots, nil
}

func scanComment(row *sql.Row) (*model.Comment, error) {
	c := &model.Comment{}
	var parentID sql.NullInt64
	err := row.Scan(&c.ID, &c.PostID, &c.AuthorID, &parentID, &c.Body, &c.CreatedAt, &c.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("comment: %w", model.ErrNotFound)
		}
		return nil, err
	}
	if parentID.Valid {
		c.ParentID = parentID
	}
	return c, nil
}
