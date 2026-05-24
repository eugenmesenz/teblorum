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
	IsBanned  bool
}

// PaginationData — данные пагинации.
type PaginationData struct {
	Page       int
	TotalPages int
	Total      int
}

// AuthPageData — данные для страниц аутентификации.
type AuthPageData struct {
	PageData
	RedirectTo string
	Error      string
}

// MessagesPageData — данные для страницы сообщений.
type MessagesPageData struct {
	PageData
	Conversations []*ConversationItem
}

// ConversationItem — элемент списка диалогов.
type ConversationItem struct {
	WithUser      string
	WithUserID    int64
	LastMessage   string
	LastMessageAt string
	UnreadCount   int
}

// ConversationPageData — данные для страницы диалога.
type ConversationPageData struct {
	PageData
	WithUser   string
	Messages   []*MessageItem
	Pagination PaginationData
}

// MessageItem — сообщение в диалоге.
type MessageItem struct {
	ID        int64
	Body      string
	CreatedAt string
	IsMine    bool
}

// SettingsPageData — данные для страницы настроек.
type SettingsPageData struct {
	PageData
	Bio      string
	Username string
	Error    string
	Success  string
}

// RootPageData — данные для панели root.
type RootPageData struct {
	PageData
	Backups []string
}

// NewPostPageData — данные для формы создания публикации.
type NewPostPageData struct {
	PageData
	PostType string // "article" или "thread"
}

// EditPostPageData — данные для формы редактирования.
type EditPostPageData struct {
	PageData
	PostID          int64
	Title           string
	Body            string
	CommentsEnabled bool
}
