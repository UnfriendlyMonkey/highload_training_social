package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DialogRepo struct{ cluster *Cluster }

func NewDialogRepo(cluster *Cluster) *DialogRepo { return &DialogRepo{cluster: cluster} }

type DialogMessage struct {
	ID          uuid.UUID
	DialogID    uuid.UUID
	SenderID    uuid.UUID
	RecipientID uuid.UUID
	Text        string
	CreatedAt   time.Time
}

type DialogCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func (r *DialogRepo) UserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.cluster.Master().QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func (r *DialogRepo) Send(ctx context.Context, message DialogMessage) error {
	_, err := r.cluster.Master().ExecContext(ctx, `INSERT INTO dialog_messages
		(dialog_id, id, sender_id, recipient_id, text)
		VALUES ($1, $2, $3, $4, $5)`,
		message.DialogID, message.ID, message.SenderID, message.RecipientID, message.Text)
	return err
}

func (r *DialogRepo) List(ctx context.Context, dialogID uuid.UUID, cursor *DialogCursor, limit int) ([]DialogMessage, error) {
	var cursorTime any
	var cursorID uuid.UUID
	if cursor != nil {
		cursorTime = cursor.CreatedAt
		cursorID = cursor.ID
	}
	rows, err := r.cluster.Master().QueryContext(ctx, `SELECT id, dialog_id, sender_id, recipient_id, text, created_at
		FROM dialog_messages
		WHERE dialog_id = $1
			AND ($2::timestamptz IS NULL OR (created_at, id) > ($2::timestamptz, $3::uuid))
		ORDER BY created_at, id
		LIMIT $4`, dialogID, cursorTime, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]DialogMessage, 0, limit)
	for rows.Next() {
		var message DialogMessage
		if err := rows.Scan(&message.ID, &message.DialogID, &message.SenderID, &message.RecipientID, &message.Text, &message.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}
