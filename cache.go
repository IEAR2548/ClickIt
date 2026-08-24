package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedLinkStore struct {
	LinkStorer
	cache *redis.Client
	ttl   time.Duration
}

func (c *CachedLinkStore) GetLongURL(ctx context.Context, code string) (string, error) {
	key := cacheKey(code)

	val, err := c.cache.Get(ctx, key).Result()
	if err == nil {
		return val, nil // cache HIT
	}
	if err != redis.Nil {
		log.Printf("cache get error for %q: %v", key, err)
	}

	// cache MISS
	longURL, err := c.LinkStorer.GetLongURL(ctx, code)
	if err != nil {
		return "", err
	}

	if setErr := c.cache.Set(ctx, key, longURL, c.ttl).Err(); setErr != nil {
		log.Printf("cache set error for %q: %v", key, setErr)
	}

	return longURL, nil
}

func cacheKey(code string) string {
	return "link:" + code
}
