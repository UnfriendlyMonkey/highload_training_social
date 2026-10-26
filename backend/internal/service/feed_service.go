package service

import (
	"context"
	"errors"
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/cache"
	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/google/uuid"
)

const feedRebuildBatch = 100

type FeedService struct {
	repo  *postgres.SocialRepo
	cache *cache.FeedCache
}

func NewFeedService(repo *postgres.SocialRepo, feedCache *cache.FeedCache) *FeedService {
	return &FeedService{repo: repo, cache: feedCache}
}

func ValidateFeedPage(offset, limit int) (int, int, error) {
	if offset < 0 || limit < 1 || limit > cache.MaxFeedItems {
		return 0, 0, ErrInvalidValue
	}
	return offset, limit, nil
}

func (s *FeedService) Get(ctx context.Context, user uuid.UUID, offset, limit int) ([]dto.PostResponse, error) {
	if err := s.RebuildUserIfPending(ctx, user); err != nil {
		return nil, err
	}
	posts, err := s.cache.Get(ctx, user, offset, limit)
	if errors.Is(err, cache.ErrFeedNotReady) {
		if err = s.RebuildUser(ctx, user); err != nil {
			return nil, err
		}
		posts, err = s.cache.Get(ctx, user, offset, limit)
	}
	if err != nil {
		return nil, err
	}
	result := make([]dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		result = append(result, *new(dto.PostResponse).FromEntity(post))
	}
	return result, nil
}

func (s *FeedService) RebuildUser(ctx context.Context, user uuid.UUID) error {
	posts, err := s.repo.FeedPosts(ctx, user, cache.MaxFeedItems)
	if err != nil {
		return err
	}
	return s.cache.Rebuild(ctx, user, posts)
}

func (s *FeedService) RebuildUserIfPending(ctx context.Context, user uuid.UUID) error {
	revision, pending, err := s.repo.PendingFeedRebuild(ctx, user)
	if err != nil || !pending {
		return err
	}
	if err = s.RebuildUser(ctx, user); err != nil {
		return err
	}
	return s.repo.AcknowledgeFeedRebuild(ctx, user, revision)
}

func (s *FeedService) RebuildPending(ctx context.Context, limit int) (int, error) {
	pending, err := s.repo.PendingFeedRebuilds(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, rebuild := range pending {
		if err = s.RebuildUser(ctx, rebuild.UserID); err != nil {
			return 0, err
		}
		if err = s.repo.AcknowledgeFeedRebuild(ctx, rebuild.UserID, rebuild.Revision); err != nil {
			return 0, err
		}
	}
	return len(pending), nil
}

func (s *FeedService) RunRebuilder(ctx context.Context, reportError func(error)) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	lastError := ""
	for {
		if _, err := s.RebuildPending(ctx, feedRebuildBatch); err != nil {
			if reportError != nil && err.Error() != lastError {
				reportError(err)
			}
			lastError = err.Error()
		} else {
			lastError = ""
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
