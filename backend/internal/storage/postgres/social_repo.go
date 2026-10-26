package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type SocialRepo struct{ cluster *Cluster }

func NewSocialRepo(cluster *Cluster) *SocialRepo { return &SocialRepo{cluster: cluster} }

type Post struct {
	ID        uuid.UUID
	AuthorID  uuid.UUID
	Text      string
	CreatedAt time.Time
}

type FeedRebuild struct {
	UserID   uuid.UUID
	Revision int64
}

// Locking the author serializes friendship changes with post changes, including
// when no friendship row exists yet. Hash collisions only add contention.
func lockAuthor(ctx context.Context, tx *sql.Tx, author uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, author.String())
	return err
}

func (r *SocialRepo) SetFriend(ctx context.Context, user, friend uuid.UUID) error {
	return r.changeFriend(ctx, user, friend, true)
}

func (r *SocialRepo) DeleteFriend(ctx context.Context, user, friend uuid.UUID) error {
	return r.changeFriend(ctx, user, friend, false)
}

func (r *SocialRepo) changeFriend(ctx context.Context, user, friend uuid.UUID, add bool) error {
	tx, err := r.cluster.Master().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockAuthor(ctx, tx, friend); err != nil {
		return err
	}
	var result sql.Result
	if add {
		result, err = tx.ExecContext(ctx, `INSERT INTO friendships (user_id, friend_id)
			SELECT $1, id FROM users WHERE id = $2 ON CONFLICT DO NOTHING`, user, friend)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			var exists bool
			if err = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, friend).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return sql.ErrNoRows
			}
		}
	} else {
		result, err = tx.ExecContext(ctx, `DELETE FROM friendships WHERE user_id = $1 AND friend_id = $2`, user, friend)
		if err != nil {
			return err
		}
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count > 0 {
		if err = enqueueFeedRebuild(ctx, tx, user); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func enqueueFeedRebuild(ctx context.Context, tx *sql.Tx, user uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO feed_rebuild_queue (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO UPDATE
		SET revision = feed_rebuild_queue.revision + 1, enqueued_at = now()`, user)
	return err
}

func enqueueFollowerRebuilds(ctx context.Context, tx *sql.Tx, author uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO feed_rebuild_queue (user_id)
		SELECT user_id FROM friendships WHERE friend_id = $1
		ON CONFLICT (user_id) DO UPDATE
		SET revision = feed_rebuild_queue.revision + 1, enqueued_at = now()`, author)
	return err
}

func (r *SocialRepo) CreatePost(ctx context.Context, author uuid.UUID, text string) (uuid.UUID, error) {
	tx, err := r.cluster.Master().BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()
	if err = lockAuthor(ctx, tx, author); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	if err = tx.QueryRowContext(ctx, `INSERT INTO posts (author_id, text) VALUES ($1, $2) RETURNING id`, author, text).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	if err = enqueueFollowerRebuilds(ctx, tx, author); err != nil {
		return uuid.Nil, err
	}
	if err = tx.Commit(); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *SocialRepo) UpdatePost(ctx context.Context, author, id uuid.UUID, text string) error {
	return r.changePost(ctx, author, id, text, false)
}

func (r *SocialRepo) DeletePost(ctx context.Context, author, id uuid.UUID) error {
	return r.changePost(ctx, author, id, "", true)
}

func (r *SocialRepo) changePost(ctx context.Context, author, id uuid.UUID, text string, deletePost bool) error {
	tx, err := r.cluster.Master().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockAuthor(ctx, tx, author); err != nil {
		return err
	}
	var result sql.Result
	if deletePost {
		result, err = tx.ExecContext(ctx, `DELETE FROM posts WHERE id = $1 AND author_id = $2`, id, author)
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE posts SET text = $3 WHERE id = $1 AND author_id = $2`, id, author, text)
	}
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	if err = enqueueFollowerRebuilds(ctx, tx, author); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SocialRepo) GetPost(ctx context.Context, id uuid.UUID) (Post, error) {
	var p Post
	err := r.cluster.Master().QueryRowContext(ctx, `SELECT id, author_id, text, created_at FROM posts WHERE id = $1`, id).Scan(&p.ID, &p.AuthorID, &p.Text, &p.CreatedAt)
	return p, err
}

func (r *SocialRepo) FeedPosts(ctx context.Context, user uuid.UUID, limit int) ([]Post, error) {
	rows, err := r.cluster.Master().QueryContext(ctx, `SELECT p.id, p.author_id, p.text, p.created_at
		FROM friendships f
		JOIN posts p ON p.author_id = f.friend_id
		WHERE f.user_id = $1
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $2`, user, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.AuthorID, &post.Text, &post.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (r *SocialRepo) PendingFeedRebuild(ctx context.Context, user uuid.UUID) (int64, bool, error) {
	var revision int64
	err := r.cluster.Master().QueryRowContext(ctx, `SELECT revision FROM feed_rebuild_queue WHERE user_id = $1`, user).Scan(&revision)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return revision, true, nil
}

func (r *SocialRepo) PendingFeedRebuilds(ctx context.Context, limit int) ([]FeedRebuild, error) {
	rows, err := r.cluster.Master().QueryContext(ctx, `SELECT user_id, revision
		FROM feed_rebuild_queue
		ORDER BY enqueued_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rebuilds := make([]FeedRebuild, 0)
	for rows.Next() {
		var rebuild FeedRebuild
		if err := rows.Scan(&rebuild.UserID, &rebuild.Revision); err != nil {
			return nil, err
		}
		rebuilds = append(rebuilds, rebuild)
	}
	return rebuilds, rows.Err()
}

func (r *SocialRepo) AcknowledgeFeedRebuild(ctx context.Context, user uuid.UUID, revision int64) error {
	_, err := r.cluster.Master().ExecContext(ctx, `DELETE FROM feed_rebuild_queue
		WHERE user_id = $1 AND revision = $2`, user, revision)
	return err
}
