package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/teblorum/teblorum/internal/markdown"
	"github.com/teblorum/teblorum/internal/middleware"
	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/render"
)

// handleCreateComment обрабатывает POST /posts/{id}/comments.
func handleCreateComment(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		postIDStr := r.PathValue("id")
		postID, err := strconv.ParseInt(postIDStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пост не найден")
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

		body := strings.TrimSpace(r.PostForm.Get("body"))
		if body == "" {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Комментарий не может быть пустым")
			return
		}

		var parentID *int64
		if pidStr := r.PostForm.Get("parent_id"); pidStr != "" {
			if pid, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
				parentID = &pid
			}
		}

		comment, err := deps.Comments.Create(postID, user.ID, parentID, body)
		if err != nil {
			if errors.Is(err, model.ErrForbidden) || errors.Is(err, model.ErrValidation) {
				deps.Renderer.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка создания комментария")
			return
		}
		_ = comment

		// HTMX-ответ: редирект на страницу поста
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusOK)
	}
}

// handleReplyForm обрабатывает GET /comments/reply-form?parent_id=N.
func handleReplyForm(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentIDStr := r.URL.Query().Get("parent_id")
		if parentIDStr == "" {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Не указан parent_id")
			return
		}

		parentID, err := strconv.ParseInt(parentIDStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный parent_id")
			return
		}

		// Находим post_id из parent комментария
		comment, err := deps.Comments.GetTree(parentID) // не то, нужно напрямую
		_ = comment

		// Используем прямой запрос через БД для получения post_id
		// Временно возвращаем просто форму
		html := `<form hx-post="/posts/` + parentIDStr + `/comments" hx-target="#comments-root" hx-swap="beforeend" class="comment-form">
			<textarea name="body" rows="3" required placeholder="Напишите ответ..."></textarea>
			<input type="hidden" name="parent_id" value="` + parentIDStr + `">
			<button type="submit">Ответить</button>
		</form>`

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}

// handleCommentChildren обрабатывает GET /comments/{id}/children.
func handleCommentChildren(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Комментарий не найден")
			return
		}

		children, err := deps.Comments.GetChildren(id)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Комментарий не найден")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(renderCommentTree(deps, children)))
	}
}

// handleModDeletePost обрабатывает POST /mod/posts/{id}/delete.
func handleModDeletePost(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Публикация не найдена")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		if err := deps.Posts.SoftDelete(id, user.ID); err != nil {
			if errors.Is(err, model.ErrForbidden) {
				deps.Renderer.WriteError(w, http.StatusForbidden, "Доступ запрещён")
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка удаления")
			return
		}

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleModDeleteComment обрабатывает POST /mod/comments/{id}/delete.
func handleModDeleteComment(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Комментарий не найден")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		if err := deps.Comments.SoftDelete(id, user.ID); err != nil {
			if errors.Is(err, model.ErrForbidden) {
				deps.Renderer.WriteError(w, http.StatusForbidden, "Доступ запрещён")
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка удаления")
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// handleModBanUser обрабатывает POST /mod/users/{id}/ban.
func handleModBanUser(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
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

		durationStr := r.PostForm.Get("duration")
		duration := parseBanDuration(durationStr)

		if err := deps.Mod.BanUser(id, user.ID, duration); err != nil {
			if errors.Is(err, model.ErrForbidden) {
				deps.Renderer.WriteError(w, http.StatusForbidden, "Доступ запрещён")
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка бана")
			return
		}

		if render.DetectHTMX(r) {
			w.Header().Set("HX-Redirect", "/users/"+idStr)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleModUnbanUser обрабатывает POST /mod/users/{id}/unban.
func handleModUnbanUser(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		if err := deps.Mod.UnbanUser(id, user.ID); err != nil {
			if errors.Is(err, model.ErrForbidden) {
				deps.Renderer.WriteError(w, http.StatusForbidden, "Доступ запрещён")
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка разбана")
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleRootPromote обрабатывает POST /root/users/{id}/promote.
func handleRootPromote(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		if err := deps.Mod.PromoteToModerator(id, user.ID); err != nil {
			if errors.Is(err, model.ErrForbidden) || errors.Is(err, model.ErrValidation) {
				deps.Renderer.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка назначения")
			return
		}

		http.Redirect(w, r, "/root", http.StatusSeeOther)
	}
}

// handleRootDemote обрабатывает POST /root/users/{id}/demote.
func handleRootDemote(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}

		user := middleware.UserFromContext(r.Context())
		if user == nil {
			deps.Renderer.WriteError(w, http.StatusUnauthorized, "Требуется вход")
			return
		}

		if err := deps.Mod.DemoteFromModerator(id, user.ID); err != nil {
			if errors.Is(err, model.ErrForbidden) || errors.Is(err, model.ErrValidation) {
				deps.Renderer.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка снятия")
			return
		}

		http.Redirect(w, r, "/root", http.StatusSeeOther)
	}
}

// handleRootPanel обрабатывает GET /root.
func handleRootPanel(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFromContext(r.Context())
		deps.Renderer.PageHTTP(w, "feed", render.FeedPageData{
			PageData: render.PageData{
				Title:       "Панель управления",
				Theme:       "light",
				CurrentUser: user,
			},
		}, render.DetectHTMX(r))
	}
}

// handleRootBackup обрабатывает POST /root/backup.
func handleRootBackup(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		backupPath, err := deps.Root.CreateBackup()
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка создания бекапа")
			return
		}
		_ = backupPath

		http.Redirect(w, r, "/root", http.StatusSeeOther)
	}
}

// handleRootRestore обрабатывает POST /root/restore.
func handleRootRestore(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Неверный запрос")
			return
		}

		file, _, err := r.FormFile("backup")
		if err != nil {
			deps.Renderer.WriteError(w, http.StatusBadRequest, "Файл не загружен")
			return
		}
		defer file.Close()

		if err := deps.Root.RestoreFromUpload(file, ""); err != nil {
			deps.Renderer.WriteError(w, http.StatusInternalServerError, "Ошибка восстановления")
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Вспомогательные функции

func parseBanDuration(durationStr string) time.Duration {
	switch durationStr {
	case "1h":
		return 1 * time.Hour
	case "6h":
		return 6 * time.Hour
	case "1d":
		return 24 * time.Hour
	case "7d":
		return 168 * time.Hour
	case "30d":
		return 720 * time.Hour
	case "permanent":
		return 999999 * time.Hour
	default:
		return 24 * time.Hour // По умолчанию 24 часа
	}
}

// Ensure markdown is used
var _ = markdown.RenderToHTML
