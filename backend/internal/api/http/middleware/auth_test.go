package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

type tokenLookupFunc func(context.Context, uuid.UUID) (uuid.UUID, error)

func (f tokenLookupFunc) LookupUserID(ctx context.Context, token uuid.UUID) (uuid.UUID, error) {
	return f(ctx, token)
}

func TestAuthenticate(t *testing.T) {
	token := uuid.New()
	userID := uuid.New()
	tests := []struct {
		name    string
		headers []string
		lookup  error
		status  int
		called  bool
	}{
		{"valid", []string{"Bearer " + token.String()}, nil, http.StatusNoContent, true},
		{"case insensitive scheme", []string{"bearer " + token.String()}, nil, http.StatusNoContent, true},
		{"missing", nil, nil, http.StatusUnauthorized, false},
		{"multiple headers", []string{"Bearer " + token.String(), "Bearer " + token.String()}, nil, http.StatusUnauthorized, false},
		{"wrong scheme", []string{"Basic " + token.String()}, nil, http.StatusUnauthorized, false},
		{"malformed token", []string{"Bearer invalid"}, nil, http.StatusUnauthorized, false},
		{"extra field", []string{"Bearer " + token.String() + " extra"}, nil, http.StatusUnauthorized, false},
		{"unknown expired or revoked", []string{"Bearer " + token.String()}, sql.ErrNoRows, http.StatusUnauthorized, false},
		{"database failure", []string{"Bearer " + token.String()}, errors.New("database unavailable"), http.StatusInternalServerError, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			lookup := tokenLookupFunc(func(_ context.Context, got uuid.UUID) (uuid.UUID, error) {
				if got != token {
					t.Errorf("token = %s, want %s", got, token)
				}
				return userID, tt.lookup
			})
			h := Authenticate(lookup)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if got, ok := UserIDFromContext(r.Context()); !ok || got != userID {
					t.Errorf("user ID = %s, present = %v; want %s", got, ok, userID)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, header := range tt.headers {
				req.Header.Add("Authorization", header)
			}
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != tt.status || called != tt.called {
				t.Errorf("status = %d, handler called = %v; want %d, %v", res.Code, called, tt.status, tt.called)
			}
			if res.Code == http.StatusUnauthorized && res.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Error("missing bearer challenge")
			}
		})
	}
}

func TestUserIDFromContextWithoutAuthentication(t *testing.T) {
	if _, ok := UserIDFromContext(context.Background()); ok {
		t.Fatal("unexpected user ID")
	}
}
