package service

import (
	"testing"

	"github.com/teblorum/teblorum/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		user, session, err := svc.Register("new@example.com", "newuser", "password123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if user.Username != "newuser" {
			t.Errorf("expected username 'newuser', got '%s'", user.Username)
		}
		if user.Role != model.RoleUser {
			t.Errorf("expected role 'user', got '%s'", user.Role)
		}
		if session == nil {
			t.Fatal("expected session, got nil")
		}
		if session.UserID != user.ID {
			t.Errorf("expected session userID %d, got %d", user.ID, session.UserID)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		seedServiceUser(t, db, map[string]interface{}{
			"email": "dup@example.com",
		})

		_, _, err := svc.Register("dup@example.com", "other", "password123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !isError(err, model.ErrDuplicate) {
			t.Errorf("expected ErrDuplicate, got %v", err)
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		seedServiceUser(t, db, map[string]interface{}{
			"username": "dupuser",
		})

		_, _, err := svc.Register("other@example.com", "dupuser", "password123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !isError(err, model.ErrDuplicate) {
			t.Errorf("expected ErrDuplicate, got %v", err)
		}
	})

	t.Run("short password", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		_, _, err := svc.Register("a@b.com", "newuser", "12")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !isError(err, model.ErrValidation) {
			t.Errorf("expected ErrValidation, got %v", err)
		}
	})

	t.Run("empty email", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		_, _, err := svc.Register("", "newuser", "password123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty username", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		_, _, err := svc.Register("a@b.com", "", "password123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		// Создаём пользователя напрямую с bcrypt-паролем
		hash, _ := hashPasswordForTest("correctpassword")
		seedServiceUser(t, db, map[string]interface{}{
			"email":    "login@example.com",
			"password": hash,
		})

		user, session, err := svc.Login("login@example.com", "correctpassword")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if session == nil {
			t.Fatal("expected session, got nil")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		hash, _ := hashPasswordForTest("correctpassword")
		seedServiceUser(t, db, map[string]interface{}{
			"email":    "login@example.com",
			"password": hash,
		})

		_, _, err := svc.Login("login@example.com", "wrongpassword")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("non-existent email", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		_, _, err := svc.Login("nonexistent@example.com", "password123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty credentials", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		_, _, err := svc.Login("", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestLoginOrRegisterGoogle(t *testing.T) {
	t.Run("existing google_id", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		seedServiceUser(t, db, map[string]interface{}{
			"email":     "google@example.com",
			"google_id": "google123",
			"username":  "googleuser",
		})

		user, session, err := svc.LoginOrRegisterGoogle("google123", "google@example.com", "Google User")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if session == nil {
			t.Fatal("expected session, got nil")
		}
		if user.GoogleID != "google123" {
			t.Errorf("expected google_id 'google123', got '%s'", user.GoogleID)
		}
	})

	t.Run("existing email, new google_id", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		seedServiceUser(t, db, map[string]interface{}{
			"email":    "existing@example.com",
			"username": "existinguser",
		})

		user, session, err := svc.LoginOrRegisterGoogle("new_google_id", "existing@example.com", "Existing User")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if session == nil {
			t.Fatal("expected session, got nil")
		}
	})

	t.Run("new user", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		user, session, err := svc.LoginOrRegisterGoogle("brand_new_google", "brandnew@example.com", "Brand New")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if session == nil {
			t.Fatal("expected session, got nil")
		}
		if user.GoogleID != "brand_new_google" {
			t.Errorf("expected google_id 'brand_new_google', got '%s'", user.GoogleID)
		}
	})
}

func TestCanModerate(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	user := seedServiceUser(t, db, map[string]interface{}{"username": "regular", "email": "regular@b.com", "role": model.RoleUser})
	mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "mod@b.com", "role": model.RoleModerator})
	admin := seedServiceUser(t, db, map[string]interface{}{"username": "admin", "email": "admin@b.com", "role": model.RoleAdmin})
	root := seedServiceUser(t, db, map[string]interface{}{"username": "root", "email": "root@b.com", "role": model.RoleRoot})

	tests := []struct {
		name   string
		actor  *model.User
		target *model.User
		want   bool
	}{
		{"mod can act on user", mod, user, true},
		{"admin can act on user", admin, user, true},
		{"root can act on user", root, user, true},
		{"user cannot act on mod", user, mod, false},
		{"user cannot act on admin", user, admin, false},
		{"mod cannot act on admin", mod, admin, false},
		{"root cannot act on root", root, root, false},
		{"nil actor", nil, user, false},
		{"nil target", user, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.CanModerate(tt.actor, tt.target)
			if got != tt.want {
				t.Errorf("CanModerate(%v, %v) = %v, want %v", tt.actor, tt.target, got, tt.want)
			}
		})
	}
}

func TestUpdateProfile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		user := seedServiceUser(t, db, nil)

		err := svc.UpdateProfile(user.ID, model.UpdateUserRequest{
			Username: "newname",
			Bio:      "new bio",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated, err := svc.GetByID(user.ID)
		if err != nil {
			t.Fatalf("get user: %v", err)
		}
		if updated.Username != "newname" {
			t.Errorf("expected username 'newname', got '%s'", updated.Username)
		}
		if updated.Bio != "new bio" {
			t.Errorf("expected bio 'new bio', got '%s'", updated.Bio)
		}
	})

	t.Run("empty username", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)
		user := seedServiceUser(t, db, nil)

		err := svc.UpdateProfile(user.ID, model.UpdateUserRequest{Username: "", Bio: ""})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		seedServiceUser(t, db, map[string]interface{}{"username": "existing", "email": "a@b.com"})
		user := seedServiceUser(t, db, map[string]interface{}{"username": "me", "email": "me@b.com"})

		err := svc.UpdateProfile(user.ID, model.UpdateUserRequest{Username: "existing", Bio: ""})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetByUsername(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	seedServiceUser(t, db, map[string]interface{}{"username": "findme"})

	user, err := svc.GetByUsername("findme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "findme" {
		t.Errorf("expected username 'findme', got '%s'", user.Username)
	}

	_, err = svc.GetByUsername("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestChangePassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		hash, _ := hashPasswordForTest("oldpass")
		user := seedServiceUser(t, db, map[string]interface{}{"password": hash})

		err := svc.ChangePassword(user.ID, "oldpass", "newpass123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Проверяем, что можно войти с новым паролем
		_, _, err = svc.Login(user.Email, "newpass123")
		if err != nil {
			t.Fatalf("login with new password failed: %v", err)
		}

		// Старый пароль не работает
		_, _, err = svc.Login(user.Email, "oldpass")
		if err == nil {
			t.Fatal("expected error with old password")
		}
	})

	t.Run("wrong old password", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		user := seedServiceUser(t, db, map[string]interface{}{"password": "$2a$10$dummyhash"})
		err := svc.ChangePassword(user.ID, "wrong", "newpass123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("short new password", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewUserService(db)

		hash, _ := hashPasswordForTest("oldpass")
		user := seedServiceUser(t, db, map[string]interface{}{"password": hash})

		err := svc.ChangePassword(user.ID, "oldpass", "12")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// hashPasswordForTest хеширует пароль с помощью bcrypt (cost=4 для скорости в тестах).
func hashPasswordForTest(password string) (string, error) {
	// Используем bcrypt напрямую с уменьшенным cost для тестов
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// isError проверяет, содержится ли target-ошибка в err.
func isError(err, target error) bool {
	if err == nil {
		return false
	}
	// Проверяем через Error() string
	return contains(err.Error(), target.Error())
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			contains(s[1:], substr)))
}
