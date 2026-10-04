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

func withCORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
	var rdb *redis.Client

	switch {
	case os.Getenv("REDIS_URL") != "":
		opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
		if err != nil {
			log.Fatalf("invalid REDIS_URL: %v", err)
		}
		rdb = redis.NewClient(opts)
	case os.Getenv("REDIS_ADDR") != "":
		rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})
	}

	if rdb != nil {
		defer rdb.Close()
		if err := rdb.Ping(connectCtx).Err(); err != nil {
			log.Fatalf("unable to reach redis: %v", err)
		}
		store = &CachedLinkStore{LinkStorer: store, cache: rdb, ttl: time.Hour}
		log.Println("redis cache enabled")
	} else {
		log.Println("REDIS_URL/REDIS_ADDR not set -- running without cache")
	}

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 bytes long")
	}

	clicklogger := NewClickLogger(pool, 1000)
	analyticsStore := &AnalyticsStore{db: pool}
	userStore := &UserStore{db: pool}

	workerCtx, stopWorker := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clicklogger.Run(workerCtx)
	}()

	h := &Handler{
		store:     store,
		users:     userStore,
		clicks:    clicklogger,
		analytics: analyticsStore,
		jwtSecret: jwtSecret,
	}

	withRateLimit := func(limit int, window time.Duration, h http.Handler) http.Handler {
		if rdb == nil {
			return h
		}
		return RateLimit(rdb, limit, window)(h)
	}

	mux := http.NewServeMux()
	mux.Handle("POST /register", withRateLimit(5, time.Hour, http.HandlerFunc(h.Register)))
	mux.Handle("POST /login", withRateLimit(10, time.Minute, http.HandlerFunc(h.Login)))
	mux.Handle("POST /shorten", withRateLimit(20, time.Minute, RequireAuth(jwtSecret)(http.HandlerFunc(h.CreateShortLink))))
	mux.Handle("GET /links", RequireAuth(jwtSecret)(http.HandlerFunc(h.ListMyLinks)))

	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/stats/{code}", h.AnalyticsSummary)
	mux.HandleFunc("GET /{code}", h.Redirect)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:5173"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      withCORS(frontendOrigin, mux),
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
