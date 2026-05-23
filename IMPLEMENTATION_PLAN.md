# План имплементации: teblorum — платформа текстовых публикаций

**Версия плана:** 1.0  
**Дата:** май 2026  
**Архитектор:** Senior Fullstack Developer / Software Architect

---

## Принципы

- **SOLID**: каждый пакет имеет единственную ответственность; зависимости направлены от handler → service → repo.
- **YAGNI**: ничего лишнего — только то, что описано в спецификации.
- **KISS**: без абстракций «на вырост»; без DI-контейнеров; без глобальных синглтонов.
- **TDD-first**: каждый шаг включает написание тестов до или сразу после реализации. Переход к следующему шагу — только когда `go test ./... -count=1 -race` проходит без ошибок.

---

## Подход к тестированию

### Инструменты

- **Стандартная библиотека** (`testing` + `testing/fstest` + `net/http/httptest`) — zero external dependencies для тестов.
- **Table-driven tests** — Go-идиома для покрытия кейсов (success, validation error, not found, forbidden).
- **In-memory SQLite** (`:memory:`) — для всех тестов repo-слоя. Каждый тест создаёт свою чистую БД.
- **`httptest.NewServer`** + `httptest.NewRequest` — для интеграционных тестов handler-ов.
- **Golden files** (`testdata/*.golden`) — для шаблонов и Markdown-рендеринга.
- **Subtests** (`t.Run`) — для группировки кейсов внутри table-driven тестов.

### Структура тестовых файлов

```
internal/
├── model/
│   ├── role.go
│   ├── role_test.go          # тесты уровней ролей, CanActOn
│   └── errors_test.go        # тесты sentinel-ошибок (Is, As)
├── repo/
│   ├── users.go
│   ├── users_test.go         # in-memory SQLite, каждая таблица
│   ├── posts.go
│   ├── posts_test.go
│   └── ...
├── service/
│   ├── user.go
│   ├── user_service_test.go  # repo mock или in-memory SQLite
│   └── ...
├── handler/
│   ├── posts.go
│   ├── posts_handler_test.go # httptest.NewServer, полный HTTP-цикл
│   └── ...
├── middleware/
│   ├── auth.go
│   ├── auth_test.go          # цепочка middleware, httptest
│   └── ...
├── markdown/
│   ├── render.go
│   └── render_test.go        # golden files
└── render/
    ├── render.go
    └── render_test.go         # template execution, golden files
```

### Правило перехода между шагами

```bash
# Обязательно после каждого шага:
go vet ./...           # статический анализ
go test ./... -count=1 -race   # без кеша, с race detector

# Если OK → коммит → следующий шаг.
# Если FAIL → стоп, исправление, повтор.
```

### Фикстуры для тестов

- `internal/repo/testhelpers.go` — вспомогательные функции:
  - `NewTestDB(t *testing.T) *sql.DB` — открывает `:memory:`, накатывает миграцию, регистрирует `t.Cleanup`.
  - `SeedUser(t, db, attrs) model.User` — создаёт пользователя с кастомными полями.
  - `SeedPost(t, db, attrs) model.Post`.

---

## Легенда этапов

| Этап | Описание |
|------|----------|
| **Core** | Фундамент, без которого ничего не работает |
| **Feature** | Функциональность, доступная пользователю |
| **Infra** | Инфраструктура, деплой, безопасность |

---

## Этап 0: Скаффолдинг проекта и базовая структура

**Цель:** создать дерево директорий, инициализировать Go-модуль, определить глобальные типы и error sentinels.

### Шаг 0.1 — Инициализация модуля

```bash
go mod init github.com/teblorum/teblorum
```

### Шаг 0.2 — Создание директорий по спецификации п.18

```
teblorum/
├── cmd/teblorum/
├── internal/
│   ├── auth/
│   ├── handler/
│   ├── middleware/
│   ├── model/
│   ├── repo/
│   ├── service/
│   ├── render/
│   └── markdown/
├── web/templates/
│   ├── partials/
│   └── pages/
├── web/static/
└── migrations/
```

### Шаг 0.3 — Определение общих типов и констант

- `internal/model/role.go`: тип `Role` со списком констант `RoleUser`, `RoleModerator`, `RoleAdmin`, `RoleRoot` и методом `Level() int`, а также функциями `CanActOn(actor, target Role) bool` и `MinRole(required Role) bool`.
- `internal/model/errors.go`: sentinel-ошибки — `ErrNotFound`, `ErrForbidden`, `ErrUnauthorized`, `ErrValidation`, `ErrRateLimited`, `ErrBanned`.
- `internal/model/types.go`: общие типы — `ContextKey` для ключей контекста (user, session, csrf), `PaginationParams` (page, limit), `PaginationMeta` (total, page, totalPages).

### Шаг 0.4 — Установка зависимостей

```go
go get modernc.org/sqlite@v1.29.9
go get golang.org/x/crypto@v0.23.0
go get golang.org/x/oauth2@v0.20.0
go get github.com/microcosm-cc/bluemonday@v1.0.27
go get github.com/yuin/goldmark@v1.7.1
```

### Шаг 0.5 — Тестирование моделей

