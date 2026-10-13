package service

import (
	"context"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
)

type LoadTestService struct {
	repo *postgres.LoadTestRepo
}

func NewLoadTestService(repo *postgres.LoadTestRepo) *LoadTestService {
	return &LoadTestService{repo: repo}
}

func (s *LoadTestService) RecordEvent(ctx context.Context, req dto.LoadTestEventRequest) (int64, error) {
	if req.RunID == "" || req.Seq < 0 {
		return 0, ErrInvalidValue
	}

	return s.repo.Create(ctx, postgres.CreateLoadTestEventParams{
		RunID: req.RunID,
		Seq:   req.Seq,
	})
}

func (s *LoadTestService) Count(ctx context.Context, runID string) (int64, error) {
	if runID == "" {
		return 0, ErrInvalidValue
	}
	return s.repo.CountByRunID(ctx, runID)
}
