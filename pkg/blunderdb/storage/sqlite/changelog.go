package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// EnsureChangeLog creates the change journal and the triggers that write it,
// the twin of PostgreSQL's 055_change_log.sql. A row is journaled when its
// uuid is set: at insert when it comes with one, otherwise when the uuid
// trigger of EnsureObjectUUIDs names it (NULL to a value), so a backfill that
// runs before the triggers exist leaves no entries. Idempotent; call it after
// EnsureObjectUUIDs.
func EnsureChangeLog(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS change_log (
		seq        INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid       TEXT NOT NULL UNIQUE,
		family     TEXT NOT NULL,
		op         TEXT NOT NULL CHECK (op IN ('create', 'update', 'delete', 'trash')),
		changed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')))`); err != nil {
		return fmt.Errorf("sqlite: create change_log: %w", err)
	}
	tables, err := objectNames(ctx, db, "table")
	if err != nil {
		return err
	}
	for _, f := range sqlshared.ObjectUUIDFamilies {
		if !tables[f.Table] {
			continue
		}
		// A plain DELETE then INSERT: an ON CONFLICT clause inside a trigger
		// is overridden by the policy of the statement that fired it.
		ins := func(uuid, op, cond string) string {
			and, where := "", ""
			if cond != "" {
				and, where = " AND "+cond, " WHERE "+cond
			}
			return fmt.Sprintf(`DELETE FROM change_log WHERE uuid = %[1]s%[4]s; INSERT INTO change_log (uuid, family, op) SELECT %[1]s, '%[2]s', %[3]s%[5]s;`,
				uuid, f.Family, op, and, where)
		}
		stmts := []string{
			`CREATE TRIGGER IF NOT EXISTS change_log_` + f.Table + `_i AFTER INSERT ON ` + f.Table + `
				WHEN NEW.uuid IS NOT NULL BEGIN ` + ins("NEW.uuid", "'create'", "") + ` END`,
			`CREATE TRIGGER IF NOT EXISTS change_log_` + f.Table + `_u AFTER UPDATE ON ` + f.Table + `
				WHEN NEW.uuid IS NOT NULL BEGIN ` +
				ins("OLD.uuid", "'delete'", `OLD.uuid IS NOT NULL AND OLD.uuid <> NEW.uuid`) + ` ` +
				ins("NEW.uuid", `CASE WHEN OLD.uuid IS NULL OR OLD.uuid <> NEW.uuid THEN 'create' ELSE 'update' END`, "") + ` END`,
			`CREATE TRIGGER IF NOT EXISTS change_log_` + f.Table + `_d AFTER DELETE ON ` + f.Table + `
				WHEN OLD.uuid IS NOT NULL BEGIN ` + ins("OLD.uuid", "'delete'", "") + ` END`,
		}
		for _, s := range stmts {
			if _, err := db.ExecContext(ctx, s); err != nil {
				return fmt.Errorf("sqlite: change_log trigger on %s: %w", f.Table, err)
			}
		}
	}
	return nil
}

// ForgetChangeLog drops the journal and its triggers: a file an export writes
// carries no history, only the rows.
func ForgetChangeLog(ctx context.Context, db *sql.DB) error {
	for _, f := range sqlshared.ObjectUUIDFamilies {
		for _, sfx := range []string{"_i", "_u", "_d"} {
			if _, err := db.ExecContext(ctx, `DROP TRIGGER IF EXISTS change_log_`+f.Table+sfx); err != nil {
				return fmt.Errorf("sqlite: drop the change_log trigger on %s: %w", f.Table, err)
			}
		}
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS change_log`); err != nil {
		return fmt.Errorf("sqlite: drop change_log: %w", err)
	}
	return nil
}
