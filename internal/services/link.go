package services

import (
	"errors"
	"fmt"
	"log/slog"
	"shrt/internal/apierr"
	"shrt/internal/models"
	"shrt/internal/utils/shortcode"
	"shrt/internal/utils/validators"
	"time"

	"github.com/google/uuid"
)

const maxShortCodeRetries = 5

type LinkRepository interface {
	CreateShortURL(url models.ShortURL) error
	GetByShortCode(shortCode string) (string, error)
	GetLinkByShortCode(shortCode string) (models.ShortURL, error)
	UpdateOriginalURL(shortCode, originalURL string) error
	SoftDelete(shortCode string) error
	AddClick(linkID fmt.Stringer, referer string) error
	GetStats(linkID fmt.Stringer, days int) (models.LinkStats, error)
}

type LinkService struct {
	repo LinkRepository
}

func NewLinkService(repo LinkRepository) *LinkService {
	return &LinkService{repo: repo}
}

func (s *LinkService) CreateShortURL(userID *uuid.UUID, shortCode string, originalURL string, expiresAt *time.Time) (models.ShortURL, error) {
	if originalURL == "" {
		return models.ShortURL{}, apierr.NewValidation("original_url is required")
	}
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return models.ShortURL{}, apierr.NewValidation("expires_at must be in the future")
	}

	if shortCode == "" {
		return s.createWithAutoCode(userID, originalURL, expiresAt)
	}
	return s.createWithUserCode(userID, shortCode, originalURL, expiresAt)
}

func (s *LinkService) GetByShortCode(shortCode, referer string) (string, error) {
	link, err := s.repo.GetLinkByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, ErrShortCodeNotFound) {
			return "", apierr.NewNotFound("short URL not found")
		}
		return "", apierr.NewInternal("failed to get short URL", err)
	}

	if err := s.repo.AddClick(link.ID, referer); err != nil {
		slog.Error("failed to record click", "short_code", shortCode, "err", err)
	}

	return link.OriginalURL, nil
}

func (s *LinkService) RetrieveLink(userID uuid.UUID, shortCode string) (models.ShortURL, error) {
	link, err := s.repo.GetLinkByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, ErrShortCodeNotFound) {
			return models.ShortURL{}, apierr.NewNotFound("short URL not found")
		}
		return models.ShortURL{}, apierr.NewInternal("failed to get short URL", err)
	}

	if link.UserID == nil || *link.UserID != userID {
		return models.ShortURL{}, apierr.NewForbidden("you don't own this short URL")
	}

	return link, nil
}

func (s *LinkService) UpdateLink(userID uuid.UUID, shortCode, originalURL string) (models.ShortURL, error) {
	if originalURL == "" {
		return models.ShortURL{}, apierr.NewValidation("original_url is required")
	}

	link, err := s.RetrieveLink(userID, shortCode)
	if err != nil {
		return models.ShortURL{}, err
	}

	if err := s.repo.UpdateOriginalURL(shortCode, originalURL); err != nil {
		return models.ShortURL{}, apierr.NewInternal("failed to update short URL", err)
	}

	link.OriginalURL = originalURL
	return link, nil
}

func (s *LinkService) DeleteLink(userID uuid.UUID, shortCode string) error {
	if _, err := s.RetrieveLink(userID, shortCode); err != nil {
		return err
	}

	if err := s.repo.SoftDelete(shortCode); err != nil {
		return apierr.NewInternal("failed to delete short URL", err)
	}

	return nil
}

func (s *LinkService) GetStats(userID uuid.UUID, shortCode string, days int) (models.LinkStats, error) {
	link, err := s.RetrieveLink(userID, shortCode)
	if err != nil {
		return models.LinkStats{}, err
	}

	stats, err := s.repo.GetStats(link.ID, days)
	if err != nil {
		return models.LinkStats{}, apierr.NewInternal("failed to get stats", err)
	}

	return stats, nil
}

func (s *LinkService) createWithUserCode(userID *uuid.UUID, shortCode, originalURL string, expiresAt *time.Time) (models.ShortURL, error) {
	if err := validators.ValidateSlug(shortCode); err != nil {
		return models.ShortURL{}, err
	}
	url := models.ShortURL{
		ID:          uuid.New(),
		UserID:      userID,
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

func (s *LinkService) createWithAutoCode(userID *uuid.UUID, originalURL string, expiresAt *time.Time) (models.ShortURL, error) {
	for range maxShortCodeRetries {
		code, err := shortcode.Generate()
		if err != nil {
			return models.ShortURL{}, apierr.NewInternal("failed to generate short code", err)
		}
		url := models.ShortURL{
			ID:          uuid.New(),
			UserID:      userID,
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