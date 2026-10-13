package postgres

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
)

type Cluster struct {
	master     *sql.DB
	replicas   []*sql.DB
	replicaIdx atomic.Uint64
}

func NewCluster(ctx context.Context, masterURL string, replicaURLs []string) (*Cluster, error) {
	master, err := Open(ctx, masterURL)
	if err != nil {
		return nil, err
	}

	var replicas []*sql.DB
	for _, url := range replicaURLs {
		if url == "" {
			continue
		}
		replica, err := Open(ctx, url)
		if err != nil {
			for _, db := range replicas {
				_ = db.Close()
			}
			_ = master.Close()
			return nil, err
		}
		replicas = append(replicas, replica)
	}

	return &Cluster{
		master:   master,
		replicas: replicas,
	}, nil
}

func (c *Cluster) Master() *sql.DB {
	return c.master
}

func (c *Cluster) Replica() *sql.DB {
	if len(c.replicas) == 0 {
		return c.master
	}
	idx := c.replicaIdx.Add(1) - 1
	return c.replicas[idx%uint64(len(c.replicas))]
}

func (c *Cluster) Close() error {
	var errs []error
	if err := c.master.Close(); err != nil {
		errs = append(errs, err)
	}
	for _, db := range c.replicas {
		if err := db.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
