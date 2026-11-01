package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// EnsureChangeLog creates the change journal and the triggers that write it,
// the twin of PostgreSQL's 055_change_log.sql. A uuid that changes (a restore giving back the identifier the object held) was a
// fresh draw nobody has seen: its entry is dropped, not turned into a delete. A row is journaled when its
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
		// A match rewrites its row for what is local to the library too.
		changedOnly := ""
		if f.Table == "match" {
			if changedOnly, err = matchChangeCondition(ctx, db); err != nil {
				return err
			}
		}
		stmts := []string{
			`CREATE TRIGGER change_log_` + f.Table + `_i AFTER INSERT ON ` + f.Table + `
				WHEN NEW.uuid IS NOT NULL BEGIN ` + ins("NEW.uuid", "'create'", "") + ` END`,
			`CREATE TRIGGER change_log_` + f.Table + `_u AFTER UPDATE ON ` + f.Table + `
				WHEN NEW.uuid IS NOT NULL` + changedOnly + ` BEGIN ` +
				`DELETE FROM change_log WHERE uuid = OLD.uuid AND OLD.uuid <> NEW.uuid;` + ` ` +
				ins("NEW.uuid", `CASE WHEN OLD.uuid IS NULL OR OLD.uuid <> NEW.uuid THEN 'create' ELSE 'update' END`, "") + ` END`,
			`CREATE TRIGGER change_log_` + f.Table + `_d AFTER DELETE ON ` + f.Table + `
				WHEN OLD.uuid IS NOT NULL BEGIN ` + ins("OLD.uuid", "'delete'", "") + ` END`,
		}
		for _, s := range stmts {
			if err := ensureTrigger(ctx, db, s); err != nil {
				return fmt.Errorf("sqlite: change_log trigger on %s: %w", f.Table, err)
			}
		}
	}
	if tables["analysis"] && tables["position"] {
		// An analysis has no uuid: a re-analysis is an update of its position.
		const body = `DELETE FROM change_log WHERE uuid = (SELECT uuid FROM position WHERE id = NEW.position_id);` +
			` INSERT INTO change_log (uuid, family, op) SELECT uuid, 'position', 'update' FROM position WHERE id = NEW.position_id AND uuid IS NOT NULL;`
		for _, ev := range []string{"INSERT", "UPDATE"} {
			stmt := `CREATE TRIGGER change_log_analysis_` + strings.ToLower(ev[:1]) + ` AFTER ` + ev + ` ON analysis
				WHEN NEW.position_id IS NOT NULL BEGIN ` + body + ` END`
			if err := ensureTrigger(ctx, db, stmt); err != nil {
				return fmt.Errorf("sqlite: change_log trigger on analysis: %w", err)
			}
		}
	}
	return nil
}

// matchChangeCondition is the part of a WHEN that holds when a column of match
// other than sqlshared.LocalMatchColumns changed.
func matchChangeCondition(ctx context.Context, db *sql.DB) (string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM pragma_table_info('match')`)
	if err != nil {
		return "", fmt.Errorf("sqlite: columns of match: %w", err)
	}
	defer rows.Close()
	var diff []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return "", fmt.Errorf("sqlite: columns of match: %w", err)
		}
		if !slices.Contains(sqlshared.LocalMatchColumns, col) {
			diff = append(diff, fmt.Sprintf(`OLD."%[1]s" IS NOT NEW."%[1]s"`, col))
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("sqlite: columns of match: %w", err)
	}
	return " AND (" + strings.Join(diff, " OR ") + ")", nil
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
	for _, sfx := range []string{"_i", "_u"} {
		if _, err := db.ExecContext(ctx, `DROP TRIGGER IF EXISTS change_log_analysis`+sfx); err != nil {
			return fmt.Errorf("sqlite: drop the change_log trigger on analysis: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS change_log`); err != nil {
		return fmt.Errorf("sqlite: drop change_log: %w", err)
	}
	return nil
}

// ensureTrigger creates the trigger a CREATE TRIGGER statement describes, and
// replaces one of the same name whose body differs (SQLite keeps the text it
// was created with, so a plain IF NOT EXISTS would freeze an old body for good).
func ensureTrigger(ctx context.Context, db *sql.DB, create string) error {
	name := strings.Fields(create)[2]
	var have sql.NullString
	err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type = 'trigger' AND name = ?`, name).Scan(&have)
	if err == nil && have.String == create {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DROP TRIGGER IF EXISTS `+name); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, create); err != nil {
		return err
	}
	return tx.Commit()
}
