# План локального End-to-End тестирования: teblorum

**Версия:** 1.0  
**Дата:** май 2026  
**Автор:** AI Automation Expert

---

## 1. Технологический стек тестирования

### Текущее состояние

| Компонент | Инструмент | Статус |
|-----------|-----------|--------|
| Unit-тесты | `testing` (stdlib) | ✅ Есть (19 файлов) |
| Table-driven tests | `testing` + `t.Run` | ✅ Есть |
| In-memory SQLite | `:memory:` + `modernc.org/sqlite` | ✅ Есть |
| HTTP-тесты middleware | `net/http/httptest` | ✅ Есть |
| Golden-файлы | `testdata/*.golden` | ✅ Есть |
| Handler-тесты (E2E) | `net/http/httptest` | ❗ Отсутствуют |

### Предлагаемый стек для E2E

| Инструмент | Назначение | Обоснование |
|-----------|-----------|-------------|
| **`testing`** (stdlib) | Runner тестов | Уже используется, zero external deps |
| **`net/http/httptest`** | HTTP-сервер/клиент | Полный цикл запрос-ответ, проверка статусов, заголовков, тела |
| **`net/http/cookiejar`** | Сессии | Для тестов аутентификации (сохранение cookie между запросами) |
| **`testing/fstest.MapFS`** | Шаблоны в тестах | Уже используется в `render/` |
| **In-memory SQLite** | База данных | Уже используется в repo/service тестах |
| **HTMX-симуляция** | Кастомные заголовки | HTMX не требует браузера — HX-Request/HX-Redirect в заголовках |
| **Golden-файлы** | Верификация HTML | Уже используется; сравнение ответов с эталонными HTML-файлами |

### Почему не выбраны альтернативы

| Инструмент | Причина отказа |
|-----------|---------------|
| **Selenium / Playwright / Rod** | 500 MB RAM, Docker запрещён; полноценный браузер избыточен для SSR+HTMX |
| **Testcontainers-go** | Требует Docker |
| **Cypress** | Node.js + браузер, оверхед для Go-проекта |
| **httpexpect** | Внешняя зависимость; httptest достаточно |
| **golang.org/x/net/html** | Для парсинга HTML полезен, но не обязателен |

### Рекомендуемая структура тестового пакета

```
internal/
└── handler/
    ├── e2e_test.go          ← основной файл E2E-тестов
    ├── e2e_testdata/         ← golden-файлы для E2E
    │   ├── feed_anonymous.golden
    │   ├── login_form.golden
    │   ├── article_page.golden
    │   └── ...
    └── e2e_helpers.go       ← вспомогательные функции (setupTestServer, seedData)
```

---

## 2. Функциональные группы тестов

Каждая группа представляет законченный пользовательский сценарий «от и до» через HTTP API.

### Группа A: Анонимный пользователь (публичный доступ)

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| A1 | **Просмотр ленты** | GET `/` | 200 OK, HTML содержит «teblorum», пагинация |
| A2 | **Просмотр статей** | GET `/articles` | 200 OK, список статей или пусто |
| A3 | **Просмотр тредов** | GET `/threads` | 200 OK, список тредов |
| A4 | **Просмотр статьи** | GET `/articles/{id}` | 200 OK, заголовок, тело в HTML |
| A5 | **Просмотр треда** | GET `/threads/{id}` | 200 OK, заголовок, тело |
| A6 | **Просмотр профиля** | GET `/users/{username}` | 200 OK, username на странице |
| A7 | **Форма логина** | GET `/auth/login` | 200 OK, форма входа |
| A8 | **Форма регистрации** | GET `/auth/register` | 200 OK, форма регистрации |
| A9 | **Форма восстановления** | GET `/auth/forgot` | 200 OK, форма |
| A10 | **404 страница** | GET `/nonexistent` | 404, сообщение об ошибке |
| A11 | **HTMX-пагинация ленты** | GET `/` с `HX-Request: true` | 200, только пагинация, без layout |
| A12 | **Google OAuth redirect** | GET `/auth/google` | 302 или 200, редирект на Google |

