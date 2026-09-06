package storage

import (
	"context"
	"iter"

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
	// FilterQuery makes the collection LIVING (#282): its membership is the
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
	// it back into a hand-made list. It is a method of its own rather than a
	// parameter of Update because the two are different gestures: renaming a
	// collection is not the same act as changing what it contains.
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
