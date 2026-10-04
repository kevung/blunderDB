package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
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
	t.Parallel()
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
		"transcription",
	}
}

// TestMigrationFromV100 tests migration from a v1.0.0 database (only base tables)
func TestMigrationFromV100(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	for _, idx := range []string{"idx_position_zobrist", "idx_position_pip_diff", "idx_analysis_position"} {
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestMigration_TablesAheadOfVersion: a file carrying tables its recorded
// version does not know (filled in by ensureAllTablesExist without stamping,
// or hand-repaired) must still be walked to DatabaseVersion — a step that
// stops the chain on finding its table would strand the file at 1.0.0.
func TestMigration_TablesAheadOfVersion(t *testing.T) {
	t.Parallel()
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

// TestMigrate_2_16_0_to_2_17_0_SessionState builds a 2.16.0 file (session as
// metadata rows, empty scope and one "<scope>:" tenant) and checks both
// sessions move to session_state, each scope keeps its own, and metadata keeps
// only its infrastructure rows.
func TestMigrate_2_16_0_to_2_17_0_SessionState(t *testing.T) {
	t.Parallel()
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

// TestMigrate_2_17_0_to_2_18_0_JacobyAndBeaverLeaveTheIdentity (ADR-0028):
// the rule flags come out of the Zobrist hash, every stored hash
// that carried one is converted, and the rows the conversion brings together —
// which were one position all along — are merged onto the oldest of them.
func TestMigrate_2_17_0_to_2_18_0_JacobyAndBeaverLeaveTheIdentity(t *testing.T) {
	t.Parallel()
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

// TestMigrate_2_17_0_to_2_18_0_OneAnalysisPerPosition: racing
// SELECT-then-INSERT saves could leave two analysis rows; the migration keeps
// the last one written and the UNIQUE index makes the state unreachable.
func TestMigrate_2_17_0_to_2_18_0_OneAnalysisPerPosition(t *testing.T) {
	t.Parallel()
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

// TestMigrate_2_19_0_to_2_20_0_MaxCube: an older row stated no cube ceiling,
// so 0 is its truth; the migration must invent nothing, nor rehash — max_cube
// is not part of the position's identity.
func TestMigrate_2_19_0_to_2_20_0_MaxCube(t *testing.T) {
	t.Parallel()
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v2190.db")
	createOldDatabase(t, dbPath, "2.19.0")

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	res, err := raw.Exec(`INSERT INTO position (state) VALUES ('{}')`)
	if err != nil {
		t.Fatalf("insert position: %v", err)
	}
	posID, _ := res.LastInsertId()
	raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.19.0 database: %v", err)
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
	if !columnExists(t, d.db, "position", "max_cube") {
		t.Fatal("position.max_cube should exist after migration")
	}

	var maxCube int
	if err := d.db.QueryRow(`SELECT max_cube FROM position WHERE id = ?`, posID).Scan(&maxCube); err != nil {
		t.Fatalf("read max_cube: %v", err)
	}
	if maxCube != 0 {
		t.Errorf("migration must not invent a cube ceiling: got max_cube=%d, want 0", maxCube)
	}
}

// TestMigrate_2_20_0_to_2_21_0_Transcription pins the 2.21.0 wave's one table.
// A database that predates it holds no draft — a transcription is the typing
// that produces a Match, not something derivable from a saved one (ADR-0045
// §2) — so the migration creates the table and leaves it empty, and the
// matches the file already carried are untouched.
func TestMigrate_2_20_0_to_2_21_0_Transcription(t *testing.T) {
	t.Parallel()
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v2200.db")
	createOldDatabase(t, dbPath, "2.20.0")

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.20.0 database: %v", err)
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
	if !tableExists(d.db, "transcription") {
		t.Fatal("transcription should exist after migration")
	}
	for _, col := range []string{"id", "created_at", "updated_at", "format_version", "match_id", "label", "document"} {
		if !columnExists(t, d.db, "transcription", col) {
			t.Errorf("transcription.%s should exist after migration", col)
		}
	}

	var n int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM transcription`).Scan(&n); err != nil {
		t.Fatalf("count transcriptions: %v", err)
	}
	if n != 0 {
		t.Errorf("the migration must invent no draft: got %d rows, want 0", n)
	}

	// The link to a match is nullable and gives way: a draft survives the
	// deletion of the match it produced (ADR-0045 §2).
	res, err := d.db.Exec(`INSERT INTO match (player1_name, player2_name, match_length) VALUES ('A','B',5)`)
	if err != nil {
		t.Fatalf("insert match: %v", err)
	}
	matchID, _ := res.LastInsertId()
	if _, err := d.db.Exec(
		`INSERT INTO transcription (format_version, match_id, label, document) VALUES ('1', ?, 'A — B', '{}')`,
		matchID); err != nil {
		t.Fatalf("insert transcription: %v", err)
	}
	if _, err := d.db.Exec(`DELETE FROM match WHERE id = ?`, matchID); err != nil {
		t.Fatalf("delete match: %v", err)
	}
	var linked sql.NullInt64
	var document string
	if err := d.db.QueryRow(`SELECT match_id, document FROM transcription`).Scan(&linked, &document); err != nil {
		t.Fatalf("read transcription back: %v", err)
	}
	if linked.Valid {
		t.Errorf("deleting the match must null the link, not keep it: got %d", linked.Int64)
	}
	if document != "{}" {
		t.Errorf("the typing must survive its match: got document %q", document)
	}
}

// TestMigrate_2_19_0_to_2_22_0_TrainingJournal walks the chain across TWO
// waves on purpose: a 2.19.0 file has to pass through the transcription step
// (2.20.0 → 2.21.0) and then the Training journal's (2.21.0 → 2.22.0). A
// renumbering that left the two steps colliding, or the journal registered
// before the wave it follows, is invisible on a file that only needs one step
// and shows up here.
//
// The journal's own statement is the second half: the two tables exist and
// they are EMPTY. The fifty-session JSON key the training bar used to write in
// `metadata` is deliberately not imported — it held a per-session summary with
// no per-number detail, which is the one thing the journal exists for
// (ADR-0040 rule 6).
func TestMigrate_2_19_0_to_2_22_0_TrainingJournal(t *testing.T) {
	t.Parallel()
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v2190_chain.db")
	createOldDatabase(t, dbPath, "2.19.0")

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.19.0 database: %v", err)
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

	// Both waves landed, in order: the earlier one's table is there too.
	if !tableExists(d.db, "transcription") {
		t.Error("the transcription wave (2.21.0) must have run before the journal's")
	}
	for _, table := range []string{"training_session", "training_item"} {
		var n int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
			t.Fatalf("%s should exist after migration: %v", table, err)
		}
		if n != 0 {
			t.Errorf("%s holds %d row(s) after migration, want 0: nothing is backfilled", table, n)
		}
	}

	// And the journal works through the wrapper the GUI binds, on the file
	// that was just migrated — a table that exists but refuses a write would
	// pass the check above and fail the user.
	id, err := d.SaveTrainingSession(storage.TrainingSession{
		Exercise:     "scores",
		SeedSource:   "pool",
		NumbersAsked: 1,
		Items:        []storage.TrainingItem{{NumberType: "gv1", Wrong: true}},
	})
	if err != nil {
		t.Fatalf("SaveTrainingSession on the migrated database: %v", err)
	}
	if id == 0 {
		t.Error("SaveTrainingSession returned id 0")
	}
	stats, err := d.LoadTrainingNumberStats("scores")
	if err != nil {
		t.Fatalf("LoadTrainingNumberStats: %v", err)
	}
	if len(stats) != 1 || stats[0].NumberType != "gv1" || stats[0].Faults != 1 {
		t.Errorf("LoadTrainingNumberStats = %+v, want one gv1 with 1 fault", stats)
	}
}

// TestMigrate_2_22_0_to_2_23_0_AnkiCardKinds walks the step that lets an Anki
// card be something other than a position (ADR-0042) over a
// database holding the two anki tables in their pre-2.23.0 shape — the shape
// createOldDatabase does not build, because every other test gets them fresh
// from EnsureSchema and would never see the rebuild at all.
//
// What the step owes the user, in order: the cards already there keep their
// meaning and gain a key; the review journal keeps its rows and its foreign
// keys through the rebuild; and a card with no position becomes writable,
// which is the whole point.
func TestMigrate_2_22_0_to_2_23_0_AnkiCardKinds(t *testing.T) {
	t.Parallel()
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test_v2220.db")
	createOldDatabase(t, dbPath, "2.22.0")

	// The anki tables as 2.22.0 declared them: position_id mandatory, and a
	// deck holding one card per position.
	func() {
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			t.Fatalf("open the old database: %v", err)
		}
		defer db.Close()
		if _, err := db.Exec(`
			CREATE TABLE anki_deck (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				description TEXT DEFAULT '',
				source_type TEXT NOT NULL DEFAULT 'collection',
				source_id INTEGER DEFAULT 0,
				source_command TEXT DEFAULT '',
				request_retention REAL DEFAULT 0.9,
				maximum_interval REAL DEFAULT 36500,
				enable_fuzz INTEGER DEFAULT 1,
				session_limit INTEGER,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE anki_card (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				deck_id INTEGER NOT NULL,
				position_id INTEGER NOT NULL,
				due DATETIME DEFAULT CURRENT_TIMESTAMP,
				stability REAL DEFAULT 0,
				difficulty REAL DEFAULT 0,
				elapsed_days INTEGER DEFAULT 0,
				scheduled_days INTEGER DEFAULT 0,
				reps INTEGER DEFAULT 0,
				lapses INTEGER DEFAULT 0,
				state INTEGER DEFAULT 0,
				last_review DATETIME DEFAULT '',
				suspended INTEGER NOT NULL DEFAULT 0,
				buried_until DATETIME,
				FOREIGN KEY(deck_id) REFERENCES anki_deck(id) ON DELETE CASCADE,
				FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
				UNIQUE(deck_id, position_id)
			);
			CREATE TABLE anki_review_log (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				card_id INTEGER NOT NULL,
				deck_id INTEGER NOT NULL,
				position_id INTEGER NOT NULL,
				rating INTEGER NOT NULL,
				state INTEGER NOT NULL DEFAULT 0,
				stability REAL DEFAULT 0,
				difficulty REAL DEFAULT 0,
				elapsed_days INTEGER DEFAULT 0,
				scheduled_days INTEGER DEFAULT 0,
				reviewed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			INSERT INTO position (id, state) VALUES (7, '');
			INSERT INTO anki_deck (id, name) VALUES (1, 'blunders');
			INSERT INTO anki_card (id, deck_id, position_id) VALUES (3, 1, 7);
			INSERT INTO anki_review_log (id, card_id, deck_id, position_id, rating)
				VALUES (5, 3, 1, 7, 3);
		`); err != nil {
			t.Fatalf("build the 2.22.0 anki tables: %v", err)
		}
	}()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.22.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	defer d.db.Close()

	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration: got %s (err %v), want %s", v, err, DatabaseVersion)
	}

	// The card that was there is a position card, and its key names the
	// position it always named.
	var kind, key string
	if err := d.db.QueryRow(`SELECT kind, key FROM anki_card WHERE id = 3`).Scan(&kind, &key); err != nil {
		t.Fatalf("read the migrated card: %v", err)
	}
	if kind != domain.AnkiKindPosition || key != "7" {
		t.Errorf("migrated card: got kind %q key %q, want %q / \"7\"", kind, key, domain.AnkiKindPosition)
	}
	if err := d.db.QueryRow(`SELECT kind, key FROM anki_review_log WHERE id = 5`).Scan(&kind, &key); err != nil {
		t.Fatalf("read the migrated review: %v", err)
	}
	if kind != domain.AnkiKindPosition || key != "7" {
		t.Errorf("migrated review: got kind %q key %q, want %q / \"7\"", kind, key, domain.AnkiKindPosition)
	}

	// The journal survived the rebuild of its own table — and came out of it
	// with the foreign keys a fresh database declares and it never had.
	var reviews int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM anki_review_log`).Scan(&reviews); err != nil {
		t.Fatalf("count the reviews: %v", err)
	}
	if reviews != 1 {
		t.Errorf("reviews after the rebuild: got %d, want 1 — the rebuild must not lose the journal", reviews)
	}
	var fks int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_list('anki_review_log')`).Scan(&fks); err != nil {
		t.Fatalf("read the review log's foreign keys: %v", err)
	}
	if fks != 3 {
		t.Errorf("review log foreign keys: got %d, want 3 (card, deck, position)", fks)
	}

	// And the point of the whole step: a card that asks about a score, with no
	// position at all, is writable.
	if _, err := d.db.Exec(
		`INSERT INTO anki_card (deck_id, kind, key, position_id) VALUES (1, ?, '3:5', NULL)`,
		domain.AnkiKindScore); err != nil {
		t.Fatalf("insert a score card: %v", err)
	}
	// Twice is once: the deck holds one card per question.
	if _, err := d.db.Exec(
		`INSERT INTO anki_card (deck_id, kind, key, position_id) VALUES (1, ?, '3:5', NULL)`,
		domain.AnkiKindScore); err == nil {
		t.Error("inserting the same score twice in one deck must fail on idx_anki_card_identity")
	}
}

// TestDirectionSchema_2_24_0 covers the Direction tables and the Slot column
// (ADR-0047). SQLite cannot forbid an UPDATE, so the test pins what makes
// append-only enforceable: the (tournament_id, seq) primary key, where a
// second write at the same seq collides instead of overwriting.
func TestDirectionSchema_2_24_0(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	tID, err := d.CreateTournament("Open de Lyon", "2026-09-07", "Lyon")
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	if _, err := d.db.ExecContext(ctx,
		`INSERT INTO direction (tournament_id, format_version, engine_version, state, config)
		 VALUES (?, 1, 'v0.1.0', 'draft', '{}')`, tID); err != nil {
		t.Fatalf("insert direction: %v", err)
	}
	for seq := 0; seq < 3; seq++ {
		if _, err := d.db.ExecContext(ctx,
			`INSERT INTO direction_event (tournament_id, seq, kind, time, payload)
			 VALUES (?, ?, 'created', CURRENT_TIMESTAMP, '{}')`, tID, seq); err != nil {
			t.Fatalf("insert event %d: %v", seq, err)
		}
	}

	// Append-only: writing sequence 1 again collides rather than replacing it.
	if _, err := d.db.ExecContext(ctx,
		`INSERT INTO direction_event (tournament_id, seq, kind, time, payload)
		 VALUES (?, 1, 'result', CURRENT_TIMESTAMP, '{}')`, tID); err == nil {
		t.Error("a second event at the same sequence number must collide: the log is append-only")
	}

	// Deleting the Tournament takes its Direction with it and unlinks its
	// Matches, which is the rule ADR-0047 states: the Matches keep their data.
	res, err := d.db.ExecContext(ctx, `INSERT INTO match (player1_name, player2_name, match_length) VALUES ('a', 'b', 7)`)
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	mID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.AddMatchToTournament(tID, mID); err != nil {
		t.Fatalf("add match: %v", err)
	}
	if _, err := d.db.ExecContext(ctx,
		`UPDATE match SET direction_match_id = 'M1' WHERE id = ?`, mID); err != nil {
		t.Fatalf("fill slot: %v", err)
	}
	// A Slot carries at most one Match: a second Match on the same slot collides.
	res2, err := d.db.ExecContext(ctx, `INSERT INTO match (player1_name, player2_name, match_length) VALUES ('c', 'd', 7)`)
	if err != nil {
		t.Fatalf("create second match: %v", err)
	}
	m2, err := res2.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.AddMatchToTournament(tID, m2); err != nil {
		t.Fatalf("add second match: %v", err)
	}
	if _, err := d.db.ExecContext(ctx,
		`UPDATE match SET direction_match_id = 'M1' WHERE id = ?`, m2); err == nil {
		t.Error("two Matches on the same Slot must collide")
	}
	// But several Matches with no slot coexist: the partial index leaves the
	// empty string free, and that is what every ordinary Match carries.
	if _, err := d.db.ExecContext(ctx,
		`UPDATE match SET direction_match_id = '' WHERE id = ?`, m2); err != nil {
		t.Errorf("a Match with no Slot must be allowed alongside another: %v", err)
	}

	if err := d.DeleteTournament(tID); err != nil {
		t.Fatalf("delete tournament: %v", err)
	}
	var n int
	if err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM direction_event WHERE tournament_id = ?`, tID).Scan(&n); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if n != 0 {
		t.Errorf("deleting the Tournament must take its Direction with it, %d event(s) left", n)
	}
	if err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match WHERE id = ?`, mID).Scan(&n); err != nil {
		t.Fatalf("count matches: %v", err)
	}
	if n != 1 {
		t.Error("deleting the Tournament unlinks its Matches, it does not delete them")
	}
}

// TestRencontreSchema_2_25_0 covers the Rencontre and the doubles members
// (ADR-0056): deleting a Rencontre detaches its Tournaments and never deletes
// one; deleting a Tournament takes its pair members with it.
func TestRencontreSchema_2_25_0(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	res, err := d.db.ExecContext(ctx,
		`INSERT INTO rencontre (name, starts_on, ends_on, tables, output_dir)
		 VALUES ('Festival', '2026-10-03', '2026-10-04', 14, '')`)
	if err != nil {
		t.Fatalf("insert rencontre: %v", err)
	}
	rID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	tID, err := d.CreateTournament("Principal", "2026-10-03", "Lyon")
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	if _, err := d.db.ExecContext(ctx, `UPDATE tournament SET rencontre_id = ? WHERE id = ?`, rID, tID); err != nil {
		t.Fatalf("attach: %v", err)
	}
	for seat, name := range []string{"Ana", "Bea"} {
		if _, err := d.db.ExecContext(ctx,
			`INSERT INTO direction_pair_member (tournament_id, player_id, seat, name, club, rating)
			 VALUES (?, 'ana-bea', ?, ?, 'BC', 5.5)`, tID, seat, name); err != nil {
			t.Fatalf("insert member %d: %v", seat, err)
		}
	}
	if _, err := d.db.ExecContext(ctx,
		`INSERT INTO direction_pair_member (tournament_id, player_id, seat, name) VALUES (?, 'ana-bea', 0, 'Ana')`, tID); err == nil {
		t.Error("a pair has one person per seat")
	}

	if _, err := d.db.ExecContext(ctx, `DELETE FROM rencontre WHERE id = ?`, rID); err != nil {
		t.Fatalf("delete rencontre: %v", err)
	}
	var attached sql.NullInt64
	if err := d.db.QueryRowContext(ctx, `SELECT rencontre_id FROM tournament WHERE id = ?`, tID).Scan(&attached); err != nil {
		t.Fatalf("deleting a Rencontre must not delete its Tournament: %v", err)
	}
	if attached.Valid {
		t.Errorf("deleting a Rencontre detaches its Tournament, rencontre_id = %d", attached.Int64)
	}

	if err := d.DeleteTournament(tID); err != nil {
		t.Fatalf("delete tournament: %v", err)
	}
	var n int
	if err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM direction_pair_member WHERE tournament_id = ?`, tID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("deleting the Tournament must take its pair members, %d left", n)
	}
}

// TestMigrate_2_25_0_to_2_26_0_GameWinner runs the chain on a 2.25.0 library
// holding one match per source of storagetest.WinnerMigrationCases, and
// checks every game ends in the one encoding. PostgreSQL runs the same
// fixture through 028 (TestMigrate_028_GameWinner).
func TestMigrate_2_25_0_to_2_26_0_GameWinner(t *testing.T) {
	t.Parallel()
	dbPath, cases, ids := seedWinnerLibrary(t)
	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open 2.25.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration = %q, %v; want %q", v, err, DatabaseVersion)
	}
	checkWinners(t, d.db, cases, ids, true)
}

// TestMigrate_2_25_0_to_2_26_0_InterruptedThenRetried: the conversion is not
// idempotent, so it and the version stamp commit together. A failure while
// stamping must leave the games as they were and the version at 2.25.0, and
// the next open converts them once.
func TestMigrate_2_25_0_to_2_26_0_InterruptedThenRetried(t *testing.T) {
	t.Parallel()
	dbPath, cases, ids := seedWinnerLibrary(t)
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TRIGGER stamp_fails BEFORE UPDATE ON metadata
		WHEN NEW.key = 'database_version' AND NEW.value = '2.26.0'
		BEGIN SELECT RAISE(ABORT, 'interrupted'); END`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err == nil {
		t.Fatal("open succeeded although the version stamp failed")
	}
	d.Close()

	raw, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	var version string
	if err := raw.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&version); err != nil || version != "2.25.0" {
		t.Fatalf("version after the failed open = %q, %v; want 2.25.0", version, err)
	}
	checkWinners(t, raw, cases, ids, false)
	if _, err := raw.Exec(`DROP TRIGGER stamp_fails`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	d = NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open after the interruption: %v", err)
	}
	closeOnCleanup(t, d)
	checkWinners(t, d.db, cases, ids, true)
}

// seedWinnerLibrary writes storagetest.WinnerMigrationCases into a library
// stamped 2.25.0 and returns its path and every game's id.
func seedWinnerLibrary(t *testing.T) (string, []storagetest.WinnerMigrationMatch, [][]int64) {
	t.Helper()
	dbPath := filepath.Join(tempDir(t), "winner.db")
	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	cases := storagetest.WinnerMigrationCases()
	gameIDs := make([][]int64, len(cases))
	for i, c := range cases {
		var batchID any
		if c.BatchFormat != "" {
			res, err := d.db.Exec(`INSERT INTO import_batch (source, format) VALUES (?, ?)`, c.FilePath, c.BatchFormat)
			if err != nil {
				t.Fatalf("%s: insert batch: %v", c.Name, err)
			}
			batchID, _ = res.LastInsertId()
		}
		res, err := d.db.Exec(`INSERT INTO match (player1_name, player2_name, match_length, file_path, import_batch_id)
			VALUES ('A', 'B', ?, ?, ?)`, c.Length, c.FilePath, batchID)
		if err != nil {
			t.Fatalf("%s: insert match: %v", c.Name, err)
		}
		matchID, _ := res.LastInsertId()
		for n, g := range c.Games {
			res, err := d.db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won)
				VALUES (?, ?, ?, ?, ?, ?)`, matchID, n+1, g.S1, g.S2, g.Winner, g.PointsWon)
			if err != nil {
				t.Fatalf("%s: insert game: %v", c.Name, err)
			}
			id, _ := res.LastInsertId()
			gameIDs[i] = append(gameIDs[i], id)
		}
	}
	if _, err := d.db.Exec(`UPDATE metadata SET value = '2.25.0' WHERE key = 'database_version'`); err != nil {
		t.Fatalf("stamp 2.25.0: %v", err)
	}
	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return dbPath, cases, gameIDs
}

// checkWinners compares every game's stored winner with the fixture's: the
// normalized one when converted, the one it was seeded with otherwise.
func checkWinners(t *testing.T, db *sql.DB, cases []storagetest.WinnerMigrationMatch, ids [][]int64, converted bool) {
	t.Helper()
	for i, c := range cases {
		for n, g := range c.Games {
			var got sql.NullInt32
			if err := db.QueryRow(`SELECT winner FROM game WHERE id = ?`, ids[i][n]).Scan(&got); err != nil {
				t.Fatalf("%s: read game %d: %v", c.Name, n+1, err)
			}
			switch {
			case converted && got.Int32 != g.Want:
				t.Errorf("%s, game %d: winner %d, want %d", c.Name, n+1, got.Int32, g.Want)
			case !converted && g.Winner != nil && got.Int32 != *g.Winner:
				t.Errorf("%s, game %d: winner %d moved before the migration committed (seeded %d)", c.Name, n+1, got.Int32, *g.Winner)
			}
		}
	}
}

// TestMigrate_2_26_0_to_2_27_0_TranscriptionRevision opens a 2.26.0 library
// holding a draft written before the column existed: the draft keeps its
// typing and reads revision 1, the revision a fresh insert starts at, and
// the next write advances it.
func TestMigrate_2_26_0_to_2_27_0_TranscriptionRevision(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(tempDir(t), "test_v2260.db")
	createOldDatabase(t, dbPath, "2.26.0")
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE IF NOT EXISTS transcription (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		format_version TEXT NOT NULL,
		match_id INTEGER,
		label TEXT DEFAULT '',
		document TEXT NOT NULL)`); err != nil {
		t.Fatalf("create 2.26.0 transcription: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO transcription (format_version, label, document) VALUES ('3', 'A vs B', '{}')`); err != nil {
		t.Fatalf("insert draft: %v", err)
	}
	_ = raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.26.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration = %q, %v; want %q", v, err, DatabaseVersion)
	}
	if !columnExists(t, d.db, "transcription", "revision") {
		t.Fatal("transcription.revision should exist after migration")
	}
	ctx := context.Background()
	row, err := d.store.Transcriptions().Get(ctx, "", 1)
	if err != nil {
		t.Fatalf("read the draft back: %v", err)
	}
	if row.Revision != 1 || row.Document != "{}" {
		t.Fatalf("migrated draft = revision %d, document %q; want 1, {}", row.Revision, row.Document)
	}
	if _, err := d.store.Transcriptions().Save(ctx, "", row); err != nil || row.Revision != 2 {
		t.Fatalf("a write must advance the revision: got %d, %v", row.Revision, err)
	}
}

