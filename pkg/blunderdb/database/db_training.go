package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Training journal, GUI-facing (ADR-0040 rule 6): an adapter that takes
// d.mu and delegates to storage (storage/sqlshared/training.go) with the ""
// scope. Storage types on purpose: Wails generates the frontend model from
// them, and a mirror type would declare the journal twice.

// SaveTrainingSession records a finished session and its items, and returns
// the new session's id. There is no update: an editable record measures
// nothing.
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
