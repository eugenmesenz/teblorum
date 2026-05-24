package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/teblorum/teblorum/internal/markdown"
	"github.com/teblorum/teblorum/internal/middleware"
	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/render"
	"github.com/teblorum/teblorum/internal/service"
)

// handleFeed обрабатывает GET / — лента публикаций.
func handleFeed(deps *Dependencies, w http.ResponseWriter, r *http.Request) {
	page := parsePage(r)

	posts, err := deps.Posts.GetFeed(page)
	if err != nil {
		deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка загрузки ленты")
		return
	}

	meta, err := deps.Posts.GetFeedMeta(page)
	if err != nil {
		deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка пагинации")
		return
	}

	feedPosts := make([]*render.FeedPost, 0, len(posts))
	for _, p := range posts {
		author, err := deps.Users.GetByID(p.AuthorID)
		username := ""
		if err == nil {
			username = author.Username
		}

		feedPosts = append(feedPosts, &render.FeedPost{
			ID:             p.ID,
			Type:           p.Type,
			Title:          p.Title,
			AuthorUsername: username,
			CreatedAt:      p.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	currentUser := middleware.UserFromContext(r.Context())
	data := render.FeedPageData{
		PageData: render.PageData{
			Title:       "Лента",
			Theme:       "light",
			CurrentUser: currentUser,
		},
		Posts: feedPosts,
		Pagination: render.PaginationData{
			Page:       page.Page,
			TotalPages: meta.TotalPages(),
			Total:      meta.Total,
		},
	}

	if render.DetectHTMX(r) {
		// HTMX-пагинация: рендерим только список
		deps.Renderer.Partial(w, "pagination.html", data.Pagination)
		return
	}

	deps.Renderer.PageHTTP(w, "feed", data, false)
}

// handleArticlesList обрабатывает GET /articles.
func handleArticlesList(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := parsePage(r)

		posts, err := deps.Posts.GetByType("article", page)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка загрузки статей")
			return
		}

		meta, err := deps.Posts.GetByTypeMeta("article", page)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка пагинации")
			return
		}

		feedPosts := buildFeedPosts(deps, posts)
		currentUser := middleware.UserFromContext(r.Context())

		data := render.FeedPageData{
			PageData: render.PageData{
				Title:       "Статьи",
				Theme:       "light",
				CurrentUser: currentUser,
			},
			Posts: feedPosts,
			Pagination: render.PaginationData{
				Page:       page.Page,
				TotalPages: meta.TotalPages(),
				Total:      meta.Total,
			},
		}

		deps.Renderer.PageHTTP(w, "feed", data, render.DetectHTMX(r))
	}
}

// handleThreadsList обрабатывает GET /threads.
func handleThreadsList(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := parsePage(r)

		posts, err := deps.Posts.GetByType("thread", page)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка загрузки тредов")
			return
		}

		meta, err := deps.Posts.GetByTypeMeta("thread", page)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка пагинации")
			return
		}

		feedPosts := buildFeedPosts(deps, posts)
		currentUser := middleware.UserFromContext(r.Context())

		data := render.FeedPageData{
			PageData: render.PageData{
				Title:       "Треды",
				Theme:       "light",
				CurrentUser: currentUser,
			},
			Posts: feedPosts,
			Pagination: render.PaginationData{
				Page:       page.Page,
				TotalPages: meta.TotalPages(),
				Total:      meta.Total,
			},
		}

		deps.Renderer.PageHTTP(w, "feed", data, render.DetectHTMX(r))
	}
}

