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

	// Similar returns the neighbours of target, nearest first and excluding
	// target itself, by the transport distance engine.SimilarityDistance
	// defines (issue #293, ADR-0043).
	//
	// It is an EXHAUSTIVE scan, deliberately: below about a hundred thousand
	// positions an exact scan beats any approximate index on both recall and
	// on the amount of machinery to keep in step with every write
	// (docs/recherche/P7-similarite-knn-go.md). The contract therefore
	// promises exact nearest neighbours, not approximate ones.
	//
	// What it ranks is opts, not the whole library: a neighbour is the same
	// PROBLEM nearby, and SimilarOptions carries the class that says so.
	Similar(ctx context.Context, scope string, target *domain.Position, opts SimilarOptions) ([]SimilarPosition, error)
}

// SimilarPosition is one neighbour and how far it stands, in checker-pips: the
// amount of checker movement separating it from the position asked about.
type SimilarPosition struct {
	Position domain.Position `json:"position"`
	Distance int             `json:"distance"`
}

// SimilarOptions is the SET a similarity ranking is taken over — the target's
// equivalence class — plus how much of it comes back (ADR-0043).
//
// The distance was never the problem. Ranking the whole library WAS: measured
// on the demo library, the ten nearest of any position were the checker play
// twinning the cube decision on the same board (distance 0) and then the plies
// before and after it in the same match, because two plies are one roll and no
// other game comes that close. "Positions like this one" asks for the same
// problem met elsewhere, not for the nearest drawing.
//
// The dice, the score and the cube value are deliberately absent: they are not
// part of the class, and the ordinary search filters narrow on them when the
// user wants them to. Nothing is ever folded into the distance either — one
// unit, readable in checker-pips, is what the metric has going for it.
type SimilarOptions struct {
	// Limit is how many neighbours come back. Zero or less returns nothing:
	// a ranking of nobody is a caller's mistake, not an empty library.
	Limit int

	// MaxDistance drops every neighbour further than this many checker-pips.
	// Zero means no ceiling. A ranking whose ceiling nothing passes comes back
	// EMPTY — never padded with the least distant of the unrelated, which is
	// what a fixed count alone produces on a small library.
	MaxDistance int

	// DecisionType, when set, keeps only neighbours of that kind — a cube
	// decision and the checker play on the same board are two problems, and
	// they are two rows one pip-pip apart.
	DecisionType *int

	// Money, when set, keeps only money positions (true) or only positions at
	// a match score (false). The regime is part of the class for a CUBE
	// decision, where it changes what the position asks, and not for a checker
	// play, where the same plan of play recurs at every score.
	Money *bool
}

// The third rule of the class — "another match" — is not in SimilarOptions on
// purpose. It is never a caller's choice: the plies around the target are its
// closest structures in every match that played through it, and Similar reads
// them off the target itself. A drawn board or a position imported on its own
// belongs to no match, and then nothing is excluded.


// ClassOf returns the equivalence class of target: same kind of decision, and
// for a cube decision the same regime (ADR-0043 rule 1). It carries neither
// the limit nor the match exclusion, which are the caller's to add — the class
// is a property of the position, the other two are properties of the question.
func ClassOf(target *domain.Position) SimilarOptions {
	if target == nil {
		return SimilarOptions{}
	}
	kind := target.DecisionType
	opts := SimilarOptions{DecisionType: &kind}
	if kind == domain.CubeAction {
		money := target.IsMoney()
		opts.Money = &money
	}
	return opts
}
