package sqlshared

import (
	"context"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The blunder count reads every analysis, so it is cached in the library's
// settings table (LibrarySettingsTable: metadata on a single-library SQLite
// file, the tenant-scoped library_settings on PostgreSQL, under RLS and purged
// with the tenant) as the row storage.LibrarySettingBlunderCountKey, value
// "<threshold>:<count>", valid only at the threshold it names.
//
// Triggers on BlunderCountTriggerTables delete the row on every write that
// could change the count, whatever path the write takes. A present row is
// therefore never stale — provided the triggers exist, which the reader checks
// before believing it.

// BlunderCountTriggerTables are the tables whose writes can change the blunder
// count; the schema puts one trigger per write kind on each.
var BlunderCountTriggerTables = []string{"position", "analysis", "move", "action_label"}

// cachedOrRecountedBlunders answers from the cache when it holds for the
// library's threshold, and otherwise counts and caches.
//
// The count and its caching share one transaction, so that no write can land
// between them unseen. On SQLite a write committed after the count began makes
// the upsert fail (the transaction's snapshot is stale). On PostgreSQL every
// trigger holds a shared advisory lock on the tenant until its transaction
// ends (blunder_count_share_lock): taking it exclusively here means no write
// is in flight, and every later write waits for this commit and then deletes
// the row. When the lock is taken, the count is returned uncached.
func cachedOrRecountedBlunders(ctx context.Context, db Execer, scope string) (int, error) {
	settings, err := librarySettings(ctx, db, scope)
	if err != nil {
		return 0, err
	}
	threshold := settings.BlunderThresholdMP
	if n, ok := cachedBlunderCount(ctx, db, scope, threshold); ok {
		return n, nil
	}
	if RefusesWrites(ctx, db) || !blunderCountTriggersInPlace(ctx, db) {
		return recountBlunders(ctx, db, scope, threshold)
	}

	n, counted := 0, false
	err = db.Transact(ctx, func(tx Execer) error {
		lockedOut := false
		if tx.Name() == "postgres" {
			var got bool
			if err := tx.QueryRow(ctx, `SELECT blunder_count_try_lock(?)`, tx.ScopeArg(scope)).Scan(&got); err != nil {
				return err
			}
			lockedOut = !got
		}
		c, err := recountBlunders(ctx, tx, scope, threshold)
		if err != nil {
			return err
		}
		n, counted = c, true
		if lockedOut {
			return nil
		}
		query, args := blunderCountUpsert(tx, scope, strconv.Itoa(threshold)+":"+strconv.Itoa(c))
		_, err = tx.Exec(ctx, query, args...)
		return err
	})
	if counted {
		return n, nil
	}
	return 0, err
}

// blunderCountWhere selects the scope's cache row in its table.
func blunderCountWhere(db Dialect, scope string) (table, where string, args []any) {
	table, scoped := db.LibrarySettingsTable()
	if scoped {
		return table, db.ScopeColumn() + ` = ? AND key = ?`, []any{db.ScopeArg(scope), storage.LibrarySettingBlunderCountKey}
	}
	return table, `key = ?`, []any{storage.LibrarySettingBlunderCountKey}
}

// blunderCountUpsert writes the scope's cache row.
func blunderCountUpsert(db Dialect, scope, value string) (string, []any) {
	table, scoped := db.LibrarySettingsTable()
	cols, conflict := []string{"key", "value"}, []string{"key"}
	args := []any{storage.LibrarySettingBlunderCountKey, value}
	if scoped {
		cols = append([]string{db.ScopeColumn()}, cols...)
		conflict = append([]string{db.ScopeColumn()}, conflict...)
		args = append([]any{db.ScopeArg(scope)}, args...)
	}
	return `INSERT INTO ` + table + ` (` + strings.Join(cols, ", ") + `) VALUES (` +
		strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",") + `)
		ON CONFLICT (` + strings.Join(conflict, ", ") + `) DO UPDATE SET value = EXCLUDED.value`, args
}

// cachedBlunderCount reads the cached count, valid only at the threshold it
// was counted at and only while the triggers that invalidate it exist: a
// database whose triggers were lost (a table rebuilt by hand, an older
// binary's schema) gets no answer from the cache rather than a stale one.
func cachedBlunderCount(ctx context.Context, db Execer, scope string, threshold int) (int, bool) {
	table, where, args := blunderCountWhere(db, scope)
	var v string
	if err := db.QueryRow(ctx, `SELECT COALESCE(value,'') FROM `+table+` WHERE `+where, args...).Scan(&v); err != nil {
		return 0, false
	}
	at, count, ok := strings.Cut(v, ":")
	if !ok || at != strconv.Itoa(threshold) {
		return 0, false
	}
	n, err := strconv.Atoi(count)
	if err != nil || n < 0 || !blunderCountTriggersInPlace(ctx, db) {
		return 0, false
	}
	return n, true
}

// blunderCountTriggersInPlace reports whether every invalidating trigger
// exists on the tables the queries name: three per table (insert, update,
// delete), plus on PostgreSQL the one that refuses TRUNCATE, all on the
// tables of the current schema.
func blunderCountTriggersInPlace(ctx context.Context, db Execer) bool {
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name LIKE 'blunder_count_forget_%'`
	want := 3 * len(BlunderCountTriggerTables)
	if db.Name() == "postgres" {
		query = `SELECT COUNT(*) FROM pg_trigger t JOIN pg_class c ON c.oid = t.tgrelid
			WHERE NOT t.tgisinternal AND t.tgname LIKE 'blunder_count_forget_%'
			  AND c.relnamespace = current_schema()::regnamespace`
		want = 4 * len(BlunderCountTriggerTables)
	}
	var n int
	if err := db.QueryRow(ctx, query).Scan(&n); err != nil {
		return false
	}
	return n == want
}
