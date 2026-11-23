package service

import (
	"testing"
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
	"github.com/google/uuid"
)

func TestDialogIDIsIndependentOfParticipantOrder(t *testing.T) {
	first := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	second := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	if DialogID(first, second) != DialogID(second, first) {
		t.Fatal("dialog id depends on participant order")
	}
}

func TestDialogCursorRoundTrip(t *testing.T) {
	message := postgres.DialogMessage{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		CreatedAt: time.Date(2026, time.November, 11, 12, 0, 0, 123, time.UTC),
	}
	raw, err := encodeDialogCursor(message)
	if err != nil {
		t.Fatal(err)
	}
	cursor, err := decodeDialogCursor(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.ID != message.ID || !cursor.CreatedAt.Equal(message.CreatedAt) {
		t.Fatalf("unexpected cursor: %#v", cursor)
	}
}
