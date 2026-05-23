package render

import "github.com/teblorum/teblorum/internal/model"

// PageData — базовые данные для всех страниц.
type PageData struct {
	Title       string
	CurrentUser *model.User
	IsHTMX      bool
	Theme       string
	FontSize    string
	UnreadCount int
	CsrfToken   string
}

// FeedPageData — данные для ленты.
type FeedPageData struct {
	PageData
	Posts      []*FeedPost
	Pagination PaginationData
}

// FeedPost — пост в ленте.
type FeedPost struct {
	ID             int64
	Type           string
	Title          string
	AuthorUsername string
	CreatedAt      string
}

// ArticlePageData — данные для страницы статьи.
type ArticlePageData struct {
	PageData
	Title           string
	AuthorUsername  string
	Body            string
	CreatedAt       string
	CommentsEnabled bool
	CommentTree     string
}

// ThreadPageData — данные для страницы треда.
type ThreadPageData struct {
	PageData
	Title          string
	AuthorUsername string
	Body           string
	CreatedAt      string
	CommentTree    string
}

// UserPageData — данные для страницы автора.
type UserPageData struct {
	PageData
	Profile *ProfileData
}

// ProfileData — данные профиля.
type ProfileData struct {
	Username  string
	Bio       string
	Role      string
	CreatedAt string
}

// PaginationData — данные пагинации.
type PaginationData struct {
	Page       int
	TotalPages int
	Total      int
}
