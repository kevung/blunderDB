package anki_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/anki"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

const (
	stateNew        = 0
	stateLearning   = 1
	stateReview     = 2
	stateRelearning = 3

	again = 1
	good  = 3
	easy  = 4
)

var (
	now    = time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	params = anki.Params{RequestRetention: 0.9, MaximumInterval: 365, EnableFuzz: false}
)

func ts(t time.Time) string { return t.UTC().Format(anki.TimeLayout) }

// newCard is a card exactly as SyncWithPositions creates it: due now, never
// reviewed.
func newCard() domain.AnkiCard {
	return domain.AnkiCard{ID: 7, DeckID: 3, PositionID: 42, Due: ts(now), State: stateNew}
}

// reviewCard is a card in review state, last seen five days ago.
func reviewCard() domain.AnkiCard {
	return domain.AnkiCard{
		ID: 7, DeckID: 3, PositionID: 42,
		Due: ts(now.Add(-24 * time.Hour)), Stability: 5, Difficulty: 5,
		ElapsedDays: 4, ScheduledDays: 4, Reps: 3, Lapses: 0, State: stateReview,
		LastReview: ts(now.Add(-5 * 24 * time.Hour)),
	}
}

func TestScheduleNext_NewCardGood(t *testing.T) {
	next, log, err := anki.ScheduleNext(newCard(), params, good, now)
	if err != nil {
		t.Fatalf("ScheduleNext: %v", err)
	}
	if next.ID != 7 || next.DeckID != 3 || next.PositionID != 42 {
		t.Errorf("identity not preserved: %+v", next)
	}
	if next.State != stateLearning {
		t.Errorf("state: got %d, want learning (%d)", next.State, stateLearning)
	}
	if next.Reps != 1 || next.Lapses != 0 {
		t.Errorf("reps/lapses: got %d/%d, want 1/0", next.Reps, next.Lapses)
	}
	if next.Due != ts(now.Add(10*time.Minute)) {
		t.Errorf("due: got %q, want %q (now + 10 min)", next.Due, ts(now.Add(10*time.Minute)))
	}
	if next.LastReview != ts(now) {
		t.Errorf("last review: got %q, want %q", next.LastReview, ts(now))
	}
	if next.Stability <= 0 || next.Difficulty <= 0 {
		t.Errorf("stability/difficulty not initialised: %+v", next)
	}

	if log.CardID != 7 || log.DeckID != 3 || log.PositionID != 42 {
		t.Errorf("log identity: %+v", log)
	}
	if log.Rating != good || log.State != stateNew {
		t.Errorf("log rating/state: got %d/%d, want %d/%d (state before the review)", log.Rating, log.State, good, stateNew)
	}
	if log.ElapsedDays != 0 {
		t.Errorf("log elapsed days on a first review: got %d, want 0", log.ElapsedDays)
	}
	if log.Stability != next.Stability || log.Difficulty != next.Difficulty || log.ScheduledDays != next.ScheduledDays {
		t.Errorf("log does not carry the post-review values: log %+v card %+v", log, next)
	}
	if log.ReviewedAt != ts(now) {
		t.Errorf("reviewed at: got %q, want %q", log.ReviewedAt, ts(now))
	}
}

func TestScheduleNext_NewCardEasyGoesStraightToReview(t *testing.T) {
	next, _, err := anki.ScheduleNext(newCard(), params, easy, now)
	if err != nil {
		t.Fatalf("ScheduleNext: %v", err)
	}
	if next.State != stateReview {
		t.Errorf("state: got %d, want review (%d)", next.State, stateReview)
	}
	if next.ScheduledDays < 1 {
		t.Errorf("scheduled days: got %d, want >= 1", next.ScheduledDays)
	}
	due, err := anki.ParseTime(next.Due)
	if err != nil {
		t.Fatalf("ParseTime(due): %v", err)
	}
	if !due.After(now.Add(24 * time.Hour)) {
		t.Errorf("due %v is not at least a day after now", due)
	}
}

func TestScheduleNext_LearningCardGraduates(t *testing.T) {
	first, _, err := anki.ScheduleNext(newCard(), params, good, now)
	if err != nil {
		t.Fatalf("first review: %v", err)
	}
	later := now.Add(10 * time.Minute)
	next, log, err := anki.ScheduleNext(first, params, good, later)
	if err != nil {
		t.Fatalf("second review: %v", err)
	}
	if next.State != stateReview {
		t.Errorf("state: got %d, want review (%d)", next.State, stateReview)
	}
	if log.State != stateLearning {
		t.Errorf("log state: got %d, want learning (%d)", log.State, stateLearning)
	}
	if next.Reps != 2 {
		t.Errorf("reps: got %d, want 2", next.Reps)
	}
	if next.LastReview != ts(later) {
		t.Errorf("last review: got %q, want %q", next.LastReview, ts(later))
	}
}

