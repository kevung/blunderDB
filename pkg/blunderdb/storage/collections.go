package storage

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Collection is a named, ordered group of positions.
type Collection struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	SortOrder     int    `json:"sortOrder"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	PositionCount int    `json:"positionCount"`
	// FilterQuery makes the collection LIVING: its membership is the
	// result of this query, in the grammar the command bar speaks,
	// re-evaluated every time it is opened. Empty is the ordinary case — a
	// hand-made list whose membership is the stored rows.
	//
	// A living collection keeps no membership rows, so PositionCount is 0 for
	// it: the count is a property of the query's result, and answering it here
	// would mean running the search on every listing.
	FilterQuery string `json:"filterQuery"`
}

// CollectionPosition is a position's membership in a collection, with order.
type CollectionPosition struct {
	ID           int64           `json:"id"`
	CollectionID int64           `json:"collectionId"`
	PositionID   int64           `json:"positionId"`
	SortOrder    int             `json:"sortOrder"`
	AddedAt      string          `json:"addedAt"`
	Position     domain.Position `json:"position"`
}

// CollectionStore persists position collections and their membership.
type CollectionStore interface {
	Create(ctx context.Context, scope string, name, description string) (int64, error)
	Get(ctx context.Context, scope string, id int64) (*Collection, error)
	List(ctx context.Context, scope string) iter.Seq2[*Collection, error]
	Update(ctx context.Context, scope string, id int64, name, description string) error

	// SetFilterQuery makes a collection living, or (with an empty query) turns
	// it back into a hand-made list. Not a parameter of Update: renaming and
	// changing what it contains are different gestures.
	SetFilterQuery(ctx context.Context, scope string, id int64, query string) error
	Delete(ctx context.Context, scope string, id int64) error
	Reorder(ctx context.Context, scope string, collectionIDs []int64) error

	AddPosition(ctx context.Context, scope string, collectionID, positionID int64) error
	AddPositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error
	RemovePosition(ctx context.Context, scope string, collectionID, positionID int64) error
	RemovePositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error
	ReorderPositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error
	MovePosition(ctx context.Context, scope string, fromCollectionID, toCollectionID, positionID int64) error
	CopyPosition(ctx context.Context, scope string, toCollectionID, positionID int64) error

	// Positions streams the positions of a collection in order.
	Positions(ctx context.Context, scope string, collectionID int64, opts ListOpts) iter.Seq2[*domain.Position, error]

	// PositionIDs returns the window of a collection's position ids, in the
	// order Positions walks them; CountPositions and IndexOfPosition
	// complete it, so a caller browses a collection by windows without ever
	// holding its whole membership. IndexOfPosition reports false for a
	// position the collection does not hold.
	PositionIDs(ctx context.Context, scope string, collectionID int64, opts ListOpts) ([]int64, error)
	CountPositions(ctx context.Context, scope string, collectionID int64) (int, error)
	IndexOfPosition(ctx context.Context, scope string, collectionID, positionID int64) (int, bool, error)

	// Members streams a collection's membership rows in collection order,
	// each carrying the position it links. Positions is the same walk
	// projected onto the positions; Members is for a caller that needs the
	// membership itself — its rank and the moment it was added — such as an
	// export that must reproduce a collection exactly.
	Members(ctx context.Context, scope string, collectionID int64) iter.Seq2[*CollectionPosition, error]

	// Coverage reports, for every collection, how many of its positions are
	// among positionIDs. Every collection is a key: one with no member in the
	// selection — an empty collection included — maps to 0. The export
	// screen uses it to say that a collection will arrive truncated before
	// anything is written.
	Coverage(ctx context.Context, scope string, positionIDs []int64) (map[int64]int, error)

	// CollectionsOf streams the collections a position belongs to.
	CollectionsOf(ctx context.Context, scope string, positionID int64) iter.Seq2[*Collection, error]

	// PositionIndexMap returns, for every stored position id, its display index.
	PositionIndexMap(ctx context.Context, scope string) (map[int64]int, error)
}

// NotLivingError says a hand-made collection has no query to freeze. It is the
// caller's mistake (ErrInvalid), so every mode reports it as a client error.
func NotLivingError(id int64) error {
	return fmt.Errorf("%w: collection %d is not living: nothing to freeze", ErrInvalid, id)
}

// FreezeCollection turns a LIVING collection into an ordinary one: the
// positions its query selects now become its membership rows, in the order
// the search returns them, and the query is cleared. filters is the query
// already resolved by the caller, through the parser the command bar runs.
//
// The query is cleared last, so a failure midway leaves the collection living
// rather than half-frozen under a query it no longer obeys. It returns how
// many positions the collection now holds.
func FreezeCollection(ctx context.Context, st Storage, scope string, id int64, filters domain.SearchFilters) (int, error) {
	cs := st.Collections()
	ids, err := st.Search().FindIDs(ctx, scope, filters, ListOpts{})
	if err != nil {
		return 0, err
	}
	old, err := cs.PositionIDs(ctx, scope, id, ListOpts{})
	if err != nil {
		return 0, err
	}
	if len(old) > 0 {
		if err := cs.RemovePositions(ctx, scope, id, old); err != nil {
			return 0, err
		}
	}
	if len(ids) > 0 {
		if err := cs.AddPositions(ctx, scope, id, ids); err != nil {
			return 0, err
		}
	}
	if err := cs.SetFilterQuery(ctx, scope, id, ""); err != nil {
		return 0, err
	}
	return len(ids), nil
}

// CollectionCopyTarget says where CopyCollection copies to: exactly one of
// ToID (an existing collection) and Name (a new collection) is set.
type CollectionCopyTarget struct {
	ToID int64
	Name string
}

// CollectionCopyResult is what CopyCollection wrote: the target collection
// and how many positions it gained.
type CollectionCopyResult struct {
	ID     int64 `json:"id"`
	Copied int   `json:"copied"`
}

// CollectionCopier is implemented by a backend that runs CopyCollection
// itself, whole: the remote backend, which has no transaction to offer and
// hands the copy to the daemon's own.
type CollectionCopier interface {
	CopyCollection(ctx context.Context, scope string, fromID int64, target CollectionCopyTarget) (CollectionCopyResult, error)
}

// CopyCollection copies the membership of collection fromID, in its order,
// to a new collection named target.Name or to the end of the existing
// collection target.ToID, in one transaction: a failure copies nothing and
// creates nothing. A position the target already holds is skipped, so
// Copied counts only the positions added.
//
// A new collection takes the source's description and is placed last in the
// collection list; it is a hand-made list. A living collection is refused on
// either side: its membership is a query's result, not rows to copy, and rows
// added to it would be ignored — freezing it first makes the gesture explicit.
// Names are not unique, so a new name never conflicts.
func CopyCollection(ctx context.Context, st Storage, scope string, fromID int64, target CollectionCopyTarget) (CollectionCopyResult, error) {
	name := strings.TrimSpace(target.Name)
	if (target.ToID != 0) == (name != "") {
		return CollectionCopyResult{}, fmt.Errorf("%w: copy collection %d: give exactly one of a target collection and a new name", ErrInvalid, fromID)
	}
	if target.ToID == fromID {
		return CollectionCopyResult{}, fmt.Errorf("%w: copy collection %d onto itself", ErrInvalid, fromID)
	}
	if c, ok := st.(CollectionCopier); ok {
		return c.CopyCollection(ctx, scope, fromID, CollectionCopyTarget{ToID: target.ToID, Name: name})
	}
	// Guarded: the target's membership read below must still hold when the
	// missing positions are appended, or a concurrent copy could interleave.
	tx, err := BeginGuarded(ctx, st, fmt.Sprintf("collection-copy:%s:%d", scope, target.ToID))
	if err != nil {
		return CollectionCopyResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	cs := tx.Collections()

	src, err := cs.Get(ctx, scope, fromID)
	if err != nil {
		return CollectionCopyResult{}, err
	}
	if strings.TrimSpace(src.FilterQuery) != "" {
		return CollectionCopyResult{}, fmt.Errorf("%w: collection %d is living: freeze it before copying it", ErrInvalid, fromID)
	}
	ids, err := cs.PositionIDs(ctx, scope, fromID, ListOpts{})
	if err != nil {
		return CollectionCopyResult{}, err
	}

	held := map[int64]bool{}
	toID := target.ToID
	if toID == 0 {
		if toID, err = cs.Create(ctx, scope, name, src.Description); err != nil {
			return CollectionCopyResult{}, err
		}
	} else {
		dst, err := cs.Get(ctx, scope, toID)
		if err != nil {
			return CollectionCopyResult{}, err
		}
		if strings.TrimSpace(dst.FilterQuery) != "" {
			return CollectionCopyResult{}, fmt.Errorf("%w: collection %d is living: its content is its query's result", ErrInvalid, toID)
		}
		have, err := cs.PositionIDs(ctx, scope, toID, ListOpts{})
		if err != nil {
			return CollectionCopyResult{}, err
		}
		for _, id := range have {
			held[id] = true
		}
	}

	missing := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !held[id] {
			held[id] = true
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		if err := cs.AddPositions(ctx, scope, toID, missing); err != nil {
			return CollectionCopyResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return CollectionCopyResult{}, err
	}
	return CollectionCopyResult{ID: toID, Copied: len(missing)}, nil
}
