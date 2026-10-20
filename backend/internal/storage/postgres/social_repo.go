package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type SocialRepo struct{ cluster *Cluster }

func NewSocialRepo(cluster *Cluster) *SocialRepo { return &SocialRepo{cluster: cluster} }

type Post struct {
	ID       uuid.UUID
	AuthorID uuid.UUID
	Text     string
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
		if _, err = tx.ExecContext(ctx, `INSERT INTO feed_revisions (user_id, revision) VALUES ($1, 1)
			ON CONFLICT (user_id) DO UPDATE SET revision = feed_revisions.revision + 1`, user); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Sort user IDs to acquire revision row locks in a consistent order across authors.
func bumpFollowers(ctx context.Context, tx *sql.Tx, author uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO feed_revisions (user_id, revision)
		SELECT user_id, 1 FROM friendships WHERE friend_id = $1 ORDER BY user_id
		ON CONFLICT (user_id) DO UPDATE SET revision = feed_revisions.revision + 1`, author)
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
	if err = bumpFollowers(ctx, tx, author); err != nil {
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
	if err = bumpFollowers(ctx, tx, author); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SocialRepo) GetPost(ctx context.Context, id uuid.UUID) (Post, error) {
	var p Post
	err := r.cluster.Master().QueryRowContext(ctx, `SELECT id, author_id, text FROM posts WHERE id = $1`, id).Scan(&p.ID, &p.AuthorID, &p.Text)
	return p, err
}