- `internal/model/role_test.go`:
  - Table-driven тесты для `Level()`: все 4 роли + пустая строка (panics or returns -1).
  - Таблица для `CanActOn(actor, target)`: минимум 20 комбинаций (mod→user=true, mod→admin=false, root→root=false, user→user=false, и т.д.).
- `internal/model/errors_test.go`:
  - Проверка, что все sentinel-ошибки правильно работают с `errors.Is()`.
- `internal/model/types_test.go`:
  - Тесты для `PaginationParams.Offset()`, `PaginationMeta.TotalPages()`.
- **Gate**: `go vet ./internal/model/... && go test ./internal/model/... -count=1 -race` → OK.
## Этап 1: База данных и слой доступа к данным (Core)

**Цель:** работа с SQLite, миграции, репозитории.

### Шаг 1.1 — Миграция

- `migrations/001_init.sql`: полная схема из спецификации п.4.1 (6 таблиц + индексы + `ALTER TABLE` для `banned_by`).
- `internal/repo/migrate.go`: функция `RunMigrations(db *sql.DB)`. Читает `001_init.sql` из `embed.FS`.
  - Первая строка: проверка таблицы `schema_migrations` (версия), если таблицы нет — применить `001_init.sql`.
  - Простейший versioning: запись `(1, datetime('now'))` в `schema_migrations`.
  - **Решение**: хранить версию миграции в отдельной таблице или проверять наличие одной из ключевых таблиц (`users`). Выбрать таблицу `schema_migrations`.

### Шаг 1.2 — Подключение к БД

- `internal/repo/db.go`:
  - Функция `OpenDB(path string) (*sql.DB, error)`.
  - Открывает SQLite, выполняет PRAGMA из п.4.2.
  - `db.SetMaxOpenConns(1)`, `db.SetMaxIdleConns(1)`.
  - Вызывает `RunMigrations`.

### Шаг 1.3 — Репозитории

Каждый репозиторий — отдельный файл в `internal/repo/`. Принимает `*sql.DB`. Все методы возвращают `(model, error)`. Только параметризованные запросы.

| Файл | Методы |
|------|--------|
| `users.go` | `Create`, `GetByID`, `GetByEmail`, `GetByUsername`, `GetByGoogleID`, `Update`, `UpdateRole`, `SetBan`, `RemoveBan`, `ExistsByEmail`, `ExistsByUsername` |
| `posts.go` | `Create`, `GetByID`, `GetByType` (пагинация), `GetFeed` (смешанная лента, пагинация), `Update`, `SoftDelete`, `GetByAuthorID` (пагинация) |
| `comments.go` | `Create`, `GetByPostID` (с рекурсивным CTE для дерева), `GetChildren` (для HTMX-подгрузки), `SoftDelete` |
| `messages.go` | `Send`, `GetConversations` (список диалогов с последним сообщением и unread count), `GetConversation` (с конкретным пользователем, пагинация), `MarkRead`, `GetUnreadCount` |
| `sessions.go` | `Create`, `GetByID`, `DeleteByID`, `DeleteByUserID`, `CleanExpired` |

### Шаг 1.4 — Модели

- `internal/model/user.go`: `User` struct (все поля из `users` таблицы), `CreateUserRequest`, `UpdateUserRequest`.
- `internal/model/post.go`: `Post` struct, `CreatePostRequest`, `UpdatePostRequest`.
- `internal/model/comment.go`: `Comment` struct, `CommentTreeNode` (Comment + Children []CommentTreeNode), `CreateCommentRequest`.
- `internal/model/message.go`: `Message` struct, `ConversationSummary` (собеседник, последнее сообщение, unread count).

### Шаг 1.5 — Тестирование репозиториев

- `internal/repo/testhelpers.go`: `NewTestDB(t) *sql.DB` — `:memory:` + миграция + `t.Cleanup(db.Close)`.
- `internal/repo/users_test.go` — для каждого метода:
  - `TestCreateUser`: success, duplicate email, duplicate username.
  - `TestGetByID`: found, not found.
  - `TestGetByEmail`: found, not found.
  - `TestGetByUsername`: found, not found.
  - `TestUpdate`: смена bio, username.
  - `TestSetBan` / `TestRemoveBan`.
- `internal/repo/posts_test.go`:
  - `TestCreatePost`: article vs thread, проверка полей.
  - `TestGetFeed`: пагинация, порядок по `created_at DESC`.
  - `TestSoftDelete`: `deleted_at` заполнен, GetByID игнорирует мягко-удалённые.
- `internal/repo/comments_test.go`:
  - `TestCreateComment`: root, reply (parent_id).
  - `TestGetByPostID`: рекурсивный CTE — проверка глубины и порядка.
- `internal/repo/messages_test.go`:
  - `TestSend`, `TestGetConversations`, `TestGetUnreadCount`.
- `internal/repo/sessions_test.go`:
  - `TestCreateSession`, `TestGetByID`, `TestCleanExpired`.
- **Gate**: `go test ./internal/repo/... -count=1 -race` → OK.

---

## Этап 2: Аутентификация (Core)

