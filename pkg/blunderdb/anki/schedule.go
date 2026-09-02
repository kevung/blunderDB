// Package anki holds the spaced-repetition scheduling every storage backend
// runs when a card is reviewed. It is the one place the FSRS library is
// called on a card: a backend reads the card and its deck's parameters,
// calls ScheduleNext, and writes back what it returns — the advanced card
// and the review-log entry — in one transaction.
//
// It lives beside domain rather than inside it because domain is
// dependency-free (CLAUDE.md) and this package depends on go-fsrs. It does
// not depend on storage: the backends wrap its errors with theirs.
package anki

import (
	"errors"
	"fmt"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TimeLayout is the textual form of every timestamp on a domain.AnkiCard and
// domain.AnkiReviewLog (Due, LastReview, ReviewedAt), always in UTC. SQLite
// stores this string as is; PostgreSQL formats its TIMESTAMPTZ columns into
// it and parses it back.
const TimeLayout = "2006-01-02 15:04:05"

var (
	// ErrInvalidRating is returned when the rating is outside 1 (Again) to
	// 4 (Easy). go-fsrs indexes its weights with the rating and would panic
	// on 0 or return nonsense on 5, so the check happens before it is called.
	ErrInvalidRating = errors.New("anki: rating must be between 1 (Again) and 4 (Easy)")

	// ErrUnreadableTimestamp is returned when a card's Due or LastReview is
	// neither empty nor in TimeLayout. The card is refused rather than
	// scheduled from a zero time: an unreadable row is corrupt and silently
	// treating it as "never reviewed" would rewrite it as if it were sound.
	ErrUnreadableTimestamp = errors.New("anki: unreadable card timestamp")
)

// Params are the per-deck FSRS knobs a user can tune (see domain.AnkiDeck).
// Everything else — the model weights — is go-fsrs's default.
type Params struct {
	RequestRetention float64
	MaximumInterval  float64
	EnableFuzz       bool
}

// ScheduleNext applies one review to a card: it returns the card as FSRS
// schedules it after being graded with rating at now, and the review-log
// entry that records the event.
//
// The returned card keeps the identity of the input (ID, DeckID,
// PositionID); its LastReview is now and its Due is the next date FSRS
// picked. The log records the state the card was in BEFORE the review, the
// stability and difficulty AFTER it, the days elapsed since the previous
// review and the interval granted by this one. now is read in UTC at
// second precision, the precision of TimeLayout, so the strings on both
// values round-trip exactly.
//
// An empty Due or LastReview means "never scheduled" / "never reviewed" and
// is read as the zero time (a fresh card has neither); any other string
// that does not parse is ErrUnreadableTimestamp.
func ScheduleNext(card domain.AnkiCard, params Params, rating int, now time.Time) (domain.AnkiCard, domain.AnkiReviewLog, error) {
	if rating < int(fsrs.Again) || rating > int(fsrs.Easy) {
		return domain.AnkiCard{}, domain.AnkiReviewLog{}, fmt.Errorf("%w: got %d", ErrInvalidRating, rating)
	}
	due, err := parseTime("due", card.Due)
	if err != nil {
		return domain.AnkiCard{}, domain.AnkiReviewLog{}, err
	}
	lastReview, err := parseTime("last_review", card.LastReview)
	if err != nil {
		return domain.AnkiCard{}, domain.AnkiReviewLog{}, err
	}
	now = now.UTC().Truncate(time.Second)

	fsrsParams := fsrs.DefaultParam()
	fsrsParams.RequestRetention = params.RequestRetention
	fsrsParams.MaximumInterval = params.MaximumInterval
	fsrsParams.EnableFuzz = params.EnableFuzz
	info := fsrs.NewFSRS(fsrsParams).Next(fsrs.Card{
		Due:           due,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		ElapsedDays:   uint64(card.ElapsedDays),
		ScheduledDays: uint64(card.ScheduledDays),
		Reps:          uint64(card.Reps),
		Lapses:        uint64(card.Lapses),
		State:         fsrs.State(card.State),
		LastReview:    lastReview,
	}, now, fsrs.Rating(rating))
	nowText := now.Format(TimeLayout)

	next := card
	next.Due = info.Card.Due.UTC().Format(TimeLayout)
	next.Stability = info.Card.Stability
	next.Difficulty = info.Card.Difficulty
	next.ElapsedDays = int(info.Card.ElapsedDays)
	next.ScheduledDays = int(info.Card.ScheduledDays)
	next.Reps = int(info.Card.Reps)
	next.Lapses = int(info.Card.Lapses)
	next.State = int(info.Card.State)
	next.LastReview = nowText

	log := domain.AnkiReviewLog{
		CardID:        card.ID,
		DeckID:        card.DeckID,
		PositionID:    card.PositionID,
		Rating:        rating,
		State:         int(info.ReviewLog.State),
		Stability:     next.Stability,
		Difficulty:    next.Difficulty,
		ElapsedDays:   int(info.ReviewLog.ElapsedDays),
		ScheduledDays: next.ScheduledDays,
		ReviewedAt:    nowText,
	}
	return next, log, nil
}

// ParseTime reads a TimeLayout timestamp as UTC; the empty string is the
// zero time. It is what a backend uses to turn a scheduled card's strings
// back into typed columns.
func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.ParseInLocation(TimeLayout, s, time.UTC)
}

func parseTime(field, s string) (time.Time, error) {
	t, err := ParseTime(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s %q", ErrUnreadableTimestamp, field, s)
	}
	return t, nil
}