// handleArticlePage обрабатывает GET /articles/{id}-{slug}.
func handleArticlePage(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		post, err := deps.Posts.GetByID(id)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		if post.Type != "article" {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		author, err := deps.Users.GetByID(post.AuthorID)
		authorUsername := ""
		if err == nil {
			authorUsername = author.Username
		}

		// Рендерим Markdown body
		bodyHTML, err := markdown.RenderToHTML(post.Body)
		if err != nil {
			bodyHTML = post.Body
		}

		// Рендерим дерево комментариев
		commentTree, _ := deps.Comments.GetTree(post.ID)
		commentHTML := renderCommentTree(deps, commentTree)

		currentUser := middleware.UserFromContext(r.Context())

		data := render.ArticlePageData{
			PageData: render.PageData{
				Title:       post.Title,
				Theme:       "light",
				CurrentUser: currentUser,
			},
			Title:           post.Title,
			AuthorUsername:  authorUsername,
			Body:            bodyHTML,
			CreatedAt:       post.CreatedAt.Format("2006-01-02 15:04"),
			CommentsEnabled: post.CommentsEnabled,
			CommentTree:     commentHTML,
		}

		deps.Renderer.PageHTTP(w, "article", data, render.DetectHTMX(r))
	}
}

// handleThreadPage обрабатывает GET /threads/{id}-{slug}.
func handleThreadPage(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		post, err := deps.Posts.GetByID(id)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		if post.Type != "thread" {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		author, err := deps.Users.GetByID(post.AuthorID)
		authorUsername := ""
		if err == nil {
			authorUsername = author.Username
		}

		bodyHTML, err := markdown.RenderToHTML(post.Body)
		if err != nil {
			bodyHTML = post.Body
		}

		commentTree, _ := deps.Comments.GetTree(post.ID)
		commentHTML := renderCommentTree(deps, commentTree)

		currentUser := middleware.UserFromContext(r.Context())

		data := render.ThreadPageData{
			PageData: render.PageData{
				Title:       post.Title,
				Theme:       "light",
				CurrentUser: currentUser,
			},
			Title:          post.Title,
			AuthorUsername: authorUsername,
			Body:           bodyHTML,
			CreatedAt:      post.CreatedAt.Format("2006-01-02 15:04"),
			CommentTree:    commentHTML,
		}

		deps.Renderer.PageHTTP(w, "thread", data, render.DetectHTMX(r))
	}
}

// handleNewArticleForm обрабатывает GET /articles/new.
func handleNewArticleForm(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps.Renderer.PageHTTP(w, "feed", render.FeedPageData{
			PageData: render.PageData{
				Title:       "Новая статья",
				Theme:       "light",
				CurrentUser: middleware.UserFromContext(r.Context()),
			},
		}, render.DetectHTMX(r))
	}
}

