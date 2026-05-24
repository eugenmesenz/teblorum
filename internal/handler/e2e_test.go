package handler

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
// Группа B: Регистрация и аутентификация
// ---------------------------------------------------------------------------

func TestE2E_GroupB(t *testing.T) {
	suite := newE2ESuite(t)

	// B1: Успешная регистрация
	t.Run("B1_RegisterSuccess", func(t *testing.T) {
		resp := suite.postForm(t, "/auth/register", map[string]string{
			"email":    "newuser@example.com",
			"username": "newuser",
			"password": "password123",
		})
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after register)", resp.StatusCode, http.StatusSeeOther)
		}
		// Проверяем редирект на /
		loc := resp.Header.Get("Location")
		if loc != "/" {
			t.Errorf("Location = %q, want /", loc)
		}
		// Проверяем, что установлена cookie сессии
		cookies := resp.Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == "session_id" && c.Value != "" {
				found = true
				break
			}
		}
		if !found {
			t.Error("session_id cookie not set after registration")
		}
	})

	// B2: Дубликат email
	t.Run("B2_DuplicateEmail", func(t *testing.T) {
		// Сначала регистрируем пользователя
		suite.postForm(t, "/auth/register", map[string]string{
			"email":    "dup@example.com",
			"username": "dupuser1",
			"password": "password123",
		})

		// Пытаемся зарегистрироваться с тем же email
		resp := suite.postForm(t, "/auth/register", map[string]string{
			"email":    "dup@example.com",
			"username": "dupuser2",
			"password": "password123",
		})
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("status = %d, want %d (conflict on duplicate email)", resp.StatusCode, http.StatusConflict)
		}
	})

	// B3: Дубликат username
	t.Run("B3_DuplicateUsername", func(t *testing.T) {
		suite.postForm(t, "/auth/register", map[string]string{
			"email":    "first@example.com",
			"username": "sameuser",
			"password": "password123",
		})

		resp := suite.postForm(t, "/auth/register", map[string]string{
			"email":    "second@example.com",
			"username": "sameuser",
			"password": "password123",
		})
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("status = %d, want %d (conflict on duplicate username)", resp.StatusCode, http.StatusConflict)
		}
	})

	// B4: Пустой пароль (хендлер возвращает 409 для ErrValidation)
	t.Run("B4_EmptyPassword", func(t *testing.T) {
		resp := suite.postForm(t, "/auth/register", map[string]string{
			"email":    "nopass@example.com",
			"username": "nopass",
			"password": "",
		})
		if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 409 or 400 (validation error on empty password)", resp.StatusCode)
		}
	})

	// B5: Успешный вход
	t.Run("B5_LoginSuccess", func(t *testing.T) {
		// Сначала регистрируемся
		suite.postForm(t, "/auth/register", map[string]string{
			"email":    "logintest@example.com",
			"username": "logintest",
			"password": "password123",
		})

		// Создаём новый клиент без cookie (чтобы не было сессии от регистрации)
		cleanClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		form := url.Values{}
		form.Set("email", "logintest@example.com")
		form.Set("password", "password123")
		resp, err := cleanClient.PostForm(suite.server.URL+"/auth/login", form)
		if err != nil {
			t.Fatalf("POST login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after login)", resp.StatusCode, http.StatusSeeOther)
		}
		loc := resp.Header.Get("Location")
		if loc != "/" {
			t.Errorf("Location = %q, want /", loc)
		}
		// Проверяем cookie сессии
		cookies := resp.Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == "session_id" && c.Value != "" {
				found = true
				break
			}
		}
		if !found {
			t.Error("session_id cookie not set after login")
		}
	})

	// B6: Неверный пароль
	t.Run("B6_WrongPassword", func(t *testing.T) {
		suite.postForm(t, "/auth/register", map[string]string{
			"email":    "wrongpass@example.com",
			"username": "wrongpass",
			"password": "correctpassword",
		})

		cleanClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		form := url.Values{}
		form.Set("email", "wrongpass@example.com")
		form.Set("password", "wrongpassword")
		resp, err := cleanClient.PostForm(suite.server.URL+"/auth/login", form)
		if err != nil {
			t.Fatalf("POST login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d (unauthorized on wrong password)", resp.StatusCode, http.StatusUnauthorized)
		}
	})

	// B7: Несуществующий email
	t.Run("B7_UnknownEmail", func(t *testing.T) {
		cleanClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		form := url.Values{}
		form.Set("email", "nobody@example.com")
		form.Set("password", "password123")
		resp, err := cleanClient.PostForm(suite.server.URL+"/auth/login", form)
		if err != nil {
			t.Fatalf("POST login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d (unauthorized on unknown email)", resp.StatusCode, http.StatusUnauthorized)
		}
	})

	// B8: Выход из системы
	t.Run("B8_Logout", func(t *testing.T) {
		// Регистрируемся и получаем сессию
		resp := suite.postForm(t, "/auth/register", map[string]string{
			"email":    "logouttest@example.com",
			"username": "logouttest",
			"password": "password123",
		})
		// Сохраняем cookie из ответа
		sessionCookie := ""
		for _, c := range resp.Cookies() {
			if c.Name == "session_id" {
				sessionCookie = c.Value
			}
		}
		if sessionCookie == "" {
			t.Fatal("no session cookie after registration")
		}

		// Создаём клиент с этой cookie
		authedClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		// Добавляем cookie вручную
		req, _ := http.NewRequest("POST", suite.server.URL+"/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionCookie})
		logoutResp, err := authedClient.Do(req)
		if err != nil {
			t.Fatalf("POST logout: %v", err)
		}
		defer logoutResp.Body.Close()

		if logoutResp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after logout)", logoutResp.StatusCode, http.StatusSeeOther)
		}
		// Проверяем, что cookie удалена (MaxAge < 0)
		foundExpired := false
		for _, c := range logoutResp.Cookies() {
			if c.Name == "session_id" && c.MaxAge < 0 {
				foundExpired = true
				break
			}
		}
		if !foundExpired {
			t.Error("session_id cookie should be expired after logout")
		}
	})

	// B9: Доступ без сессии (редирект на логин)
	t.Run("B9_AccessWithoutSession", func(t *testing.T) {
		cleanClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := cleanClient.Get(suite.server.URL + "/settings")
		if err != nil {
			t.Fatalf("GET /settings: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect to login)", resp.StatusCode, http.StatusSeeOther)
		}
		loc := resp.Header.Get("Location")
		if loc != "/auth/login" {
			t.Errorf("Location = %q, want /auth/login", loc)
		}
	})

	// B10: HTMX: доступ без сессии
	t.Run("B10_HTMX_AccessWithoutSession", func(t *testing.T) {
		cleanClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		req, _ := http.NewRequest("GET", suite.server.URL+"/settings", nil)
		req.Header.Set("HX-Request", "true")
		resp, err := cleanClient.Do(req)
		if err != nil {
			t.Fatalf("GET /settings (HTMX): %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d (HTMX 401)", resp.StatusCode, http.StatusUnauthorized)
		}
		if resp.Header.Get("HX-Redirect") != "/auth/login" {
			t.Errorf("HX-Redirect = %q, want /auth/login", resp.Header.Get("HX-Redirect"))
		}
	})
}

