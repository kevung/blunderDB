package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// createOldDatabase creates a minimal database simulating a given schema version.
// It creates only the tables that existed at that version, with the version stored in metadata.
func createOldDatabase(t *testing.T, path string, version string) {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Error enabling foreign keys: %v", err)
	}

	// All versions have these base tables
	_, err = db.Exec(`
		CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		);
		CREATE TABLE analysis (
			id INTEGER PRIMARY KEY,
			position_id INTEGER,
			data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE comment (
			id INTEGER PRIMARY KEY,
			position_id INTEGER,
			text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		);
	`)
	if err != nil {
		t.Fatalf("Error creating base tables: %v", err)
	}

	// v1.1.0+: command_history
	if version >= "1.1.0" {
		_, err = db.Exec(`
			CREATE TABLE command_history (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				command TEXT,
				timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Error creating command_history table: %v", err)
		}
	}

	// v1.2.0+: filter_library
	if version >= "1.2.0" {
		_, err = db.Exec(`
			CREATE TABLE filter_library (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT,
				command TEXT,
				edit_position TEXT
			)
		`)
		if err != nil {
			t.Fatalf("Error creating filter_library table: %v", err)
		}
	}

	// v1.3.0+: search_history
	if version >= "1.3.0" {
		_, err = db.Exec(`
			CREATE TABLE search_history (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				command TEXT,
				position TEXT,
				timestamp INTEGER
			)
		`)
		if err != nil {
			t.Fatalf("Error creating search_history table: %v", err)
		}
	}

	// v1.4.0+: match, game, move, move_analysis
	if version >= "1.4.0" {
		_, err = db.Exec(`
			CREATE TABLE match (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				player1_name TEXT,
				player2_name TEXT,
				event TEXT,
				location TEXT,
				round TEXT,
				match_length INTEGER,
				match_date DATETIME,
				import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
				file_path TEXT,
				game_count INTEGER DEFAULT 0,
				match_hash TEXT
			);
			CREATE INDEX idx_match_hash ON match(match_hash);
			CREATE TABLE game (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				match_id INTEGER,
				game_number INTEGER,
				initial_score_1 INTEGER,
				initial_score_2 INTEGER,
				winner INTEGER,
				points_won INTEGER,
				move_count INTEGER DEFAULT 0,
				FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
			);
			CREATE TABLE move (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				game_id INTEGER,
				move_number INTEGER,
				move_type TEXT,
				position_id INTEGER,
				player INTEGER,
				dice_1 INTEGER,
				dice_2 INTEGER,
				checker_move TEXT,
				cube_action TEXT,
				FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
				FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL
			);
			CREATE TABLE move_analysis (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				move_id INTEGER,
				analysis_type TEXT,
				depth TEXT,
				equity REAL,
				equity_error REAL,
				win_rate REAL,
				gammon_rate REAL,
				backgammon_rate REAL,
				opponent_win_rate REAL,
				opponent_gammon_rate REAL,
				opponent_backgammon_rate REAL,
				FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
			);
		`)
		if err != nil {
			t.Fatalf("Error creating match-related tables: %v", err)
		}
	}

	// v1.5.0+: collection, collection_position
	if version >= "1.5.0" {
		_, err = db.Exec(`
			CREATE TABLE collection (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				description TEXT,
				sort_order INTEGER DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE collection_position (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				collection_id INTEGER NOT NULL,
				position_id INTEGER NOT NULL,
				sort_order INTEGER DEFAULT 0,
				added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
				FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
				UNIQUE(collection_id, position_id)
			);
			CREATE INDEX idx_collection_position_collection ON collection_position(collection_id);
		`)
		if err != nil {
			t.Fatalf("Error creating collection tables: %v", err)
		}
	}

	// v1.6.0+: tournament
	if version >= "1.6.0" {
		_, err = db.Exec(`
			CREATE TABLE tournament (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				date TEXT,
				location TEXT,
				sort_order INTEGER DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			t.Fatalf("Error creating tournament table: %v", err)
		}
		_, err = db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)
		if err != nil {
			t.Fatalf("Error adding tournament_id column: %v", err)
		}
	}

	// v1.7.0+: last_visited_position column on match
	if version >= "1.7.0" {
		_, err = db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)
		if err != nil {
			t.Fatalf("Error adding last_visited_position column: %v", err)
		}
	}

	// Set the database version
	_, err = db.Exec(`INSERT INTO metadata (key, value) VALUES ('database_version', ?)`, version)
	if err != nil {
		t.Fatalf("Error inserting database version: %v", err)
	}
}

// columnExists checks if a column exists on a table. A failure to read the
// table's layout fails the test rather than reading as "absent".
func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info(%s): %v", table, err)
	}
	return false
}

// TestMigrate_2_7_0_to_2_8_0 verifies the exclude_position column is added to
// search_history and filter_library and that the "Sauf" structure round-trips.
func TestMigrate_2_7_0_to_2_8_0(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v270.db")
	createOldDatabase(t, dbPath, "2.7.0")

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("Failed to open v2.7.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	if !columnExists(t, d.db, "search_history", "exclude_position") {
		t.Errorf("search_history.exclude_position should exist after migration")
	}
	if !columnExists(t, d.db, "filter_library", "exclude_position") {
		t.Errorf("filter_library.exclude_position should exist after migration")
	}

	// search_history round-trip
	if err := d.SaveSearchHistory("s x", `{"include":1}`, `{"exclude":1}`); err != nil {
		t.Fatalf("SaveSearchHistory: %v", err)
	}
	hist, err := d.LoadSearchHistory()
	if err != nil {
		t.Fatalf("LoadSearchHistory: %v", err)
	}
	if len(hist) == 0 || hist[0].ExcludePosition != `{"exclude":1}` {
		t.Errorf("exclude position not persisted in search_history, got %+v", hist)
	}

	// filter_library round-trip
	if err := d.SaveFilter("f1", "s x"); err != nil {
		t.Fatalf("SaveFilter: %v", err)
	}
	if err := d.SaveExcludePosition("f1", `{"exclude":2}`); err != nil {
		t.Fatalf("SaveExcludePosition: %v", err)
	}
	got, err := d.LoadExcludePosition("f1")
	if err != nil {
		t.Fatalf("LoadExcludePosition: %v", err)
	}
	if got != `{"exclude":2}` {
		t.Errorf("exclude position not persisted in filter_library, got %q", got)
	}
}

// tableExists checks if a table exists in the database
func tableExists(db *sql.DB, tableName string) bool {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tableName).Scan(&name)
	return err == nil && name == tableName
}

// allExpectedTables returns all tables expected at the latest version
func allExpectedTables() []string {
	return []string{
		"position", "analysis", "comment", "metadata",
		"command_history",
		"filter_library",
		"search_history", "session_state",
		"match", "game", "move", "move_analysis",
		"collection", "collection_position",
		"tournament",
	}
}

// TestMigrationFromV100 tests migration from a v1.0.0 database (only base tables)
func TestMigrationFromV100(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v100.db")
	createOldDatabase(t, dbPath, "1.0.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.0.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	// Verify it was migrated to the latest version
	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	// Verify all tables exist
	for _, table := range allExpectedTables() {
		if !tableExists(d.db, table) {
			t.Errorf("Table %s should exist after migration from v1.0.0", table)
		}
	}
}

// TestMigrationFromV110 tests migration from v1.1.0 (has command_history)
func TestMigrationFromV110(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v110.db")
	createOldDatabase(t, dbPath, "1.1.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.1.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	for _, table := range allExpectedTables() {
		if !tableExists(d.db, table) {
			t.Errorf("Table %s should exist after migration from v1.1.0", table)
		}
	}
}

// TestMigrationFromV120 tests migration from v1.2.0 (has filter_library)
func TestMigrationFromV120(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v120.db")
	createOldDatabase(t, dbPath, "1.2.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.2.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	for _, table := range allExpectedTables() {
		if !tableExists(d.db, table) {
			t.Errorf("Table %s should exist after migration from v1.2.0", table)
		}
	}
}

// TestMigrationFromV130 tests migration from v1.3.0 (has search_history)
func TestMigrationFromV130(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v130.db")
	createOldDatabase(t, dbPath, "1.3.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.3.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	for _, table := range allExpectedTables() {
		if !tableExists(d.db, table) {
			t.Errorf("Table %s should exist after migration from v1.3.0", table)
		}
	}
}

// TestMigrationFromV140 tests migration from v1.4.0 (has match tables)
func TestMigrationFromV140(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v140.db")
	createOldDatabase(t, dbPath, "1.4.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.4.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	for _, table := range allExpectedTables() {
		if !tableExists(d.db, table) {
			t.Errorf("Table %s should exist after migration from v1.4.0", table)
		}
	}
}

// TestMigrationFromV150 tests migration from v1.5.0 (has collection tables)
func TestMigrationFromV150(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v150.db")
	createOldDatabase(t, dbPath, "1.5.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.5.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	for _, table := range allExpectedTables() {
		if !tableExists(d.db, table) {
			t.Errorf("Table %s should exist after migration from v1.5.0", table)
		}
	}
}

// TestCurrentVersionNoMigration tests that an old database opens and migrates to current version
func TestCurrentVersionNoMigration(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v170.db")
	createOldDatabase(t, dbPath, "1.7.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.7.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s (auto-migrated), got %s", DatabaseVersion, version)
	}
}

