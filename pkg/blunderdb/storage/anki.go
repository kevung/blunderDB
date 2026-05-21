package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

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

	// NextCard returns the next card due for review in a deck, or ErrNotFound.
	NextCard(ctx context.Context, scope string, deckID int64) (*domain.AnkiReviewCard, error)

	// ReviewCard records a review rating and returns the next card to review.
	ReviewCard(ctx context.Context, scope string, cardID int64, rating int) (*domain.AnkiReviewCard, error)
}
