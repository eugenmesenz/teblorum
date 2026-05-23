package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return repo.NewTestDB(t)
}

func TestRequireAuthNoCookie(t *testing.T) {
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d (redirect to login)", w.Code, http.StatusSeeOther)
	}
}

func TestRequireAuthHTMX(t *testing.T) {
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if w.Header().Get("HX-Redirect") != "/auth/login" {
		t.Errorf("HX-Redirect = %q, want /auth/login", w.Header().Get("HX-Redirect"))
	}
}

func TestRequireAuthValidSession(t *testing.T) {
	db := newTestDB(t)
	user := repo.SeedUser(t, db, nil)
	session := repo.SeedSession(t, db, user.ID)

	middleware := SessionMiddleware(db)
	handler := middleware(RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil {
			t.Error("user should be in context")
			return
		}
		if u.ID != user.ID {
			t.Errorf("user.ID = %d, want %d", u.ID, user.ID)
		}
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: session.ID,
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOptionalAuthAnonymous(t *testing.T) {
	db := newTestDB(t)
	middleware := SessionMiddleware(db)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u != nil {
			t.Error("anonymous request should not have user in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireRole(t *testing.T) {
	db := newTestDB(t)

	t.Run("user cannot access moderator route", func(t *testing.T) {
		user := repo.SeedUser(t, db, nil)
		session := repo.SeedSession(t, db, user.ID)

		handler := SessionMiddleware(db)(RequireRole(model.RoleModerator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: session.ID})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
		}
	})

	t.Run("moderator can access moderator route", func(t *testing.T) {
		mod := repo.SeedUser(t, db, map[string]interface{}{
			"username": "moderator",
			"email":    "mod@example.com",
			"role":     model.RoleModerator,
		})
		session := repo.SeedSession(t, db, mod.ID)

		handler := SessionMiddleware(db)(RequireRole(model.RoleModerator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := UserFromContext(r.Context())
			if u == nil || u.Role != model.RoleModerator {
				t.Error("moderator should be in context")
			}
			w.WriteHeader(http.StatusOK)
		})))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: session.ID})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestUserFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxUser, &model.User{ID: 1, Username: "test"})
	u := UserFromContext(ctx)
	if u == nil || u.ID != 1 {
		t.Error("UserFromContext should return user from context")
	}

	emptyCtx := context.Background()
	if u := UserFromContext(emptyCtx); u != nil {
		t.Error("UserFromContext should return nil for empty context")
	}
}

func TestSessionFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxSession, &model.Session{ID: "test-session"})
	s := SessionFromContext(ctx)
	if s == nil || s.ID != "test-session" {
		t.Error("SessionFromContext should return session from context")
	}

	emptyCtx := context.Background()
	if s := SessionFromContext(emptyCtx); s != nil {
		t.Error("SessionFromContext should return nil for empty context")
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options header should be DENY")
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options header should be nosniff")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	defer rl.Stop()

	callCount := 0
	handler := rl.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))

	// Первые 2 запроса должны пройти
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}

	// Третий запрос — 429
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}