// TestMigrate_2_27_0_to_2_28_0_TableSettings opens a 2.27.0 library holding a
// Rencontre with a member: the library gains table_setting and
// tournament.rencontre_rooms, the member keeps its membership with no room
// restriction, and the Rencontre takes table properties (ADR-0058).
func TestMigrate_2_27_0_to_2_28_0_TableSettings(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(tempDir(t), "test_v2270.db")
	createOldDatabase(t, dbPath, "2.27.0")
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS rencontre (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			starts_on TEXT DEFAULT '',
			ends_on TEXT DEFAULT '',
			tables INTEGER NOT NULL DEFAULT 0,
			output_dir TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`DROP TABLE IF EXISTS tournament`,
		`CREATE TABLE tournament (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			date TEXT,
			location TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			comment TEXT DEFAULT '',
			rencontre_id INTEGER REFERENCES rencontre(id) ON DELETE SET NULL)`,
		`INSERT INTO rencontre (name, tables) VALUES ('Open', 32)`,
		`INSERT INTO tournament (name, rencontre_id) VALUES ('Principal', 1)`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("seed 2.27.0: %v", err)
		}
	}
	_ = raw.Close()

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.27.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration = %q, %v; want %q", v, err, DatabaseVersion)
	}
	if !tableExists(d.db, "table_setting") || !columnExists(t, d.db, "tournament", "rencontre_rooms") {
		t.Fatal("table_setting and tournament.rencontre_rooms should exist after migration")
	}
	ctx := context.Background()
	rs := d.store.Rencontres()
	r, err := rs.Get(ctx, "", 1)
	if err != nil {
		t.Fatalf("read the Rencontre back: %v", err)
	}
	if len(r.TournamentIDs) != 1 || len(r.EventRooms) != 0 || len(r.TableSettings) != 0 {
		t.Fatalf("migrated Rencontre = %+v; want one member, no rooms, no settings", r)
	}
	if err := rs.SetTableSettings(ctx, "", 1, []domain.TableSetting{{Number: 21, Name: "Stream", Room: "B"}}); err != nil {
		t.Fatalf("SetTableSettings on a migrated library: %v", err)
	}
	if err := rs.SetEventRooms(ctx, "", 1, []string{"B"}); err != nil {
		t.Fatalf("SetEventRooms on a migrated library: %v", err)
	}
	if r, _ = rs.Get(ctx, "", 1); len(r.TableSettings) != 1 || r.TableSettings[0].Room != "B" || len(r.EventRooms[1]) != 1 {
		t.Fatalf("after writing = %+v", r)
	}
}

