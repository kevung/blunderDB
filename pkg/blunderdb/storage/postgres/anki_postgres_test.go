//go:build postgres

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestAnkiDeckCRUD covers CreateDeck, ListDecks, UpdateDeck, UpdateDeckParams
// and DeleteDeck.
func TestAnkiDeckCRUD(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	id, err := s.Anki().CreateDeck(ctx, "", "Backgame", "tricky spots",
		domain.AnkiSourceCollection, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if id == 0 {
		t.Fatal("CreateDeck returned id 0")
	}

	if err := s.Anki().UpdateDeck(ctx, "", id, "Backgames", "edited"); err != nil {
		t.Fatalf("UpdateDeck: %v", err)
	}
	if err := s.Anki().UpdateDeckParams(ctx, "", id, 0.85, 180, false); err != nil {
		t.Fatalf("UpdateDeckParams: %v", err)
	}

	var decks []*domain.AnkiDeck
	for d, err := range s.Anki().ListDecks(ctx, "") {
		if err != nil {
			t.Fatalf("ListDecks: %v", err)
		}
		decks = append(decks, d)
	}
	if len(decks) != 1 {
		t.Fatalf("ListDecks count: got %d, want 1", len(decks))
	}
	d := decks[0]
	if d.Name != "Backgames" || d.Description != "edited" {
		t.Errorf("deck header: %+v", d)
	}
	if d.RequestRetention != 0.85 || d.MaximumInterval != 180 || d.EnableFuzz {
		t.Errorf("deck params: %+v", d)
	}

	if err := s.Anki().DeleteDeck(ctx, "", id); err != nil {
		t.Fatalf("DeleteDeck: %v", err)
	}
	n := 0
	for _, err := range s.Anki().ListDecks(ctx, "") {
		if err != nil {
			t.Fatalf("ListDecks: %v", err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("decks after delete: got %d, want 0", n)
	}
}

// TestAnkiSyncFromCollection checks that Sync pulls a collection's positions
// into the deck as new cards and is idempotent.
func TestAnkiSyncFromCollection(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	cID, _ := s.Collections().Create(ctx, "", "src", "")
	p1 := savePos(t, s, domain.CheckerAction)
	p2 := savePos(t, s, domain.CubeAction)
	if err := s.Collections().AddPositions(ctx, "", cID, []int64{p1, p2}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}

	deckID, _ := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceCollection, cID, "")
	if err := s.Anki().Sync(ctx, "", deckID); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	stats, err := s.Anki().DeckStats(ctx, "", deckID)
	if err != nil {
		t.Fatalf("DeckStats: %v", err)
	}
	if stats.TotalCount != 2 || stats.NewCount != 2 || stats.DueCount != 2 {
		t.Errorf("DeckStats after sync: %+v", stats)
	}

	var posIDs []int64
	for p, err := range s.Anki().DeckPositions(ctx, "", deckID) {
		if err != nil {
			t.Fatalf("DeckPositions: %v", err)
		}
		posIDs = append(posIDs, p.ID)
	}
	if len(posIDs) != 2 {
		t.Errorf("DeckPositions count: got %d, want 2", len(posIDs))
	}

	// Sync is idempotent: re-running adds no duplicate cards.
	if err := s.Anki().Sync(ctx, "", deckID); err != nil {
		t.Fatalf("Sync again: %v", err)
	}
	stats, _ = s.Anki().DeckStats(ctx, "", deckID)
	if stats.TotalCount != 2 {
		t.Errorf("TotalCount after re-sync: got %d, want 2", stats.TotalCount)
	}
}

// TestAnkiSyncMissingDeck verifies Sync reports ErrNotFound for an unknown
// deck.
func TestAnkiSyncMissingDeck(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)
	if err := s.Anki().Sync(ctx, "", 999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Sync unknown deck: got %v, want ErrNotFound", err)
	}
}

// TestAnkiReviewMissingCard verifies ReviewCard reports ErrNotFound for an
// unknown card.
func TestAnkiReviewMissingCard(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)
	if _, err := s.Anki().ReviewCard(ctx, "", 999, 3); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("ReviewCard unknown card: got %v, want ErrNotFound", err)
	}
}
