package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"

	sqlitedriver "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

func init() {
	sqlitedriver.MustRegisterDeterministicScalarFunction(sqlshared.ObjectUUIDDeriveFunc, 3,
		func(_ *sqlitedriver.FunctionContext, args []driver.Value) (driver.Value, error) {
			family, ok1 := args[0].(string)
			key, ok2 := args[1].(string)
			rank, ok3 := args[2].(int64)
			if !ok1 || !ok2 || !ok3 {
				return nil, nil
			}
			return sqlshared.DeriveObjectUUID(family, key, rank), nil
		})
}

// objectUUIDExpr is a version 7 UUID in plain SQL (millisecond clock, 74
// random bits). It must not call a function this package registers: the
// trigger below outlives the process, and a build that does not know the
// function would fail every insert it makes.
const objectUUIDExpr = `(SELECT printf('%08x-%04x-7%s-%s%s-%s', ms / 65536, ms % 65536, substr(r, 1, 3),
		substr('89ab', 1 + abs(random()) % 4, 1), substr(r, 4, 3), substr(r, 7, 12))
	FROM (SELECT CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) AS ms, lower(hex(randomblob(9))) AS r))`

// EnsureObjectUUIDs gives every object a uuid: a trigger names the rows
// written from now on, and the rows held before get the identifier derived
// from their natural key (sqlshared.DeriveObjectUUID). Once per library
// (sqlshared.ObjectUUIDKey); batched, so an interrupted pass resumes where it
// stopped and no write lasts longer than one batch. The uuid columns and
// their unique indexes come from EnsureSchema, which runs first.
func EnsureObjectUUIDs(ctx context.Context, db *sql.DB) error {
	tables, err := objectNames(ctx, db, "table")
	if err != nil {
		return err
	}
	var present []sqlshared.ObjectUUIDFamily
	for _, f := range sqlshared.ObjectUUIDFamilies {
		if !tables[f.Table] {
			continue
		}
		cols, err := columnNames(ctx, db, f.Table)
		if err != nil {
			return err
		}
		if !cols["uuid"] {
			return fmt.Errorf("sqlite: %s has no uuid column", f.Table)
		}
		present = append(present, f)
	}
	for _, f := range present {
		if _, err := db.ExecContext(ctx, `CREATE TRIGGER IF NOT EXISTS object_uuid_`+f.Table+` AFTER INSERT ON `+f.Table+`
			WHEN NEW.uuid IS NULL
			BEGIN UPDATE `+f.Table+` SET uuid = `+objectUUIDExpr+` WHERE id = NEW.id; END`); err != nil {
			return fmt.Errorf("sqlite: uuid trigger on %s: %w", f.Table, err)
		}
	}
	var done string
	err = db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, sqlshared.ObjectUUIDKey).Scan(&done)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("sqlite: probe the uuid backfill: %w", err)
	}
	total := 0
	for _, f := range present {
		if f.Ranked {
			res, err := db.ExecContext(ctx, f.BackfillSQL(false, 0, 0))
			if err != nil {
				return fmt.Errorf("sqlite: derive the uuids of %s: %w", f.Table, err)
			}
			n, _ := res.RowsAffected()
			total += int(n)
			continue
		}
		var after int64
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			var upTo sql.NullInt64
			if err := db.QueryRowContext(ctx, f.NextBoundSQL(after, sqlshared.ObjectUUIDBatch)).Scan(&upTo); err != nil {
				return fmt.Errorf("sqlite: bound the uuids of %s: %w", f.Table, err)
			}
			if !upTo.Valid {
				break
			}
			res, err := db.ExecContext(ctx, f.BackfillSQL(false, after, upTo.Int64))
			if err != nil {
				return fmt.Errorf("sqlite: derive the uuids of %s: %w", f.Table, err)
			}
			n, _ := res.RowsAffected()
			total += int(n)
			after = upTo.Int64
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, '1')`, sqlshared.ObjectUUIDKey); err != nil {
		return fmt.Errorf("sqlite: record the uuid backfill: %w", err)
	}
	if total > 0 {
		slog.Info("gave the stored objects their uuid", "rows", total)
	}
	return nil
}

// ForgetObjectUUIDs puts a freshly bootstrapped library back to "no uuid
// yet": no trigger, no mark. A file an export writes holds rows whose
// identifiers it does not carry; whoever opens it derives them from the
// natural keys (EnsureObjectUUIDs), so an export is the same bytes whenever
// it is made, and two recipients of one file agree.
func ForgetObjectUUIDs(ctx context.Context, db *sql.DB) error {
	for _, f := range sqlshared.ObjectUUIDFamilies {
		if _, err := db.ExecContext(ctx, `DROP TRIGGER IF EXISTS object_uuid_`+f.Table); err != nil {
			return fmt.Errorf("sqlite: drop the uuid trigger on %s: %w", f.Table, err)
		}
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, sqlshared.ObjectUUIDKey); err != nil {
		return fmt.Errorf("sqlite: forget the uuid backfill: %w", err)
	}
	return nil
}
