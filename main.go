package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	connectCtx, connectCancel := context.WithTimeout(ctx, 5*time.Second)
	defer connectCancel()

	pool, err := pgxpool.New(connectCtx, dbURL)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(connectCtx); err != nil {
		log.Fatalf("Unable to reach database:%v", err)
	}

	var store LinkStorer = &LinkStore{db: pool}

	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
		defer rdb.Close()
		if err := rdb.Ping(connectCtx).Err(); err != nil {
			log.Fatalf("unable to reach redis: %v", err)
		}
		store = &CachedLinkStore{LinkStorer: store, cache: rdb, ttl: time.Hour}
		log.Println("redis cache enabled")
	} else {
		log.Println("REDIS_ADDR not set -- running without cache")
	}

	clicklogger := NewClickLogger(pool, 1000)
	analyticsStore := &AnalyticsStore{db: pool}

	workerCtx, stopWorker := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clicklogger.Run(workerCtx)
	}()

	h := &Handler{store: store, clicks: clicklogger, analytics: analyticsStore}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", h.CreateShortLink)
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/stats/{code}", h.AnalyticsSummary)
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

	go func() {
		log.Printf("listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	// Wait worker drain queue in db
	stopWorker()
	wg.Wait()
	log.Printf("shutdown complete")
}
