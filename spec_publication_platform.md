# Спецификация веб-приложения: teblorum

**Версия:** 1.1  
**Дата:** май 2026

---

## 1. Обзор системы

**teblorum** — многопользовательская платформа для текстовых публикаций двух типов: статьи в формате блога и обсуждения в формате форума. Публикации доступны без регистрации; создание контента требует авторизации. Минималистичный интерфейс, серверный рендеринг с реактивностью через HTMX.

Система использует четырёхуровневую иерархию ролей: **user → moderator → admin → root**.

---

## 2. Технический стек

| Компонент | Выбор |
|-----------|-------|
| Язык | Go 1.22+ |
| Рендеринг | SSR (html/template) + HTMX 2.x |
| База данных | SQLite 3 (через `modernc.org/sqlite` — pure Go, без CGO) |
| HTTP-сервер | стандартный `net/http` Go |
| Аутентификация | сессии в cookie (httponly, secure, sameSite=Lax) |
| OAuth | Google OAuth 2.0 (`golang.org/x/oauth2`) |
| TLS | Caddy (встраиваемый) **или** `autocert` (`golang.org/x/crypto/acme/autocert`) |
| Шаблоны | `html/template` (встроенный в бинарник через `embed.FS`) |
| CSS | единый минималистичный файл, CSS custom properties |
| Процесс | systemd unit, один бинарный файл |

> **Примечание по TLS:** `autocert` напрямую в Go-процессе — наименьший overhead. Сертификаты Let's Encrypt обновляются автоматически; порты 80 (challenge) и 443 (HTTPS) слушает сам процесс. Caddy как встраиваемая библиотека — альтернатива, если нужна более гибкая конфигурация.

---

## 3. Ограничения среды исполнения

| Параметр | Значение |
|----------|---------|
| ОС | Ubuntu 24.04 LTS |
| CPU | 1 ядро |
| RAM | 500 МБ |
| Диск | 5 ГБ |
| Docker | запрещён |
| Nginx / reverse-proxy | запрещён |
| Деплой | systemd + единый Go-бинарник |

### Оптимизации под ограничения

- SQLite WAL-режим (`PRAGMA journal_mode=WAL`) — параллельные чтения без блокировки.
- Один пул соединений с БД: `SetMaxOpenConns(1)` (SQLite однопоточная запись).
- Шаблоны компилируются при старте и кешируются в памяти.
- Статика (CSS, favicon) встраивается в бинарник через `embed.FS`.
- Пагинация везде — не более 30 записей на страницу.
- Фоновые задачи (очистка сессий, снятие банов) — легкий тикер `time.Ticker`, не cron.
- Бекапы БД — SQLite Online Backup API (`VACUUM INTO`), без блокировки основной базы.

---

## 4. База данных

### 4.1 Схема

```sql
-- Пользователи
CREATE TABLE users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT NOT NULL UNIQUE,          -- отображаемое имя
    email       TEXT NOT NULL UNIQUE,          -- только логин, не показывается публично
    password    TEXT,                          -- bcrypt hash; NULL для OAuth-пользователей
    google_id   TEXT UNIQUE,                   -- NULL для email/password пользователей
    bio         TEXT NOT NULL DEFAULT '',      -- краткая биография (plain text или Markdown)
    role        TEXT NOT NULL DEFAULT 'user',  -- 'user' | 'moderator' | 'admin' | 'root'
    banned_until DATETIME,                     -- NULL = не забанен
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Публикации
CREATE TABLE posts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id       INTEGER NOT NULL REFERENCES users(id),
    type            TEXT NOT NULL,             -- 'article' | 'thread'
    title           TEXT NOT NULL,
    body            TEXT NOT NULL,             -- Markdown или plain text
    comments_enabled INTEGER NOT NULL DEFAULT 1, -- только для type='article'
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    deleted_at      DATETIME                   -- мягкое удаление
);

-- Комментарии (иерархические, самореферентная таблица)
CREATE TABLE comments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id     INTEGER NOT NULL REFERENCES posts(id),
    author_id   INTEGER NOT NULL REFERENCES users(id),
    parent_id   INTEGER REFERENCES comments(id), -- NULL = корневой комментарий
    body        TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    deleted_at  DATETIME                         -- мягкое удаление
);

-- Личные сообщения
CREATE TABLE messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    from_user_id    INTEGER NOT NULL REFERENCES users(id),
    to_user_id      INTEGER NOT NULL REFERENCES users(id),
    thread_id       INTEGER,                   -- NULL = первое сообщение в диалоге; иначе id первого сообщения
    body            TEXT NOT NULL,
    read_at         DATETIME,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Сессии
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,              -- случайный 32-байтный hex
    user_id     INTEGER NOT NULL REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    expires_at  DATETIME NOT NULL
);

-- Индексы
CREATE INDEX idx_posts_author    ON posts(author_id);
CREATE INDEX idx_posts_type      ON posts(type);
CREATE INDEX idx_comments_post   ON comments(post_id);
CREATE INDEX idx_comments_parent ON comments(parent_id);
CREATE INDEX idx_messages_to     ON messages(to_user_id);
CREATE INDEX idx_messages_thread ON messages(thread_id);
CREATE INDEX idx_sessions_user   ON sessions(user_id);
CREATE INDEX idx_sessions_exp    ON sessions(expires_at);
```

