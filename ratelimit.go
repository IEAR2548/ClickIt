package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

func RateLimit(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := fmt.Sprintf("ratelimit:%s:%s", r.URL.Path, clientIP(r))

			count, err := rdb.Incr(r.Context(), key).Result()
			if err != nil {
				log.Printf("rate limit check failed, allowing request: %v", err)
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				rdb.Expire(context.Background(), key, window)
			}

			if count > int64(limit) {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
				http.Error(w, "rate limit exceeded, try again later", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