**Цель:** регистрация, вход, сессии, OAuth, сброс пароля.

### Шаг 2.1 — Bcrypt + сессии

- `internal/auth/password.go`: `HashPassword(plain) (hash, error)`, `CheckPassword(hash, plain) bool`.
- `internal/auth/session.go`:
  - `GenerateSessionID() string` (32 байта → hex).
  - `CreateSession(db, userID, ttlDays) (Session, error)`.
  - `GetSession(db, sessionID) (Session, error)`.
  - `RefreshSession(db, sessionID, ttlDays) error` (скользящее окно — обновляем `expires_at`).
  - `DeleteSession(db, sessionID) error`.

### Шаг 2.2 — Google OAuth

- `internal/auth/oauth.go`:
  - `NewGoogleOAuthConfig(clientID, secret, redirectURL) *oauth2.Config`.
  - `GetGoogleLoginURL(config, state) string`.
  - `ExchangeCode(config, code) (*oauth2.Token, error)`.
  - `GetGoogleUserInfo(token) (email, googleID, name, error)`.

### Шаг 2.3 — Сброс пароля

- `internal/auth/password_reset.go`:
  - Таблица `password_resets` в миграции: `email TEXT, token TEXT UNIQUE, expires_at DATETIME`.
  - `CreateResetToken(db, email) (token, error)`.
  - `ValidateResetToken(db, token) (email, error)`.
  - `DeleteResetToken(db, token)`.

### Шаг 2.4 — Middleware-проверка сессии

- `internal/middleware/auth.go`:
  - `RequireAuth(next http.Handler) http.Handler`: читает cookie `session_id`, ищет сессию в БД, кладёт `user` и `session` в контекст. Если сессия невалидна — редирект на `/auth/login`.
  - `OptionalAuth(next http.Handler) http.Handler` (для публичных страниц): если есть валидная сессия — кладёт в контекст; если нет — анонимный контекст.
  - Оба middleware обновляют `expires_at` (скользящее окно).

### Шаг 2.5 — Тестирование auth-пакета

- `internal/auth/password_test.go`:
  - `TestHashAndCheck`: хеш + проверка правильного пароля.
  - `TestWrongPassword`: хеш + проверка неверного пароля → false.
  - `TestHashCost`: проверка, что bcrypt cost = 12 (через извлечение cost из хеша).
- `internal/auth/session_test.go`:
  - `TestGenerateSessionID`: длина, hex-формат.
  - `TestCreateAndGetSession`: in-memory SQLite, проверка TTL.
  - `TestRefreshSession`: обновление `expires_at`.
  - `TestCleanExpired`: удаление просроченных сессий.
- `internal/middleware/auth_test.go`:
  - `TestRequireAuthNoCookie`: редирект 302.
  - `TestRequireAuthValidSession`: 200, user в контексте.
  - `TestOptionalAuthAnonymous`: 200, user = nil в контексте.
  - `TestSlidingWindow`: `expires_at` обновляется после запроса.
- **Gate**: `go test ./internal/auth/... ./internal/middleware/... -count=1 -race` → OK.

---

## Этап 3: Рендеринг шаблонов (Core)

**Цель:** SSR с HTMX-поддержкой.

### Шаг 3.1 — Базовая структура шаблонов

- `web/templates/layout.html`: `<html>` с `<head>`, `<body>` + навигация (п.10.5), `<main>`, `<footer>`.
  - Использует `block "content"` для вставки страниц.
  - Использует `block "head"` для дополнительных `<link>`/`<script>`.
- `web/templates/pages/*.html`: каждая страница — расширение `layout.html`.
- `web/templates/partials/*.html`: HTMX-фрагменты без layout.

### Шаг 3.2 — Render-пакет

- `internal/render/render.go`:
  - `type TemplateRenderer struct { layout *template.Template, partials *template.Template, pages map[string]*template.Template }`.
  - `NewTemplateRenderer(fs embed.FS) (*TemplateRenderer, error)` — парсит все шаблоны при старте, кеширует.
  - `func (r *TemplateRenderer) Page(w http.ResponseWriter, name string, data any, isHTMX bool)`:
    - Если `isHTMX` → рендерит только блок контента (через `{{ template "content" . }}`).
    - Иначе → рендерит полный `layout.html`.
  - `func (r *TemplateRenderer) Partial(w http.ResponseWriter, name string, data any)` — рендерит partial-шаблон.
  - `DetectHTMX(r *http.Request) bool`: проверяет заголовок `HX-Request`.

### Шаг 3.3 — Данные для шаблонов

- `internal/render/data.go`: общие структуры данных для шаблонов.
  - `PageData`: `Title string`, `CurrentUser *model.User`, `IsHTMX bool`, `Theme string`, `FontSize string`, `UnreadCount int`, `CsrfToken string`.
  - Все структуры для конкретных страниц — `PostPageData`, `FeedPageData`, `UserPageData` и т.д.

### Шаг 3.4 — Обработка ошибок

- `internal/render/errors.go`: страницы 404, 403, 500 в виде шаблонов или функций-заглушек.

### Шаг 3.5 — Тестирование рендеринга

