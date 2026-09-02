// Contract cases for spaced-repetition (Anki) scheduling.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testAnkiReviewUpdatesScheduling(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	deckID, err := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{posID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	next, err := s.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard: %v", err)
	}
	if next.Card.PositionID != posID {
		t.Errorf("NextCard position: got %d, want %d", next.Card.PositionID, posID)
	}
	if next.Card.State != 0 {
		t.Errorf("NextCard state: got %d, want 0 (new)", next.Card.State)
	}

	// Reviewing the only card with Easy schedules it into the future, so it
	// leaves the new state and no card remains due.
	following, err := s.Anki().ReviewCard(ctx, "", next.Card.ID, 4)
	if err != nil {
		t.Fatalf("ReviewCard: %v", err)
	}
	if following != nil {
		t.Errorf("ReviewCard next card: got %+v, want nil", following)
	}

	stats, err := s.Anki().DeckStats(ctx, "", deckID)
	if err != nil {
		t.Fatalf("DeckStats: %v", err)
	}
	if stats.NewCount != 0 {
		t.Errorf("NewCount after review: got %d, want 0", stats.NewCount)
	}
	if _, err := s.Anki().NextCard(ctx, "", deckID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("NextCard after review: got %v, want ErrNotFound", err)
	}

	// Resetting the deck returns every card to the new, due state.
	if err := s.Anki().ResetDeck(ctx, "", deckID); err != nil {
		t.Fatalf("ResetDeck: %v", err)
	}
	stats, _ = s.Anki().DeckStats(ctx, "", deckID)
	if stats.NewCount != 1 || stats.DueCount != 1 {
		t.Errorf("DeckStats after reset: %+v", stats)
	}
}

// testAnkiRandomCardIgnoresSchedule pins the cram contract: RandomCard serves
// any card of the deck regardless of schedule or availability, honours the
// exclusion while another card remains, falls back to the lone card of a
// single-card deck, and never touches the FSRS state.
func testAnkiRandomCardIgnoresSchedule(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	deckID, err := s.Anki().CreateDeck(ctx, "", "cram", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if _, err := s.Anki().RandomCard(ctx, "", deckID, 0); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("RandomCard on an empty deck: got %v, want ErrNotFound", err)
	}

	p1, p2 := checkerPos(), cubePos()
	id1, err := s.Positions().Save(ctx, "", &p1)
	if err != nil {
		t.Fatalf("Save position 1: %v", err)
	}
	id2, err := s.Positions().Save(ctx, "", &p2)
	if err != nil {
		t.Fatalf("Save position 2: %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{id1, id2}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	// Reviewing the first card with Easy schedules it into the future and
	// suspending the second removes it from the review queue: NextCard has
	// nothing to serve, but cram still draws from the whole deck.
	next, err := s.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard: %v", err)
	}
	if _, err := s.Anki().ReviewCard(ctx, "", next.Card.ID, 4); err != nil {
		t.Fatalf("ReviewCard: %v", err)
	}
	other, err := s.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard (second card): %v", err)
	}
	if err := s.Anki().SetCardSuspended(ctx, "", other.Card.ID, true); err != nil {
		t.Fatalf("SetCardSuspended: %v", err)
	}
	if _, err := s.Anki().NextCard(ctx, "", deckID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("NextCard with nothing due: got %v, want ErrNotFound", err)
	}
	before, err := s.Anki().DeckStats(ctx, "", deckID)
	if err != nil {
		t.Fatalf("DeckStats: %v", err)
	}

	seen := map[int64]bool{}
	for i := 0; i < 20; i++ {
		card, err := s.Anki().RandomCard(ctx, "", deckID, 0)
		if err != nil {
			t.Fatalf("RandomCard draw %d: %v", i, err)
		}
		if card.Card.PositionID != card.Position.ID {
			t.Errorf("RandomCard: card position %d, loaded position %d", card.Card.PositionID, card.Position.ID)
		}
		seen[card.Card.PositionID] = true
	}
	if !seen[id1] || !seen[id2] {
		t.Errorf("RandomCard over 20 draws served %v, want both %d (reviewed) and %d (suspended)", seen, id1, id2)
	}

	// The exclusion makes a two-card deck deterministic.
	for i := 0; i < 5; i++ {
		card, err := s.Anki().RandomCard(ctx, "", deckID, id1)
		if err != nil {
			t.Fatalf("RandomCard excluding %d: %v", id1, err)
		}
		if card.Card.PositionID != id2 {
			t.Errorf("RandomCard excluding %d: got position %d, want %d", id1, card.Card.PositionID, id2)
		}
	}

	after, err := s.Anki().DeckStats(ctx, "", deckID)
	if err != nil {
		t.Fatalf("DeckStats after cram: %v", err)
	}
	if *before != *after {
		t.Errorf("cram mutated the schedule: before=%+v after=%+v", *before, *after)
	}

	// A single-card deck still serves its lone card when it is the excluded one.
	soloID, err := s.Anki().CreateDeck(ctx, "", "solo", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck solo: %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", soloID, []int64{id1}); err != nil {
		t.Fatalf("SyncWithPositions solo: %v", err)
	}
	card, err := s.Anki().RandomCard(ctx, "", soloID, id1)
	if err != nil {
		t.Fatalf("RandomCard single-card fallback: %v", err)
	}
	if card.Card.PositionID != id1 {
		t.Errorf("RandomCard single-card fallback: got position %d, want %d", card.Card.PositionID, id1)
	}
}
