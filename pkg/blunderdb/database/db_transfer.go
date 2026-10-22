package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// transferMu serialises transfers between open libraries. A transfer holds two
// Database locks at once — the source's for reading, the destination's for
// writing — and every other method takes one: with transfers taken one at a
// time, A → B and B → A cannot each hold the lock the other waits for.
var transferMu sync.Mutex

// TransferTo copies what opts selects in this library into dst (ADR-0102):
// the export's selection, merged by the native import, nothing written that
// says where the copy came from. It is the CLI's `transfer` and the desktop's
// transfer between open libraries; the daemon's is ingest.Transfer between two
// tenants.
func (d *Database) TransferTo(ctx context.Context, dst *Database, opts ExportOptions) (ingest.TransferReport, error) {
	return d.transferWith(ctx, dst, opts, nil, nil)
}

// transferWith is TransferTo with two hooks run while both libraries are
// locked: before ahead of the copy, after once it is written, with its
// report. The locks keep out the Database methods of either library, not a
// borrowed handle or another Database on the same file.
func (d *Database) transferWith(ctx context.Context, dst *Database, opts ExportOptions, before func(ctx context.Context) error, after func(ctx context.Context, rep ingest.TransferReport) error) (ingest.TransferReport, error) {
	if dst == nil || d == dst {
		return ingest.TransferReport{}, fmt.Errorf("%w: %w", storage.ErrInvalid, ingest.ErrSelfTransfer)
	}
	if open := dst.openPath.Load(); open != nil && d.IsOpenDatabase(*open) {
		return ingest.TransferReport{}, fmt.Errorf("%w: %w", storage.ErrInvalid, ingest.ErrSelfTransfer)
	}
	// The source counts it too: a vacuum would close the pool the export
	// reads through.
	release, err := d.holdForRead()
	if err != nil {
		return ingest.TransferReport{}, err
	}
	defer release()
	// The destination counts the transfer as one of its imports: a vacuum of
	// it waits, and its CancelImport stops the transfer.
	imported, done := dst.beginCancellableImport()
	defer done()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer context.AfterFunc(imported, cancel)()
	transferMu.Lock()
	defer transferMu.Unlock()
	d.mu.RLock()
	defer d.mu.RUnlock()
	dst.mu.Lock()
	defer dst.mu.Unlock()
	if d.db == nil || dst.db == nil {
		return ingest.TransferReport{}, fmt.Errorf("no database is currently open")
	}
	if before != nil {
		if err := before(ctx); err != nil {
			return ingest.TransferReport{}, err
		}
	}
	rep, err := ingest.Transfer(ctx, ingest.TransferEnd{S: d.store}, ingest.TransferEnd{S: dst.store}, opts, nil)
	if err != nil {
		return rep, err
	}
	// The rows written above carry their blob alone; derive their provenance
	// now rather than on the next open, as an import of the file would.
	if err := dst.backfillAnalysisProvenance(ctx); err != nil {
		return rep, err
	}
	if after != nil {
		return rep, after(ctx, rep)
	}
	return rep, nil
}

// holdForRead counts a reader of d's library among its imports, so that a
// vacuum is refused while it reads; it is refused itself while one runs.
func (d *Database) holdForRead() (func(), error) {
	d.cancelMu.Lock()
	defer d.cancelMu.Unlock()
	if d.exchanging {
		return nil, &BusyError{Job: JobExchange}
	}
	g := d.gateLocked()
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.job != nil && g.job.name == JobVacuum {
		return nil, &BusyError{Job: JobVacuum}
	}
	g.imports++
	return func() {
		g.mu.Lock()
		g.imports--
		g.mu.Unlock()
	}, nil
}