### Группа B: Регистрация и аутентификация

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| B1 | **Успешная регистрация** | POST `/auth/register` (email, username, password) | 302 → `/`, Set-Cookie: session_id |
| B2 | **Дубликат email** | POST `/auth/register` (существующий email) | 409 Conflict, сообщение об ошибке |
| B3 | **Дубликат username** | POST `/auth/register` (существующий username) | 409 Conflict |
| B4 | **Пустой пароль** | POST `/auth/register` (password = "") | 422 или 400, ошибка валидации |
| B5 | **Успешный вход** | POST `/auth/login` (email + password) | 302 → `/`, Set-Cookie: session_id |
| B6 | **Неверный пароль** | POST `/auth/login` (wrong password) | 401 Unauthorized |
| B7 | **Несуществующий email** | POST `/auth/login` (unknown email) | 401 Unauthorized |
| B8 | **Выход** | POST `/auth/logout` (с cookie) | 302 → `/`, cookie удалена |
| B9 | **Доступ без сессии** | GET `/settings` без cookie | 302 → `/auth/login` |
| B10 | **HTMX: вход без сессии** | GET `/settings` с `HX-Request: true` | 401, HX-Redirect: /auth/login |

### Группа C: Создание и управление публикациями

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| C1 | **Создание статьи** | POST `/articles` (title + body) | 302 → `/articles/{id}`, статья в БД |
| C2 | **Создание треда** | POST `/threads` (title + body) | 302 → `/threads/{id}`, тред в БД |
| C3 | **Пустой заголовок** | POST `/articles` (title = "") | 400 Bad Request |
| C4 | **Слишком длинный заголовок** | POST `/articles` (title > 200 chars) | 400 Bad Request |
| C5 | **Редактирование статьи** | POST `/articles/{id}` (новый title) | 302, title обновлён |
| C6 | **Редактирование треда** | POST `/threads/{id}` (новый body) | 302, body обновлён |
| C7 | **Редактирование чужой статьи** | POST `/articles/{id}` (другой автор) | 403 Forbidden |
| C8 | **Редактирование удалённой статьи** | POST `/articles/{id}` (deleted) | 404 Not Found |
| C9 | **Форма создания статьи** | GET `/articles/new` | 200 OK, форма |
| C10 | **Форма создания треда** | GET `/threads/new` | 200 OK, форма |
| C11 | **Форма редактирования** | GET `/articles/{id}/edit` | 200 OK, форма с данными |
| C12 | **HTMX: создание статьи** | POST `/articles` с `HX-Request` | 200, HX-Redirect |

### Группа D: Комментарии

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| D1 | **Создание комментария** | POST `/posts/{id}/comments` (body) | 200, HX-Redirect на пост |
| D2 | **Пустой комментарий** | POST `/posts/{id}/comments` (body = "") | 400 Bad Request |
| D3 | **Ответ на комментарий** | POST `/posts/{id}/comments` (body + parent_id) | 200, комментарий с parent_id |
| D4 | **Комментарий к несуществующему посту** | POST `/posts/99999/comments` | 404 Not Found |
| D5 | **Комментарий без аутентификации** | POST `/posts/{id}/comments` (без cookie) | 302 → `/auth/login` |
| D6 | **Форма ответа** | GET `/comments/reply-form?parent_id=N` | 200, HTML-форма |
| D7 | **Загрузка дочерних комментариев** | GET `/comments/{id}/children` | 200, HTML-список |
| D8 | **Комментарий при отключенных комментариях** | POST к article с comments_enabled=0 | 403 Forbidden |

