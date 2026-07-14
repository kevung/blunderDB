package database

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

// columnExists checks if a column exists on a table.
func columnExists(db *sql.DB, table, column string) bool {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

// TestMigrate_2_7_0_to_2_8_0 verifies the exclude_position column is added to
// search_history and filter_library and that the "Sauf" structure round-trips.
func TestMigrate_2_7_0_to_2_8_0(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_v270.db")
	createOldDatabase(t, dbPath, "2.7.0")

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("Failed to open v2.7.0 database: %v", err)
	}

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("Expected version %s after migration, got %s", DatabaseVersion, version)
	}

	if !columnExists(d.db, "search_history", "exclude_position") {
		t.Errorf("search_history.exclude_position should exist after migration")
	}
	if !columnExists(d.db, "filter_library", "exclude_position") {
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
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	var posCount2 int
	d2.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount2)
	if posCount2 != posCount1 {
		t.Errorf("position count changed on second open: %d → %d", posCount1, posCount2)
	}
}

// TestMigrate_2_3_0_to_2_4_0_RepairsMoveError verifies that the 2.3.0→2.4.0
// migration correctly backfills best_move_equity_error for positions where
// PlayedMoves was missing from the analysis JSON blob.
func TestMigrate_2_3_0_to_2_4_0_RepairsMoveError(t *testing.T) {
	tmpDir := t.TempDir()
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
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (1, '{}', 0)`)
	rawDB.Exec(`INSERT INTO match (id, player1_name, player2_name, match_length) VALUES (1,'A','B',7)`)
	rawDB.Exec(`INSERT INTO game (id, match_id, game_number, initial_score_1, initial_score_2, winner, points_won) VALUES (1,1,1,0,0,0,1)`)
	rawDB.Exec(`INSERT INTO move (id, game_id, move_number, move_type, position_id, player, checker_move) VALUES (1,1,1,'checker',1,1,'24/18 13/11')`)
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

	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	var moveErr float64
	d.db.QueryRow(`SELECT best_move_equity_error FROM analysis WHERE id = 1`).Scan(&moveErr)
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
	tmpDir := t.TempDir()
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

	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (1,'{}',0)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (2,'{}',0)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (3,'{}',1)`)
	rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (1, 1, ?)`, forced)
	rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (2, 2, ?)`, unforced)
	rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (3, 3, ?)`, cube)
	rawDB.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}

	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	// is_forced column must exist
	var colExists int
	d.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('analysis') WHERE name='is_forced'`).Scan(&colExists)
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
		d.db.QueryRow(`SELECT is_forced FROM analysis WHERE id = ?`, tc.id).Scan(&got)
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
	tmpDir := t.TempDir()
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

	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (1,'{}',1)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (2,'{}',1)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (3,'{}',1)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (4,'{}',0)`)
	rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (1, 1, ?)`, close1)
	rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (2, 2, ?)`, notClose)
	rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (3, 3, ?)`, takeDec)
	rawDB.Exec(`INSERT INTO analysis (id, position_id, data) VALUES (4, 4, ?)`, checker)
	rawDB.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}

	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	var colExists int
	d.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('analysis') WHERE name='is_close_cube'`).Scan(&colExists)
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
		d.db.QueryRow(`SELECT is_close_cube FROM analysis WHERE id = ?`, tc.id).Scan(&got)
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
	tmpDir := t.TempDir()
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
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (1,'{}',1)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (2,'{}',1)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (3,'{}',1)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (4,'{}',1)`)
	rawDB.Exec(`INSERT INTO position (id, state, decision_type) VALUES (5,'{}',0)`)
	rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, cube_action) VALUES (1,1,'cube',1,'Take')`)
	rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, cube_action) VALUES (1,2,'cube',2,'Double')`)
	rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, cube_action) VALUES (1,3,'cube',3,'No Double')`)
	rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, cube_action) VALUES (1,4,'cube',4,'Pass')`)
	rawDB.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}

	ver, _ := d.CheckDatabaseVersion()
	if ver != DatabaseVersion {
		t.Errorf("expected version %s, got %s", DatabaseVersion, ver)
	}

	var colExists int
	d.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('position') WHERE name='is_cube_response'`).Scan(&colExists)
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
		d.db.QueryRow(`SELECT is_cube_response FROM position WHERE id = ?`, tc.id).Scan(&got)
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
	tmpDir := t.TempDir()
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
	defer d.db.Close()

	version, err := d.CheckDatabaseVersion()
	if err != nil {
		t.Fatalf("CheckDatabaseVersion: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("version after migration: got %s, want %s", version, DatabaseVersion)
	}
	if !columnExists(d.db, "position", "individually_imported") {
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
