package services

import "errors"

var (
	ErrDuplicateShortCode = errors.New("duplicate short code")
	ErrShortCodeNotFound  = errors.New("short code not found")
)