// TestMigrationFromV160 tests migration from v1.6.0 (has tournament table)
func TestMigrationFromV160(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v160.db")
	createOldDatabase(t, dbPath, "1.6.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.6.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	for _, table := range allExpectedTables() {
		if !tableExists(d.db, table) {
			t.Errorf("Table %s should exist after migration from v1.6.0", table)
		}
	}
}

// TestMigrationPreservesData tests that existing data survives migration
func TestMigrationPreservesData(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_data_preserve.db")

	// Build two distinct normalized positions for insertion.
	pos1 := initialPosition()
	pos2 := bearoffPosition()
	norm1, _ := json.Marshal(pos1.NormalizeForStorage())
	norm2, _ := json.Marshal(pos2.NormalizeForStorage())

	// Create a v1.0.0 database with some data
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		);
		CREATE TABLE analysis (
			id INTEGER PRIMARY KEY,
			position_id INTEGER,
			data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE comment (
			id INTEGER PRIMARY KEY,
			position_id INTEGER,
			text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		);
		INSERT INTO metadata (key, value) VALUES ('database_version', '1.0.0');
	`)
	if err != nil {
		t.Fatalf("Error setting up test data: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO position (state) VALUES (?)`, string(norm1)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO position (state) VALUES (?)`, string(norm2)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO comment (position_id, text) VALUES (1, 'test comment')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Open with migration
	d := NewDatabase()
	err = d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	closeOnCleanup(t, d)

	// Verify data survived
	var count int
	err = d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&count)
	if err != nil {
		t.Fatalf("Error counting positions: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 positions, got %d", count)
	}

	var commentText string
	err = d.db.QueryRow(`SELECT text FROM comment WHERE position_id = 1`).Scan(&commentText)
	if err != nil {
		t.Fatalf("Error reading comment: %v", err)
	}
	if commentText != "test comment" {
		t.Errorf("Expected 'test comment', got '%s'", commentText)
	}

	// Verify version was updated
	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s, got %s", DatabaseVersion, version)
	}
}

// TestMigrationChainVersionProgression tests version is correctly updated at each step
func TestMigrationChainVersionProgression(t *testing.T) {
	versions := []string{"1.0.0", "1.1.0", "1.2.0", "1.3.0", "1.4.0", "1.5.0", "1.6.0"}

	for _, startVersion := range versions {
		t.Run(fmt.Sprintf("from_%s", startVersion), func(t *testing.T) {
			tmpDir := tempDir(t)
			dbPath := filepath.Join(tmpDir, "test.db")
			createOldDatabase(t, dbPath, startVersion)

			d := NewDatabase()
			err := d.OpenDatabase(dbPath)
			if err != nil {
				t.Fatalf("Failed to open %s database: %v", startVersion, err)
			}
			closeOnCleanup(t, d)

			// After migration, version should always be the latest
			version, err := d.CheckDatabaseVersion()
			if err != nil {
				t.Fatalf("Failed to get version: %v", err)
			}
			if version != DatabaseVersion {
				t.Errorf("Starting from %s: expected final version %s, got %s", startVersion, DatabaseVersion, version)
			}

			// Re-open to verify it can be reopened without errors
			d2 := NewDatabase()
			err = d2.OpenDatabase(dbPath)
			if err != nil {
				t.Fatalf("Failed to reopen migrated database (from %s): %v", startVersion, err)
			}
			closeOnCleanup(t, d2)
		})
	}
}

// TestSetupThenOpen tests that a database created by SetupDatabase can be opened
func TestSetupThenOpen(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_setup.db")

	d := NewDatabase()
	err := d.SetupDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to setup database: %v", err)
	}
	closeOnCleanup(t, d)

	// Insert a test position so the DB has some data
	_, err = d.db.Exec(`INSERT INTO position (state) VALUES ('{"test":"data"}')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create a new instance and open
	d2 := NewDatabase()
	err = d2.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database created by SetupDatabase: %v", err)
	}
	closeOnCleanup(t, d2)

	version, err := d2.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s, got %s", DatabaseVersion, version)
	}

	// Verify last_visited_position column exists on fresh database
	var colSQL string
	err = d2.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&colSQL)
	if err != nil {
		t.Fatalf("Failed to get match table schema: %v", err)
	}
	if !strings.Contains(colSQL, "last_visited_position") {
		t.Errorf("Fresh database match table missing last_visited_position column. Schema: %s", colSQL)
	}

	// Cleanup
	os.Remove(dbPath)
}

// TestOpenDatabaseMissingFilterLibrary reproduces the bug where databases at v1.7.0
// are missing the filter_library table (skipped during a past migration path).
// OpenDatabase must repair such databases instead of failing.
func TestOpenDatabaseMissingFilterLibrary(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "missing_filter_library.db")

	// Create a v1.7.0 database WITHOUT filter_library (simulates the real bug)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE position (id INTEGER PRIMARY KEY AUTOINCREMENT, state TEXT);
		CREATE TABLE analysis (id INTEGER PRIMARY KEY, position_id INTEGER, data JSON, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE);
		CREATE TABLE comment (id INTEGER PRIMARY KEY, position_id INTEGER, text TEXT, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE search_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, position TEXT, timestamp INTEGER);
		CREATE TABLE match (id INTEGER PRIMARY KEY AUTOINCREMENT, player1_name TEXT, player2_name TEXT, event TEXT, location TEXT, round TEXT, match_length INTEGER, match_date DATETIME, import_date DATETIME DEFAULT CURRENT_TIMESTAMP, file_path TEXT, game_count INTEGER DEFAULT 0, match_hash TEXT, tournament_id INTEGER, last_visited_position INTEGER DEFAULT -1);
		CREATE INDEX idx_match_hash ON match(match_hash);
		CREATE TABLE game (id INTEGER PRIMARY KEY AUTOINCREMENT, match_id INTEGER, game_number INTEGER, initial_score_1 INTEGER, initial_score_2 INTEGER, winner INTEGER, points_won INTEGER, move_count INTEGER DEFAULT 0, FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE);
		CREATE TABLE move (id INTEGER PRIMARY KEY AUTOINCREMENT, game_id INTEGER, move_number INTEGER, move_type TEXT, position_id INTEGER, player INTEGER, dice_1 INTEGER, dice_2 INTEGER, checker_move TEXT, cube_action TEXT, FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL);
		CREATE TABLE move_analysis (id INTEGER PRIMARY KEY AUTOINCREMENT, move_id INTEGER, analysis_type TEXT, depth TEXT, equity REAL, equity_error REAL, win_rate REAL, gammon_rate REAL, backgammon_rate REAL, opponent_win_rate REAL, opponent_gammon_rate REAL, opponent_backgammon_rate REAL, FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE);
		CREATE TABLE collection (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE collection_position (id INTEGER PRIMARY KEY AUTOINCREMENT, collection_id INTEGER NOT NULL, position_id INTEGER NOT NULL, sort_order INTEGER DEFAULT 0, added_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE, UNIQUE(collection_id, position_id));
		CREATE TABLE tournament (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, date TEXT, location TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO metadata (key, value) VALUES ('database_version', '1.7.0');
	`)
	if err != nil {
		t.Fatalf("Error creating test database: %v", err)
	}

	// Verify filter_library does NOT exist (reproducing the bug)
	if tableExists(db, "filter_library") {
		t.Fatal("Test setup error: filter_library should NOT exist yet")
	}
	db.Close()

	// OpenDatabase should succeed and repair the missing table
	d := NewDatabase()
	err = d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase failed on database missing filter_library: %v", err)
	}
	closeOnCleanup(t, d)

	// Verify filter_library was created
	if !tableExists(d.db, "filter_library") {
		t.Error("filter_library table should have been created during OpenDatabase")
	}

	// Verify version is still correct
	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s, got %s", DatabaseVersion, version)
	}
}

// TestOpenDatabaseMissingCanonicalHash tests that databases migrated to v1.7.0
// without the canonical_hash column on match table get repaired.
func TestOpenDatabaseMissingCanonicalHash(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "missing_canonical_hash.db")

	// Create a v1.7.0 database without canonical_hash column
	createOldDatabase(t, dbPath, "1.7.0")

	// Verify canonical_hash does NOT exist
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	var colInfo string
	err = db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&colInfo)
	if err != nil {
		t.Fatalf("Error getting match schema: %v", err)
	}
	if strings.Contains(colInfo, "canonical_hash") {
		t.Fatal("Test setup error: canonical_hash should NOT exist in createOldDatabase v1.7.0")
	}
	db.Close()

	// OpenDatabase should succeed and add the missing column
	d := NewDatabase()
	err = d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase failed on database missing canonical_hash: %v", err)
	}
	closeOnCleanup(t, d)

	// Verify canonical_hash was added
	err = d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&colInfo)
	if err != nil {
		t.Fatalf("Error getting match schema after open: %v", err)
	}
	if !strings.Contains(colInfo, "canonical_hash") {
		t.Errorf("canonical_hash column should have been added during OpenDatabase. Schema: %s", colInfo)
	}
}

// TestOpenDatabaseMissingMultipleTables tests repair of a database missing
// multiple tables (e.g. filter_library AND search_history).
func TestOpenDatabaseMissingMultipleTables(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "missing_multiple.db")

	// Create a minimal v1.7.0 database missing filter_library, search_history, and collection tables
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE position (id INTEGER PRIMARY KEY AUTOINCREMENT, state TEXT);
		CREATE TABLE analysis (id INTEGER PRIMARY KEY, position_id INTEGER, data JSON, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE);
		CREATE TABLE comment (id INTEGER PRIMARY KEY, position_id INTEGER, text TEXT, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE match (id INTEGER PRIMARY KEY AUTOINCREMENT, player1_name TEXT, player2_name TEXT, event TEXT, location TEXT, round TEXT, match_length INTEGER, match_date DATETIME, import_date DATETIME DEFAULT CURRENT_TIMESTAMP, file_path TEXT, game_count INTEGER DEFAULT 0, match_hash TEXT, tournament_id INTEGER, last_visited_position INTEGER DEFAULT -1);
		CREATE TABLE game (id INTEGER PRIMARY KEY AUTOINCREMENT, match_id INTEGER, game_number INTEGER, initial_score_1 INTEGER, initial_score_2 INTEGER, winner INTEGER, points_won INTEGER, move_count INTEGER DEFAULT 0, FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE);
		CREATE TABLE move (id INTEGER PRIMARY KEY AUTOINCREMENT, game_id INTEGER, move_number INTEGER, move_type TEXT, position_id INTEGER, player INTEGER, dice_1 INTEGER, dice_2 INTEGER, checker_move TEXT, cube_action TEXT, FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL);
		CREATE TABLE move_analysis (id INTEGER PRIMARY KEY AUTOINCREMENT, move_id INTEGER, analysis_type TEXT, depth TEXT, equity REAL, equity_error REAL, win_rate REAL, gammon_rate REAL, backgammon_rate REAL, opponent_win_rate REAL, opponent_gammon_rate REAL, opponent_backgammon_rate REAL, FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE);
		CREATE TABLE tournament (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, date TEXT, location TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO metadata (key, value) VALUES ('database_version', '1.7.0');
	`)
	if err != nil {
		t.Fatalf("Error creating test database: %v", err)
	}
	db.Close()

	d := NewDatabase()
	err = d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase failed on database missing multiple tables: %v", err)
	}
	closeOnCleanup(t, d)

	// Verify all missing tables were created
	for _, table := range []string{"filter_library", "search_history", "collection", "collection_position"} {
		if !tableExists(d.db, table) {
			t.Errorf("Table %s should have been created during repair", table)
		}
	}
}

