package database

// Schema migrations for the SQLite .db file.
//
// The chain is a registry, migrationSteps: one entry per schema version,
// each naming the version it starts from, the version it produces and the
// function that does the work. runMigrationChain walks it from the version
// the file records up to DatabaseVersion, stamping and logging each step it
// completes. The steps themselves live beside their versions:
//
//   - db_migration_v1.go    — 1.0.0 → 2.0.0 (the pre-2.0 DDL and the 2.0.0
//     scalar-column backfill)
//   - db_migration_v2.go    — 2.0.0 → 2.5.0 (storage-format rewrites)
//   - db_migration_v2_5.go  — 2.5.0 → DatabaseVersion (flag columns, small DDL)
//
// Adding a schema version means: a migrate_X_to_Y function in the file that
// holds its predecessor, one line in migrationSteps, the matching DDL in
// db_schema.go (ensureAllTablesExist), the storage/sqlite fresh schema and a
// storage/postgres migration — and a test in migration_test.go.
// TestMigrationSteps_ContinuousChain fails the build when the registry has a
// gap, a duplicate or does not end at DatabaseVersion.
//
// A step does the schema or data work only. It must not write
// database_version: the loop stamps `to` after the step returns nil and logs
// the upgrade, so an interrupted step leaves the file at `from` and is
// retried on the next open. Every step is therefore written to be
// re-runnable (CREATE ... IF NOT EXISTS, tolerated duplicate columns).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// migrationStep upgrades a database from schema version from to schema
// version to. run receives the Database whose d.db is the file being
// upgraded and the cancellation context of the open.
type migrationStep struct {
	from, to string
	run      func(d *Database, ctx context.Context) error
}

// migrationSteps is the migration chain, in version order. See the package
// comment at the top of this file for the rules a new entry follows.
var migrationSteps = []migrationStep{
	{"1.0.0", "1.1.0", (*Database).migrate_1_0_0_to_1_1_0},
	{"1.1.0", "1.2.0", (*Database).migrate_1_1_0_to_1_2_0},
	{"1.2.0", "1.3.0", (*Database).migrate_1_2_0_to_1_3_0},
	{"1.3.0", "1.4.0", (*Database).migrate_1_3_0_to_1_4_0},
	{"1.4.0", "1.5.0", (*Database).migrate_1_4_0_to_1_5_0},
	{"1.5.0", "1.6.0", (*Database).migrate_1_5_0_to_1_6_0},
	{"1.6.0", "1.7.0", (*Database).migrate_1_6_0_to_1_7_0},
	{"1.7.0", "1.8.0", (*Database).migrate_1_7_0_to_1_8_0},
	{"1.8.0", "1.9.0", (*Database).migrate_1_8_0_to_1_9_0},
	{"1.9.0", "2.0.0", (*Database).migrate_1_9_0_to_2_0_0},
	{"2.0.0", "2.1.0", (*Database).migrate_2_0_0_to_2_1_0},
	{"2.1.0", "2.2.0", (*Database).migrate_2_1_0_to_2_2_0},
	{"2.2.0", "2.3.0", (*Database).migrate_2_2_0_to_2_3_0},
	{"2.3.0", "2.4.0", (*Database).migrate_2_3_0_to_2_4_0},
	{"2.4.0", "2.5.0", (*Database).migrate_2_4_0_to_2_5_0},
	{"2.5.0", "2.6.0", (*Database).migrate_2_5_0_to_2_6_0},
	{"2.6.0", "2.7.0", (*Database).migrate_2_6_0_to_2_7_0},
	{"2.7.0", "2.8.0", (*Database).migrate_2_7_0_to_2_8_0},
	{"2.8.0", "2.9.0", (*Database).migrate_2_8_0_to_2_9_0},
	{"2.9.0", "2.10.0", (*Database).migrate_2_9_0_to_2_10_0},
	{"2.10.0", "2.11.0", (*Database).migrate_2_10_0_to_2_11_0},
	{"2.11.0", "2.12.0", (*Database).migrate_2_11_0_to_2_12_0},
	{"2.12.0", "2.13.0", (*Database).migrate_2_12_0_to_2_13_0},
	{"2.13.0", "2.14.0", (*Database).migrate_2_13_0_to_2_14_0},
	{"2.14.0", "2.15.0", (*Database).migrate_2_14_0_to_2_15_0},
}

// errStepNotApplicable is returned by the 1.0.0 → 1.6.0 steps when the table
// they would create is already there. The historical chain then left
// database_version untouched and went no further (ensureAllTablesExist still
// runs), and the loop keeps that behaviour rather than stamping a version
// whose step did not run.
var errStepNotApplicable = errors.New("migration step not applicable")

// findMigrationStep returns the registered step that starts from the given
// schema version.
func findMigrationStep(from string) (migrationStep, bool) {
	for _, s := range migrationSteps {
		if s.from == from {
			return s, true
		}
	}
	return migrationStep{}, false
}

// SetMigrationProgress registers a callback that is invoked during the v2.0.0
// backfill migration to report progress. The GUI wires in a Wails event emitter
// here; the CLI and tests can leave it nil.
func (d *Database) SetMigrationProgress(fn func(phase string, done, total int)) {
	d.migrationProgress = fn
}

// emitMigrationProgress calls the progress callback if one was registered.
func (d *Database) emitMigrationProgress(phase string, done, total int) {
	if d.migrationProgress != nil {
		d.migrationProgress(phase, done, total)
	}
}

