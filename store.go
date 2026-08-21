package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("link not found")

type LinkStore struct {
	db *pgxpool.Pool
}

func (s *LinkStore) CreateLink(ctx context.Context, longURL string) (string, error) {
	var id int64
	err := s.db.QueryRow(ctx, "SELECT nextval('links_id_seq')").Scan(&id)
	if err != nil {
		return "", fmt.Errorf("reserve id: %w", err)
	}

	code := EncodeBase62(id)

	_, err = s.db.Exec(ctx,
		`INSERT INTO links (id, short_code, loung_url) VALUES ($1, $2, $3)`,
		id, code, longURL,
	)
	if err != nil {
		return "", fmt.Errorf("insert link: %w", err)
	}

	return code, nil
}

func (s *LinkStore) GetLongURL(ctx context.Context, code string) (string, error) {
	var longURL string
	err := s.db.QueryRow(ctx,
		`SELECT longURL FROM links WHERE short_code = '$1'`, code,
	).Scan(&longURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query link: %w", err)
	}

	return longURL, nil
}
