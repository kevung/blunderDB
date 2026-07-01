package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestAnkiDeckCRUD covers CreateDeck, ListDecks, UpdateDeck, UpdateDeckParams
// and DeleteDeck.
func TestAnkiDeckCRUD(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

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
// into the deck as new cards.
func TestAnkiSyncFromCollection(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

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
	s := openMem(t)
	if err := s.Anki().Sync(ctx, "", 999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Sync unknown deck: got %v, want ErrNotFound", err)
	}
}

// TestAnkiReviewUpdatesScheduling checks that NextCard, ReviewCard and
// ResetDeck move a card through its FSRS lifecycle.
func TestAnkiReviewUpdatesScheduling(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	deckID, _ := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	pos := savePos(t, s, domain.CheckerAction)
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{pos}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	next, err := s.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard: %v", err)
	}
	if next.Card.PositionID != pos {
		t.Errorf("NextCard position: got %d, want %d", next.Card.PositionID, pos)
	}
	if next.Card.State != 0 {
		t.Errorf("NextCard state: got %d, want 0 (new)", next.Card.State)
	}

	// Reviewing the only card with Easy schedules it into the future, so it
	// leaves the new state and is no longer due.
	following, err := s.Anki().ReviewCard(ctx, "", next.Card.ID, 4)
	if err != nil {
		t.Fatalf("ReviewCard: %v", err)
	}
	if following != nil {
		t.Errorf("ReviewCard next card: got %+v, want nil", following)
	}

	stats, _ := s.Anki().DeckStats(ctx, "", deckID)
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

// TestAnkiReviewMissingCard verifies ReviewCard reports ErrNotFound for an
// unknown card.
func TestAnkiReviewMissingCard(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)
	if _, err := s.Anki().ReviewCard(ctx, "", 999, 3); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("ReviewCard unknown card: got %v, want ErrNotFound", err)
	}
}

// TestAnkiOptimizeParams covers the retention tuner plumbing (ANK-E2/B10):
// unknown deck → ErrNotFound; a deck with no review-state history leaves the
// request_retention unchanged (not enough signal).
func TestAnkiOptimizeParams(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	if _, err := s.Anki().OptimizeParams(ctx, "", 999, false); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("OptimizeParams unknown deck: got %v, want ErrNotFound", err)
	}

	deckID, _ := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	pos := savePos(t, s, domain.CheckerAction)
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{pos}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}
	// Reviewing the only card logs it from the New state, so it is not a
	// review-state (state=2) sample: the tuner sees no signal.
	next, _ := s.Anki().NextCard(ctx, "", deckID)
	if next != nil {
		s.Anki().ReviewCard(ctx, "", next.Card.ID, 3)
	}

	res, err := s.Anki().OptimizeParams(ctx, "", deckID, true)
	if err != nil {
		t.Fatalf("OptimizeParams: %v", err)
	}
	if res.SampleSize != 0 {
		t.Errorf("expected 0 review-state samples, got %d", res.SampleSize)
	}
	if res.Applied {
		t.Errorf("nothing to optimize → should not apply")
	}
	if res.SuggestedRetention != res.CurrentRetention {
		t.Errorf("retention should be unchanged: %+v", res)
	}
}

// TestAnkiSuspendBuryRemove checks that suspending, burying and removing cards
// take them out of the review queue and that the operations report ErrNotFound
// for an unknown card.
func TestAnkiSuspendBuryRemove(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	deckID, _ := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	mk := func(d1, d2 int) int64 {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Dice = [2]int{d1, d2}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}
	posA, posB := mk(3, 1), mk(5, 2)
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{posA, posB}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	// Resolve the card backing posA. NextCard surfaces one of the two; if it is
	// not posA, suspend it to expose the other, then restore the clean state.
	next, _ := s.Anki().NextCard(ctx, "", deckID)
	cardA := next.Card.ID
	if next.Card.PositionID != posA {
		if err := s.Anki().SetCardSuspended(ctx, "", next.Card.ID, true); err != nil {
			t.Fatalf("SetCardSuspended: %v", err)
		}
		other, _ := s.Anki().NextCard(ctx, "", deckID)
		cardA = other.Card.ID
		if err := s.Anki().SetCardSuspended(ctx, "", next.Card.ID, false); err != nil {
			t.Fatalf("unsuspend: %v", err)
		}
	}

	// Baseline: both cards due.
	if st, _ := s.Anki().DeckStats(ctx, "", deckID); st.DueCount != 2 || st.TotalCount != 2 {
		t.Fatalf("baseline stats: %+v", st)
	}

	// Suspend cardA → it leaves the queue but stays in the deck.
	if err := s.Anki().SetCardSuspended(ctx, "", cardA, true); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	st, _ := s.Anki().DeckStats(ctx, "", deckID)
	if st.DueCount != 1 || st.NewCount != 1 || st.TotalCount != 2 {
		t.Errorf("stats after suspend: %+v", st)
	}
	fc, _ := s.Anki().Forecast(ctx, "", deckID, 7)
	if fc[0].Due != 1 {
		t.Errorf("forecast day0 after suspend: got %d, want 1", fc[0].Due)
	}

	// Unsuspend → back to two due.
	if err := s.Anki().SetCardSuspended(ctx, "", cardA, false); err != nil {
		t.Fatalf("unsuspend: %v", err)
	}
	if st, _ := s.Anki().DeckStats(ctx, "", deckID); st.DueCount != 2 {
		t.Errorf("DueCount after unsuspend: got %d, want 2", st.DueCount)
	}

	// Bury cardA → hidden until tomorrow, so only one card is due now.
	if err := s.Anki().BuryCard(ctx, "", cardA); err != nil {
		t.Fatalf("bury: %v", err)
	}
	if st, _ := s.Anki().DeckStats(ctx, "", deckID); st.DueCount != 1 {
		t.Errorf("DueCount after bury: got %d, want 1", st.DueCount)
	}

	// Remove cardA → deck shrinks to one card.
	if err := s.Anki().RemoveCard(ctx, "", cardA); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if st, _ := s.Anki().DeckStats(ctx, "", deckID); st.TotalCount != 1 {
		t.Errorf("TotalCount after remove: got %d, want 1", st.TotalCount)
	}

	// Unknown card → ErrNotFound for each op.
	if err := s.Anki().SetCardSuspended(ctx, "", 9999, true); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("suspend unknown: got %v, want ErrNotFound", err)
	}
	if err := s.Anki().BuryCard(ctx, "", 9999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("bury unknown: got %v, want ErrNotFound", err)
	}
	if err := s.Anki().RemoveCard(ctx, "", 9999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("remove unknown: got %v, want ErrNotFound", err)
	}
}