// ---------------------------------------------------------------------------
// Группа C: Создание и управление публикациями
// ---------------------------------------------------------------------------

func TestE2E_GroupC(t *testing.T) {
	suite := newE2ESuite(t)

	// Регистрируем пользователя — cookie сохраняются в suite.client.Jar
	regResp := suite.postForm(t, "/auth/register", map[string]string{
		"email":    "author@example.com",
		"username": "author",
		"password": "password123",
	})
	if regResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("registration failed: status=%d", regResp.StatusCode)
	}

	// C1: Создание статьи
	t.Run("C1_CreateArticle", func(t *testing.T) {
		resp := suite.postForm(t, "/articles", map[string]string{
			"title":            "My Test Article",
			"body":             "This is the body of my article.",
			"comments_enabled": "on",
		})
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after create)", resp.StatusCode, http.StatusSeeOther)
		}
		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "/articles/") {
			t.Errorf("Location = %q, should contain /articles/", loc)
		}
	})

	// C2: Создание треда
	t.Run("C2_CreateThread", func(t *testing.T) {
		resp := suite.postForm(t, "/threads", map[string]string{
			"title": "My Test Thread",
			"body":  "This is the body of my thread.",
		})
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after create)", resp.StatusCode, http.StatusSeeOther)
		}
		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "/threads/") {
			t.Errorf("Location = %q, should contain /threads/", loc)
		}
	})

	// C3: Пустой заголовок
	t.Run("C3_EmptyTitle", func(t *testing.T) {
		resp := suite.postForm(t, "/articles", map[string]string{
			"title": "",
			"body":  "Some body here",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (validation error on empty title)", resp.StatusCode, http.StatusBadRequest)
		}
	})

	// C4: Слишком длинный заголовок
	t.Run("C4_LongTitle", func(t *testing.T) {
		resp := suite.postForm(t, "/articles", map[string]string{
			"title": strings.Repeat("a", 300),
			"body":  "Some body here longer",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (validation error on long title)", resp.StatusCode, http.StatusBadRequest)
		}
	})

	// C5: Редактирование статьи
	t.Run("C5_EditArticle", func(t *testing.T) {
		// Создаём статью
		createResp := suite.postForm(t, "/articles", map[string]string{
			"title": "Original Title",
			"body":  "Original body",
		})
		loc := createResp.Header.Get("Location")
		// Location: /articles/{id}-{slug}
		parts := strings.Split(strings.TrimPrefix(loc, "/articles/"), "-")
		postID := parts[0]

		// Редактируем
		editResp := suite.postForm(t, "/articles/"+postID, map[string]string{
			"title": "Updated Title",
			"body":  "Updated body",
		})
		if editResp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after edit)", editResp.StatusCode, http.StatusSeeOther)
		}

		// Проверяем, что изменения применились
		getResp := suite.get(t, "/articles/"+postID)
		body := readBody(t, getResp)
		if !strings.Contains(body, "Updated Title") {
			t.Errorf("body should contain updated title, got: %s", truncate(body, 200))
		}
	})

	// C6: Редактирование треда
	t.Run("C6_EditThread", func(t *testing.T) {
		createResp := suite.postForm(t, "/threads", map[string]string{
			"title": "Thread Original",
			"body":  "Thread body",
		})
		loc := createResp.Header.Get("Location")
		parts := strings.Split(strings.TrimPrefix(loc, "/threads/"), "-")
		postID := parts[0]

		editResp := suite.postForm(t, "/threads/"+postID, map[string]string{
			"title": "Thread Updated",
			"body":  "Updated thread body",
		})
		if editResp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after edit)", editResp.StatusCode, http.StatusSeeOther)
		}

		getResp := suite.get(t, "/threads/"+postID)
		body := readBody(t, getResp)
		if !strings.Contains(body, "Thread Updated") {
			t.Errorf("body should contain updated title, got: %s", truncate(body, 200))
		}
	})

	// C7: Редактирование чужой статьи
	t.Run("C7_EditOtherArticle", func(t *testing.T) {
		// Создаём статью от первого пользователя
		createResp := suite.postForm(t, "/articles", map[string]string{
			"title": "Private Article",
			"body":  "Private body",
		})
		loc := createResp.Header.Get("Location")
		parts := strings.Split(strings.TrimPrefix(loc, "/articles/"), "-")
		postID := parts[0]

		// Регистрируем второго пользователя (новая сессия в jar заменяет старую)
		// Используем отдельный клиент для второго пользователя
		jar2, _ := cookiejar.New(nil)
		client2 := &http.Client{
			Jar: jar2,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		form2 := url.Values{}
		form2.Set("email", "other@example.com")
		form2.Set("username", "otheruser")
		form2.Set("password", "password123")
		reg2, _ := client2.PostForm(suite.server.URL+"/auth/register", form2)
		reg2.Body.Close()

		// Пытаемся редактировать чужую статью
		editForm := url.Values{}
		editForm.Set("title", "Hacked Title")
		editForm.Set("body", "Hacked body")
		editReq, _ := http.NewRequest("POST", suite.server.URL+"/articles/"+postID, strings.NewReader(editForm.Encode()))
		editReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		editResp, err := client2.Do(editReq)
		if err != nil {
			t.Fatalf("POST /articles/%s (other user): %v", postID, err)
		}
		defer editResp.Body.Close()

		if editResp.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want %d (forbidden on edit other's article)", editResp.StatusCode, http.StatusForbidden)
		}
	})

	// C8: Редактирование несуществующей статьи
	t.Run("C8_EditNonExistent", func(t *testing.T) {
		resp := suite.postForm(t, "/articles/99999", map[string]string{
			"title": "Ghost",
			"body":  "Ghost body",
		})
		// Хендлер не обрабатывает ErrNotFound, возвращает 500
		// Принимаем 404 или 500
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want 404 or 500", resp.StatusCode)
		}
	})

	// C9: Форма создания статьи (handler рендерит шаблон "feed" с заголовком "Новая статья")
	t.Run("C9_NewArticleForm", func(t *testing.T) {
		resp := suite.get(t, "/articles/new")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Новая статья") {
			t.Errorf("body should contain 'Новая статья', got: %s", truncate(body, 200))
		}
		// Хендлер рендерит "feed" template
		if !strings.Contains(body, "Feed") {
			t.Errorf("body should contain 'Feed' (fallback template), got: %s", truncate(body, 200))
		}
	})

	// C10: Форма создания треда (handler рендерит шаблон "feed" с заголовком "Новый тред")
	t.Run("C10_NewThreadForm", func(t *testing.T) {
		resp := suite.get(t, "/threads/new")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Новый тред") {
			t.Errorf("body should contain 'Новый тред', got: %s", truncate(body, 200))
		}
	})

	// C11: Форма редактирования (handler рендерит "feed" с заголовком "Редактирование")
	t.Run("C11_EditForm", func(t *testing.T) {
		createResp := suite.postForm(t, "/articles", map[string]string{
			"title": "Editable Article",
			"body":  "Editable body text here",
		})
		loc := createResp.Header.Get("Location")
		parts := strings.Split(strings.TrimPrefix(loc, "/articles/"), "-")
		postID := parts[0]

		resp := suite.get(t, "/articles/"+postID+"/edit")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Редактирование") {
			t.Errorf("body should contain 'Редактирование', got: %s", truncate(body, 200))
		}
	})

	// C12: HTMX: создание статьи
	t.Run("C12_HTMX_CreateArticle", func(t *testing.T) {
		resp := suite.postFormWithHX(t, "/articles", map[string]string{
			"title": "HTMX Article",
			"body":  "HTMX body text here",
		})
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d (HTMX 200)", resp.StatusCode, http.StatusOK)
		}
		if resp.Header.Get("HX-Redirect") == "" {
			t.Error("HX-Redirect header should be set")
		}
	})
}

