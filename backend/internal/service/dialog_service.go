package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/dto"
	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/google/uuid"
)

const (
	DefaultDialogPageLimit = 50
	MaxDialogPageLimit     = 100
)

var dialogIDNamespace = uuid.MustParse("5e43dbe0-c409-4e00-bf84-ef738fdb4b17")

type DialogService struct{ repo *postgres.DialogRepo }

func NewDialogService(repo *postgres.DialogRepo) *DialogService { return &DialogService{repo: repo} }

func DialogID(first, second uuid.UUID) uuid.UUID {
	if bytes.Compare(first[:], second[:]) > 0 {
		first, second = second, first
	}
	return uuid.NewSHA1(dialogIDNamespace, []byte(first.String()+":"+second.String()))
}

func (s *DialogService) Send(ctx context.Context, sender uuid.UUID, recipientRaw string, req dto.SendDialogMessageRequest) error {
	recipient, err := parseID(recipientRaw)
	if err != nil || recipient == sender || req.Text == nil || strings.TrimSpace(*req.Text) == "" {
		return ErrInvalidValue
	}
	exists, err := s.repo.UserExists(ctx, recipient)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return s.repo.Send(ctx, postgres.DialogMessage{
		ID:          id,
		DialogID:    DialogID(sender, recipient),
		SenderID:    sender,
		RecipientID: recipient,
		Text:        *req.Text,
	})
}

func (s *DialogService) List(ctx context.Context, user uuid.UUID, otherRaw, rawCursor string, limit int) (dto.DialogPageResponse, error) {
	other, err := parseID(otherRaw)
	if err != nil || other == user || limit < 1 || limit > MaxDialogPageLimit {
		return dto.DialogPageResponse{}, ErrInvalidValue
	}
	exists, err := s.repo.UserExists(ctx, other)
	if err != nil {
		return dto.DialogPageResponse{}, err
	}
	if !exists {
		return dto.DialogPageResponse{}, ErrNotFound
	}
	cursor, err := decodeDialogCursor(rawCursor)
	if err != nil {
		return dto.DialogPageResponse{}, ErrInvalidValue
	}
	messages, err := s.repo.List(ctx, DialogID(user, other), cursor, limit+1)
	if err != nil {
		return dto.DialogPageResponse{}, err
	}
	page := dto.DialogPageResponse{Items: make([]dto.DialogMessageResponse, 0, min(len(messages), limit))}
	if len(messages) > limit {
		messages = messages[:limit]
		encoded, err := encodeDialogCursor(messages[len(messages)-1])
		if err != nil {
			return dto.DialogPageResponse{}, err
		}
		page.NextCursor = &encoded
	}
	for _, message := range messages {
		page.Items = append(page.Items, dto.DialogMessageResponse{
			From: message.SenderID.String(),
			To:   message.RecipientID.String(),
			Text: message.Text,
		})
	}
	return page, nil
}

type dialogCursorPayload struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeDialogCursor(message postgres.DialogMessage) (string, error) {
	payload, err := json.Marshal(dialogCursorPayload{CreatedAt: message.CreatedAt, ID: message.ID.String()})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeDialogCursor(raw string) (*postgres.DialogCursor, error) {
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var decoded dialogCursorPayload
	if err = json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(decoded.ID)
	if err != nil || id == uuid.Nil || decoded.CreatedAt.IsZero() {
		return nil, ErrInvalidValue
	}
	return &postgres.DialogCursor{CreatedAt: decoded.CreatedAt, ID: id}, nil
}
