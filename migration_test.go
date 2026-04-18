package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
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

// tableExists checks if a table exists in the database
func tableExists(db *sql.DB, tableName string) bool {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tableName).Scan(&name)
	return err == nil && name == tableName
}

// getDBVersion reads the database_version from metadata
func getDBVersion(db *sql.DB) string {
	var version string
	err := db.QueryRow(`SELECT value FROM metadata WHERE key='database_version'`).Scan(&version)
	if err != nil {
		return ""
	}
	return version
}

// allExpectedTables returns all tables expected at the latest version
func allExpectedTables() []string {
	return []string{
		"position", "analysis", "comment", "metadata",
		"command_history",
		"filter_library",
		"search_history",
		"match", "game", "move", "move_analysis",
		"collection", "collection_position",
		"tournament",
	}
}

// TestMigrationFromV100 tests migration from a v1.0.0 database (only base tables)
func TestMigrationFromV100(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v100.db")
	createOldDatabase(t, dbPath, "1.0.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.0.0 database: %v", err)
	}

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
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v110.db")
	createOldDatabase(t, dbPath, "1.1.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.1.0 database: %v", err)
	}

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
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v120.db")
	createOldDatabase(t, dbPath, "1.2.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.2.0 database: %v", err)
	}

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
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v130.db")
	createOldDatabase(t, dbPath, "1.3.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.3.0 database: %v", err)
	}

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
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v140.db")
	createOldDatabase(t, dbPath, "1.4.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.4.0 database: %v", err)
	}

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
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v150.db")
	createOldDatabase(t, dbPath, "1.5.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.5.0 database: %v", err)
	}

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
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v170.db")
	createOldDatabase(t, dbPath, "1.7.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.7.0 database: %v", err)
	}

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
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v160.db")
	createOldDatabase(t, dbPath, "1.6.0")

	d := NewDatabase()
	err := d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open v1.6.0 database: %v", err)
	}

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
	tmpDir := t.TempDir()
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
	db.Exec(`INSERT INTO position (state) VALUES (?)`, string(norm1))
	db.Exec(`INSERT INTO position (state) VALUES (?)`, string(norm2))
	db.Exec(`INSERT INTO comment (position_id, text) VALUES (1, 'test comment')`)
	db.Close()

	// Open with migration
	d := NewDatabase()
	err = d.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

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
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test.db")
			createOldDatabase(t, dbPath, startVersion)

			d := NewDatabase()
			err := d.OpenDatabase(dbPath)
			if err != nil {
				t.Fatalf("Failed to open %s database: %v", startVersion, err)
			}

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
		})
	}
}

