package handler

import (
	"database/sql"
	"io/fs"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/teblorum/teblorum/internal/render"
	"github.com/teblorum/teblorum/internal/repo"
	"github.com/teblorum/teblorum/internal/service"
)

// e2eSuite — полностью настроенный тестовый стенд для E2E-тестов.
type e2eSuite struct {
	db     *sql.DB
	server *httptest.Server
	client *http.Client
}

// newE2ESuite создаёт тестовый стенд:
//   - in-memory SQLite с миграциями
//   - все сервисы (User, Post, Comment, Message, Moderation, Root)
//   - TemplateRenderer с упрощёнными шаблонами
//   - полный HTTP-сервер со всеми middleware и маршрутами
//   - HTTP-клиент с cookie jar (без авто-следования редиректам)
func newE2ESuite(t *testing.T) *e2eSuite {
	t.Helper()

	db := repo.NewTestDB(t)

	// Сервисы
	userSvc := service.NewUserService(db)
	postSvc := service.NewPostService(db)
	commentSvc := service.NewCommentService(db)
	messageSvc := service.NewMessageService(db)
	modSvc := service.NewModerationService(db)
	rootSvc := service.NewRootService(db, t.TempDir())

	// Рендерер с тестовыми шаблонами
	renderer, err := render.NewTemplateRenderer(e2eTestFS(t))
	if err != nil {
		t.Fatalf("create template renderer: %v", err)
	}

	// Статика
	staticFS := e2eStaticFS(t)

	deps := &Dependencies{
		DB:       db,
		Renderer: renderer,
		Users:    userSvc,
		Posts:    postSvc,
		Comments: commentSvc,
		Messages: messageSvc,
		Mod:      modSvc,
		Root:     rootSvc,

		GoogleClientID:     "",
		GoogleClientSecret: "",
		GoogleRedirectURL:  "",
		SessionTTL:         30,
	}

	h := SetupRoutes(deps, staticFS)
	server := httptest.NewServer(h)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}

	// Не следуем редиректам автоматически — проверяем 302 вручную
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	t.Cleanup(server.Close)

	return &e2eSuite{
		db:     db,
		server: server,
		client: client,
	}
}

// get выполняет GET-запрос и возвращает ответ (редиректы не следуются).
func (s *e2eSuite) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := s.client.Get(s.server.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// getWithHX выполняет GET-запрос с HTMX-заголовком.
func (s *e2eSuite) getWithHX(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, s.server.URL+path, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("HX-Request", "true")
	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("GET (HTMX) %s: %v", path, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// e2eTestFS возвращает минимальную файловую систему шаблонов для тестов.
func e2eTestFS(t *testing.T) fstest.MapFS {
	t.Helper()

	// Достаточно, чтобы TemplateRenderer.parseAll() не упал
	// и рендеринг страниц возвращал валидный HTML.
	return fstest.MapFS{
		"web/templates/layout.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html lang="ru"><head><title>{{ .Title }} — teblorum</title></head><body><main>{{ block "content" . }}{{ end }}</main></body></html>`),
		},
		"web/templates/pages/feed.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Feed</h1>{{ range .Posts }}<article><h2><a href="/{{ if eq .Type "article" }}articles{{ else }}threads{{ end }}/{{ .ID }}">{{ .Title }}</a></h2><p>{{ .AuthorUsername }}</p></article>{{ else }}<p>No posts.</p>{{ end }}{{ end }}`),
		},
		"web/templates/pages/article.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<article><h1>{{ .Title }}</h1><div class="post-body">{{ .Body }}</div></article>{{ end }}`),
		},
		"web/templates/pages/thread.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<article><h1>{{ .Title }}</h1><div class="post-body">{{ .Body }}</div></article>{{ end }}`),
		},
		"web/templates/pages/user.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<div class="user-profile"><h1>{{ .Profile.Username }}</h1></div>{{ end }}`),
		},
		"web/templates/pages/login.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Login</h1>{{ end }}`),
		},
		"web/templates/pages/register.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Register</h1>{{ end }}`),
		},
		"web/templates/pages/forgot.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Forgot</h1>{{ end }}`),
		},
		"web/templates/pages/reset.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Reset</h1>{{ end }}`),
		},
		"web/templates/pages/messages.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Messages</h1>{{ end }}`),
		},
		"web/templates/pages/conversation.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Conversation</h1>{{ end }}`),
		},
		"web/templates/pages/settings.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Settings</h1>{{ end }}`),
		},
		"web/templates/pages/new_post.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>New Post</h1>{{ end }}`),
		},
		"web/templates/pages/edit_post.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Edit Post</h1>{{ end }}`),
		},
		"web/templates/pages/root.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Root Panel</h1>{{ end }}`),
		},
		"web/templates/partials/comment.html": &fstest.MapFile{
			Data: []byte(`<div class="comment">{{ .Body }}</div>`),
		},
		"web/templates/partials/comment_form.html": &fstest.MapFile{
			Data: []byte(`<form><textarea name="body"></textarea></form>`),
		},
		"web/templates/partials/pagination.html": &fstest.MapFile{
			Data: []byte(`{{ if gt .TotalPages 1 }}<p>Page {{ .Page }} of {{ .TotalPages }}</p>{{ end }}`),
		},
	}
}

// e2eStaticFS возвращает минимальную статику для тестов.
func e2eStaticFS(t *testing.T) fs.FS {
	t.Helper()
	return fstest.MapFS{
		"style.css":   &fstest.MapFile{Data: []byte("/* test */")},
		"htmx.min.js": &fstest.MapFile{Data: []byte("// test")},
	}
}