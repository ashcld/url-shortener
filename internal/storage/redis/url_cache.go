package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ashcloud/url-shortener/internal/domain"
	"github.com/redis/go-redis/v9"
)

const keyPrefix = "url:"

type URLCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewURLCache(client *redis.Client, ttl time.Duration) *URLCache {
	return &URLCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *URLCache) Set(ctx context.Context, shortCode, originalURL string) error {
	if err := c.client.Set(ctx, keyPrefix+shortCode, originalURL, c.ttl).Err(); err != nil {
		return fmt.Errorf("cache set %s: %w", shortCode, err)
	}
	return nil
}

// Get возвращает оригинальный URL из кэша.
func (c *URLCache) Get(ctx context.Context, shortCode string) (string, error) {
	val, err := c.client.Get(ctx, keyPrefix+shortCode).Result()
	if errors.Is(err, redis.Nil) {
		return "", domain.ErrURLNotFound
	}
	if err != nil {
		return "", fmt.Errorf("cache get %s: %w", shortCode, err)
	}
	return val, nil
}

func (c *URLCache) Delete(ctx context.Context, shortCode string) error {
	if err := c.client.Del(ctx, keyPrefix+shortCode).Err(); err != nil {
		return fmt.Errorf("cache del %s: %w", shortCode, err)
	}
	return nil
}
