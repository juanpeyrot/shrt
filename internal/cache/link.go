package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	linkPrefix = "link:"
	linkTTL    = 10 * time.Minute
)

type CachedLink struct {
	ID          uuid.UUID `json:"id"`
	OriginalURL string    `json:"original_url"`
}

type LinkCache struct {
	client *redis.Client
}

func NewLinkCache(client *redis.Client) *LinkCache {
	return &LinkCache{client: client}
}

func (c *LinkCache) Get(shortCode string) (CachedLink, error) {
	data, err := c.client.Get(context.Background(), linkPrefix+shortCode).Bytes()
	if err != nil {
		return CachedLink{}, err
	}
	var link CachedLink
	if err := json.Unmarshal(data, &link); err != nil {
		return CachedLink{}, err
	}
	return link, nil
}

func (c *LinkCache) Set(shortCode string, link CachedLink) error {
	data, err := json.Marshal(link)
	if err != nil {
		return err
	}
	return c.client.Set(context.Background(), linkPrefix+shortCode, data, linkTTL).Err()
}

func (c *LinkCache) Delete(shortCode string) error {
	return c.client.Del(context.Background(), linkPrefix+shortCode).Err()
}
