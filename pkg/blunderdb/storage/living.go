package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
)

// LivingCollectionCap is the declared ceiling of one evaluation of a
// collection (ADR-0084): no evaluation answers more ids than this, and one
// that stops there says so. A query is cheap to write and can select the whole
// library; a study deck fed from it cannot take a hundred thousand cards.
const LivingCollectionCap = 5000

// CollectionEvaluation is a collection read whole, under a declared ceiling.
// Total is how many positions the collection holds in truth; Truncated is
// true exactly when PositionIDs stops short of it. A truncation is never
// silent: the caller always learns both numbers.
type CollectionEvaluation struct {
	CollectionID int64   `json:"collectionId"`
	Living       bool    `json:"living"`
	FilterQuery  string  `json:"filterQuery"`
	PositionIDs  []int64 `json:"positionIds"`
	Total        int     `json:"total"`
	Cap          int     `json:"cap"`
	Truncated    bool    `json:"truncated"`
}

// ErrUnreadableFilter is a filter query a token of which no rule claims. It is
// refused when it is set, so a saved collection never means "everything". It
// is an ErrInvalid: the request, not the store, is at fault.
var ErrUnreadableFilter = fmt.Errorf("%w: filter query carries tokens nothing claims", ErrInvalid)

// ValidateFilterQuery refuses a query the parser cannot read whole. A blank
// query is valid: it turns a collection back into a hand-made list.
func ValidateFilterQuery(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	_, diags := searchquery.Parse(query)
	var unknown []string
	for _, d := range diags {
		if d.Kind == searchquery.DiagUnknown {
			unknown = append(unknown, d.Token)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("%w: %s", ErrUnreadableFilter, strings.Join(unknown, ", "))
	}
	return nil
}

// SetCollectionFilter makes a collection living after checking its query, so
// every mode refuses the same queries.
func SetCollectionFilter(ctx context.Context, st Storage, scope string, id int64, query string) error {
	query = strings.TrimSpace(query)
	if err := ValidateFilterQuery(query); err != nil {
		return err
	}
	return st.Collections().SetFilterQuery(ctx, scope, id, query)
}

// CreateCollection creates a collection, living when query is not blank. The
// query is checked before anything is written, so a refusal leaves no empty
// collection behind.
func CreateCollection(ctx context.Context, st Storage, scope, name, description, query string) (int64, error) {
	query = strings.TrimSpace(query)
	if err := ValidateFilterQuery(query); err != nil {
		return 0, err
	}
	id, err := st.Collections().Create(ctx, scope, name, description)
	if err != nil || query == "" {
		return id, err
	}
	if err := st.Collections().SetFilterQuery(ctx, scope, id, query); err != nil {
		_ = st.Collections().Delete(ctx, scope, id)
		return 0, err
	}
	return id, nil
}

// LivingFilters reads a collection's saved query and resolves it through the
// parser the command bar runs. living is false for a hand-made list.
func LivingFilters(ctx context.Context, st Storage, scope string, collectionID int64) (filters domain.SearchFilters, query string, living bool, err error) {
	col, err := st.Collections().Get(ctx, scope, collectionID)
	if err != nil {
		return domain.SearchFilters{}, "", false, err
	}
	query = strings.TrimSpace(col.FilterQuery)
	filters, living, err = searchquery.Living(collectionID, query)
	return filters, query, living, err
}

// clampCap bounds a requested limit by the declared ceiling; zero or less asks
// for the ceiling itself.
func clampCap(limit int) int {
	if limit <= 0 || limit > LivingCollectionCap {
		return LivingCollectionCap
	}
	return limit
}

// EvaluateCollection reads a collection's position ids, at most limit of them
// (bounded by LivingCollectionCap), in the order GetCollectionPositions uses.
// A living collection is evaluated by the search engine within scope, so the
// tenant's row-level security confines it like any search.
func EvaluateCollection(ctx context.Context, st Storage, scope string, collectionID int64, limit int) (*CollectionEvaluation, error) {
	limit = clampCap(limit)
	filters, query, living, err := LivingFilters(ctx, st, scope, collectionID)
	if err != nil {
		return nil, err
	}
	ev := &CollectionEvaluation{CollectionID: collectionID, Living: living, FilterQuery: query, Cap: limit}
	if living {
		if ev.PositionIDs, err = st.Search().FindIDs(ctx, scope, filters, ListOpts{Limit: limit}); err != nil {
			return nil, err
		}
		if ev.Total, err = st.Search().Count(ctx, scope, filters); err != nil {
			return nil, err
		}
	} else {
		if ev.PositionIDs, err = st.Collections().PositionIDs(ctx, scope, collectionID, ListOpts{Limit: limit}); err != nil {
			return nil, err
		}
		if ev.Total, err = st.Collections().CountPositions(ctx, scope, collectionID); err != nil {
			return nil, err
		}
	}
	if ev.PositionIDs == nil {
		ev.PositionIDs = []int64{}
	}
	ev.Truncated = ev.Total > len(ev.PositionIDs)
	return ev, nil
}

// DeckSync reports one resynchronisation of a deck with its source. Source is
// set when the deck is fed by a living collection; a truncated source fed the
// deck its first Source.Cap positions only.
type DeckSync struct {
	DeckID int64                 `json:"deckId"`
	Source *CollectionEvaluation `json:"source,omitempty"`
}

// SyncDeck reconciles a deck's cards with its source. A deck fed by a living
// collection reads it through EvaluateCollection, so its query is re-evaluated
// at every sync — the moment a study session opens — and its ceiling
// reported. Other sources go to AnkiStore.Sync.
func SyncDeck(ctx context.Context, st Storage, scope string, deckID int64) (*DeckSync, error) {
	deck, err := findDeck(ctx, st, scope, deckID)
	if err != nil {
		return nil, err
	}
	report := &DeckSync{DeckID: deckID}
	if deck.SourceType != domain.AnkiSourceCollection {
		return report, st.Anki().Sync(ctx, scope, deckID)
	}
	_, _, living, err := LivingFilters(ctx, st, scope, deck.SourceID)
	if err != nil {
		return nil, err
	}
	if !living {
		// A hand-made list was chosen position by position: it feeds its
		// deck whole, as it always has.
		return report, st.Anki().Sync(ctx, scope, deckID)
	}
	ev, err := EvaluateCollection(ctx, st, scope, deck.SourceID, 0)
	if err != nil {
		return nil, err
	}
	report.Source = ev
	return report, st.Anki().SyncWithPositions(ctx, scope, deckID, ev.PositionIDs)
}

func findDeck(ctx context.Context, st Storage, scope string, deckID int64) (*domain.AnkiDeck, error) {
	for d, err := range st.Anki().ListDecks(ctx, scope) {
		if err != nil {
			return nil, err
		}
		if d.ID == deckID {
			return d, nil
		}
	}
	return nil, fmt.Errorf("anki deck %d: %w", deckID, ErrNotFound)
}
