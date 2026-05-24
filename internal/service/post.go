package service

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

// PostService — бизнес-логика работы с публикациями.
type PostService struct {
	posts   *repo.PostRepo
	authors *repo.UserRepo
}

func NewPostService(db *sql.DB) *PostService {
	return &PostService{
		posts:   repo.NewPostRepo(db),
		authors: repo.NewUserRepo(db),
	}
}

// CreateArticle создаёт новую статью.
func (s *PostService) CreateArticle(authorID int64, title, body string, commentsEnabled bool) (*model.Post, error) {
	if err := validatePost(title, body); err != nil {
		return nil, err
	}

	return s.posts.Create(model.CreatePostRequest{
		AuthorID:        authorID,
		Type:            "article",
		Title:           title,
		Body:            body,
		CommentsEnabled: commentsEnabled,
	})
}

// CreateThread создаёт новый тред.
func (s *PostService) CreateThread(authorID int64, title, body string) (*model.Post, error) {
	if err := validatePost(title, body); err != nil {
		return nil, err
	}

	return s.posts.Create(model.CreatePostRequest{
		AuthorID:        authorID,
		Type:            "thread",
		Title:           title,
		Body:            body,
		CommentsEnabled: true, // В тредах комментарии всегда включены
	})
}

// GetFeed возвращает ленту публикаций с пагинацией.
func (s *PostService) GetFeed(page model.PaginationParams) ([]*model.Post, error) {
	if page.Limit < 1 || page.Limit > 30 {
		page.Limit = 30
	}
	if page.Page < 1 {
		page.Page = 1
	}
	return s.posts.GetFeed(page)
}

// GetFeedMeta возвращает мета-информацию пагинации для ленты.
func (s *PostService) GetFeedMeta(page model.PaginationParams) (model.PaginationMeta, error) {
	total, err := s.posts.CountFeed()
	if err != nil {
		return model.PaginationMeta{}, err
	}
	return model.PaginationMeta{
		Total: total,
		Page:  page.Page,
		Limit: page.Limit,
	}, nil
}

// GetByID возвращает публикацию по ID.
func (s *PostService) GetByID(id int64) (*model.Post, error) {
	return s.posts.GetByID(id)
}

// GetByType возвращает публикации по типу с пагинацией.
func (s *PostService) GetByType(postType string, page model.PaginationParams) ([]*model.Post, error) {
	if postType != "article" && postType != "thread" {
		return nil, fmt.Errorf("type: %w: invalid post type", model.ErrValidation)
	}
	if page.Limit < 1 || page.Limit > 30 {
		page.Limit = 30
	}
	if page.Page < 1 {
		page.Page = 1
	}
	return s.posts.GetByType(postType, page)
}

// GetByTypeMeta возвращает мета-информацию пагинации для типа.
func (s *PostService) GetByTypeMeta(postType string, page model.PaginationParams) (model.PaginationMeta, error) {
	total, err := s.posts.CountByType(postType)
	if err != nil {
		return model.PaginationMeta{}, err
	}
	return model.PaginationMeta{
		Total: total,
		Page:  page.Page,
		Limit: page.Limit,
	}, nil
}

// GetByAuthorID возвращает публикации автора с пагинацией.
func (s *PostService) GetByAuthorID(authorID int64, page model.PaginationParams) ([]*model.Post, error) {
	if page.Limit < 1 || page.Limit > 30 {
		page.Limit = 30
	}
	if page.Page < 1 {
		page.Page = 1
	}
	return s.posts.GetByAuthorID(authorID, page)
}

// Update обновляет публикацию. actor — пользователь, выполняющий действие.
func (s *PostService) Update(postID int64, actorID int64, req model.UpdatePostRequest) error {
	post, err := s.posts.GetByID(postID)
	if err != nil {
		return err
	}

	// Проверка прав: автор или admin/root
	actor, err := s.authors.GetByID(actorID)
	if err != nil {
		return err
	}

	if post.AuthorID != actorID && !model.CanActOn(actor.Role, model.RoleUser) {
		return fmt.Errorf("update post: %w", model.ErrForbidden)
	}

	if req.Title != "" {
		if utf8.RuneCountInString(req.Title) < 3 {
			return fmt.Errorf("title: %w: too short", model.ErrValidation)
		}
		if utf8.RuneCountInString(req.Title) > 200 {
			return fmt.Errorf("title: %w: too long", model.ErrValidation)
		}
	}
	if req.Body != "" {
		if utf8.RuneCountInString(req.Body) < 10 {
			return fmt.Errorf("body: %w: too short", model.ErrValidation)
		}
		if utf8.RuneCountInString(req.Body) > 50000 {
			return fmt.Errorf("body: %w: too long", model.ErrValidation)
		}
	}

	return s.posts.Update(postID, req)
}

// SoftDelete мягко удаляет публикацию.
func (s *PostService) SoftDelete(postID int64, actorID int64) error {
	post, err := s.posts.GetByID(postID)
	if err != nil {
		return err
	}

	actor, err := s.authors.GetByID(actorID)
	if err != nil {
		return err
	}

	// Автор может удалить свой пост; модератор+ может удалять посты пользователей с меньшей ролью
	if post.AuthorID != actorID {
		author, err := s.authors.GetByID(post.AuthorID)
		if err != nil {
			return err
		}
		if !model.CanActOn(actor.Role, author.Role) {
			return fmt.Errorf("soft delete post: %w", model.ErrForbidden)
		}
	}

	return s.posts.SoftDelete(postID)
}

// GenerateSlug создаёт URL-совместимый slug из заголовка.
func GenerateSlug(title string) string {
	// Приводим к нижнему регистру
	title = strings.ToLower(title)

	// Транслитерация: заменяем кириллицу на латиницу
	translit := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
		'е': "e", 'ё': "yo", 'ж': "zh", 'з': "z", 'и': "i",
		'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
		'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
		'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch",
		'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "",
		'э': "e", 'ю': "yu", 'я': "ya",
	}

	var result strings.Builder
	for _, r := range title {
		if tr, ok := translit[r]; ok {
			result.WriteString(tr)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			result.WriteRune('-')
		}
	}

	slug := result.String()

	// Убираем множественные дефисы
	re := regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")

	// Убираем ведущие и замыкающие дефисы
	slug = strings.Trim(slug, "-")

	if slug == "" {
		return "post"
	}

	// Ограничиваем длину
	if len(slug) > 80 {
		slug = slug[:80]
	}
	slug = strings.TrimRight(slug, "-")

	return slug
}

func validatePost(title, body string) error {
	if utf8.RuneCountInString(title) < 3 {
		return fmt.Errorf("title: %w: too short (min 3 characters)", model.ErrValidation)
	}
	if utf8.RuneCountInString(title) > 200 {
		return fmt.Errorf("title: %w: too long (max 200 characters)", model.ErrValidation)
	}
	if utf8.RuneCountInString(body) < 10 {
		return fmt.Errorf("body: %w: too short (min 10 characters)", model.ErrValidation)
	}
	if utf8.RuneCountInString(body) > 50000 {
		return fmt.Errorf("body: %w: too long (max 50000 characters)", model.ErrValidation)
	}
	return nil
}