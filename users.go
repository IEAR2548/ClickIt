package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailTaken = errors.New("email already registered")

type User struct {
	ID           int64
	Email        string
	PasswordHash string
}

type UserStorer interface {
	Create(ctx context.Context, email, passwordHash string) (int64, error)
	GetByEmail(ctx context.Context, email string) (User, error)
}

var _ UserStorer = (*UserStore)(nil)

type UserStore struct {
	db querier
}

func (s *UserStore) Create(ctx context.Context, email, passwordHash string) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id
	`, email, passwordHash).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrEmailTaken
		}
		return 0, fmt.Errorf("insert user: %w", err)
	}

	return id, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user: %w", err)
	}

	return u, nil
}