### 4.2 Pragma при старте

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -8000;   -- ~8 МБ page cache
```

---

## 5. Аутентификация и авторизация

### 5.1 Email / пароль

- Регистрация: `username` (уникальный) + `email` (уникальный, не публичный) + `password`.
- Пароль хранится как bcrypt hash (cost=12).
- Логин: email + пароль → создание сессии (cookie `session_id`, httponly, secure).
- Сессия: TTL 30 дней, обновляется при каждом запросе (скользящее окно).
- Восстановление пароля: токен на email (таблица `password_resets`; TTL 1 час).

### 5.2 Google OAuth 2.0

- Scope: `openid email profile`.
- Колбэк: `/auth/google/callback`.
- При первом входе: создаётся пользователь с `google_id`, `email`, `username` из Google-профиля (с уникализацией, если занят).
- При повторном входе: ищем по `google_id` или `email`; если найден по email — привязываем `google_id`.

### 5.3 Иерархия ролей

```
user  <  moderator  <  admin  <  root
```

| Роль | Описание |
|------|----------|
| **user** | Зарегистрированный пользователь. Создаёт контент, комментирует, отправляет ЛС. |
| **moderator** | Расширенные права модерации: удаляет контент `user`-аккаунтов, банит `user`-аккаунты. |
| **admin** | Надмодератор: удаляет контент любых аккаунтов (кроме `root`), банит `user` и `moderator`. |
| **root** | Суперпользователь: всё что может `admin` + управление БД + назначение/снятие роли `moderator`. |

**Правила защиты по иерархии:**
- Каждая роль может воздействовать только на аккаунты с **меньшим** уровнем привилегий.
- `root` не может быть забанен никем.
- Назначить или снять роль `moderator` может **только** `root`.
- Роль `admin` и `root` не могут быть назначены через веб-интерфейс — только через прямое изменение БД (первоначальная настройка). Исключение: `root` назначает `moderator`.
- Первый `root`-аккаунт создаётся при первом запуске через CLI-флаг (`--bootstrap-root email@example.com`) или из конфига.

### 5.4 Матрица доступа

| Действие | Аноним | User | Moderator | Admin | Root |
|----------|:------:|:----:|:---------:|:-----:|:----:|
| Читать публикации и комментарии | ✅ | ✅ | ✅ | ✅ | ✅ |
| Читать страницы авторов | ✅ | ✅ | ✅ | ✅ | ✅ |
| Создавать статьи и треды | ❌ | ✅ | ✅ | ✅ | ✅ |
| Комментировать | ❌ | ✅ | ✅ | ✅ | ✅ |
| Отправлять личные сообщения | ❌ | ✅ | ✅ | ✅ | ✅ |
| Редактировать свой профиль | ❌ | ✅ | ✅ | ✅ | ✅ |
| Удалять контент `user` | ❌ | ❌ | ✅ | ✅ | ✅ |
| Удалять контент `moderator` | ❌ | ❌ | ❌ | ✅ | ✅ |
| Удалять контент `admin` | ❌ | ❌ | ❌ | ❌ | ✅ |
| Банить `user` | ❌ | ❌ | ✅ | ✅ | ✅ |
| Банить `moderator` | ❌ | ❌ | ❌ | ✅ | ✅ |
| Банить `admin` | ❌ | ❌ | ❌ | ❌ | ✅ |
| Банить `root` | ❌ | ❌ | ❌ | ❌ | ❌ |
| Назначать / снимать роль `moderator` | ❌ | ❌ | ❌ | ❌ | ✅ |
| Создавать / восстанавливать бекап БД | ❌ | ❌ | ❌ | ❌ | ✅ |

### 5.5 Бан

- Параметры: `user_id`, срок (1д / 7д / 30д / навсегда), инициатор (`banned_by`).
- Забаненный пользователь может читать контент и отправлять личные сообщения, но не может создавать публикации и комментарии.
- По истечении `banned_until` бан снимается автоматически (проверка при каждом запросе).
- `root` не может быть забанен ни при каких обстоятельствах — попытка игнорируется.
- Страница автора показывает статус «Заблокирован до [дата]» для пользователей с правом бана.
- Поле `banned_by` (id инициатора) сохраняется в БД для аудита — добавить в схему:

```sql
ALTER TABLE users ADD COLUMN banned_by INTEGER REFERENCES users(id);
```

---

## 6. Публикации

### 6.1 Статья (type = 'article')

- Поля: `title`, `body` (Markdown), `comments_enabled` (bool, автор выбирает при создании и может изменить позже).
- URL: `/articles/{id}-{slug}`
- Страница статьи: заголовок, автор (ссылка на страницу), дата, тело, раздел комментариев (если включены).
- Автор может редактировать свою статью.

### 6.2 Тред (type = 'thread')

- Поля: `title`, `body` (стартовое сообщение).
- Комментарии всегда включены; автор не может их отключить.
- URL: `/threads/{id}-{slug}`
- Отображается как первый пост + дерево ответов.

### 6.3 Общие правила публикаций

- Slug генерируется транслитерацией из заголовка (latin, lowercase, дефисы), уникальность по `post_id`.
- Мягкое удаление: `deleted_at` заполняется, запись остаётся в БД, на странице выводится «[удалено]».
- Пагинация списков: 30 публикаций на страницу, сортировка по `created_at DESC`.

---

## 7. Комментарии

### 7.1 Структура

- Самореферентная таблица: `parent_id` → `id` в той же таблице.
- Глубина вложенности: не ограничена технически, но визуально — максимум 6 уровней отступа.
- Корневые комментарии: `parent_id IS NULL`.

### 7.2 Отображение дерева

- Дерево строится на сервере (рекурсивный CTE или `WITH RECURSIVE`):

```sql
WITH RECURSIVE tree AS (
    SELECT *, 0 AS depth FROM comments
    WHERE post_id = ? AND parent_id IS NULL AND deleted_at IS NULL
    UNION ALL
    SELECT c.*, t.depth + 1 FROM comments c
    JOIN tree t ON c.parent_id = t.id
    WHERE c.deleted_at IS NULL
)
SELECT * FROM tree ORDER BY created_at;
```

- Каждый комментарий содержит визуальную плашку «В ответ на @username» со ссылкой на родительский комментарий.
- Ветки с более чем N дочерних (N = 5 по умолчанию) показываются свёрнутыми; кнопка «Показать ответы (K)» раскрывает через HTMX (`hx-get`, `hx-swap`).
- Сворачивание/разворачивание — без перезагрузки страницы (HTMX + `details`/`summary` HTML или кастомный toggle).

### 7.3 Форма ответа

- «Ответить» под каждым комментарием — HTMX вставляет форму прямо под комментарием (`hx-get="/comments/reply-form?parent_id={id}"`, `hx-swap="afterend"`).
- Отправка формы — HTMX POST, в ответ сервер возвращает HTML нового комментария.

---

## 8. Личные сообщения

- Структура: диалоги (flat). Диалог между двумя пользователями — список сообщений по хронологии.
- `thread_id` — id первого сообщения в диалоге между двумя пользователями. Для поиска диалога: `MIN(from_user_id, to_user_id)` + `MAX(from_user_id, to_user_id)`.
- Страница `/messages` — список диалогов (последнее сообщение + непрочитанные).
- Страница `/messages/{user_id}` — конкретный диалог.
- Отправка нового сообщения — кнопка «Написать» на странице автора.
- Непрочитанные сообщения: счётчик в навигации (обновляется через HTMX polling каждые 60 с или при фокусе вкладки).
- Email платформы **не показывается** нигде публично.

---

## 9. Страница автора

**URL:** `/users/{username}`

**Содержимое:**
- Отображаемое имя (`username`).
- Биография (произвольный текст, до 1000 символов).
- Дата регистрации.
- Метка роли пользователя (отображается публично): `moderator` / `admin` / `root` — если роль выше `user`.
- Вкладки (HTMX):
  - **Статьи** — список статей автора (пагинация).
  - **Треды** — список инициированных тредов (пагинация).
  - **Комментарии** — хронологический список комментариев автора с контекстом (ссылка на публикацию).
- Кнопка «Написать сообщение» (только для авторизованных).
- **Кнопки модерации** (видны только тем, кто имеет право воздействовать на данного пользователя):
  - `moderator` видит кнопки «Забанить» для `user`-аккаунтов.
  - `admin` видит кнопки «Забанить» для `user` и `moderator`-аккаунтов.
  - `root` видит кнопку «Забанить» для всех (кроме `root`) + кнопки «Назначить модератором» / «Снять модератора» для `user`/`moderator`-аккаунтов.
  - Кнопка «Разбанить» отображается вместо «Забанить», если пользователь уже забанен.

---

## 10. Интерфейс и дизайн

### 10.1 Принципы

- Минималистичный, только типографика и пространство.
- Никакой графики, иконок-изображений (только текстовые символы Unicode / CSS-символы).
- Максимум 2 шрифта: один для заголовков, один для текста (системный стек или Google Fonts через `<link preload>`).
- Цветовая палитра: 2–3 цвета на тему (фон, текст, акцент).

### 10.2 Темы (CSS custom properties)

```css
:root {
  --bg:        #ffffff;
  --bg-alt:    #f5f5f5;
  --fg:        #111111;
  --fg-muted:  #555555;
  --accent:    #1a1a1a;
  --border:    #dddddd;
  --font-size: 16px;       /* изменяется пользователем */
}

