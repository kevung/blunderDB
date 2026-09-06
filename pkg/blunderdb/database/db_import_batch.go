package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Import batches: what an import brought in, and how to ask afterwards
// (issue #257, fiche I.1).
//
// An import used to end on nothing — the progress bar reached the end and the
// window went back to what it was. The BATCH is what makes an end-of-import
// report possible at all: the matches an import writes point back at it, so
// the report can speak about *this import* rather than about the database.
//
// One batch is open at a time, exactly as one import is: the wrapper already
// assumes a single in-flight import (beginCancellableImport keeps one cancel
// function). A second BeginImportBatch replaces the first, and a match written
// with no batch open carries none — which is what a match saved outside an
// import should carry.

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

// FinishImportBatch closes the batch and stores the counts the import
// observed, which two parties see between them: the writing path counted the
// matches written, skipped and enriched and the positions saved; `failures`
// carries what only the CALLER sees — the files it could not read at all,
// which never reached the writing path.
//
// Everything else in the report is measured afterwards, by ImportReport.
//
// It always clears the open batch, even when storing the counts fails: a batch
// left open would silently stamp the NEXT import's matches.
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
// The measured half is recomputed on every call rather than cached. A position
// the batch brought in can be analysed afterwards, and a report still claiming
// "12 positions without analysis" once they had been analysed would be worse
// than no report at all.
//
// The PR is the reference player's when the database names one (the `user`
// metadata key, which the identity dialog writes), and both seats' otherwise.
// The report says which — a rate mixing two players is a fact about the
// import, but only if it is labelled as one.
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