// createV190Database creates a minimal v1.9.0 database with the old schema
// (no scalar columns) and a small set of positions / analyses / matches.
func createV190Database(t *testing.T, path string) {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("createV190Database: open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		);
		CREATE TABLE analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER,
			data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE comment (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER,
			text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE filter_library (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, command TEXT, edit_position TEXT);
		CREATE TABLE search_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, position TEXT, timestamp INTEGER);
		CREATE TABLE match (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			player1_name TEXT, player2_name TEXT, event TEXT, location TEXT, round TEXT,
			match_length INTEGER, match_date DATETIME,
			import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			file_path TEXT, game_count INTEGER DEFAULT 0, match_hash TEXT,
			tournament_id INTEGER, last_visited_position INTEGER DEFAULT -1
		);
		CREATE INDEX idx_match_hash ON match(match_hash);
		CREATE TABLE game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER, game_number INTEGER,
			initial_score_1 INTEGER, initial_score_2 INTEGER,
			winner INTEGER, points_won INTEGER, move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		);
		CREATE TABLE move (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER, move_number INTEGER, move_type TEXT,
			position_id INTEGER, player INTEGER, dice_1 INTEGER, dice_2 INTEGER,
			checker_move TEXT, cube_action TEXT,
			FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL
		);
		CREATE TABLE move_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			move_id INTEGER, analysis_type TEXT, depth TEXT,
			equity REAL, equity_error REAL,
			win_rate REAL, gammon_rate REAL, backgammon_rate REAL,
			opponent_win_rate REAL, opponent_gammon_rate REAL, opponent_backgammon_rate REAL,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
		);
		CREATE TABLE collection (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL, description TEXT, sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE collection_position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL, position_id INTEGER NOT NULL,
			sort_order INTEGER DEFAULT 0, added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(collection_id, position_id)
		);
		CREATE INDEX idx_collection_position_collection ON collection_position(collection_id);
		CREATE TABLE tournament (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL, date TEXT, location TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO metadata (key, value) VALUES ('database_version', '1.9.0');
	`)
	if err != nil {
		t.Fatalf("createV190Database: schema: %v", err)
	}

	// Insert 3 positions with old-style state JSON (no scalar columns).
	// Use real Position structs so ZobristHash and pip counts can be verified.
	pos1 := initialPosition()
	pos2 := bearoffPosition()
	pos3 := cubePosition(2, Black)

	for i, p := range []Position{pos1, pos2, pos3} {
		norm := p.NormalizeForStorage()
		data, err := json.Marshal(norm)
		if err != nil {
			t.Fatalf("createV190Database: marshal pos%d: %v", i+1, err)
		}
		if _, err := db.Exec(`INSERT INTO position (state) VALUES (?)`, string(data)); err != nil {
			t.Fatal(err)
		}
	}

	// Insert analyses for pos1 and pos2 (pos3 has none)
	if _, err := db.Exec(`INSERT INTO analysis (position_id, data) VALUES (1, '{"bestMove":"13/11 24/23","playedMove":"13/11 24/23"}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO analysis (position_id, data) VALUES (2, '{}')`); err != nil {
		t.Fatal(err)
	}

	// Insert a match -> 2 games -> 5 moves (referencing all 3 positions)
	if _, err := db.Exec(`INSERT INTO match (player1_name, player2_name, match_length) VALUES ('Alice','Bob',7)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won) VALUES (1,1,0,0,0,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won) VALUES (1,2,1,0,1,1)`); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		posID := ((i - 1) % 3) + 1
		if _, err := db.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id) VALUES (?,?,?,?)`, 1, i, "checker", posID); err != nil {
			t.Fatal(err)
		}
	}

	// Insert a collection with pos1 in it
	if _, err := db.Exec(`INSERT INTO collection (name) VALUES ('Test collection')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO collection_position (collection_id, position_id) VALUES (1, 1)`); err != nil {
		t.Fatal(err)
	}

	// Insert a tournament
	if _, err := db.Exec(`INSERT INTO tournament (name) VALUES ('Test tournament')`); err != nil {
		t.Fatal(err)
	}
}

// TestMigrate_1_9_0_to_2_0_0 opens a v1.9.0 database and verifies that:
//   - version is bumped to 2.0.0
//   - all new scalar columns are non-NULL for every position
//   - stored column values match what populatePositionColumns recomputes
//   - the v2.0.0 indexes exist
func TestMigrate_1_9_0_to_2_0_0(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "v190.db")
	createV190Database(t, dbPath)

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase failed: %v", err)
	}
	closeOnCleanup(t, d)

	// Version must be current (auto-migrated through all steps).
	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Fatalf("expected version %s, got %s", DatabaseVersion, ver)
	}

	// Every position row must have non-NULL zobrist_hash and pip_1
	rows, err := d.db.Query(`SELECT id, state, zobrist_hash, pip_1, pip_2, pip_diff, off_1, off_2 FROM position`)
	if err != nil {
		t.Fatalf("query position: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		var id int64
		var state string
		var zobrist, pip1, pip2, pipDiff, off1, off2 sql.NullInt64
		if err := rows.Scan(&id, &state, &zobrist, &pip1, &pip2, &pipDiff, &off1, &off2); err != nil {
			t.Fatalf("scan position: %v", err)
		}
		if !zobrist.Valid {
			t.Errorf("position %d: zobrist_hash is NULL", id)
			continue
		}
		if !pip1.Valid || !pip2.Valid {
			t.Errorf("position %d: pip columns are NULL", id)
			continue
		}

		// Recompute and compare
		pos, err := d.loadPositionByIDUnlocked(id)
		if err != nil {
			t.Fatalf("position %d: load: %v", id, err)
		}
		c := populatePositionColumns(&pos)
		if int64(c.ZobristHash) != zobrist.Int64 {
			t.Errorf("position %d: zobrist_hash mismatch: stored %d, computed %d", id, zobrist.Int64, int64(c.ZobristHash))
		}
		if int64(c.Pip1) != pip1.Int64 {
			t.Errorf("position %d: pip_1 mismatch: stored %d, computed %d", id, pip1.Int64, c.Pip1)
		}
		if int64(c.Pip2) != pip2.Int64 {
			t.Errorf("position %d: pip_2 mismatch: stored %d, computed %d", id, pip2.Int64, c.Pip2)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("expected 3 positions, got %d", count)
	}

	// Check that key indexes exist
	for _, idx := range []string{"idx_position_zobrist", "idx_position_decision_pip", "idx_analysis_position"} {
		var name string
		if err := d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&name); err != nil {
			t.Fatal(err)
		}
		if name != idx {
			t.Errorf("index %s not found after migration", idx)
		}
	}
}

// TestMigrate_1_9_0_Duplicates verifies that two positions with the same
// Zobrist hash are merged during migration and FK references are remapped.
func TestMigrate_1_9_0_Duplicates(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "v190_dups.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE position (id INTEGER PRIMARY KEY AUTOINCREMENT, state TEXT);
		CREATE TABLE analysis (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id INTEGER, data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE);
		CREATE TABLE comment (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id INTEGER, text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE filter_library (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, command TEXT, edit_position TEXT);
		CREATE TABLE search_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, position TEXT, timestamp INTEGER);
		CREATE TABLE match (id INTEGER PRIMARY KEY AUTOINCREMENT, player1_name TEXT, player2_name TEXT,
			match_length INTEGER, match_date DATETIME, import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			file_path TEXT, game_count INTEGER DEFAULT 0, match_hash TEXT,
			tournament_id INTEGER, last_visited_position INTEGER DEFAULT -1);
		CREATE TABLE game (id INTEGER PRIMARY KEY AUTOINCREMENT, match_id INTEGER, game_number INTEGER,
			initial_score_1 INTEGER, initial_score_2 INTEGER, winner INTEGER, points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE);
		CREATE TABLE move (id INTEGER PRIMARY KEY AUTOINCREMENT, game_id INTEGER, move_number INTEGER,
			move_type TEXT, position_id INTEGER, player INTEGER, dice_1 INTEGER, dice_2 INTEGER,
			checker_move TEXT, cube_action TEXT,
			FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL);
		CREATE TABLE move_analysis (id INTEGER PRIMARY KEY AUTOINCREMENT, move_id INTEGER, analysis_type TEXT,
			depth TEXT, equity REAL, equity_error REAL, win_rate REAL, gammon_rate REAL, backgammon_rate REAL,
			opponent_win_rate REAL, opponent_gammon_rate REAL, opponent_backgammon_rate REAL,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE);
		CREATE TABLE collection (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE collection_position (id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL, position_id INTEGER NOT NULL,
			sort_order INTEGER DEFAULT 0, added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(collection_id, position_id));
		CREATE TABLE tournament (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, date TEXT,
			location TEXT, sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO metadata (key, value) VALUES ('database_version', '1.9.0');
	`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Insert the same normalized position twice (identical JSON → same Zobrist hash after migration)
	pos := initialPosition()
	norm := pos.NormalizeForStorage()
	posJSON, _ := json.Marshal(norm)
	jsonStr := string(posJSON)
	// id=1, then id=2 as an exact duplicate.
	for range 2 {
		if _, err := db.Exec(`INSERT INTO position (state) VALUES (?)`, jsonStr); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO match (player1_name, player2_name, match_length) VALUES ('A','B',7)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won) VALUES (1,1,0,0,0,1)`); err != nil {
		t.Fatal(err)
	}
	// Move pointing at the duplicate (id=2)
	if _, err := db.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id) VALUES (1, 1, 'checker', 2)`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	closeOnCleanup(t, d)

	// Only one position should remain
	var posCount int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount); err != nil {
		t.Fatal(err)
	}
	if posCount != 1 {
		t.Errorf("expected 1 position after dedup, got %d", posCount)
	}

	// The move must now point at the kept position (id=1)
	var movePosID sql.NullInt64
	if err := d.db.QueryRow(`SELECT position_id FROM move WHERE id=1`).Scan(&movePosID); err != nil {
		t.Fatal(err)
	}
	if !movePosID.Valid || movePosID.Int64 != 1 {
		t.Errorf("move.position_id should be 1 after dedup, got %v", movePosID)
	}
}