[data-theme="dark"] {
  --bg:        #111111;
  --bg-alt:    #1c1c1c;
  --fg:        #e8e8e8;
  --fg-muted:  #888888;
  --accent:    #e8e8e8;
  --border:    #333333;
}
```

Тема сохраняется в `localStorage` и в cookie (для SSR первого рендера без мигания).

### 10.3 Размер шрифта

- Три ступени: Small (14px) / Medium (16px) / Large (18px).
- Переключатель в шапке: `A- / A / A+`.
- Значение сохраняется в `localStorage` + cookie `font_size`.
- Применяется через `document.documentElement.style.setProperty('--font-size', ...)`.

### 10.4 Адаптивность

Брейкпоинты:

| Устройство | Ширина | Поведение |
|------------|--------|-----------|
| Телефон | < 640px | 1 колонка, компактные отступы, гамбургер-меню |
| Планшет | 640–1024px | 1 колонка, средние отступы, nav горизонтально |
| Десктоп | > 1024px | макс. ширина 760px по центру, полный nav |

CSS-only адаптивность через `@media`, без JS-переключателей.

### 10.5 Структура навигации

```
[Название платформы]   [Статьи] [Треды]          [Войти] / [Username ▾] [A-][A][A+][☀/☾]
```

Для авторизованного пользователя выпадающее меню Username: Новая статья / Новый тред / Сообщения (N) / Настройки / Выйти.

---

## 11. Маршруты (URL Map)

### Публичные

| Метод | URL | Описание |
|-------|-----|----------|
| GET | `/` | Лента публикаций (статьи + треды смешанно, `created_at DESC`) |
| GET | `/articles` | Только статьи |
| GET | `/threads` | Только треды |
| GET | `/articles/{id}-{slug}` | Страница статьи |
| GET | `/threads/{id}-{slug}` | Страница треда |
| GET | `/users/{username}` | Страница автора |
| GET | `/auth/login` | Форма входа |
| POST | `/auth/login` | Обработка входа |
| GET | `/auth/register` | Форма регистрации |
| POST | `/auth/register` | Обработка регистрации |
| GET | `/auth/google` | Редирект на Google OAuth |
| GET | `/auth/google/callback` | Колбэк Google OAuth |
| GET | `/auth/forgot` | Форма сброса пароля |
| POST | `/auth/forgot` | Отправка письма |
| GET | `/auth/reset?token=...` | Форма нового пароля |
| POST | `/auth/reset` | Сохранение нового пароля |

### Авторизованные (User + Admin)

| Метод | URL | Описание |
|-------|-----|----------|
| POST | `/auth/logout` | Выход |
| GET | `/articles/new` | Форма новой статьи |
| POST | `/articles` | Создание статьи |
| GET | `/articles/{id}/edit` | Редактирование (только автор/admin) |
| POST | `/articles/{id}` | Сохранение изменений |
| GET | `/threads/new` | Форма нового треда |
| POST | `/threads` | Создание треда |
| POST | `/posts/{id}/comments` | Добавить комментарий |
| GET | `/comments/reply-form` | Форма ответа (HTMX partial) |
| GET | `/comments/{id}/children` | Дочерние комментарии (HTMX partial) |
| GET | `/messages` | Список диалогов |
| GET | `/messages/{username}` | Диалог с пользователем |
| POST | `/messages/{username}` | Отправить сообщение |
| GET | `/settings` | Настройки профиля |
| POST | `/settings` | Сохранение настроек |

### Moderator + Admin + Root

| Метод | URL | Описание | Минимальная роль |
|-------|-----|----------|-----------------|
| POST | `/mod/posts/{id}/delete` | Удалить публикацию | moderator |
| POST | `/mod/comments/{id}/delete` | Удалить комментарий | moderator |
| POST | `/mod/users/{id}/ban` | Забанить пользователя | moderator |
| POST | `/mod/users/{id}/unban` | Разбанить | moderator |

### Root only

| Метод | URL | Описание |
|-------|-----|----------|
| GET | `/root` | Панель root-администратора |
| POST | `/root/backup` | Создать бекап БД |
| POST | `/root/restore` | Восстановить БД из бекапа |
| GET | `/root/backup/download/{filename}` | Скачать бекап |
| POST | `/root/users/{id}/promote` | Назначить роль `moderator` пользователю `user` |
| POST | `/root/users/{id}/demote` | Снять роль `moderator` (→ `user`) |

---

## 12. HTMX-паттерны

| Взаимодействие | HTMX-атрибуты |
|----------------|---------------|
| Вкладки на странице автора | `hx-get="/users/{u}/tab?t=articles"` `hx-target="#tab-content"` `hx-push-url="true"` |
| Форма ответа на комментарий | `hx-get="/comments/reply-form?parent_id={id}"` `hx-swap="afterend"` |
| Раскрыть дочерние комментарии | `hx-get="/comments/{id}/children"` `hx-swap="outerHTML"` |
| Отправка комментария | `hx-post="/posts/{id}/comments"` `hx-swap="beforeend"` `hx-target="#comments-list"` |
| Счётчик сообщений | `hx-get="/messages/unread-count"` `hx-trigger="every 60s"` `hx-target="#msg-count"` |
| Переключение темы | JS → cookie + `document.documentElement.dataset.theme` |
| Пагинация | `hx-get="?page=2"` `hx-push-url="true"` `hx-target="#posts-list"` `hx-swap="outerHTML"` |

Все HTMX-запросы сервер определяет по заголовку `HX-Request: true` и возвращает только HTML-фрагмент, без полного layout.

---

## 13. Безопасность

### 13.1 Стандартные меры

| Угроза | Защита |
|--------|--------|
| XSS | `html/template` автоматически экранирует всё; Markdown рендерится с sanitizer (`bluemonday`) |
| CSRF | Double-submit cookie (`csrf_token`); проверка на всех POST/PUT/DELETE |
| SQL-инъекции | Только параметризованные запросы, никаких строковых конкатенаций |
| Перебор паролей | Rate limit на `/auth/login`: 5 попыток / 1 минута / IP (in-memory counter + `sync.Map`) |
| Session hijacking | `httponly`, `secure`, `SameSite=Lax`; регенерация session_id при логине |
| Brute-force регистрации | Rate limit на `/auth/register`: 3 регистрации / час / IP |
| Открытые редиректы | Whitelist разрешённых URL для `?next=` параметра |
| HTTPS | TLS 1.2+; HTTP → HTTPS редирект |
| Заголовки безопасности | `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin` |
| Паролb | bcrypt cost=12; минимум 8 символов |

### 13.2 Content Security Policy

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self' 'nonce-{per_request}';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data:;
  connect-src 'self';
  frame-ancestors 'none';
```

