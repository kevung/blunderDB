package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
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
	// OnBulk is called once, before any file is written, when the list is
	// large enough for the bulk mode; unsafe tells that synchronous writes are
	// off (an empty database: a cut means importing again).
	OnBulk func(unsafe bool)
	// OnProgress receives the run's progress at most four times a second,
	// and once more at the end (Done set).
	OnProgress func(ingest.BatchProgress)
}

// bulkImportMinFiles is the list size from which ImportFiles writes in bulk
// mode: a larger page cache, rare checkpoints, and on an empty database no
// secondary index until the end and no synchronous writes. Below it, the
// cost of rebuilding indexes and the risk outweigh the gain.
var bulkImportMinFiles = 200

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
	var store ingest.TxBeginner = d.store
	batchID, sqlDB := d.importBatchID, d.db
	journal := d.importJournal
	batches := d.store.ImportBatches()
	d.mu.RUnlock()

	// An in-memory database has one connection: a dedicated bulk connection
	// would leave none to anyone else.
	if len(paths) >= bulkImportMinFiles && sqlDB.Stats().MaxOpenConnections != 1 {
		var held bool
		if err := sqlDB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM position)`).Scan(&held); err != nil {
			return nil, err
		}
		fresh := !held
		sess, err := sqlite.BeginBulk(ctx, sqlDB, sqlite.BulkOptions{Unsafe: fresh, DropIndexes: fresh})
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := sess.Close(context.Background()); err != nil {
				slog.Error("closing the bulk import session", "err", err)
			}
		}()
		store = sess
		if opts.OnBulk != nil {
			opts.OnBulk(fresh)
		}
	}

	onRead, onFile := opts.OnRead, opts.OnFile
	if opts.OnProgress != nil {
		var total int64
		for _, p := range paths {
			if fi, err := os.Stat(p); err == nil {
				total += fi.Size()
			}
		}
		meter := ingest.NewProgressMeter(len(paths), total, ingest.ProgressInterval, opts.OnProgress)
		defer meter.Finish()
		onRead = func(size int64) {
			meter.Read(size)
			if opts.OnRead != nil {
				opts.OnRead(size)
			}
		}
		onFile = func(o ingest.FileOutcome) {
			meter.File(o)
			if opts.OnFile != nil {
				opts.OnFile(o)
			}
		}
	}

	popts := ingest.PipelineOptions{
		Workers:       opts.Workers,
		FilesPerTx:    opts.FilesPerTx,
		ImportBatchID: batchID,
		Lock: func() func() {
			d.mu.Lock()
			return d.mu.Unlock
		},
		OnCommit: func(group []ingest.FileOutcome) {
			ingest.RecordOutcomes(context.Background(), batches, "", batchID, group)
			for _, o := range group {
				if o.Status == ingest.FileFailed {
					// The journal is where the full list of refused files
					// lives: the GUI shows the first hundred.
					slog.Warn("import: file refused", "file", o.Path, "err", o.Error)
				}
				d.countImported(o)
				if onFile != nil {
					onFile(o)
				}
			}
		},
		OnRead: onRead,
	}
	if journal != nil {
		popts.Known = journal.Known
	}
	out, err := ingest.ImportFiles(ctx, store, paths, popts)
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
