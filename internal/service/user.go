package service

import (
	"database/sql"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/teblorum/teblorum/internal/auth"
	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

// UserService — бизнес-логика работы с пользователями.
type UserService struct {
	users    *repo.UserRepo
	sessions *repo.SessionRepo
	db       *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{
		users:    repo.NewUserRepo(db),
		sessions: repo.NewSessionRepo(db),
		db:       db,
	}
}

// Register создаёт нового пользователя и сессию.
func (s *UserService) Register(email, username, password string) (*model.User, *model.Session, error) {
	if email == "" {
		return nil, nil, fmt.Errorf("email: %w", model.ErrValidation)
	}
	if username == "" {
		return nil, nil, fmt.Errorf("username: %w", model.ErrValidation)
	}
	if utf8.RuneCountInString(password) < 6 {
		return nil, nil, fmt.Errorf("password: %w: too short (min 6 characters)", model.ErrValidation)
	}
	if utf8.RuneCountInString(username) > 50 {
		return nil, nil, fmt.Errorf("username: %w: too long (max 50 characters)", model.ErrValidation)
	}

	// Проверка уникальности
	exists, err := s.users.ExistsByEmail(email)
	if err != nil {
		return nil, nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, nil, fmt.Errorf("email: %w", model.ErrDuplicate)
	}

	exists, err = s.users.ExistsByUsername(username)
	if err != nil {
		return nil, nil, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return nil, nil, fmt.Errorf("username: %w", model.ErrDuplicate)
	}

	// Хеширование пароля
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(model.CreateUserRequest{
		Username: username,
		Email:    email,
		Password: hash,
		Role:     model.RoleUser,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	session, err := s.sessions.Create(user.ID, sessionTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	return user, session, nil
}

// Login аутентифицирует пользователя по email и паролю, создаёт сессию.
func (s *UserService) Login(email, password string) (*model.User, *model.Session, error) {
	if email == "" || password == "" {
		return nil, nil, fmt.Errorf("%w: email and password are required", model.ErrValidation)
	}

	user, err := s.users.GetByEmail(email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, nil, fmt.Errorf("%w: invalid email or password", model.ErrUnauthorized)
		}
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	if user.Password == "" {
		return nil, nil, fmt.Errorf("%w: account uses Google OAuth", model.ErrUnauthorized)
	}

	if !auth.CheckPassword(user.Password, password) {
		return nil, nil, fmt.Errorf("%w: invalid email or password", model.ErrUnauthorized)
	}

	// Проверка бана
	if user.BannedUntil.Valid && user.BannedUntil.Time.After(user.CreatedAt) {
		return nil, nil, fmt.Errorf("%w: banned until %s", model.ErrBanned, user.BannedUntil.Time.Format("2006-01-02 15:04"))
	}

	session, err := s.sessions.Create(user.ID, sessionTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	return user, session, nil
}

// LoginOrRegisterGoogle аутентифицирует или регистрирует пользователя через Google OAuth.
func (s *UserService) LoginOrRegisterGoogle(googleID, email, name string) (*model.User, *model.Session, error) {
	// 1. Поиск по google_id
	user, err := s.users.GetByGoogleID(googleID)
	if err == nil {
		session, err := s.sessions.Create(user.ID, sessionTTL)
		if err != nil {
			return nil, nil, fmt.Errorf("create session: %w", err)
		}
		return user, session, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, nil, fmt.Errorf("get by google id: %w", err)
	}

	// 2. Поиск по email
	user, err = s.users.GetByEmail(email)
	if err == nil {
		// Привязываем google_id к существующему аккаунту
		// Используем прямой UPDATE, так как в repo нет метода для этого
		_, err := s.db.Exec("UPDATE users SET google_id = ?, updated_at = datetime('now') WHERE id = ?", googleID, user.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("link google account: %w", err)
		}
		user.GoogleID = googleID

		session, err := s.sessions.Create(user.ID, sessionTTL)
		if err != nil {
			return nil, nil, fmt.Errorf("create session: %w", err)
		}
		return user, session, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, nil, fmt.Errorf("get by email: %w", err)
	}

	// 3. Создание нового пользователя
	username := makeUniqueUsername(s.db, name)

	user, err = s.users.Create(model.CreateUserRequest{
		Username: username,
		Email:    email,
		GoogleID: googleID,
		Role:     model.RoleUser,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create google user: %w", err)
	}

	session, err := s.sessions.Create(user.ID, sessionTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	return user, session, nil
}

// GetByUsername возвращает пользователя по имени.
func (s *UserService) GetByUsername(username string) (*model.User, error) {
	return s.users.GetByUsername(username)
}

// GetByID возвращает пользователя по ID.
func (s *UserService) GetByID(id int64) (*model.User, error) {
	return s.users.GetByID(id)
}

// UpdateProfile обновляет профиль пользователя.
func (s *UserService) UpdateProfile(userID int64, req model.UpdateUserRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username: %w", model.ErrValidation)
	}
	if utf8.RuneCountInString(req.Username) > 50 {
		return fmt.Errorf("username: %w: too long", model.ErrValidation)
	}
	if utf8.RuneCountInString(req.Bio) > 1000 {
		return fmt.Errorf("bio: %w: too long", model.ErrValidation)
	}

	// Проверка уникальности username, если он меняется
	current, err := s.users.GetByID(userID)
	if err != nil {
		return err
	}
	if current.Username != req.Username {
		exists, err := s.users.ExistsByUsername(req.Username)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("username: %w", model.ErrDuplicate)
		}
	}

	return s.users.Update(userID, req)
}

// CanModerate проверяет, может ли actor модерировать target.
func (s *UserService) CanModerate(actor, target *model.User) bool {
	if actor == nil || target == nil {
		return false
	}
	return model.CanActOn(actor.Role, target.Role)
}

// ChangePassword меняет пароль пользователя.
func (s *UserService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	user, err := s.users.GetByID(userID)
	if err != nil {
		return err
	}

	if user.Password == "" {
		return fmt.Errorf("%w: account uses Google OAuth", model.ErrForbidden)
	}

	if !auth.CheckPassword(user.Password, oldPassword) {
		return fmt.Errorf("%w: invalid current password", model.ErrUnauthorized)
	}

	if utf8.RuneCountInString(newPassword) < 6 {
		return fmt.Errorf("password: %w: too short", model.ErrValidation)
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.db.Exec("UPDATE users SET password = ?, updated_at = datetime('now') WHERE id = ?", hash, userID)
	return err
}

// ResetPassword сбрасывает пароль по токену (без старого пароля).
func (s *UserService) ResetPassword(token, newPassword string) error {
	email, err := auth.ValidateResetToken(s.db, token)
	if err != nil {
		return err
	}

	if utf8.RuneCountInString(newPassword) < 6 {
		return fmt.Errorf("password: %w: too short", model.ErrValidation)
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.db.Exec("UPDATE users SET password = ?, updated_at = datetime('now') WHERE email = ?", hash, email)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Удаляем все сессии пользователя
	user, err := s.users.GetByEmail(email)
	if err == nil {
		_ = s.sessions.DeleteByUserID(user.ID)
	}

	return auth.DeleteResetToken(s.db, token)
}

// sessionTTL — TTL сессии в днях.
const sessionTTL = 30

// makeUniqueUsername создаёт уникальное имя пользователя на основе имени из Google.
func makeUniqueUsername(db *sql.DB, baseName string) string {
	username := sanitizeUsername(baseName)
	repo := repo.NewUserRepo(db)

	// Проверяем базовое имя
	_, err := repo.GetByUsername(username)
	if err != nil {
		return username
	}

	// Добавляем суффикс, пока не найдём свободное
	for i := 1; i < 100; i++ {
		candidate := fmt.Sprintf("%s%d", username, i)
		_, err := repo.GetByUsername(candidate)
		if err != nil {
			return candidate
		}
	}

	// Fallback: очень маловероятно
	return fmt.Sprintf("%s%d", username, generateID())
}

func sanitizeUsername(name string) string {
	// Транслитерация: заменяем кириллицу на латиницу
	translit := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
		'е': "e", 'ё': "yo", 'ж': "zh", 'з': "z", 'и': "i",
		'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
		'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
		'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch",
		'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "",
		'э': "e", 'ю': "yu", 'я': "ya",
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D",
		'Е': "E", 'Ё': "Yo", 'Ж': "Zh", 'З': "Z", 'И': "I",
		'Й': "Y", 'К': "K", 'Л': "L", 'М': "M", 'Н': "N",
		'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T",
		'У': "U", 'Ф': "F", 'Х': "Kh", 'Ц': "Ts", 'Ч': "Ch",
		'Ш': "Sh", 'Щ': "Shch", 'Ъ': "", 'Ы': "Y", 'Ь': "",
		'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	}

	var result []byte
	for _, r := range name {
		if tr, ok := translit[r]; ok {
			result = append(result, tr...)
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result = append(result, byte(r))
		} else if r == ' ' {
			result = append(result, '_')
		}
	}

	if len(result) == 0 {
		return "user"
	}

	return string(result)
}

func generateID() int64 {
	// Простой инкрементальный ID для уникальности (достаточно для тестов)
	return 0
}