// TestSetupThenOpen tests that a database created by SetupDatabase can be opened
func TestSetupThenOpen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_setup.db")

	d := NewDatabase()
	err := d.SetupDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to setup database: %v", err)
	}

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
	tmpDir := t.TempDir()
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
	tmpDir := t.TempDir()
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
	tmpDir := t.TempDir()
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
	db.Exec(`PRAGMA foreign_keys = ON`)

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
		db.Exec(`INSERT INTO position (state) VALUES (?)`, string(data))
	}

	// Insert analyses for pos1 and pos2 (pos3 has none)
	db.Exec(`INSERT INTO analysis (position_id, data) VALUES (1, '{"bestMove":"13/11 24/23","playedMove":"13/11 24/23"}')`)
	db.Exec(`INSERT INTO analysis (position_id, data) VALUES (2, '{}')`)

	// Insert a match -> 2 games -> 5 moves (referencing all 3 positions)
	db.Exec(`INSERT INTO match (player1_name, player2_name, match_length) VALUES ('Alice','Bob',7)`)
	db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won) VALUES (1,1,0,0,0,1)`)
	db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won) VALUES (1,2,1,0,1,1)`)
	for i := 1; i <= 5; i++ {
		posID := ((i - 1) % 3) + 1
		db.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id) VALUES (?,?,?,?)`, 1, i, "checker", posID)
	}

	// Insert a collection with pos1 in it
	db.Exec(`INSERT INTO collection (name) VALUES ('Test collection')`)
	db.Exec(`INSERT INTO collection_position (collection_id, position_id) VALUES (1, 1)`)

	// Insert a tournament
	db.Exec(`INSERT INTO tournament (name) VALUES ('Test tournament')`)
}

// TestMigrate_1_9_0_to_2_0_0 opens a v1.9.0 database and verifies that:
//   - version is bumped to 2.0.0
//   - all new scalar columns are non-NULL for every position
//   - stored column values match what populatePositionColumns recomputes
//   - the v2.0.0 indexes exist
func TestMigrate_1_9_0_to_2_0_0(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "v190.db")
	createV190Database(t, dbPath)

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase failed: %v", err)
	}

	// Version must be 2.0.0
	ver, _ := d.CheckDatabaseVersion()
	if ver != "2.0.0" {
		t.Fatalf("expected version 2.0.0, got %s", ver)
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
		var pos Position
		if err := json.Unmarshal([]byte(state), &pos); err != nil {
			t.Fatalf("position %d: unmarshal: %v", id, err)
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
	if count != 3 {
		t.Errorf("expected 3 positions, got %d", count)
	}

	// Check that key indexes exist
	for _, idx := range []string{"idx_position_zobrist", "idx_position_decision_pip", "idx_analysis_position"} {
		var name string
		d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&name)
		if name != idx {
			t.Errorf("index %s not found after migration", idx)
		}
	}
}

// TestMigrate_1_9_0_Duplicates verifies that two positions with the same
// Zobrist hash are merged during migration and FK references are remapped.
func TestMigrate_1_9_0_Duplicates(t *testing.T) {
	tmpDir := t.TempDir()
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
	db.Exec(`INSERT INTO position (state) VALUES (?)`, jsonStr) // id=1
	db.Exec(`INSERT INTO position (state) VALUES (?)`, jsonStr) // id=2 — exact duplicate
	db.Exec(`INSERT INTO match (player1_name, player2_name, match_length) VALUES ('A','B',7)`)
	db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won) VALUES (1,1,0,0,0,1)`)
	// Move pointing at the duplicate (id=2)
	db.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id) VALUES (1, 1, 'checker', 2)`)
	db.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}

	// Only one position should remain
	var posCount int
	d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount)
	if posCount != 1 {
		t.Errorf("expected 1 position after dedup, got %d", posCount)
	}

	// The move must now point at the kept position (id=1)
	var movePosID sql.NullInt64
	d.db.QueryRow(`SELECT position_id FROM move WHERE id=1`).Scan(&movePosID)
	if !movePosID.Valid || movePosID.Int64 != 1 {
		t.Errorf("move.position_id should be 1 after dedup, got %v", movePosID)
	}
}

// TestMigrate_Idempotent verifies that running migration twice (opening a
// fully migrated 2.0.0 DB a second time) is a no-op.
func TestMigrate_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "v190_idempotent.db")
	createV190Database(t, dbPath)

	// First open → migrates 1.9.0 → 2.0.0
	d1 := NewDatabase()
	if err := d1.OpenDatabase(dbPath); err != nil {
		t.Fatalf("first open: %v", err)
	}
	var posCount1 int
	d1.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount1)

	// Second open → must succeed without error and leave data unchanged
	d2 := NewDatabase()
	if err := d2.OpenDatabase(dbPath); err != nil {
		t.Fatalf("second open (idempotent): %v", err)
	}

	ver, _ := d2.CheckDatabaseVersion()
	if ver != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", ver)
	}

	var posCount2 int
	d2.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount2)
	if posCount2 != posCount1 {
		t.Errorf("position count changed on second open: %d → %d", posCount1, posCount2)
	}
}
