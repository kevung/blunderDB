package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// « Des positions comme celle-ci » (#293, fiche J.3 ; ADR-0043), côté base.

// SimilarPositionsLimit is how many neighbours a ranking returns when the
// query names no number of its own.
const SimilarPositionsLimit = 10

// RankPositionsByFilters answers a query carrying the `like` token: the
// positions it selects, nearest first, each with the distance that ranked it.
//
// It is the ranked counterpart of LoadPositionsByFilters, and a separate call
// for one reason: the DISTANCE is part of the answer. A neighbour without its
// distance is unreadable — it is the only thing that says whether one is
// looking at a neighbour or at a coincidence — and a Position has no business
// carrying a number that belongs to a comparison rather than to itself.
//
// The bare `like` token has no target of its own: whoever typed it resolves it
// to whatever "this position" means where it was typed, and hands the id over
// here. Nothing downstream guesses it.
func (d *Database) RankPositionsByFilters(f domain.SearchFilters, limit int) ([]storage.SimilarPosition, error) {
	if limit <= 0 {
		limit = SimilarPositionsLimit
	}
	f.LikeFilter = true
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Search().Rank(context.Background(), "", f, storage.ListOpts{Limit: limit})
}

// SimilarPositions is the ranking with no other filter: the neighbours of one
// position, in its class, nearest first.
func (d *Database) SimilarPositions(positionID int, limit int) ([]storage.SimilarPosition, error) {
	return d.RankPositionsByFilters(domain.SearchFilters{
		LikeFilter:   true,
		LikeTargetID: int64(positionID),
	}, limit)
}

// RankedID is one neighbour as the interface needs it: which position, and how
// far. Only ids cross the Wails bridge (D.8, #208) — the board itself is
// fetched later, for the window actually shown — and the distance travels with
// the id because it is not a property of the position and cannot be recomputed
// on the other side without the target.
type RankedID struct {
	ID       int64 `json:"id"`
	Distance int   `json:"distance"`
}

// RankPositionIDsByFilters is RankPositionsByFilters for the interface: the
// same ranking, reported as ids and distances.
func (d *Database) RankPositionIDsByFilters(f domain.SearchFilters, limit int) ([]RankedID, error) {
	neighbours, err := d.RankPositionsByFilters(f, limit)
	if err != nil {
		return nil, err
	}
	out := make([]RankedID, len(neighbours))
	for i, n := range neighbours {
		out[i] = RankedID{ID: n.Position.ID, Distance: n.Distance}
	}
	return out, nil
}
