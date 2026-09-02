package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/anki"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// openReviewFixture returns an in-memory Storage holding one deck with one
// new card, and that card's id. The tests reach the raw connection to
// sabotage or corrupt rows the public contract cannot.
func openReviewFixture(t *testing.T) (*Storage, int64) {
	t.Helper()
	ctx := context.Background()
	st, err := Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	deckID, err := st.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	p := domain.InitializePosition()
	posID, err := st.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	if err := st.Anki().SyncWithPositions(ctx, "", deckID, []int64{posID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}
	next, err := st.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard: %v", err)
	}
	return st, next.Card.ID
}

// cardRow reads the scheduling columns and the log count straight from the
// table, bypassing every COALESCE the store applies.
func cardRow(t *testing.T, st *Storage, cardID int64) (state, reps, logs int, due, lastReview string) {
	t.Helper()
	if err := st.sqlDB.QueryRow(
		`SELECT state, reps, due, COALESCE(last_review, '') FROM anki_card WHERE id = ?`, cardID).
		Scan(&state, &reps, &due, &lastReview); err != nil {
		t.Fatalf("read card: %v", err)
	}
	if err := st.sqlDB.QueryRow(
		`SELECT COUNT(*) FROM anki_review_log WHERE card_id = ?`, cardID).Scan(&logs); err != nil {
		t.Fatalf("count log: %v", err)
	}
	return state, reps, logs, due, lastReview
}

// TestAnkiReview_LogFailureLeavesCardUntouched: when the review-log INSERT
// fails, the card advance is rolled back with it — a grade is either fully
// recorded or not at all.
func TestAnkiReview_LogFailureLeavesCardUntouched(t *testing.T) {
	ctx := context.Background()
	st, cardID := openReviewFixture(t)
	_, _, _, dueBefore, _ := cardRow(t, st, cardID)

	if _, err := st.sqlDB.Exec(`CREATE TRIGGER anki_log_refuses BEFORE INSERT ON anki_review_log
		BEGIN SELECT RAISE(ABORT, 'journal indisponible'); END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	_, err := st.Anki().ReviewCard(ctx, "", cardID, 3)
	if err == nil {
		t.Fatal("ReviewCard succeeded although the log insert was refused")
	}

	state, reps, logs, due, lastReview := cardRow(t, st, cardID)
	if state != 0 || reps != 0 || due != dueBefore || lastReview != "" {
		t.Errorf("card advanced despite the failed log: state=%d reps=%d due=%q last_review=%q (due before %q)",
			state, reps, due, lastReview, dueBefore)
	}
	if logs != 0 {
		t.Errorf("review log rows: got %d, want 0", logs)
	}

	// The trigger gone, the same review goes through and both rows agree.
	if _, err := st.sqlDB.Exec(`DROP TRIGGER anki_log_refuses`); err != nil {
		t.Fatalf("drop trigger: %v", err)
	}
	if _, err := st.Anki().ReviewCard(ctx, "", cardID, 3); err != nil {
		t.Fatalf("ReviewCard after repair: %v", err)
	}
	state, reps, logs, _, _ = cardRow(t, st, cardID)
	if state == 0 || reps != 1 || logs != 1 {
		t.Errorf("after a sound review: state=%d reps=%d logs=%d", state, reps, logs)
	}
}

// TestAnkiReview_UnreadableTimestampIsRefused: a card whose due or
// last_review cannot be read is reported, not silently scheduled from a
// zero time — and it is left as it was.
func TestAnkiReview_UnreadableTimestampIsRefused(t *testing.T) {
	for _, tc := range []struct{ name, column string }{
		{"due", "due"},
		{"last review", "last_review"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			st, cardID := openReviewFixture(t)
			if _, err := st.sqlDB.Exec(
				`UPDATE anki_card SET `+tc.column+` = 'not a date' WHERE id = ?`, cardID); err != nil {
				t.Fatalf("corrupt %s: %v", tc.column, err)
			}

			_, err := st.Anki().ReviewCard(ctx, "", cardID, 3)
			if !errors.Is(err, anki.ErrUnreadableTimestamp) {
				t.Fatalf("ReviewCard on a corrupt %s: got %v, want ErrUnreadableTimestamp", tc.column, err)
			}
			state, reps, logs, _, _ := cardRow(t, st, cardID)
			if state != 0 || reps != 0 || logs != 0 {
				t.Errorf("refused card was still written: state=%d reps=%d logs=%d", state, reps, logs)
			}
		})
	}
}

// TestAnkiReview_RatingOutOfRangeIsInvalid: a rating outside 1..4 is a
// caller error (storage.ErrInvalid), not a panic inside the scheduler.
func TestAnkiReview_RatingOutOfRangeIsInvalid(t *testing.T) {
	ctx := context.Background()
	st, cardID := openReviewFixture(t)
	for _, r := range []int{0, 5} {
		_, err := st.Anki().ReviewCard(ctx, "", cardID, r)
		if !errors.Is(err, storage.ErrInvalid) || !errors.Is(err, anki.ErrInvalidRating) {
			t.Errorf("rating %d: got %v, want storage.ErrInvalid wrapping anki.ErrInvalidRating", r, err)
		}
	}
	if _, _, logs, _, _ := cardRow(t, st, cardID); logs != 0 {
		t.Errorf("invalid ratings were logged: %d rows", logs)
	}
}
