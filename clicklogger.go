package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// execer is satisfied by *pgxpool.Pool
type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type ClickEvent struct {
	ShortCode string
	IPAddress string
	UserAgent string
	Referrer  string
	Timestamp time.Time
}

type ClickLogger struct {
	events chan ClickEvent
	db     execer
}

func NewClickLogger(db execer, bufferSize int) *ClickLogger {
	return &ClickLogger{
		events: make(chan ClickEvent, bufferSize),
		db:     db,
	}
}

// HTTP Handler puts job into channel
func (c *ClickLogger) Log(e ClickEvent) {
	select {
	case c.events <- e:
	default:
		log.Printf("click event dropped (buffer full): %s", e.ShortCode)
	}
}

// Worker
func (c *ClickLogger) Run(ctx context.Context) {
	for {
		select {
		case e := <-c.events:
			c.write(context.Background(), e)
		case <-ctx.Done():
			c.drain()
			return
		}
	}
}

// Clear Queue
func (c *ClickLogger) drain() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for {
		select {
		case e := <-c.events:
			c.write(ctx, e)
		default:
			return // channel empty
		}
	}
}

func (c *ClickLogger) write(ctx context.Context, e ClickEvent) {
	_, err := c.db.Exec(ctx,
		`INSERT INTO clicks (short_code, ip_address, user_agent, referrer, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		e.ShortCode, e.IPAddress, e.UserAgent, e.Referrer, e.Timestamp)
	if err != nil {
		log.Printf("failed to write click for %q: %v", e.ShortCode, err)
	}
}
