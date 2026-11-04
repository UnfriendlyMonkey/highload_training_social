// Package cache contains Redis-backed, derived application data.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const MaxFeedItems = 1000

var ErrFeedNotReady = errors.New("feed cache is not ready")

type FeedCache struct {
	client redis.UniversalClient
}

func NewFeedCache(redisURL string) (*FeedCache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &FeedCache{client: redis.NewClient(options)}, nil
}

func (c *FeedCache) Close() error { return c.client.Close() }

func (c *FeedCache) Ping(ctx context.Context) error { return c.client.Ping(ctx).Err() }

func feedKey(user uuid.UUID) string      { return "feed:" + user.String() }
func feedReadyKey(user uuid.UUID) string { return feedKey(user) + ":ready" }

type cachedPost struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	AuthorID string `json:"author_user_id"`
}

func toCachedPost(post postgres.Post) cachedPost {
	return cachedPost{
		ID:       post.ID.String(),
		Text:     post.Text,
		AuthorID: post.AuthorID.String(),
	}
}

func (p cachedPost) toPost() (postgres.Post, error) {
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return postgres.Post{}, err
	}
	authorID, err := uuid.Parse(p.AuthorID)
	if err != nil {
		return postgres.Post{}, err
	}
	return postgres.Post{ID: id, AuthorID: authorID, Text: p.Text}, nil
}

// Rebuild atomically replaces the complete cached timeline. The sorted-set
// members contain the response payload, so feed reads never query PostgreSQL.
func (c *FeedCache) Rebuild(ctx context.Context, user uuid.UUID, posts []postgres.Post) error {
	if len(posts) > MaxFeedItems {
		posts = posts[:MaxFeedItems]
	}
	key := feedKey(user)
	_, err := c.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, key)
		if len(posts) > 0 {
			entries := make([]redis.Z, 0, len(posts))
			for _, post := range posts {
				encoded, err := json.Marshal(toCachedPost(post))
				if err != nil {
					return err
				}
				// Redis scores are IEEE-754 doubles. A Unix microsecond timestamp
				// is exact at present and matches PostgreSQL's timestamp precision.
				entries = append(entries, redis.Z{Score: float64(post.CreatedAt.UnixMicro()), Member: string(encoded)})
			}
			pipe.ZAdd(ctx, key, entries...)
		}
		pipe.Set(ctx, feedReadyKey(user), "1", 0)
		return nil
	})
	return err
}

func (c *FeedCache) Get(ctx context.Context, user uuid.UUID, offset, limit int) ([]postgres.Post, error) {
	ready, err := c.client.Exists(ctx, feedReadyKey(user)).Result()
	if err != nil {
		return nil, err
	}
	if ready == 0 {
		return nil, ErrFeedNotReady
	}
	values, err := c.client.ZRevRange(ctx, feedKey(user), int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, err
	}
	posts := make([]postgres.Post, 0, len(values))
	for _, value := range values {
		var cached cachedPost
		if err := json.Unmarshal([]byte(value), &cached); err != nil {
			return nil, fmt.Errorf("decode cached feed: %w", err)
		}
		post, err := cached.toPost()
		if err != nil {
			return nil, fmt.Errorf("decode cached post identifiers: %w", err)
		}
		posts = append(posts, post)
	}
	return posts, nil
}
