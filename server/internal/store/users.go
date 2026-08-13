package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// CreateAnonymousUser inserts a fresh anonymous user owning the given token
// hash and returns its id. The caller generates the token and hashes it —
// only the hash ever crosses this boundary or touches disk.
func (s *Store) CreateAnonymousUser(ctx context.Context, tokenHash string) (string, error) {
	var id string
	if err := s.pool.QueryRow(ctx,
		`INSERT INTO users (token_hash) VALUES ($1) RETURNING id;`,
		tokenHash).Scan(&id); err != nil {
		return "", fmt.Errorf("create anonymous user: %w", err)
	}
	return id, nil
}

// UserForToken resolves a token hash to a user id, stamping last_seen on the
// way through. An unknown token returns "" with no error — identity is
// optional on every request, so a stale token must never fail a call.
func (s *Store) UserForToken(ctx context.Context, tokenHash string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`UPDATE users SET last_seen = now() WHERE token_hash = $1 RETURNING id;`,
		tokenHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve token: %w", err)
	}
	return id, nil
}
