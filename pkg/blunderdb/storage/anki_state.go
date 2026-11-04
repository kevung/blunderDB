package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// AnkiStateStore reads and writes a deck and a card whole, scheduling state
// included, which the review gestures of AnkiStore never do: it is what lets
// the changeset carry a deck and its cards to another library and put them
// back exactly as FSRS had them.
type AnkiStateStore interface {
	// Deck is one deck, ErrNotFound when it is not in the scope.
	Deck(ctx context.Context, scope string, id int64) (*domain.AnkiDeck, error)
	// PutDeck inserts d when d.ID is 0 and returns the new id, otherwise
	// rewrites the deck's name, description, source and scheduling settings
	// (ErrNotFound when it is gone). The counters and dates of d are ignored.
	PutDeck(ctx context.Context, scope string, d domain.AnkiDeck) (int64, error)
	// Card is one card, ErrNotFound when it is not in the scope.
	Card(ctx context.Context, scope string, id int64) (*domain.AnkiCard, error)
	// FindCard is the id of the card a deck holds under (kind, key), false
	// when it holds none.
	FindCard(ctx context.Context, scope string, deckID int64, kind, key string) (int64, bool, error)
	// PutCard inserts c when c.ID is 0 and returns the new id, otherwise
	// rewrites the card's scheduling state, suspension and burial (the deck,
	// kind and key are its identity and stay). ErrConflict when the deck
	// already holds a card of that identity, ErrNotFound when the deck (or the
	// card) is not in the scope.
	PutCard(ctx context.Context, scope string, c domain.AnkiCard) (int64, error)
}

// AnkiStateProvider is implemented by the Stores of a backend that keeps
// decks and cards (the remote client does not).
type AnkiStateProvider interface {
	AnkiStates() AnkiStateStore
}

// MatchDetached is the tournament order of a match detached by a gesture, which
// says more than the order of a match that never had a tournament.
const MatchDetached = -1

// MatchPlaceStore writes the order a peer gave a match in its tournament, which
// ReorderMatches, renumbering a whole list, cannot.
type MatchPlaceStore interface {
	SetMatchOrder(ctx context.Context, scope string, matchID int64, order int) error
}

// MatchPlaceProvider is implemented by the Stores of a backend that merges a
// changeset (the remote client does not).
type MatchPlaceProvider interface {
	MatchPlaces() MatchPlaceStore
}
