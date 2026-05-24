package service

import (
	"database/sql"
	"fmt"
	"unicode/utf8"

	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

const maxCommentDepth = 6

// CommentService — бизнес-логика работы с комментариями.
type CommentService struct {
	comments *repo.CommentRepo
	posts    *repo.PostRepo
	authors  *repo.UserRepo
}

func NewCommentService(db *sql.DB) *CommentService {
	return &CommentService{
		comments: repo.NewCommentRepo(db),
		posts:    repo.NewPostRepo(db),
		authors:  repo.NewUserRepo(db),
	}
}

// Create создаёт новый комментарий.
func (s *CommentService) Create(postID, authorID int64, parentID *int64, body string) (*model.Comment, error) {
	if utf8.RuneCountInString(body) < 1 {
		return nil, fmt.Errorf("body: %w: cannot be empty", model.ErrValidation)
	}
	if utf8.RuneCountInString(body) > 5000 {
		return nil, fmt.Errorf("body: %w: too long (max 5000 characters)", model.ErrValidation)
	}

	// Проверяем существование поста
	post, err := s.posts.GetByID(postID)
	if err != nil {
		return nil, fmt.Errorf("post: %w", err)
	}

	// Для article проверяем, включены ли комментарии
	if post.Type == "article" && !post.CommentsEnabled {
		return nil, fmt.Errorf("comments disabled: %w", model.ErrForbidden)
	}

	// Проверяем parent_id и глубину
	if parentID != nil {
		parent, err := s.comments.GetByID(*parentID)
		if err != nil {
			return nil, fmt.Errorf("parent comment: %w", err)
		}

		// Проверяем, что parent принадлежит тому же посту
		if parent.PostID != postID {
			return nil, fmt.Errorf("parent comment belongs to different post: %w", model.ErrValidation)
		}

		// Вычисляем глубину parent'а
		depth, err := s.commentDepth(parent.ID)
		if err != nil {
			return nil, fmt.Errorf("compute depth: %w", err)
		}

		if depth >= maxCommentDepth {
			return nil, fmt.Errorf("max comment depth exceeded: %w", model.ErrValidation)
		}
	}

	return s.comments.Create(model.CreateCommentRequest{
		PostID:   postID,
		AuthorID: authorID,
		ParentID: parentID,
		Body:     body,
	})
}

// GetTree возвращает дерево комментариев для поста.
func (s *CommentService) GetTree(postID int64) ([]*model.CommentTreeNode, error) {
	return s.comments.GetByPostID(postID)
}

// GetChildren возвращает дочерние комментарии для HTMX-подгрузки.
func (s *CommentService) GetChildren(parentID int64) ([]*model.CommentTreeNode, error) {
	return s.comments.GetChildren(parentID)
}

// SoftDelete мягко удаляет комментарий.
func (s *CommentService) SoftDelete(commentID int64, actorID int64) error {
	comment, err := s.comments.GetByID(commentID)
	if err != nil {
		return err
	}

	actor, err := s.authors.GetByID(actorID)
	if err != nil {
		return err
	}

	// Автор может удалить свой комментарий
	if comment.AuthorID == actorID {
		return s.comments.SoftDelete(commentID)
	}

	// Модератор+ может удалять комментарии пользователей с меньшей ролью
	author, err := s.authors.GetByID(comment.AuthorID)
	if err != nil {
		return err
	}
	if !model.CanActOn(actor.Role, author.Role) {
		return fmt.Errorf("soft delete comment: %w", model.ErrForbidden)
	}

	return s.comments.SoftDelete(commentID)
}

// commentDepth вычисляет глубину комментария в дереве.
func (s *CommentService) commentDepth(id int64) (int, error) {
	depth := 0
	currentID := id

	for {
		comment, err := s.comments.GetByID(currentID)
		if err != nil {
			return 0, err
		}
		if !comment.ParentID.Valid {
			break
		}
		depth++
		currentID = comment.ParentID.Int64
	}

	return depth, nil
}