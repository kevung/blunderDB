package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// VacuumResult reports the file-size effect of a Vacuum call, in bytes. A
// struct because Wails v2 binds at most (value, error): a third return value
// reaches JS as nil. Not an alias of storage.VacuumResult, so the generated
// Wails model keeps its namespace.
type VacuumResult struct {
	SizeBefore int64
	SizeAfter  int64
	// TrashPurged is how many trash entries the run dropped for being older
	// than domain.TrashRetentionDays (ADR-0036). The trash is purged HERE and
	// nowhere else, never on open.
	TrashPurged int
}

// Vacuum reclaims disk space left behind by deletions (matches, tournaments,
// purges) that SQLite never shrinks the file for on its own. It is an
// explicit, user-triggered action only — never run automatically at open,
// since its cost is unpredictable on a large database.
//
// The mechanics live in sqlite.Storage.Vacuum, shared by every mode. This
// wrapper takes d.mu exclusively: VACUUM rewrites the whole file and must not
// overlap a reader.
func (d *Database) Vacuum() (VacuumResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return VacuumResult{}, fmt.Errorf("vacuum: no database open")
	}
	ctx := context.Background()
	// Purge before compacting, so the space the purge frees is space this
	// VACUUM reclaims rather than space the next one will.
	purged, err := d.store.Trash().Purge(ctx, "", domain.TrashRetentionDays)
	if err != nil {
		return VacuumResult{}, err
	}
	res, err := d.store.Vacuum(ctx)
	return VacuumResult{SizeBefore: res.SizeBefore, SizeAfter: res.SizeAfter, TrashPurged: purged}, err
}