HTMX подключается из `embed.FS`, не CDN.

### 13.3 Ограничения на ввод

- Заголовок публикации: макс. 200 символов.
- Тело публикации: макс. 50 000 символов.
- Комментарий: макс. 5 000 символов.
- Биография: макс. 1 000 символов.
- Личное сообщение: макс. 5 000 символов.
- Имя пользователя: 3–30 символов, `[a-zA-Z0-9_-]`.

---

## 14. TLS и сертификаты

Используется `autocert` из `golang.org/x/crypto/acme/autocert`:

```go
m := &autocert.Manager{
    Cache:      autocert.DirCache("/var/lib/teblorum/certs"),
    Prompt:     autocert.AcceptTOS,
    HostPolicy: autocert.HostWhitelist("example.com", "www.example.com"),
}

// HTTP-сервер для ACME challenge (порт 80)
go http.ListenAndServe(":80", m.HTTPHandler(nil))

// HTTPS-сервер (порт 443)
srv := &http.Server{
    Addr:      ":443",
    TLSConfig: m.TLSConfig(),
    Handler:   router,
}
srv.ListenAndServeTLS("", "")
```

- Сертификаты кешируются на диске в `/var/lib/teblorum/certs/`.
- Автоматическое обновление за 30 дней до истечения (библиотека делает сама).
- Никаких cron-задач для certbot не нужно.

