package storage

import (
	"context"
	"iter"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// BuildForecast expands a sparse {day-offset → count} map into a dense slice of
// `days` AnkiForecastDay entries, one per calendar day starting at `now` (UTC),
// filling absent offsets with zero. Backends compute the offset→count map in
// SQL; this keeps the calendar-date formatting identical across them.
func BuildForecast(now time.Time, days int, counts map[int]int) []domain.AnkiForecastDay {
	day0 := now.UTC().Truncate(24 * time.Hour)
	out := make([]domain.AnkiForecastDay, days)
	for i := 0; i < days; i++ {
		out[i] = domain.AnkiForecastDay{
			Day: day0.AddDate(0, 0, i).Format("2006-01-02"),
			Due: counts[i],
		}
	}
	return out
}

// AnkiStore persists spaced-repetition decks and their FSRS review state.
type AnkiStore interface {
	CreateDeck(ctx context.Context, scope string, name, description, sourceType string, sourceID int64, sourceCommand string) (int64, error)
	ListDecks(ctx context.Context, scope string) iter.Seq2[*domain.AnkiDeck, error]
	UpdateDeck(ctx context.Context, scope string, id int64, name, description string) error
	UpdateDeckParams(ctx context.Context, scope string, id int64, requestRetention, maximumInterval float64, enableFuzz bool) error
	DeleteDeck(ctx context.Context, scope string, id int64) error
	ResetDeck(ctx context.Context, scope string, deckID int64) error

	// Sync reconciles a deck's cards with its source (collection or search).
	Sync(ctx context.Context, scope string, deckID int64) error

	// SyncWithPositions reconciles a deck's cards with an explicit position set.
	SyncWithPositions(ctx context.Context, scope string, deckID int64, positionIDs []int64) error

	// DeckPositions streams the positions of a deck.
	DeckPositions(ctx context.Context, scope string, deckID int64) iter.Seq2[*domain.Position, error]

	// DeckStats returns the review counters for a deck.
	DeckStats(ctx context.Context, scope string, deckID int64) (*domain.AnkiDeckStats, error)

	// Forecast projects how many cards come due over the next `days` calendar
	// days (offset 0 absorbs overdue cards). A deckID of 0 spans every deck in
	// the tenant. The returned slice has one entry per day, including zeros.
	Forecast(ctx context.Context, scope string, deckID int64, days int) ([]domain.AnkiForecastDay, error)

	// NextCard returns the next card due for review in a deck, or ErrNotFound.
	NextCard(ctx context.Context, scope string, deckID int64) (*domain.AnkiReviewCard, error)

	// ReviewCard records a review rating and returns the next card to review.
	ReviewCard(ctx context.Context, scope string, cardID int64, rating int) (*domain.AnkiReviewCard, error)

	// SetCardSuspended suspends or unsuspends a card. A suspended card never
	// surfaces for review until it is unsuspended.
	SetCardSuspended(ctx context.Context, scope string, cardID int64, suspended bool) error

	// BuryCard hides a card until the start of the next day, after which it is
	// available for review again.
	BuryCard(ctx context.Context, scope string, cardID int64) error

	// RemoveCard deletes a single card from its deck.
	RemoveCard(ctx context.Context, scope string, cardID int64) error

	// ReviewLog streams the recorded review events, most recent first. A deckID
	// of 0 spans every deck in the tenant; limit <= 0 means no limit.
	ReviewLog(ctx context.Context, scope string, deckID int64, limit int) iter.Seq2[*domain.AnkiReviewLog, error]

	// OptimizeParams derives a request-retention tuning suggestion for a deck
	// from its review log and, when apply is true, writes it back. Returns
	// ErrNotFound for an unknown deck.
	OptimizeParams(ctx context.Context, scope string, deckID int64, apply bool) (*domain.AnkiOptimizeResult, error)
}