// TestMigrate_Idempotent verifies that running migration twice (opening a
// fully migrated 2.0.0 DB a second time) is a no-op.
func TestMigrate_Idempotent(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "v190_idempotent.db")
	createV190Database(t, dbPath)

	// First open → migrates 1.9.0 → 2.0.0
	d1 := NewDatabase()
	if err := d1.OpenDatabase(dbPath); err != nil {
		t.Fatalf("first open: %v", err)
	}
	closeOnCleanup(t, d1)
	var posCount1 int
	if err := d1.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount1); err != nil {
		t.Fatal(err)
	}

	// Second open → must succeed without error and leave data unchanged
	d2 := NewDatabase()
	if err := d2.OpenDatabase(dbPath); err != nil {
		t.Fatalf("second open (idempotent): %v", err)
	}
	closeOnCleanup(t, d2)

	ver, _ := d2.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	var posCount2 int
	if err := d2.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount2); err != nil {
		t.Fatal(err)
	}
	if posCount2 != posCount1 {
		t.Errorf("position count changed on second open: %d → %d", posCount1, posCount2)
	}
}

// TestMigrate_2_3_0_to_2_4_0_RepairsMoveError verifies that the 2.3.0→2.4.0
// migration correctly backfills best_move_equity_error for positions where
// PlayedMoves was missing from the analysis JSON blob.
func TestMigrate_2_3_0_to_2_4_0_RepairsMoveError(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "v230_repair.db")

	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Build a minimal v2.3.0 schema.
	_, err = rawDB.Exec(`
		CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT,
			decision_type INTEGER DEFAULT 0
		);
		CREATE TABLE analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER,
			data BLOB,
			best_cube_action TEXT,
			cube_error REAL DEFAULT 0,
			best_move_equity_error REAL DEFAULT 0,
			player1_win_rate REAL DEFAULT 0,
			player1_gammon_rate REAL DEFAULT 0,
			player1_backgammon_rate REAL DEFAULT 0,
			player2_win_rate REAL DEFAULT 0,
			player2_gammon_rate REAL DEFAULT 0,
			player2_backgammon_rate REAL DEFAULT 0,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE comment (id INTEGER PRIMARY KEY, position_id INTEGER, text TEXT);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE filter_library (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, command TEXT, edit_position TEXT);
		CREATE TABLE search_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, position TEXT, timestamp INTEGER);
		CREATE TABLE match (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			player1_name TEXT, player2_name TEXT,
			match_length INTEGER, match_date DATETIME,
			import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			file_path TEXT, game_count INTEGER DEFAULT 0,
			match_hash TEXT, tournament_id INTEGER,
			last_visited_position INTEGER DEFAULT -1
		);
		CREATE TABLE game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER, game_number INTEGER,
			initial_score_1 INTEGER, initial_score_2 INTEGER,
			winner INTEGER, points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		);
		CREATE TABLE move (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER, move_number INTEGER,
			move_type TEXT, position_id INTEGER,
			player INTEGER, dice_1 INTEGER, dice_2 INTEGER,
			checker_move TEXT, cube_action TEXT,
			FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL
		);
		CREATE TABLE move_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			move_id INTEGER, analysis_type TEXT,
			depth TEXT, equity REAL, equity_error REAL,
			win_rate REAL, gammon_rate REAL, backgammon_rate REAL,
			opponent_win_rate REAL, opponent_gammon_rate REAL, opponent_backgammon_rate REAL,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
		);
		CREATE TABLE collection (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE collection_position (id INTEGER PRIMARY KEY AUTOINCREMENT, collection_id INTEGER NOT NULL, position_id INTEGER NOT NULL, sort_order INTEGER DEFAULT 0, added_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE, UNIQUE(collection_id, position_id));
		CREATE TABLE tournament (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, date TEXT, location TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO metadata (key, value) VALUES ('database_version', '2.3.0');
	`)
	if err != nil {
		t.Fatalf("setup schema: %v", err)
	}

	// Build analysis blob with 2 checker moves — played move is NOT the best.
	// best move equity = 0.200, played move equity = 0.100 → error = 0.100 → 100 millipoints.
	errVal := 0.100
	ana := PositionAnalysis{
		PositionID:   1,
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 0, Move: "13/7 8/5", Equity: 0.200, EquityError: nil},
				{Index: 1, Move: "24/18 13/11", Equity: 0.100, EquityError: &errVal},
			},
		},
		// PlayedMoves intentionally empty — simulates the bug.
	}
	anaData, err := encodeAnalysisForStorage(&ana)
	if err != nil {
		t.Fatalf("encode analysis: %v", err)
	}

	// Insert position, move (checker_move = the non-best move), and analysis.
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (1, '{}', 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO match (id, player1_name, player2_name, match_length) VALUES (1,'A','B',7)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO game (id, match_id, game_number, initial_score_1, initial_score_2, winner, points_won) VALUES (1,1,1,0,0,0,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO move (id, game_id, move_number, move_type, position_id, player, checker_move) VALUES (1,1,1,'checker',1,1,'24/18 13/11')`); err != nil {
		t.Fatal(err)
	}
	_, err = rawDB.Exec(`INSERT INTO analysis (id, position_id, data, best_move_equity_error) VALUES (1, 1, ?, 0)`, anaData)
	if err != nil {
		t.Fatalf("insert analysis: %v", err)
	}
	rawDB.Close()

	// Open database — triggers migration 2.3.0→2.4.0.
	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	closeOnCleanup(t, d)

	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	var moveErr float64
	if err := d.db.QueryRow(`SELECT best_move_equity_error FROM analysis WHERE id = 1`).Scan(&moveErr); err != nil {
		t.Fatal(err)
	}
	// Expected: 100 millipoints (0.100 EMG × 1000)
	if moveErr != 100 {
		t.Errorf("expected best_move_equity_error = 100 millipoints after repair, got %g", moveErr)
	}
}

// TestMigrate_2_4_0_to_2_5_0_IsForced verifies that the 2.4.0→2.5.0 migration:
//   - adds the is_forced column
//   - sets is_forced=1 for checker positions with exactly one legal move
//   - leaves is_forced=0 for positions with multiple legal moves
//   - leaves is_forced=0 for cube positions
func TestMigrate_2_4_0_to_2_5_0_IsForced(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "v240_is_forced.db")

	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Build a minimal v2.4.0 schema (no is_forced column).
	_, err = rawDB.Exec(`
		CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT,
			decision_type INTEGER DEFAULT 0
		);
		CREATE TABLE analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER,
			data BLOB,
			best_cube_action TEXT,
			cube_error INTEGER DEFAULT 0,
			best_move_equity_error INTEGER DEFAULT 0,
			player1_win_rate INTEGER DEFAULT 0,
			player1_gammon_rate INTEGER DEFAULT 0,
			player1_backgammon_rate INTEGER DEFAULT 0,
			player2_win_rate INTEGER DEFAULT 0,
			player2_gammon_rate INTEGER DEFAULT 0,
			player2_backgammon_rate INTEGER DEFAULT 0,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE comment (id INTEGER PRIMARY KEY, position_id INTEGER, text TEXT);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE filter_library (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, command TEXT, edit_position TEXT);
		CREATE TABLE search_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, position TEXT, timestamp INTEGER);
		CREATE TABLE match (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			player1_name TEXT, player2_name TEXT,
			match_length INTEGER, match_date DATETIME,
			import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			file_path TEXT, game_count INTEGER DEFAULT 0,
			match_hash TEXT, tournament_id INTEGER,
			last_visited_position INTEGER DEFAULT -1
		);
		CREATE TABLE game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER, game_number INTEGER,
			initial_score_1 INTEGER, initial_score_2 INTEGER,
			winner INTEGER, points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		);
		CREATE TABLE move (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER, move_number INTEGER,
			move_type TEXT, position_id INTEGER,
			player INTEGER, dice_1 INTEGER, dice_2 INTEGER,
			checker_move TEXT, cube_action TEXT,
			FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL
		);
		CREATE TABLE move_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			move_id INTEGER, analysis_type TEXT,
			depth TEXT, equity REAL, equity_error REAL,
			win_rate REAL, gammon_rate REAL, backgammon_rate REAL,
			opponent_win_rate REAL, opponent_gammon_rate REAL, opponent_backgammon_rate REAL,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
		);
		CREATE TABLE collection (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE collection_position (id INTEGER PRIMARY KEY AUTOINCREMENT, collection_id INTEGER NOT NULL, position_id INTEGER NOT NULL, sort_order INTEGER DEFAULT 0, added_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE, UNIQUE(collection_id, position_id));
		CREATE TABLE tournament (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, date TEXT, location TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO metadata (key, value) VALUES ('database_version', '2.4.0');
	`)
	if err != nil {
		t.Fatalf("setup schema: %v", err)
	}

	// Three analysis rows:
	//   id=1: checker, 1 move → should become is_forced=1
	//   id=2: checker, 2 moves → should stay is_forced=0
	//   id=3: cube (decision_type=1) → should stay is_forced=0

	encodeAna := func(a PositionAnalysis) []byte {
		data, err := encodeAnalysisForStorage(&a)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		return data
	}

	// id=1: forced checker (1 move)
	forced := encodeAna(PositionAnalysis{
		PositionID: 1,
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{{Index: 0, Move: "bar/20", Equity: 0.5}},
		},
	})
	// id=2: unforced checker (2 moves)
	unforced := encodeAna(PositionAnalysis{
		PositionID: 2,
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 0, Move: "13/7 8/5", Equity: 0.300},
				{Index: 1, Move: "24/18 13/11", Equity: 0.200},
			},
		},
	})
	// id=3: cube decision (decision_type=1)
	cube := encodeAna(PositionAnalysis{
		PositionID: 3,
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			BestCubeAction: "NoDouble",
		},
	})

	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (1,'{}',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (2,'{}',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (3,'{}',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (1, 1, ?)`, forced); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (2, 2, ?)`, unforced); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (3, 3, ?)`, cube); err != nil {
		t.Fatal(err)
	}
	rawDB.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	closeOnCleanup(t, d)

	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	// is_forced column must exist
	var colExists int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('analysis') WHERE name='is_forced'`).Scan(&colExists); err != nil {
		t.Fatal(err)
	}
	if colExists != 1 {
		t.Fatalf("is_forced column not found in analysis table after migration")
	}

	cases := []struct {
		id         int
		wantForced int
		label      string
	}{
		{1, 1, "forced checker (1 move)"},
		{2, 0, "unforced checker (2 moves)"},
		{3, 0, "cube decision"},
	}
	for _, tc := range cases {
		var got int
		if err := d.db.QueryRow(`SELECT is_forced FROM analysis WHERE id = ?`, tc.id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != tc.wantForced {
			t.Errorf("analysis id=%d (%s): is_forced=%d, want %d", tc.id, tc.label, got, tc.wantForced)
		}
	}
}

// TestMigrate_2_5_0_to_2_6_0_IsCloseCube verifies that the 2.5.0→2.6.0 migration:
//   - adds the is_close_cube column
//   - sets is_close_cube=1 for cube positions that meet the 0.16-threshold predicate
//   - sets is_close_cube=1 for Take/Pass positions
//   - leaves is_close_cube=0 for clearly-not-close cube decisions
//   - leaves is_close_cube=0 for checker positions
func TestMigrate_2_5_0_to_2_6_0_IsCloseCube(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "v250_is_close_cube.db")

	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	_, err = rawDB.Exec(`
		CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT,
			decision_type INTEGER DEFAULT 0
		);
		CREATE TABLE analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER,
			data BLOB,
			best_cube_action TEXT,
			cube_error INTEGER DEFAULT 0,
			best_move_equity_error INTEGER DEFAULT 0,
			player1_win_rate INTEGER DEFAULT 0,
			player1_gammon_rate INTEGER DEFAULT 0,
			player1_backgammon_rate INTEGER DEFAULT 0,
			player2_win_rate INTEGER DEFAULT 0,
			player2_gammon_rate INTEGER DEFAULT 0,
			player2_backgammon_rate INTEGER DEFAULT 0,
			is_forced INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE comment (id INTEGER PRIMARY KEY, position_id INTEGER, text TEXT);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE filter_library (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, command TEXT, edit_position TEXT);
		CREATE TABLE search_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, position TEXT, timestamp INTEGER);
		CREATE TABLE match (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			player1_name TEXT, player2_name TEXT,
			match_length INTEGER, match_date DATETIME,
			import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			file_path TEXT, game_count INTEGER DEFAULT 0,
			match_hash TEXT, tournament_id INTEGER,
			last_visited_position INTEGER DEFAULT -1
		);
		CREATE TABLE game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER, game_number INTEGER,
			initial_score_1 INTEGER, initial_score_2 INTEGER,
			winner INTEGER, points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		);
		CREATE TABLE move (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER, move_number INTEGER,
			move_type TEXT, position_id INTEGER,
			player INTEGER, dice_1 INTEGER, dice_2 INTEGER,
			checker_move TEXT, cube_action TEXT,
			FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL
		);
		CREATE TABLE move_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			move_id INTEGER, analysis_type TEXT,
			depth TEXT, equity REAL, equity_error REAL,
			win_rate REAL, gammon_rate REAL, backgammon_rate REAL,
			opponent_win_rate REAL, opponent_gammon_rate REAL, opponent_backgammon_rate REAL,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
		);
		CREATE TABLE collection (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE collection_position (id INTEGER PRIMARY KEY AUTOINCREMENT, collection_id INTEGER NOT NULL, position_id INTEGER NOT NULL, sort_order INTEGER DEFAULT 0, added_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE, FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE, UNIQUE(collection_id, position_id));
		CREATE TABLE tournament (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, date TEXT, location TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO metadata (key, value) VALUES ('database_version', '2.5.0');
	`)
	if err != nil {
		t.Fatalf("setup schema: %v", err)
	}

	enc := func(a PositionAnalysis) []byte {
		data, err := encodeAnalysisForStorage(&a)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		return data
	}

	// id=1: cube, close (noDouble=0.52, DT=0.50 → diff=0.02 < 0.16)
	close1 := enc(PositionAnalysis{
		PositionID:           1,
		PlayedCubeActions:    []string{"No Double"},
		DoublingCubeAnalysis: &DoublingCubeAnalysis{BestCubeAction: "No Double", CubefulNoDoubleEquity: 0.52, CubefulDoubleTakeEquity: 0.50, CubefulDoublePassEquity: 1.0},
	})
	// id=2: cube, NOT close (noDouble=0.80, DT=0.40 → diff=0.40 >= 0.16)
	notClose := enc(PositionAnalysis{
		PositionID:           2,
		PlayedCubeActions:    []string{"No Double"},
		DoublingCubeAnalysis: &DoublingCubeAnalysis{BestCubeAction: "No Double", CubefulNoDoubleEquity: 0.80, CubefulDoubleTakeEquity: 0.40, CubefulDoublePassEquity: 1.0},
	})
	// id=3: Take decision — always close
	takeDec := enc(PositionAnalysis{
		PositionID:           3,
		PlayedCubeActions:    []string{"Take"},
		DoublingCubeAnalysis: &DoublingCubeAnalysis{BestCubeAction: "Double, Take", CubefulNoDoubleEquity: 0.40, CubefulDoubleTakeEquity: 0.60, CubefulDoublePassEquity: 1.0},
	})
	// id=4: checker position — is_close_cube must stay 0
	checker := enc(PositionAnalysis{
		PositionID:      4,
		CheckerAnalysis: &CheckerAnalysis{Moves: []CheckerMove{{Index: 0, Move: "13/7", Equity: 0.3}}},
	})

	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (1,'{}',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (2,'{}',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (3,'{}',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (4,'{}',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (1, 1, ?)`, close1); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (2, 2, ?)`, notClose); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (3, 3, ?)`, takeDec); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (4, 4, ?)`, checker); err != nil {
		t.Fatal(err)
	}
	rawDB.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	closeOnCleanup(t, d)

	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	var colExists int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('analysis') WHERE name='is_close_cube'`).Scan(&colExists); err != nil {
		t.Fatal(err)
	}
	if colExists != 1 {
		t.Fatalf("is_close_cube column not found after migration")
	}

	cases := []struct {
		id        int
		wantClose int
		label     string
	}{
		{1, 1, "close cube (diff 0.02 < 0.16)"},
		{2, 0, "not close (diff 0.40 >= 0.16)"},
		{3, 1, "Take — always close"},
		{4, 0, "checker position"},
	}
	for _, tc := range cases {
		var got int
		if err := d.db.QueryRow(`SELECT is_close_cube FROM analysis WHERE id = ?`, tc.id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != tc.wantClose {
			t.Errorf("analysis id=%d (%s): is_close_cube=%d, want %d", tc.id, tc.label, got, tc.wantClose)
		}
	}
}