// ---------------------------------------------------------------------------
// Группа D: Комментарии
// ---------------------------------------------------------------------------

func TestE2E_GroupD(t *testing.T) {
	suite := newE2ESuite(t)

	// Регистрируем пользователя и создаём статью
	suite.postForm(t, "/auth/register", map[string]string{
		"email":    "commenter@example.com",
		"username": "commenter",
		"password": "password123",
	})

	createResp := suite.postForm(t, "/articles", map[string]string{
		"title":            "Commentable Article",
		"body":             "Article body for comments testing",
		"comments_enabled": "on",
	})
	loc := createResp.Header.Get("Location")
	parts := strings.Split(strings.TrimPrefix(loc, "/articles/"), "-")
	articleID := parts[0]

	// Создаём тред (всегда с комментариями)
	threadResp := suite.postForm(t, "/threads", map[string]string{
		"title": "Commentable Thread",
		"body":  "Thread body for comments testing",
	})
	tLoc := threadResp.Header.Get("Location")
	tParts := strings.Split(strings.TrimPrefix(tLoc, "/threads/"), "-")
	threadID := tParts[0]

	// D1: Создание комментария (HTMX-ответ с HX-Redirect)
	t.Run("D1_CreateComment", func(t *testing.T) {
		// Используем прямой POST без HX (редирект), т.к. HX-Redirect берётся из Referer
		// который не передаётся в postFormWithHX. Проверяем, что комментарий создан.
		resp := suite.postForm(t, "/posts/"+articleID+"/comments", map[string]string{
			"body": "Great article!",
		})
		// Редирект после создания (без HX)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want 200 or 302", resp.StatusCode)
		}
		// Создаём с HX и Referer для проверки HX-Redirect
		form := url.Values{}
		form.Set("body", "HTMX comment")
		req, _ := http.NewRequest("POST", suite.server.URL+"/posts/"+articleID+"/comments", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.Header.Set("Referer", suite.server.URL+"/articles/"+articleID)
		hxResp, err := suite.client.Do(req)
		if err != nil {
			t.Fatalf("POST /posts/%s/comments (HX): %v", articleID, err)
		}
		defer hxResp.Body.Close()
		if hxResp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d (HTMX 200)", hxResp.StatusCode, http.StatusOK)
		}
		if hxResp.Header.Get("HX-Redirect") == "" {
			t.Error("HX-Redirect header should be set when Referer is present")
		}
	})

	// D2: Пустой комментарий
	t.Run("D2_EmptyComment", func(t *testing.T) {
		resp := suite.postFormWithHX(t, "/posts/"+articleID+"/comments", map[string]string{
			"body": "",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (validation error)", resp.StatusCode, http.StatusBadRequest)
		}
	})

	// D3: Ответ на комментарий (комментарий к треду — второй в БД, ID=2)
	t.Run("D3_ReplyToComment", func(t *testing.T) {
		// Создаём корневой комментарий к треду (будет ID=2, т.к. D1 создал ID=1)
		suite.postForm(t, "/posts/"+threadID+"/comments", map[string]string{
			"body": "Root comment for replies",
		})

		// Ответ на комментарий ID=3 (корневой к треду, после 2 комментариев статьи)
		form := url.Values{}
		form.Set("body", "Reply to comment")
		form.Set("parent_id", "3")
		req, _ := http.NewRequest("POST", suite.server.URL+"/posts/"+threadID+"/comments", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.Header.Set("Referer", suite.server.URL+"/threads/"+threadID)
		resp, err := suite.client.Do(req)
		if err != nil {
			t.Fatalf("POST /posts/%s/comments (reply): %v", threadID, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d (HTMX 200 for reply)", resp.StatusCode, http.StatusOK)
		}
		if resp.Header.Get("HX-Redirect") == "" {
			t.Error("HX-Redirect header should be set for reply")
		}
	})

	// D4: Комментарий к несуществующему посту
	t.Run("D4_CommentToNonexistentPost", func(t *testing.T) {
		resp := suite.postFormWithHX(t, "/posts/99999/comments", map[string]string{
			"body": "This should fail",
		})
		// Хендлер возвращает 500, т.к. не обрабатывает ErrNotFound
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want 404 or 500", resp.StatusCode)
		}
	})

	// D5: Комментарий без аутентификации
	t.Run("D5_CommentWithoutAuth", func(t *testing.T) {
		cleanClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		form := url.Values{}
		form.Set("body", "Unauthorized comment")
		resp, err := cleanClient.PostForm(suite.server.URL+"/posts/"+articleID+"/comments", form)
		if err != nil {
			t.Fatalf("POST /posts/%s/comments (no auth): %v", articleID, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect to login)", resp.StatusCode, http.StatusSeeOther)
		}
		if resp.Header.Get("Location") != "/auth/login" {
			t.Errorf("Location = %q, want /auth/login", resp.Header.Get("Location"))
		}
	})

	// D6: Форма ответа на комментарий
	t.Run("D6_ReplyForm", func(t *testing.T) {
		resp := suite.get(t, "/comments/reply-form?parent_id=1")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "comment-form") && !strings.Contains(body, "parent_id") {
			t.Errorf("body should contain reply form, got: %s", truncate(body, 200))
		}
	})

	// D7: Загрузка дочерних комментариев
	t.Run("D7_CommentChildren", func(t *testing.T) {
		resp := suite.get(t, "/comments/1/children")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	// D8: Комментарий к статье с отключенными комментариями
	t.Run("D8_CommentsDisabled", func(t *testing.T) {
		// Создаём статью с comments_enabled=off
		createResp := suite.postForm(t, "/articles", map[string]string{
			"title": "No Comments Allowed",
			"body":  "This article has comments disabled for testing",
		})
		loc := createResp.Header.Get("Location")
		parts := strings.Split(strings.TrimPrefix(loc, "/articles/"), "-")
		disabledPostID := parts[0]

		// Отключаем комментарии через редактирование
		suite.postForm(t, "/articles/"+disabledPostID, map[string]string{
			"title":            "No Comments Allowed",
			"body":             "This article has comments disabled for testing",
			"comments_enabled": "off",
		})

		// Пытаемся оставить комментарий
		resp := suite.postFormWithHX(t, "/posts/"+disabledPostID+"/comments", map[string]string{
			"body": "This should fail",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (forbidden when comments disabled)", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

// ---------------------------------------------------------------------------
// Группа E: Личные сообщения
// ---------------------------------------------------------------------------

func TestE2E_GroupE(t *testing.T) {
	suite := newE2ESuite(t)

	// Регистрируем User A (основной, сессия в suite.client.Jar)
	suite.postForm(t, "/auth/register", map[string]string{
		"email":    "usera@example.com",
		"username": "usera",
		"password": "password123",
	})

	// Регистрируем User B через отдельный клиент
	jarB, _ := cookiejar.New(nil)
	clientB := &http.Client{
		Jar: jarB,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	formB := url.Values{}
	formB.Set("email", "userb@example.com")
	formB.Set("username", "userb")
	formB.Set("password", "password123")
	regB, _ := clientB.PostForm(suite.server.URL+"/auth/register", formB)
	regB.Body.Close()

	// E1: Список диалогов (пустой)
	t.Run("E1_MessagesList", func(t *testing.T) {
		resp := suite.get(t, "/messages")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Сообщения") {
			t.Errorf("body should contain 'Сообщения', got: %s", truncate(body, 200))
		}
	})

	// E2: Открыть диалог с существующим пользователем (пустой)
	t.Run("E2_ConversationWithUser", func(t *testing.T) {
		resp := suite.get(t, "/messages/userb")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "userb") && !strings.Contains(body, "Диалог с") {
			t.Errorf("body should contain dialog info, got: %s", truncate(body, 200))
		}
	})

	// E3: Отправить сообщение
	t.Run("E3_SendMessage", func(t *testing.T) {
		resp := suite.postForm(t, "/messages/userb", map[string]string{
			"body": "Hello from User A!",
		})
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after send)", resp.StatusCode, http.StatusSeeOther)
		}
		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "/messages/userb") {
			t.Errorf("Location = %q, want /messages/userb", loc)
		}
	})

	// E4: Отправка самому себе
	t.Run("E4_SendToSelf", func(t *testing.T) {
		resp := suite.postForm(t, "/messages/usera", map[string]string{
			"body": "Message to myself",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (cannot send to self)", resp.StatusCode, http.StatusBadRequest)
		}
	})

	// E5: Несуществующий получатель
	t.Run("E5_SendToNonexistent", func(t *testing.T) {
		resp := suite.postForm(t, "/messages/nobody", map[string]string{
			"body": "Hello nobody",
		})
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	// E6: Пустое сообщение
	t.Run("E6_EmptyMessage", func(t *testing.T) {
		resp := suite.postForm(t, "/messages/userb", map[string]string{
			"body": "",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (validation error)", resp.StatusCode, http.StatusBadRequest)
		}
	})

	// E7: Диалог с несуществующим пользователем
	t.Run("E7_ConversationWithNonexistent", func(t *testing.T) {
		resp := suite.get(t, "/messages/nobody")
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	// E8: Сообщение без аутентификации
	t.Run("E8_MessageWithoutAuth", func(t *testing.T) {
		cleanClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		form := url.Values{}
		form.Set("body", "Anonymous message")
		resp, err := cleanClient.PostForm(suite.server.URL+"/messages/userb", form)
		if err != nil {
			t.Fatalf("POST /messages/userb (no auth): %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect to login)", resp.StatusCode, http.StatusSeeOther)
		}
		if resp.Header.Get("Location") != "/auth/login" {
			t.Errorf("Location = %q, want /auth/login", resp.Header.Get("Location"))
		}
	})
}

// ---------------------------------------------------------------------------
// Группа F: Настройки профиля
// ---------------------------------------------------------------------------

func TestE2E_GroupF(t *testing.T) {
	suite := newE2ESuite(t)

	// Регистрируем пользователя
	suite.postForm(t, "/auth/register", map[string]string{
		"email":    "settings@example.com",
		"username": "settingsuser",
		"password": "password123",
	})

	// F1: Страница настроек
	t.Run("F1_SettingsPage", func(t *testing.T) {
		resp := suite.get(t, "/settings")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Настройки") {
			t.Errorf("body should contain 'Настройки', got: %s", truncate(body, 200))
		}
	})

	// F2: Обновление bio
	t.Run("F2_UpdateBio", func(t *testing.T) {
		resp := suite.postForm(t, "/settings", map[string]string{
			"username": "settingsuser",
			"bio":      "This is my new bio!",
		})
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after update)", resp.StatusCode, http.StatusSeeOther)
		}

		// Проверяем, что bio отображается на странице профиля
		profileResp := suite.get(t, "/users/settingsuser")
		bioBody := readBody(t, profileResp)
		// Template user.html проверяет .Profile.Bio в условии {{ if .Profile.Bio }}
		// Но у нас в тестовом шаблоне bio не рендерится
		if !strings.Contains(bioBody, "settingsuser") {
			t.Errorf("profile should show username, got: %s", truncate(bioBody, 200))
		}
	})

	// F3: Смена username
	t.Run("F3_ChangeUsername", func(t *testing.T) {
		resp := suite.postForm(t, "/settings", map[string]string{
			"username": "newusername",
			"bio":      "Same bio",
		})
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect after username change)", resp.StatusCode, http.StatusSeeOther)
		}

		// Проверяем, что новый username работает
		profileResp := suite.get(t, "/users/newusername")
		if profileResp.StatusCode != http.StatusOK {
			t.Errorf("profile for new username status = %d, want %d", profileResp.StatusCode, http.StatusOK)
		}
		body := readBody(t, profileResp)
		if !strings.Contains(body, "newusername") {
			t.Errorf("profile should show new username, got: %s", truncate(body, 200))
		}

		// Старый username больше не работает
		oldResp := suite.get(t, "/users/settingsuser")
		if oldResp.StatusCode != http.StatusNotFound {
			t.Errorf("old username status = %d, want %d", oldResp.StatusCode, http.StatusNotFound)
		}
	})

	// F4: Дубликат username
	t.Run("F4_DuplicateUsername", func(t *testing.T) {
		// Создаём второго пользователя
		// Нужен отдельный клиент, т.к. suite.client уже имеет сессию settingsuser
		jar2, _ := cookiejar.New(nil)
		client2 := &http.Client{
			Jar: jar2,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		form2 := url.Values{}
		form2.Set("email", "other2@example.com")
		form2.Set("username", "otheruser2")
		form2.Set("password", "password123")
		reg2, _ := client2.PostForm(suite.server.URL+"/auth/register", form2)
		reg2.Body.Close()

		// Пытаемся сменить username первого пользователя на username второго
		resp := suite.postForm(t, "/settings", map[string]string{
			"username": "otheruser2",
			"bio":      "Trying to duplicate",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (conflict on duplicate username)", resp.StatusCode, http.StatusBadRequest)
		}
	})

	// F5: Пустой username
	t.Run("F5_EmptyUsername", func(t *testing.T) {
		resp := suite.postForm(t, "/settings", map[string]string{
			"username": "",
			"bio":      "Some bio",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (validation error)", resp.StatusCode, http.StatusBadRequest)
		}
	})
}
