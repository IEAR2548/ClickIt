package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to reach database:%v", err)
	}

	var store LinkStorer = &LinkStore{db: pool}

	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Fatalf("unable to reach redis: %v", err)
		}
		store = &CachedLinkStore{LinkStorer: store, cache: rdb, ttl: time.Hour}
		log.Println("redis cache enabled")
	} else {
		log.Println("REDIS_ADDR not set -- running without cache")
	}

	h := &Handler{store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", h.CreateShortLink)
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /{code}", h.Redirect)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
