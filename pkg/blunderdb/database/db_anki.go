package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// =====================================================================
// Anki / FSRS functions
// =====================================================================
//
// Every method below is an adapter over the Storage backend
// (d.store.Anki(), see storage.AnkiStore): it takes d.mu the way the GUI and
// CLI expect, then delegates with the wrapper's implicit scope. The SQL and
// the FSRS scheduling (go-fsrs, DefaultParam tuned by the deck's parameters)
// live in storage/sqlite/anki_sqlite.go, held to the shared contract suite
// alongside the PostgreSQL backend.

// CreateAnkiDeck creates a new spaced repetition deck
func (d *Database) CreateAnkiDeck(name, description, sourceType string, sourceID int64, sourceCommand string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().CreateDeck(context.Background(), "", name, description, sourceType, sourceID, sourceCommand)
}

// GetAllAnkiDecks returns all Anki decks with card counts. The result is
// never nil: the GUI iterates it straight from JSON.
func (d *Database) GetAllAnkiDecks() ([]AnkiDeck, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	decks := []AnkiDeck{}
	for dk, err := range d.store.Anki().ListDecks(context.Background(), "") {
		if err != nil {
			return nil, err
		}
		decks = append(decks, *dk)
	}
	return decks, nil
}

// UpdateAnkiDeck updates name and description of a deck
func (d *Database) UpdateAnkiDeck(id int64, name, description string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().UpdateDeck(context.Background(), "", id, name, description)
}

// UpdateAnkiDeckParams updates the scheduling settings of a deck.
//
// sessionLimit nil means the deck has no session limit; a pointer to 0 means a
// limit that serves no card. The two are different states (ADR-0026 rule 3),
// which is why this crosses the Wails boundary as a pointer and not as a
// sentinel number.
func (d *Database) UpdateAnkiDeckParams(id int64, requestRetention float64, maximumInterval float64, enableFuzz bool, sessionLimit *int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().UpdateDeckParams(context.Background(), "", id, requestRetention, maximumInterval, enableFuzz, sessionLimit)
}

// DeleteAnkiDeck deletes a deck and all its cards
func (d *Database) DeleteAnkiDeck(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().DeleteDeck(context.Background(), "", id)
}

// SyncAnkiDeck populates cards from the deck's source (collection or search)
func (d *Database) SyncAnkiDeck(deckID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().Sync(context.Background(), "", deckID)
}

// SyncAnkiDeckWithPositions syncs a deck with explicit position IDs (for search-based decks)
func (d *Database) SyncAnkiDeckWithPositions(deckID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().SyncWithPositions(context.Background(), "", deckID, positionIDs)
}

// GetAnkiDeckPositions returns all positions associated with a deck's cards.
func (d *Database) GetAnkiDeckPositions(deckID int64) ([]Position, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var positions []Position
	for pos, err := range d.store.Anki().DeckPositions(context.Background(), "", deckID) {
		if err != nil {
			return nil, err
		}
		positions = append(positions, *pos)
	}
	return positions, nil
}

// GetAnkiDeckStats returns review statistics for a deck. The queue counters
// leave suspended and buried cards out; TotalCount counts the whole deck.
func (d *Database) GetAnkiDeckStats(deckID int64) (AnkiDeckStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return AnkiDeckStats{}, fmt.Errorf("no database is currently open")
	}

	stats, err := d.store.Anki().DeckStats(context.Background(), "", deckID)
	if err != nil {
		return AnkiDeckStats{}, err
	}
	return *stats, nil
}

// GetAnkiForecast projects how many cards of a deck come due over the next
// days calendar days (offset 0 absorbing every overdue card); deckID 0 covers
// every deck. It delegates to the Storage backend so the CLI's `anki forecast`
// and the daemon's /v1/anki.forecast read the same projection.
func (d *Database) GetAnkiForecast(deckID int64, days int) ([]AnkiForecastDay, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().Forecast(context.Background(), "", deckID, days)
}

// GetNextAnkiCard returns the next card due for review in a deck, or nil when
// nothing is due (suspended and buried cards never surface here).
func (d *Database) GetNextAnkiCard(deckID int64) (*AnkiReviewCard, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	card, err := d.store.Anki().NextCard(context.Background(), "", deckID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	return card, err
}

// GetRandomAnkiCard returns a random card from the deck for a "cram" (free
// drill) session: it ignores the FSRS schedule — both the due date and the card
// state — so the user can practise any position on demand (e.g. a warm-up
// before a tournament). Unlike GetNextAnkiCard paired with ReviewAnkiCard, cram
// never mutates scheduling, so it can't disturb the real review plan.
//
// excludePositionID, when non-zero, is skipped so two consecutive draws don't
// repeat the same position; for a single-card deck it falls back to the full
// deck so the lone card is still served. Returns nil when the deck has no cards.
func (d *Database) GetRandomAnkiCard(deckID int64, excludePositionID int64) (*AnkiReviewCard, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	card, err := d.store.Anki().RandomCard(context.Background(), "", deckID, excludePositionID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	return card, err
}

// ReviewAnkiCard submits a review rating for a card, updates its FSRS state,
// appends the review to the deck's log, and returns the next card still due
// in the same deck (nil when none remain).
func (d *Database) ReviewAnkiCard(cardID int64, rating int) (*AnkiReviewCard, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().ReviewCard(context.Background(), "", cardID, rating)
}

// ResetAnkiDeck resets all cards in a deck to new state
func (d *Database) ResetAnkiDeck(deckID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().ResetDeck(context.Background(), "", deckID)
}

// GetAnkiDeckRetention reports what a deck's review log measures against the
// target retention its owner chose.
//
// Read-only, and deliberately so (ADR-0026 rule 5): the target is a choice on
// the work/knowledge trade-off, the measurement is its outcome, and steering
// one by the other is the mechanism FSRS's authors reject. Exposed here because
// the GUI must be able to show what the daemon and the CLI can already read —
// this method is the parity that was missing while the same capability was
// served over HTTP and reachable from nowhere else.
func (d *Database) GetAnkiDeckRetention(deckID int64) (*domain.AnkiRetention, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.Anki().Retention(context.Background(), "", deckID)
}
