// import-posts prepares a reproducible social graph and posts for local load
// tests. It writes directly to PostgreSQL and queues cache rebuilds once, after
// the complete batch is committed.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const copyProgressEvery = 100_000

func main() {
	filePath := flag.String("file", "", "optional text file: one post per line")
	postCount := flag.Int("posts", 100_000, "number of posts to create")
	friendsPerUser := flag.Int("friends-per-user", 10, "outgoing friends created for every user")
	days := flag.Int("days", 30, "spread post timestamps across this many days")
	seed := flag.Uint64("seed", uint64(time.Now().UnixNano()), "random seed; print it to repeat a dataset")
	reset := flag.Bool("reset", false, "delete existing posts, friendships and queued rebuilds before import")
	flag.Parse()

	if *postCount < 1 || *friendsPerUser < 0 || *days < 1 {
		log.Fatal("posts must be positive; friends-per-user must be non-negative; days must be positive")
	}
	texts, err := readPostTexts(*filePath)
	if err != nil {
		log.Fatalf("read post texts: %v", err)
	}

	ctx := context.Background()
	cfg := config.Load()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connect: %v", err)
	}
	defer conn.Close(ctx)

	users, err := loadUserIDs(ctx, conn)
	if err != nil {
		log.Fatalf("load users: %v", err)
	}
	if len(users) == 0 {
		log.Fatal("no users found; run import-users or register users first")
	}

	start := time.Now()
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	if *reset {
		if _, err = tx.Exec(ctx, `TRUNCATE TABLE feed_rebuild_queue, posts, friendships`); err != nil {
			log.Fatalf("reset social data: %v", err)
		}
	}

	friends := min(*friendsPerUser, len(users)-1)
	friendCount, err := importFriendships(ctx, tx, users, friends)
	if err != nil {
		log.Fatalf("import friendships: %v", err)
	}

	rng := rand.New(rand.NewPCG(*seed, *seed^0x9e3779b97f4a7c15))
	posts := newPostSource(users, texts, *postCount, *days, rng)
	postRows, err := tx.CopyFrom(ctx, pgx.Identifier{"posts"}, []string{"author_id", "text", "created_at"}, posts)
	if err != nil {
		log.Fatalf("import posts: %v (imported %d rows before failure)", err, posts.imported)
	}

	// Direct bulk import intentionally bypasses SocialRepo. Queue each existing
	// feed once rather than enqueueing it for every individual post.
	queueResult, err := tx.Exec(ctx, `INSERT INTO feed_rebuild_queue (user_id)
		SELECT DISTINCT user_id FROM friendships
		ON CONFLICT (user_id) DO UPDATE
		SET revision = feed_rebuild_queue.revision + 1, enqueued_at = now()`)
	if err != nil {
		log.Fatalf("queue feed rebuilds: %v", err)
	}
	if err = tx.Commit(ctx); err != nil {
		log.Fatalf("commit import: %v", err)
	}

	log.Printf("imported %d posts, added %d friendships, queued %d feed rebuilds in %s (seed=%d)",
		postRows, friendCount, queueResult.RowsAffected(), time.Since(start).Round(time.Millisecond), *seed)
}

func loadUserIDs(ctx context.Context, conn *pgx.Conn) ([]uuid.UUID, error) {
	rows, err := conn.Query(ctx, `SELECT id FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		users = append(users, id)
	}
	return users, rows.Err()
}

func readPostTexts(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	texts := make([]string, 0)
	for scanner.Scan() {
		if text := strings.TrimSpace(scanner.Text()); text != "" {
			texts = append(texts, text)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("%q contains no non-empty post texts", path)
	}
	return texts, nil
}

func importFriendships(ctx context.Context, tx pgx.Tx, users []uuid.UUID, friendsPerUser int) (int64, error) {
	if friendsPerUser == 0 {
		return 0, nil
	}
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE import_friendships (
		user_id UUID NOT NULL,
		friend_id UUID NOT NULL,
		PRIMARY KEY (user_id, friend_id)
	) ON COMMIT DROP`); err != nil {
		return 0, err
	}
	source := newFriendshipSource(users, friendsPerUser)
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"import_friendships"}, []string{"user_id", "friend_id"}, source); err != nil {
		return 0, err
	}
	result, err := tx.Exec(ctx, `INSERT INTO friendships (user_id, friend_id)
		SELECT user_id, friend_id FROM import_friendships
		ON CONFLICT DO NOTHING`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

type friendshipSource struct {
	users          []uuid.UUID
	friendsPerUser int
	userIndex      int
	friendOffset   int
}

func newFriendshipSource(users []uuid.UUID, friendsPerUser int) *friendshipSource {
	return &friendshipSource{users: users, friendsPerUser: friendsPerUser}
}

func (s *friendshipSource) Next() bool {
	if s.userIndex >= len(s.users) {
		return false
	}
	s.friendOffset++
	if s.friendOffset > s.friendsPerUser {
		s.userIndex++
		s.friendOffset = 1
	}
	return s.userIndex < len(s.users)
}

func (s *friendshipSource) Values() ([]any, error) {
	friendIndex := (s.userIndex + s.friendOffset) % len(s.users)
	return []any{s.users[s.userIndex], s.users[friendIndex]}, nil
}

func (s *friendshipSource) Err() error { return nil }

type postSource struct {
	users    []uuid.UUID
	texts    []string
	count    int
	days     int
	rng      *rand.Rand
	imported int64
}

func newPostSource(users []uuid.UUID, texts []string, count, days int, rng *rand.Rand) *postSource {
	return &postSource{users: users, texts: texts, count: count, days: days, rng: rng}
}

func (s *postSource) Next() bool {
	if s.imported >= int64(s.count) {
		return false
	}
	s.imported++
	if s.imported%copyProgressEvery == 0 {
		log.Printf("prepared %d posts...", s.imported)
	}
	return true
}

func (s *postSource) Values() ([]any, error) {
	var text string
	if len(s.texts) == 0 {
		text = fmt.Sprintf("Generated post %d", s.imported)
	} else {
		text = s.texts[s.rng.IntN(len(s.texts))]
	}
	window := int64(time.Duration(s.days) * 24 * time.Hour)
	createdAt := time.Now().UTC().Add(-time.Duration(s.rng.Int64N(window)))
	return []any{s.users[s.rng.IntN(len(s.users))], text, createdAt}, nil
}

func (s *postSource) Err() error { return nil }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
