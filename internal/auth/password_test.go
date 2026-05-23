package auth

import (
	"testing"
)

func TestHashAndCheck(t *testing.T) {
	plain := "correct-horse-battery-staple"

	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}

	if !CheckPassword(hash, plain) {
		t.Error("CheckPassword() = false, want true")
	}
}

func TestWrongPassword(t *testing.T) {
	hash, _ := HashPassword("real-password")

	if CheckPassword(hash, "wrong-password") {
		t.Error("CheckPassword() = true, want false for wrong password")
	}
}

func TestHashCost(t *testing.T) {
	hash, _ := HashPassword("test")

	cost, err := HashCost(hash)
	if err != nil {
		t.Fatalf("HashCost() error = %v", err)
	}
	if cost != bcryptCost {
		t.Errorf("cost = %d, want %d", cost, bcryptCost)
	}
}

func TestEmptyPassword(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}
