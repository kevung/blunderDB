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
	Positions(ctx context.Context, scope string, collectionID int64) iter.Seq2[*domain.Position, error]

	// CollectionsOf streams the collections a position belongs to.
	CollectionsOf(ctx context.Context, scope string, positionID int64) iter.Seq2[*Collection, error]

	// PositionIndexMap returns, for every stored position id, its display index.
	PositionIndexMap(ctx context.Context, scope string) (map[int64]int, error)
}