// TestMigrate_2_28_0_to_2_29_0_Lessons opens a 2.28.0 library: it gains
// lesson and lesson_step, and a Lesson can be written and read back with a
// Step that shows one of the library's Collections (ADR-0066).
func TestMigrate_2_28_0_to_2_29_0_Lessons(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(tempDir(t), "test_v2280.db")
	createOldDatabase(t, dbPath, "2.28.0")

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.28.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration = %q, %v; want %q", v, err, DatabaseVersion)
	}
	if !tableExists(d.db, "lesson") || !tableExists(d.db, "lesson_step") {
		t.Fatal("lesson and lesson_step should exist after migration")
	}
	ctx := context.Background()
	collID, err := d.store.Collections().Create(ctx, "", "Primes", "")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	ls := d.store.Lessons()
	id, err := ls.Create(ctx, "", "Jouer contre une prime", "")
	if err != nil {
		t.Fatalf("create lesson on a migrated library: %v", err)
	}
	if _, err := ls.AddStep(ctx, "", id, domain.LessonStep{Title: "Voir", Text: "Regardez.", CollectionID: collID}); err != nil {
		t.Fatalf("add step on a migrated library: %v", err)
	}
	l, err := ls.Get(ctx, "", id)
	if err != nil || len(l.Steps) != 1 || l.Steps[0].CollectionID != collID {
		t.Fatalf("lesson read back = %+v, %v", l, err)
	}
}

