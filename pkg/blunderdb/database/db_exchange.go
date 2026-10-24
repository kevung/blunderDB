package database

// OpenPath returns the file d has open, "" when none. A function, not a
// method, so the Wails binding of *Database does not grow; the desktop's
// library groups compare it to refuse one file in two groups.
func OpenPath(d *Database) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.path
}

// Exchange swaps the open library of d with other's (a function, not a
// method, so the frontend cannot reach it through the binding): the connection, its
// Storage, the single-writer lock, the read-only flag, the path, and the
// sessions (transcription drafts, Duels) opened on that library. What belongs
// to the handle rather than to the file stays: the before-switch hook, the
// comment author, the duplicate policy, the direction memory.
//
// It is how the desktop holds several libraries while one *Database stays
// bound to Wails: the bound handle always carries the active library and the
// others wait, open, in handles of their own. No file is opened or closed,
// so a switch costs no I/O.
//
// d's before-switch hook runs first, as for an Open. Both generations move,
// so a job started on either side writes nothing after the exchange. An
// import or a job running on either handle refuses the exchange with a
// *BusyError: its writes go through the handle, which is about to carry the
// other file. A job on a handle Borrow lent keeps its library: the job slot
// moves with the library, the borrowed handle stays on it.
func Exchange(d, other *Database) error {
	if other == d {
		return nil
	}
	if err := d.idleForExchange(); err != nil {
		return err
	}
	defer d.endExchange()
	if err := other.idleForExchange(); err != nil {
		return err
	}
	defer other.endExchange()
	d.switchFile()
	d.CancelSearch()
	other.CancelSearch()

	d.transcriptMu.Lock()
	other.transcriptMu.Lock()
	d.transcriptSvc, other.transcriptSvc = other.transcriptSvc, d.transcriptSvc
	d.transcriptOn, other.transcriptOn = other.transcriptOn, d.transcriptOn
	other.transcriptMu.Unlock()
	d.transcriptMu.Unlock()

	d.duelMu.Lock()
	other.duelMu.Lock()
	d.duelSvc, other.duelSvc = other.duelSvc, d.duelSvc
	d.duelOn, other.duelOn = other.duelOn, d.duelOn
	d.duelBoard, other.duelBoard = other.duelBoard, d.duelBoard
	other.duelMu.Unlock()
	d.duelMu.Unlock()

	d.mu.Lock()
	other.mu.Lock()
	defer d.mu.Unlock()
	defer other.mu.Unlock()
	d.db, other.db = other.db, d.db
	d.store, other.store = other.store, d.store
	d.lock, other.lock = other.lock, d.lock
	d.readOnly, other.readOnly = other.readOnly, d.readOnly
	d.path, other.path = other.path, d.path
	dOpen := d.openPath.Load()
	d.openPath.Store(other.openPath.Load())
	other.openPath.Store(dOpen)
	d.positionsSinceStats, other.positionsSinceStats = other.positionsSinceStats, d.positionsSinceStats
	d.pendingPhaseBackfill, other.pendingPhaseBackfill = other.pendingPhaseBackfill, d.pendingPhaseBackfill
	d.pendingAnkiCardKinds, other.pendingAnkiCardKinds = other.pendingAnkiCardKinds, d.pendingAnkiCardKinds
	d.cancelMu.Lock()
	other.cancelMu.Lock()
	d.gate, other.gate = other.gate, d.gate
	other.cancelMu.Unlock()
	d.cancelMu.Unlock()
	d.bumpGeneration()
	other.bumpGeneration()
	d.storeEpoch++
	other.storeEpoch++
	return nil
}

// idleForExchange refuses while an import, a migration or a job holds d;
// otherwise it marks d as exchanging, under the same lock, so that no job or
// import starts between this check and the swap. endExchange clears the mark.
func (d *Database) idleForExchange() error {
	d.cancelMu.Lock()
	defer d.cancelMu.Unlock()
	if d.exchanging {
		return &BusyError{Job: JobExchange}
	}
	g := d.gateLocked()
	g.mu.Lock()
	held := g.job
	g.mu.Unlock()
	if held != nil && held.by == d {
		// A job on a borrowed handle keeps its library through the swap; one
		// on this handle would find the other file under it.
		return &BusyError{Job: held.name}
	}
	if d.importCancel != nil {
		return &BusyError{Job: "import"}
	}
	if d.heldLocked() {
		return &BusyError{Job: JobWrite}
	}
	d.exchanging = true
	return nil
}

func (d *Database) endExchange() {
	d.cancelMu.Lock()
	d.exchanging = false
	d.cancelMu.Unlock()
}
