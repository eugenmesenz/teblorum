package render

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/teblorum/teblorum/internal/model"
)

// testFS создаёт in-memory файловую систему для тестов рендеринга.
func testFS(t *testing.T) fstest.MapFS {
	t.Helper()

	return fstest.MapFS{
		"web/templates/layout.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html>
<html>
<head><title>{{ .Title }} — teblorum</title></head>
<body>
<nav><a href="/">teblorum</a></nav>
<main>{{ block "content" . }}{{ end }}</main>
</body>
</html>`),
		},
		"web/templates/pages/feed.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>Feed</h1>{{ end }}`),
		},
		"web/templates/pages/article.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<article>{{ .Title }}</article>{{ end }}`),
		},
		"web/templates/pages/thread.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>{{ .Title }}</h1>{{ end }}`),
		},
		"web/templates/pages/user.html": &fstest.MapFile{
			Data: []byte(`{{ define "content" }}<h1>{{ .Profile.Username }}</h1>{{ end }}`),
		},
		"web/templates/partials/comment.html": &fstest.MapFile{
			Data: []byte(`<div class="comment">{{ .Body }}</div>`),
		},
		"web/templates/partials/comment_form.html": &fstest.MapFile{
			Data: []byte(`<form hx-post="/comments">{{ if .ParentID }}<input type="hidden" name="parent_id" value="{{ .ParentID }}">{{ end }}<textarea name="body"></textarea><button>Send</button></form>`),
		},
		"web/templates/partials/pagination.html": &fstest.MapFile{
			Data: []byte(`<div class="pagination">{{ if gt .TotalPages 1 }}Page {{ .Page }} of {{ .TotalPages }}{{ end }}</div>`),
		},
	}
}

func TestNewTemplateRenderer(t *testing.T) {
	r, err := NewTemplateRenderer(testFS(t))
	if err != nil {
		t.Fatalf("NewTemplateRenderer() error = %v", err)
	}
	if r == nil {
		t.Fatal("renderer should not be nil")
	}
}

func TestRenderPage(t *testing.T) {
	r, _ := NewTemplateRenderer(testFS(t))

	t.Run("full page", func(t *testing.T) {
		out, err := r.PageToString("feed", PageData{Title: "Test"}, false)
		if err != nil {
			t.Fatalf("PageToString() error = %v", err)
		}
		if len(out) == 0 {
			t.Fatal("empty output")
		}
	})

	t.Run("htmx fragment", func(t *testing.T) {
		out, err := r.PageToString("feed", PageData{Title: "Test"}, true)
		if err != nil {
			t.Fatalf("PageToString() error = %v", err)
		}
		if len(out) == 0 {
			t.Fatal("empty output")
		}
	})

	t.Run("unknown page", func(t *testing.T) {
		_, err := r.PageToString("nonexistent", nil, false)
		if err == nil {
			t.Fatal("expected error for unknown page")
		}
	})
}

func TestRenderPartial(t *testing.T) {
	r, _ := NewTemplateRenderer(testFS(t))

	t.Run("comment partial", func(t *testing.T) {
		out, err := r.PartialString("comment.html", map[string]interface{}{
			"Body": "Hello!",
		})
		if err != nil {
			t.Fatalf("PartialString() error = %v", err)
		}
		if len(out) == 0 {
			t.Fatal("empty output")
		}
	})

	t.Run("unknown partial", func(t *testing.T) {
		_, err := r.PartialString("nonexistent.html", nil)
		if err == nil {
			t.Fatal("expected error for unknown partial")
		}
	})
}

func TestDetectHTMX(t *testing.T) {
	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")
		if !DetectHTMX(req) {
			t.Error("expected true for HTMX request")
		}
	})

	t.Run("regular request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if DetectHTMX(req) {
			t.Error("expected false for regular request")
		}
	})
}

func TestWriteError(t *testing.T) {
	r, _ := NewTemplateRenderer(testFS(t))

	t.Run("404 error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.WriteError(w, 404, "not found")
		if w.Code != 404 {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("403 error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.WriteError(w, 403, "forbidden")
		if w.Code != 403 {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})

	t.Run("500 error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.WriteError(w, 500, "internal error")
		if w.Code != 500 {
			t.Errorf("status = %d, want 500", w.Code)
		}
	})
}

func TestFeedPageData(t *testing.T) {
	data := FeedPageData{
		PageData: PageData{
			Title:       "Лента",
			CurrentUser: &model.User{ID: 1, Username: "alice"},
			IsHTMX:      false,
			Theme:       "light",
		},
		Posts: []*FeedPost{
			{ID: 1, Type: "article", Title: "Post 1", AuthorUsername: "alice", CreatedAt: "2025-01-01"},
			{ID: 2, Type: "thread", Title: "Thread 1", AuthorUsername: "bob", CreatedAt: "2025-01-02"},
		},
		Pagination: PaginationData{Page: 1, TotalPages: 1, Total: 2},
	}

	r, _ := NewTemplateRenderer(testFS(t))
	out, err := r.PageToString("feed", data, false)
	if err != nil {
		t.Fatalf("PageToString() error = %v", err)
	}
	if len(out) == 0 {
		t.Fatal("empty output")
	}
}
