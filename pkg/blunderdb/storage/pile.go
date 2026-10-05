package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// PileName is the name the Pile collection is created under. It is only a
// first name: the Pile is designated by a library setting, so renaming it
// changes nothing about which collection the gesture aims at (CONTEXT.md
// "Pile").
const PileName = "Pile"

// PileToggle reports what the Pile gesture did.
type PileToggle struct {
	// OnPile is true when the position is on the Pile after the gesture.
	OnPile bool `json:"onPile"`
	// PositionID is the stored position the gesture acted on; for a draft it
	// is the row the gesture just wrote.
	PositionID int64 `json:"positionId"`
	// CollectionID is the Pile collection.
	CollectionID int64 `json:"collectionId"`
	// Brought is true when the position was not in the library and the
	// gesture wrote it, as a position brought in on its own.
	Brought bool `json:"brought"`
}

// pileCollection returns the Pile's collection id, or 0 when the library has
// no Pile (never named, or its collection was deleted).
func pileCollection(ctx context.Context, st Storage, scope string) (int64, error) {
	id, err := st.LibrarySettings().PileCollection(ctx, scope)
	if err != nil || id == 0 {
		return 0, err
	}
	if _, err := st.Collections().Get(ctx, scope, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

// EnsurePile returns the Pile's collection id, creating the collection on
// first use and again when it was deleted.
func EnsurePile(ctx context.Context, st Storage, scope string) (int64, error) {
	id, err := pileCollection(ctx, st, scope)
	if err != nil || id != 0 {
		return id, err
	}
	id, err = st.Collections().Create(ctx, scope, PileName, "")
	if err != nil {
		return 0, err
	}
	if err := st.LibrarySettings().SetPileCollection(ctx, scope, id); err != nil {
		return 0, err
	}
	return id, nil
}

// PositionOnPile reports whether a stored position is on the Pile. A library
// with no Pile yet answers false without creating one: reading writes nothing.
func PositionOnPile(ctx context.Context, st Storage, scope string, positionID int64) (bool, error) {
	if positionID <= 0 {
		return false, nil
	}
	id, err := pileCollection(ctx, st, scope)
	if err != nil || id == 0 {
		return false, err
	}
	_, found, err := st.Collections().IndexOfPosition(ctx, scope, id, positionID)
	return found, err
}

// TogglePile puts the position on the Pile, or takes it off when it is there.
// A position with no stored row (ID 0: a draft board, a Duel position) is
// first written as a position brought in on its own — the one write path for
// a user's own position — then put on the Pile.
func TogglePile(ctx context.Context, st Storage, scope string, pos *domain.Position) (PileToggle, error) {
	if pos == nil {
		return PileToggle{}, fmt.Errorf("pile: no position: %w", ErrInvalid)
	}
	out := PileToggle{PositionID: pos.ID}
	if out.PositionID <= 0 {
		p := *pos
		p.IndividuallyImported = true
		id, created, err := st.Positions().SaveCreated(ctx, scope, &p)
		if err != nil {
			return PileToggle{}, err
		}
		out.PositionID, out.Brought = id, created
	}
	cid, err := EnsurePile(ctx, st, scope)
	if err != nil {
		return PileToggle{}, err
	}
	out.CollectionID = cid
	c, err := st.Collections().Get(ctx, scope, cid)
	if err != nil {
		return PileToggle{}, err
	}
	if c.FilterQuery != "" {
		return PileToggle{}, fmt.Errorf("pile: the Pile collection is living, it holds no hand-picked positions: %w", ErrInvalid)
	}
	_, on, err := st.Collections().IndexOfPosition(ctx, scope, cid, out.PositionID)
	if err != nil {
		return PileToggle{}, err
	}
	if on {
		err = st.Collections().RemovePosition(ctx, scope, cid, out.PositionID)
	} else {
		err = st.Collections().AddPosition(ctx, scope, cid, out.PositionID)
	}
	if err != nil {
		return PileToggle{}, err
	}
	out.OnPile = !on
	return out, nil
}

// PileCollectionID returns the Pile's collection id, or 0 when the library has
// none yet. Nothing is created.
func PileCollectionID(ctx context.Context, st Storage, scope string) (int64, error) {
	return pileCollection(ctx, st, scope)
}
