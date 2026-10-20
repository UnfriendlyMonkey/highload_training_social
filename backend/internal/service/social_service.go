package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/google/uuid"
)

type SocialService struct{ repo *postgres.SocialRepo }

func NewSocialService(repo *postgres.SocialRepo) *SocialService { return &SocialService{repo: repo} }

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrInvalidValue
	}
	return id, nil
}
func socialError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
func (s *SocialService) SetFriend(ctx context.Context, user uuid.UUID, friend string) error {
	id, err := parseID(friend)
	if err != nil || id == user {
		return ErrInvalidValue
	}
	return socialError(s.repo.SetFriend(ctx, user, id))
}

func (s *SocialService) DeleteFriend(ctx context.Context, user uuid.UUID, friend string) error {
	id, err := parseID(friend)
	if err != nil || id == user {
		return ErrInvalidValue
	}
	return socialError(s.repo.DeleteFriend(ctx, user, id))
}
func (s *SocialService) CreatePost(ctx context.Context, author uuid.UUID, req dto.CreatePostRequest) (dto.CreatePostResponse, error) {
	if req.Text == nil || strings.TrimSpace(*req.Text) == "" {
		return "", ErrInvalidValue
	}
	id, err := s.repo.CreatePost(ctx, author, *req.Text)
	if err != nil {
		return "", err
	}
	return dto.CreatePostResponse(id.String()), nil
}

func (s *SocialService) UpdatePost(ctx context.Context, author uuid.UUID, req dto.UpdatePostRequest) error {
	postID, err := parseID(req.ID)
	if err != nil || req.Text == nil || strings.TrimSpace(*req.Text) == "" {
		return ErrInvalidValue
	}
	return socialError(s.repo.UpdatePost(ctx, author, postID, *req.Text))
}

func (s *SocialService) DeletePost(ctx context.Context, author uuid.UUID, id string) error {
	postID, err := parseID(id)
	if err != nil {
		return ErrInvalidValue
	}
	return socialError(s.repo.DeletePost(ctx, author, postID))
}

func (s *SocialService) GetPost(ctx context.Context, id string) (*dto.PostResponse, error) {
	postID, err := parseID(id)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.GetPost(ctx, postID)
	if err != nil {
		return nil, socialError(err)
	}
	return new(dto.PostResponse).FromEntity(post), nil
}
