package services

import "time"

type Repository interface {
	CreateShortURL(shortCode, originalURL string, expiresAt *time.Time) error
	GetByShortCode(shortCode string) (string, error)
}

type LinkService struct {
	repo Repository
}

func NewLinkService(repo Repository) *LinkService {
	return &LinkService{repo: repo}
}

func (s *LinkService) CreateShortURL(shortCode, originalURL string, expiresAt *time.Time) error {
	return s.repo.CreateShortURL(shortCode, originalURL, expiresAt)
}

func (s *LinkService) GetByShortCode(shortCode string) (string, error) {
	return s.repo.GetByShortCode(shortCode)
}