// runMigrationChain reads the recorded schema version and applies the
// registered steps (migrationSteps) up to DatabaseVersion, then verifies
// the expected tables and metadata keys exist. It is shared by the GUI/CLI
// Database wrapper (OpenDatabase) and, via the registered storage migrator
// (migrate_hook.go), by the headless storage backend opening a pre-existing
// database. The caller must hold d.mu when d is a shared instance; the
// storage path uses a transient Database, so no lock is needed there.
//
// A database stamped with a version this build does not know is an error,
// unless that version is newer than DatabaseVersion: the schema only ever
// grows, so a file written by a later blunderDB is opened as it is.
func (d *Database) runMigrationChain(ctx context.Context) error {
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}
	if _, err := parseVersion(dbVersion); err != nil {
		return fmt.Errorf("database_version %q: %w", dbVersion, err)
	}

	for dbVersion != DatabaseVersion {
		step, ok := findMigrationStep(dbVersion)
		if !ok {
			if newer, _ := compareVersions(dbVersion, DatabaseVersion); newer > 0 {
				slog.Warn("database is newer than this build; opening it unchanged",
					"database_version", dbVersion, "supported", DatabaseVersion)
				break
			}
			return fmt.Errorf("no migration step from database version %s", dbVersion)
		}
		err := step.run(d, ctx)
		if errors.Is(err, errStepNotApplicable) {
			break
		}
		if err != nil {
			return fmt.Errorf("migration %s→%s failed: %w", step.from, step.to, err)
		}
		if _, err := d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, step.to); err != nil {
			return fmt.Errorf("migration %s→%s failed: migrate version bump: %w", step.from, step.to, err)
		}
		dbVersion = step.to
		slog.Info("database upgraded", "from", step.from, "to", step.to)
	}

	// Ensure all required tables and columns exist.
	// This repairs databases that were migrated through versions that skipped
	// creating some tables (e.g. filter_library was missing from some migration paths).
	if err := d.ensureAllTablesExist(); err != nil {
		return err
	}

	// Repair position rows the native .db importer once stored without their
	// Zobrist hash and scalar search columns (see repairPositionsWithoutScalars).
	// No version bump: the schema is unchanged, only the values are restored.
	if err := d.repairPositionsWithoutScalars(ctx); err != nil {
		return fmt.Errorf("repairing positions without scalar columns: %w", err)
	}

	// Build required tables list based on the FINAL dbVersion (after all migrations)
	requiredTables := []string{"position", "analysis", "comment", "metadata"}
	if versionAtLeast(dbVersion, "1.1.0") {
		requiredTables = append(requiredTables, "command_history")
	}
	if versionAtLeast(dbVersion, "1.2.0") {
		requiredTables = append(requiredTables, "filter_library")
	}
	if versionAtLeast(dbVersion, "1.3.0") {
		requiredTables = append(requiredTables, "search_history")
	}
	if versionAtLeast(dbVersion, "1.4.0") {
		requiredTables = append(requiredTables, "match", "game", "move", "move_analysis")
	}
	if versionAtLeast(dbVersion, "1.5.0") {
		requiredTables = append(requiredTables, "collection", "collection_position")
	}
	if versionAtLeast(dbVersion, "1.6.0") {
		requiredTables = append(requiredTables, "tournament")
	}

	for _, table := range requiredTables {
		var tableName string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&tableName)
		if err != nil {
			return err
		}
		if tableName != table {
			return fmt.Errorf("required table %s does not exist", table)
		}
	}

	// Check if the required metadata keys exist
	requiredKeys := []string{"database_version"}
	for _, key := range requiredKeys {
		var value string
		err = d.db.QueryRow(`SELECT value FROM metadata WHERE key=?`, key).Scan(&value)
		if err != nil {
			return err
		}
		if value == "" {
			return fmt.Errorf("required metadata key %s does not exist", key)
		}
	}
	return nil
}

// parseVersion splits a schema version "MAJOR.MINOR.PATCH" into its three
// numeric components.
func parseVersion(v string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("want MAJOR.MINOR.PATCH, got %q", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, fmt.Errorf("component %q of %q is not a non-negative integer", p, v)
		}
		out[i] = n
	}
	return out, nil
}

// compareVersions orders two schema versions component by component, so that
// 2.10.0 sorts after 2.9.0 — a plain string comparison puts it before. It
// returns -1, 0 or +1 as a is older than, equal to or newer than b.
func compareVersions(a, b string) (int, error) {
	va, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	vb, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	for i := range va {
		switch {
		case va[i] < vb[i]:
			return -1, nil
		case va[i] > vb[i]:
			return 1, nil
		}
	}
	return 0, nil
}

// versionAtLeast reports whether v is min or newer. An unparseable v counts
// as older than anything (runMigrationChain rejects such a version before
// it gets here).
func versionAtLeast(v, min string) bool {
	c, err := compareVersions(v, min)
	return err == nil && c >= 0
}

// tableAbsent reports whether no table of that name exists — precisely,
// whether the sqlite_master lookup returned sql.ErrNoRows, which is the test
// the 1.x steps have always made (any other lookup error counts as present).
func (d *Database) tableAbsent(name string) bool {
	var found string
	err := d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&found)
	return errors.Is(err, sql.ErrNoRows)
}

// addColumn runs an ALTER TABLE ... ADD COLUMN statement and tolerates the
// column already being there, which is what a re-run after an interrupted
// step looks like. col is the column name SQLite names in that error.
func (d *Database) addColumn(stmt, col string) error {
	if _, err := d.db.Exec(stmt); err != nil && err.Error() != `duplicate column name: `+col {
		return err
	}
	return nil
}

// forEachRow runs query and hands every row to each, then reports the
// iteration error. The rows are closed before it returns, so a batch loop
// never holds a cursor open across iterations. Scan errors are each's
// business: the migrations skip an unreadable row rather than abort.
func (d *Database) forEachRow(query string, args []any, each func(*sql.Rows)) error {
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		each(rows)
	}
	return rows.Err()
}
