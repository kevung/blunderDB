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

// testAnkiReviewLogAgreesWithCard pins what a review writes: the log entry
// and the card come out of one scheduling and one transaction, so the log's
// state is the one the card was in before, its interval is the one the card
// now carries, and its timestamp is the card's last review.
func testAnkiReviewLogAgreesWithCard(t *testing.T, s storage.Storage) {
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
	cardID := next.Card.ID

	// Good on a new card keeps it in learning and due within minutes, so it
	// is served again; Again then records a second entry against it.
	for _, rating := range []int{3, 1} {
		if _, err := s.Anki().ReviewCard(ctx, "", cardID, rating); err != nil {
			t.Fatalf("ReviewCard %d: %v", rating, err)
		}
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
	latest, first := logs[0], logs[1]
	if first.State != 0 {
		t.Errorf("first entry state: got %d, want 0 (new, the state before the review)", first.State)
	}
	if latest.State != 1 {
		t.Errorf("second entry state: got %d, want 1 (learning, the state before the review)", latest.State)
	}

	card, err := s.Anki().RandomCard(ctx, "", deckID, 0)
	if err != nil {
		t.Fatalf("RandomCard: %v", err)
	}
	if card.Card.ID != cardID {
		t.Fatalf("RandomCard: got card %d, want %d", card.Card.ID, cardID)
	}
	if card.Card.Reps != 2 || card.Card.State != 1 {
		t.Errorf("card after two reviews: reps=%d state=%d, want 2/1", card.Card.Reps, card.Card.State)
	}
	if latest.ScheduledDays != card.Card.ScheduledDays {
		t.Errorf("latest log interval %d != card interval %d", latest.ScheduledDays, card.Card.ScheduledDays)
	}
	if latest.Stability != card.Card.Stability || latest.Difficulty != card.Card.Difficulty {
		t.Errorf("latest log memory (%v, %v) != card memory (%v, %v)",
			latest.Stability, latest.Difficulty, card.Card.Stability, card.Card.Difficulty)
	}
	if latest.ReviewedAt == "" || latest.ReviewedAt != card.Card.LastReview {
		t.Errorf("latest log reviewed at %q != card last review %q", latest.ReviewedAt, card.Card.LastReview)
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

// testAnkiDrawOrderInterleavesTies pins ADR-0026 rule 9: cards tied on due date
// are drawn at random, while the priority that IS meaningful — state first,
// then an older due date — is untouched.
//
// The tie is not a corner case, it is the normal state of a fresh deck: every
// new card is synced with the same due timestamp, so ordering by due date
// separates none of them and the engine falls back on insertion order, which is
// the order of the match the positions came from. A session then walked the
// moves of one game in sequence.
func testAnkiDrawOrderInterleavesTies(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	deckID, err := s.Anki().CreateDeck(ctx, "", "order", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}

	// Eight positions, hence eight new cards sharing one due timestamp.
	ids := make([]int64, 0, 8)
	for i := 0; i < 8; i++ {
		p := checkerPos()
		p.Board.Points[i+1] = domain.Point{Checkers: 1, Color: domain.White}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, ids); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	// NextCard reads without consuming, so repeated draws expose the order.
	// With ties broken at random the first card varies; with insertion order it
	// would be the same row every time. Eight candidates make a false failure
	// (8/8 identical draws by chance) about 1 in 8^29.
	seen := map[int64]bool{}
	for i := 0; i < 30; i++ {
		card, err := s.Anki().NextCard(ctx, "", deckID)
		if err != nil {
			t.Fatalf("NextCard draw %d: %v", i, err)
		}
		seen[card.Card.PositionID] = true
	}
	if len(seen) < 2 {
		t.Errorf("NextCard served the same card on all 30 draws (%v): ties are not interleaved", seen)
	}

	// The priority that is NOT random stays covered where it already was: a card
	// graded away from "due" stops being served at all (see
	// testAnkiReviewUpdatesScheduling). It cannot be re-checked here by grading
	// one card Again, because a learning step pushes its due date minutes into
	// the future — it leaves the queue rather than moving to the head of it.
	if _, err := s.Anki().NextCard(ctx, "", deckID); err != nil {
		t.Fatalf("NextCard still serves the deck: %v", err)
	}
}

// testAnkiCubePairsAreChained pins the two halves of #276.
//
// A cube decision is TWO questions, and blunderDB already stores them as two
// positions. What the contract promises is that a deck built from one half
// gets the other, and that reviewing the first offers the second — without
// either becoming a card that takes one grade for two answers.
func testAnkiCubePairsAreChained(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	// The double and the take: two positions, one decision — two move rows of
	// the same game at the same move number, which is exactly what pairs them.
	double := statsDecisionPos(t, 0)
	doubleID, err := s.Positions().Save(ctx, "", &double)
	if err != nil {
		t.Fatalf("Save double position: %v", err)
	}
	take := statsDecisionPos(t, 1)
	takeID, err := s.Positions().Save(ctx, "", &take)
	if err != nil {
		t.Fatalf("Save take position: %v", err)
	}
	for _, mv := range []domain.Move{
		{GameID: gameID, MoveNumber: 4, MoveType: "cube", PositionID: doubleID, Player: 1, CubeAction: "Double"},
		{GameID: gameID, MoveNumber: 4, MoveType: "cube", PositionID: takeID, Player: -1, CubeAction: "Take"},
	} {
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}

	deckID, err := s.Anki().CreateDeck(ctx, "", "videau", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	// Only the DOUBLE half is asked for: the sync must complete the decision.
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{doubleID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	var inDeck []int64
	for p, err := range s.Anki().DeckPositions(ctx, "", deckID) {
		if err != nil {
			t.Fatalf("DeckPositions: %v", err)
		}
		inDeck = append(inDeck, p.ID)
	}
	if len(inDeck) != 2 {
		t.Fatalf("a cube decision is two questions: deck holds %d position(s) %v", len(inDeck), inDeck)
	}

	first, err := s.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard: %v", err)
	}
	linked, err := s.Anki().LinkedCard(ctx, "", deckID, first.Card.ID)
	if err != nil {
		t.Fatalf("LinkedCard: %v", err)
	}
	if linked.Card.PositionID == first.Card.PositionID {
		t.Error("the linked card must be the OTHER half, not the same one")
	}
	if linked.Card.ID == first.Card.ID {
		t.Error("the linked card must be a different card: two questions, two grades")
	}

	// A checker decision has no other half, and saying so is not an error
	// condition the caller has to guess at. A deck of its own, so the card
	// NextCard returns is unambiguously the one meant.
	lonely := checkerPos()
	lonelyID, err := s.Positions().Save(ctx, "", &lonely)
	if err != nil {
		t.Fatalf("Save lonely position: %v", err)
	}
	soloDeck, err := s.Anki().CreateDeck(ctx, "", "pions", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck(solo): %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", soloDeck, []int64{lonelyID}); err != nil {
		t.Fatalf("SyncWithPositions(lonely): %v", err)
	}
	solo, err := s.Anki().NextCard(ctx, "", soloDeck)
	if err != nil {
		t.Fatalf("NextCard(solo): %v", err)
	}
	if _, err := s.Anki().LinkedCard(ctx, "", soloDeck, solo.Card.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("a checker decision has no other half: got %v, want ErrNotFound", err)
	}
}
