package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// ImportBatchStore persists the import batches — one row per import the user
// launched — and measures what each of them brought in (issue #257).
//
// The counts an import observes as it runs (matches imported, skipped,
// enriched, files that failed) are written by the importer, because only it
// sees them. Everything else in the report is MEASURED afterwards, over the
// batch's matches: flagged positions, positions without analysis, the error
// rate, the worst decisions. That split is deliberate — a report built from
// measurement can be recomputed from a batch id alone, and does not have to be
// trusted to have been written correctly at a moment when the import may have
// been cancelled halfway.
type ImportBatchStore interface {
	// Begin opens a batch for an import that is starting and returns its id.
	// source is shown to the user verbatim (a file path, a folder); format is
	// the import format, or "mixed" for a folder holding several.
	Begin(ctx context.Context, scope string, source, format string) (int64, error)

	// Finish stamps the batch as done and stores the counts the import
	// observed. It does not measure anything: Report does that.
	Finish(ctx context.Context, scope string, batchID int64, counts domain.ImportReport) error

	// Load returns a batch with its stored counts, or ErrNotFound.
	Load(ctx context.Context, scope string, batchID int64) (*domain.ImportBatch, error)

	// List returns the batches, most recent first, bounded by opts.
	List(ctx context.Context, scope string, opts ListOpts) ([]*domain.ImportBatch, error)

	// Report returns the batch's stored counts completed by what can be
	// measured over its matches now: flagged positions, positions no engine
	// has judged, the batch's own error rate, and its worst decisions.
	//
	// It is recomputed on every call rather than cached: a position the batch
	// brought in can be analysed afterwards, and a report that still claimed
	// "12 positions without analysis" once they had been analysed would be
	// worse than no report.
	//
	// players names whose decisions the error rate is about — the reference
	// player and their alternate spellings, as the statistics take them. Empty
	// scores BOTH seats, which is the honest answer when nothing says who the
	// user is: an error rate mixing two players is still a fact about the
	// import, as long as the report says so (ImportReport.Player).
	Report(ctx context.Context, scope string, batchID int64, players []string) (*domain.ImportBatch, error)
}
