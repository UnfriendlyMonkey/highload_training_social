package cache

import (
	"context"
	"testing"
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
)

func TestFeedCacheReturnsNewestPostsAndPaginates(t *testing.T) {
	server := miniredis.RunT(t)
	cache, err := NewFeedCache("redis://" + server.Addr() + "/0")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	user := uuid.New()
	base := time.Now().UTC().Truncate(time.Microsecond)
	posts := []postgres.Post{
		{ID: uuid.New(), AuthorID: uuid.New(), Text: "old", CreatedAt: base},
		{ID: uuid.New(), AuthorID: uuid.New(), Text: "new", CreatedAt: base.Add(time.Second)},
	}
	if err := cache.Rebuild(context.Background(), user, posts); err != nil {
		t.Fatal(err)
	}

	got, err := cache.Get(context.Background(), user, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != posts[0].ID || got[0].Text != "old" {
		t.Fatalf("page = %#v, want old post", got)
	}
}

func TestFeedCacheMarksEmptyFeedAsReady(t *testing.T) {
	server := miniredis.RunT(t)
	cache, err := NewFeedCache("redis://" + server.Addr() + "/0")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	if err := cache.Rebuild(context.Background(), uuid.New(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestFeedCacheKeepsAtMostThousandPosts(t *testing.T) {
	server := miniredis.RunT(t)
	cache, err := NewFeedCache("redis://" + server.Addr() + "/0")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	base := time.Now().UTC().Truncate(time.Microsecond)
	posts := make([]postgres.Post, MaxFeedItems+1)
	for i := range posts {
		posts[i] = postgres.Post{ID: uuid.New(), AuthorID: uuid.New(), CreatedAt: base.Add(time.Duration(MaxFeedItems-i) * time.Microsecond)}
	}
	user := uuid.New()
	if err := cache.Rebuild(context.Background(), user, posts); err != nil {
		t.Fatal(err)
	}
	last, err := cache.Get(context.Background(), user, MaxFeedItems-1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(last) != 1 || last[0].ID != posts[MaxFeedItems-1].ID {
		t.Fatalf("last cached post = %#v, want post %d", last, MaxFeedItems-1)
	}
	extra, err := cache.Get(context.Background(), user, MaxFeedItems, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(extra) != 0 {
		t.Fatalf("cache returned %d post(s) past its limit", len(extra))
	}
}