- `internal/render/render_test.go`:
  - `TestDetectHTMX`: запрос с заголовком `HX-Request: true` → true; без заголовка → false.
  - `TestRenderPage`: вызов `Page()` → проверка, что layout + контент срендерены.
  - `TestRenderPartial`: вызов `Partial()` → только фрагмент, без layout.
  - `TestRenderPageHTMX`: `isHTMX=true` → без layout, только блок `content`.
- Golden files в `internal/render/testdata/`:
  - `page_full.golden` — ожидаемый HTML полной страницы.
  - `partial_comment.golden` — ожидаемый HTML фрагмента.
- **Gate**: `go test ./internal/render/... -count=1 -race` → OK.

---

## Этап 4: CSS и статика (Core)

**Цель:** встроить статические файлы в бинарник, написать минималистичный CSS.

### Шаг 4.1 — embed.FS

- В `cmd/teblorum/main.go` или отдельном пакете:
  ```go
  //go:embed web/templates/** web/static/**
  var webFS embed.FS
  ```
  Передаётся в `render.NewTemplateRenderer(webFS)` и файловый сервер.

### Шаг 4.2 — CSS

- `web/static/style.css`:
  - CSS custom properties из п.10.2 (светлая + тёмная тема).
  - Адаптивность: брейкпоинты из п.10.4.
  - Минималистичная типографика: системный стек шрифтов (Stack Overflow style: `-apple-system, sans-serif`).
  - Стили для навигации, статей, тредов, комментариев, форм, кнопок, пагинации.
  - Стили для depth-отступов комментариев (`.comment-depth-1` … `.comment-depth-6`).
  - Стиль для `[data-theme]` на `:root`.

### Шаг 4.3 — HTMX

- Скачать `htmx.min.js` (v2.x) в `web/static/htmx.min.js` или через GOARCH-зависимость. **Решение**: скопировать файл вручную (однократно).

### Шаг 4.4 — Раздача статики

- В роутере: `mux.Handle("GET /static/", http.FileServer(http.FS(webFS)))`.

---

## Этап 5: Маршрутизация и HTTP-сервер (Core)

**Цель:** настроить роутер, зарегистрировать все маршруты.

### Шаг 5.1 — Инициализация роутера

- Использовать стандартный `net/http.ServeMux` (Go 1.22+ с `{id}`-паттернами).
- В `cmd/teblorum/main.go`:
  ```go
  mux := http.NewServeMux()
  // регистрация всех маршрутов
  ```

### Шаг 5.2 — Middleware-цепочка

Порядок:
1. `SecurityHeaders` — CSP, X-Frame-Options, и т.д.
2. `RateLimit` — in-memory счётчики.
3. `OptionalAuth` — сессия (если есть), иначе аноним.
4. `CSRF` — проверка double-submit cookie на POST/PUT/DELETE (кроме OAuth-колбэка).

Затем каждый handler получает уже готовый контекст.

### Шаг 5.3 — Регистрация всех маршрутов

Сгруппировать по категориям согласно п.11:

- **Публичные** — без middleware-проверки роли.
- **Авторизованные (User+)** — `RequireAuth`.
- **Moderator+ (модерация)** — `RequireRole(model.RoleModerator)`.
- **Root** — `RequireRole(model.RoleRoot)`.

```go
// Пример группировки через замыкания
mux.Handle("GET /articles/new", requireAuth(requireRole(RoleModerator, handleNewArticle)))
```

---

## Этап 6: Service-слой (Core)

**Цель:** бизнес-логика, отделённая от handler и repo.

### Шаг 6.1 — UserService

- `Register(email, username, password)`: валидация → проверка уникальности → bcrypt → repo.Create.
- `Login(email, password)`: repo.GetByEmail → CheckPassword → CreateSession.
- `LoginOrRegisterGoogle(googleID, email, name)`: поиск по google_id → если нет, поиск по email → если нет, создание → CreateSession.
- `GetByUsername(username)`: repo.GetByUsername.
- `UpdateProfile(userID, bio, ...)`: repo.Update.
- `CanModerate(actor, target User) bool`: сравнение уровней ролей.

### Шаг 6.2 — PostService

- `CreateArticle(authorID, title, body, commentsEnabled)`: валидация → repo.Create.
- `CreateThread(authorID, title, body)`: валидация → repo.Create.
- `GetFeed(page)`: repo.GetFeed.
- `GetByID(id)`: repo.GetByID → генерация slug.
- `Update(id, actor, title, body, ...)`: проверка прав (автор или admin/root) → repo.Update.
- `SoftDelete(id, actor)`: проверка `CanActOn` → repo.SoftDelete.
- `GenerateSlug(title)`: транслитерация + lowercase + дефисы.

### Шаг 6.3 — CommentService

- `Create(postID, authorID, parentID, body)`: проверка `post.comments_enabled` (для article), проверка `parent_id` существует, проверка depth ≤ 6 → repo.Create.
- `GetTree(postID)`: repo.GetByPostID → построение дерева.
- `SoftDelete(id, actor)`: проверка `CanActOn` → repo.SoftDelete.

### Шаг 6.4 — MessageService

