package model

import "errors"

var (
	ErrNotFound      = errors.New("resource not found")
	ErrForbidden     = errors.New("access denied")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrValidation    = errors.New("validation error")
	ErrRateLimited   = errors.New("rate limit exceeded")
	ErrBanned        = errors.New("user is banned")
	ErrDuplicate     = errors.New("resource already exists")
	ErrSelfAction    = errors.New("cannot perform action on yourself")
)