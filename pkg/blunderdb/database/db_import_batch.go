package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Import batches: the matches an import writes point back at its batch, so
// the end-of-import report speaks about *this import*. One batch is open at a
// time, like one import; a second BeginImportBatch replaces the first, and a
// match written outside any batch carries none.

// BeginImportBatch opens a batch for an import that is starting and returns
// its id. Every match written until FinishImportBatch is stamped with it.
//
// source is shown to the user verbatim — a file path, a folder — and format is
// the import format, or "mixed" for a folder holding several.
func (d *Database) BeginImportBatch(source, format string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	id, err := d.store.ImportBatches().Begin(context.Background(), "", source, format)
	if err != nil {
		return 0, err
	}
	d.importBatchID = id
	d.importBatchCounts = domain.ImportReport{}
	return id, nil
}

// FinishImportBatch closes the batch and stores its counts: the writing
// path's, plus `failures`, the unreadable files only the CALLER sees. It
// always clears the open batch, even on error, or it would stamp the NEXT
// import's matches.
func (d *Database) FinishImportBatch(batchID int64, failures domain.ImportReport) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	counts := d.importBatchCounts
	counts.FilesFailed = failures.FilesFailed
	counts.Failures = failures.Failures
	if d.importBatchID == batchID {
		d.importBatchID = 0
		d.importBatchCounts = domain.ImportReport{}
	}
	return d.store.ImportBatches().Finish(context.Background(), "", batchID, counts)
}

// ImportReport returns what a batch brought in: the counts the import stored,
// completed by what can be measured over its matches now — positions the
// source tool had flagged, positions no engine has judged, the batch's own PR
// and its worst decisions.
//
// The measured half is recomputed on every call, since positions can be
// analysed afterwards. The PR is the reference player's (`user` metadata) when
// set, both seats' otherwise, and the report says which.
func (d *Database) ImportReport(batchID int64) (*domain.ImportBatch, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	ctx := context.Background()
	var players []string
	if meta, err := d.store.Metadata().Load(ctx, ""); err == nil {
		if user := meta["user"]; user != "" {
			players = append(players, user)
		}
	}
	return d.store.ImportBatches().Report(ctx, "", batchID, players)
}

// ImportStudyQueue returns the batch's positions worth a second look, in the
// order they should be walked: what cost something, then what the source tool
// had marked, then the close cube decisions. Nothing records that a position
// was seen. Same reference-player convention as ImportReport.
func (d *Database) ImportStudyQueue(batchID int64, limit int) ([]domain.StudyQueueEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	ctx := context.Background()
	var players []string
	if meta, err := d.store.Metadata().Load(ctx, ""); err == nil {
		if user := meta["user"]; user != "" {
			players = append(players, user)
		}
	}
	return d.store.ImportBatches().StudyQueue(ctx, "", batchID, players, limit)
}

// ListImportBatches returns the recorded batches, most recent first, with the
// counts stored at the end of each import (not the measured half — that is
// ImportReport, one batch at a time).
func (d *Database) ListImportBatches(limit, offset int) ([]*domain.ImportBatch, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.ImportBatches().List(context.Background(), "", storage.ListOpts{Limit: limit, Offset: offset})
}