- `Send(fromID, toID, body)`: проверка существования `to`, проверка `from != to` → repo.Send.
- `GetConversations(userID)`: repo.GetConversations.
- `GetConversation(userID, withUserID)`: repo.GetConversation.
- `GetUnreadCount(userID)`: repo.GetUnreadCount.

### Шаг 6.5 — ModerationService

- `BanUser(targetID, actorID, duration)`: проверка `CanActOn(actor, target)` → repo.SetBan.
- `UnbanUser(targetID, actorID)`: проверка `CanActOn(actor, target)` → repo.RemoveBan.
- `PromoteToModerator(targetID, actorID)`: проверка `actor.role == RoleRoot` → repo.UpdateRole.
- `DemoteFromModerator(targetID, actorID)`: проверка `actor.role == RoleRoot` → repo.UpdateRole.

### Шаг 6.6 — RootService

- `CreateBackup() (filename, error)`: `VACUUM INTO`.
- `RestoreFromUpload(file multipart.File) error`: `PRAGMA integrity_check` → замена файла БД.
- `CleanupOldBackups(maxCount int)`: удаление старых бекапов (хранить 10 последних).

### Шаг 6.7 — Тестирование service-слоя

- Каждый сервис тестируется с in-memory SQLite + репозитории (без моков — integration-style).
- `internal/service/user_service_test.go`:
  - `TestRegister`: success, duplicate email, duplicate username, short password.
  - `TestLogin`: success, wrong password, non-existent email.
  - `TestCanModerate`: moderator→user=true, user→moderator=false, root→all=true.
- `internal/service/post_service_test.go`:
  - `TestCreateArticle`: success, валидация лимитов.
  - `TestCreateThread`: success.
  - `TestGenerateSlug`: "Привет Мир" → "privet-mir".
  - `TestSoftDelete`: автор удаляет свой пост; moderator удаляет чужой пост user.
- `internal/service/comment_service_test.go`:
  - `TestCreateComment`: success, parent depth > 6 rejected, comments_disabled rejected.
- `internal/service/message_service_test.go`:
  - `TestSend`: success, самому себе rejected.
- `internal/service/moderation_service_test.go`:
  - `TestBanUser`: moderator→user=ok, mod→admin=forbidden.
  - `TestPromoteToModerator`: root→user=ok, admin→user=forbidden.
- **Gate**: `go test ./internal/service/... -count=1 -race` → OK.

---

## Этап 7: Handler-ы (Feature)

**Цель:** HTTP-обработчики, связывающие роуты с Service-слоем.

### Шаг 7.1 — Auth handler

- `GET /auth/login` — рендер формы входа.
- `POST /auth/login` — валидация → service.Login → установка cookie `session_id`.
- `GET /auth/register` — рендер формы регистрации.
- `POST /auth/register` — валидация → service.Register → установка cookie `session_id`.
- `POST /auth/logout` — удаление сессии → удаление cookie.
- `GET /auth/google` — редирект на Google OAuth URL.
- `GET /auth/google/callback` — обмен code → service.LoginOrRegisterGoogle → установка cookie.
- `GET /auth/forgot` / `POST /auth/forgot` / `GET /auth/reset` / `POST /auth/reset` — сброс пароля.
- `GET /auth/forgot` + `POST /auth/forgot` — отправка токена на email.
- `GET /auth/reset?token=...` + `POST /auth/reset` — проверка токена + смена пароля.

### Шаг 7.2 — Post handler

- `GET /` — лента (смешанная, пагинация).
- `GET /articles` — список статей.
- `GET /threads` — список тредов.
- `GET /articles/new` + `POST /articles` — создание статьи.
- `GET /threads/new` + `POST /threads` — создание треда.
- `GET /articles/{id}-{slug}` — страница статьи (с комментариями).
- `GET /threads/{id}-{slug}` — страница треда.
- `GET /articles/{id}/edit` + `POST /articles/{id}` — редактирование статьи.
- HTMX-пагинация: `hx-get="?page=N"` → возвращает только список.

### Шаг 7.3 — Comment handler

- `POST /posts/{id}/comments` — добавление комментария (HTMX: возвращает HTML комментария).
- `GET /comments/reply-form?parent_id=N` — HTMX-форма ответа.
- `GET /comments/{id}/children` — подгрузка дочерних комментариев (HTMX).

### Шаг 7.4 — Message handler

- `GET /messages` — список диалогов.
- `GET /messages/{username}` — диалог с пользователем.
- `POST /messages/{username}` — отправка сообщения.
- `GET /messages/unread-count` — HTMX-счётчик.

### Шаг 7.5 — User handler

- `GET /users/{username}` — страница автора.
- HTMX-вкладки: `GET /users/{username}/tab?t=articles|threads|comments`.

### Шаг 7.6 — Settings handler

- `GET /settings` + `POST /settings` — профиль (bio, theme, font_size).

### Шаг 7.7 — Moderation handler

- `POST /mod/posts/{id}/delete` — удаление публикации.
- `POST /mod/comments/{id}/delete` — удаление комментария.
- `POST /mod/users/{id}/ban` — бан пользователя.
- `POST /mod/users/{id}/unban` — разбан.

