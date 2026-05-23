package repo

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/teblorum/teblorum/internal/model"
)

// UserRepo — репозиторий пользователей.
type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(req model.CreateUserRequest) (*model.User, error) {
	now := time.Now()
	result, err := r.db.Exec(`
		INSERT INTO users (username, email, password, google_id, bio, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, req.Username, req.Email, req.Password, nullString(req.GoogleID), req.Bio, string(req.Role), now, now)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *UserRepo) GetByID(id int64) (*model.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, email, COALESCE(password,''), COALESCE(google_id,''), bio, role,
		       banned_until, banned_by, created_at, updated_at
		FROM users WHERE id = ?
	`, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(email string) (*model.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, email, COALESCE(password,''), COALESCE(google_id,''), bio, role,
		       banned_until, banned_by, created_at, updated_at
		FROM users WHERE email = ?
	`, email)
	return scanUser(row)
}

func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, email, COALESCE(password,''), COALESCE(google_id,''), bio, role,
		       banned_until, banned_by, created_at, updated_at
		FROM users WHERE username = ?
	`, username)
	return scanUser(row)
}

func (r *UserRepo) GetByGoogleID(googleID string) (*model.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, email, COALESCE(password,''), COALESCE(google_id,''), bio, role,
		       banned_until, banned_by, created_at, updated_at
		FROM users WHERE google_id = ?
	`, googleID)
	return scanUser(row)
}

func (r *UserRepo) Update(id int64, req model.UpdateUserRequest) error {
	_, err := r.db.Exec(`
		UPDATE users SET username = ?, bio = ?, updated_at = datetime('now') WHERE id = ?
	`, req.Username, req.Bio, id)
	return err
}

func (r *UserRepo) UpdateRole(id int64, role model.Role) error {
	_, err := r.db.Exec(`
		UPDATE users SET role = ?, updated_at = datetime('now') WHERE id = ?
	`, string(role), id)
	return err
}

func (r *UserRepo) SetBan(id int64, until time.Time, bannedBy int64) error {
	_, err := r.db.Exec(`
		UPDATE users SET banned_until = ?, banned_by = ?, updated_at = datetime('now') WHERE id = ?
	`, until, bannedBy, id)
	return err
}

func (r *UserRepo) RemoveBan(id int64) error {
	_, err := r.db.Exec(`
		UPDATE users SET banned_until = NULL, banned_by = NULL, updated_at = datetime('now') WHERE id = ?
	`, id)
	return err
}

func (r *UserRepo) ExistsByEmail(email string) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	return count > 0, err
}

func (r *UserRepo) ExistsByUsername(username string) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	return count > 0, err
}

func scanUser(scanner interface {
	Scan(dest ...any) error
}) (*model.User, error) {
	u := &model.User{}
	var password, googleID, bio string
	var bannedBy sql.NullInt64

	err := scanner.Scan(&u.ID, &u.Username, &u.Email, &password, &googleID, &bio, (*string)(&u.Role),
		&u.BannedUntil, &bannedBy, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user: %w", model.ErrNotFound)
		}
		return nil, err
	}

	u.Password = password
	u.GoogleID = googleID
	u.Bio = bio
	if bannedBy.Valid {
		u.BannedBy = bannedBy
	}

	return u, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
