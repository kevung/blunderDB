package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// PositionStore persists backgammon positions. Positions are deduplicated by
// their Zobrist hash; Save is idempotent for an already-stored position and
// returns the existing id.
type PositionStore interface {
	// Save stores p (or returns the id of an identical existing position).
	Save(ctx context.Context, scope string, p *domain.Position) (int64, error)

	// Update overwrites the stored position with the same id as p.
	Update(ctx context.Context, scope string, p *domain.Position) error

	// Load returns the position with the given id, or ErrNotFound.
	Load(ctx context.Context, scope string, id int64) (*domain.Position, error)

	// Exists reports whether a position with the given Zobrist hash is stored,
	// returning its id when found.
	Exists(ctx context.Context, scope string, zobrist uint64) (id int64, found bool, err error)

	// Delete removes the position with the given id (analysis, comments and
	// collection links cascade).
	Delete(ctx context.Context, scope string, id int64) error

	// List streams stored positions.
	List(ctx context.Context, scope string, opts ListOpts) iter.Seq2[*domain.Position, error]

	// ListIDs returns the ids of the stored positions, in List's order and
	// bounded the same way by opts. It is the cheap face of List: a client
	// that browses a library keeps this list and fetches the positions it
	// shows with LoadByIDs, instead of materialising every row up front.
	ListIDs(ctx context.Context, scope string, opts ListOpts) ([]int64, error)

	// LoadByIDs returns the positions whose ids are listed, in the order the
	// caller gave them, in one round trip per batch rather than one per id.
	// Unknown ids are skipped rather than failing the call: callers hand
	// over lists gathered earlier (a search result, a saved selection, an id
	// window from ListIDs), and a position deleted in between is not a
	// reason to fail — or lose the rest of — the batch.
	LoadByIDs(ctx context.Context, scope string, ids []int64) ([]domain.Position, error)

	// ReclassifyDerived recomputes the derived phase of every position whose
	// stored value disagrees with engine.ClassifyGamePhase, and returns how
	// many rows changed (issue #264, ADR-0035).
	//
	// The phase is derived, never edited, and this is what makes that true:
	// change the classifier or its threshold, run this, and every row agrees
	// with the new rule. `blunderdb repair` runs it, so does the 2.19.0
	// migration, and so does /v1/positions.reclassifyPhases. Running it on a
	// database that is already up to date rewrites nothing.
	ReclassifyDerived(ctx context.Context, scope string) (int, error)

	// Similar returns the positions closest to target, nearest first and
	// excluding target itself, by the transport distance
	// engine.SimilarityDistance defines (issue #293).
	//
	// It is an EXHAUSTIVE scan, deliberately: below about a hundred thousand
	// positions an exact scan beats any approximate index on both recall and
	// on the amount of machinery to keep in step with every write
	// (docs/recherche/P7-similarite-knn-go.md). The contract therefore
	// promises exact nearest neighbours, not approximate ones.
	Similar(ctx context.Context, scope string, target *domain.Position, limit int) ([]SimilarPosition, error)
}

// SimilarPosition is one neighbour and how far it stands, in checker-pips: the
// amount of checker movement separating it from the position asked about.
type SimilarPosition struct {
	Position domain.Position `json:"position"`
	Distance int             `json:"distance"`
}
