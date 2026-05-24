package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/teblorum/teblorum/internal/auth"
	"github.com/teblorum/teblorum/internal/middleware"
	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/render"
)

// handleLoginForm показывает форму входа.
func handleLoginForm(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			render.PageData
			GoogleClientID string
		}{
			PageData:       render.PageData{Title: "Вход", Theme: "light", CurrentUser: middleware.UserFromContext(r.Context())},
			GoogleClientID: deps.GoogleClientID,
		}
		deps.Renderer.PageHTTP(w, "login", &data, render.DetectHTMX(r))
	}
}

// handleLogin обрабатывает POST /auth/login.
func handleLogin(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный запрос")
			return
		}

		email := strings.TrimSpace(r.PostForm.Get("email"))
		password := r.PostForm.Get("password")

		_, session, err := deps.Users.Login(email, password)
		if err != nil {
			if errors.Is(err, model.ErrUnauthorized) || errors.Is(err, model.ErrBanned) {
				deps.Renderer.WriteError(w, http.StatusUnauthorized, err.Error())
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка входа")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    session.ID,
			Path:     "/",
			MaxAge:   deps.SessionTTL * 86400,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleRegisterForm показывает форму регистрации.
func handleRegisterForm(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps.Renderer.PageHTTP(w, "register", &render.AuthPageData{
			PageData: render.PageData{Title: "Регистрация", Theme: "light", CurrentUser: middleware.UserFromContext(r.Context())},
		}, render.DetectHTMX(r))
	}
}

// handleRegister обрабатывает POST /auth/register.
func handleRegister(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный запрос")
			return
		}

		email := strings.TrimSpace(r.PostForm.Get("email"))
		username := strings.TrimSpace(r.PostForm.Get("username"))
		password := r.PostForm.Get("password")

		user, session, err := deps.Users.Register(email, username, password)
		if err != nil {
			if errors.Is(err, model.ErrDuplicate) || errors.Is(err, model.ErrValidation) {
				deps.Renderer.WriteError(w, http.StatusConflict, err.Error())
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка регистрации")
			return
		}
		_ = user

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    session.ID,
			Path:     "/",
			MaxAge:   deps.SessionTTL * 86400,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleGoogleLogin редиректит на Google OAuth.
func handleGoogleLogin(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.GoogleClientID == "" {
			deps.Renderer.WriteError(w, http.StatusNotImplemented, "Google OAuth не настроен")
			return
		}

		config := auth.NewGoogleOAuthConfig(
			deps.GoogleClientID,
			deps.GoogleClientSecret,
			deps.GoogleRedirectURL,
		)
		state := auth.GenerateSessionID() // используем как state
		url := auth.GetGoogleLoginURL(config, state)

		http.Redirect(w, r, url, http.StatusFound)
	}
}

// handleGoogleCallback обрабатывает колбэк Google OAuth.
func handleGoogleCallback(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.GoogleClientID == "" {
			deps.Renderer.WriteError(w, http.StatusNotImplemented, "Google OAuth не настроен")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Отсутствует код авторизации")
			return
		}

		config := auth.NewGoogleOAuthConfig(
			deps.GoogleClientID,
			deps.GoogleClientSecret,
			deps.GoogleRedirectURL,
		)

		token, err := auth.ExchangeCode(config, code)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка обмена кода")
			return
		}

		info, err := auth.GetGoogleUserInfo(token)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка получения данных пользователя")
			return
		}

		user, session, err := deps.Users.LoginOrRegisterGoogle(info.ID, info.Email, info.Name)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка аутентификации")
			return
		}
		_ = user

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    session.ID,
			Path:     "/",
			MaxAge:   deps.SessionTTL * 86400,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleLogout обрабатывает POST /auth/logout.
func handleLogout(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := middleware.SessionFromContext(r.Context())
		if session != nil {
			db := deps.DB
			_, _ = db.Exec("DELETE FROM sessions WHERE id = ?", session.ID)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleForgotForm показывает форму сброса пароля.
func handleForgotForm(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps.Renderer.PageHTTP(w, "forgot", render.AuthPageData{
			PageData: render.PageData{Title: "Сброс пароля", Theme: "light", CurrentUser: middleware.UserFromContext(r.Context())},
		}, render.DetectHTMX(r))
	}
}

// handleForgot обрабатывает POST /auth/forgot.
func handleForgot(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный запрос")
			return
		}

		email := strings.TrimSpace(r.PostForm.Get("email"))
		if email == "" {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Email обязателен")
			return
		}

		// Всегда возвращаем успех (не раскрываем, существует ли email)
		token, err := auth.GenerateResetToken(deps.DB, email)
		if err != nil {
			// Если пользователь не найден, всё равно говорим "проверьте email"
			// чтобы не раскрывать информацию
		}
		_ = token

		deps.Renderer.PageHTTP(w, "feed", &render.FeedPageData{
			PageData: render.PageData{Title: "Проверьте email", Theme: "light"},
		}, render.DetectHTMX(r))
	}
}

// handleResetForm показывает форму ввода нового пароля.
func handleResetForm(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Отсутствует токен")
			return
		}

		deps.Renderer.PageHTTP(w, "reset", &render.AuthPageData{
			PageData: render.PageData{Title: "Новый пароль", Theme: "light", CurrentUser: middleware.UserFromContext(r.Context())},
		}, render.DetectHTMX(r))
	}
}

// handleReset обрабатывает POST /auth/reset.
func handleReset(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный запрос")
			return
		}

		token := r.PostForm.Get("token")
		newPassword := r.PostForm.Get("password")

		if token == "" || newPassword == "" {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Все поля обязательны")
			return
		}

		if err := deps.Users.ResetPassword(token, newPassword); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/auth/login")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	}
}

// Ensure sql is used
var _ = (*sql.DB)(nil)
