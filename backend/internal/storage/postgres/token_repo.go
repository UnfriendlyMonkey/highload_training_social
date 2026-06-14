package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type TokenRepo struct {
	db *sql.DB
}

func NewTokenRepo(db *sql.DB) *TokenRepo {
	return &TokenRepo{db: db}
}

const insertToken = `
INSERT INTO auth_tokens (user_id, expires_at)
VALUES ($1, $2)
RETURNING token`

func (r *TokenRepo) Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (uuid.UUID, error) {
	var token uuid.UUID
	err := r.db.QueryRowContext(ctx, insertToken, userID, expiresAt).Scan(&token)
	return token, err
}