// handleCreateArticle обрабатывает POST /articles.
func handleCreateArticle(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный запрос")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		title := strings.TrimSpace(r.PostForm.Get("title"))
		body := r.PostForm.Get("body")
		commentsEnabled := r.PostForm.Get("comments_enabled") == "on"

		post, err := deps.Posts.CreateArticle(user.ID, title, body, commentsEnabled)
		if err != nil {
			if errors.Is(err, model.ErrValidation) {
				deps.Renderer.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка создания статьи")
			return
		}

		slug := service.GenerateSlug(title)
		redirectURL := "/articles/" + strconv.FormatInt(post.ID, 10) + "-" + slug

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", redirectURL)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

// handleNewThreadForm обрабатывает GET /threads/new.
func handleNewThreadForm(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps.Renderer.PageHTTP(w, "feed", render.FeedPageData{
			PageData: render.PageData{
				Title:       "Новый тред",
				Theme:       "light",
				CurrentUser: middleware.UserFromContext(r.Context()),
			},
		}, render.DetectHTMX(r))
	}
}

// handleCreateThread обрабатывает POST /threads.
func handleCreateThread(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный запрос")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		title := strings.TrimSpace(r.PostForm.Get("title"))
		body := r.PostForm.Get("body")

		post, err := deps.Posts.CreateThread(user.ID, title, body)
		if err != nil {
			if errors.Is(err, model.ErrValidation) {
				deps.Renderer.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка создания треда")
			return
		}

		slug := service.GenerateSlug(title)
		redirectURL := "/threads/" + strconv.FormatInt(post.ID, 10) + "-" + slug

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", redirectURL)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

// handleEditArticleForm обрабатывает GET /articles/{id}/edit.
func handleEditArticleForm(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		post, err := deps.Posts.GetByID(id)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil || post.AuthorID != user.ID {
			deps.Renderer.WriteError(w, http.StatusForbidden, "Доступ запрещён")
			return
		}

		deps.Renderer.PageHTTP(w, "feed", render.FeedPageData{
			PageData: render.PageData{
				Title:       "Редактирование",
				Theme:       "light",
				CurrentUser: user,
			},
		}, render.DetectHTMX(r))
	}
}

// handleUpdateArticle обрабатывает POST /articles/{id}.
func handleUpdateArticle(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		if err := r.ParseForm(); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный запрос")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		title := strings.TrimSpace(r.PostForm.Get("title"))
		body := r.PostForm.Get("body")
		commentsEnabled := r.PostForm.Get("comments_enabled") == "on"

		req := model.UpdatePostRequest{
			Title:           title,
			Body:            body,
			CommentsEnabled: &commentsEnabled,
		}

		if err := deps.Posts.Update(id, user.ID, req); err != nil {
			if errors.Is(err, model.ErrForbidden) {
				deps.Renderer.WriteError(w, http.StatusForbidden, "Доступ запрещён")
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка обновления")
			return
		}

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/articles/"+idStr)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/articles/"+idStr, http.StatusSeeOther)
	}
}

// Вспомогательные функции

func parsePage(r *http.Request) model.PaginationParams {
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	return model.PaginationParams{Page: page, Limit: 30}
}

func buildFeedPosts(deps *Dependencies, posts []*model.Post) []*render.FeedPost {
	feedPosts := make([]*render.FeedPost, 0, len(posts))
	for _, p := range posts {
		author, err := deps.Users.GetByID(p.AuthorID)
		username := ""
		if err == nil {
			username = author.Username
		}
		feedPosts = append(feedPosts, &render.FeedPost{
			ID:             p.ID,
			Type:           p.Type,
			Title:          p.Title,
			AuthorUsername: username,
			CreatedAt:      p.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	return feedPosts
}

func renderCommentTree(deps *Dependencies, tree []*model.CommentTreeNode) string {
	if len(tree) == 0 {
		return "<p>Нет комментариев.</p>"
	}
	return renderNodes(tree, 0)
}

func renderNodes(nodes []*model.CommentTreeNode, depth int) string {
	if len(nodes) == 0 {
		return ""
	}

	var result strings.Builder
	for _, node := range nodes {
		node.Depth = depth
		result.WriteString(renderSingleNode(node, depth))
	}
	return result.String()
}

func renderSingleNode(node *model.CommentTreeNode, depth int) string {
	bodyHTML, _ := markdown.RenderToHTMLSimple(node.Body)

	var result strings.Builder
	result.WriteString(`<div class="comment comment-depth-`)
	result.WriteString(strconv.Itoa(depth))
	result.WriteString(`" id="comment-`)
	result.WriteString(strconv.FormatInt(node.ID, 10))
	result.WriteString(`">`)
	result.WriteString(`<div class="comment-meta">`)
	result.WriteString(`<span class="comment-author">Пользователь #`)
	result.WriteString(strconv.FormatInt(node.AuthorID, 10))
	result.WriteString(`</span>`)
	result.WriteString(`<time datetime="`)
	result.WriteString(node.CreatedAt.Format("2006-01-02T15:04:05Z"))
	result.WriteString(`">`)
	result.WriteString(node.CreatedAt.Format("2006-01-02 15:04"))
	result.WriteString(`</time>`)
	result.WriteString(`</div>`)
	result.WriteString(`<div class="comment-body">`)
	result.WriteString(bodyHTML)
	result.WriteString(`</div>`)
	result.WriteString(`</div>`)

	if len(node.Children) > 0 {
		result.WriteString(`<div class="comment-children">`)
		result.WriteString(renderNodes(node.Children, depth+1))
		result.WriteString(`</div>`)
	}

	return result.String()
}
