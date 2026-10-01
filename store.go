package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("link not found")

type LinkStorer interface {
	CreateLink(ctx context.Context, longURL string, ownerID int64) (string, error)
	GetLongURL(ctx context.Context, code string) (string, error)
	ListByOwner(ctx context.Context, ownerID int64) ([]Link, error)
}

type Link struct {
	ShortCode string    `json:"short_code"`
	LongURL   string    `json:"long_url"`
	CreatedAt time.Time `json:"created_at"`
}

var _ LinkStorer = (*LinkStore)(nil)

type LinkStore struct {
	db *pgxpool.Pool
}

func (s *LinkStore) CreateLink(ctx context.Context, longURL string, ownerID int64) (string, error) {
	var id int64
	err := s.db.QueryRow(ctx, "SELECT nextval('links_id_seq')").Scan(&id)
	if err != nil {
		return "", fmt.Errorf("reserve id: %w", err)
	}

	code := EncodeBase62(id)

	_, err = s.db.Exec(ctx,
		`INSERT INTO links (id, short_code, long_url, user_id) VALUES ($1, $2, $3, $4)`,
		id, code, longURL, ownerID,
	)
	if err != nil {
		return "", fmt.Errorf("insert link: %w", err)
	}

	return code, nil
}

func (s *LinkStore) GetLongURL(ctx context.Context, code string) (string, error) {
	var longURL string
	err := s.db.QueryRow(ctx,
		`SELECT long_url FROM links WHERE short_code = $1`, code,
	).Scan(&longURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query link: %w", err)
	}

	return longURL, nil
}

func (s *LinkStore) ListByOwner(ctx context.Context, ownerID int64) ([]Link, error) {
	rows, err := s.db.Query(ctx, `
		SELECT short_code, long_url, created_at
		FROM links
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	defer rows.Close()

	links := []Link{}
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ShortCode, &l.LongURL, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan link row: %w", err)
		}
		links = append(links, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate link rows: %w", err)
	}

	return links, nil
}
