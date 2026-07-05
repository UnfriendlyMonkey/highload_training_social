package postgres

import (
	"context"
)

type CreateLoadTestEventParams struct {
	RunID string
	Seq   int64
}

type LoadTestRepo struct {
	cluster *Cluster
}

func NewLoadTestRepo(cluster *Cluster) *LoadTestRepo {
	return &LoadTestRepo{cluster: cluster}
}

const insertLoadTestEvent = `
	INSERT INTO load_test_events (run_id, seq)
	VALUES ($1, $2)
	RETURNING id`

func (r *LoadTestRepo) Create(ctx context.Context, e CreateLoadTestEventParams) (int64, error) {
	var id int64
	err := r.cluster.Master().QueryRowContext(ctx, insertLoadTestEvent, e.RunID, e.Seq).Scan(&id)
	return id, err
}

const countLoadTestEvents = `
	SELECT count(*) FROM load_test_events WHERE run_id = $1`

func (r *LoadTestRepo) CountByRunID(ctx context.Context, runID string) (int64, error) {
	var count int64
	err := r.cluster.Master().QueryRowContext(ctx, countLoadTestEvents, runID).Scan(&count)
	return count, err
}
