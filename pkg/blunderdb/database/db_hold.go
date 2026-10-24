package database

import (
	"errors"
	"path/filepath"
	"time"
)

// ErrLibraryReplaced refuses a write whose rows were read in a library this
// handle no longer carries: their ids would name other rows of the open one.
var ErrLibraryReplaced = errors.New("the library these rows were read in is no longer the open one")

// JobWrite is a write that names rows by id, holding its library (HoldLibrary)
// so that no other library comes under the handle before it is answered.
const JobWrite = "write"

// holdLease bounds a hold its holder never released (a window reloaded in the
// middle of a write): past it, the library may be replaced again.
const holdLease = 2 * time.Minute

// HoldLibrary vets, for a write naming rows by id, that the handle still
// carries the library at path, the one its rows were read in, and holds it
// there: Open, Setup and Exchange refuse with a *BusyError until
// ReleaseLibrary(ticket). Otherwise ErrLibraryReplaced. The path names the
// library because no file is open twice and the caller knows it at once, when
// it reads the rows (ADR-0107). Close is never held off: the window may
// always quit.
func (d *Database) HoldLibrary(path string) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.path == "" || !samePath(d.path, path) {
		return 0, ErrLibraryReplaced
	}
	d.cancelMu.Lock()
	defer d.cancelMu.Unlock()
	if d.exchanging || d.replacing {
		return 0, ErrLibraryReplaced
	}
	if d.holds == nil {
		d.holds = map[int64]time.Time{}
	}
	d.holdSeq++
	d.holds[d.holdSeq] = time.Now()
	return d.holdSeq, nil
}

// ReleaseLibrary lets go of a hold HoldLibrary gave; an unknown ticket is
// ignored.
func (d *Database) ReleaseLibrary(ticket int64) {
	d.cancelMu.Lock()
	defer d.cancelMu.Unlock()
	delete(d.holds, ticket)
}

// heldLocked says whether a write holds the library; the caller holds cancelMu.
func (d *Database) heldLocked() bool {
	for ticket, at := range d.holds {
		if time.Since(at) > holdLease {
			delete(d.holds, ticket)
		}
	}
	return len(d.holds) > 0
}

// beginReplace refuses to take another file while a write holds the library;
// otherwise it marks the handle so that no hold starts until endReplace.
func (d *Database) beginReplace() error {
	d.cancelMu.Lock()
	defer d.cancelMu.Unlock()
	if d.heldLocked() {
		return &BusyError{Job: JobWrite}
	}
	d.replacing = true
	return nil
}

func (d *Database) endReplace() {
	d.cancelMu.Lock()
	d.replacing = false
	d.cancelMu.Unlock()
}

// samePath says whether two spellings name one file.
func samePath(a, b string) bool {
	if a == b {
		return true
	}
	ca, errA := filepath.Abs(a)
	cb, errB := filepath.Abs(b)
	return errA == nil && errB == nil && ca == cb
}
