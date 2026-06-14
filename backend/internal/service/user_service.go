package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *postgres.UserRepo
}

func NewUserService(repo *postgres.UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(ctx context.Context, req dto.RegisterRequest) (string, error) {
	if req.FirstName == "" || req.SecondName == "" || req.Password == "" {
		return "", ErrInvalidValue
	}
	if req.Birthdate != "" {
		if _, err := time.Parse(time.DateOnly, req.Birthdate); err != nil {
			return "", ErrInvalidValue
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("error generating password hash", err)
		return "", err
	}

	id, err := s.repo.Create(ctx, postgres.CreateUserParams{
		FirstName:    req.FirstName,
		SecondName:   req.SecondName,
		Biography:    req.Biography,
		Birthdate:    req.Birthdate,
		City:         req.City,
		PasswordHash: string(hash),
	})
	if err != nil {
		log.Println("error creating user", err)
		return "", err
	}
	return id.String(), nil
}

func (s *UserService) Get(ctx context.Context, idStr string) (*dto.UserResponse, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, ErrInvalidValue
	}

	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	resp := new(dto.UserResponse).FromEntity(user)
	return resp, nil
}

func (s *UserService) Search(ctx context.Context, firstName, secondName string) ([]*dto.UserResponse, error) {
	users, err := s.repo.Search(ctx, firstName, secondName)
	if err != nil {
		return nil, err
	}

	return dto.UserResponsesFromEntities(users), nil
}
