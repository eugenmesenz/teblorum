package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/teblorum/teblorum/internal/middleware"
	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/render"
)

// handleMessagesList обрабатывает GET /messages.
func handleMessagesList(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		conversations, err := deps.Messages.GetConversations(user.ID)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка загрузки диалогов")
			return
		}
		_ = conversations

		deps.Renderer.PageHTTP(w, "feed", render.FeedPageData{
			PageData: render.PageData{
				Title:       "Сообщения",
				Theme:       "light",
				CurrentUser: user,
			},
		}, render.DetectHTMX(r))
	}
}

// handleConversation обрабатывает GET /messages/{username}.
func handleConversation(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		if username == "" {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		otherUser, err := deps.Users.GetByUsername(username)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}

		page := parsePage(r)
		messages, err := deps.Messages.GetConversation(user.ID, otherUser.ID, page)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка загрузки сообщений")
			return
		}
		_ = messages

		deps.Renderer.PageHTTP(w, "feed", render.FeedPageData{
			PageData: render.PageData{
				Title:       "Диалог с " + otherUser.Username,
				Theme:       "light",
				CurrentUser: user,
			},
		}, render.DetectHTMX(r))
	}
}

// handleSendMessage обрабатывает POST /messages/{username}.
func handleSendMessage(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		if username == "" {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
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

		otherUser, err := deps.Users.GetByUsername(username)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}

		body := strings.TrimSpace(r.PostForm.Get("body"))
		if body == "" {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Сообщение не может быть пустым")
			return
		}

		_, err = deps.Messages.Send(user.ID, otherUser.ID, body)
		if err != nil {
			if errors.Is(err, model.ErrSelfAction) {
				deps.Renderer.WriteError(w, http.StatusBadRequest, "Нельзя отправить сообщение самому себе")
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка отправки")
			return
		}

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/messages/"+username)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/messages/"+username, http.StatusSeeOther)
	}
}

// handleSettings обрабатывает GET /settings.
func handleSettings(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		deps.Renderer.PageHTTP(w, "feed", render.FeedPageData{
			PageData: render.PageData{
				Title:       "Настройки",
				Theme:       "light",
				CurrentUser: user,
			},
		}, render.DetectHTMX(r))
	}
}

// handleUpdateSettings обрабатывает POST /settings.
func handleUpdateSettings(deps *Dependencies) http.HandlerFunc {
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

		username := strings.TrimSpace(r.PostForm.Get("username"))
		bio := strings.TrimSpace(r.PostForm.Get("bio"))

		if err := deps.Users.UpdateProfile(user.ID, model.UpdateUserRequest{
			Username: username,
			Bio:      bio,
		}); err != nil {
			if errors.Is(err, model.ErrValidation) || errors.Is(err, model.ErrDuplicate) {
				deps.Renderer.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка обновления")
			return
		}

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/settings")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	}
}

// handleUserPage обрабатывает GET /users/{username}.
func handleUserPage(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		if username == "" {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}

		profileUser, err := deps.Users.GetByUsername(username)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}

		currentUser := middleware.UserFromContext(r.Context())

		data := render.UserPageData{
			PageData: render.PageData{
				Title:       profileUser.Username,
				Theme:       "light",
				CurrentUser: currentUser,
			},
			Profile: &render.ProfileData{
				Username:  profileUser.Username,
				Bio:       profileUser.Bio,
				Role:      string(profileUser.Role),
				CreatedAt: profileUser.CreatedAt.Format("2006-01-02"),
			},
		}

		deps.Renderer.PageHTTP(w, "user", data, render.DetectHTMX(r))
	}
}

// Ensure time is used
var _ = time.Now
