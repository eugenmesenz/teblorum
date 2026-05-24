package config

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/teblorum/teblorum/internal/auth"
	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

// BootstrapRoot создаёт root-пользователя при первом запуске,
// если указан email в конфигурации.
func BootstrapRoot(db *sql.DB, email, password string) error {
	if email == "" {
		return nil
	}

	userRepo := repo.NewUserRepo(db)

	// Проверяем, существует ли уже root-пользователь
	existing, err := userRepo.GetByEmail(email)
	if err == nil && existing != nil {
		// Пользователь уже существует — проверяем, root ли он
		if existing.Role == model.RoleRoot {
			log.Printf("root-пользователь %s уже существует", email)
			return nil
		}
		// Повышаем до root
		if err := userRepo.UpdateRole(existing.ID, model.RoleRoot); err != nil {
			return fmt.Errorf("promote to root: %w", err)
		}
		log.Printf("пользователь %s повышен до root", email)
		return nil
	}

	// Создаём нового root-пользователя
	if password == "" {
		return fmt.Errorf("bootstrap password required for email %s", email)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}

	user, err := userRepo.Create(model.CreateUserRequest{
		Username: "root",
		Email:    email,
		Password: hash,
		Role:     model.RoleRoot,
	})
	if err != nil {
		return fmt.Errorf("create root user: %w", err)
	}

	log.Printf("создан root-пользователь %s (id=%d)", email, user.ID)
	return nil
}