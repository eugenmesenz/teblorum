package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword возвращает bcrypt-хеш пароля.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", errors.New("password cannot be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword проверяет пароль против bcrypt-хеша.
func CheckPassword(hash, plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}

// HashCost возвращает bcrypt cost из существующего хеша (для тестов).
func HashCost(hash string) (int, error) {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return 0, fmt.Errorf("get bcrypt cost: %w", err)
	}
	return cost, nil
}
