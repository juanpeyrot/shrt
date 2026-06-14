package services

import "errors"

var (
	ErrDuplicateShortCode = errors.New("duplicate short code")
	ErrShortCodeNotFound  = errors.New("short code not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenReuse         = errors.New("refresh token reuse detected")
)
