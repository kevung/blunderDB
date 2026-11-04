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

const (
	journalNow  = `strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`
	notApplying = `NOT EXISTS (SELECT 1 FROM change_log_applying)`
	// supersededPrefix names, for the length of a trigger, the entry the write
	// replaces (see journalWrite).
	supersededPrefix = `'~' || `
)

// stampAfter is the date of a local write over a date held in the expression
// prev: now, or the millisecond after prev when the clock has not passed it (a
// second write in the same millisecond, a clock set back). Dates of one layout
// compare as text.
func stampAfter(prev string) string {
	return `MAX(` + journalNow + `, COALESCE(strftime('%Y-%m-%dT%H:%M:%fZ', ` + prev + `, '+0.001 seconds'), ''))`
}

// journalWrite moves the object named by the expression uuid to the end of the
// journal. The stamp is read from the entry it replaces, which has to outlive
// the insert: seq must be a new one, the uuid is unique, and a plain DELETE then
// INSERT is the only form a trigger can rely on (an ON CONFLICT clause inside a
// trigger is overridden by the policy of the statement that fired it). stamp
// gets the expression of the date the object holds, NULL when it holds none.
func journalWrite(uuid, family, op string, stamp func(prev string) string) string {
	held := `(SELECT changed_at FROM change_log WHERE uuid = ` + supersededPrefix + uuid + `)`
	return fmt.Sprintf(`UPDATE change_log SET uuid = %[5]s%[1]s WHERE uuid = %[1]s; `+
		`INSERT INTO change_log (uuid, family, op, changed_at) SELECT %[1]s, '%[2]s', %[3]s, %[4]s WHERE %[1]s IS NOT NULL; `+
		`DELETE FROM change_log WHERE uuid = %[5]s%[1]s;`,
		uuid, family, op, stamp(held), supersededPrefix)
}

// appearedStamp dates a row that appears: below the floor when it comes with an
// applied record, whose real record then always wins.
func appearedStamp(prev string) string {
	return `CASE WHEN ` + notApplying + ` THEN ` + stampAfter(prev) + ` ELSE '1969-12-31T00:00:00.000Z' END`
}

// goneStamp dates a row that goes: as the applied record says, if one does.
func goneStamp(prev string) string {
	return `COALESCE((SELECT at FROM change_log_applying), ` + stampAfter(prev) + `)`
}

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
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS change_log_applying (at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("sqlite: create change_log_applying: %w", err)
	}
	tables, err := objectNames(ctx, db, "table")
	if err != nil {
		return err
	}
	for _, f := range sqlshared.ObjectUUIDFamilies {
		if !tables[f.Table] {
			continue
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
				WHEN NEW.uuid IS NOT NULL BEGIN ` + journalWrite("NEW.uuid", f.Family, "'create'", appearedStamp) + ` END`,
			`CREATE TRIGGER change_log_` + f.Table + `_u AFTER UPDATE ON ` + f.Table + `
				WHEN NEW.uuid IS NOT NULL` + changedOnly + ` AND NOT (OLD.uuid IS NEW.uuid AND EXISTS (SELECT 1 FROM change_log_applying)) BEGIN ` +
				`DELETE FROM change_log WHERE uuid = OLD.uuid AND OLD.uuid <> NEW.uuid;` + ` ` +
				journalWrite("NEW.uuid", f.Family, `CASE WHEN OLD.uuid IS NULL OR OLD.uuid <> NEW.uuid THEN 'create' ELSE 'update' END`, appearedStamp) + ` END`,
			`CREATE TRIGGER change_log_` + f.Table + `_d AFTER DELETE ON ` + f.Table + `
				WHEN OLD.uuid IS NOT NULL BEGIN ` + journalWrite("OLD.uuid", f.Family, "'delete'", goneStamp) + ` END`,
		}
		for _, s := range stmts {
			if err := ensureTrigger(ctx, db, s); err != nil {
				return fmt.Errorf("sqlite: change_log trigger on %s: %w", f.Table, err)
			}
		}
	}
	if tables["analysis"] && tables["position"] {
		// An analysis has no uuid: a re-analysis is an update of its position.
		body := journalWrite(`(SELECT uuid FROM position WHERE id = NEW.position_id)`, "position", "'update'", stampAfter)
		for _, ev := range []string{"INSERT", "UPDATE"} {
			stmt := `CREATE TRIGGER change_log_analysis_` + strings.ToLower(ev[:1]) + ` AFTER ` + ev + ` ON analysis
				WHEN NEW.position_id IS NOT NULL AND ` + notApplying + ` BEGIN ` + body + ` END`
			if err := ensureTrigger(ctx, db, stmt); err != nil {
				return fmt.Errorf("sqlite: change_log trigger on analysis: %w", err)
			}
		}
	}
	// A drawing and a derivation link have no uuid either: they are part of
	// the position they belong to (the variant, for a link), so writing one
	// is an update of it. The position must still exist: when it is the
	// position that is going, its own delete entry stands.
	for _, tbl := range []string{"position_drawing", "position_derivation"} {
		if !tables[tbl] || !tables["position"] {
			continue
		}
		part := strings.TrimPrefix(tbl, "position_")
		for _, ev := range []struct{ name, ref string }{{"INSERT", "NEW"}, {"UPDATE", "NEW"}, {"DELETE", "OLD"}} {
			pos := `(SELECT uuid FROM position WHERE id = ` + ev.ref + `.position_id)`
			fact := `position_id = ` + ev.ref + `.position_id AND part = '` + part + `'`
			stmt := `CREATE TRIGGER change_log_` + tbl + `_` + strings.ToLower(ev.name[:1]) + ` AFTER ` + ev.name + ` ON ` + tbl + ` WHEN ` + notApplying + ` BEGIN ` +
				journalWrite(pos, "position", "'update'", stampAfter) + ` ` +
				`UPDATE position_part_fact SET changed_at = ` + stampAfter("changed_at") + `, pending_origin = NULL, pending_move = NULL WHERE ` + fact + `; ` +
				`INSERT INTO position_part_fact (position_id, part, changed_at) SELECT id, '` + part + `', ` + journalNow + ` FROM position WHERE id = ` + ev.ref + `.position_id AND NOT EXISTS (SELECT 1 FROM position_part_fact WHERE ` + fact + `); END`
			if err := ensureTrigger(ctx, db, stmt); err != nil {
				return fmt.Errorf("sqlite: change_log trigger on %s: %w", tbl, err)
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
	// Sorted: a migrated library holds its columns in another order than a fresh one.
	slices.Sort(diff)
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
	for _, tbl := range []string{"position_drawing", "position_derivation"} {
		for _, sfx := range []string{"_i", "_u", "_d"} {
			if _, err := db.ExecContext(ctx, `DROP TRIGGER IF EXISTS change_log_`+tbl+sfx); err != nil {
				return fmt.Errorf("sqlite: drop the change_log trigger on %s: %w", tbl, err)
			}
		}
	}
	for _, tbl := range []string{"change_log", "change_log_applying"} {
		if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS `+tbl); err != nil {
			return fmt.Errorf("sqlite: drop %s: %w", tbl, err)
		}
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