### Шаг 7.8 — Root handler

- `GET /root` — панель управления.
- `POST /root/backup` — создание бекапа.
- `POST /root/restore` — восстановление.
- `GET /root/backup/download/{filename}` — скачивание бекапа.
- `POST /root/users/{id}/promote` — назначение модератора.
- `POST /root/users/{id}/demote` — снятие модератора.

### Шаг 7.9 — Тестирование handler-ов

- Каждый handler тестируется через `httptest.NewServer` с in-memory SQLite + полный DI (service + repo + render).
- `internal/handler/auth_handler_test.go`:
  - `TestLoginPage`: GET → 200, форма.
  - `TestLoginSuccess`: POST → 302, установлена cookie `session_id`.
  - `TestLoginWrongPassword`: POST → 200, сообщение об ошибке.
  - `TestLoginRateLimit`: 6 POST-запросов → 429 на 6-м.
  - `TestRegisterSuccess`: POST → 302 + cookie.
- `internal/handler/posts_handler_test.go`:
  - `TestFeedPage`: GET `/` → 200, список публикаций.
  - `TestArticlePage`: GET `/articles/1-moya-statya` → 200.
  - `TestCreateArticleNoAuth`: POST `/articles` без сессии → 302.
  - `TestCreateArticleSuccess`: POST `/articles` с сессией → 302.
- `internal/handler/comments_handler_test.go`:
  - `TestPostComment`: POST с HTMX-заголовком → 200, HTML фрагмент.
  - `TestReplyForm`: HTMX GET → 200, форма.
- `internal/handler/mod_handler_test.go`:
  - `TestBanUserModerator`: moderator банит user → 302.
  - `TestBanUserForbidden`: moderator пытается забанить admin → 403.
- **Gate**: `go test ./internal/handler/... -count=1 -race` → OK.

---

## Этап 8: Middleware (Infra)

**Цель:** защита и консистентность запросов.

### Шаг 8.1 — SecurityHeaders

- `Content-Security-Policy` из п.13.2.
- `X-Frame-Options: DENY`.
- `X-Content-Type-Options: nosniff`.
- `Referrer-Policy: same-origin`.
- Использует `nonce` для script-src (генерировать на каждый запрос, класть в контекст).

### Шаг 8.2 — RateLimit

- In-memory: `sync.Map` с ключом `IP:action`, значением — `[counters, resetTime]`.
- `func RateLimit(action string, maxRequests int, window time.Duration) func(http.Handler) http.Handler`.
- Применение:
  - `/auth/login` → 5 попыток / мин.
  - `/auth/register` → 3 регистрации / час.

### Шаг 8.3 — CSRF

- Double-submit cookie:
  - На GET-запросах без сессии: установить cookie `csrf_token` со случайным значением.
  - На POST/PUT/DELETE: проверить, что `X-CSRF-Token` header совпадает с cookie.
  - Исключения: `/auth/google/callback` (внешний редирект).
  - Также: HTMX-запросы браузер автоматически отправляет cookie, клиент читает из `meta`-тега в `<head>` и шлёт в заголовке.

### Шаг 8.4 — RequireRole

- `func RequireRole(minRole model.Role) func(http.Handler) http.Handler`.
- Читает `user` из контекста (установлен `OptionalAuth`).
- Если `user.role.Level() < minRole.Level()` → 403.
- Если пользователь забанен и действие требует создания контента — отдельная проверка.

### Шаг 8.5 — Тестирование middleware

- `internal/middleware/security_test.go`:
  - `TestSecurityHeaders`: проверка CSP, X-Frame-Options, X-Content-Type-Options.
  - `TestCSPNonce`: nonce присутствует и уникален на каждый запрос.
- `internal/middleware/ratelimit_test.go`:
  - `TestRateLimitUnder`: 4 запроса при лимите 5 → все 200.
  - `TestRateLimitOver`: 6 запросов → последний 429.
  - `TestRateLimitReset`: после окончания окна — снова 200.
  - `TestRateLimitIsolated`: разные IP не влияют друг на друга.
- `internal/middleware/csrf_test.go`:
  - `TestCSRFNoToken`: POST без токена → 403.
  - `TestCSRFValidToken`: POST с совпадающим токеном → 200.
  - `TestCSRFMismatch`: POST с несовпадающим токеном → 403.
- `internal/middleware/role_test.go`:
  - `TestRequireRolePass`: user + RoleUser → 200.
  - `TestRequireRoleFail`: user + RoleModerator → 403.
  - `TestRequireRoleBanned`: забаненный user пытается создать пост → 403.
- **Gate**: `go test ./internal/middleware/... -count=1 -race` → OK.

---

## Этап 9: Markdown + Sanitizer (Feature)

**Цель:** безопасный рендеринг Markdown.

### Шаг 9.1 — Пакет markdown

- `internal/markdown/render.go`:
  - `RenderToHTML(input string) (string, error)`.
  - Использует `goldmark` для конвертации Markdown → HTML.
  - Пропускает результат через `bluemonday.UGCPolicy().Sanitize()`.
  - Кешировать конвертер/парсер (один раз при старте).

