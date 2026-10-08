package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// PlayedActions is the checker move and cube action played at a decision, as
// the caller knows them. An empty field is a known "nothing played".
type PlayedActions struct {
	CheckerMove string
	CubeAction  string
	// Position is the position the decision was made in, when the caller
	// holds it; nil otherwise. Whether a checker play was forced depends on
	// its legal plays, which the store then counts without reading the row.
	Position *domain.Position
}

// AnalysisRecord is one stored analysis with the position it belongs to.
type AnalysisRecord struct {
	PositionID int64
	Analysis   *domain.PositionAnalysis
}

// AnalysisStore persists the engine analysis attached to a position. The
// backend transparently compresses/decompresses the analysis payload; callers
// always see a decoded *domain.PositionAnalysis.
type AnalysisStore interface {
	// Save stores (or replaces) the analysis for positionID.
	Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error

	// Merge reads the stored analysis of positionID once, hands it to merge
	// (nil when there is none) and saves what merge returns, rounded and
	// encoded as Save would. When the result equals the stored analysis apart
	// from LastModifiedDate, and the played-action columns derived from it are
	// unchanged, nothing is written: re-importing what is already there costs
	// a read, not a rewrite. merge may mutate its argument; returning nil
	// writes nothing. Reports whether a row was written.
	//
	// played is what the caller already knows was played at the position —
	// the decision of the match being imported. The columns take it where the
	// analysis names no played action, without reading the move table; nil
	// (a position imported on its own) reads the match's record as Save does.
	Merge(ctx context.Context, scope string, positionID int64, played *PlayedActions, merge func(existing *domain.PositionAnalysis) *domain.PositionAnalysis) (bool, error)

	// LoadMany decodes the analyses of the given positions, keyed by position
	// id, in one round trip per batch. A position without an analysis has no
	// entry. A stored payload that cannot be decoded has no entry either, and
	// is logged: one corrupt row must not block reading everything else, and
	// the caller of a batch — an export, a listing — has nothing to do about
	// it but leave it out.
	LoadMany(ctx context.Context, scope string, ids []int64) (map[int64]*domain.PositionAnalysis, error)

	// Load returns the analysis for positionID, or ErrNotFound.
	Load(ctx context.Context, scope string, positionID int64) (*domain.PositionAnalysis, error)

	// Delete removes the analysis for positionID.
	Delete(ctx context.Context, scope string, positionID int64) error

	// RepairDenormalisedColumns recomputes the scalar columns of every analysis
	// in scope from its stored JSON, and returns how many rows actually changed.
	//
	// The columns are a projection of `data`, which stays intact, so a bug in
	// the projection is repairable without re-importing anything.
	//
	// An explicit operation, NOT a schema migration: opening a database must
	// not rewrite its analysis columns behind the user's back.
	RepairDenormalisedColumns(ctx context.Context, scope string) (int, error)

	// WithoutAnalysis streams the positions in scope that carry no analysis at
	// all, by ascending id, bounded by opts.
	//
	// One join instead of a Load per position (the daemon's catch-up sweep).
	//
	// A stream, not a snapshot. The caller must NOT write analyses while
	// reading it, or the result would depend on the backend's isolation: drain
	// it first. A fresh call finds whatever is still missing (ADR-0013 resume).
	WithoutAnalysis(ctx context.Context, scope string, opts ListOpts) iter.Seq2[*domain.Position, error]

	// WithEngine streams, by ascending position id, the analyses whose
	// provenance column (analysis_engine) starts with enginePrefix or is not
	// derived yet (NULL). Rows of any other engine are skipped on the column
	// alone, their payload never decoded: a corpus of XG analyses costs the
	// read of one column per row, not a decompression and a JSON decode per row.
	//
	// The column describes the entry the verdict is read from, not every entry
	// of the blob: the caller confirms on the decoded analysis whatever needs
	// all of them (gammonnet.IsStaleAnalysis). A payload that cannot be decoded
	// is logged and left out, as in LoadMany.
	//
	// Pulled by keyset batches with no connection or lock held between two
	// yields, so the caller may take as long as it likes per record and holds
	// no list of ids in memory.
	WithEngine(ctx context.Context, scope, enginePrefix string) iter.Seq2[AnalysisRecord, error]

	// ReencodeAnalyses rewrites in the current binary format (ADR-0070) at
	// most limit analyses past the position id after whose blob is still in a
	// legacy format, in id order, and returns the position id of the last row
	// it examined (0 when none is left) and how many it rewrote. A blob that
	// does not decode is left as it is. It is the resumable pass that
	// re-encodes a library older than the format: the caller loops from 0 on
	// the id it gets back, and a pass interrupted and restarted skips, in
	// SQL, what it already rewrote.
	//
	// An explicit operation, NOT a schema migration: the format is told by
	// the blob's header, so a legacy row reads as well as a binary one.
	ReencodeAnalyses(ctx context.Context, scope string, after int64, limit int) (next int64, rewritten int, err error)
}

// ReencodeBatchSize bounds the rows one ReencodeAnalyses call rewrites, in
// one transaction, as compaction's batches do.
const ReencodeBatchSize = 2000

// ReencodeAllAnalyses runs the AnalysisStore.ReencodeAnalyses pass over the
// whole scope and returns how many blobs it rewrote. before, when non-nil,
// runs around each batch (the GUI's wrapper takes its lock there); it returns
// the function that ends the batch.
func ReencodeAllAnalyses(ctx context.Context, store AnalysisStore, scope string, before func() func()) (int, error) {
	var next int64
	total := 0
	for {
		done := func() {}
		if before != nil {
			done = before()
		}
		n, k, err := store.ReencodeAnalyses(ctx, scope, next, ReencodeBatchSize)
		done()
		if err != nil {
			return total, err
		}
		total += k
		if n == 0 {
			return total, nil
		}
		next = n
	}
}
