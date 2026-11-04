package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type TokenLookup interface {
	LookupUserID(context.Context, uuid.UUID) (uuid.UUID, error)
}

type userIDKey struct{}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	return id, ok
}

func Authenticate(tokens TokenLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := r.Header.Values("Authorization")
			if len(headers) != 1 {
				unauthorized(w)
				return
			}
			fields := strings.Fields(headers[0])
			if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
				unauthorized(w)
				return
			}
			token, err := uuid.Parse(fields[1])
			if err != nil {
				unauthorized(w)
				return
			}
			userID, err := tokens.LookupUserID(r.Context(), token)
			if errors.Is(err, sql.ErrNoRows) {
				unauthorized(w)
				return
			}
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey{}, userID)))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}
