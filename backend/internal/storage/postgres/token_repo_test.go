package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestTokenLookupUserIDPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.ExecContext(ctx, `CREATE TEMP TABLE auth_tokens (
		token UUID PRIMARY KEY, user_id UUID NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL, is_revoked BOOLEAN NOT NULL DEFAULT FALSE)`)
	if err != nil {
		t.Fatal(err)
	}
	userID, valid, expired, revoked := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO auth_tokens (token, user_id, expires_at, is_revoked) VALUES
		($1, $4, now() + interval '1 hour', false),
		($2, $4, now() - interval '1 hour', false),
		($3, $4, now() + interval '1 hour', true)`, valid, expired, revoked, userID)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewTokenRepo(&Cluster{master: db})
	got, err := repo.LookupUserID(ctx, valid)
	if err != nil || got != userID {
		t.Fatalf("valid token: got %s, %v; want %s", got, err, userID)
	}
	for _, token := range []uuid.UUID{expired, revoked, uuid.New()} {
		if _, err := repo.LookupUserID(ctx, token); !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("invalid token %s: got error %v, want sql.ErrNoRows", token, err)
		}
	}
}
