package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TokenRepo struct {
	cluster *Cluster
}

func NewTokenRepo(cluster *Cluster) *TokenRepo {
	return &TokenRepo{cluster: cluster}
}

const insertToken = `
INSERT INTO auth_tokens (user_id, expires_at)
VALUES ($1, $2)
RETURNING token`

func (r *TokenRepo) Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (uuid.UUID, error) {
	var token uuid.UUID
	err := r.cluster.Master().QueryRowContext(ctx, insertToken, userID, expiresAt).Scan(&token)
	return token, err
}

const getUserIDByToken = `SELECT user_id FROM auth_tokens
	WHERE token = $1 AND NOT is_revoked AND expires_at > now()`

func (r *TokenRepo) LookupUserID(ctx context.Context, token uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.cluster.Master().QueryRowContext(ctx, getUserIDByToken, token).Scan(&userID)
	return userID, err
}