### Группа E: Личные сообщения

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| E1 | **Список диалогов** | GET `/messages` | 200 OK, список |
| E2 | **Открыть диалог** | GET `/messages/{username}` | 200 OK, история сообщений |
| E3 | **Отправить сообщение** | POST `/messages/{username}` (body) | 302 или 200, сообщение в БД |
| E4 | **Отправка самому себе** | POST `/messages/{self}` | 400 Bad Request |
| E5 | **Несуществующий получатель** | POST `/messages/nobody` | 404 Not Found |
| E6 | **Пустое сообщение** | POST `/messages/{username}` (body = "") | 400 Bad Request |
| E7 | **Диалог с несуществующим** | GET `/messages/nobody` | 404 Not Found |
| E8 | **Сообщение без аутентификации** | POST `/messages/{username}` (без cookie) | 302 → `/auth/login` |

### Группа F: Настройки профиля

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| F1 | **Страница настроек** | GET `/settings` | 200 OK, форма |
| F2 | **Обновление bio** | POST `/settings` (new bio) | 302 или 200, bio обновлён |
| F3 | **Смена username** | POST `/settings` (new username) | 302, username изменён |
| F4 | **Дубликат username** | POST `/settings` (существующий username) | 409 Conflict |
| F5 | **Пустой username** | POST `/settings` (username = "") | 400 Bad Request |

### Группа G: Модерация (Moderator+)

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| G1 | **Удаление поста (модератор)** | POST `/mod/posts/{id}/delete` (mod session) | 302, post.deleted_at ≠ NULL |
| G2 | **Удаление комментария (модератор)** | POST `/mod/comments/{id}/delete` (mod session) | 302, comment.deleted_at ≠ NULL |
| G3 | **Бан пользователя** | POST `/mod/users/{id}/ban` (mod session) | 302, user.banned_until ≠ NULL |
| G4 | **Разбан пользователя** | POST `/mod/users/{id}/unban` (mod session) | 302, user.banned_until = NULL |
| G5 | **Бан пользователя (user role)** | POST `/mod/users/{id}/ban` (user session) | 403 Forbidden |
| G6 | **Бан root (moderator)** | POST `/mod/users/{id}/ban` (target=root) | 403 Forbidden |
| G7 | **Удаление поста (user)** | POST `/mod/posts/{id}/delete` (user session) | 403 Forbidden |
| G8 | **Бан уже забаненного** | POST `/mod/users/{id}/ban` (already banned) | 200, дата обновлена |

### Группа H: Root-панель

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| H1 | **Панель root** | GET `/root` (root session) | 200 OK, панель |
| H2 | **Панель root (user)** | GET `/root` (user session) | 403 Forbidden |
| H3 | **Создание бекапа** | POST `/root/backup` (root session) | 200, файл .db создан |
| H4 | **Повышение до moderator** | POST `/root/users/{id}/promote` (root→user) | 302, user.role = moderator |
| H5 | **Понижение до user** | POST `/root/users/{id}/demote` (root→moderator) | 302, user.role = user |
| H6 | **Повышение root (user)** | POST `/root/users/{id}/promote` (user session) | 403 Forbidden |
| H7 | **Восстановление из бекапа** | POST `/root/restore` (root session, file) | 200, БД восстановлена |

### Группа I: Markdown-рендеринг

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| I1 | **Базовая статья с Markdown** | Создать статью с `# Title\n\n**bold**` | HTML содержит `<h1>`, `<strong>` |
| I2 | **XSS-безопасность** | Создать пост с `<script>alert(1)</script>` | HTML экранирован, нет <script> |
| I3 | **Ссылки** | Создать пост с `[link](https://x.com)` | HTML содержит `<a>` с rel="noopener" |
| I4 | **Изображения** | Создать пост с `![img](url)` | HTML содержит `<img>` (или удалено) |
| I5 | **Код** | Создать пост с `code` | HTML содержит `<code>` |