### Шаг 9.2 — Тестирование Markdown

- `internal/markdown/render_test.go`:
  - `TestBold`: `**bold**` → `<strong>bold</strong>`.
  - `TestLink`: `[text](url)` → `<a href="url" rel="nofollow">text</a>`.
  - `TestCodeBlock`: ` ```go ... ``` ` → `<pre><code>`.
  - `TestSanitizeScript`: `<script>alert(1)</script>` → удалён.
  - `TestSanitizeXSS`: `[click](javascript:alert(1))` → sanitized.
  - `TestEmptyInput`: `""` → `""`.
- Golden files в `internal/markdown/testdata/`:
  - `bold_input.md` / `bold_output.golden`.
  - `xss_input.md` / `xss_output.golden`.
- **Gate**: `go test ./internal/markdown/... -count=1 -race` → OK.

---

## Этап 10: Тема и размер шрифта (Feature)

**Цель:** светлая/тёмная тема, переключение размера шрифта.

### Шаг 10.1 — Cookie + SSR

- Middleware или отдельный handler читает cookie `theme` и `font_size`.
- Кладёт в контекст или в `PageData`.
- В шаблоне `layout.html`: `<html data-theme="{{ .Theme }}" style="--font-size: {{ .FontSize }}px">`.

### Шаг 10.2 — JavaScript

- `web/static/theme.js` (встроенный через `<script>` или отдельный файл):
  - При загрузке: читает localStorage → устанавливает тему и размер.
  - Переключатель темы: пишет в localStorage + cookie.
  - Переключатель размера: пишет в localStorage + cookie + `document.documentElement.style.setProperty`.

---

## Этап 11: Конфигурация и bootstrap root (Infra)

**Цель:** чтение конфига, создание root-пользователя.

### Шаг 11.1 — Конфиг

- `internal/config/config.go`:
  - Чтение TOML-файла.
  - Override из переменных окружения (`TEBLORUM_*`).
  - Возвращает структуру `Config`.

### Шаг 11.2 — Bootstrap root

- В `cmd/teblorum/main.go` после инициализации БД:
  1. Проверить, существует ли пользователь с role='root'.
  2. Если нет: найти пользователя по `config.Bootstrap.RootEmail`.
  3. Если найден: обновить role='root'.
  4. Если не найден: создать пользователя с role='root' (без пароля — потребуется сброс, или с временным паролем, который выводится в лог).

### Шаг 11.3 — Тестирование конфига и bootstrap

- `internal/config/config_test.go`:
  - `TestLoadConfig`: загрузка из временного TOML-файла, проверка всех полей.
  - `TestConfigEnvOverride`: `TEBLORUM_DOMAIN=test.com` переопределяет `domain`.
  - `TestConfigDefaults`: отсутствующие поля имеют значения по умолчанию.
- `internal/service/user_service_test.go` (дополнить):
  - `TestBootstrapRoot`: пустая БД → создание root. БД с root → пропуск.
- **Gate**: `go test ./internal/config/... ./internal/service/... -count=1 -race` → OK.

---

## Этап 12: TLS / autocert (Infra)

**Цель:** HTTPS через Let's Encrypt.

### Шаг 12.1 — autocert integration

- В `cmd/teblorum/main.go`:
  ```go
  m := &autocert.Manager{
      Cache:      autocert.DirCache(config.Server.CertDir),
      Prompt:     autocert.AcceptTOS,
      HostPolicy: autocert.HostWhitelist(config.Server.Domain),
  }
  ```
- HTTP-сервер на `:80` для ACME HTTP-01 challenge.
- HTTPS-сервер на `:443` с TLS-конфигурацией от `m.TLSConfig()`.
- Функция `createHTTPServer(handler, addr, tlsConfig) *http.Server`.

### Шаг 12.2 — Graceful shutdown

- `signal.NotifyContext` для SIGINT/SIGTERM.
- Shutdown HTTP-серверов с таймаутом 10 секунд.
- Закрытие пула соединений БД.

---

## Этап 13: Systemd unit и деплой (Infra)

**Цель:** корректный запуск как systemd-сервиса.

### Шаг 13.1 — Unit-файл

- `deploy/teblorum.service` по спецификации п.16.
- Скрипт установки: создание пользователя `teblorum`, директорий `/etc/teblorum`, `/var/lib/teblorum/{db,backups,certs}`, копирование бинарника и конфига.

### Шаг 13.2 — Makefile

```makefile
.PHONY: build install clean

build:
	go build -ldflags="-s -w" -o build/teblorum ./cmd/teblorum/

install:
	sudo cp build/teblorum /usr/local/bin/
	sudo cp deploy/teblorum.service /etc/systemd/system/
	sudo systemctl daemon-reload
```

---

## Этап 14: Фоновая очистка (Infra)

**Цель:** автоматическая очистка просроченных сессий, банов, старых бекапов.

- В `cmd/teblorum/main.go`:
  ```go
  go func() {
      ticker := time.NewTicker(1 * time.Hour)
      defer ticker.Stop()
      for range ticker.C {
          repo.CleanExpiredSessions(db)
          // проверка banned_until < now() для всех users — делается при каждом запросе на лету
          // CleanupOldBackups(config.Database.BackupDir, 10)
      }
  }()
  ```