---

## 15. Конфигурация приложения

Файл `/etc/teblorum/config.toml`:

```toml
[server]
domain = "example.com"
port_http  = 80
port_https = 443
cert_dir   = "/var/lib/teblorum/certs"

[database]
path    = "/var/lib/teblorum/db/main.db"
backup_dir = "/var/lib/teblorum/backups"

[auth]
session_ttl_days = 30
google_client_id     = "..."
google_client_secret = "..."

[mail]
smtp_host = "..."
smtp_port = 587
smtp_user = "..."
smtp_pass = "..."
from_addr = "no-reply@example.com"

[app]
name = "teblorum"
max_post_body    = 50000
max_comment_body = 5000

[bootstrap]
# Email пользователя, которому при первом запуске будет выдана роль root.
# После выдачи роли значение игнорируется.
root_email = "admin@example.com"
```

Секреты можно также передавать через переменные окружения (`TEBLORUM_GOOGLE_CLIENT_SECRET` и т.д.).

---

## 16. Systemd Unit

`/etc/systemd/system/teblorum.service`:

```ini
[Unit]
Description=teblorum — платформа текстовых публикаций
After=network.target

[Service]
Type=simple
User=teblorum
Group=teblorum
WorkingDirectory=/var/lib/teblorum
ExecStart=/usr/local/bin/teblorum --config /etc/teblorum/config.toml
Restart=on-failure
RestartSec=5s

# Ограничения ресурсов
MemoryMax=450M
CPUQuota=95%

# Безопасность
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/teblorum

# Capabilities для портов < 1024
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
```

