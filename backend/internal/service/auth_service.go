package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 24 * time.Hour // TODO: take from config

type AuthService struct {
	users  *postgres.UserRepo
	tokens *postgres.TokenRepo
}

func NewAuthService(users *postgres.UserRepo, tokens *postgres.TokenRepo) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (string, error) {
	if req.ID == "" || req.Password == "" {
		return "", ErrInvalidValue
	}

	userID, err := uuid.Parse(req.ID)
	if err != nil {
		return "", ErrInvalidValue
	}

	hash, err := s.users.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		return "", ErrNotFound
	}

	token, err := s.tokens.Create(ctx, userID, time.Now().Add(tokenTTL))
	if err != nil {
		return "", err
	}

	return token.String(), nil
}