// TestMigrate_2_9_0_to_2_10_0_IsCubeResponse verifies the is_cube_response column
// is added and backfilled from the move table: cube positions whose played cube
// action is a take/pass response get 1, doubling decisions and checker positions
// stay 0.
func TestMigrate_2_9_0_to_2_10_0_IsCubeResponse(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "v290_is_cube_response.db")

	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	_, err = rawDB.Exec(`
		CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT,
			decision_type INTEGER DEFAULT 0
		);
		CREATE TABLE analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER,
			data BLOB
		);
		CREATE TABLE comment (id INTEGER PRIMARY KEY, position_id INTEGER, text TEXT);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE filter_library (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, command TEXT, edit_position TEXT);
		CREATE TABLE search_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, position TEXT, timestamp INTEGER);
		CREATE TABLE match (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			player1_name TEXT, player2_name TEXT,
			match_length INTEGER, match_date DATETIME,
			import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			file_path TEXT, game_count INTEGER DEFAULT 0,
			match_hash TEXT, tournament_id INTEGER,
			last_visited_position INTEGER DEFAULT -1
		);
		CREATE TABLE game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER, game_number INTEGER,
			initial_score_1 INTEGER, initial_score_2 INTEGER,
			winner INTEGER, points_won INTEGER,
			move_count INTEGER DEFAULT 0
		);
		CREATE TABLE move (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER, move_number INTEGER,
			move_type TEXT, position_id INTEGER,
			player INTEGER, dice_1 INTEGER, dice_2 INTEGER,
			checker_move TEXT, cube_action TEXT
		);
		CREATE TABLE move_analysis (id INTEGER PRIMARY KEY AUTOINCREMENT, move_id INTEGER, analysis_type TEXT);
		CREATE TABLE collection (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE collection_position (id INTEGER PRIMARY KEY AUTOINCREMENT, collection_id INTEGER NOT NULL, position_id INTEGER NOT NULL, sort_order INTEGER DEFAULT 0, added_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(collection_id, position_id));
		CREATE TABLE tournament (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, date TEXT, location TEXT, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO metadata (key, value) VALUES ('database_version', '2.9.0');
	`)
	if err != nil {
		t.Fatalf("setup schema: %v", err)
	}

	// id=1: cube, Take response → 1
	// id=2: cube, Double (doubling decision) → 0
	// id=3: cube, No Double → 0
	// id=4: cube, Pass response → 1
	// id=5: checker position → 0
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (1,'{}',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (2,'{}',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (3,'{}',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (4,'{}',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (5,'{}',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, cube_action) VALUES (1,1,'cube',1,'Take')`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, cube_action) VALUES (1,2,'cube',2,'Double')`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, cube_action) VALUES (1,3,'cube',3,'No Double')`); err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, cube_action) VALUES (1,4,'cube',4,'Pass')`); err != nil {
		t.Fatal(err)
	}
	rawDB.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	closeOnCleanup(t, d)

	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	var colExists int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('position') WHERE name='is_cube_response'`).Scan(&colExists); err != nil {
		t.Fatal(err)
	}
	if colExists != 1 {
		t.Fatalf("is_cube_response column not found after migration")
	}

	cases := []struct {
		id       int
		wantResp int
		label    string
	}{
		{1, 1, "Take response"},
		{2, 0, "Double decision"},
		{3, 0, "No Double decision"},
		{4, 1, "Pass response"},
		{5, 0, "checker position"},
	}
	for _, tc := range cases {
		var got int
		if err := d.db.QueryRow(`SELECT is_cube_response FROM position WHERE id = ?`, tc.id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != tc.wantResp {
			t.Errorf("position id=%d (%s): is_cube_response=%d, want %d", tc.id, tc.label, got, tc.wantResp)
		}
	}
}

// TestMigrate_2_12_0_to_2_13_0_Backfill checks the one-shot reconstruction of
// provenance on an existing database (ADR-0001). The move graph is the only
// signal such a database carries: a position reachable from no move never came
// from a match, so it must be the user's own.
func TestMigrate_2_12_0_to_2_13_0_Backfill(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v2120.db")
	createOldDatabase(t, dbPath, "2.12.0")

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}

	// Two positions: one the user had saved on their own (no move references it),
	// one a match brought in (a move does).
	standalone, err := raw.Exec(`INSERT INTO position (state) VALUES ('{}')`)
	if err != nil {
		t.Fatalf("insert standalone position: %v", err)
	}
	standaloneID, _ := standalone.LastInsertId()

	inMatch, err := raw.Exec(`INSERT INTO position (state) VALUES ('{"a":1}')`)
	if err != nil {
		t.Fatalf("insert match position: %v", err)
	}
	inMatchID, _ := inMatch.LastInsertId()

	m, err := raw.Exec(`INSERT INTO match (player1_name, player2_name) VALUES ('A','B')`)
	if err != nil {
		t.Fatalf("insert match: %v", err)
	}
	matchID, _ := m.LastInsertId()
	g, err := raw.Exec(`INSERT INTO game (match_id, game_number) VALUES (?, 1)`, matchID)
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	gameID, _ := g.LastInsertId()
	if _, err := raw.Exec(
		`INSERT INTO move (game_id, move_number, move_type, position_id, player) VALUES (?, 1, 'checker', ?, 0)`,
		gameID, inMatchID); err != nil {
		t.Fatalf("insert move: %v", err)
	}
	raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.12.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	defer d.db.Close()

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("CheckDatabaseVersion: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("version after migration: got %s, want %s", version, DatabaseVersion)
	}
	if !columnExists(t, d.db, "position", "individually_imported") {
		t.Fatal("position.individually_imported should exist after migration")
	}

	flag := func(id int64) int {
		var v int
		if err := d.db.QueryRow(`SELECT individually_imported FROM position WHERE id = ?`, id).Scan(&v); err != nil {
			t.Fatalf("read flag for position %d: %v", id, err)
		}
		return v
	}
	if got := flag(standaloneID); got != 1 {
		t.Errorf("a position no move references should be backfilled as individually imported, got %d", got)
	}
	if got := flag(inMatchID); got != 0 {
		t.Errorf("a position a move references came from a match, got individually_imported=%d", got)
	}
}

// TestMigrate_2_13_0_to_2_14_0_Flagged checks that an existing database gains
// position.flagged. There is deliberately nothing to backfill: unlike
// individually_imported, no signal inside an already-imported database records a
// source-file mark, so every existing position starts unflagged and only gains
// the mark when its match is imported again (docs/adr/0006).
func TestMigrate_2_13_0_to_2_14_0_Flagged(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v2130.db")
	createOldDatabase(t, dbPath, "2.13.0")

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.Exec(`ALTER TABLE position ADD COLUMN individually_imported INTEGER NOT NULL DEFAULT 0`); err != nil {
		t.Fatalf("prepare v2.13.0 position table: %v", err)
	}
	res, err := raw.Exec(`INSERT INTO position (state) VALUES ('{}')`)
	if err != nil {
		t.Fatalf("insert position: %v", err)
	}
	posID, _ := res.LastInsertId()
	raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.13.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	defer d.db.Close()

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("CheckDatabaseVersion: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("version after migration: got %s, want %s", version, DatabaseVersion)
	}
	if !columnExists(t, d.db, "position", "flagged") {
		t.Fatal("position.flagged should exist after migration")
	}

	var flagged int
	if err := d.db.QueryRow(`SELECT flagged FROM position WHERE id = ?`, posID).Scan(&flagged); err != nil {
		t.Fatalf("read flagged: %v", err)
	}
	if flagged != 0 {
		t.Errorf("migration must not invent marks: got flagged=%d, want 0", flagged)
	}
}

func TestMigrate_2_14_0_to_2_15_0_LuckMP(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v2140.db")
	createOldDatabase(t, dbPath, "2.14.0")

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	for _, stmt := range []string{
		`ALTER TABLE position ADD COLUMN individually_imported INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE position ADD COLUMN flagged INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("prepare v2.14.0 position table: %v", err)
		}
	}
	if _, err := raw.Exec(`INSERT INTO game (id, match_id, game_number) VALUES (1, 1, 1)`); err != nil {
		t.Fatalf("insert game: %v", err)
	}
	res, err := raw.Exec(
		`INSERT INTO move (game_id, move_number, move_type, player, dice_1, dice_2) VALUES (1, 1, 'checker', 1, 3, 1)`)
	if err != nil {
		t.Fatalf("insert move: %v", err)
	}
	moveID, _ := res.LastInsertId()
	raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.14.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	defer d.db.Close()

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("CheckDatabaseVersion: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("version after migration: got %s, want %s", version, DatabaseVersion)
	}
	if !columnExists(t, d.db, "move", "luck_mp") {
		t.Fatal("move.luck_mp should exist after migration")
	}

	// The roll must read "unknown", not "neutral": zero is a real luck value,
	// so a migration that defaulted the column to 0 would fabricate 189 neutral
	// rolls per imported match. There is nothing to back-fill from either —
	// luck was never stored in the analysis JSON.
	var luck sql.NullInt64
	if err := d.db.QueryRow(`SELECT luck_mp FROM move WHERE id = ?`, moveID).Scan(&luck); err != nil {
		t.Fatalf("read luck_mp: %v", err)
	}
	if luck.Valid {
		t.Errorf("migration must not invent luck: got luck_mp=%d, want NULL", luck.Int64)
	}
}

