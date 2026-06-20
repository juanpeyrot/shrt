package models

import (
	"time"

	"github.com/google/uuid"
)

type ShortURL struct {
	ID          uuid.UUID  `json:"id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	ClickCount  int64      `json:"click_count"`
}