---

## Этап 15: Шаблоны (HTML) (Feature)

**Цель:** все HTML-шаблоны согласно маршрутам.

### 15.1 — Layout

- `layout.html`: DOCTYPE, `<html data-theme>`, `<head>` (CSP nonce, CSS, JS), `<body>`, навигация, `<main>{{ block "content" . }}</main>`.
- Общие partials: `nav.html`, `pagination.html`, `comment_tree.html`.

### 15.2 — Страницы

- `login.html`, `register.html`, `forgot.html`, `reset.html`.
- `feed.html`, `articles.html`, `threads.html`.
- `article.html`, `thread.html` (с деревом комментариев).
- `user.html` (с HTMX-вкладками).
- `settings.html`.
- `messages.html`, `conversation.html`.
- `mod_panel.html` (для moderator/admin — список действий).
- `root_panel.html` (панель root).

### 15.3 — HTMX-partials

- `comment.html` (один комментарий с ответами).
- `comment_form.html` (форма ответа).
- `comments_list.html` (блок комментариев).
- `user_tab_articles.html`, `user_tab_threads.html`, `user_tab_comments.html`.
- `unread_count.html` (просто число).
- `post_list.html` (список публикаций, для HTMX-пагинации).

---

## Порядок имплементации (рекомендуемый)

Таблица порядка вынесена в отдельный файл: [`ORDER_OF_IMPLEMENTATION.md`](ORDER_OF_IMPLEMENTATION.md).

Включает:
- 17 этапов с зависимостями и привязкой тестовых пакетов.
- Mermaid-граф зависимостей.
- Правило перехода `go vet ./... && go test ./... -count=1 -race`.

---

## Архитектурные решения (ADR)

### ADR-1: Стандартный ServeMux vs gorilla/mux

**Решение:** стандартный `net/http.ServeMux` (Go 1.22+).  
**Обоснование:** Go 1.22 добавил поддержку `{id}`-паттернов, метод-матчинг (`GET /path`, `POST /path`). Нулевая зависимость. Полностью покрывает потребности спецификации.

### ADR-2: Service-слой — один или много файлов

**Решение:** один файл на домен (user, post, comment, message, moderation, root), собранные в пакет `internal/service`.  
**Обоснование:** Каждый файл имеет единственную ответственность (SRP). Разделение очевидно по спецификации.

### ADR-3: Repository pattern — транзакции

**Решение:** каждый метод репозитория принимает `*sql.DB` (или `*sql.Tx`), но внутри всегда использует переданное соединение. Транзакции — на уровне service.  
**Обоснование:** `SetMaxOpenConns(1)` делает транзакции безопасными без блокировок. Service вызывает `db.Begin()` → передаёт `*sql.Tx` в методы repo.

### ADR-4: Slug — генерация на уровне service

**Решение:** функция `GenerateSlug(title)` в PostService. Транслитерация кириллицы в латиницу (простая таблица замен), lowercase, замена пробелов на дефисы, удаление не-alphanumeric символов.  
**Обоснование:** Не нужна отдельная библиотека. Ссылка вида `/articles/42-eto-moya-statya`.

### ADR-5: Бан — проверка на уровне middleware + service

**Решение:** middleware проверяет `user.banned_until` при каждом запросе. Если бан активен и запрос на создание контента — возвращает 403. Service-слой дублирует проверку для надёжности.  
**Обоснование:** Defense in depth. Middleware отсекает рано, service не пропустит при прямом вызове.

### ADR-6: CSP nonce — генерация на запрос

**Решение:** каждый HTTP-запрос генерирует случайный nonce (crypto/rand, 16 байт → hex), кладётся в контекст рендера.  
**Обоснование:** inline-скрипты (тема, размер шрифта) требуют nonce. Per-request nonce — стандартная практика безопасности.

### ADR-7: Пагинация — offset-based

**Решение:** `LIMIT 30 OFFSET (page-1)*30`.  
**Обоснование:** Проще cursor-based, достаточно для объёмов < 10K записей. SQLite с offset на малых страницах не деградирует.

### ADR-8: graceful shutdown для восстановления БД

**Решение:** при запросе восстановления БД (root) — остановить HTTP-сервер, закрыть пул, выполнить замену файла, переоткрыть пул, перезапустить сервер.  
**Обоснование:** Целостность БД критична. Никаких операций во время замены файла.

---

## Риски и mitigation

| Риск | Mitigation |
|------|-----------|
| SQLite-конкуренция при `SetMaxOpenConns(1)` | HTMX-запросы потенциально могут накладываться; `busy_timeout=5000` + очередь в Go-горутинах |
| OAuth-токены Google не имеют срока в нашей БД | Храним только `google_id`; OAuth-токен не нужен после получения user info |
| Размер Markdown-контента | Лимиты на уровне валидации в service + в БД (`TEXT` безлимитен, но валидация отсекает > 50К символов) |
| Потеря БД при restore | Backup текущей БД перед восстановлением; атомарная замена файла |