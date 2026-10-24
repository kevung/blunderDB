package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// An embedder that never imports package database still gets the breakdown
// tables of a library written with the old rowid layout rebuilt by Migrate.
func TestMigrate_WithoutDatabasePackage_ReclustersStatsCells(t *testing.T) {
	// Another test file of this package imports package database, whose init
	// registers the migrator: unregister it to run the embedder's path.
	saved := registeredMigrator
	registeredMigrator = nil
	defer func() { registeredMigrator = saved }()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "recluster.db")

	st, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`DROP TABLE match_stats_cell`,
		`DROP TABLE match_stats_position`,
		`CREATE TABLE match_stats_cell (match_id INTEGER NOT NULL, seat INTEGER NOT NULL, decision_type INTEGER NOT NULL,
			met_id INTEGER NOT NULL, kind INTEGER NOT NULL, k1 INTEGER NOT NULL, k2 INTEGER NOT NULL, decisions INTEGER NOT NULL,
			error_mp INTEGER NOT NULL, max_error_mp INTEGER NOT NULL, blunders INTEGER NOT NULL, mwc_loss REAL NOT NULL,
			mwc_decisions INTEGER NOT NULL, positions INTEGER NOT NULL,
			PRIMARY KEY (match_id, seat, decision_type, met_id, kind, k1, k2))`,
		`CREATE TABLE match_stats_position (match_id INTEGER NOT NULL, seat INTEGER NOT NULL, position_id INTEGER NOT NULL,
			decision_type INTEGER NOT NULL, met_id INTEGER NOT NULL, PRIMARY KEY (match_id, seat, position_id, met_id))`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	if _, err := raw.Exec(`INSERT INTO match (id) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO match_stats (match_id, seat) VALUES (1, 1)`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	raw, err = OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var rows int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM match_stats`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Errorf("match_stats keeps %d rows without their cells", rows)
	}
	for _, table := range []string{"match_stats_cell", "match_stats_position"} {
		var ddl string
		if err := raw.QueryRow(`SELECT sql FROM sqlite_master WHERE name = ?`, table).Scan(&ddl); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(ddl, "WITHOUT ROWID") {
			t.Errorf("%s keeps its rowid layout", table)
		}
	}
}