### Группа J: Комплексные сценарии (multi-step)

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| J1 | **Полный цикл: регистрация → создание статьи → комментарий** | POST register → POST article → POST comment | Все 3 запроса успешны |
| J2 | **Регистрация → создание треда → ответ в тред** | POST register → POST thread → POST comment | Все 3 запроса успешны |
| J3 | **Вход → создание статьи → редактирование → удаление модератором** | POST login → POST article → POST edit → POST mod delete | Все шаги успешны |
| J4 | **Диалог: UserA → UserB → ответ** | POST message A→B → GET conversation B | Сообщение в истории обоих |
| J5 | **Бан пользователя → попытка входа** | POST ban → POST login | 401 Banned |
| J6 | **Регистрация → создание поста → просмотр ленты анонимом** | POST register → POST article → GET / (no auth) | Пост в ленте |
| J7 | **Пагинация: 35 постов** | Создать 35 постов → GET / → GET /?page=2 | 30 на стр.1, 5 на стр.2 |
| J8 | **Root: promote до модератора → модератор банит пользователя** | POST promote → POST ban (mod session) | Бан успешен |

### Группа K: Ошибки и граничные случаи

| # | Сценарий | Шаги | Проверки |
|---|---------|------|---------|
| K1 | **SQL-инъекция в username** | POST register (username = `' OR 1=1 --`) | 409 или 400, не 500 |
| K2 | **SQL-инъекция в email** | POST register (email = `'; DROP TABLE users;--`) | 409, таблица users существует |
| K3 | **Очень длинный username** | POST register (username = 256 chars) | 400 Bad Request |
| K4 | **Очень длинное bio** | POST settings (bio = 10000 chars) | 400 или 413 |
| K5 | **Невалидный email** | POST register (email = "notanemail") | 400 Bad Request |
| K6 | **ID = 0 или отрицательный** | GET `/articles/-1` | 404 Not Found |
| K7 | **ID = строка** | GET `/articles/abc` | 404 Not Found |
| K8 | **Пустой POST body** | POST `/auth/login` (no form data) | 400 Bad Request |
| K9 | **Неверный Content-Type** | POST `/auth/login` (application/json) | 400 Bad Request |
| K10 | **Rate limit превышен** | 61+ запросов за минуту | 429 Too Many Requests |

---

## 3. Инструкции по запуску тестов

### 3.1 Запуск всех тестов (единая команда)

```bash
# Полный прогон всех тестов, включая E2E
go test ./... -count=1 -race -timeout 120s
```

Флаги:
- `-count=1` — отключает кеширование
- `-race` — детектор гонок данных
- `-timeout 120s` — таймаут на весь прогон (E2E тесты медленнее)

### 3.2 Запуск конкретных групп

#### Все unit-тесты (без E2E)

```bash
# Модели
go test ./internal/model/... -count=1 -race -v

# Репозитории (in-memory SQLite)
go test ./internal/repo/... -count=1 -race -v

# Аутентификация
go test ./internal/auth/... -count=1 -race -v

# Middleware
go test ./internal/middleware/... -count=1 -race -v

# Сервисы
go test ./internal/service/... -count=1 -race -v

# Рендеринг
go test ./internal/render/... -count=1 -race -v

# Markdown
go test ./internal/markdown/... -count=1 -race -v
```

#### E2E-тесты handler-ов (группы A–K)

```bash
# Все E2E-тесты
go test ./internal/handler/... -count=1 -race -v -run "^TestE2E"

# Конкретная группа
go test ./internal/handler/... -count=1 -race -v -run "^TestE2E_GroupA"
go test ./internal/handler/... -count=1 -race -v -run "^TestE2E_GroupB"
go test ./internal/handler/... -count=1 -race -v -run "^TestE2E_GroupC"
# и т.д.

# Один конкретный сценарий
go test ./internal/handler/... -count=1 -race -v -run "^TestE2E_GroupA/A1_Feed"

# Комплексные сценарии
go test ./internal/handler/... -count=1 -race -v -run "^TestE2E_GroupJ"
```

### 3.3 Запуск с coverage

```bash
# Покрытие всех тестов
go test ./... -count=1 -race -coverprofile=coverage.out -timeout 120s
go tool cover -html=coverage.out -o coverage.html

# Покрытие только handler/E2E
go test ./internal/handler/... -count=1 -race -coverprofile=handler_coverage.out
go tool cover -html=handler_coverage.out -o handler_coverage.html

# Покрытие конкретной группы
go test ./internal/handler/... -count=1 -race -coverprofile=e2e_coverage.out \
  -run "^TestE2E_Group[ABC]"
```

