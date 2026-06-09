package services

import (
	"errors"
	"shrt/internal/apierr"
	"shrt/internal/models"
	"shrt/internal/utils/shortcode"
	"shrt/internal/utils/validators"
	"time"

	"github.com/google/uuid"
)

const maxShortCodeRetries = 5

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
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return models.ShortURL{}, apierr.NewValidation("expires_at must be in the future")
	}

	if shortCode == "" {
		return s.createWithAutoCode(originalURL, expiresAt)
	}
	return s.createWithUserCode(shortCode, originalURL, expiresAt)
}

func (s *LinkService) createWithUserCode(shortCode, originalURL string, expiresAt *time.Time) (models.ShortURL, error) {
	if err := validators.ValidateSlug(shortCode); err != nil {
		return models.ShortURL{}, err
	}
	url := models.ShortURL{
		ID:          uuid.New(),
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}
	if err := s.repo.CreateShortURL(url); err != nil {
		if errors.Is(err, ErrDuplicateShortCode) {
			return models.ShortURL{}, apierr.NewConflict("short_code already in use")
		}
		return models.ShortURL{}, apierr.NewInternal("failed to create short URL", err)
	}
	return url, nil
}

func (s *LinkService) createWithAutoCode(originalURL string, expiresAt *time.Time) (models.ShortURL, error) {
	for range maxShortCodeRetries {
		code, err := shortcode.Generate()
		if err != nil {
			return models.ShortURL{}, apierr.NewInternal("failed to generate short code", err)
		}
		url := models.ShortURL{
			ID:          uuid.New(),
			ShortCode:   code,
			OriginalURL: originalURL,
			CreatedAt:   time.Now(),
			ExpiresAt:   expiresAt,
		}
		err = s.repo.CreateShortURL(url)
		if err == nil {
			return url, nil
		}
		if !errors.Is(err, ErrDuplicateShortCode) {
			return models.ShortURL{}, apierr.NewInternal("failed to create short URL", err)
		}
	}
	return models.ShortURL{}, apierr.NewInternal("could not generate a unique short code", nil)
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