---

## 17. Бекап и восстановление

### Создание бекапа

```go
// SQLite Online Backup через VACUUM INTO — без блокировки основной БД
filename := fmt.Sprintf("backup_%s.db", time.Now().Format("20060102_150405"))
_, err := db.Exec("VACUUM INTO ?", filepath.Join(backupDir, filename))
```

- Хранятся последние 10 бекапов (старые удаляются автоматически).
- Бекапы доступны для скачивания только пользователю с ролью `root`.

### Восстановление

1. Принять загруженный `.db` файл через форму (multipart/form-data).
2. Проверить валидность: `PRAGMA integrity_check`.
3. Остановить пул соединений.
4. Переименовать текущую БД в `main.db.bak`.
5. Скопировать загруженный файл в `main.db`.
6. Переоткрыть пул соединений.

---

## 18. Структура проекта

```
teblorum/
├── cmd/
│   └── teblorum/
│       └── main.go            # точка входа, config, DI, bootstrap root
├── internal/
│   ├── auth/                  # сессии, OAuth, bcrypt
│   ├── handler/               # http.HandlerFunc-ы по доменам
│   │   ├── posts.go
│   │   ├── comments.go
│   │   ├── messages.go
│   │   ├── users.go
│   │   ├── mod.go             # moderator + admin actions (ban, delete)
│   │   └── root.go            # root actions (backup, restore, promote/demote)
│   ├── middleware/            # auth check, role check, CSRF, rate limit, security headers
│   │   ├── auth.go
│   │   ├── role.go            # RequireRole(minRole), CanActOn(target)
│   │   └── ratelimit.go
│   ├── model/                 # структуры данных + Role type с методами (CanBan, CanDelete, ...)
│   │   ├── user.go            # type Role string; func (r Role) Level() int
│   │   ├── post.go
│   │   ├── comment.go
│   │   └── message.go
│   ├── repo/                  # SQL-запросы (repository pattern)
│   │   ├── users.go
│   │   ├── posts.go
│   │   ├── comments.go
│   │   └── messages.go
│   ├── service/               # бизнес-логика (в т.ч. проверка иерархии ролей)
│   ├── render/                # SSR: template.ExecuteTemplate + HTMX-partial detection
│   └── markdown/              # Markdown → HTML с sanitizer
├── web/
│   ├── templates/
│   │   ├── layout.html        # base layout
│   │   ├── partials/          # htmx-фрагменты
│   │   └── pages/             # полные страницы
│   └── static/
│       ├── htmx.min.js
│       └── style.css
├── migrations/
│   └── 001_init.sql
├── go.mod
└── go.sum
```

