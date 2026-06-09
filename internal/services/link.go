package services

import (
	"errors"
	"shrt/internal/apierr"
	"shrt/internal/models"
	"shrt/internal/utils/validators"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreateShortURL(url models.ShortURL) error
	GetByShortCode(shortCode string) (string, error)
}

type LinkService struct {
	repo Repository
}

func NewLinkService(repo Repository) *LinkService {
	return &LinkService{repo: repo}
}

func (s *LinkService) CreateShortURL(shortCode string, originalURL string, expiresAt *time.Time) (models.ShortURL, error) {
	
	if originalURL == "" {
		return models.ShortURL{}, apierr.NewValidation("original_url is required")
	}

	if err := validators.ValidateSlug(shortCode); err != nil {
		return models.ShortURL{}, err
	}

	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return models.ShortURL{}, apierr.NewValidation("expires_at must be in the future")
	}

	shortURL := models.ShortURL{
		ID:          uuid.New(),
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	err := s.repo.CreateShortURL(shortURL)
	if err != nil {
		if errors.Is(err, ErrDuplicateShortCode) {
			return models.ShortURL{}, apierr.NewConflict("short_code already in use")
		}
		return models.ShortURL{}, apierr.NewInternal("failed to create short URL", err)
	}

	return shortURL, nil
}

func (s *LinkService) GetByShortCode(shortCode string) (string, error) {
	url, err := s.repo.GetByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, ErrShortCodeNotFound) {
			return "", apierr.NewNotFound("short URL not found")
		}
		return "", apierr.NewInternal("failed to get short URL", err)
	}
	return url, nil
}