// TestMigrate_2_29_0_to_2_30_0_LargeLibraryWave rolls a real library back to
// its 2.29.0 shape — the derived columns gone, the pruned indexes back — and
// checks that the open crossing 2.30.0 drops the indexes, creates the new
// tables, and derives match_date and the analysis provenance exactly as the
// write path would.
func TestMigrate_2_29_0_to_2_30_0_LargeLibraryWave(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(tempDir(t), "test_v2290.db")
	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := d.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	// What the write path stored: the reference the backfill must reproduce.
	wantEngine := map[int64]string{}
	wantDepth := map[int64]int64{}
	func() {
		rows, err := d.db.Query(`SELECT id, analysis_engine, analysis_depth FROM analysis`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var id, depth int64
			var eng string
			if err := rows.Scan(&id, &eng, &depth); err != nil {
				t.Fatal(err)
			}
			wantEngine[id], wantDepth[id] = eng, depth
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	}()
	if len(wantEngine) == 0 {
		t.Fatal("the fixture stored no analysis")
	}
	var withDate int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position WHERE match_date IS NOT NULL`).Scan(&withDate); err != nil || withDate == 0 {
		t.Fatalf("positions dated by the import = %d, %v; want some", withDate, err)
	}
	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_analysis_engine`, `DROP INDEX IF EXISTS idx_analysis_depth`, `DROP INDEX IF EXISTS idx_analysis_provenance_pending`,
		`DROP INDEX idx_analysis_creation_date`, `DROP INDEX idx_position_match_date`,
		`ALTER TABLE analysis DROP COLUMN analysis_engine`,
		`ALTER TABLE analysis DROP COLUMN analysis_depth`,
		`ALTER TABLE analysis DROP COLUMN creation_date`,
		`ALTER TABLE position DROP COLUMN match_date`,
		`DROP TABLE import_batch_file`, `DROP TABLE player_alias`, `DROP TABLE event_alias`,
		`DROP TABLE match_stats`,
		`DROP INDEX idx_match_dice_hash`, `DROP INDEX idx_training_item_position`,
		`ALTER TABLE match DROP COLUMN dice_hash`,
		`ALTER TABLE match DROP COLUMN player1_elo`, `ALTER TABLE match DROP COLUMN player2_elo`,
		`ALTER TABLE match DROP COLUMN player1_experience`, `ALTER TABLE match DROP COLUMN player2_experience`,
		`ALTER TABLE match DROP COLUMN transcriber`, `ALTER TABLE match DROP COLUMN has_jacoby`,
		`ALTER TABLE match DROP COLUMN has_beaver`, `ALTER TABLE match DROP COLUMN engine_version`,
		`ALTER TABLE training_item DROP COLUMN position_id`,
		`ALTER TABLE training_item DROP COLUMN answer`, `ALTER TABLE training_item DROP COLUMN error_mp`,
		`ALTER TABLE comment DROP COLUMN author`,
		`CREATE INDEX idx_position_decision_dice ON position(decision_type, dice_1, dice_2)`,
		`CREATE INDEX idx_analysis_win2 ON analysis(player2_win_rate)`,
		`CREATE INDEX idx_position_game_phase ON position(game_phase)`,
		`DROP INDEX idx_position_phase_off`,
		// Statistics of the 2.29.0 library: present, so ensureSearchStats
		// would keep them, and silent on the indexes the step creates.
		`ANALYZE`,
		`UPDATE metadata SET value = '2.29.0' WHERE key = 'database_version'`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	d = NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.29.0 database: %v", err)
	}
	closeOnCleanup(t, d)
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration = %q, %v; want %q", v, err, DatabaseVersion)
	}
	for _, name := range []string{"idx_position_decision_dice", "idx_analysis_win2", "idx_position_game_phase"} {
		var n int
		_ = d.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&n)
		if n != 0 {
			t.Errorf("index %s survived the migration", name)
		}
	}
	for _, table := range []string{"import_batch_file", "player_alias", "event_alias", "match_stats"} {
		if !tableExists(d.db, table) {
			t.Errorf("table %s missing after migration", table)
		}
	}
	for table, cols := range map[string][]string{
		"match": {"dice_hash", "player1_elo", "player2_elo", "player1_experience", "player2_experience",
			"transcriber", "has_jacoby", "has_beaver", "engine_version"},
		"training_item": {"position_id", "answer", "error_mp"},
		"comment":       {"author"},
	} {
		for _, c := range cols {
			if !columnExists(t, d.db, table, c) {
				t.Errorf("column %s.%s missing after migration", table, c)
			}
		}
	}
	for _, name := range []string{"idx_match_dice_hash", "idx_training_item_position", "idx_match_stats_pr", "idx_position_phase_off"} {
		var n int
		_ = d.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&n)
		if n != 1 {
			t.Errorf("index %s missing after migration", name)
		}
	}
	var pending int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM metadata WHERE key = ?`, matchDateBackfillKey).Scan(&pending)
	if pending != 0 {
		t.Error("the match_date backfill key outlived the pass")
	}
	var matchDateStat string
	if err := d.db.QueryRow(`SELECT stat FROM sqlite_stat1 WHERE idx = 'idx_position_match_date'`).Scan(&matchDateStat); err != nil {
		t.Errorf("planner statistics not refreshed by the migration: idx_position_match_date has no sqlite_stat1 row (%v)", err)
	}
	var redated int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position WHERE match_date IS NOT NULL`).Scan(&redated); err != nil || redated != withDate {
		t.Errorf("positions dated after migration = %d, %v; want %d", redated, err, withDate)
	}
	var wrongDate int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position WHERE match_date IS NOT
		(SELECT MIN(` + sqlite.UnixFromMatchDateSQL("m.match_date") + `) FROM move mv JOIN game g ON g.id = mv.game_id JOIN match m ON m.id = g.match_id
		  WHERE mv.position_id = position.id)`).Scan(&wrongDate); err != nil || wrongDate != 0 {
		t.Errorf("positions whose match_date is not their earliest match's = %d, %v", wrongDate, err)
	}
	rows, err := d.db.Query(`SELECT id, analysis_engine, analysis_depth FROM analysis`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var eng sql.NullString
		var depth sql.NullInt64
		if err := rows.Scan(&id, &eng, &depth); err != nil {
			t.Fatal(err)
		}
		if !eng.Valid || eng.String != wantEngine[id] || depth.Int64 != wantDepth[id] {
			t.Errorf("analysis %d: provenance after backfill = (%v, %v), want (%q, %d)", id, eng, depth, wantEngine[id], wantDepth[id])
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	// A deleted position leaves the quiz answer, without its position.
	if _, err := d.db.Exec(`INSERT INTO training_session (exercise) VALUES ('decision')`); err != nil {
		t.Fatal(err)
	}
	var posID int64
	if err := d.db.QueryRow(`SELECT id FROM position ORDER BY id LIMIT 1`).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.Exec(`INSERT INTO training_item (session_id, number_type, position_id, answer, error_mp)
		VALUES ((SELECT MAX(id) FROM training_session), 'decision', ?, '13/7 8/7', 120)`, posID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.Exec(`DELETE FROM position WHERE id = ?`, posID); err != nil {
		t.Fatal(err)
	}
	var kept sql.NullInt64
	if err := d.db.QueryRow(`SELECT position_id FROM training_item WHERE answer = '13/7 8/7'`).Scan(&kept); err != nil || kept.Valid {
		t.Errorf("training_item after its position was deleted: position_id = %v, %v; want the row with NULL", kept, err)
	}
}

// A library an earlier 2.30.0 build migrated has match_stats without the
// error split and the Snowie parts. The version does not move, so the open
// itself must add the columns and recompute the rows, and do nothing more
// on the next open.
func TestOpen_2_30_0_RepairsMatchStatsShape(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(tempDir(t), "test_v2300_shape.db")
	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := d.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	readRows := func(d *Database) map[[2]int64][3]int64 {
		t.Helper()
		got := map[[2]int64][3]int64{}
		rows, err := d.db.Query(`SELECT match_id, seat, checker_error_mp, snowie_moves, checker_moves FROM match_stats`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var id, seat int64
			var chk, sm, cm sql.NullInt64
			if err := rows.Scan(&id, &seat, &chk, &sm, &cm); err != nil {
				t.Fatal(err)
			}
			if !chk.Valid || !sm.Valid || !cm.Valid {
				t.Fatalf("match %d seat %d: a late column is NULL after the open", id, seat)
			}
			got[[2]int64{id, seat}] = [3]int64{chk.Int64, sm.Int64, cm.Int64}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return got
	}
	want := readRows(d)
	if len(want) == 0 {
		t.Fatal("the import computed no match_stats row")
	}
	// The earlier shape: the late columns absent, the rows still there.
	for _, col := range []string{"checker_error_mp", "cube_error_mp", "errors", "snowie_error_mp", "snowie_moves", "checker_moves"} {
		if _, err := d.db.Exec(`ALTER TABLE match_stats DROP COLUMN ` + col); err != nil {
			t.Fatalf("drop %s: %v", col, err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	for open := 1; open <= 2; open++ {
		d = NewDatabase()
		if err := d.OpenDatabase(dbPath); err != nil {
			t.Fatalf("open %d: %v", open, err)
		}
		var version string
		if err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&version); err != nil || version != DatabaseVersion {
			t.Fatalf("open %d: version %q, %v; want %s", open, version, err, DatabaseVersion)
		}
		got := readRows(d)
		if len(got) != len(want) {
			t.Fatalf("open %d: %d rows, want %d", open, len(got), len(want))
		}
		for k, w := range want {
			if got[k] != w {
				t.Errorf("open %d: match %d seat %d = %v, want %v", open, k[0], k[1], got[k], w)
			}
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// TestMigrate_2_30_0_to_2_31_0 rolls an imported library back to its 2.30.0
// shape and opens it: the MET, progress and error columns and tables come
// back, the open writes no move error (the pass is not part of the
// migration), and the resumable ScoreMoves pass then stores, for every move,
// exactly the error the read path computes from the analysis.
func TestMigrate_2_30_0_to_2_31_0(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(tempDir(t), "test_v2300.db")
	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := d.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	// The values the 2.31.0 representation must read back: Unix seconds.
	wantMatchDate := unixColumn(t, d.db, "position", "match_date")
	wantCreation := unixColumn(t, d.db, "analysis", "creation_date")
	wantBoards := boardsByID(t, d.db)
	// 2.30.0 stored the board as the compact JSON array.
	for id, b := range wantBoards {
		if _, err := d.db.Exec(`UPDATE position SET state = ? WHERE id = ?`, engine.EncodeBoardCompact(b), id); err != nil {
			t.Fatal(err)
		}
	}
	if len(wantMatchDate) == 0 || len(wantCreation) == 0 {
		t.Fatalf("fixture dates: %d positions, %d analyses; want some of each", len(wantMatchDate), len(wantCreation))
	}
	for _, stmt := range []string{
		`ALTER TABLE move DROP COLUMN error_mp`,
		`ALTER TABLE analysis DROP COLUMN met_id`,
		`DROP TABLE lesson_progress`,
		`DROP TABLE match_equity_table`,
		// 2.30.0 stored both dates as text, and indexed engine and depth.
		`DROP INDEX idx_analysis_provenance_pending`,
		`UPDATE position SET match_date = (SELECT MIN(m.match_date) FROM move mv
		   JOIN game g ON g.id = mv.game_id JOIN match m ON m.id = g.match_id
		  WHERE mv.position_id = position.id)`,
		`UPDATE analysis SET creation_date = datetime(creation_date, 'unixepoch')`,
		`CREATE INDEX idx_analysis_engine ON analysis(analysis_engine)`,
		`CREATE INDEX idx_analysis_depth ON analysis(analysis_depth)`,
		`UPDATE metadata SET value = '2.30.0' WHERE key = 'database_version'`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	// 2.30.0 stored the action labels as text, with no action_label table;
	// a label outside the fixed list and a NULL must survive the conversion.
	for _, c := range actionColumns2_31 {
		for _, stmt := range []string{
			`ALTER TABLE ` + c.table + ` ADD COLUMN ` + c.column + `_text TEXT`,
			`UPDATE ` + c.table + ` SET ` + c.column + `_text = ` + sqlshared.ActionLabelSQL(c.column),
			`ALTER TABLE ` + c.table + ` DROP COLUMN ` + c.column,
			`ALTER TABLE ` + c.table + ` RENAME COLUMN ` + c.column + `_text TO ` + c.column,
		} {
			if _, err := d.db.Exec(stmt); err != nil {
				t.Fatalf("%s: %v", stmt, err)
			}
		}
	}
	for _, stmt := range []string{
		`DROP TABLE action_label`,
		`UPDATE move SET cube_action = 'Unknown(-1)' WHERE id = (SELECT MIN(id) FROM move WHERE move_type = 'cube')`,
		`UPDATE move SET cube_action = NULL WHERE id = (SELECT MAX(id) FROM move WHERE move_type = 'cube')`,
		`UPDATE analysis SET best_cube_action = 'Doppel, Annahme' WHERE id = (SELECT MIN(id) FROM analysis)`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	wantLabels := actionLabelsByRow(t, d.db, func(col string) string { return col })
	for _, edge := range []string{"=Unknown(-1)", "=Doppel, Annahme", "<NULL>", "=cube", "=checker"} {
		if !slices.Contains(slices.Collect(maps.Values(wantLabels)), edge) {
			t.Fatalf("the fixture lacks the label %s", edge)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	d = NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open v2.30.0 database: %v", err)
	}
	if got := actionLabelsByRow(t, d.db, sqlshared.ActionLabelSQL); !maps.Equal(got, wantLabels) {
		t.Errorf("action labels after migration differ from the text before:\n got %v\nwant %v", got, wantLabels)
	}
	for _, c := range actionColumns2_31 {
		var text int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM ` + c.table + ` WHERE typeof(` + c.column + `) NOT IN ('integer', 'null')`).Scan(&text); err != nil || text != 0 {
			t.Errorf("%s.%s: %d values not coded, %v", c.table, c.column, text, err)
		}
	}
	closeOnCleanup(t, d)
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration = %q, %v; want %q", v, err, DatabaseVersion)
	}
	if !columnExists(t, d.db, "move", "error_mp") || !columnExists(t, d.db, "analysis", "met_id") {
		t.Fatal("move.error_mp and analysis.met_id should exist after migration")
	}
	for _, c := range []struct {
		table, column string
		want          map[int64]int64
	}{{"position", "match_date", wantMatchDate}, {"analysis", "creation_date", wantCreation}} {
		var text int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM ` + c.table + ` WHERE typeof(` + c.column + `) = 'text'`).Scan(&text); err != nil || text != 0 {
			t.Errorf("%s.%s: %d text values left, %v", c.table, c.column, text, err)
		}
		if got := unixColumn(t, d.db, c.table, c.column); !maps.Equal(got, c.want) {
			t.Errorf("%s.%s after migration = %v, want %v", c.table, c.column, got, c.want)
		}
	}
	var textStates int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position WHERE typeof(state) <> 'blob'`).Scan(&textStates); err != nil || textStates != 0 {
		t.Errorf("positions whose state is not binary after migration = %d, %v", textStates, err)
	}
	if got := boardsByID(t, d.db); !maps.Equal(got, wantBoards) {
		t.Errorf("boards after migration differ from the boards before (%d vs %d positions)", len(got), len(wantBoards))
	}
	for name, want := range map[string]bool{
		"idx_analysis_engine": false, "idx_analysis_depth": false,
		"idx_analysis_provenance_pending": true, "idx_position_match_date": true, "idx_analysis_creation_date": true,
	} {
		var n int
		_ = d.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name).Scan(&n)
		if (n == 1) != want {
			t.Errorf("index %s present = %v after migration, want %v", name, n == 1, want)
		}
	}
	if !tableExists(d.db, "lesson_progress") || !tableExists(d.db, "match_equity_table") {
		t.Fatal("lesson_progress and match_equity_table should exist after migration")
	}
	var scored int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM move WHERE error_mp IS NOT NULL`).Scan(&scored); err != nil || scored != 0 {
		t.Fatalf("moves scored by the open = %d, %v; want 0", scored, err)
	}

	ctx := context.Background()
	ms := d.store.Matches()
	var next int64
	total := 0
	for {
		n, k, err := ms.ScoreMoves(ctx, "", next, 7)
		if err != nil {
			t.Fatalf("ScoreMoves: %v", err)
		}
		if n == 0 {
			break
		}
		next, total = n, total+k
	}
	if total == 0 {
		t.Fatal("the pass scored no move of an analysed match")
	}
	var matchID int64
	if err := d.db.QueryRow(`SELECT id FROM match LIMIT 1`).Scan(&matchID); err != nil {
		t.Fatal(err)
	}
	stored := map[int64]sql.NullInt64{}
	func() {
		rows, err := d.db.Query(`SELECT id, error_mp FROM move`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var e sql.NullInt64
			if err := rows.Scan(&id, &e); err != nil {
				t.Fatal(err)
			}
			stored[id] = e
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	}()
	checked := 0
	for mv, err := range ms.MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatal(err)
		}
		got := stored[mv.ID]
		switch {
		case mv.ErrorMP == nil && got.Valid:
			t.Errorf("move %d: stored %d, read path unscored", mv.ID, got.Int64)
		case mv.ErrorMP != nil && (!got.Valid || got.Int64 != int64(*mv.ErrorMP)):
			t.Errorf("move %d: stored %v, read path %d", mv.ID, got, *mv.ErrorMP)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("the match has no move")
	}
	if n, _, err := ms.ScoreMoves(ctx, "", 0, 1000); err != nil {
		t.Fatalf("second pass: %v", err)
	} else if n != 0 {
		// Only analysed plays the analysis cannot score are visited again.
		var unscorable int
		_ = d.db.QueryRow(`SELECT COUNT(*) FROM move mv JOIN analysis a ON a.position_id = mv.position_id WHERE mv.error_mp IS NULL`).Scan(&unscorable)
		if unscorable == 0 {
			t.Errorf("second pass visited move %d with nothing left to score", n)
		}
	}
}

// unixColumn reads the non-NULL integer values of table.column by row id.
func unixColumn(t *testing.T, db *sql.DB, table, column string) map[int64]int64 {
	t.Helper()
	rows, err := db.Query(`SELECT id, ` + column + ` FROM ` + table + ` WHERE typeof(` + column + `) = 'integer'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]int64{}
	for rows.Next() {
		var id, v int64
		if err := rows.Scan(&id, &v); err != nil {
			t.Fatal(err)
		}
		out[id] = v
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// boardsByID decodes every stored position.state by position id.
func boardsByID(t *testing.T, db *sql.DB) map[int64]domain.Board {
	t.Helper()
	rows, err := db.Query(`SELECT id, state FROM position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]domain.Board{}
	for rows.Next() {
		var id int64
		var state []byte
		if err := rows.Scan(&id, &state); err != nil {
			t.Fatal(err)
		}
		out[id] = engine.DecodeBoardCompact(string(state))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// actionColumns2_31 are the columns 2.31.0 stores as action codes.
var actionColumns2_31 = []struct{ table, column string }{
	{"analysis", "best_cube_action"}, {"move", "move_type"}, {"move", "cube_action"},
}

// actionLabelsByRow reads every action label, keyed "table.column:id", through
// read (the bare column on a text library, sqlshared.ActionLabelSQL on a coded
// one); a NULL reads as "<NULL>", a label as "=" and the label.
func actionLabelsByRow(t *testing.T, db *sql.DB, read func(col string) string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, c := range actionColumns2_31 {
		readColumn(t, db, `SELECT id, `+read(c.column)+` FROM `+c.table, func(rows *sql.Rows) {
			var id int64
			var v sql.NullString
			if err := rows.Scan(&id, &v); err != nil {
				t.Fatal(err)
			}
			label := "<NULL>"
			if v.Valid {
				label = "=" + v.String
			}
			out[fmt.Sprintf("%s.%s:%d", c.table, c.column, id)] = label
		})
	}
	if len(out) == 0 {
		t.Fatal("no action label to compare")
	}
	return out
}

// readColumn runs query and hands each row to scan.
func readColumn(t *testing.T, db *sql.DB, query string, scan func(*sql.Rows)) {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		scan(rows)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}
