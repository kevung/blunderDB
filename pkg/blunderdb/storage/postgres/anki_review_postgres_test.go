//go:build postgres

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/anki"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// openReviewFixture returns a Storage holding one deck with one new card, a
// raw connection for sabotage, and the card's id.
func openReviewFixture(t *testing.T) (*pg.Storage, *pgx.Conn, int64) {
	t.Helper()
	ctx := context.Background()
	s, dsn := openMatchStore(t)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("pgx.Connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	deckID, err := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	posID := savePos(t, s, domain.CheckerAction)
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{posID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}
	next, err := s.Anki().NextCard(ctx, "", deckID)
	if err != nil {
		t.Fatalf("NextCard: %v", err)
	}
	return s, conn, next.Card.ID
}

func cardRow(t *testing.T, conn *pgx.Conn, cardID int64) (state, reps, logs int) {
	t.Helper()
	ctx := context.Background()
	if err := conn.QueryRow(ctx,
		`SELECT state, reps FROM anki_card WHERE id = $1`, cardID).Scan(&state, &reps); err != nil {
		t.Fatalf("read card: %v", err)
	}
	if err := conn.QueryRow(ctx,
		`SELECT COUNT(*) FROM anki_review_log WHERE card_id = $1`, cardID).Scan(&logs); err != nil {
		t.Fatalf("count log: %v", err)
	}
	return state, reps, logs
}

// TestAnkiReview_LogFailureLeavesCardUntouched: when the review-log INSERT
// fails, the card advance is rolled back with it.
func TestAnkiReview_LogFailureLeavesCardUntouched(t *testing.T) {
	ctx := context.Background()
	s, conn, cardID := openReviewFixture(t)

	if _, err := conn.Exec(ctx, `
		CREATE FUNCTION anki_log_refuses() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'journal indisponible'; END $$;
		CREATE TRIGGER anki_log_refuses BEFORE INSERT ON anki_review_log
		FOR EACH ROW EXECUTE FUNCTION anki_log_refuses()`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	if _, err := s.Anki().ReviewCard(ctx, "", cardID, 3); err == nil {
		t.Fatal("ReviewCard succeeded although the log insert was refused")
	}
	if state, reps, logs := cardRow(t, conn, cardID); state != 0 || reps != 0 || logs != 0 {
		t.Errorf("card advanced despite the failed log: state=%d reps=%d logs=%d", state, reps, logs)
	}

	if _, err := conn.Exec(ctx, `DROP TRIGGER anki_log_refuses ON anki_review_log`); err != nil {
		t.Fatalf("drop trigger: %v", err)
	}
	if _, err := s.Anki().ReviewCard(ctx, "", cardID, 3); err != nil {
		t.Fatalf("ReviewCard after repair: %v", err)
	}
	if state, reps, logs := cardRow(t, conn, cardID); state == 0 || reps != 1 || logs != 1 {
		t.Errorf("after a sound review: state=%d reps=%d logs=%d", state, reps, logs)
	}
}

// TestAnkiReview_RatingOutOfRangeIsInvalid: a rating outside 1..4 is a
// caller error (storage.ErrInvalid), not a panic inside the scheduler.
func TestAnkiReview_RatingOutOfRangeIsInvalid(t *testing.T) {
	ctx := context.Background()
	s, conn, cardID := openReviewFixture(t)
	for _, r := range []int{0, 5} {
		_, err := s.Anki().ReviewCard(ctx, "", cardID, r)
		if !errors.Is(err, storage.ErrInvalid) || !errors.Is(err, anki.ErrInvalidRating) {
			t.Errorf("rating %d: got %v, want storage.ErrInvalid wrapping anki.ErrInvalidRating", r, err)
		}
	}
	if _, _, logs := cardRow(t, conn, cardID); logs != 0 {
		t.Errorf("invalid ratings were logged: %d rows", logs)
	}
}
