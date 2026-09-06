package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// VacuumResult reports the file-size effect of a Vacuum call, in bytes. It
// exists (rather than Vacuum returning two int64s) because Wails v2's method
// binding only understands a bound method returning exactly (value, error)
// or a single value — a third return value is silently dropped at the JS
// boundary, not just left untyped, so a (before, after int64, error) shape
// would work from Go/CLI callers but hand the GUI button back nil, nil
// unconditionally. See IndividualSaveResult for the same reason repeated
// elsewhere in this package. It stays a type of this package (rather than an
// alias of storage.VacuumResult) so the generated Wails model keeps its
// namespace.
type VacuumResult struct {
	SizeBefore int64
	SizeAfter  int64
	// TrashPurged is how many trash entries the run dropped for being older
	// than domain.TrashRetentionDays (#285, ADR-0036). The trash is purged
	// HERE and nowhere else: nothing empties it on open, so a user who does
	// not vacuum keeps everything they deleted, and one who does gets the
	// space back — which is what "reclaims disk space left behind by
	// deletions" already meant.
	TrashPurged int
}

// Vacuum reclaims disk space left behind by deletions (matches, tournaments,
// purges) that SQLite never shrinks the file for on its own. It is an
// explicit, user-triggered action only — never run automatically at open,
// since its cost is unpredictable on a large database.
//
// The mechanics (WAL checkpoint, free-space guard, VACUUM outside a
// transaction, trailing ANALYZE, second checkpoint) live on the SQLite
// backend, sqlite.Storage.Vacuum, so the GUI button, the CLI's `vacuum` and
// the daemon's /ops/maintenance.vacuum run one implementation. This wrapper
// only takes the desktop lock: VACUUM rewrites the whole file and must not
// overlap a concurrent reader of this Database.
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
