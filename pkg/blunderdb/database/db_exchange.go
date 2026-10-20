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
// import or a job holding either library refuses the exchange with a
// *BusyError: its writes go through the handle, which is about to carry the
// other file.
func Exchange(d, other *Database) error {
	if other == d {
		return nil
	}
	if err := d.idleForExchange(); err != nil {
		return err
	}
	if err := other.idleForExchange(); err != nil {
		return err
	}
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
	d.positionsSinceStats, other.positionsSinceStats = other.positionsSinceStats, d.positionsSinceStats
	d.pendingPhaseBackfill, other.pendingPhaseBackfill = other.pendingPhaseBackfill, d.pendingPhaseBackfill
	d.pendingAnkiCardKinds, other.pendingAnkiCardKinds = other.pendingAnkiCardKinds, d.pendingAnkiCardKinds
	d.bumpGeneration()
	other.bumpGeneration()
	d.storeEpoch++
	other.storeEpoch++
	return nil
}

// idleForExchange refuses while an import, a migration or a job holds d.
func (d *Database) idleForExchange() error {
	d.cancelMu.Lock()
	defer d.cancelMu.Unlock()
	if d.job != nil {
		return &BusyError{Job: d.job.name}
	}
	if d.importCancel != nil {
		return &BusyError{Job: "import"}
	}
	return nil
}
