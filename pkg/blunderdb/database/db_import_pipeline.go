package database

import (
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ImportFilesOptions tunes Database.ImportFiles; the zero value is usable.
type ImportFilesOptions struct {
	// Workers reading files in parallel; 0 means one per CPU.
	Workers int
	// FilesPerTx groups that many files per transaction; 0 means the default.
	FilesPerTx int
	// OnFile is called once per decided file, in file order, after its
	// transaction committed. It runs with d.mu held: it must not call back
	// into the Database.
	OnFile func(ingest.FileOutcome)
	// OnRead is called from reader goroutines with each file's size once read.
	OnRead func(size int64)
}

// ImportFiles imports a list of match and position files through the
// parallel pipeline (ingest.ImportFiles): readers in parallel, one writer, d.mu
// held only around each group's transaction so the rest of the application
// keeps reading while files are parsed. Matches are stamped with the open
// import batch, and the batch counts follow what each group committed.
// CancelImport stops it; groups already committed stay.
func (d *Database) ImportFiles(paths []string, opts ImportFilesOptions) ([]ingest.FileOutcome, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("no database is currently open")
	}
	store, batchID := d.store, d.importBatchID
	d.mu.RUnlock()

	out, err := ingest.ImportFiles(ctx, store, paths, ingest.PipelineOptions{
		Workers:       opts.Workers,
		FilesPerTx:    opts.FilesPerTx,
		ImportBatchID: batchID,
		Lock: func() func() {
			d.mu.Lock()
			return d.mu.Unlock
		},
		OnCommit: func(group []ingest.FileOutcome) {
			for _, o := range group {
				d.countImported(o)
				if opts.OnFile != nil {
					opts.OnFile(o)
				}
			}
		},
		OnRead: opts.OnRead,
	})
	slog.Info("imported files", "files", len(out), "of", len(paths), "err", err)
	return out, err
}

// countImported adds one committed file to the open batch's counts, as
// writeImportedMatch does for a single file. Callers hold d.mu.
func (d *Database) countImported(o ingest.FileOutcome) {
	switch o.Status {
	case ingest.FileImported:
		d.importBatchCounts.MatchesImported++
	case ingest.FileEnriched:
		d.importBatchCounts.MatchesEnriched++
	case ingest.FileDuplicate:
		if o.MatchID != 0 {
			d.importBatchCounts.MatchesSkipped++
		}
	}
	if o.Status == ingest.FileImported || o.Status == ingest.FileEnriched {
		d.importBatchCounts.PositionsSaved += o.Positions
	}
	d.positionsSinceStats += o.Positions
}
