package model

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", ErrNotFound},
		{"ErrForbidden", ErrForbidden},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrValidation", ErrValidation},
		{"ErrRateLimited", ErrRateLimited},
		{"ErrBanned", ErrBanned},
		{"ErrDuplicate", ErrDuplicate},
		{"ErrSelfAction", ErrSelfAction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			// Проверка, что errors.Is работает
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("errors.Is(%[1]v, %[1]v) should be true", tt.err)
			}
			// Проверка, что они не пересекаются
			for _, other := range tests {
				if other.name != tt.name && errors.Is(tt.err, other.err) {
					t.Errorf("%s should not match %s", tt.name, other.name)
				}
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrNotFound, "resource not found"},
		{ErrForbidden, "access denied"},
		{ErrUnauthorized, "unauthorized"},
		{ErrValidation, "validation error"},
		{ErrRateLimited, "rate limit exceeded"},
		{ErrBanned, "user is banned"},
		{ErrDuplicate, "resource already exists"},
		{ErrSelfAction, "cannot perform action on yourself"},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("got %q, want %q", tt.err.Error(), tt.want)
			}
		})
	}
}
