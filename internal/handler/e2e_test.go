package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/teblorum/teblorum/internal/repo"
)

// ---------------------------------------------------------------------------
// Группа A: Анонимный пользователь (публичный доступ)
// ---------------------------------------------------------------------------

func TestE2E_GroupA(t *testing.T) {
	suite := newE2ESuite(t)

	// A1: Просмотр ленты
	t.Run("A1_Feed", func(t *testing.T) {
		resp := suite.get(t, "/")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Feed") {
			t.Errorf("body should contain 'Feed', got: %s", truncate(body, 200))
		}
		if !strings.Contains(body, "teblorum") {
			t.Errorf("body should contain 'teblorum', got: %s", truncate(body, 200))
		}
	})

	// A2: Просмотр статей (пустая лента)
	t.Run("A2_ArticlesList", func(t *testing.T) {
		resp := suite.get(t, "/articles")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Feed") {
			t.Errorf("body should contain 'Feed', got: %s", truncate(body, 200))
		}
	})

	// A3: Просмотр тредов (пустая лента)
	t.Run("A3_ThreadsList", func(t *testing.T) {
		resp := suite.get(t, "/threads")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Feed") {
			t.Errorf("body should contain 'Feed', got: %s", truncate(body, 200))
		}
	})

	// A4: Просмотр статьи (нужно создать статью сначала)
	t.Run("A4_ArticlePage", func(t *testing.T) {
		user := repo.SeedUser(t, suite.db, map[string]interface{}{
			"username": "author1",
			"email":    "author1@example.com",
		})
		post := repo.SeedPost(t, suite.db, map[string]interface{}{
			"title":     "Test Article E2E",
			"body":      "This is the article body.",
			"type":      "article",
			"author_id": user.ID,
		})

		resp := suite.get(t, "/articles/"+fmt.Sprintf("%d", post.ID))
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Test Article E2E") {
			t.Errorf("body should contain article title, got: %s", truncate(body, 200))
		}
		if !strings.Contains(body, "article body") {
			t.Errorf("body should contain article body, got: %s", truncate(body, 200))
		}
	})

	// A5: Просмотр треда (создаём отдельно, т.к. после A4 post.ID может быть != 1)
	t.Run("A5_ThreadPage", func(t *testing.T) {
		user := repo.SeedUser(t, suite.db, map[string]interface{}{
			"username": "author2",
			"email":    "author2@example.com",
		})
		post := repo.SeedPost(t, suite.db, map[string]interface{}{
			"title":     "Test Thread E2E",
			"body":      "This is the thread body.",
			"type":      "thread",
			"author_id": user.ID,
		})

		resp := suite.get(t, "/threads/"+fmt.Sprintf("%d", post.ID))
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Test Thread E2E") {
			t.Errorf("body should contain thread title, got: %s", truncate(body, 200))
		}
		if !strings.Contains(body, "thread body") {
			t.Errorf("body should contain thread body, got: %s", truncate(body, 200))
		}
	})

	// A6: Просмотр профиля пользователя
	t.Run("A6_UserProfile", func(t *testing.T) {
		repo.SeedUser(t, suite.db, map[string]interface{}{
			"username": "profileuser",
			"email":    "profile@example.com",
			"bio":      "Hello, I am a test user.",
		})

		resp := suite.get(t, "/users/profileuser")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "profileuser") {
			t.Errorf("body should contain 'profileuser', got: %s", truncate(body, 200))
		}
	})

	// A7: Форма логина (handler рендерит шаблон "feed", но с заголовком "Вход")
	t.Run("A7_LoginForm", func(t *testing.T) {
		resp := suite.get(t, "/auth/login")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Вход") {
			t.Errorf("body should contain 'Вход', got: %s", truncate(body, 200))
		}
		// Хендлер рендерит страницу "feed" (базовый лендинг с заголовком Feed)
		if !strings.Contains(body, "Feed") {
			t.Errorf("body should contain 'Feed' (fallback template), got: %s", truncate(body, 200))
		}
	})

	// A8: Форма регистрации (handler рендерит шаблон "feed", заголовок "Регистрация")
	t.Run("A8_RegisterForm", func(t *testing.T) {
		resp := suite.get(t, "/auth/register")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Регистрация") {
			t.Errorf("body should contain 'Регистрация', got: %s", truncate(body, 200))
		}
	})

	// A9: Форма восстановления пароля (handler рендерит шаблон "feed", заголовок "Сброс пароля")
	t.Run("A9_ForgotForm", func(t *testing.T) {
		resp := suite.get(t, "/auth/forgot")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Сброс пароля") {
			t.Errorf("body should contain 'Сброс пароля', got: %s", truncate(body, 200))
		}
	})

	// A10: 404 — несуществующая страница
	t.Run("A10_NotFound", func(t *testing.T) {
		resp := suite.get(t, "/nonexistent")
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "404") && !strings.Contains(body, "не найдена") {
			t.Errorf("body should contain error message, got: %s", truncate(body, 200))
		}
	})

	// A11: HTMX-пагинация ленты
	t.Run("A11_HTMX_Pagination", func(t *testing.T) {
		// Создаём достаточно постов для пагинации (31+)
		user := repo.SeedUser(t, suite.db, map[string]interface{}{
			"username": "paginator",
			"email":    "paginator@example.com",
		})
		for i := 1; i <= 35; i++ {
			repo.SeedPost(t, suite.db, map[string]interface{}{
				"title":     fmt.Sprintf("Post %d", i),
				"body":      fmt.Sprintf("Body %d", i),
				"type":      "article",
				"author_id": user.ID,
			})
		}

		t.Run("page1", func(t *testing.T) {
			resp := suite.getWithHX(t, "/")
			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
			}
			body := readBody(t, resp)
			// HTMX-ответ: должен содержать пагинацию, но не полный layout
			if strings.Contains(body, "<html") {
				t.Errorf("HTMX response should not contain full layout, got html tag")
			}
		})

		t.Run("page2", func(t *testing.T) {
			resp := suite.getWithHX(t, "/?page=2")
			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
			}
		})
	})

	// A12: Google OAuth redirect (без настроек — должен быть 501)
	t.Run("A12_GoogleOAuth", func(t *testing.T) {
		resp := suite.get(t, "/auth/google")
		if resp.StatusCode != http.StatusNotImplemented {
			t.Errorf("status = %d, want %d (Google OAuth not configured)", resp.StatusCode, http.StatusNotImplemented)
		}
	})
}

// ---------------------------------------------------------------------------
// Вспомогательные функции
// ---------------------------------------------------------------------------

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}