### 3.4 Быстрая проверка перед коммитом

```bash
# Минимальный прогон
go vet ./...
go test ./internal/model/... ./internal/auth/... ./internal/repo/... -count=1 -race
```

### 3.5 Запуск с verbose для диагностики

```bash
# Подробный вывод по E2E-тестам
go test ./internal/handler/... -count=1 -race -v -run "^TestE2E" 2>&1 | head -200
```

---

## 4. Структура тестового файла (шаблон)

```go
// internal/handler/e2e_helpers.go
package handler

import (
    "database/sql"
    "net/http"
    "net/http/cookiejar"
    "net/http/httptest"
    "testing"

    "github.com/teblorum/teblorum/internal/repo"
    "github.com/teblorum/teblorum/webassets"
)

// e2eTestSuite содержит состояние E2E-теста.
type e2eTestSuite struct {
    db     *sql.DB
    server *httptest.Server
    client *http.Client
}

// newE2ETestSuite создаёт полностью настроенный тестовый стенд.
// — In-memory SQLite с миграциями
// — Полный HTTP-сервер со всеми middleware и маршрутами
// — HTTP-клиент с cookie jar для сессий
func newE2ETestSuite(t *testing.T) *e2eTestSuite {
    t.Helper()

    db := repo.NewTestDB(t)
    // ... инициализация сервисов и Dependencies ...
    // ... SetupRoutes с тестовыми шаблонами ...
    server := httptest.NewServer(h)
    jar, _ := cookiejar.New(nil)

    t.Cleanup(server.Close)

    return &e2eTestSuite{
        db:     db,
        server: server,
        client: &http.Client{Jar: jar},
    }
}

// request выполняет HTTP-запрос и возвращает ответ.
func (s *e2eTestSuite) request(t *testing.T, method, path string, body ...string) *http.Response {
    t.Helper()
    // ...
}
```

```go
// internal/handler/e2e_test.go
package handler

import (
    "testing"
    "net/http"
)

// TestE2E_GroupA — анонимный пользователь, публичный доступ.
func TestE2E_GroupA(t *testing.T) {
    suite := newE2ETestSuite(t)

    t.Run("A1_Feed", func(t *testing.T) {
        resp := suite.request(t, http.MethodGet, "/")
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
        }
        // Проверка содержимого: содержит "teblorum", пагинацию
    })

    t.Run("A4_ArticlePage", func(t *testing.T) {
        // Сначала создаём статью через API
        // Затем GET /articles/{id}
    })
}
```

---

## 5. Рекомендуемый порядок реализации тестов

1. **Фаза 1 — Helpers** (e2e_helpers.go): `newE2ETestSuite`, вспомогательные функции
2. **Фаза 2 — Группа A** (анонимный доступ): базовая проверка, что сервер отвечает
3. **Фаза 3 — Группа B** (регистрация/логин): основа для всех остальных тестов
4. **Фаза 4 — Группы C–F** (посты, комментарии, сообщения, настройки)
5. **Фаза 5 — Группы G–H** (модерация, root)
6. **Фаза 6 — Группы I–K** (рендеринг, комплексные сценарии, ошибки)

> **Правило:** каждый тест должен быть изолирован — использовать свою БД (`:memory:`) и свой `httptest.Server`. Параллельный запуск (`t.Parallel`) безопасен, т.к. нет общего состояния.

---

## 6. Ожидаемые метрики

| Метрика | Цель |
|---------|------|
| Количество E2E-сценариев | ~60 (A:12, B:10, C:12, D:8, E:8, F:5, G:8, H:7, I:5, J:8, K:10) |
| Покрытие handler-ов | >85% |
| Покрытие service-слоя | >90% |
| Время полного прогона | <30 сек (in-memory SQLite, без сети) |
| Время прогона одной группы | <3 сек |