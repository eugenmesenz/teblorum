package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/teblorum/teblorum/internal/auth"
	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

// sessionTTL — TTL сессии в днях (скользящее окно).
const sessionTTL = 30

// context keys
type ctxKey string

const (
	CtxUser    ctxKey = "user"
	CtxSession ctxKey = "session"
)

// contextUser достаёт пользователя из контекста.
func contextUser(ctx context.Context) (*model.User, bool) {
	u, ok := ctx.Value(CtxUser).(*model.User)
	return u, ok
}

// contextSession достаёт сессию из контекста.
func contextSession(ctx context.Context) (*model.Session, bool) {
	s, ok := ctx.Value(CtxSession).(*model.Session)
	return s, ok
}

// SessionMiddleware читает cookie session_id, находит сессию и пользователя,
// и кладёт их в контекст. Если сессия невалидна — удаляет cookie.
// Этот middleware не блокирует запрос, он только заполняет контекст.
func SessionMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			sessionRepo := repo.NewSessionRepo(db)
			session, err := sessionRepo.GetByID(cookie.Value)
			if err != nil {
				// Невалидная сессия — удаляем cookie
				http.SetCookie(w, &http.Cookie{
					Name:     "session_id",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
				next.ServeHTTP(w, r)
				return
			}

			if session.IsExpired() {
				sessionRepo.DeleteByID(session.ID)
				http.SetCookie(w, &http.Cookie{
					Name:     "session_id",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
				next.ServeHTTP(w, r)
				return
			}

			// Обновляем скользящее окно
			if err := sessionRepo.RefreshSession(session.ID, sessionTTL); err != nil {
				// Не фатально — продолжаем
			}

			// Получаем пользователя
			userRepo := repo.NewUserRepo(db)
			user, err := userRepo.GetByID(session.UserID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Кладём в контекст
			ctx := context.WithValue(r.Context(), CtxUser, user)
			ctx = context.WithValue(ctx, CtxSession, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth требует аутентификации.
// Если пользователь не аутентифицирован — 401 или редирект.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := contextUser(r.Context())
		if !ok || user == nil {
			// HTMX-запросы — 401, обычные — редирект
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/auth/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// OptionalAuth — псевдоним для SessionMiddleware.
// Он не блокирует запрос, а только заполняет контекст, если сессия есть.
type OptionalAuth = func(http.Handler) http.Handler

// NewOptionalAuth создаёт OptionalAuth middleware.
func NewOptionalAuth(db *sql.DB) OptionalAuth {
	return SessionMiddleware(db)
}

// RequireRole проверяет, что роль пользователя не ниже требуемой.
func RequireRole(required model.Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := contextUser(r.Context())
		if !ok || user == nil {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/auth/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		if !model.MinRole(user.Role, required) {
			if r.Header.Get("HX-Request") == "true" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders добавляет заголовки безопасности.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// RateLimiter — простой in-memory rate limiter.
type RateLimiter struct {
	requests map[string]int
	limit    int
	window   time.Duration
	cleanup  *time.Ticker
}

// NewRateLimiter создаёт rate limiter на limit запросов в window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]int),
		limit:    limit,
		window:   window,
		cleanup:  time.NewTicker(window),
	}
	go func() {
		for range rl.cleanup.C {
			rl.requests = make(map[string]int)
		}
	}()
	return rl
}

// Stop останавливает cleanup-горутину.
func (rl *RateLimiter) Stop() {
	rl.cleanup.Stop()
}

// RateLimit middleware ограничивает количество запросов с одного IP.
func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		rl.requests[ip]++
		if rl.requests[ip] > rl.limit {
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserFromContext извлекает пользователя из контекста (для handler-ов).
func UserFromContext(ctx context.Context) *model.User {
	user, _ := contextUser(ctx)
	return user
}

// SessionFromContext извлекает сессию из контекста.
func SessionFromContext(ctx context.Context) *model.Session {
	session, _ := contextSession(ctx)
	return session
}

// Compile-time checks
var (
	_ = RequireAuth
	_ = errors.New
	_ = auth.GenerateSessionID
)
