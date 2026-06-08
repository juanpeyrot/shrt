package memory

import (
	"fmt"
	"shrt/internal/models"
	"sync"
)

type LinkRepository struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewLinkRepository() *LinkRepository {
	return &LinkRepository{store: make(map[string]string)}
}

func (r *LinkRepository) CreateShortURL(url models.ShortURL) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[url.ShortCode] = url.OriginalURL
	return nil
}

func (r *LinkRepository) GetByShortCode(shortCode string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	url, ok := r.store[shortCode]
	if !ok {
		return "", fmt.Errorf("short code %q not found", shortCode)
	}
	return url, nil
}
