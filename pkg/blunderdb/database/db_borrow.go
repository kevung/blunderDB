package database

import (
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Borrow lends d's open library to a background job: a handle of its own on
// the same connection and Storage, sharing the library's job slot. It stays on
// that library when d is exchanged for another (Exchange), so a job started in
// one library group of the desktop keeps writing its library while the bound
// handle shows the next one.
//
// The borrowed handle owns nothing: no file lock, no connection to close. Its
// Close only lets the library go. The lender keeps every duty: whoever closes
// or replaces the library stops the borrowers' jobs first, or their next write
// fails on a closed connection — never on another file, since the borrowed
// handle holds this library's connection itself, not d.
//
// A function, not a method, so the Wails binding of *Database does not grow.
func Borrow(d *Database) (*Database, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, errors.New("no database is currently open")
	}
	d.cancelMu.Lock()
	exchanging := d.exchanging
	gate := d.gateLocked()
	d.cancelMu.Unlock()
	if exchanging {
		return nil, &BusyError{Job: JobExchange}
	}
	b := &Database{
		db:            d.db,
		store:         d.store,
		path:          d.path,
		readOnly:      d.readOnly,
		commentAuthor: d.commentAuthor,
		storeEpoch:    d.storeEpoch,
		gate:          gate,
		borrowed:      true,
		watchWrites:   d.watchWrites,
	}
	b.openPath.Store(d.openPath.Load())
	b.skipDuplicates.Store(d.skipDuplicates.Load())
	return b, nil
}

// HandOverImportBatch moves the import batch open on from (BeginImportBatch,
// ResumeImportBatch) to to, a handle on the same library: the matches to
// writes are stamped with it, and to's FinishImportBatch closes it with the
// counts of both. from is left with no batch open.
func HandOverImportBatch(from, to *Database) {
	from.mu.Lock()
	id, counts, journal := from.importBatchID, from.importBatchCounts, from.importJournal
	from.importBatchID, from.importBatchCounts, from.importJournal = 0, domain.ImportReport{}, nil
	from.mu.Unlock()
	to.mu.Lock()
	to.importBatchID, to.importBatchCounts, to.importJournal = id, counts, journal
	to.mu.Unlock()
}

// OpenImportBatch returns the import batch open on d, 0 when none.
func OpenImportBatch(d *Database) int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.importBatchID
}
