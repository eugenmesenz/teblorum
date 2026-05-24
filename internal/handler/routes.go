package handler

import (
	"database/sql"
	"io/fs"
	"net/http"
	"time"

	"github.com/teblorum/teblorum/internal/middleware"
	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/render"
	"github.com/teblorum/teblorum/internal/service"
)

// Dependencies — собранные зависимости для handler-ов.
type Dependencies struct {
	DB       *sql.DB
	Renderer *render.TemplateRenderer
	Users    *service.UserService
	Posts    *service.PostService
	Comments *service.CommentService
	Messages *service.MessageService
	Mod      *service.ModerationService
	Root     *service.RootService

	// Конфигурация
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	SessionTTL         int
}

// SetupRoutes регистрирует все маршруты и возвращает готовый http.Handler
// с полной middleware-цепочкой.
func SetupRoutes(deps *Dependencies, staticFS fs.FS) http.Handler {
	mux := http.NewServeMux()
	rl := middleware.NewRateLimiter(60, time.Minute)

	// --- Статика ---
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// --- Публичные маршруты (без аутентификации) ---
	// Лента
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Страница не найдена")
			return
		}
		handleFeed(deps, w, r)
	})

	mux.HandleFunc("GET /articles", handleArticlesList(deps))
	mux.HandleFunc("GET /threads", handleThreadsList(deps))
	mux.HandleFunc("GET /articles/{id}-{slug}", handleArticlePage(deps))
	mux.HandleFunc("GET /threads/{id}-{slug}", handleThreadPage(deps))

	// Пользователи
	mux.HandleFunc("GET /users/{username}", handleUserPage(deps))

	// Аутентификация (публичные страницы)
	mux.HandleFunc("GET /auth/login", handleLoginForm(deps))
	mux.HandleFunc("POST /auth/login", handleLogin(deps))
	mux.HandleFunc("GET /auth/register", handleRegisterForm(deps))
	mux.HandleFunc("POST /auth/register", handleRegister(deps))
	mux.HandleFunc("GET /auth/google", handleGoogleLogin(deps))
	mux.HandleFunc("GET /auth/google/callback", handleGoogleCallback(deps))
	mux.HandleFunc("GET /auth/forgot", handleForgotForm(deps))
	mux.HandleFunc("POST /auth/forgot", handleForgot(deps))
	mux.HandleFunc("GET /auth/reset", handleResetForm(deps))
	mux.HandleFunc("POST /auth/reset", handleReset(deps))

	// --- Маршруты, требующие аутентификации (User+) ---
	authMux := http.NewServeMux()

	authMux.HandleFunc("POST /auth/logout", handleLogout(deps))
	authMux.HandleFunc("GET /articles/new", handleNewArticleForm(deps))
	authMux.HandleFunc("POST /articles", handleCreateArticle(deps))
	authMux.HandleFunc("GET /threads/new", handleNewThreadForm(deps))
	authMux.HandleFunc("POST /threads", handleCreateThread(deps))
	authMux.HandleFunc("GET /articles/{id}/edit", handleEditArticleForm(deps))
	authMux.HandleFunc("POST /articles/{id}", handleUpdateArticle(deps))

	// Комментарии
	authMux.HandleFunc("POST /posts/{id}/comments", handleCreateComment(deps))
	authMux.HandleFunc("GET /comments/reply-form", handleReplyForm(deps))
	authMux.HandleFunc("GET /comments/{id}/children", handleCommentChildren(deps))

	// Сообщения
	authMux.HandleFunc("GET /messages", handleMessagesList(deps))
	authMux.HandleFunc("GET /messages/{username}", handleConversation(deps))
	authMux.HandleFunc("POST /messages/{username}", handleSendMessage(deps))

	// Настройки
	authMux.HandleFunc("GET /settings", handleSettings(deps))
	authMux.HandleFunc("POST /settings", handleUpdateSettings(deps))

	mux.Handle("POST /auth/logout", middleware.RequireAuth(authMux))
	mux.Handle("GET /articles/new", middleware.RequireAuth(authMux))
	mux.Handle("POST /articles", middleware.RequireAuth(authMux))
	mux.Handle("GET /threads/new", middleware.RequireAuth(authMux))
	mux.Handle("POST /threads", middleware.RequireAuth(authMux))
	mux.Handle("GET /articles/{id}/edit", middleware.RequireAuth(authMux))
	mux.Handle("POST /articles/{id}", middleware.RequireAuth(authMux))
	mux.Handle("POST /posts/{id}/comments", middleware.RequireAuth(authMux))
	mux.Handle("GET /comments/reply-form", middleware.RequireAuth(authMux))
	mux.Handle("GET /comments/{id}/children", middleware.RequireAuth(authMux))
	mux.Handle("GET /messages", middleware.RequireAuth(authMux))
	mux.Handle("GET /messages/{username}", middleware.RequireAuth(authMux))
	mux.Handle("POST /messages/{username}", middleware.RequireAuth(authMux))
	mux.Handle("GET /settings", middleware.RequireAuth(authMux))
	mux.Handle("POST /settings", middleware.RequireAuth(authMux))

	// --- Маршруты модератора (Moderator+) ---
	modMux := http.NewServeMux()
	modMux.HandleFunc("POST /mod/posts/{id}/delete", handleModDeletePost(deps))
	modMux.HandleFunc("POST /mod/comments/{id}/delete", handleModDeleteComment(deps))
	modMux.HandleFunc("POST /mod/users/{id}/ban", handleModBanUser(deps))
	modMux.HandleFunc("POST /mod/users/{id}/unban", handleModUnbanUser(deps))

	mux.Handle("POST /mod/posts/{id}/delete", middleware.RequireRole(model.RoleModerator, modMux))
	mux.Handle("POST /mod/comments/{id}/delete", middleware.RequireRole(model.RoleModerator, modMux))
	mux.Handle("POST /mod/users/{id}/ban", middleware.RequireRole(model.RoleModerator, modMux))
	mux.Handle("POST /mod/users/{id}/unban", middleware.RequireRole(model.RoleModerator, modMux))

	// --- Маршруты root ---
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("GET /root", handleRootPanel(deps))
	rootMux.HandleFunc("POST /root/backup", handleRootBackup(deps))
	rootMux.HandleFunc("POST /root/restore", handleRootRestore(deps))
	rootMux.HandleFunc("POST /root/users/{id}/promote", handleRootPromote(deps))
	rootMux.HandleFunc("POST /root/users/{id}/demote", handleRootDemote(deps))

	mux.Handle("GET /root", middleware.RequireRole(model.RoleRoot, rootMux))
	mux.Handle("POST /root/backup", middleware.RequireRole(model.RoleRoot, rootMux))
	mux.Handle("POST /root/restore", middleware.RequireRole(model.RoleRoot, rootMux))
	mux.Handle("POST /root/users/{id}/promote", middleware.RequireRole(model.RoleRoot, rootMux))
	mux.Handle("POST /root/users/{id}/demote", middleware.RequireRole(model.RoleRoot, rootMux))

	// --- Middleware-цепочка ---
	var h http.Handler = mux
	h = middleware.SecurityHeaders(h)
	h = rl.RateLimit(h)
	h = middleware.SessionMiddleware(deps.DB)(h)
	// CSRF будет добавлен позже (когда handler-ы будут готовы)

	return h
}

// Ensure dependencies are used
var _ = (&Dependencies{}).GoogleClientID
var _ = (&Dependencies{}).GoogleClientSecret
var _ = (&Dependencies{}).GoogleRedirectURL
var _ = service.GenerateSlug