func TestScheduleNext_ReviewCardGood(t *testing.T) {
	card := reviewCard()
	next, log, err := anki.ScheduleNext(card, params, good, now)
	if err != nil {
		t.Fatalf("ScheduleNext: %v", err)
	}
	if next.State != stateReview {
		t.Errorf("state: got %d, want review (%d)", next.State, stateReview)
	}
	if next.Reps != 4 || next.Lapses != 0 {
		t.Errorf("reps/lapses: got %d/%d, want 4/0", next.Reps, next.Lapses)
	}
	if log.ElapsedDays != 5 {
		t.Errorf("log elapsed days: got %d, want 5 (days since last review)", log.ElapsedDays)
	}
	if next.Stability <= card.Stability {
		t.Errorf("stability should grow on Good: %v -> %v", card.Stability, next.Stability)
	}
	if next.ScheduledDays > 365 {
		t.Errorf("scheduled days %d exceeds the deck's maximum interval", next.ScheduledDays)
	}
}

func TestScheduleNext_ReviewCardAgainRelapses(t *testing.T) {
	card := reviewCard()
	next, log, err := anki.ScheduleNext(card, params, again, now)
	if err != nil {
		t.Fatalf("ScheduleNext: %v", err)
	}
	if next.State != stateRelearning {
		t.Errorf("state: got %d, want relearning (%d)", next.State, stateRelearning)
	}
	if next.Lapses != card.Lapses+1 {
		t.Errorf("lapses: got %d, want %d", next.Lapses, card.Lapses+1)
	}
	if log.State != stateReview || log.Rating != again {
		t.Errorf("log: %+v", log)
	}
	if next.Due != ts(now.Add(5*time.Minute)) {
		t.Errorf("due: got %q, want %q (relearning step)", next.Due, ts(now.Add(5*time.Minute)))
	}
}

func TestScheduleNext_EmptyTimestampsAreNeverReviewed(t *testing.T) {
	card := newCard()
	card.Due, card.LastReview = "", ""
	if _, _, err := anki.ScheduleNext(card, params, good, now); err != nil {
		t.Errorf("empty timestamps must be accepted: %v", err)
	}
}

func TestScheduleNext_UnreadableTimestampIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name  string
		mut   func(*domain.AnkiCard)
		field string
	}{
		{"due", func(c *domain.AnkiCard) { c.Due = "2026-09-03T10:00:00Z" }, "due"},
		{"last review", func(c *domain.AnkiCard) { c.LastReview = "garbage" }, "last_review"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := reviewCard()
			tc.mut(&card)
			_, _, err := anki.ScheduleNext(card, params, good, now)
			if !errors.Is(err, anki.ErrUnreadableTimestamp) {
				t.Fatalf("got %v, want ErrUnreadableTimestamp", err)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Errorf("error %q does not name the field %q", err, tc.field)
			}
		})
	}
}

func TestScheduleNext_RatingOutOfRange(t *testing.T) {
	for _, r := range []int{-1, 0, 5} {
		if _, _, err := anki.ScheduleNext(newCard(), params, r, now); !errors.Is(err, anki.ErrInvalidRating) {
			t.Errorf("rating %d: got %v, want ErrInvalidRating", r, err)
		}
	}
}

// The second precision of TimeLayout is applied to now before scheduling, so
// a caller passing a nanosecond clock still gets strings that round-trip.
func TestScheduleNext_NowIsReadAtSecondPrecision(t *testing.T) {
	noisy := now.Add(750 * time.Millisecond).In(time.FixedZone("CEST", 2*3600))
	next, log, err := anki.ScheduleNext(newCard(), params, good, noisy)
	if err != nil {
		t.Fatalf("ScheduleNext: %v", err)
	}
	if next.LastReview != ts(now) || log.ReviewedAt != ts(now) {
		t.Errorf("now not normalised to UTC seconds: card %q log %q", next.LastReview, log.ReviewedAt)
	}
	if next.Due != ts(now.Add(10*time.Minute)) {
		t.Errorf("due: got %q, want %q", next.Due, ts(now.Add(10*time.Minute)))
	}
}