// TestAnkiForecast checks the due-cards projection: new cards land on day 0, a
// reviewed card leaves day 0, the slice is dense (one entry per day) and the
// deck filter and default horizon behave.
func TestAnkiForecast(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	deckID, _ := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	// Three distinct positions (zobrist includes the dice, so vary them).
	mk := func(d1, d2 int) int64 {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Dice = [2]int{d1, d2}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}
	posIDs := []int64{mk(3, 1), mk(5, 2), mk(6, 4)}
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, posIDs); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	fc, err := s.Anki().Forecast(ctx, "", deckID, 7)
	if err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	if len(fc) != 7 {
		t.Fatalf("Forecast length: got %d, want 7", len(fc))
	}
	if fc[0].Due != 3 {
		t.Errorf("Forecast day 0: got %d due, want 3 (all new)", fc[0].Due)
	}
	if fc[0].Day != time.Now().UTC().Format("2006-01-02") {
		t.Errorf("Forecast day 0 date: got %q, want today", fc[0].Day)
	}

	// Reviewing one card with Easy schedules it ahead, so it leaves day 0.
	next, _ := s.Anki().NextCard(ctx, "", deckID)
	if _, err := s.Anki().ReviewCard(ctx, "", next.Card.ID, 4); err != nil {
		t.Fatalf("ReviewCard: %v", err)
	}
	fc, _ = s.Anki().Forecast(ctx, "", deckID, 365)
	if fc[0].Due != 2 {
		t.Errorf("Forecast day 0 after review: got %d, want 2", fc[0].Due)
	}
	total := 0
	for _, d := range fc {
		total += d.Due
	}
	if total != 3 {
		t.Errorf("Forecast total over a year: got %d, want 3", total)
	}

	// A non-positive horizon falls back to the 30-day default.
	fc, _ = s.Anki().Forecast(ctx, "", deckID, 0)
	if len(fc) != 30 {
		t.Errorf("Forecast default horizon: got %d, want 30", len(fc))
	}

	// deckID 0 spans every deck.
	fc, _ = s.Anki().Forecast(ctx, "", 0, 7)
	if fc[0].Due != 2 {
		t.Errorf("Forecast all decks day 0: got %d, want 2", fc[0].Due)
	}
}

// TestAnkiReviewLog checks that each ReviewCard appends a review_log row that
// records the rating, the card identity and the FSRS outcome, newest first.
func TestAnkiReviewLog(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	deckID, _ := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	pos := savePos(t, s, domain.CheckerAction)
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{pos}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}
	next, err := s.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard: %v", err)
	}
	cardID := next.Card.ID

	// Empty log before any review.
	for range s.Anki().ReviewLog(ctx, "", deckID, 0) {
		t.Fatal("ReviewLog should be empty before any review")
	}

	// Two reviews: Good then Again.
	if _, err := s.Anki().ReviewCard(ctx, "", cardID, 3); err != nil {
		t.Fatalf("ReviewCard Good: %v", err)
	}
	if _, err := s.Anki().ReviewCard(ctx, "", cardID, 1); err != nil {
		t.Fatalf("ReviewCard Again: %v", err)
	}

	var logs []*domain.AnkiReviewLog
	for l, err := range s.Anki().ReviewLog(ctx, "", deckID, 0) {
		if err != nil {
			t.Fatalf("ReviewLog: %v", err)
		}
		logs = append(logs, l)
	}
	if len(logs) != 2 {
		t.Fatalf("ReviewLog count: got %d, want 2", len(logs))
	}
	// Newest first: the most recent rating (Again=1) leads.
	if logs[0].Rating != 1 || logs[1].Rating != 3 {
		t.Errorf("ReviewLog order: got ratings [%d, %d], want [1, 3]", logs[0].Rating, logs[1].Rating)
	}
	for _, l := range logs {
		if l.CardID != cardID || l.DeckID != deckID || l.PositionID != pos {
			t.Errorf("ReviewLog identity: %+v (card %d deck %d pos %d)", l, cardID, deckID, pos)
		}
		if l.ReviewedAt == "" {
			t.Errorf("ReviewLog missing reviewed_at: %+v", l)
		}
	}

	// The limit caps the number of returned rows.
	n := 0
	for range s.Anki().ReviewLog(ctx, "", deckID, 1) {
		n++
	}
	if n != 1 {
		t.Errorf("ReviewLog with limit 1: got %d rows, want 1", n)
	}

	// deckID 0 spans every deck.
	n = 0
	for range s.Anki().ReviewLog(ctx, "", 0, 0) {
		n++
	}
	if n != 2 {
		t.Errorf("ReviewLog all decks: got %d rows, want 2", n)
	}
}