// TestMigration_TablesAheadOfVersion (issue #177): a database whose file
// carries tables its recorded version does not yet know — one that an
// earlier build's ensureAllTablesExist filled in without stamping, or a
// hand-repaired one — must still be walked to DatabaseVersion. The 1.x
// steps used to stop the chain on finding their table already there, which
// left such a file at 1.0.0 for good and skipped every later step.
func TestMigration_TablesAheadOfVersion(t *testing.T) {
	dbPath := filepath.Join(tempDir(t), "ahead.db")
	// Every 1.6.0 table, stamped 1.0.0.
	createOldDatabase(t, dbPath, "1.6.0")
	stampVersion(t, dbPath, "1.0.0")

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	closeOnCleanup(t, d)

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version != DatabaseVersion {
		t.Errorf("version after open = %s, want %s (chain stopped on a table already present)", version, DatabaseVersion)
	}
	for _, table := range allExpectedTables() {
		if !tableExists(d.db, table) {
			t.Errorf("table %s missing after migration", table)
		}
	}
	// The 2.0.0 step ran too: the scalar columns are there.
	if !columnExists(t, d.db, "position", "zobrist_hash") {
		t.Error("position.zobrist_hash missing: the 1.9.0→2.0.0 step did not run")
	}
}

// TestMigrate_2_16_0_to_2_17_0_SessionState builds a 2.16.0 database the way
// 2.16.0 wrote it — the current schema minus session_state, the session as
// six metadata rows for the desktop's empty scope and six "<scope>:"-prefixed
// rows for a tenant of a multi-tenant SQLite daemon — and checks that opening
// it moves both sessions into session_state, that the desktop reopens on the
// same search and views, that the tenant's session still belongs to that
// tenant only, and that metadata keeps nothing but its infrastructure rows.
func TestMigrate_2_16_0_to_2_17_0_SessionState(t *testing.T) {
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v2160.db")

	raw, err := sql.Open("sqlite", sqlite.DSN(dbPath))
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if err := sqlite.Bootstrap(context.Background(), raw); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	for _, stmt := range []string{
		`DROP TABLE session_state`,
		`UPDATE metadata SET value = '2.16.0' WHERE key = 'database_version'`,
		`INSERT INTO metadata (key, value) VALUES ('user', 'kevin'), ('description', 'ma base')`,
		// The desktop's session, unprefixed.
		`INSERT INTO metadata (key, value) VALUES
			('session_last_search_command', 'decision_type checker'),
			('session_last_search_position', 'xgid-desktop'),
			('session_last_position_index', '4'),
			('session_last_position_ids', '[1,2,3]'),
			('session_has_active_search', 'true'),
			('session_views', '{"tabs":["desktop"]}')`,
		// A daemon tenant's session, prefixed by its scope.
		`INSERT INTO metadata (key, value) VALUES
			('7:session_last_search_command', 'cube'),
			('7:session_last_search_position', 'xgid-seven'),
			('7:session_last_position_index', '1'),
			('7:session_last_position_ids', '[9]'),
			('7:session_has_active_search', 'false'),
			('7:session_views', '{"tabs":["seven"]}')`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("prepare v2.16.0 database (%s): %v", stmt, err)
		}
	}
	raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.16.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	defer d.db.Close()

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("CheckDatabaseVersion: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("version after migration: got %s, want %s", version, DatabaseVersion)
	}
	if !tableExists(d.db, "session_state") {
		t.Fatal("session_state should exist after migration")
	}

	// The desktop reopens on its own session.
	desktop, err := d.LoadSessionState()
	if err != nil {
		t.Fatalf("LoadSessionState: %v", err)
	}
	if desktop.LastSearchCommand != "decision_type checker" || desktop.LastSearchPosition != "xgid-desktop" ||
		desktop.LastPositionIndex != 4 || len(desktop.LastPositionIDs) != 3 || !desktop.HasActiveSearch ||
		desktop.ViewsJSON != `{"tabs":["desktop"]}` {
		t.Errorf("desktop session after migration: %+v", *desktop)
	}

	// The tenant's session followed it into its scope, and nowhere else.
	ctx := context.Background()
	seven, err := d.store.Session().Load(ctx, "7")
	if err != nil {
		t.Fatalf("load scope 7: %v", err)
	}
	if seven.LastSearchCommand != "cube" || seven.LastSearchPosition != "xgid-seven" ||
		seven.LastPositionIndex != 1 || len(seven.LastPositionIDs) != 1 || seven.HasActiveSearch ||
		seven.ViewsJSON != `{"tabs":["seven"]}` {
		t.Errorf("scope 7 session after migration: %+v", *seven)
	}
	if other, _ := d.store.Session().Load(ctx, "8"); other.LastSearchCommand != "" {
		t.Errorf("scope 8 sees another scope's session: %+v", *other)
	}

	// metadata is infrastructure again: version, user-facing fields, no
	// session row of any scope.
	md, err := d.store.Metadata().Load(ctx, "")
	if err != nil {
		t.Fatalf("Metadata.Load: %v", err)
	}
	for key := range md {
		if strings.Contains(key, "session_") {
			t.Errorf("metadata still holds %q after migration", key)
		}
	}
	if md["user"] != "kevin" || md["description"] != "ma base" {
		t.Errorf("metadata infrastructure rows disturbed: %v", md)
	}

	// Re-runnable: a second pass over the step finds nothing to move and
	// changes nothing.
	if err := d.migrate_2_16_0_to_2_17_0(ctx); err != nil {
		t.Fatalf("second run of migrate_2_16_0_to_2_17_0: %v", err)
	}
	if again, _ := d.LoadSessionState(); again.ViewsJSON != desktop.ViewsJSON {
		t.Errorf("second run altered the desktop session: %+v", *again)
	}
}

