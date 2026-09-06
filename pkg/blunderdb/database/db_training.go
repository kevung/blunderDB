package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Training journal, GUI-facing (issue #320, ADR-0040 rule 6).
//
// Like db_session.go, this file is an adapter and nothing else: it takes the
// wrapper's lock and delegates to the Storage backend under the single
// implicit tenant ("" scope). The SQL lives in storage/sqlshared/training.go
// and is held to the shared contract suite, so the desktop, the CLI and the
// daemon record and read the same journal — CLI/GUI/server parity, even
// though v1 gives the CLI and the daemon no command of their own for it.
//
// The signatures take and return storage types on purpose: Wails generates
// the frontend model from them (models.ts `storage` namespace), and a mirror
// type in this package would be a second declaration of the same journal.

// SaveTrainingSession records a finished session and its items, and returns
// the new session's id. Nothing else writes to the journal: « Quitter »
// discards a session, and there is no update — a record that can be edited
// measures nothing.
func (d *Database) SaveTrainingSession(session storage.TrainingSession) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return 0, errNotOpened
	}
	return d.store.Training().Save(context.Background(), "", session)
}

// LoadTrainingSessions returns the recorded sessions of one exercise, most
// recent first. An empty exercise means every exercise; a limit of zero means
// no bound — the journal has no cap (ADR-0040 rule 6).
func (d *Database) LoadTrainingSessions(exercise string, limit int) ([]storage.TrainingSession, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, errNotOpened
	}
	return d.store.Training().Sessions(context.Background(), "", exercise, limit)
}

// LoadTrainingNumberStats returns one exercise's items aggregated by number
// type — the detail the summary unfolds into.
func (d *Database) LoadTrainingNumberStats(exercise string) ([]storage.TrainingNumberStat, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, errNotOpened
	}
	return d.store.Training().NumberStats(context.Background(), "", exercise)
}
