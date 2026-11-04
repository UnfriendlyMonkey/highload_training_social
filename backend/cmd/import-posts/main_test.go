package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestFriendshipSourceSkipsSelfAndProducesRing(t *testing.T) {
	users := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	source := newFriendshipSource(users, 2)
	var got [][2]uuid.UUID
	for source.Next() {
		values, err := source.Values()
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, [2]uuid.UUID{values[0].(uuid.UUID), values[1].(uuid.UUID)})
	}
	if len(got) != 6 {
		t.Fatalf("friendships = %d, want 6", len(got))
	}
	for _, pair := range got {
		if pair[0] == pair[1] {
			t.Fatalf("self friendship generated: %v", pair)
		}
	}
	if got[0] != [2]uuid.UUID{users[0], users[1]} || got[1] != [2]uuid.UUID{users[0], users[2]} {
		t.Fatalf("first friendships = %v, want ring successors", got[:2])
	}
}

func TestReadPostTexts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "posts.txt")
	if err := os.WriteFile(path, []byte(" first \n\nsecond\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readPostTexts(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("texts = %#v", got)
	}
}
