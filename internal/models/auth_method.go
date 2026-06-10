package models

import (
	"time"

	"github.com/google/uuid"
)

type AuthMethod struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"user_id"`
	Provider          string    `json:"provider"`
	ProviderUserID    string    `json:"provider_id"`
	PasswordHash      *string   `json:"password_hash,omitempty"`
	RefreshTokenHash  *string   `json:"-"`
	RefreshTokenJWTID *string   `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