// legacyPositionColumns is the row copy the 2.18.0 test uses to plant, beside a
// position, the second row the old hash let in: every column but the hash and
// has_jacoby is carried over, so the two rows genuinely describe one position.
const legacyPositionColumns = `zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
	cube_value, cube_owner, score_1, score_2, match_length, has_jacoby, has_beaver,
	pip_1, pip_2, pip_diff, off_1, off_2, back_checkers_1, back_checkers_2,
	no_contact, occupancy_1, occupancy_2, point_mask_1, point_mask_2, state,
	is_cube_response, individually_imported, flagged`

// planLegacyJacobyTwin inserts the row a pre-2.18.0 blunderDB created when the
// same position was reached once with the Jacoby flag and once without: same
// board, same everything, hash XORed with the retired Jacoby key. It returns
// the new row's id.
func planLegacyJacobyTwin(t *testing.T, db *sql.DB, sourceID int64, hash uint64) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO position (`+legacyPositionColumns+`)
		SELECT ?, decision_type, player_on_roll, dice_1, dice_2,
			cube_value, cube_owner, score_1, score_2, match_length, 1, has_beaver,
			pip_1, pip_2, pip_diff, off_1, off_2, back_checkers_1, back_checkers_2,
			no_contact, occupancy_1, occupancy_2, point_mask_1, point_mask_2, state,
			is_cube_response, individually_imported, flagged
		FROM position WHERE id = ?`,
		int64(hash^engine.RetiredFlagDelta(1, 0)), sourceID)
	if err != nil {
		t.Fatalf("plant legacy twin of position %d: %v", sourceID, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// TestMigrate_2_17_0_to_2_18_0_JacobyAndBeaverLeaveTheIdentity (issue #171,
// ADR-0028): the rule flags come out of the Zobrist hash, every stored hash
// that carried one is converted, and the rows the conversion brings together —
// which were one position all along — are merged onto the oldest of them.
func TestMigrate_2_17_0_to_2_18_0_JacobyAndBeaverLeaveTheIdentity(t *testing.T) {
	dbPath := filepath.Join(tempDir(t), "test_v2170.db")

	// Build the fixture at the current schema through the normal write path,
	// then walk the hashes back to what 2.17.0 would have stored.
	setup := NewDatabase()
	if err := setup.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}

	untouched := InitializePosition()

	lone := InitializePosition()
	lone.Board.Points[13].Checkers = 4
	lone.Board.Points[8].Checkers = 2
	lone.HasJacoby = 1

	younger := InitializePosition() // the twin planted below is NEWER than it
	younger.Board.Points[6].Checkers = 4
	younger.Board.Points[24].Checkers = 1

	elder := InitializePosition() // the flagged row is OLDER than its twin
	elder.Board.Points[19].Checkers = 4
	elder.Board.Points[17].Checkers = 1
	elder.HasJacoby = 1

	ids := map[string]int64{}
	for name, pos := range map[string]*Position{
		"untouched": &untouched, "lone": &lone, "younger": &younger,
	} {
		id, err := setup.SavePosition(pos)
		if err != nil {
			t.Fatalf("SavePosition(%s): %v", name, err)
		}
		ids[name] = id
	}
	// The elder flagged row must precede its twin, so it is written first and
	// the twin is planted afterwards.
	elderID, err := setup.SavePosition(&elder)
	if err != nil {
		t.Fatalf("SavePosition(elder): %v", err)
	}
	ids["elder"] = elderID
	if err := setup.SaveComment(ids["younger"], "note du plus ancien"); err != nil {
		t.Fatalf("SaveComment: %v", err)
	}
	setup.Close()

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	// The twin of `younger`: a newer row carrying the Jacoby flag, hashed the
	// 2.17.0 way. It is the one that must disappear.
	youngerTwin := planLegacyJacobyTwin(t, raw, ids["younger"], engine.ZobristHash(&younger))
	if _, err := raw.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`,
		youngerTwin, "note du doublon"); err != nil {
		t.Fatalf("comment on the twin: %v", err)
	}
	if _, err := raw.Exec(`UPDATE position SET individually_imported = 1 WHERE id = ?`, youngerTwin); err != nil {
		t.Fatalf("mark the twin: %v", err)
	}
	// Walk every flagged row's hash back to what 2.17.0 stored, and stamp the
	// file with that version.
	for _, tc := range []struct {
		id   int64
		hash uint64
	}{
		{ids["lone"], engine.ZobristHash(&lone) ^ engine.RetiredFlagDelta(1, 0)},
		{ids["elder"], engine.ZobristHash(&elder) ^ engine.RetiredFlagDelta(1, 0)},
	} {
		if _, err := raw.Exec(`UPDATE position SET zobrist_hash = ? WHERE id = ?`, int64(tc.hash), tc.id); err != nil {
			t.Fatalf("legacy hash for %d: %v", tc.id, err)
		}
	}

	// The twin of `elder`: a NEWER row without the flag, at the hash `elder`
	// is about to take — which is free only now that `elder` has moved back to
	// its 2.17.0 hash. Here the flagged row is the keeper.
	res, err := raw.Exec(`INSERT INTO position (`+legacyPositionColumns+`)
		SELECT ?, decision_type, player_on_roll, dice_1, dice_2,
			cube_value, cube_owner, score_1, score_2, match_length, 0, has_beaver,
			pip_1, pip_2, pip_diff, off_1, off_2, back_checkers_1, back_checkers_2,
			no_contact, occupancy_1, occupancy_2, point_mask_1, point_mask_2, state,
			is_cube_response, individually_imported, flagged
		FROM position WHERE id = ?`, int64(engine.ZobristHash(&elder)), ids["elder"])
	if err != nil {
		t.Fatalf("plant the elder's twin: %v", err)
	}
	elderTwin, _ := res.LastInsertId()

	if _, err := raw.Exec(`UPDATE metadata SET value = '2.17.0' WHERE key = 'database_version'`); err != nil {
		t.Fatalf("stamp 2.17.0: %v", err)
	}
	raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.17.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	if version, err := d.CheckDatabaseVersion(); err != nil || version != DatabaseVersion {
		t.Fatalf("version after migration: got %q (%v), want %s", version, err, DatabaseVersion)
	}

	hashOf := func(id int64) (uint64, bool) {
		var h sql.NullInt64
		switch err := d.db.QueryRow(`SELECT zobrist_hash FROM position WHERE id = ?`, id).Scan(&h); {
		case errors.Is(err, sql.ErrNoRows):
			return 0, false
		case err != nil:
			t.Fatalf("read hash of %d: %v", id, err)
		}
		return uint64(h.Int64), true
	}

	// A row that never carried a flag is not touched at all.
	if got, ok := hashOf(ids["untouched"]); !ok || got != engine.ZobristHash(&untouched) {
		t.Errorf("untouched position: hash %#x (present=%v), want %#x", got, ok, engine.ZobristHash(&untouched))
	}
	// A flagged row with no counterpart is simply rehashed.
	if got, ok := hashOf(ids["lone"]); !ok || got != engine.ZobristHash(&lone) {
		t.Errorf("lone flagged position: hash %#x (present=%v), want %#x", got, ok, engine.ZobristHash(&lone))
	}
	// The newer flagged twin is folded into the older row, which keeps its id,
	// takes the duplicate's comment and inherits its sticky mark.
	if _, ok := hashOf(youngerTwin); ok {
		t.Error("the newer duplicate should have been merged away")
	}
	if got, ok := hashOf(ids["younger"]); !ok || got != engine.ZobristHash(&younger) {
		t.Errorf("kept position: hash %#x (present=%v), want %#x", got, ok, engine.ZobristHash(&younger))
	}
	var comments int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM comment WHERE position_id = ?`, ids["younger"]).Scan(&comments); err != nil {
		t.Fatal(err)
	}
	if comments != 2 {
		t.Errorf("comments on the kept position: got %d, want 2 (its own and the duplicate's)", comments)
	}
	var individual int
	if err := d.db.QueryRow(`SELECT individually_imported FROM position WHERE id = ?`, ids["younger"]).Scan(&individual); err != nil {
		t.Fatal(err)
	}
	if individual != 1 {
		t.Error("the duplicate's sticky provenance must be raised on the kept row (ADR-0001)")
	}
	// The other direction: the flagged row is the older one, so it survives and
	// takes the hash the unflagged twin was holding.
	if _, ok := hashOf(elderTwin); ok {
		t.Error("the newer unflagged duplicate should have been merged away")
	}
	if got, ok := hashOf(ids["elder"]); !ok || got != engine.ZobristHash(&elder) {
		t.Errorf("elder flagged position: hash %#x (present=%v), want %#x", got, ok, engine.ZobristHash(&elder))
	}

	// The step is not idempotent — XOR is its own inverse — so what has to hold
	// is that reopening the file never runs it again: the version is stamped,
	// and a second open leaves every hash where the first one put it.
	d.Close()
	again := NewDatabase()
	if err := again.OpenDatabase(dbPath); err != nil {
		t.Fatalf("reopen the migrated database: %v", err)
	}
	closeOnCleanup(t, again)
	for name, want := range map[string]uint64{
		"untouched": engine.ZobristHash(&untouched),
		"lone":      engine.ZobristHash(&lone),
		"younger":   engine.ZobristHash(&younger),
		"elder":     engine.ZobristHash(&elder),
	} {
		var h int64
		if err := again.db.QueryRow(`SELECT zobrist_hash FROM position WHERE id = ?`, ids[name]).Scan(&h); err != nil {
			t.Fatalf("reopen: read hash of %s: %v", name, err)
		}
		if uint64(h) != want {
			t.Errorf("reopen moved the hash of %s: got %#x, want %#x", name, uint64(h), want)
		}
	}
}

// TestMigrate_2_17_0_to_2_18_0_OneAnalysisPerPosition (issue #173): the second
// half of the 2.18.0 wave. A position had no constraint saying it holds one
// analysis, and Save's SELECT-then-INSERT let two rows through; the migration
// keeps the last one written and the index makes the state unreachable.
func TestMigrate_2_17_0_to_2_18_0_OneAnalysisPerPosition(t *testing.T) {
	dbPath := filepath.Join(tempDir(t), "test_v2170_analysis.db")

	setup := NewDatabase()
	if err := setup.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	pos := InitializePosition()
	posID, err := setup.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	setup.Close()

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	// Back to 2.17.0: the index of that name was not unique, which is what let
	// the second row in.
	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_analysis_position`,
		`CREATE INDEX idx_analysis_position ON analysis(position_id)`,
		`UPDATE metadata SET value = '2.17.0' WHERE key = 'database_version'`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("prepare v2.17.0 database (%s): %v", stmt, err)
		}
	}
	for _, action := range []string{"superseded", "kept"} {
		if _, err := raw.Exec(
			`INSERT INTO analysis (position_id, data, best_cube_action) VALUES (?, ?, ?)`,
			posID, []byte(`{}`), action); err != nil {
			t.Fatalf("plant analysis %q: %v", action, err)
		}
	}
	raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.17.0 database: %v", err)
	}
	closeOnCleanup(t, d)

	if version, err := d.CheckDatabaseVersion(); err != nil || version != DatabaseVersion {
		t.Fatalf("version after migration: got %q (%v), want %s", version, err, DatabaseVersion)
	}

	var rows int
	var kept string
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM analysis WHERE position_id = ?`, posID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("analyses for the position after migration: got %d, want 1", rows)
	}
	if err := d.db.QueryRow(`SELECT best_cube_action FROM analysis WHERE position_id = ?`, posID).Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if kept != "kept" {
		t.Errorf("the surviving analysis is %q, want the last one written (%q)", kept, "kept")
	}

	// The index is unique now, so the state cannot come back.
	var unique int
	if err := d.db.QueryRow(
		`SELECT "unique" FROM pragma_index_list('analysis') WHERE name = 'idx_analysis_position'`).Scan(&unique); err != nil {
		t.Fatalf("read idx_analysis_position: %v", err)
	}
	if unique != 1 {
		t.Error("idx_analysis_position must be UNIQUE after the migration")
	}
	if _, err := d.db.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, posID, []byte(`{}`)); err == nil {
		t.Error("a second analysis row for one position must be refused")
	}

	// And CheckConstraints agrees the file is clean.
	violations, err := d.CheckConstraints()
	if err != nil {
		t.Fatalf("CheckConstraints: %v", err)
	}
	if n := TotalConstraintViolations(violations); n != 0 {
		t.Errorf("CheckConstraints on the migrated database: %d violation(s), want 0: %+v", n, violations)
	}
}
