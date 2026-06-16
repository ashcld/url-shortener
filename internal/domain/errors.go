package domain

import "errors"

var (
	ErrURLNotFound   = errors.New("url not found")
	ErrURLExpired    = errors.New("url expired")
	ErrCodeCollision = errors.New("short code collision, retry")
	ErrInvalidURL    = errors.New("invalid url")
)