---

## 19. Ключевые зависимости Go

```go
// go.mod (основные)
require (
    golang.org/x/crypto              v0.23.0  // bcrypt, autocert
    golang.org/x/oauth2              v0.20.0  // Google OAuth
    modernc.org/sqlite               v1.29.9  // SQLite pure Go (без CGO)
    github.com/microcosm-cc/bluemonday v1.0.27 // HTML sanitizer для Markdown
    github.com/yuin/goldmark         v1.7.1   // Markdown → HTML
    github.com/gorilla/mux           v1.8.1   // роутер (или chi)
)
```

> Альтернатива роутеру: `net/http` со стандартным `ServeMux` (Go 1.22+ поддерживает `{id}` паттерны) — нулевые зависимости.

---

## 20. Маркетинговое резюме / контрольный список реализации

- [ ] Регистрация и вход (email/пароль + Google OAuth)
- [ ] Сессии, CSRF, rate limiting
- [ ] Создание/редактирование статей и тредов
- [ ] Иерархические комментарии с HTMX-разворачиванием
- [ ] Управление комментариями (включить/выключить для статьи)
- [ ] Личные сообщения (диалоги)
- [ ] Страница автора с вкладками и меткой роли
- [ ] Четыре роли с иерархией: user / moderator / admin / root
- [ ] Middleware проверки роли: `RequireRole(min)` + `CanActOn(actor, target)`
- [ ] Bootstrap root-аккаунта при первом запуске (из конфига)
- [ ] Бан пользователя (с периодом) — с проверкой иерархии ролей
- [ ] Назначение / снятие роли `moderator` — только `root`
- [ ] Мягкое удаление публикаций и комментариев — с проверкой иерархии ролей
- [ ] Панель root: бекап / восстановление БД
- [ ] Светлая / тёмная темы (cookie + localStorage)
- [ ] Переключение размера шрифта
- [ ] Адаптивная вёрстка (телефон / планшет / десктоп)
- [ ] TLS через autocert (Let's Encrypt), автообновление
- [ ] Systemd unit с ограничениями ресурсов
- [ ] Заголовки безопасности, CSP
- [ ] Sanitizer для Markdown-контента
- [ ] Пагинация всех списков
```
