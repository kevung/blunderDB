package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync" // Import sync package
	"sync/atomic"
	"time"

	"github.com/kevung/bgfparser"
	"github.com/kevung/gnubgparser"
	"github.com/kevung/xgparser/xgparser"
	_ "modernc.org/sqlite"
)

type Database struct {
	db              *sql.DB
	mu              sync.Mutex // Add a mutex to the Database struct
	importCancelled int32      // Flag to cancel ongoing import (atomic)
}

func NewDatabase() *Database {
	return &Database{}
}

func (d *Database) SetupDatabase(path string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	if d.db != nil {
		d.db.Close() // Close the currently opened database
	}

	// Open the database using string path
	var err error
	d.db, err = sql.Open("sqlite", path)
	if err != nil {
		fmt.Println("Error opening database:", err)
		return err
	}

	// Enable foreign key constraints
	_, err = d.db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		fmt.Println("Error enabling foreign keys:", err)
		return err
	}

	// Erase any content in the database
	_, err = d.db.Exec(`
		PRAGMA writable_schema = 1;
		DELETE FROM sqlite_master WHERE type IN ('table', 'index', 'trigger');
		PRAGMA writable_schema = 0;
		VACUUM;
		PRAGMA INTEGRITY_CHECK;
	`)
	if err != nil {
		fmt.Println("Error erasing database content:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS position (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            state TEXT
        )
    `)
	if err != nil {
		fmt.Println("Error creating position table:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS analysis (
            id INTEGER PRIMARY KEY,
            position_id INTEGER,
            data JSON,
            FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
        )
    `)
	if err != nil {
		fmt.Println("Error creating analysis table:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS comment (
            id INTEGER PRIMARY KEY,
            position_id INTEGER,
            text TEXT,
            FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
        )
    `)
	if err != nil {
		fmt.Println("Error creating comment table:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS metadata (
            key TEXT PRIMARY KEY,
            value TEXT
        )
    `)
	if err != nil {
		fmt.Println("Error creating metadata table:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS command_history (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            command TEXT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		fmt.Println("Error creating command_history table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS filter_library (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			command TEXT,
			edit_position TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating filter_library table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			position TEXT,
			timestamp INTEGER
		)
	`)
	if err != nil {
		fmt.Println("Error creating search_history table:", err)
		return err
	}

	// Create match-related tables for XG import (v1.4.0)
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS match (
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
		)
	`)
	if err != nil {
		fmt.Println("Error creating match table:", err)
		return err
	}

	// Create index on match_hash for fast duplicate detection
	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
	if err != nil {
		fmt.Println("Error creating match_hash index:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER,
			game_number INTEGER,
			initial_score_1 INTEGER,
			initial_score_2 INTEGER,
			winner INTEGER,
			points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		fmt.Println("Error creating game table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS move (
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
		)
	`)
	if err != nil {
		fmt.Println("Error creating move table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS move_analysis (
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
		)
	`)
	if err != nil {
		fmt.Println("Error creating move_analysis table:", err)
		return err
	}

	// Create collection-related tables for position collections (v1.5.0)
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS collection (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		fmt.Println("Error creating collection table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS collection_position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL,
			position_id INTEGER NOT NULL,
			sort_order INTEGER DEFAULT 0,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(collection_id, position_id)
		)
	`)
	if err != nil {
		fmt.Println("Error creating collection_position table:", err)
		return err
	}

	// Create index for faster collection lookups
	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_collection_position_collection ON collection_position(collection_id)`)
	if err != nil {
		fmt.Println("Error creating collection_position index:", err)
		return err
	}

	// Create tournament table for organizing matches (v1.6.0)
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS tournament (
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
		fmt.Println("Error creating tournament table:", err)
		return err
	}

	// Add tournament_id column to match table if it doesn't exist
	_, err = d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)
	// Ignore error if column already exists

	// Add last_visited_position column to match table if it doesn't exist (v1.7.0)
	_, err = d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)
	// Ignore error if column already exists

	// Add canonical_hash column to match table if it doesn't exist
	// canonical_hash is format-independent (same match imported from XG and SGF will have the same canonical_hash)
	_, err = d.db.Exec(`ALTER TABLE match ADD COLUMN canonical_hash TEXT`)
	// Ignore error if column already exists

	// Insert or update the database version
	_, err = d.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		fmt.Println("Error inserting database version:", err)
		return err
	}

	return nil
}

func (d *Database) OpenDatabase(path string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	if d.db != nil {
		d.db.Close() // Close the currently opened database
	}

	// Open the database using string path
	var err error
	d.db, err = sql.Open("sqlite", path)
	if err != nil {
		fmt.Println("Error opening database:", err)
		return err
	}

	// Enable foreign key constraints
	_, err = d.db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		fmt.Println("Error enabling foreign keys:", err)
		return err
	}

	// Check the database version
	var dbVersion string
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	// Auto-migrate from 1.0.0 to 1.1.0
	if dbVersion == "1.0.0" {
		var cmdHistExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='command_history'`).Scan(&cmdHistExists)
		if err == sql.ErrNoRows {
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS command_history (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					command TEXT,
					timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
				)
			`)
			if err != nil {
				fmt.Println("Error creating command_history table:", err)
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.1.0")
			if err != nil {
				fmt.Println("Error updating database version:", err)
				return err
			}
			dbVersion = "1.1.0"
			fmt.Println("Database automatically upgraded from 1.0.0 to 1.1.0")
		}
	}

	// Auto-migrate from 1.1.0 to 1.2.0
	if dbVersion == "1.1.0" {
		var filterLibExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='filter_library'`).Scan(&filterLibExists)
		if err == sql.ErrNoRows {
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS filter_library (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT,
					command TEXT,
					edit_position TEXT
				)
			`)
			if err != nil {
				fmt.Println("Error creating filter_library table:", err)
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.2.0")
			if err != nil {
				fmt.Println("Error updating database version:", err)
				return err
			}
			dbVersion = "1.2.0"
			fmt.Println("Database automatically upgraded from 1.1.0 to 1.2.0")
		}
	}

	// Auto-migrate from 1.2.0 to 1.3.0
	if dbVersion == "1.2.0" {
		var searchHistoryExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='search_history'`).Scan(&searchHistoryExists)
		if err == sql.ErrNoRows {
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS search_history (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					command TEXT,
					position TEXT,
					timestamp INTEGER
				)
			`)
			if err != nil {
				fmt.Println("Error creating search_history table:", err)
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.3.0")
			if err != nil {
				fmt.Println("Error updating database version:", err)
				return err
			}
			dbVersion = "1.3.0"
			fmt.Println("Database automatically upgraded from 1.2.0 to 1.3.0")
		}
	}

	// Auto-migrate from 1.3.0 to 1.4.0
	if dbVersion == "1.3.0" {
		var matchExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&matchExists)
		if err == sql.ErrNoRows {
			// Create match-related tables
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS match (
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
				)
			`)
			if err != nil {
				fmt.Println("Error creating match table:", err)
				return err
			}

			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS game (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					match_id INTEGER,
					game_number INTEGER,
					initial_score_1 INTEGER,
					initial_score_2 INTEGER,
					winner INTEGER,
					points_won INTEGER,
					move_count INTEGER DEFAULT 0,
					FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				fmt.Println("Error creating game table:", err)
				return err
			}

			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS move (
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
				)
			`)
			if err != nil {
				fmt.Println("Error creating move table:", err)
				return err
			}

			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS move_analysis (
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
				)
			`)
			if err != nil {
				fmt.Println("Error creating move_analysis table:", err)
				return err
			}

			// Create index on match_hash for fast duplicate detection
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
			if err != nil {
				fmt.Println("Error creating match_hash index:", err)
				return err
			}

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.4.0")
			if err != nil {
				fmt.Println("Error updating database version:", err)
				return err
			}
			dbVersion = "1.4.0"
			fmt.Println("Database automatically upgraded from 1.3.0 to 1.4.0")
		}
	}

	// Add match_hash column to existing v1.4.0 databases if it doesn't exist
	if dbVersion == "1.4.0" {
		// Check if match_hash column exists
		var colInfo string
		err = d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&colInfo)
		if err == nil && !strings.Contains(colInfo, "match_hash") {
			// Add match_hash column
			_, err = d.db.Exec(`ALTER TABLE match ADD COLUMN match_hash TEXT`)
			if err != nil {
				fmt.Println("Error adding match_hash column:", err)
				return err
			}

			// Create index on match_hash
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
			if err != nil {
				fmt.Println("Error creating match_hash index:", err)
				return err
			}

			// Populate match_hash for existing matches using stored data
			// This uses a fallback hash based on stored moves since we don't have the original file
			matchRows, err := d.db.Query(`SELECT id, player1_name, player2_name, match_length FROM match`)
			if err == nil {
				defer matchRows.Close()
				for matchRows.Next() {
					var matchID int64
					var p1Name, p2Name string
					var matchLength int32
					if err := matchRows.Scan(&matchID, &p1Name, &p2Name, &matchLength); err != nil {
						continue
					}
					hash := computeMatchHashFromStoredData(d.db, matchID, p1Name, p2Name, matchLength)
					_, _ = d.db.Exec(`UPDATE match SET match_hash = ? WHERE id = ?`, hash, matchID)
				}
				if err := matchRows.Err(); err != nil {
					return err
				}
			}

			fmt.Println("Added match_hash column and populated existing matches")
		}
	}

	// Auto-migrate from 1.4.0 to 1.5.0
	if dbVersion == "1.4.0" {
		var collectionExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='collection'`).Scan(&collectionExists)
		if err == sql.ErrNoRows {
			// Create collection-related tables
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS collection (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					description TEXT,
					sort_order INTEGER DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)
			`)
			if err != nil {
				fmt.Println("Error creating collection table:", err)
				return err
			}

			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS collection_position (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					collection_id INTEGER NOT NULL,
					position_id INTEGER NOT NULL,
					sort_order INTEGER DEFAULT 0,
					added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
					FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
					UNIQUE(collection_id, position_id)
				)
			`)
			if err != nil {
				fmt.Println("Error creating collection_position table:", err)
				return err
			}

			// Create index for faster collection lookups
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_collection_position_collection ON collection_position(collection_id)`)
			if err != nil {
				fmt.Println("Error creating collection_position index:", err)
				return err
			}

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.5.0")
			if err != nil {
				fmt.Println("Error updating database version:", err)
				return err
			}
			dbVersion = "1.5.0"
			fmt.Println("Database automatically upgraded from 1.4.0 to 1.5.0")
		}
	}

	// Auto-migrate from 1.5.0 to 1.6.0
	if dbVersion == "1.5.0" {
		var tournamentExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='tournament'`).Scan(&tournamentExists)
		if err == sql.ErrNoRows {
			// Create tournament table
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS tournament (
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
				fmt.Println("Error creating tournament table:", err)
				return err
			}

			// Add tournament_id column to match table
			_, err = d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)
			// Ignore error if column already exists

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.6.0")
			if err != nil {
				fmt.Println("Error updating database version:", err)
				return err
			}
			dbVersion = "1.6.0"
			fmt.Println("Database automatically upgraded from 1.5.0 to 1.6.0")
		}
	}

	// Auto-migrate from 1.6.0 to 1.7.0
	if dbVersion == "1.6.0" {
		// Add last_visited_position column to match table
		_, err = d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)
		// Ignore error if column already exists

		_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.7.0")
		if err != nil {
			fmt.Println("Error updating database version:", err)
			return err
		}
		dbVersion = "1.7.0"
		fmt.Println("Database automatically upgraded from 1.6.0 to 1.7.0")
	}

	// Ensure all required tables and columns exist.
	// This repairs databases that were migrated through versions that skipped
	// creating some tables (e.g. filter_library was missing from some migration paths).
	if err := d.ensureAllTablesExist(); err != nil {
		fmt.Println("Error ensuring all tables exist:", err)
		return err
	}

	// Build required tables list based on the FINAL dbVersion (after all migrations)
	requiredTables := []string{"position", "analysis", "comment", "metadata"}
	if dbVersion >= "1.1.0" {
		requiredTables = append(requiredTables, "command_history")
	}
	if dbVersion >= "1.2.0" {
		requiredTables = append(requiredTables, "filter_library")
	}
	if dbVersion >= "1.3.0" {
		requiredTables = append(requiredTables, "search_history")
	}
	if dbVersion >= "1.4.0" {
		requiredTables = append(requiredTables, "match", "game", "move", "move_analysis")
	}
	if dbVersion >= "1.5.0" {
		requiredTables = append(requiredTables, "collection", "collection_position")
	}
	if dbVersion >= "1.6.0" {
		requiredTables = append(requiredTables, "tournament")
	}

	for _, table := range requiredTables {
		var tableName string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&tableName)
		if err != nil {
			fmt.Printf("Error checking table %s: %v\n", table, err)
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
			fmt.Printf("Error checking metadata key %s: %v\n", key, err)
			return err
		}
		if value == "" {
			return fmt.Errorf("required metadata key %s does not exist", key)
		}
	}

	return nil
}

// ensureAllTablesExist creates any missing tables and columns that should exist
// at the current database version. This repairs databases that were migrated
// through code paths that skipped creating some schema elements.
func (d *Database) ensureAllTablesExist() error {
	// v1.1.0: command_history
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring command_history table: %w", err)
	}

	// v1.2.0: filter_library
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS filter_library (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			command TEXT,
			edit_position TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring filter_library table: %w", err)
	}

	// v1.3.0: search_history
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			position TEXT,
			timestamp INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring search_history table: %w", err)
	}

	// v1.4.0: match, game, move, move_analysis
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS match (
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
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring match table: %w", err)
	}

	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
	if err != nil {
		return fmt.Errorf("error ensuring match_hash index: %w", err)
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER,
			game_number INTEGER,
			initial_score_1 INTEGER,
			initial_score_2 INTEGER,
			winner INTEGER,
			points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring game table: %w", err)
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS move (
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
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring move table: %w", err)
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS move_analysis (
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
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring move_analysis table: %w", err)
	}

	// v1.5.0: collection, collection_position
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS collection (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring collection table: %w", err)
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS collection_position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL,
			position_id INTEGER NOT NULL,
			sort_order INTEGER DEFAULT 0,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(collection_id, position_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring collection_position table: %w", err)
	}

	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_collection_position_collection ON collection_position(collection_id)`)
	if err != nil {
		return fmt.Errorf("error ensuring collection_position index: %w", err)
	}

	// v1.6.0: tournament
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS tournament (
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
		return fmt.Errorf("error ensuring tournament table: %w", err)
	}

	// Ensure columns added in later versions exist on match table
	// tournament_id (v1.6.0)
	d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)
	// last_visited_position (v1.7.0)
	d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)
	// canonical_hash (v1.7.0)
	d.db.Exec(`ALTER TABLE match ADD COLUMN canonical_hash TEXT`)

	return nil
}

func (d *Database) CheckVersion(databaseVersion string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	dbMajorVersion := strings.Split(dbVersion, ".")[0]
	expectedMajorVersion := strings.Split(databaseVersion, ".")[0]

	if dbMajorVersion != expectedMajorVersion {
		return fmt.Errorf("database major version mismatch: expected %s.x.x, got %s.x.x", expectedMajorVersion, dbMajorVersion)
	}

	return nil
}

func (d *Database) CheckDatabaseVersion() (string, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return "", err
	}
	return dbVersion, nil
}

func (d *Database) PositionExists(position Position) (map[string]interface{}, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Create a copy of the position without the ID field inside the state
	positionCopy := position
	positionCopy.ID = 0

	positionJSON, err := json.Marshal(positionCopy)
	if err != nil {
		fmt.Println("Error marshalling position:", err)
		return nil, err
	}

	rows, err := d.db.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error querying positions:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stateJSON string
		var positionID int64
		if err = rows.Scan(&positionID, &stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			return nil, err
		}

		var existingPosition Position
		if err = json.Unmarshal([]byte(stateJSON), &existingPosition); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			return nil, err
		}

		// Compare the positions excluding the ID field inside the state
		existingPosition.ID = 0
		existingPositionJSON, err := json.Marshal(existingPosition)
		if err != nil {
			fmt.Println("Error marshalling existing position:", err)
			return nil, err
		}

		if string(positionJSON) == string(existingPositionJSON) {
			return map[string]interface{}{"id": positionID, "exists": true}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": 0, "exists": false}, nil
}

func (d *Database) SavePosition(position *Position) (int64, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Normalize position for storage - always store from player on roll's perspective (player_on_roll = 0)
	normalizedPosition := position.NormalizeForStorage()

	positionJSON, err := json.Marshal(normalizedPosition)
	if err != nil {
		fmt.Println("Error marshalling position:", err)
		return 0, err
	}

	result, err := d.db.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
	if err != nil {
		fmt.Println("Error inserting position:", err)
		return 0, err
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		fmt.Println("Error getting last insert ID:", err)
		return 0, err
	}

	normalizedPosition.ID = positionID // Update the position ID

	// Update the state with the new ID
	positionJSON, err = json.Marshal(normalizedPosition)
	if err != nil {
		fmt.Println("Error marshalling position with ID:", err)
		return 0, err
	}

	_, err = d.db.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSON), positionID)
	if err != nil {
		fmt.Println("Error updating position with ID:", err)
		return 0, err
	}

	// Update the original position with the saved ID and normalized state
	*position = normalizedPosition

	return positionID, nil
}

func (d *Database) UpdatePosition(position Position) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	positionJSON, err := json.Marshal(position)
	if err != nil {
		fmt.Println("Error marshalling position:", err)
		return err
	}

	_, err = d.db.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSON), position.ID)
	if err != nil {
		fmt.Println("Error updating position:", err)
		return err
	}

	return nil
}

func (d *Database) SaveAnalysis(positionID int64, analysis PositionAnalysis) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Ensure the positionID is set in the analysis
	analysis.PositionID = int(positionID)

	// Update last modified date
	analysis.LastModifiedDate = time.Now()

	// Check if an analysis already exists for the given position ID
	var existingID int64
	var existingAnalysisJSON string
	err := d.db.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisJSON)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error querying analysis:", err)
		return err
	}

	if existingID > 0 {
		// Parse existing analysis
		var existingAnalysis PositionAnalysis
		err = json.Unmarshal([]byte(existingAnalysisJSON), &existingAnalysis)
		if err != nil {
			fmt.Println("Error unmarshalling existing analysis:", err)
			return err
		}

		// Preserve the existing creation date
		analysis.CreationDate = existingAnalysis.CreationDate

		// Merge checker analysis if both exist
		if existingAnalysis.CheckerAnalysis != nil && analysis.CheckerAnalysis != nil {
			analysis.CheckerAnalysis.Moves = mergeCheckerMoves(
				existingAnalysis.CheckerAnalysis.Moves,
				analysis.CheckerAnalysis.Moves,
			)
		} else if existingAnalysis.CheckerAnalysis != nil && analysis.CheckerAnalysis == nil {
			// Keep existing checker analysis if new one is nil
			analysis.CheckerAnalysis = existingAnalysis.CheckerAnalysis
		}

		// Merge doubling cube analysis - keep existing if new is nil
		if existingAnalysis.DoublingCubeAnalysis != nil && analysis.DoublingCubeAnalysis == nil {
			analysis.DoublingCubeAnalysis = existingAnalysis.DoublingCubeAnalysis
		}

		// Merge played moves (support both old single field and new array field)
		existingPlayedMoves := existingAnalysis.PlayedMoves
		if existingAnalysis.PlayedMove != "" && len(existingPlayedMoves) == 0 {
			existingPlayedMoves = []string{existingAnalysis.PlayedMove}
		}
		incomingPlayedMoves := analysis.PlayedMoves
		if analysis.PlayedMove != "" && len(incomingPlayedMoves) == 0 {
			incomingPlayedMoves = []string{analysis.PlayedMove}
		}
		analysis.PlayedMoves = mergePlayedMoves(existingPlayedMoves, incomingPlayedMoves)

		// Merge played cube actions
		existingCubeActions := existingAnalysis.PlayedCubeActions
		if existingAnalysis.PlayedCubeAction != "" && len(existingCubeActions) == 0 {
			existingCubeActions = []string{existingAnalysis.PlayedCubeAction}
		}
		incomingCubeActions := analysis.PlayedCubeActions
		if analysis.PlayedCubeAction != "" && len(incomingCubeActions) == 0 {
			incomingCubeActions = []string{analysis.PlayedCubeAction}
		}
		analysis.PlayedCubeActions = mergePlayedMoves(existingCubeActions, incomingCubeActions)

		// Clear deprecated single fields after merging
		analysis.PlayedMove = ""
		analysis.PlayedCubeAction = ""

		// Sort checker moves by equity after merging
		if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
			sort.Slice(analysis.CheckerAnalysis.Moves, func(i, j int) bool {
				return analysis.CheckerAnalysis.Moves[i].Equity > analysis.CheckerAnalysis.Moves[j].Equity
			})
			// Recalculate indices and equity errors
			bestEquity := analysis.CheckerAnalysis.Moves[0].Equity
			for i := range analysis.CheckerAnalysis.Moves {
				analysis.CheckerAnalysis.Moves[i].Index = i
				if i == 0 {
					analysis.CheckerAnalysis.Moves[i].EquityError = nil
				} else {
					diff := bestEquity - analysis.CheckerAnalysis.Moves[i].Equity
					analysis.CheckerAnalysis.Moves[i].EquityError = &diff
				}
			}
		}

		// Update the existing analysis
		analysisJSON, err := json.Marshal(analysis)
		if err != nil {
			fmt.Println("Error marshalling analysis:", err)
			return err
		}
		_, err = d.db.Exec(`UPDATE analysis SET data = ? WHERE id = ?`, string(analysisJSON), existingID)
		if err != nil {
			fmt.Println("Error updating analysis:", err)
			return err
		}
	} else {
		// Set creation date if not already set
		if analysis.CreationDate.IsZero() {
			analysis.CreationDate = time.Now()
		}

		// Convert single played move to array if needed
		if analysis.PlayedMove != "" && len(analysis.PlayedMoves) == 0 {
			analysis.PlayedMoves = []string{analysis.PlayedMove}
			analysis.PlayedMove = ""
		}
		if analysis.PlayedCubeAction != "" && len(analysis.PlayedCubeActions) == 0 {
			analysis.PlayedCubeActions = []string{analysis.PlayedCubeAction}
			analysis.PlayedCubeAction = ""
		}

		// Sort checker moves by equity
		if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
			sort.Slice(analysis.CheckerAnalysis.Moves, func(i, j int) bool {
				return analysis.CheckerAnalysis.Moves[i].Equity > analysis.CheckerAnalysis.Moves[j].Equity
			})
			// Recalculate indices and equity errors
			bestEquity := analysis.CheckerAnalysis.Moves[0].Equity
			for i := range analysis.CheckerAnalysis.Moves {
				analysis.CheckerAnalysis.Moves[i].Index = i
				if i == 0 {
					analysis.CheckerAnalysis.Moves[i].EquityError = nil
				} else {
					diff := bestEquity - analysis.CheckerAnalysis.Moves[i].Equity
					analysis.CheckerAnalysis.Moves[i].EquityError = &diff
				}
			}
		}

		// Insert a new analysis
		analysisJSON, err := json.Marshal(analysis)
		if err != nil {
			fmt.Println("Error marshalling analysis:", err)
			return err
		}
		_, err = d.db.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, string(analysisJSON))
		if err != nil {
			fmt.Println("Error inserting analysis:", err)
			return err
		}
	}

	return nil
}

func (d *Database) LoadPosition(id int) (*Position, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var stateJSON string

	err := d.db.QueryRow(`SELECT state from position WHERE id = ?`, id).Scan(&stateJSON)
	if err != nil {
		fmt.Println("Error loading position:", err)
		return nil, err
	}

	var state Position
	err = json.Unmarshal([]byte(stateJSON), &state)
	if err != nil {
		fmt.Println("Error unmarshalling position:", err)
		return nil, err
	}

	return &state, nil
}

func (d *Database) LoadAnalysis(positionID int64) (*PositionAnalysis, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var analysisJSON string
	err := d.db.QueryRow(`SELECT data from analysis WHERE position_id = ?`, positionID).Scan(&analysisJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		fmt.Println("Error loading analysis:", err)
		return nil, err
	}

	var analysis PositionAnalysis
	err = json.Unmarshal([]byte(analysisJSON), &analysis)
	if err != nil {
		fmt.Println("Error unmarshalling analysis:", err)
		return nil, err
	}

	// Load ALL played moves/cube actions from move table for this position
	// This supplements the PlayedMoves/PlayedCubeActions arrays stored in analysis
	rows, err := d.db.Query(`
		SELECT checker_move, cube_action 
		FROM move 
		WHERE position_id = ?
	`, positionID)

	if err == nil {
		defer rows.Close()

		// Collect all moves from the database
		dbCheckerMoves := make(map[string]bool)
		dbCubeActions := make(map[string]bool)

		for rows.Next() {
			var checkerMove sql.NullString
			var cubeAction sql.NullString
			if err := rows.Scan(&checkerMove, &cubeAction); err == nil {
				if checkerMove.Valid && checkerMove.String != "" {
					dbCheckerMoves[normalizeMove(checkerMove.String)] = true
				}
				if cubeAction.Valid && cubeAction.String != "" {
					dbCubeActions[cubeAction.String] = true
				}
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}

		// Merge with existing PlayedMoves array
		existingMoves := make(map[string]bool)
		for _, m := range analysis.PlayedMoves {
			existingMoves[normalizeMove(m)] = true
		}
		// Backward compatibility: include old single PlayedMove field
		if analysis.PlayedMove != "" {
			existingMoves[normalizeMove(analysis.PlayedMove)] = true
		}

		// Combine all moves
		for m := range dbCheckerMoves {
			existingMoves[m] = true
		}

		// Convert to slice
		analysis.PlayedMoves = make([]string, 0, len(existingMoves))
		for m := range existingMoves {
			analysis.PlayedMoves = append(analysis.PlayedMoves, m)
		}
		sort.Strings(analysis.PlayedMoves)

		// Do the same for cube actions
		existingCubeActions := make(map[string]bool)
		for _, a := range analysis.PlayedCubeActions {
			existingCubeActions[a] = true
		}
		// Backward compatibility: include old single PlayedCubeAction field
		if analysis.PlayedCubeAction != "" {
			existingCubeActions[analysis.PlayedCubeAction] = true
		}

		for a := range dbCubeActions {
			existingCubeActions[a] = true
		}

		analysis.PlayedCubeActions = make([]string, 0, len(existingCubeActions))
		for a := range existingCubeActions {
			analysis.PlayedCubeActions = append(analysis.PlayedCubeActions, a)
		}
		sort.Strings(analysis.PlayedCubeActions)

		// For backward compatibility, also set the old single fields if there's exactly one
		if len(analysis.PlayedMoves) == 1 {
			analysis.PlayedMove = analysis.PlayedMoves[0]
		} else if len(analysis.PlayedMoves) > 0 {
			// Set to first one for backward compatibility with old frontend
			analysis.PlayedMove = analysis.PlayedMoves[0]
		}
		if len(analysis.PlayedCubeActions) == 1 {
			analysis.PlayedCubeAction = analysis.PlayedCubeActions[0]
		} else if len(analysis.PlayedCubeActions) > 0 {
			analysis.PlayedCubeAction = analysis.PlayedCubeActions[0]
		}
	}

	// Sort AllCubeAnalyses so XG comes first (for existing data in DB)
	if len(analysis.AllCubeAnalyses) > 0 {
		sortCubeAnalysesByEngine(analysis.AllCubeAnalyses)
	}

	return &analysis, nil
}

func (d *Database) LoadAllPositions() ([]Position, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	rows, err := d.db.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error loading positions:", err)
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			return nil, err
		}

		var position Position
		if err = json.Unmarshal([]byte(stateJSON), &position); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			return nil, err
		}
		position.ID = id // Ensure the ID is set
		positions = append(positions, position)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(positions) == 0 {
		fmt.Println("No positions found, returning empty array.")
	}

	fmt.Println("Loaded positions:", positions)
	return positions, nil
}

func (d *Database) DeletePosition(positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Delete the position — ON DELETE CASCADE handles analysis, comment, and collection_position
	_, err := d.db.Exec(`DELETE FROM position WHERE id = ?`, positionID)
	if err != nil {
		fmt.Println("Error deleting position:", err)
		return err
	}

	return nil
}

func (d *Database) DeleteAnalysis(positionID int64) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	_, err := d.db.Exec(`DELETE FROM analysis WHERE position_id = ?`, positionID)
	if err != nil {
		fmt.Println("Error deleting analysis:", err)
		return err
	}
	return nil
}

func (d *Database) DeleteComment(positionID int64) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	_, err := d.db.Exec(`DELETE FROM comment WHERE position_id = ?`, positionID)
	if err != nil {
		fmt.Println("Error deleting comment:", err)
		return err
	}
	return nil
}

// SaveComment saves a comment for a given position ID
func (d *Database) SaveComment(positionID int64, text string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Check if a comment already exists for the given position ID
	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM comment WHERE position_id = ?`, positionID).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error querying comment:", err)
		return err
	}

	if existingID > 0 {
		// Update the existing comment
		_, err = d.db.Exec(`UPDATE comment SET text = ? WHERE id = ?`, text, existingID)
		if err != nil {
			fmt.Println("Error updating comment:", err)
			return err
		}
	} else {
		// Insert a new comment
		_, err = d.db.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, positionID, text)
		if err != nil {
			fmt.Println("Error inserting comment:", err)
			return err
		}
	}

	return nil
}

// LoadComment loads a comment for a given position ID
func (d *Database) LoadComment(positionID int64) (string, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var text string
	err := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, positionID).Scan(&text)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No comment found
		}
		fmt.Println("Error loading comment:", err)
		return "", err
	}
	return text, nil
}

func (d *Database) LoadPositionsByFilters(
	filter Position,
	includeCube bool,
	includeScore bool,
	pipCountFilter string,
	winRateFilter string,
	gammonRateFilter string,
	backgammonRateFilter string,
	player2WinRateFilter string,
	player2GammonRateFilter string,
	player2BackgammonRateFilter string,
	player1CheckerOffFilter string,
	player2CheckerOffFilter string,
	player1BackCheckerFilter string,
	player2BackCheckerFilter string,
	player1CheckerInZoneFilter string,
	player2CheckerInZoneFilter string,
	searchText string,
	player1AbsolutePipCountFilter string,
	equityFilter string,
	decisionTypeFilter bool,
	diceRollFilter bool,
	movePatternFilter string,
	dateFilter string,
	player1OutfieldBlotFilter string,
	player2OutfieldBlotFilter string,
	player1JanBlotFilter string,
	player2JanBlotFilter string,
	noContactFilter bool,
	mirrorFilter bool,
	moveErrorFilter string,
) ([]Position, error) {
	d.mu.Lock()
	rows, err := d.db.Query(`SELECT id, state FROM position`)
	d.mu.Unlock()
	if err != nil {
		fmt.Println("Error loading positions:", err)
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			return nil, err
		}

		var position Position
		if err = json.Unmarshal([]byte(stateJSON), &position); err != nil {
			return nil, err
		}
		position.ID = id // Ensure the ID is set

		// Function to check if a position matches all filters
		matchesFilters := func(pos Position) bool {
			return pos.MatchesCheckerPosition(filter) &&
				(!includeCube || pos.MatchesCubePosition(filter)) &&
				(!includeScore || pos.MatchesScorePosition(filter)) &&
				(!decisionTypeFilter || pos.MatchesDecisionType(filter)) &&
				(pipCountFilter == "" || pos.MatchesPipCountFilter(pipCountFilter)) &&
				(winRateFilter == "" || pos.MatchesWinRate(winRateFilter, d)) &&
				(gammonRateFilter == "" || pos.MatchesGammonRate(gammonRateFilter, d)) &&
				(backgammonRateFilter == "" || pos.MatchesBackgammonRate(backgammonRateFilter, d)) &&
				(player2WinRateFilter == "" || pos.MatchesPlayer2WinRate(player2WinRateFilter, d)) &&
				(player2GammonRateFilter == "" || pos.MatchesPlayer2GammonRate(player2GammonRateFilter, d)) &&
				(player2BackgammonRateFilter == "" || pos.MatchesPlayer2BackgammonRate(player2BackgammonRateFilter, d)) &&
				(player1CheckerOffFilter == "" || pos.MatchesPlayer1CheckerOff(player1CheckerOffFilter)) &&
				(player2CheckerOffFilter == "" || pos.MatchesPlayer2CheckerOff(player2CheckerOffFilter)) &&
				(player1BackCheckerFilter == "" || pos.MatchesPlayer1BackChecker(player1BackCheckerFilter)) &&
				(player2BackCheckerFilter == "" || pos.MatchesPlayer2BackChecker(player2BackCheckerFilter)) &&
				(player1CheckerInZoneFilter == "" || pos.MatchesPlayer1CheckerInZone(player1CheckerInZoneFilter)) &&
				(player2CheckerInZoneFilter == "" || pos.MatchesPlayer2CheckerInZone(player2CheckerInZoneFilter)) &&
				(searchText == "" || pos.MatchesSearchText(searchText, d)) &&
				(player1AbsolutePipCountFilter == "" || pos.MatchesPlayer1AbsolutePipCount(player1AbsolutePipCountFilter)) &&
				(equityFilter == "" || pos.MatchesEquityFilter(equityFilter, d)) &&
				(!diceRollFilter || pos.MatchesDiceRoll(filter)) &&
				(dateFilter == "" || pos.MatchesDateFilter(dateFilter, d)) &&
				(player1OutfieldBlotFilter == "" || pos.MatchesPlayer1OutfieldBlot(player1OutfieldBlotFilter)) &&
				(player2OutfieldBlotFilter == "" || pos.MatchesPlayer2OutfieldBlot(player2OutfieldBlotFilter)) &&
				(player1JanBlotFilter == "" || pos.MatchesPlayer1JanBlot(player1JanBlotFilter)) &&
				(player2JanBlotFilter == "" || pos.MatchesPlayer2JanBlot(player2JanBlotFilter)) &&
				(!noContactFilter || pos.MatchesNoContact()) &&
				(moveErrorFilter == "" || pos.MatchesMoveErrorFilter(moveErrorFilter, d))
		}

		// addPosition adds a matched position to results, mirroring take/pass positions
		// so player1 (the taker/passer) is shown at the bottom of the board.
		addPosition := func(pos Position) {
			if moveErrorFilter != "" && pos.DecisionType == CubeAction && pos.IsPlayer1TakePassCubeAction(d) {
				pos = pos.Mirror()
			}
			positions = append(positions, pos)
		}

		// Check the original position
		if matchesFilters(position) {
			if movePatternFilter != "" {
				if position.MatchesMovePattern(movePatternFilter, d) {
					addPosition(position)
				}
			} else {
				addPosition(position)
			}
		} else if mirrorFilter {
			mirroredPosition := position.Mirror()
			if matchesFilters(mirroredPosition) {
				if movePatternFilter != "" {
					if mirroredPosition.MatchesMovePattern(movePatternFilter, d) {
						addPosition(mirroredPosition)
					}
				} else {
					addPosition(mirroredPosition)
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return positions, nil
}

func (p *Position) MatchesDecisionType(filter Position) bool {
	return p.DecisionType == filter.DecisionType && p.PlayerOnRoll == filter.PlayerOnRoll
}

func (p *Position) MatchesSearchText(searchText string, d *Database) bool {
	comment, err := d.LoadComment(p.ID)
	if err != nil {
		fmt.Printf("Error loading comment for position ID: %d, error: %v\n", p.ID, err)
		return false
	}

	// Extract the keyword from the raw search text filter
	searchTextMatch := strings.Trim(searchText, ` t"'`)
	searchTextArray := strings.Split(strings.ToLower(searchTextMatch), ";")
	comment = strings.ToLower(comment)
	for _, text := range searchTextArray {
		if strings.Contains(comment, text) {
			return true
		}
	}
	return false
}

// Add MatchesPlayer1CheckerOff method to Position type
func (p *Position) MatchesPlayer1CheckerOff(filter string) bool {
	checkersOff := p.Board.Bearoff[0]

	if strings.HasPrefix(filter, "o>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersOff >= value
	} else if strings.HasPrefix(filter, "o<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersOff <= value
	} else if strings.HasPrefix(filter, "o") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'ox' means 'ox,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersOff >= minValue && checkersOff <= maxValue
	}
	return false
}

// Add MatchesPlayer2CheckerOff method to Position type
func (p *Position) MatchesPlayer2CheckerOff(filter string) bool {
	checkersOff := p.Board.Bearoff[1]

	if strings.HasPrefix(filter, "O>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersOff >= value
	} else if strings.HasPrefix(filter, "O<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersOff <= value
	} else if strings.HasPrefix(filter, "O") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Ox' means 'Ox,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersOff >= minValue && checkersOff <= maxValue
	}
	return false
}

// Add MatchesPlayer2BackgammonRate method to Position type
func (p *Position) MatchesPlayer2BackgammonRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var backgammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		backgammonRate = analysis.DoublingCubeAnalysis.OpponentBackgammonChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 2 Backgammon Rate: %f\n", p.ID, backgammonRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		backgammonRate = analysis.CheckerAnalysis.Moves[0].OpponentBackgammonChance
		fmt.Printf("Position ID: %d, Checker decision, Player 2 Backgammon Rate: %f\n", p.ID, backgammonRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no backgammon rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "B>") && !strings.HasPrefix(filter, "BO>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate >= value
	} else if strings.HasPrefix(filter, "B<") && !strings.HasPrefix(filter, "BO<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate <= value
	} else if strings.HasPrefix(filter, "B") && !strings.HasPrefix(filter, "BO") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backgammonRate >= minValue && backgammonRate <= maxValue
	}
	return false
}

// Add MatchesPlayer2GammonRate method to Position type
func (p *Position) MatchesPlayer2GammonRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var gammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		gammonRate = analysis.DoublingCubeAnalysis.OpponentGammonChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 2 Gammon Rate: %f\n", p.ID, gammonRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		gammonRate = analysis.CheckerAnalysis.Moves[0].OpponentGammonChance
		fmt.Printf("Position ID: %d, Checker decision, Player 2 Gammon Rate: %f\n", p.ID, gammonRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no gammon rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "G>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return gammonRate >= value
	} else if strings.HasPrefix(filter, "G<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return gammonRate <= value
	} else if strings.HasPrefix(filter, "G") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return gammonRate >= minValue && gammonRate <= maxValue
	}
	return false
}

// Add MatchesScorePosition method to Position type
func (p *Position) MatchesScorePosition(filter Position) bool {
	return p.Score[0] == filter.Score[0] && p.Score[1] == filter.Score[1]
}

// Add MatchesCubePosition method to Position type
func (p *Position) MatchesCubePosition(filter Position) bool {
	return p.Cube.Value == filter.Cube.Value && p.Cube.Owner == filter.Cube.Owner
}

// Add MatchesPipCountFilter method to Position type
func (p *Position) MatchesPipCountFilter(filter string) bool {
	pipCountDiff := p.PipCountDifference()
	player1PipCount, player2PipCount := p.ComputePipCounts()
	fmt.Printf("Checking pip count filter: %s, Player 1 Pip Count: %d, Player 2 Pip Count: %d, Pip count difference: %d\n", filter, player1PipCount, player2PipCount, pipCountDiff)
	if strings.HasPrefix(filter, "p>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return pipCountDiff >= value
	} else if strings.HasPrefix(filter, "p<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return pipCountDiff <= value
	} else if strings.HasPrefix(filter, "p") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return pipCountDiff >= minValue && pipCountDiff <= maxValue
	}
	return false
}

// Add MatchesWinRate method to Position type
func (p *Position) MatchesWinRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var winRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		winRate = analysis.DoublingCubeAnalysis.PlayerWinChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 1 Win Rate: %f\n", p.ID, winRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		winRate = analysis.CheckerAnalysis.Moves[0].PlayerWinChance
		fmt.Printf("Position ID: %d, Checker decision, Player 1 Win Rate: %f\n", p.ID, winRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no win rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "w>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return winRate >= value
	} else if strings.HasPrefix(filter, "w<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return winRate <= value
	} else if strings.HasPrefix(filter, "w") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return winRate >= minValue && winRate <= maxValue
	}
	return false
}

// Add MatchesPlayer2WinRate method to Position type
func (p *Position) MatchesPlayer2WinRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var winRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		winRate = analysis.DoublingCubeAnalysis.OpponentWinChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 2 Win Rate: %f\n", p.ID, winRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		winRate = analysis.CheckerAnalysis.Moves[0].OpponentWinChance
		fmt.Printf("Position ID: %d, Checker decision, Player 2 Win Rate: %f\n", p.ID, winRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no win rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "W>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return winRate >= value
	} else if strings.HasPrefix(filter, "W<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return winRate <= value
	} else if strings.HasPrefix(filter, "W") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return winRate >= minValue && winRate <= maxValue
	}
	return false
}

// Add MatchesGammonRate method to Position type
func (p *Position) MatchesGammonRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var gammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		gammonRate = analysis.DoublingCubeAnalysis.PlayerGammonChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 1 Gammon Rate: %f\n", p.ID, gammonRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		gammonRate = analysis.CheckerAnalysis.Moves[0].PlayerGammonChance
		fmt.Printf("Position ID: %d, Checker decision, Player 1 Gammon Rate: %f\n", p.ID, gammonRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no gammon rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "g>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return gammonRate >= value
	} else if strings.HasPrefix(filter, "g<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return gammonRate <= value
	} else if strings.HasPrefix(filter, "g") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return gammonRate >= minValue && gammonRate <= maxValue
	}
	return false
}

// Add MatchesBackgammonRate method to Position type
func (p *Position) MatchesBackgammonRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var backgammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		backgammonRate = analysis.DoublingCubeAnalysis.PlayerBackgammonChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 1 Backgammon Rate: %f\n", p.ID, backgammonRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		backgammonRate = analysis.CheckerAnalysis.Moves[0].PlayerBackgammonChance
		fmt.Printf("Position ID: %d, Checker decision, Player 1 Backgammon Rate: %f\n", p.ID, backgammonRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no backgammon rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "b>") && !strings.HasPrefix(filter, "bo>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate >= value
	} else if strings.HasPrefix(filter, "b<") && !strings.HasPrefix(filter, "bo<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate <= value
	} else if strings.HasPrefix(filter, "b") && !strings.HasPrefix(filter, "bo") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backgammonRate >= minValue && backgammonRate <= maxValue
	}
	return false
}

// Add PipCountDifference method to Position type
func (p *Position) PipCountDifference() int {
	player1PipCount, player2PipCount := p.ComputePipCounts()
	return player1PipCount - player2PipCount
}

// Add ComputePipCounts method to Position type
func (p *Position) ComputePipCounts() (int, int) {
	player1PipCount := 0
	player2PipCount := 0

	for i, point := range p.Board.Points {
		if point.Color == 0 {
			player1PipCount += point.Checkers * i
		} else if point.Color == 1 {
			player2PipCount += point.Checkers * (25 - i)
		}
	}

	return player1PipCount, player2PipCount
}

// Add MatchesPlayer1BackChecker method to Position type with logging
func (p *Position) MatchesPlayer1BackChecker(filter string) bool {
	fmt.Printf("MatchesPlayer1BackChecker called with filter: %s\n", filter) // Add logging

	backCheckers := 0
	for i := 19; i <= 24; i++ {
		if p.Board.Points[i].Color == 0 {
			backCheckers += p.Board.Points[i].Checkers
		}
	}
	fmt.Printf("Checking back checkers filter: %s, Player 1 Back Checkers: %d\n", filter, backCheckers)

	if strings.HasPrefix(filter, "k>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backCheckers >= value
	} else if strings.HasPrefix(filter, "k<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backCheckers <= value
	} else if strings.HasPrefix(filter, "k") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'kx' means 'kx,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backCheckers >= minValue && backCheckers <= maxValue
	}
	return false
}

// Add MatchesPlayer2BackChecker method to Position type with logging
func (p *Position) MatchesPlayer2BackChecker(filter string) bool {
	fmt.Printf("MatchesPlayer2BackChecker called with filter: %s\n", filter) // Add logging

	backCheckers := 0
	for i := 1; i <= 6; i++ {
		if p.Board.Points[i].Color == 1 {
			backCheckers += p.Board.Points[i].Checkers
		}
	}
	fmt.Printf("Checking back checkers filter: %s, Player 2 Back Checkers: %d\n", filter, backCheckers)

	if strings.HasPrefix(filter, "K>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backCheckers >= value
	} else if strings.HasPrefix(filter, "K<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backCheckers <= value
	} else if strings.HasPrefix(filter, "K") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Kx' means 'Kx,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backCheckers >= minValue && backCheckers <= maxValue
	}
	return false
}

// Add MatchesPlayer1CheckerInZone method to Position type with logging
func (p *Position) MatchesPlayer1CheckerInZone(filter string) bool {
	fmt.Printf("MatchesPlayer1CheckerInZone called with filter: %s\n", filter) // Add logging

	checkersInZone := 0
	for i := 0; i <= 12; i++ {
		if p.Board.Points[i].Color == 0 {
			checkersInZone += p.Board.Points[i].Checkers
		}
	}
	fmt.Printf("Checking checkers in zone filter: %s, Player 1 Checkers in Zone: %d\n", filter, checkersInZone)

	if strings.HasPrefix(filter, "z>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersInZone >= value
	} else if strings.HasPrefix(filter, "z<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersInZone <= value
	} else if strings.HasPrefix(filter, "z") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'zx' means 'zx,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersInZone >= minValue && checkersInZone <= maxValue
	}
	return false
}

// Add MatchesPlayer2CheckerInZone method to Position type with logging
func (p *Position) MatchesPlayer2CheckerInZone(filter string) bool {
	fmt.Printf("MatchesPlayer2CheckerInZone called with filter: %s\n", filter) // Add logging

	checkersInZone := 0
	for i := 13; i <= 25; i++ {
		if p.Board.Points[i].Color == 1 {
			checkersInZone += p.Board.Points[i].Checkers
		}
	}
	fmt.Printf("Checking checkers in zone filter: %s, Player 2 Checkers in Zone: %d\n", filter, checkersInZone)

	if strings.HasPrefix(filter, "Z>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersInZone >= value
	} else if strings.HasPrefix(filter, "Z<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersInZone <= value
	} else if strings.HasPrefix(filter, "Z") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Zx' means 'Zx,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersInZone >= minValue && checkersInZone <= maxValue
	}
	return false
}

// Add MatchesPlayer1AbsolutePipCount method to Position type
func (p *Position) MatchesPlayer1AbsolutePipCount(filter string) bool {
	player1PipCount, _ := p.ComputePipCounts()

	if strings.HasPrefix(filter, "P>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return player1PipCount >= value
	} else if strings.HasPrefix(filter, "P<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return player1PipCount <= value
	} else if strings.HasPrefix(filter, "P") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			value, err := strconv.Atoi(values[0])
			if err != nil {
				fmt.Printf("Error parsing filter value: %s\n", values[0])
				return false
			}
			return player1PipCount == value
		} else if len(values) == 2 {
			value1, err1 := strconv.Atoi(values[0])
			value2, err2 := strconv.Atoi(values[1])
			if err1 != nil || err2 != nil {
				fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
				return false
			}
			minValue := value1
			maxValue := value2
			if value1 > value2 {
				minValue = value2
				maxValue = value1
			}
			return player1PipCount >= minValue && player1PipCount <= maxValue
		}
	}
	return false
}

// Add MatchesEquityFilter method to Position type with detailed logging
func (p *Position) MatchesEquityFilter(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var equity float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		equity = analysis.DoublingCubeAnalysis.CubefulNoDoubleEquity
		fmt.Printf("Position ID: %d, Doubling decision, Equity: %f\n", p.ID, equity)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		equity = analysis.CheckerAnalysis.Moves[0].Equity
		fmt.Printf("Position ID: %d, Checker decision, Equity: %f\n", p.ID, equity)
	} else {
		fmt.Printf("Excluding position ID: %d due to no equity found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "e>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		value /= 1000 // Convert millipoints to points
		fmt.Printf("Equity filter condition: >, value: %f\n", value)
		return equity >= value
	} else if strings.HasPrefix(filter, "e<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		value /= 1000 // Convert millipoints to points
		fmt.Printf("Equity filter condition: <, value: %f\n", value)
		return equity <= value
	} else if strings.HasPrefix(filter, "e") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		value1 /= 1000 // Convert millipoints to points
		value2 /= 1000 // Convert millipoints to points
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		fmt.Printf("Equity filter condition: BETWEEN, values: %f, %f\n", minValue, maxValue)
		return equity >= minValue && equity <= maxValue
	}
	return false
}

// getPlayer1MovesForPosition returns checker moves and cube actions played by player1 for a position.
// Player1 is identified by player=1 in XG encoding in the move table.
func (d *Database) getPlayer1MovesForPosition(positionID int64) ([]string, []string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`SELECT checker_move, cube_action FROM move WHERE position_id = ? AND player = 1`, positionID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	checkerMoves := make(map[string]bool)
	cubeActions := make(map[string]bool)
	for rows.Next() {
		var cm sql.NullString
		var ca sql.NullString
		if err := rows.Scan(&cm, &ca); err != nil {
			continue
		}
		if cm.Valid && cm.String != "" {
			checkerMoves[normalizeMove(cm.String)] = true
		}
		if ca.Valid && ca.String != "" {
			cubeActions[ca.String] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil
	}

	var checkerMovesList []string
	for m := range checkerMoves {
		checkerMovesList = append(checkerMovesList, m)
	}
	var cubeActionsList []string
	for a := range cubeActions {
		cubeActionsList = append(cubeActionsList, a)
	}
	return checkerMovesList, cubeActionsList
}

// IsPlayer1TakePassCubeAction returns true if player1's cube action for this position
// was a take or pass (as opposed to double or no-double).
// This is used to determine board orientation: take/pass positions should be shown
// from the taker's perspective (mirrored) so player1 appears at the bottom.
func (p *Position) IsPlayer1TakePassCubeAction(d *Database) bool {
	_, player1CubeActions := d.getPlayer1MovesForPosition(p.ID)
	for _, action := range player1CubeActions {
		actionLower := strings.ToLower(action)
		if strings.Contains(actionLower, "take") || actionLower == "dt" ||
			strings.Contains(actionLower, "pass") || strings.Contains(actionLower, "drop") || actionLower == "dp" {
			return true
		}
	}
	return false
}

// MatchesMoveErrorFilter filters positions by the equity error of the played move (in millipoints).
// By default, only considers errors made by player1 (player1 in match context).
// Supports E>x, E<x, Ex,y syntax.
func (p *Position) MatchesMoveErrorFilter(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	// Get only player1's moves for this position from the move table
	player1CheckerMoves, player1CubeActions := d.getPlayer1MovesForPosition(p.ID)

	var moveError float64
	found := false

	if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		// Use only player1's moves (from match context)
		playedMoves := player1CheckerMoves
		if len(playedMoves) == 0 {
			return false
		}
		// Find the played move in the analysis moves and get its error
		for _, played := range playedMoves {
			for i, m := range analysis.CheckerAnalysis.Moves {
				if strings.EqualFold(strings.ReplaceAll(m.Move, " ", ""), strings.ReplaceAll(played, " ", "")) {
					if i == 0 {
						moveError = 0
					} else if m.EquityError != nil {
						moveError = math.Abs(*m.EquityError)
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	} else if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		// Use only player1's cube actions (from match context)
		playedActions := player1CubeActions
		if len(playedActions) == 0 {
			return false
		}
		bestAction := strings.ToLower(analysis.DoublingCubeAnalysis.BestCubeAction)
		for _, played := range playedActions {
			playedLower := strings.ToLower(played)
			if playedLower == bestAction {
				moveError = 0
				found = true
			} else {
				switch {
				case strings.Contains(playedLower, "no double") || playedLower == "nd":
					moveError = math.Abs(analysis.DoublingCubeAnalysis.CubefulNoDoubleError)
					found = true
				case strings.Contains(playedLower, "take") || playedLower == "dt":
					moveError = math.Abs(analysis.DoublingCubeAnalysis.CubefulDoubleTakeError)
					found = true
				case strings.Contains(playedLower, "pass") || strings.Contains(playedLower, "drop") || playedLower == "dp":
					moveError = math.Abs(analysis.DoublingCubeAnalysis.CubefulDoublePassError)
					found = true
				}
			}
			if found {
				break
			}
		}
	}

	if !found {
		return false
	}

	// Convert move error from equity points to millipoints for comparison
	moveErrorMillipoints := moveError * 1000

	if strings.HasPrefix(filter, "E>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			return false
		}
		return moveErrorMillipoints >= value
	} else if strings.HasPrefix(filter, "E<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			return false
		}
		return moveErrorMillipoints <= value
	} else if strings.HasPrefix(filter, "E") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return moveErrorMillipoints >= minValue && moveErrorMillipoints <= maxValue
	}
	return false
}

// Add MatchesDiceRoll method to Position type
func (p *Position) MatchesDiceRoll(filter Position) bool {
	dice := fmt.Sprintf("%d%d", p.Dice[0], p.Dice[1])
	reverseDice := fmt.Sprintf("%d%d", p.Dice[1], p.Dice[0])
	filterDice := fmt.Sprintf("%d%d", filter.Dice[0], filter.Dice[1])
	return (dice == filterDice || reverseDice == filterDice) && p.PlayerOnRoll == filter.PlayerOnRoll && p.DecisionType == filter.DecisionType
}

func (p *Position) MatchesMovePattern(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	// Extract the move pattern from the raw string
	movePatternMatch := strings.Trim(filter, `m"'`)
	movePatterns := strings.Split(strings.ToLower(movePatternMatch), ";")

	if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		move := strings.ToLower(analysis.CheckerAnalysis.Moves[0].Move)
		for _, pattern := range movePatterns {
			if strings.Contains(move, pattern) {
				fmt.Printf("Position ID: %d, Checker decision, Move: %s, Filter: %s\n", p.ID, move, pattern) // Add logging
				return true
			}
		}
	} else if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		for _, pattern := range movePatterns {
			switch pattern {
			case "nd":
				if analysis.DoublingCubeAnalysis.CubefulNoDoubleError == 0 {
					fmt.Printf("Position ID: %d, Doubling decision, No Double Error: %f, Filter: %s\n", p.ID, analysis.DoublingCubeAnalysis.CubefulNoDoubleError, pattern) // Add logging
					return true
				}
			case "dt":
				if analysis.DoublingCubeAnalysis.CubefulDoubleTakeError == 0 {
					fmt.Printf("Position ID: %d, Doubling decision, Double Take Error: %f, Filter: %s\n", p.ID, analysis.DoublingCubeAnalysis.CubefulDoubleTakeError, pattern) // Add logging
					return true
				}
			case "dp":
				if analysis.DoublingCubeAnalysis.CubefulDoublePassError == 0 {
					fmt.Printf("Position ID: %d, Doubling decision, Double Pass Error: %f, Filter: %s\n", p.ID, analysis.DoublingCubeAnalysis.CubefulDoublePassError, pattern) // Add logging
					return true
				}
			}
		}
	}
	fmt.Printf("Position ID: %d does not match move pattern filter: %s\n", p.ID, filter) // Add logging
	return false
}

func (d *Database) GetDatabaseVersion() (string, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	return DatabaseVersion, nil
}

func (d *Database) LoadMetadata() (map[string]string, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	rows, err := d.db.Query(`SELECT key, value FROM metadata WHERE key IN ('user', 'description', 'dateOfCreation', 'database_version')`)
	if err != nil {
		fmt.Println("Error loading metadata:", err)
		return nil, err
	}
	defer rows.Close()

	metadata := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
			fmt.Println("Error scanning metadata:", err)
			return nil, err
		}
		metadata[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return metadata, nil
}

func (d *Database) SaveMetadata(metadata map[string]string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for key, value := range metadata {
		_, err := tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
		if err != nil {
			fmt.Println("Error saving metadata:", err)
			return err
		}
	}
	return tx.Commit()
}

// Add MatchesDateFilter method to Position type
func (p *Position) MatchesDateFilter(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	creationDate := analysis.CreationDate
	fmt.Printf("Position ID: %d, Creation Date: %s\n", p.ID, creationDate)

	if strings.HasPrefix(filter, "T>") {
		dateStr := filter[2:]
		date, err := time.ParseInLocation("2006/01/02", dateStr, creationDate.Location())
		if err != nil {
			fmt.Printf("Error parsing date filter value: %s\n", dateStr)
			return false
		}
		fmt.Printf("Filter: T>, Date: %s\n", date)
		match := creationDate.After(date) || creationDate.Equal(date)
		fmt.Printf("Position ID: %d, Matches: %v\n", p.ID, match)
		return match
	} else if strings.HasPrefix(filter, "T<") {
		dateStr := filter[2:]
		date, err := time.ParseInLocation("2006/01/02", dateStr, creationDate.Location())
		if err != nil {
			fmt.Printf("Error parsing date filter value: %s\n", dateStr)
			return false
		}
		date = date.Add(24 * time.Hour).Add(-1 * time.Second) // Include the entire day
		fmt.Printf("Filter: T<, Date: %s\n", date)
		match := creationDate.Before(date)
		fmt.Printf("Position ID: %d, Matches: %v\n", p.ID, match)
		return match
	} else if strings.HasPrefix(filter, "T") {
		dateRange := strings.Split(filter[1:], ",")
		if len(dateRange) != 2 {
			fmt.Printf("Error parsing date range filter values: %s\n", filter[1:])
			return false
		}
		startDate, err1 := time.ParseInLocation("2006/01/02", dateRange[0], creationDate.Location())
		endDate, err2 := time.ParseInLocation("2006/01/02", dateRange[1], creationDate.Location())
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing date range filter values: %s, %s\n", dateRange[0], dateRange[1])
			return false
		}
		if startDate.After(endDate) {
			startDate, endDate = endDate, startDate // Swap to ensure correct order
		}
		endDate = endDate.Add(24 * time.Hour).Add(-1 * time.Second) // Include the entire day
		fmt.Printf("Filter: T, Start Date: %s, End Date: %s\n", startDate, endDate)
		match := (creationDate.After(startDate) || creationDate.Equal(startDate)) && (creationDate.Before(endDate) || creationDate.Equal(endDate))
		fmt.Printf("Position ID: %d, Matches: %v\n", p.ID, match)
		return match
	}
	return false
}

// Add MatchesPlayer1OutfieldBlot method to Position type
func (p *Position) MatchesPlayer1OutfieldBlot(filter string) bool {
	outfieldBlots := 0
	for i := 7; i <= 18; i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers == 1 {
			outfieldBlots++
		}
	}

	if strings.HasPrefix(filter, "bo>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return outfieldBlots >= value
	} else if strings.HasPrefix(filter, "bo<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return outfieldBlots <= value
	} else if strings.HasPrefix(filter, "bo") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[2:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return outfieldBlots >= minValue && outfieldBlots <= maxValue
	}
	return false
}

// Add MatchesPlayer2OutfieldBlot method to Position type
func (p *Position) MatchesPlayer2OutfieldBlot(filter string) bool {
	opponentOutfieldBlots := 0
	for i := 7; i <= 18; i++ {
		if p.Board.Points[i].Color == 1 && p.Board.Points[i].Checkers == 1 {
			opponentOutfieldBlots++
		}
	}

	if strings.HasPrefix(filter, "BO>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return opponentOutfieldBlots >= value
	} else if strings.HasPrefix(filter, "BO<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return opponentOutfieldBlots <= value
	} else if strings.HasPrefix(filter, "BO") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[2:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return opponentOutfieldBlots >= minValue && opponentOutfieldBlots <= maxValue
	}
	return false
}

// Add MatchesPlayer1JanBlot method to Position type
func (p *Position) MatchesPlayer1JanBlot(filter string) bool {
	janBlots := 0
	for i := 1; i <= 6; i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers == 1 {
			janBlots++
		}
	}

	if strings.HasPrefix(filter, "bj>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return janBlots >= value
	} else if strings.HasPrefix(filter, "bj<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return janBlots <= value
	} else if strings.HasPrefix(filter, "bj") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[2:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return janBlots >= minValue && janBlots <= maxValue
	}
	return false
}

// Add MatchesPlayer2JanBlot method to Position type
func (p *Position) MatchesPlayer2JanBlot(filter string) bool {
	opponentJanBlots := 0
	for i := 19; i <= 24; i++ {
		if p.Board.Points[i].Color == 1 && p.Board.Points[i].Checkers == 1 {
			opponentJanBlots++
		}
	}

	if strings.HasPrefix(filter, "BJ>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return opponentJanBlots >= value
	} else if strings.HasPrefix(filter, "BJ<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return opponentJanBlots <= value
	} else if strings.HasPrefix(filter, "BJ") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[2:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return opponentJanBlots >= minValue && opponentJanBlots <= maxValue
	}
	return false
}

// Add MatchesNoContact method to Position type
func (p *Position) MatchesNoContact() bool {
	var furthestPlayerChecker, furthestOpponentChecker int

	// Initialize to invalid indices
	furthestPlayerChecker = -1
	furthestOpponentChecker = 26

	for i := 0; i < len(p.Board.Points); i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers > 0 {
			furthestPlayerChecker = i
		}
		if p.Board.Points[25-i].Color == 1 && p.Board.Points[25-i].Checkers > 0 {
			furthestOpponentChecker = 25 - i
		}
	}

	// Compare indices to determine if there is no contact
	return furthestPlayerChecker < furthestOpponentChecker
}

func (p *Position) MatchesMirrorPosition(filter Position) bool {
	mirroredPosition := p.Mirror()
	return p.MatchesCheckerPosition(filter) || mirroredPosition.MatchesCheckerPosition(filter)
}

// Mirror creates a mirrored version of the current Position.
// It reverses the board points, swaps the bearoff positions,
// changes the player on roll, swaps the scores, and changes the cube owner.
// Returns the mirrored Position.
func (p *Position) Mirror() Position {
	mirrored := *p
	for i, point := range p.Board.Points {
		mirrored.Board.Points[25-i] = Point{
			Color:    point.Color,
			Checkers: point.Checkers,
		}
		if point.Color != -1 {
			mirrored.Board.Points[25-i].Color = 1 - point.Color
		}
	}
	mirrored.Board.Bearoff[0], mirrored.Board.Bearoff[1] = p.Board.Bearoff[1], p.Board.Bearoff[0]
	mirrored.PlayerOnRoll = 1 - p.PlayerOnRoll
	mirrored.Score[0], mirrored.Score[1] = p.Score[1], p.Score[0]
	if p.Cube.Owner != -1 {
		mirrored.Cube.Owner = 1 - p.Cube.Owner
	}
	return mirrored
}

// NormalizeForStorage returns a normalized version of the position for storage.
// Positions are always stored from the player on roll's perspective (player_on_roll = 0).
// If player_on_roll is 1, the position is mirrored so that player_on_roll becomes 0.
// This prevents storing duplicate positions that are just mirror images of each other.
func (p *Position) NormalizeForStorage() Position {
	if p.PlayerOnRoll == 1 {
		return p.Mirror()
	}
	return *p
}

// SaveCommand saves a command to the command_history table
func (d *Database) SaveCommand(command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.1.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.1.0" {
		return fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	_, err = d.db.Exec(`INSERT INTO command_history (command) VALUES (?)`, command)
	if err != nil {
		fmt.Println("Error saving command:", err)
		return err
	}
	return nil
}

// LoadCommandHistory loads the command history from the command_history table
func (d *Database) LoadCommandHistory() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.1.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return nil, err
	}

	if dbVersion < "1.1.0" {
		return nil, fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	rows, err := d.db.Query(`SELECT command FROM command_history ORDER BY timestamp ASC`)
	if err != nil {
		fmt.Println("Error loading command history:", err)
		return nil, err
	}
	defer rows.Close()

	var history []string
	for rows.Next() {
		var command string
		if err = rows.Scan(&command); err != nil {
			fmt.Println("Error scanning command:", err)
			return nil, err
		}
		history = append(history, command)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return history, nil
}

func (d *Database) ClearCommandHistory() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.1.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.1.0" {
		return fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	_, err = d.db.Exec(`DELETE FROM command_history`)
	if err != nil {
		fmt.Println("Error clearing command history:", err)
		return err
	}
	return nil
}

// SearchHistory represents a search history entry
type SearchHistory struct {
	ID        int    `json:"id"`
	Command   string `json:"command"`
	Position  string `json:"position"`
	Timestamp int64  `json:"timestamp"`
}

// SaveSearchHistory saves a search command and position to the search_history table
func (d *Database) SaveSearchHistory(command string, position string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

	// Insert the search history entry
	_, err := d.db.Exec(`INSERT INTO search_history (command, position, timestamp) VALUES (?, ?, ?)`,
		command, position, time.Now().UnixMilli())
	if err != nil {
		fmt.Println("Error saving search history:", err)
		return err
	}

	// Keep only the last 100 entries
	_, err = d.db.Exec(`
		DELETE FROM search_history 
		WHERE id NOT IN (
			SELECT id FROM search_history 
			ORDER BY timestamp DESC 
			LIMIT 100
		)
	`)
	if err != nil {
		fmt.Println("Error pruning search history:", err)
		return err
	}

	return nil
}

// LoadSearchHistory loads the search history from the search_history table
func (d *Database) LoadSearchHistory() ([]SearchHistory, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("database is not opened")
	}

	rows, err := d.db.Query(`SELECT id, command, position, timestamp FROM search_history ORDER BY timestamp DESC LIMIT 100`)
	if err != nil {
		fmt.Println("Error loading search history:", err)
		return nil, err
	}
	defer rows.Close()

	var history []SearchHistory
	for rows.Next() {
		var entry SearchHistory
		err := rows.Scan(&entry.ID, &entry.Command, &entry.Position, &entry.Timestamp)
		if err != nil {
			fmt.Println("Error scanning search history row:", err)
			return nil, err
		}
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return history, nil
}

// DeleteSearchHistoryEntry deletes a search history entry by timestamp
func (d *Database) DeleteSearchHistoryEntry(timestamp int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

	_, err := d.db.Exec(`DELETE FROM search_history WHERE timestamp = ?`, timestamp)
	if err != nil {
		fmt.Println("Error deleting search history entry:", err)
		return err
	}

	return nil
}

// SessionState represents the last session state for restoring when reopening a database
type SessionState struct {
	LastSearchCommand  string  `json:"lastSearchCommand"`  // The last search command executed
	LastSearchPosition string  `json:"lastSearchPosition"` // The position used for the last search (JSON)
	LastPositionIndex  int     `json:"lastPositionIndex"`  // The index of the last viewed position in results
	LastPositionIDs    []int64 `json:"lastPositionIds"`    // The list of position IDs from the last search
	HasActiveSearch    bool    `json:"hasActiveSearch"`    // Whether there was an active search session
}

// SaveSessionState saves the current session state to the metadata table
func (d *Database) SaveSessionState(state SessionState) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

	// Serialize position IDs to JSON
	positionIDsJSON, err := json.Marshal(state.LastPositionIDs)
	if err != nil {
		fmt.Println("Error marshaling position IDs:", err)
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Save each session state field as a metadata entry
	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_last_search_command', ?)`, state.LastSearchCommand)
	if err != nil {
		fmt.Println("Error saving session_last_search_command:", err)
		return err
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_last_search_position', ?)`, state.LastSearchPosition)
	if err != nil {
		fmt.Println("Error saving session_last_search_position:", err)
		return err
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_last_position_index', ?)`, strconv.Itoa(state.LastPositionIndex))
	if err != nil {
		fmt.Println("Error saving session_last_position_index:", err)
		return err
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_last_position_ids', ?)`, string(positionIDsJSON))
	if err != nil {
		fmt.Println("Error saving session_last_position_ids:", err)
		return err
	}

	hasActiveSearchStr := "false"
	if state.HasActiveSearch {
		hasActiveSearchStr = "true"
	}
	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_has_active_search', ?)`, hasActiveSearchStr)
	if err != nil {
		fmt.Println("Error saving session_has_active_search:", err)
		return err
	}

	return tx.Commit()
}

// LoadSessionState loads the last session state from the metadata table
func (d *Database) LoadSessionState() (*SessionState, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("database is not opened")
	}

	state := &SessionState{}

	// Load last search command
	var lastSearchCommand sql.NullString
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_last_search_command'`).Scan(&lastSearchCommand)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error loading session_last_search_command:", err)
		return nil, err
	}
	if lastSearchCommand.Valid {
		state.LastSearchCommand = lastSearchCommand.String
	}

	// Load last search position
	var lastSearchPosition sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_last_search_position'`).Scan(&lastSearchPosition)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error loading session_last_search_position:", err)
		return nil, err
	}
	if lastSearchPosition.Valid {
		state.LastSearchPosition = lastSearchPosition.String
	}

	// Load last position index
	var lastPositionIndex sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_last_position_index'`).Scan(&lastPositionIndex)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error loading session_last_position_index:", err)
		return nil, err
	}
	if lastPositionIndex.Valid {
		index, parseErr := strconv.Atoi(lastPositionIndex.String)
		if parseErr == nil {
			state.LastPositionIndex = index
		}
	}

	// Load last position IDs
	var lastPositionIDs sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_last_position_ids'`).Scan(&lastPositionIDs)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error loading session_last_position_ids:", err)
		return nil, err
	}
	if lastPositionIDs.Valid && lastPositionIDs.String != "" {
		var ids []int64
		if parseErr := json.Unmarshal([]byte(lastPositionIDs.String), &ids); parseErr == nil {
			state.LastPositionIDs = ids
		}
	}

	// Load has active search flag
	var hasActiveSearch sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_has_active_search'`).Scan(&hasActiveSearch)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error loading session_has_active_search:", err)
		return nil, err
	}
	if hasActiveSearch.Valid {
		state.HasActiveSearch = hasActiveSearch.String == "true"
	}

	return state, nil
}

// ClearSessionState clears the session state from the metadata table
func (d *Database) ClearSessionState() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

	sessionKeys := []string{
		"session_last_search_command",
		"session_last_search_position",
		"session_last_position_index",
		"session_last_position_ids",
		"session_has_active_search",
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, key := range sessionKeys {
		_, err := tx.Exec(`DELETE FROM metadata WHERE key = ?`, key)
		if err != nil {
			fmt.Println("Error deleting session key:", key, err)
			return err
		}
	}

	return tx.Commit()
}

func (d *Database) Migrate_1_0_0_to_1_1_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion != "1.0.0" {
		return fmt.Errorf("database version is not 1.0.0, current version: %s", dbVersion)
	}

	// Create the command_history table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		fmt.Println("Error creating command_history table:", err)
		return err
	}

	// Update the database version to 1.1.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.1.0")
	if err != nil {
		fmt.Println("Error updating database version:", err)
		return err
	}

	fmt.Println("Database successfully migrated from version 1.0.0 to 1.1.0")
	return nil
}

func (d *Database) Migrate_1_1_0_to_1_2_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion != "1.1.0" {
		return fmt.Errorf("database version is not 1.1.0, current version: %s", dbVersion)
	}

	// Create the filter_library table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS filter_library (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			command TEXT,
			edit_position TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating filter_library table:", err)
		return err
	}

	// Update the database version to 1.2.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.2.0")
	if err != nil {
		fmt.Println("Error updating database version:", err)
		return err
	}

	fmt.Println("Database successfully migrated from version 1.1.0 to 1.2.0")
	return nil
}

func (d *Database) Migrate_1_2_0_to_1_3_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion != "1.2.0" {
		return fmt.Errorf("database version is not 1.2.0, current version: %s", dbVersion)
	}

	// Create the search_history table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			position TEXT,
			timestamp INTEGER
		)
	`)
	if err != nil {
		fmt.Println("Error creating search_history table:", err)
		return err
	}

	// Update the database version to 1.3.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.3.0")
	if err != nil {
		fmt.Println("Error updating database version:", err)
		return err
	}

	fmt.Println("Database successfully migrated from version 1.2.0 to 1.3.0")
	return nil
}

func (d *Database) SaveFilter(name, command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	// Check if a filter with the same name already exists
	var existingID int64
	err = d.db.QueryRow(`SELECT id FROM filter_library WHERE name = ?`, name).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error checking existing filter:", err)
		return err
	}
	if existingID > 0 {
		return fmt.Errorf("filter name already exists")
	}

	_, err = d.db.Exec(`INSERT INTO filter_library (name, command) VALUES (?, ?)`, name, command)
	if err != nil {
		fmt.Println("Error saving filter:", err)
		return err
	}
	return nil
}

func (d *Database) UpdateFilter(id int64, name, command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	result, err := d.db.Exec(`UPDATE filter_library SET name = ?, command = ? WHERE id = ?`, name, command, id)
	if err != nil {
		fmt.Println("Error updating filter:", err)
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("filter with id %d not found", id)
	}
	return nil
}

func (d *Database) DeleteFilter(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	result, err := d.db.Exec(`DELETE FROM filter_library WHERE id = ?`, id)
	if err != nil {
		fmt.Println("Error deleting filter:", err)
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("filter with id %d not found", id)
	}
	return nil
}

func (d *Database) LoadFilters() ([]map[string]interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return nil, err
	}

	if dbVersion < "1.2.0" {
		return nil, fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	rows, err := d.db.Query(`SELECT id, name, command FROM filter_library`)
	if err != nil {
		fmt.Println("Error loading filters:", err)
		return nil, err
	}
	defer rows.Close()

	var filters []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, command string
		if err = rows.Scan(&id, &name, &command); err != nil {
			fmt.Println("Error scanning filter:", err)
			return nil, err
		}
		filters = append(filters, map[string]interface{}{
			"id":      id,
			"name":    name,
			"command": command,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return filters, nil
}

func (d *Database) SaveEditPosition(filterName, editPosition string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	// Check if a filter with the same name already exists
	var existingID int64
	err = d.db.QueryRow(`SELECT id FROM filter_library WHERE name = ?`, filterName).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error checking existing filter:", err)
		return err
	}
	if existingID > 0 {
		_, err = d.db.Exec(`UPDATE filter_library SET edit_position = ? WHERE id = ?`, editPosition, existingID)
		if err != nil {
			fmt.Println("Error updating edit position:", err)
			return err
		}
	} else {
		return fmt.Errorf("filter name does not exist")
	}

	return nil
}

func (d *Database) LoadEditPosition(filterName string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return "", err
	}

	if dbVersion < "1.2.0" {
		return "", fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	var editPosition string
	err = d.db.QueryRow(`SELECT edit_position FROM filter_library WHERE name = ?`, filterName).Scan(&editPosition)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No edit position found
		}
		fmt.Println("Error loading edit position:", err)
		return "", err
	}
	return editPosition, nil
}

// AnalyzeImportDatabase analyzes what would be imported without making changes
func (d *Database) AnalyzeImportDatabase(importPath string) (map[string]interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check that the current database is open
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	// Open the import database
	importDB, err := sql.Open("sqlite", importPath)
	if err != nil {
		fmt.Println("Error opening import database:", err)
		return nil, err
	}
	defer importDB.Close()

	// Check the import database version
	var importDBVersion string
	err = importDB.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&importDBVersion)
	if err != nil {
		fmt.Println("Error querying import database version:", err)
		return nil, fmt.Errorf("import database is invalid or missing version information")
	}

	// Check the current database version
	var currentDBVersion string
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&currentDBVersion)
	if err != nil {
		fmt.Println("Error querying current database version:", err)
		return nil, err
	}

	// Compare major versions - allow importing from same or lower version
	importMajor := strings.Split(importDBVersion, ".")[0]
	currentMajor := strings.Split(currentDBVersion, ".")[0]

	if importMajor > currentMajor {
		return nil, fmt.Errorf("cannot import from a newer major database version (import: %s, current: %s)", importDBVersion, currentDBVersion)
	}

	// Count total positions to import
	var totalPositions int
	err = importDB.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&totalPositions)
	if err != nil {
		fmt.Println("Error counting positions:", err)
		return nil, err
	}

	// OPTIMIZATION: Build a hash map of all current positions ONCE
	// This converts O(n²) to O(n) complexity
	currentPositionsMap := make(map[string]int64) // map[positionJSON]positionID
	currentRows, err := d.db.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error querying current database positions:", err)
		return nil, err
	}

	for currentRows.Next() {
		var currentID int64
		var currentStateJSON string
		if err = currentRows.Scan(&currentID, &currentStateJSON); err != nil {
			continue
		}

		var currentPosition Position
		if err = json.Unmarshal([]byte(currentStateJSON), &currentPosition); err != nil {
			continue
		}

		// Reset ID for comparison
		currentPosition.ID = 0
		currentPositionJSON, err := json.Marshal(currentPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		currentPositionsMap[string(currentPositionJSON)] = currentID
	}
	if err := currentRows.Err(); err != nil {
		return nil, err
	}
	currentRows.Close()

	fmt.Printf("Built index of %d positions in current database\n", len(currentPositionsMap))

	// Analyze what would happen
	rows, err := importDB.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error loading positions from import database:", err)
		return nil, err
	}
	defer rows.Close()

	var positionsToAdd int
	var positionsToMerge int
	var positionsToSkip int

	for rows.Next() {
		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			positionsToSkip++
			continue
		}

		var importPosition Position
		if err = json.Unmarshal([]byte(stateJSON), &importPosition); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			positionsToSkip++
			continue
		}

		// Reset ID for existence check
		importPosition.ID = 0
		importPositionJSON, err := json.Marshal(importPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}

		// OPTIMIZATION: O(1) hash map lookup instead of nested loop
		existingPositionID, existsInCurrent := currentPositionsMap[string(importPositionJSON)]

		if existsInCurrent {
			// Check if there's actually something to merge
			hasNewData := false

			// Check for analysis to merge
			var importAnalysisJSON string
			err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisJSON)
			if err == nil {
				var existingAnalysisJSON string
				existingErr := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, existingPositionID).Scan(&existingAnalysisJSON)

				if existingErr == sql.ErrNoRows {
					// New analysis to add
					hasNewData = true
				} else if existingErr == nil {
					// Check if import has better analysis
					var existingAnalysis PositionAnalysis
					var importAnalysis PositionAnalysis
					json.Unmarshal([]byte(existingAnalysisJSON), &existingAnalysis)
					json.Unmarshal([]byte(importAnalysisJSON), &importAnalysis)

					if existingAnalysis.AnalysisType == "" && importAnalysis.AnalysisType != "" {
						hasNewData = true
					}
				}
			}

			// Check for comments to merge
			var importComment string
			err = importDB.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, id).Scan(&importComment)
			if err == nil && importComment != "" {
				var existingComment string
				existingErr := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, existingPositionID).Scan(&existingComment)

				trimmedImport := strings.TrimSpace(importComment)
				trimmedExisting := strings.TrimSpace(existingComment)

				if existingErr == sql.ErrNoRows {
					// New comment to add
					hasNewData = true
				} else if existingErr == nil && trimmedImport != "" && !strings.Contains(trimmedExisting, trimmedImport) {
					// Comment text to merge
					hasNewData = true
				}
			}

			if hasNewData {
				positionsToMerge++
			} else {
				positionsToSkip++
			}
		} else {
			positionsToAdd++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"toAdd":      positionsToAdd,
		"toMerge":    positionsToMerge,
		"toSkip":     positionsToSkip,
		"total":      totalPositions,
		"importPath": importPath,
	}

	fmt.Printf("Import analysis: %d to add, %d to merge, %d to skip out of %d total\n", positionsToAdd, positionsToMerge, positionsToSkip, totalPositions)
	return result, nil
}

// CommitImportDatabase performs the actual import within a transaction (ACID)
func (d *Database) CommitImportDatabase(importPath string) (map[string]interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Reset cancellation flag at start
	d.resetImportCancellation()

	// Check that the current database is open
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	// Begin transaction for ACID compliance
	tx, err := d.db.Begin()
	if err != nil {
		fmt.Println("Error starting transaction:", err)
		return nil, err
	}

	// Ensure rollback on error or cancellation
	defer func() {
		if err != nil || d.isImportCancelled() {
			tx.Rollback()
			if d.isImportCancelled() {
				fmt.Println("Transaction rolled back due to user cancellation")
			} else {
				fmt.Println("Transaction rolled back due to error")
			}
		}
	}()

	// Open the import database
	importDB, err := sql.Open("sqlite", importPath)
	if err != nil {
		fmt.Println("Error opening import database:", err)
		return nil, err
	}
	defer importDB.Close()

	// Check the import database version
	var importDBVersion string
	err = importDB.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&importDBVersion)
	if err != nil {
		fmt.Println("Error querying import database version:", err)
		return nil, fmt.Errorf("import database is invalid or missing version information")
	}

	// Check the current database version
	var currentDBVersion string
	err = tx.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&currentDBVersion)
	if err != nil {
		fmt.Println("Error querying current database version:", err)
		return nil, err
	}

	// Compare major versions - allow importing from same or lower version
	importMajor := strings.Split(importDBVersion, ".")[0]
	currentMajor := strings.Split(currentDBVersion, ".")[0]

	if importMajor > currentMajor {
		return nil, fmt.Errorf("cannot import from a newer major database version (import: %s, current: %s)", importDBVersion, currentDBVersion)
	}

	// First, count total positions to import
	var totalPositions int
	err = importDB.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&totalPositions)
	if err != nil {
		fmt.Println("Error counting positions:", err)
		return nil, err
	}

	// OPTIMIZATION: Build a hash map of all current positions ONCE
	// This converts O(n²) to O(n) complexity
	currentPositionsMap := make(map[string]int64) // map[positionJSON]positionID
	currentRows, err := tx.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error querying current database positions:", err)
		return nil, err
	}

	for currentRows.Next() {
		var currentID int64
		var currentStateJSON string
		if err = currentRows.Scan(&currentID, &currentStateJSON); err != nil {
			continue
		}

		var currentPosition Position
		if err = json.Unmarshal([]byte(currentStateJSON), &currentPosition); err != nil {
			continue
		}

		// Reset ID for comparison
		currentPosition.ID = 0
		currentPositionJSON, err := json.Marshal(currentPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		currentPositionsMap[string(currentPositionJSON)] = currentID
	}
	if err := currentRows.Err(); err != nil {
		return nil, err
	}
	currentRows.Close()

	fmt.Printf("Built index of %d positions in current database for commit\n", len(currentPositionsMap))

	// Load all positions from the import database
	rows, err := importDB.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error loading positions from import database:", err)
		return nil, err
	}
	defer rows.Close()

	var positionsAdded int
	var positionsMerged int
	var positionsSkipped int

	for rows.Next() {
		// Check for cancellation
		if d.isImportCancelled() {
			fmt.Println("Import cancelled by user during processing")
			return nil, fmt.Errorf("import cancelled by user")
		}

		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			continue
		}

		var importPosition Position
		if err = json.Unmarshal([]byte(stateJSON), &importPosition); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			continue
		}

		// Reset ID for existence check
		importPosition.ID = 0
		importPositionJSON, err := json.Marshal(importPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}

		// OPTIMIZATION: O(1) hash map lookup instead of nested loop
		existingPositionID, existsInCurrent := currentPositionsMap[string(importPositionJSON)]

		if existsInCurrent {
			// Track if we actually merge anything
			hasMerged := false

			// Merge analysis if it exists
			var importAnalysisJSON string
			err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisJSON)

			if err == nil {
				// Load existing analysis from current database (using transaction)
				var existingAnalysisJSON string
				existingErr := tx.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, existingPositionID).Scan(&existingAnalysisJSON)

				if existingErr == sql.ErrNoRows {
					// No existing analysis, insert the imported one
					_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, existingPositionID, importAnalysisJSON)
					if err != nil {
						fmt.Printf("Error inserting analysis for position %d: %v\n", existingPositionID, err)
					} else {
						hasMerged = true
					}
				} else if existingErr == nil {
					// Both have analysis - keep the existing one unless it's empty
					var existingAnalysis PositionAnalysis
					var importAnalysis PositionAnalysis

					json.Unmarshal([]byte(existingAnalysisJSON), &existingAnalysis)
					json.Unmarshal([]byte(importAnalysisJSON), &importAnalysis)

					// If import has analysis but existing doesn't, use import
					if existingAnalysis.AnalysisType == "" && importAnalysis.AnalysisType != "" {
						_, err = tx.Exec(`UPDATE analysis SET data = ? WHERE position_id = ?`, importAnalysisJSON, existingPositionID)
						if err != nil {
							fmt.Printf("Error updating analysis for position %d: %v\n", existingPositionID, err)
						} else {
							hasMerged = true
						}
					}
				}
			}

			// Merge comments
			var importComment string
			err = importDB.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, id).Scan(&importComment)

			if err == nil && importComment != "" {
				var existingComment string
				existingErr := tx.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, existingPositionID).Scan(&existingComment)

				trimmedImport := strings.TrimSpace(importComment)
				trimmedExisting := strings.TrimSpace(existingComment)

				if existingErr == sql.ErrNoRows {
					// No existing comment, insert the imported one
					_, err = tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, existingPositionID, importComment)
					if err != nil {
						fmt.Printf("Error inserting comment for position %d: %v\n", existingPositionID, err)
					} else {
						hasMerged = true
					}
				} else if existingErr == nil {
					// Merge comments - only add if not already present
					if trimmedImport != "" && !strings.Contains(trimmedExisting, trimmedImport) {
						mergedComment := trimmedExisting
						if trimmedExisting != "" {
							mergedComment = trimmedExisting + "\n\n" + trimmedImport
						} else {
							mergedComment = trimmedImport
						}
						_, err = tx.Exec(`UPDATE comment SET text = ? WHERE position_id = ?`, mergedComment, existingPositionID)
						if err != nil {
							fmt.Printf("Error updating comment for position %d: %v\n", existingPositionID, err)
						} else {
							hasMerged = true
						}
					}
				}
			}

			if hasMerged {
				positionsMerged++
			} else {
				positionsSkipped++
			}
		} else {
			// Position doesn't exist, add it (using transaction)
			result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, stateJSON)
			if err != nil {
				fmt.Println("Error inserting position:", err)
				positionsSkipped++
				continue
			}

			newPositionID, err := result.LastInsertId()
			if err != nil {
				fmt.Println("Error getting last insert ID:", err)
				positionsSkipped++
				continue
			}

			// Update the position ID in the state JSON
			importPosition.ID = newPositionID
			updatedStateJSON, err := json.Marshal(importPosition)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal JSON: %w", err)
			}
			_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(updatedStateJSON), newPositionID)
			if err != nil {
				fmt.Println("Error updating position with ID:", err)
			}

			// Copy analysis if it exists
			var importAnalysisJSON string
			err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisJSON)
			if err == nil {
				// Update position_id in the analysis JSON
				var analysis PositionAnalysis
				json.Unmarshal([]byte(importAnalysisJSON), &analysis)
				analysis.PositionID = int(newPositionID)
				updatedAnalysisJSON, err := json.Marshal(analysis)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal JSON: %w", err)
				}

				_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newPositionID, string(updatedAnalysisJSON))
				if err != nil {
					fmt.Printf("Error inserting analysis for new position %d: %v\n", newPositionID, err)
				}
			}

			// Copy comment if it exists
			var importComment string
			err = importDB.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, id).Scan(&importComment)
			if err == nil && importComment != "" {
				_, err = tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newPositionID, importComment)
				if err != nil {
					fmt.Printf("Error inserting comment for new position %d: %v\n", newPositionID, err)
				}
			}

			positionsAdded++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Final check for cancellation before committing
	if d.isImportCancelled() {
		fmt.Println("Import cancelled by user before commit")
		return nil, fmt.Errorf("import cancelled by user")
	}

	// Commit the transaction - this makes all changes atomic
	err = tx.Commit()
	if err != nil {
		fmt.Println("Error committing transaction:", err)
		return nil, err
	}

	result := map[string]interface{}{
		"added":   positionsAdded,
		"merged":  positionsMerged,
		"skipped": positionsSkipped,
		"total":   totalPositions,
	}

	fmt.Printf("Import committed: %d added, %d merged, %d skipped out of %d total\n", positionsAdded, positionsMerged, positionsSkipped, totalPositions)
	return result, nil
}

// CancelImport sets the flag to cancel any ongoing import operation
func (d *Database) CancelImport() {
	atomic.StoreInt32(&d.importCancelled, 1)
	fmt.Println("Import cancellation requested")
}

// isImportCancelled checks if import has been cancelled (internal method, no lock needed as it's called within locked context)
func (d *Database) isImportCancelled() bool {
	return atomic.LoadInt32(&d.importCancelled) == 1
}

// resetImportCancellation resets the cancellation flag (internal method)
func (d *Database) resetImportCancellation() {
	atomic.StoreInt32(&d.importCancelled, 0)
}

// Deprecated: Use AnalyzeImportDatabase followed by CommitImportDatabase instead
func (d *Database) ImportDatabase(importPath string) (map[string]interface{}, error) {
	// This function is kept for backward compatibility but redirects to the new ACID approach
	return d.CommitImportDatabase(importPath)
}

// ExportDatabase creates a new database file containing the current selection of positions
// with their analysis and comments
func (d *Database) ExportDatabase(exportPath string, positions []Position, metadata map[string]string, includeAnalysis bool, includeComments bool, includeFilterLibrary bool, includePlayedMoves bool, includeMatches bool, includeCollections bool, collectionIDs []int64, matchIDs []int64, tournamentIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check that the current database is open
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Delete the export file if it already exists
	if _, err := os.Stat(exportPath); err == nil {
		// File exists, remove it
		if err := os.Remove(exportPath); err != nil {
			return fmt.Errorf("cannot remove existing export file: %v", err)
		}
		fmt.Printf("Removed existing export file: %s\n", exportPath)
	}

	// Create a new database for export
	exportDB, err := sql.Open("sqlite", exportPath)
	if err != nil {
		fmt.Println("Error creating export database:", err)
		return err
	}
	defer exportDB.Close()

	// Create the schema for the export database
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating position table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER UNIQUE,
			data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		fmt.Println("Error creating analysis table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS comment (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER UNIQUE,
			text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		fmt.Println("Error creating comment table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating metadata table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		fmt.Println("Error creating command_history table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS filter_library (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			command TEXT,
			edit_position TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating filter_library table in export database:", err)
		return err
	}

	// Create search_history table (required for version >= 1.3.0)
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			position TEXT,
			timestamp INTEGER
		)
	`)
	if err != nil {
		fmt.Println("Error creating search_history table in export database:", err)
		return err
	}

	// Create match-related tables (required for version >= 1.4.0)
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS match (
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
			match_hash TEXT,
			tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL,
			last_visited_position INTEGER DEFAULT -1
		)
	`)
	if err != nil {
		fmt.Println("Error creating match table in export database:", err)
		return err
	}

	// Create tournament table
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS tournament (
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
		fmt.Println("Error creating tournament table in export database:", err)
		return err
	}

	// Create collection tables (required for version >= 1.5.0)
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS collection (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		fmt.Println("Error creating collection table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS collection_position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL,
			position_id INTEGER NOT NULL,
			sort_order INTEGER DEFAULT 0,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(collection_id, position_id)
		)
	`)
	if err != nil {
		fmt.Println("Error creating collection_position table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER,
			game_number INTEGER,
			initial_score_1 INTEGER,
			initial_score_2 INTEGER,
			winner INTEGER,
			points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		fmt.Println("Error creating game table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS move (
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
		)
	`)
	if err != nil {
		fmt.Println("Error creating move table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS move_analysis (
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
		)
	`)
	if err != nil {
		fmt.Println("Error creating move_analysis table in export database:", err)
		return err
	}

	// Insert database version
	_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		fmt.Println("Error inserting database version in export database:", err)
		return err
	}

	// Insert metadata (user, description, dateOfCreation)
	for key, value := range metadata {
		if value != "" {
			_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
			if err != nil {
				fmt.Printf("Error inserting metadata %s in export database: %v\n", key, err)
			}
		}
	}

	// If dateOfCreation is not provided, set it to current date
	if metadata["dateOfCreation"] == "" {
		currentDate := time.Now().Format("2006-01-02")
		_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('dateOfCreation', ?)`, currentDate)
		if err != nil {
			fmt.Println("Error inserting default creation date in export database:", err)
		}
	}

	// Begin transaction for export
	tx, err := exportDB.Begin()
	if err != nil {
		fmt.Println("Error starting transaction for export:", err)
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			fmt.Println("Transaction rolled back due to error during export")
		}
	}()

	// Export all positions with their analysis and comments
	idMapping := make(map[int64]int64) // map old position ID to new position ID

	for _, position := range positions {
		oldPositionID := position.ID

		// Reset the ID for the new database
		position.ID = 0

		// Marshal the position
		positionJSON, err := json.Marshal(position)
		if err != nil {
			fmt.Printf("Error marshalling position %d: %v\n", oldPositionID, err)
			continue
		}

		// Insert the position into the export database
		result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
		if err != nil {
			fmt.Printf("Error inserting position %d into export database: %v\n", oldPositionID, err)
			continue
		}

		newPositionID, err := result.LastInsertId()
		if err != nil {
			fmt.Printf("Error getting last insert ID for position %d: %v\n", oldPositionID, err)
			continue
		}

		// Update the position ID in the state JSON
		position.ID = newPositionID
		updatedPositionJSON, err := json.Marshal(position)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(updatedPositionJSON), newPositionID)
		if err != nil {
			fmt.Printf("Error updating position %d with new ID: %v\n", newPositionID, err)
		}

		// Store the ID mapping
		idMapping[oldPositionID] = newPositionID

		// Export analysis if it exists and if includeAnalysis is true
		if includeAnalysis {
			var analysisJSON string
			analysisErr := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, oldPositionID).Scan(&analysisJSON)
			if analysisErr == nil {
				// Update position_id in the analysis JSON
				var analysis PositionAnalysis
				if unmarshalErr := json.Unmarshal([]byte(analysisJSON), &analysis); unmarshalErr == nil {
					analysis.PositionID = int(newPositionID)

					// Handle played moves
					if includePlayedMoves {
						// Load played moves from the move table and merge with existing
						moveRows, moveErr := d.db.Query(`
							SELECT checker_move, cube_action 
							FROM move 
							WHERE position_id = ?
						`, oldPositionID)

						if moveErr == nil {

							// Collect all moves from the database
							existingMoves := make(map[string]bool)
							existingCubeActions := make(map[string]bool)

							// Include existing PlayedMoves from analysis JSON
							for _, m := range analysis.PlayedMoves {
								if m != "" {
									existingMoves[normalizeMove(m)] = true
								}
							}
							if analysis.PlayedMove != "" {
								existingMoves[normalizeMove(analysis.PlayedMove)] = true
							}

							// Include existing PlayedCubeActions from analysis JSON
							for _, a := range analysis.PlayedCubeActions {
								if a != "" {
									existingCubeActions[a] = true
								}
							}
							if analysis.PlayedCubeAction != "" {
								existingCubeActions[analysis.PlayedCubeAction] = true
							}

							// Add moves from move table
							for moveRows.Next() {
								var checkerMove sql.NullString
								var cubeAction sql.NullString
								if scanErr := moveRows.Scan(&checkerMove, &cubeAction); scanErr == nil {
									if checkerMove.Valid && checkerMove.String != "" {
										existingMoves[normalizeMove(checkerMove.String)] = true
									}
									if cubeAction.Valid && cubeAction.String != "" {
										existingCubeActions[cubeAction.String] = true
									}
								}
							}
							if err := moveRows.Err(); err != nil {
								return err
							}
							moveRows.Close()

							// Convert to slices
							analysis.PlayedMoves = make([]string, 0, len(existingMoves))
							for m := range existingMoves {
								analysis.PlayedMoves = append(analysis.PlayedMoves, m)
							}
							sort.Strings(analysis.PlayedMoves)

							analysis.PlayedCubeActions = make([]string, 0, len(existingCubeActions))
							for a := range existingCubeActions {
								analysis.PlayedCubeActions = append(analysis.PlayedCubeActions, a)
							}
							sort.Strings(analysis.PlayedCubeActions)
						}
					} else {
						// Clear played move fields if includePlayedMoves is false
						analysis.PlayedMove = ""
						analysis.PlayedCubeAction = ""
						analysis.PlayedMoves = nil
						analysis.PlayedCubeActions = nil
					}

					updatedAnalysisJSON, err := json.Marshal(analysis)
					if err != nil {
						return fmt.Errorf("failed to marshal JSON: %w", err)
					}

					if _, insertErr := tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newPositionID, string(updatedAnalysisJSON)); insertErr != nil {
						fmt.Printf("Error inserting analysis for position %d (old ID: %d): %v\n", newPositionID, oldPositionID, insertErr)
					}
				}
			} else if analysisErr != sql.ErrNoRows {
				fmt.Printf("Error querying analysis for position %d: %v\n", oldPositionID, analysisErr)
			}
		}

		// Export comment if it exists and if includeComments is true
		if includeComments {
			var comment string
			commentErr := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, oldPositionID).Scan(&comment)
			if commentErr == nil && comment != "" {
				if _, insertErr := tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newPositionID, comment); insertErr != nil {
					fmt.Printf("Error inserting comment for position %d (old ID: %d): %v\n", newPositionID, oldPositionID, insertErr)
				}
			} else if commentErr != nil && commentErr != sql.ErrNoRows {
				fmt.Printf("Error querying comment for position %d: %v\n", oldPositionID, commentErr)
			}
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		fmt.Println("Error committing transaction for export:", err)
		return err
	}

	// Export filter library if includeFilterLibrary is true
	if includeFilterLibrary {
		rows, err := d.db.Query(`SELECT name, command, COALESCE(edit_position, '') FROM filter_library`)
		if err == nil {
			defer rows.Close()

			for rows.Next() {
				var name, command, editPosition string
				err := rows.Scan(&name, &command, &editPosition)
				if err == nil {
					_, err = exportDB.Exec(`INSERT INTO filter_library (name, command, edit_position) VALUES (?, ?, ?)`, name, command, editPosition)
					if err != nil {
						fmt.Printf("Error inserting filter library entry '%s': %v\n", name, err)
					}
				}
			}
			if err := rows.Err(); err != nil {
				return err
			}
		}
	}

	// Export matches if includeMatches is true
	matchIDMapping := make(map[int64]int64) // old match ID -> new match ID (accessible for tournament linking)
	if includeMatches {
		matchCount := 0
		gameCount := 0
		moveCount := 0
		moveAnalysisCount := 0

		// Get matches - filter by matchIDs if provided, otherwise get all
		var matchRows *sql.Rows
		if len(matchIDs) > 0 {
			// Build IN clause for specific match IDs
			placeholders := make([]string, len(matchIDs))
			args := make([]interface{}, len(matchIDs))
			for i, id := range matchIDs {
				placeholders[i] = "?"
				args[i] = id
			}
			query := fmt.Sprintf(`
				SELECT id, player1_name, player2_name, event, location, round,
				       match_length, match_date, import_date, file_path, game_count, match_hash, tournament_id
				FROM match
				WHERE id IN (%s)
			`, strings.Join(placeholders, ","))
			matchRows, err = d.db.Query(query, args...)
		} else {
			matchRows, err = d.db.Query(`
				SELECT id, player1_name, player2_name, event, location, round,
				       match_length, match_date, import_date, file_path, game_count, match_hash, tournament_id
				FROM match
			`)
		}
		if err == nil {
			defer matchRows.Close()

			for matchRows.Next() {
				var oldMatchID int64
				var player1Name, player2Name, event, location, round, filePath string
				var matchLength int32
				var matchDate, importDate time.Time
				var gameCountVal int
				var matchHash sql.NullString
				var tournamentID sql.NullInt64

				err := matchRows.Scan(&oldMatchID, &player1Name, &player2Name, &event, &location, &round,
					&matchLength, &matchDate, &importDate, &filePath, &gameCountVal, &matchHash, &tournamentID)
				if err != nil {
					fmt.Printf("Error scanning match: %v\n", err)
					continue
				}

				// Insert match into export database
				var result sql.Result
				if matchHash.Valid {
					result, err = exportDB.Exec(`
						INSERT INTO match (player1_name, player2_name, event, location, round,
						                   match_length, match_date, import_date, file_path, game_count, match_hash)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, player1Name, player2Name, event, location, round,
						matchLength, matchDate, importDate, filePath, gameCountVal, matchHash.String)
				} else {
					result, err = exportDB.Exec(`
						INSERT INTO match (player1_name, player2_name, event, location, round,
						                   match_length, match_date, import_date, file_path, game_count)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, player1Name, player2Name, event, location, round,
						matchLength, matchDate, importDate, filePath, gameCountVal)
				}
				if err != nil {
					fmt.Printf("Error inserting match: %v\n", err)
					continue
				}

				newMatchID, err := result.LastInsertId()
				if err != nil {
					fmt.Printf("Error getting new match ID: %v\n", err)
					continue
				}
				matchIDMapping[oldMatchID] = newMatchID
				matchCount++
			}
			if err := matchRows.Err(); err != nil {
				return err
			}

			// Export games for each match
			gameIDMapping := make(map[int64]int64) // old game ID -> new game ID

			for oldMatchID, newMatchID := range matchIDMapping {
				gameRows, err := d.db.Query(`
					SELECT id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count
					FROM game
					WHERE match_id = ?
				`, oldMatchID)
				if err != nil {
					fmt.Printf("Error querying games for match %d: %v\n", oldMatchID, err)
					continue
				}

				for gameRows.Next() {
					var oldGameID int64
					var gameNumber, score1, score2, winner, pointsWon int32
					var moveCountVal int

					err := gameRows.Scan(&oldGameID, &gameNumber, &score1, &score2, &winner, &pointsWon, &moveCountVal)
					if err != nil {
						fmt.Printf("Error scanning game: %v\n", err)
						continue
					}

					result, err := exportDB.Exec(`
						INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count)
						VALUES (?, ?, ?, ?, ?, ?, ?)
					`, newMatchID, gameNumber, score1, score2, winner, pointsWon, moveCountVal)
					if err != nil {
						fmt.Printf("Error inserting game: %v\n", err)
						continue
					}

					newGameID, err := result.LastInsertId()
					if err != nil {
						fmt.Printf("Error getting new game ID: %v\n", err)
						continue
					}
					gameIDMapping[oldGameID] = newGameID
					gameCount++
				}
				if err := gameRows.Err(); err != nil {
					return err
				}
				gameRows.Close()
			}

			// Export moves for each game
			moveIDMapping := make(map[int64]int64) // old move ID -> new move ID

			for oldGameID, newGameID := range gameIDMapping {
				moveRows, err := d.db.Query(`
					SELECT id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action
					FROM move
					WHERE game_id = ?
				`, oldGameID)
				if err != nil {
					fmt.Printf("Error querying moves for game %d: %v\n", oldGameID, err)
					continue
				}

				for moveRows.Next() {
					var oldMoveID, positionID int64
					var moveNumber, player, dice1, dice2 int32
					var moveType string
					var checkerMove, cubeAction sql.NullString

					err := moveRows.Scan(&oldMoveID, &moveNumber, &moveType, &positionID, &player, &dice1, &dice2, &checkerMove, &cubeAction)
					if err != nil {
						fmt.Printf("Error scanning move: %v\n", err)
						continue
					}

					// Map the position ID to the new database
					newPositionID, posExists := idMapping[positionID]
					if !posExists {
						// Position might not have been exported (not in the selection)
						// Still export the move but with null position_id
						newPositionID = 0
					}

					var result sql.Result
					if newPositionID > 0 {
						result, err = exportDB.Exec(`
							INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action)
							VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
						`, newGameID, moveNumber, moveType, newPositionID, player, dice1, dice2,
							checkerMove.String, cubeAction.String)
					} else {
						result, err = exportDB.Exec(`
							INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action)
							VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?)
						`, newGameID, moveNumber, moveType, player, dice1, dice2,
							checkerMove.String, cubeAction.String)
					}
					if err != nil {
						fmt.Printf("Error inserting move: %v\n", err)
						continue
					}

					newMoveID, err := result.LastInsertId()
					if err != nil {
						fmt.Printf("Error getting new move ID: %v\n", err)
						continue
					}
					moveIDMapping[oldMoveID] = newMoveID
					moveCount++
				}
				if err := moveRows.Err(); err != nil {
					return err
				}
				moveRows.Close()
			}

			// Export move analysis for each move
			for oldMoveID, newMoveID := range moveIDMapping {
				analysisRows, err := d.db.Query(`
					SELECT analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate,
					       opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate
					FROM move_analysis
					WHERE move_id = ?
				`, oldMoveID)
				if err != nil {
					continue
				}

				for analysisRows.Next() {
					var analysisType, depth string
					var equity, equityError, winRate, gammonRate, backgammonRate float64
					var oppWinRate, oppGammonRate, oppBackgammonRate float64

					err := analysisRows.Scan(&analysisType, &depth, &equity, &equityError, &winRate, &gammonRate, &backgammonRate,
						&oppWinRate, &oppGammonRate, &oppBackgammonRate)
					if err != nil {
						continue
					}

					_, err = exportDB.Exec(`
						INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate,
						                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, newMoveID, analysisType, depth, equity, equityError, winRate, gammonRate, backgammonRate,
						oppWinRate, oppGammonRate, oppBackgammonRate)
					if err != nil {
						fmt.Printf("Error inserting move analysis: %v\n", err)
						continue
					}
					moveAnalysisCount++
				}
				if err := analysisRows.Err(); err != nil {
					return err
				}
				analysisRows.Close()
			}
		}

		fmt.Printf("Exported %d matches, %d games, %d moves, %d move analyses\n", matchCount, gameCount, moveCount, moveAnalysisCount)
	}

	// Export collections if requested
	if includeCollections && len(collectionIDs) > 0 {
		collectionCount := 0
		collectionPosCount := 0

		for _, collectionID := range collectionIDs {
			var name, description string
			var sortOrder int
			var createdAt, updatedAt string
			err := d.db.QueryRow(`SELECT name, COALESCE(description, ''), sort_order, created_at, updated_at FROM collection WHERE id = ?`, collectionID).
				Scan(&name, &description, &sortOrder, &createdAt, &updatedAt)
			if err != nil {
				fmt.Printf("Error reading collection %d: %v\n", collectionID, err)
				continue
			}

			result, err := exportDB.Exec(`INSERT INTO collection (name, description, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
				name, description, sortOrder, createdAt, updatedAt)
			if err != nil {
				fmt.Printf("Error inserting collection %d: %v\n", collectionID, err)
				continue
			}
			newCollectionID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}
			collectionCount++

			// Export collection_position mappings
			cpRows, err := d.db.Query(`SELECT position_id, sort_order, added_at FROM collection_position WHERE collection_id = ?`, collectionID)
			if err != nil {
				fmt.Printf("Error querying collection_position for collection %d: %v\n", collectionID, err)
				continue
			}
			for cpRows.Next() {
				var oldPosID int64
				var cpSortOrder int
				var addedAt string
				if err := cpRows.Scan(&oldPosID, &cpSortOrder, &addedAt); err != nil {
					continue
				}
				if newPosID, ok := idMapping[oldPosID]; ok {
					_, _ = exportDB.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order, added_at) VALUES (?, ?, ?, ?)`,
						newCollectionID, newPosID, cpSortOrder, addedAt)
					collectionPosCount++
				}
			}
			if err := cpRows.Err(); err != nil {
				return err
			}
			cpRows.Close()
		}

		fmt.Printf("Exported %d collections with %d position mappings\n", collectionCount, collectionPosCount)
	}

	// Export tournaments if requested
	if len(tournamentIDs) > 0 {
		tournamentCount := 0
		tournamentIDMapping := make(map[int64]int64)

		for _, tournamentID := range tournamentIDs {
			var name string
			var date, location sql.NullString
			var sortOrder int
			var createdAt, updatedAt string
			err := d.db.QueryRow(`SELECT name, date, location, sort_order, created_at, updated_at FROM tournament WHERE id = ?`, tournamentID).
				Scan(&name, &date, &location, &sortOrder, &createdAt, &updatedAt)
			if err != nil {
				fmt.Printf("Error reading tournament %d: %v\n", tournamentID, err)
				continue
			}

			result, err := exportDB.Exec(`INSERT INTO tournament (name, date, location, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
				name, date, location, sortOrder, createdAt, updatedAt)
			if err != nil {
				fmt.Printf("Error inserting tournament %d: %v\n", tournamentID, err)
				continue
			}
			newTournamentID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}
			tournamentIDMapping[tournamentID] = newTournamentID
			tournamentCount++
		}

		// Update tournament_id on exported matches that belong to exported tournaments
		if includeMatches && len(matchIDMapping) > 0 {
			matchTournamentRows, mterr := d.db.Query(`SELECT id, tournament_id FROM match WHERE tournament_id IS NOT NULL`)
			if mterr == nil {
				for matchTournamentRows.Next() {
					var oldMatchID int64
					var oldTournamentID int64
					if err := matchTournamentRows.Scan(&oldMatchID, &oldTournamentID); err == nil {
						newMatchID, matchExported := matchIDMapping[oldMatchID]
						newTournamentID, tournamentExported := tournamentIDMapping[oldTournamentID]
						if matchExported && tournamentExported {
							_, _ = exportDB.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, newTournamentID, newMatchID)
						}
					}
				}
				if err := matchTournamentRows.Err(); err != nil {
					return err
				}
				matchTournamentRows.Close()
			}
		}

		fmt.Printf("Exported %d tournaments\n", tournamentCount)
	}

	fmt.Printf("Successfully exported %d positions to %s\n", len(positions), exportPath)
	return nil
}

// DeleteFile is a helper function to delete a file
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}

// Match import and management functions

// Import XG match file using xgparser library
// This function uses raw segment parsing to capture complete cube action information
func (d *Database) ImportXGMatch(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Parse the XG file using raw segments for complete data
	imp := xgparser.NewImport(filePath)
	segments, err := imp.GetFileSegments()
	if err != nil {
		return 0, fmt.Errorf("failed to get file segments: %w", err)
	}

	// Also parse lightweight structure for metadata
	match, err := xgparser.ParseXG(segments)
	if err != nil {
		fmt.Printf("Error parsing XG file: %v\n", err)
		return 0, fmt.Errorf("failed to parse XG file: %w", err)
	}

	// Parse raw records for complete cube information
	rawCubeInfo := make(map[string]*RawCubeAction) // key: "game_cubeIdx"
	for _, seg := range segments {
		if seg.Type == xgparser.SegmentXGGameFile {
			records, _ := xgparser.ParseGameFile(seg.Data, -1)
			gameNum := int32(0)
			cubeIdx := 0
			for _, rec := range records {
				switch r := rec.(type) {
				case *xgparser.HeaderGameEntry:
					gameNum = r.GameNumber
					cubeIdx = 0
				case *xgparser.CubeEntry:
					if r.Double != -2 { // Skip initial positions
						key := fmt.Sprintf("%d_%d", gameNum, cubeIdx)
						rawCubeInfo[key] = &RawCubeAction{
							Double:   r.Double,
							Take:     r.Take,
							ActiveP:  r.ActiveP,
							CubeB:    r.CubeB,
							Position: r.Position,
							Doubled:  r.Doubled,
						}
						cubeIdx++
					}
				}
			}
		}
	}

	// Parse match date
	var matchDate time.Time
	if match.Metadata.DateTime != "" {
		// Try to parse various date formats
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, match.Metadata.DateTime); err == nil {
				matchDate = t
				break
			}
		}
	}
	if matchDate.IsZero() {
		matchDate = time.Now()
	}

	// Compute match hash for duplicate detection (includes full match transcription)
	matchHash := ComputeMatchHash(match)

	// Compute canonical hash (format-independent) for cross-format duplicate detection
	canonicalHash := ComputeCanonicalMatchHashFromXG(match)

	// Check if this exact match already exists (same format)
	existingMatchID, err := d.checkMatchExistsLocked(matchHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for duplicate match: %w", err)
	}
	if existingMatchID > 0 {
		return 0, ErrDuplicateMatch
	}

	// Check if same match was imported from a different format (canonical duplicate)
	canonicalMatchID, err := d.checkCanonicalMatchExistsLocked(canonicalHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for canonical duplicate: %w", err)
	}
	isCanonicalDuplicate := canonicalMatchID > 0

	// Begin transaction for atomic import
	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Insert match metadata (including match_hash and canonical_hash)
	// If canonical duplicate, don't create new match - reuse existing match ID
	var matchID int64
	if isCanonicalDuplicate {
		matchID = canonicalMatchID
		fmt.Printf("Canonical duplicate detected - reusing match ID %d, importing new analysis only\n", matchID)
	} else {
		result, err := tx.Exec(`
			INSERT INTO match (player1_name, player2_name, event, location, round, 
			                   match_length, match_date, file_path, game_count, match_hash, canonical_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, match.Metadata.Player1Name, match.Metadata.Player2Name,
			match.Metadata.Event, match.Metadata.Location, match.Metadata.Round,
			match.Metadata.MatchLength, matchDate, filePath, len(match.Games), matchHash, canonicalHash)

		if err != nil {
			return 0, fmt.Errorf("failed to insert match: %w", err)
		}

		matchID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get match ID: %w", err)
		}

		// Auto-link tournament from event metadata
		eventName := strings.TrimSpace(match.Metadata.Event)
		if eventName != "" {
			var tournamentID int64
			err2 := tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, eventName).Scan(&tournamentID)
			if err2 != nil {
				// Tournament doesn't exist yet — create it
				res2, err3 := tx.Exec(`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, eventName)
				if err3 == nil {
					tournamentID, err = res2.LastInsertId()
					if err != nil {
						return 0, fmt.Errorf("failed to get last insert ID: %w", err)
					}
				}
			}
			if tournamentID > 0 {
				_, err = tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID)
				if err != nil {
					fmt.Printf("Warning: failed to link match to tournament: %v\n", err)
				}
			}
		}
	}

	// Build a position cache for deduplication
	positionCache := make(map[string]int64) // map[positionJSON]positionID
	semanticCache := make(map[string]int64) // map[semanticKey]positionID

	// Load existing positions into cache
	existingRows, err := tx.Query(`SELECT id, state FROM position`)
	if err != nil {
		return 0, fmt.Errorf("failed to load existing positions: %w", err)
	}

	for existingRows.Next() {
		var existingID int64
		var existingStateJSON string
		if err := existingRows.Scan(&existingID, &existingStateJSON); err != nil {
			continue
		}

		var existingPosition Position
		if err := json.Unmarshal([]byte(existingStateJSON), &existingPosition); err != nil {
			continue
		}

		// Normalize for comparison (positions are now stored normalized, but older ones might not be)
		normalizedPosition := existingPosition.NormalizeForStorage()
		normalizedPosition.ID = 0
		normalizedJSON, err := json.Marshal(normalizedPosition)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		positionCache[string(normalizedJSON)] = existingID
		semanticCache[positionSemanticKey(&normalizedPosition)] = existingID
	}
	if err := existingRows.Err(); err != nil {
		return 0, err
	}
	existingRows.Close()

	fmt.Printf("Loaded %d existing positions into cache\n", len(positionCache))

	if isCanonicalDuplicate {
		// Canonical duplicate: import analysis to existing positions, create genuinely new ones
		for _, game := range match.Games {
			for _, move := range game.Moves {
				if move.MoveType == "checker" && move.CheckerMove != nil {
					pos, err := d.createPositionFromXG(move.CheckerMove.Position, &game, int32(match.Metadata.MatchLength), 0, move.CheckerMove.ActivePlayer)
					if err != nil {
						continue
					}
					pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)
					pos.DecisionType = CheckerAction
					pos.Dice = [2]int{int(move.CheckerMove.Dice[0]), int(move.CheckerMove.Dice[1])}

					posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, positionCache, semanticCache)
					if err != nil {
						continue
					}

					// Save checker analysis to position
					if len(move.CheckerMove.Analysis) > 0 {
						err = d.saveCheckerAnalysisToPositionInTx(tx, posID, move.CheckerMove.Analysis,
							&move.CheckerMove.Position, move.CheckerMove.ActivePlayer, &move.CheckerMove.PlayedMove)
						if err != nil {
							fmt.Printf("Warning: failed to save analysis for canonical duplicate: %v\n", err)
						}
					}
				} else if move.MoveType == "cube" && move.CubeMove != nil && move.CubeMove.Analysis != nil {
					pos, err := d.createPositionFromXG(move.CubeMove.Position, &game, int32(match.Metadata.MatchLength), 0, move.CubeMove.ActivePlayer)
					if err != nil {
						continue
					}
					pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
					pos.DecisionType = CubeAction
					pos.Dice = [2]int{0, 0}

					posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, positionCache, semanticCache)
					if err != nil {
						continue
					}

					err = d.saveCubeAnalysisToPositionInTx(tx, posID, move.CubeMove.Analysis)
					if err != nil {
						fmt.Printf("Warning: failed to save cube analysis for canonical duplicate: %v\n", err)
					}
				}
			}
		}
	} else {
		// Normal import: create game/move records and import everything
		if err := d.importXGGamesAndMoves(tx, matchID, match, rawCubeInfo, positionCache); err != nil {
			return 0, err
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully imported match %d with %d games from %s\n", matchID, len(match.Games), filePath)
	return matchID, nil
}

// RawCubeAction stores raw cube action data from XG file
type RawCubeAction struct {
	Double   int32
	Take     int32
	ActiveP  int32
	CubeB    int32
	Position [26]int8
	Doubled  *xgparser.EngineStructDoubleAction // Full cube analysis data
}

// importXGGamesAndMoves imports all games, moves, and analysis from an XG match
func (d *Database) importXGGamesAndMoves(tx *sql.Tx, matchID int64, match *xgparser.Match, rawCubeInfo map[string]*RawCubeAction, positionCache map[string]int64) error {
	for gameIdx, game := range match.Games {
		gameID, err := d.importGame(tx, matchID, &game)
		if err != nil {
			return fmt.Errorf("failed to import game %d: %w", game.GameNumber, err)
		}

		cubeIdx := 0
		var lastCubeAnalysis *RawCubeAction

		for moveIdx, move := range game.Moves {
			var rawCube *RawCubeAction

			if move.MoveType == "cube" {
				key := fmt.Sprintf("%d_%d", gameIdx+1, cubeIdx)
				if rc, ok := rawCubeInfo[key]; ok {
					rawCube = rc
					lastCubeAnalysis = rc
				}
				cubeIdx++
			} else if move.MoveType == "checker" {
				rawCube = lastCubeAnalysis
				lastCubeAnalysis = nil
			}

			err := d.importMoveWithCacheAndRawCube(tx, gameID, int32(moveIdx), &move, &game, int32(match.Metadata.MatchLength), positionCache, rawCube)
			if err != nil {
				return fmt.Errorf("failed to import move %d in game %d: %w", moveIdx, game.GameNumber, err)
			}
		}
	}
	return nil
}

// importGame inserts a game and returns its ID
func (d *Database) importGame(tx *sql.Tx, matchID int64, game *xgparser.Game) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2,
		                  winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, matchID, game.GameNumber, game.InitialScore[0], game.InitialScore[1],
		game.Winner, game.PointsWon, len(game.Moves))

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// importMoveWithCache imports a move using a position cache for deduplication
func (d *Database) importMoveWithCache(tx *sql.Tx, gameID int64, moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, positionCache map[string]int64) error {
	var positionID int64
	var player int32
	var dice [2]int32
	var checkerMoveStr string
	var cubeActionStr string

	if move.MoveType == "checker" && move.CheckerMove != nil {
		// Create position from checker move
		pos, err := d.createPositionFromXG(move.CheckerMove.Position, game, matchLength, moveNumber, move.CheckerMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Set position-specific attributes from move
		// Convert XG player encoding (-1, 1) to blunderDB encoding (0, 1)
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{int(move.CheckerMove.Dice[0]), int(move.CheckerMove.Dice[1])}

		// Save position with cache
		posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CheckerMove.ActivePlayer
		dice = move.CheckerMove.Dice

		// Convert move notation with hit detection for consistency with analysis display
		checkerMoveStr = d.convertXGMoveToStringWithHits(move.CheckerMove.PlayedMove, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, player, dice[0], dice[1], checkerMoveStr)
		if err != nil {
			return err
		}

		moveID, err := moveResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		// Save analysis if available
		if len(move.CheckerMove.Analysis) > 0 {
			// Save to move_analysis table
			for _, analysis := range move.CheckerMove.Analysis {
				err = d.saveMoveAnalysisInTx(tx, moveID, &analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save checker analysis: %v\n", err)
				}
			}

			// Also save to position analysis table (for UI compatibility)
			err = d.saveCheckerAnalysisToPositionInTx(tx, positionID, move.CheckerMove.Analysis, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer, &move.CheckerMove.PlayedMove)
			if err != nil {
				fmt.Printf("Warning: failed to save position analysis: %v\n", err)
			}
		}

	} else if move.MoveType == "cube" && move.CubeMove != nil {
		// Create position from cube move
		pos, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Set position-specific attributes from move
		// Convert XG player encoding (-1, 1) to blunderDB encoding (0, 1)
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
		pos.DecisionType = CubeAction
		pos.Dice = [2]int{0, 0} // No dice for cube decisions

		// Save position with cache
		posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CubeMove.ActivePlayer
		cubeActionStr = d.convertCubeAction(move.CubeMove.CubeAction)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, cube_action)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "cube", positionID, player, 0, 0, cubeActionStr)
		if err != nil {
			return err
		}

		moveID, err := moveResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		// Save cube analysis if available
		if move.CubeMove.Analysis != nil {
			// Save to move_analysis table
			err = d.saveCubeAnalysisInTx(tx, moveID, move.CubeMove.Analysis)
			if err != nil {
				fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
			}

			// Also save to position analysis table (for UI compatibility)
			err = d.saveCubeAnalysisToPositionInTx(tx, positionID, move.CubeMove.Analysis)
			if err != nil {
				fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
			}
		}
	}

	return nil
}

// importMoveWithCacheAndRawCube imports a move with raw cube data for complete action info
func (d *Database) importMoveWithCacheAndRawCube(tx *sql.Tx, gameID int64, moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, positionCache map[string]int64, rawCube *RawCubeAction) error {
	var positionID int64
	var player int32
	var dice [2]int32
	var checkerMoveStr string
	var cubeActionStr string

	if move.MoveType == "checker" && move.CheckerMove != nil {
		// Create position from checker move
		pos, err := d.createPositionFromXG(move.CheckerMove.Position, game, matchLength, moveNumber, move.CheckerMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Set position-specific attributes from move
		// Convert XG player encoding (-1, 1) to blunderDB encoding (0, 1)
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{int(move.CheckerMove.Dice[0]), int(move.CheckerMove.Dice[1])}

		// Save position with cache
		posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CheckerMove.ActivePlayer
		dice = move.CheckerMove.Dice

		// Convert move notation with hit detection for consistency with analysis display
		checkerMoveStr = d.convertXGMoveToStringWithHits(move.CheckerMove.PlayedMove, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, player, dice[0], dice[1], checkerMoveStr)
		if err != nil {
			return err
		}

		moveID, err := moveResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		// Save analysis if available
		if len(move.CheckerMove.Analysis) > 0 {
			for _, analysis := range move.CheckerMove.Analysis {
				err = d.saveMoveAnalysisInTx(tx, moveID, &analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save checker analysis: %v\n", err)
				}
			}

			err = d.saveCheckerAnalysisToPositionInTx(tx, positionID, move.CheckerMove.Analysis, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer, &move.CheckerMove.PlayedMove)
			if err != nil {
				fmt.Printf("Warning: failed to save position analysis: %v\n", err)
			}
		}

		// Save cube analysis for this checker position (from the preceding CubeEntry)
		// This allows displaying cube info when pressing 'd' on a checker decision
		if rawCube != nil && rawCube.Doubled != nil {
			err = d.saveCubeAnalysisForCheckerPositionInTx(tx, positionID, rawCube)
			if err != nil {
				fmt.Printf("Warning: failed to save cube analysis for checker position: %v\n", err)
			}
		}

	} else if move.MoveType == "cube" && move.CubeMove != nil {
		// Handle explicit cube decisions
		// Only create positions for EXPLICIT cube actions (when a player actually doubles)
		// Skip implicit "No Double" decisions

		// Check if this is an explicit cube action
		isExplicitCubeAction := false
		if rawCube != nil {
			// Explicit action: Double was offered (Double == 1)
			isExplicitCubeAction = (rawCube.Double == 1)
		} else {
			// Fallback: check CubeAction field
			isExplicitCubeAction = (move.CubeMove.CubeAction != 0) // 0 = No Double
		}

		if !isExplicitCubeAction {
			// Skip implicit "No Double" decisions - don't create position
			return nil
		}

		if rawCube != nil && rawCube.Double == 1 && rawCube.Take == 1 {
			// DOUBLE/TAKE scenario: Create two positions

			// Position 1: Doubling decision (before the double)
			// The player on roll decides whether to double
			pos1, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
			if err != nil {
				return fmt.Errorf("failed to create doubling position: %w", err)
			}
			pos1.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
			pos1.DecisionType = CubeAction
			pos1.Dice = [2]int{0, 0}

			// Save position 1 (doubling decision)
			posID1, err := d.savePositionInTxWithCache(tx, pos1, positionCache)
			if err != nil {
				return fmt.Errorf("failed to save doubling position: %w", err)
			}

			player = move.CubeMove.ActivePlayer

			// Save move 1: Double
			moveResult1, err := tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", posID1, player, 0, 0, "Double")
			if err != nil {
				return err
			}
			moveID1, err := moveResult1.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}

			// Save analysis to first position
			if move.CubeMove.Analysis != nil {
				err = d.saveCubeAnalysisInTx(tx, moveID1, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
				}
				err = d.saveCubeAnalysisToPositionInTx(tx, posID1, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
				}
			}

			// Position 2: Take/Pass decision (after the double)
			// The opponent decides whether to take or pass
			// Note: We show the position BEFORE the take decision is executed,
			// so the cube is still in the center at doubled value.
			// The cube ownership will be reflected in the NEXT checker move position.
			pos2 := *pos1                           // Clone the position
			pos2.Cube.Value++                       // Double the cube (increment exponent: 0→1, 1→2, etc.)
			opponentPlayer := 1 - pos1.PlayerOnRoll // Opponent player (blunderDB encoding)
			pos2.PlayerOnRoll = opponentPlayer      // Opponent decides whether to take
			pos2.Cube.Owner = -1                    // Cube still in center (decision not yet executed)

			// Save position 2 (take decision)
			posID2, err := d.savePositionInTxWithCache(tx, &pos2, positionCache)
			if err != nil {
				return fmt.Errorf("failed to save take position: %w", err)
			}
			positionID = posID2 // Use second position as the reference

			// Convert opponent from blunderDB (0,1) back to XG encoding (1,-1) for move table
			opponentPlayerXG := int32(1)
			if opponentPlayer == 0 {
				opponentPlayerXG = 1
			} else {
				opponentPlayerXG = -1
			}

			// Save move 2: Take
			_, err = tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", posID2, opponentPlayerXG, 0, 0, "Take")
			if err != nil {
				return err
			}

		} else {
			// Single cube action (Double without Take, or other actions)
			pos, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
			if err != nil {
				return fmt.Errorf("failed to create position: %w", err)
			}

			pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
			pos.DecisionType = CubeAction
			pos.Dice = [2]int{0, 0}

			// Save position
			posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
			if err != nil {
				return fmt.Errorf("failed to save position: %w", err)
			}
			positionID = posID
			player = move.CubeMove.ActivePlayer

			// Determine cube action string
			if rawCube != nil {
				cubeActionStr = d.convertRawCubeAction(rawCube.Double, rawCube.Take)
			} else {
				cubeActionStr = d.convertCubeAction(move.CubeMove.CubeAction)
			}

			// Save move
			moveResult, err := tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", positionID, player, 0, 0, cubeActionStr)
			if err != nil {
				return err
			}

			moveID, err := moveResult.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}

			// Save cube analysis
			if move.CubeMove.Analysis != nil {
				err = d.saveCubeAnalysisInTx(tx, moveID, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
				}

				err = d.saveCubeAnalysisToPositionInTx(tx, positionID, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
				}
			}
		}
	}

	return nil
}

// convertRawCubeAction converts raw Double/Take values to action string
func (d *Database) convertRawCubeAction(double, take int32) string {
	// XG cube action encoding:
	// Double=0, Take=-1: No Double
	// Double=1, Take=1: Double, Take
	// Double=1, Take=-1: Double/Pass (opponent passed the double)
	// Double=-2: Initial position (should be filtered before)

	if double == 0 {
		return "No Double"
	} else if double == 1 {
		if take == 1 {
			return "Double/Take"
		} else {
			return "Double/Pass"
		}
	}
	return fmt.Sprintf("Unknown(D=%d,T=%d)", double, take)
}

// importMove imports a move, creates associated position, and stores analysis (deprecated - use importMoveWithCache)
func (d *Database) importMove(tx *sql.Tx, gameID int64, moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32) error {
	var positionID int64
	var player int32
	var dice [2]int32
	var checkerMoveStr string
	var cubeActionStr string

	if move.MoveType == "checker" && move.CheckerMove != nil {
		// Create position from checker move
		pos, err := d.createPositionFromXG(move.CheckerMove.Position, game, matchLength, moveNumber, move.CheckerMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Save position
		posID, err := d.savePositionInTx(tx, pos)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CheckerMove.ActivePlayer
		dice = move.CheckerMove.Dice

		// Convert move notation with hit detection for consistency with analysis display
		checkerMoveStr = d.convertXGMoveToStringWithHits(move.CheckerMove.PlayedMove, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, player, dice[0], dice[1], checkerMoveStr)
		if err != nil {
			return err
		}

		moveID, err := moveResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		// Save analysis if available
		if len(move.CheckerMove.Analysis) > 0 {
			for _, analysis := range move.CheckerMove.Analysis {
				err = d.saveMoveAnalysisInTx(tx, moveID, &analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save checker analysis: %v\n", err)
				}
			}
		}

	} else if move.MoveType == "cube" && move.CubeMove != nil {
		// Create position from cube move
		pos, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Save position
		posID, err := d.savePositionInTx(tx, pos)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CubeMove.ActivePlayer
		cubeActionStr = d.convertCubeAction(move.CubeMove.CubeAction)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, cube_action)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "cube", positionID, player, 0, 0, cubeActionStr)
		if err != nil {
			return err
		}

		moveID, err := moveResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		// Save cube analysis if available
		if move.CubeMove.Analysis != nil {
			err = d.saveCubeAnalysisInTx(tx, moveID, move.CubeMove.Analysis)
			if err != nil {
				fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
			}
		}
	}

	return nil
}

// createPositionFromXG converts xgparser.Position to blunderDB Position
// activePlayer indicates which XG player (-1 or 1) is on roll in this position
func (d *Database) createPositionFromXG(xgPos xgparser.Position, game *xgparser.Game, matchLength int32, moveNum int32, activePlayer int32) (*Position, error) {
	// Convert XG player encoding to blunderDB encoding
	// XG: -1 = Player 1, 1 = Player 2
	// blunderDB: 0 = Player 1, 1 = Player 2
	activePlayerBlunderDB := convertXGPlayerToBlunderDB(activePlayer)
	opponentPlayerBlunderDB := 1 - activePlayerBlunderDB

	// Calculate away scores from current scores
	// In blunderDB, scores are "points away from winning"
	// game.InitialScore contains current scores (e.g., 2-3 in a 7-point match)
	// We need to convert to away scores (e.g., 5-away, 4-away)
	awayScore1 := int(matchLength) - int(game.InitialScore[0])
	awayScore2 := int(matchLength) - int(game.InitialScore[1])

	// Handle unlimited match (matchLength == 0)
	if matchLength == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Map XG cube position to blunderDB format
	// XG CubePos is RELATIVE to active player:
	//   CubePos = 0: Center (no owner)
	//   CubePos = 1: Active player owns the cube
	//   CubePos = -1: Opponent owns the cube
	// blunderDB uses absolute encoding:
	//   -1 = center (no owner)
	//   0 = Player 1 owns (bottom, black)
	//   1 = Player 2 owns (top, white)
	cubeOwner := -1 // Default: center (no owner)
	if xgPos.CubePos == 1 {
		// Active player owns the cube
		cubeOwner = activePlayerBlunderDB
	} else if xgPos.CubePos == -1 {
		// Opponent owns the cube
		cubeOwner = opponentPlayerBlunderDB
	}
	// CubePos == 0 means center, cubeOwner stays -1

	// Convert cube value from XG (direct value 1,2,4,8...) to blunderDB (exponent 0,1,2,3...)
	// blunderDB displays 2^value, so we need log2(xgCube)
	cubeValue := 0
	if xgPos.Cube > 0 {
		// Calculate log2 of cube value
		for v := int(xgPos.Cube); v > 1; v >>= 1 {
			cubeValue++
		}
	}

	pos := &Position{
		PlayerOnRoll: 0, // Will be set from move context
		DecisionType: 0, // Checker decision by default
		Score:        [2]int{awayScore1, awayScore2},
		Cube: Cube{
			Value: cubeValue,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0}, // Will be set from move data
	}

	// Convert checkers from XG format to blunderDB format
	// XG format: index 0-23 are points 1-24, index 24=bar, 25=opponent bar
	// Positive values = active player's checkers, negative = opponent's checkers
	//
	// In blunderDB:
	// - Color 0 = Player 1 (always at bottom, black, moves 24→1) - NEVER CHANGES
	// - Color 1 = Player 2 (always at top, white, moves 1→24) - NEVER CHANGES
	// - Points 1-24 with indices 1-24 in the array
	// - Index 0 = Player 2's bar (white), Index 25 = Player 1's bar (black)
	//
	// XG stores positions from the active player's perspective:
	// - Positive checkers = active player's checkers
	// - Negative checkers = opponent's checkers
	// - Point numbering is from active player's perspective
	//
	// Player mapping:
	// - XG Player 0 = blunderDB Player 1 (Color 0, bottom, black)
	// - XG Player 1 = blunderDB Player 2 (Color 1, top, white)
	//
	// Strategy:
	// 1. Determine which player owns the checkers based on sign AND activePlayer
	// 2. Assign colors based on OWNER, not on sign
	// 3. Mirror positions if activePlayer == 1 (since XG encodes from active player's view)
	for i := 0; i < 26; i++ {
		checkerCount := xgPos.Checkers[i]

		// Determine WHERE to place them (calculate targetIndex for ALL points, even empty)
		// XG uses 1-based indexing: index 1-24 = points 1-24, index 0 = opponent bar, index 25 = active bar
		// blunderDB also uses same: index 1-24 = points 1-24, index 0 = P2 bar, index 25 = P1 bar
		targetIndex := i
		if activePlayerBlunderDB == 1 {
			// Player 2's perspective, need to mirror to Player 1's perspective
			if i >= 1 && i <= 24 {
				// XG index i = Player 2's point i → Player 1's point (25 - i) → same index (25 - i)
				targetIndex = 25 - i
			} else if i == 0 {
				// Opponent's bar from Player 2's view = Player 1's bar
				targetIndex = 25
			} else if i == 25 {
				// Active player's bar (Player 2) → Player 2's bar
				targetIndex = 0
			}
		} else {
			// Player 1's perspective - direct mapping (XG index = blunderDB index)
			// XG index 1-24 = Player 1's points 1-24 → blunderDB index 1-24 (same)
			// XG index 0 = opponent bar (Player 2) → blunderDB index 0 (Player 2's bar)
			// XG index 25 = active bar (Player 1) → blunderDB index 25 (Player 1's bar)
			// No transformation needed: targetIndex = i
		}

		if checkerCount == 0 {
			pos.Board.Points[targetIndex] = Point{Checkers: 0, Color: -1}
			continue
		}

		// Determine WHO owns these checkers
		var ownerColor int // 0=Player1(black), 1=Player2(white)
		if checkerCount > 0 {
			// Positive = active player
			ownerColor = activePlayerBlunderDB
		} else {
			// Negative = opponent
			ownerColor = opponentPlayerBlunderDB
		}

		// Place checkers with FIXED color based on owner
		pos.Board.Points[targetIndex] = Point{
			Checkers: int(abs(checkerCount)),
			Color:    ownerColor,
		}
	}

	// Calculate bearoff (checkers borne off)
	player1Total := 0
	player2Total := 0
	for i := 0; i < 26; i++ {
		if pos.Board.Points[i].Color == 0 {
			player1Total += pos.Board.Points[i].Checkers
		} else if pos.Board.Points[i].Color == 1 {
			player2Total += pos.Board.Points[i].Checkers
		}
	}
	pos.Board.Bearoff = [2]int{15 - player1Total, 15 - player2Total}

	return pos, nil
}

// Helper function for absolute value
func abs(x int8) int {
	if x < 0 {
		return int(-x)
	}
	return int(x)
}

// convertXGPlayerToBlunderDB converts XG player encoding to blunderDB encoding
// XG: -1 = Player 1, 1 = Player 2
// blunderDB: 0 = Player 1, 1 = Player 2
func convertXGPlayerToBlunderDB(xgPlayer int32) int {
	// XG uses 1 for first player, -1 for second player
	// blunderDB uses 0 for Player 1 (black, bottom), 1 for Player 2 (white, top)
	if xgPlayer == 1 {
		return 0 // Player 1
	}
	return 1 // Player 2
}

// convertBlunderDBPlayerToXG converts blunderDB player encoding (0/1) to XG encoding (1/-1)
// This is used when storing GnuBG-imported moves in the DB, so that GetMatchMovePositions
// can apply convertXGPlayerToBlunderDB uniformly for both XG and GnuBG imports.
func convertBlunderDBPlayerToXG(blunderDBPlayer int) int32 {
	if blunderDBPlayer == 0 {
		return 1 // Player 1
	}
	return -1 // Player 2
}

// savePositionInTxWithCache saves a position within a transaction using a cache for deduplication
// Positions are normalized before storage (player_on_roll = 0) to prevent storing duplicates.
func (d *Database) savePositionInTxWithCache(tx *sql.Tx, position *Position, positionCache map[string]int64) (int64, error) {
	// Normalize position for storage - always store from player on roll's perspective (player_on_roll = 0)
	normalizedPosition := position.NormalizeForStorage()
	normalizedPosition.ID = 0 // Exclude ID for comparison

	positionJSON, err := json.Marshal(normalizedPosition)
	if err != nil {
		return 0, err
	}

	// Check cache first
	if existingID, exists := positionCache[string(positionJSON)]; exists {
		return existingID, nil
	}

	// Position doesn't exist, create new one
	result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
	if err != nil {
		return 0, err
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Update position with ID
	normalizedPosition.ID = positionID
	positionJSONWithID, err := json.Marshal(normalizedPosition)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSONWithID), positionID)
	if err != nil {
		return 0, err
	}

	// Add to cache for future lookups
	positionCache[string(positionJSON)] = positionID

	return positionID, nil
}

// positionSemanticKey generates a cache key based on the fields that define position identity:
// board checker layout, cube state, score, player on roll, decision type, and dice.
// This enables matching positions across different format parsers that may produce
// different metadata (HasJacoby, HasBeaver) for the same physical board state.
func positionSemanticKey(pos *Position) string {
	return fmt.Sprintf("%v|%v|%v|%d|%d|%v",
		pos.Board, pos.Cube, pos.Score,
		pos.PlayerOnRoll, pos.DecisionType, pos.Dice)
}

// findOrCreatePositionForCanonicalDuplicate looks up a position by semantic key during
// canonical duplicate imports. If the position already exists (by board, cube, score,
// player on roll, decision type, dice), returns the existing ID. If it's genuinely new,
// creates it in the database and returns the new ID.
func (d *Database) findOrCreatePositionForCanonicalDuplicate(tx *sql.Tx, position *Position, positionCache map[string]int64, semanticCache map[string]int64) (int64, error) {
	normalizedPosition := position.NormalizeForStorage()
	normalizedPosition.ID = 0

	// Check exact match first (fast path)
	positionJSON, err := json.Marshal(normalizedPosition)
	if err != nil {
		return 0, err
	}
	if existingID, exists := positionCache[string(positionJSON)]; exists {
		return existingID, nil
	}

	// Check semantic match (board, cube, score, player_on_roll, decision_type, dice)
	semKey := positionSemanticKey(&normalizedPosition)
	if existingID, exists := semanticCache[semKey]; exists {
		return existingID, nil
	}

	// Genuinely new position — create it
	result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
	if err != nil {
		return 0, err
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	normalizedPosition.ID = positionID
	positionJSONWithID, err := json.Marshal(normalizedPosition)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSONWithID), positionID)
	if err != nil {
		return 0, err
	}

	// Add to both caches
	positionCache[string(positionJSON)] = positionID
	semanticCache[semKey] = positionID

	return positionID, nil
}

// savePositionInTx saves a position within a transaction, checking for duplicates first
// Positions are normalized before storage (player_on_roll = 0) to prevent storing duplicates.
func (d *Database) savePositionInTx(tx *sql.Tx, position *Position) (int64, error) {
	// Normalize position for storage - always store from player on roll's perspective (player_on_roll = 0)
	normalizedPosition := position.NormalizeForStorage()
	normalizedPosition.ID = 0 // Exclude ID for comparison

	positionJSON, err := json.Marshal(normalizedPosition)
	if err != nil {
		return 0, err
	}

	// Query existing positions to check for duplicates
	rows, err := tx.Query(`SELECT id, state FROM position`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var stateJSON string
		var positionID int64
		if err = rows.Scan(&positionID, &stateJSON); err != nil {
			continue
		}

		var existingPosition Position
		if err = json.Unmarshal([]byte(stateJSON), &existingPosition); err != nil {
			continue
		}

		// Compare positions excluding the ID field
		existingPosition.ID = 0
		existingPositionJSON, err := json.Marshal(existingPosition)
		if err != nil {
			continue
		}

		if string(positionJSON) == string(existingPositionJSON) {
			// Position already exists, return existing ID
			return positionID, nil
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	// Position doesn't exist, create new one
	result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
	if err != nil {
		return 0, err
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Update position with ID
	normalizedPosition.ID = positionID
	positionJSON, err = json.Marshal(normalizedPosition)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSON), positionID)

	return positionID, err
}

// saveMoveAnalysisInTx saves checker move analysis within a transaction
func (d *Database) saveMoveAnalysisInTx(tx *sql.Tx, moveID int64, analysis *xgparser.CheckerAnalysis) error {
	// Calculate win rates (player1 is player on roll)
	player1WinRate := analysis.Player1WinRate * 100.0 // Convert to percentage
	player2WinRate := (1.0 - analysis.Player1WinRate) * 100.0

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "checker", translateAnalysisDepth(int(analysis.AnalysisDepth)),
		analysis.Equity, 0.0, // No separate equity error in CheckerAnalysis
		player1WinRate, analysis.Player1GammonRate*100.0, analysis.Player1BgRate*100.0,
		player2WinRate, analysis.Player2GammonRate*100.0, analysis.Player2BgRate*100.0)

	return err
}

// saveCubeAnalysisInTx saves cube analysis within a transaction
func (d *Database) saveCubeAnalysisInTx(tx *sql.Tx, moveID int64, analysis *xgparser.CubeAnalysis) error {
	// Calculate win rates
	player1WinRate := analysis.Player1WinRate * 100.0
	player2WinRate := (1.0 - analysis.Player1WinRate) * 100.0

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "cube", translateAnalysisDepth(int(analysis.AnalysisDepth)),
		analysis.CubefulNoDouble, 0.0,
		player1WinRate, analysis.Player1GammonRate*100.0, analysis.Player1BgRate*100.0,
		player2WinRate, analysis.Player2GammonRate*100.0, analysis.Player2BgRate*100.0)

	return err
}

// saveCheckerAnalysisToPositionInTx converts XG checker analysis to PositionAnalysis and saves it
// playedMove is optional - if provided, it will be used as the source of truth for the first analysis
// (workaround for xgparser bug where analysis.Move may be incomplete for multi-submove bear-offs)
func (d *Database) saveCheckerAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analyses []xgparser.CheckerAnalysis, initialPosition *xgparser.Position, activePlayer int32, playedMove *[8]int32) error {
	if len(analyses) == 0 {
		return nil
	}

	// Create PositionAnalysis structure
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	// Build checker analysis with all moves
	checkerMoves := make([]CheckerMove, 0, len(analyses))
	for i, analysis := range analyses {
		// Convert move from [8]int8 to [8]int32 for convertXGMoveToString
		var move [8]int32

		// For the first analysis (i=0), use playedMove if available
		// This is a workaround for xgparser bug where analysis.Move may be incomplete
		// for multi-submove bear-offs or other complex moves
		if i == 0 && playedMove != nil {
			// Check if playedMove has more info than analysis.Move
			playedMoveCount := 0
			analysisMoveCount := 0
			for j := 0; j < 8; j += 2 {
				if (*playedMove)[j] != -1 {
					playedMoveCount++
				}
				if analysis.Move[j] != -1 {
					analysisMoveCount++
				}
			}
			// Use playedMove if it has more sub-moves
			if playedMoveCount > analysisMoveCount {
				for j := 0; j < 8; j++ {
					move[j] = (*playedMove)[j]
				}
			} else {
				for j := 0; j < 8; j++ {
					move[j] = int32(analysis.Move[j])
				}
			}
		} else {
			for j := 0; j < 8; j++ {
				move[j] = int32(analysis.Move[j])
			}
		}
		// Infer multipliers from position changes
		// XG stores moves compactly - e.g., 1/off(4) is stored as just 1/off once
		if initialPosition != nil {
			move = inferMoveMultipliers(move, initialPosition, &analysis.Position, activePlayer)
		}

		// Use move string with hit detection if initial position is available
		var moveStr string
		if initialPosition != nil {
			moveStr = d.convertXGMoveToStringWithHits(move, initialPosition, activePlayer)
		} else {
			moveStr = d.convertXGMoveToString(move, activePlayer)
		}

		var equityError *float64
		if i > 0 {
			diff := float64(analyses[0].Equity - analysis.Equity)
			equityError = &diff
		}

		checkerMove := CheckerMove{
			Index:                    i,
			AnalysisDepth:            translateAnalysisDepth(int(analysis.AnalysisDepth)),
			AnalysisEngine:           "XG",
			Move:                     moveStr,
			Equity:                   float64(analysis.Equity),
			EquityError:              equityError,
			PlayerWinChance:          float64(analysis.Player1WinRate) * 100.0,
			PlayerGammonChance:       float64(analysis.Player1GammonRate) * 100.0,
			PlayerBackgammonChance:   float64(analysis.Player1BgRate) * 100.0,
			OpponentWinChance:        float64(1.0-analysis.Player1WinRate) * 100.0,
			OpponentGammonChance:     float64(analysis.Player2GammonRate) * 100.0,
			OpponentBackgammonChance: float64(analysis.Player2BgRate) * 100.0,
		}
		checkerMoves = append(checkerMoves, checkerMove)
	}

	posAnalysis.CheckerAnalysis = &CheckerAnalysis{
		Moves: checkerMoves,
	}

	// Save to analysis table
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveCubeAnalysisToPositionInTx converts XG cube analysis to PositionAnalysis and saves it
func (d *Database) saveCubeAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analysis *xgparser.CubeAnalysis) error {
	if analysis == nil {
		return nil
	}

	// Create PositionAnalysis structure
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubefulNoDouble := float64(analysis.CubefulNoDouble)
	cubefulDoubleTake := float64(analysis.CubefulDoubleTake)
	cubefulDoublePass := float64(analysis.CubefulDoublePass)

	// Calculate best equity considering opponent's optimal response
	// If player doubles, opponent will choose the action that minimizes player's equity
	// So effective double equity = min(DoubleTake, DoublePass)
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	// Player's best achievable equity = max(NoDouble, effectiveDoubleEquity)
	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		// Best action is "Double, Take" or "Double, Pass" depending on opponent's response
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	// Build doubling cube analysis
	// Error is negative when this decision loses equity vs best (current - best)
	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateAnalysisDepth(int(analysis.AnalysisDepth)),
		AnalysisEngine:            "XG",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BgRate) * 100.0,
		OpponentWinChances:        float64(1.0-analysis.Player1WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BgRate) * 100.0,
		CubelessNoDoubleEquity:    float64(analysis.CubelessNoDouble),
		CubelessDoubleEquity:      float64(analysis.CubelessDouble),
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
		WrongTakePercentage:       0.0, // XG provides WrongPassTakePercent which covers both
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Save to analysis table
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveCubeAnalysisForCheckerPositionInTx saves cube analysis from a RawCubeAction to a checker position
// This is used to attach the cube decision analysis to checker moves (from the preceding CubeEntry)
// It merges the cube info with existing checker analysis if present
func (d *Database) saveCubeAnalysisForCheckerPositionInTx(tx *sql.Tx, positionID int64, rawCube *RawCubeAction) error {
	if rawCube == nil || rawCube.Doubled == nil {
		return nil
	}

	doubled := rawCube.Doubled

	// Try to load existing analysis for this position
	var existingAnalysisJSON string
	var existingID int64
	err := tx.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisJSON)

	var posAnalysis PositionAnalysis
	if err == nil && existingID > 0 {
		// Existing analysis found - merge cube info with it
		err = json.Unmarshal([]byte(existingAnalysisJSON), &posAnalysis)
		if err != nil {
			return err
		}
	} else {
		// No existing analysis - create new one
		posAnalysis = PositionAnalysis{
			PositionID:            int(positionID),
			AnalysisType:          "CheckerMove",
			AnalysisEngineVersion: "XG",
			CreationDate:          time.Now(),
		}
	}

	posAnalysis.LastModifiedDate = time.Now()

	// Extract data from EngineStructDoubleAction
	// XG Eval array mapping (from xgparser convertCubeEntry):
	// Eval[0] = opponent's backgammon rate
	// Eval[1] = opponent's gammon rate
	// Eval[2] = opponent's win rate (so player on roll's win rate = 1 - Eval[2])
	// Eval[4] = player on roll's gammon rate
	// Eval[5] = player on roll's backgammon rate
	// Eval[6] = cubeless equity
	cubefulNoDouble := float64(doubled.EquB)
	cubefulDoubleTake := float64(doubled.EquDouble)
	cubefulDoublePass := float64(doubled.EquDrop)

	// Win/gammon/bg rates from player on roll's perspective
	playerWin := (1.0 - float64(doubled.Eval[2])) * 100.0
	playerGammon := float64(doubled.Eval[4]) * 100.0
	playerBg := float64(doubled.Eval[5]) * 100.0
	opponentWin := float64(doubled.Eval[2]) * 100.0
	opponentGammon := float64(doubled.Eval[1]) * 100.0
	opponentBg := float64(doubled.Eval[0]) * 100.0

	// Calculate best equity considering opponent's optimal response
	// If player doubles, opponent will choose the action that minimizes player's equity
	// So effective double equity = min(DoubleTake, DoublePass)
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	// Player's best achievable equity = max(NoDouble, effectiveDoubleEquity)
	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		// Best action is "Double, Take" or "Double, Pass" depending on opponent's response
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	// Build doubling cube analysis
	// Error is negative when this decision loses equity vs best (current - best)
	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateAnalysisDepth(int(doubled.Level)),
		AnalysisEngine:            "XG",
		PlayerWinChances:          playerWin,
		PlayerGammonChances:       playerGammon,
		PlayerBackgammonChances:   playerBg,
		OpponentWinChances:        opponentWin,
		OpponentGammonChances:     opponentGammon,
		OpponentBackgammonChances: opponentBg,
		CubelessNoDoubleEquity:    float64(doubled.Eval[6]),
		CubelessDoubleEquity:      float64(doubled.Eval[6]), // Same as no double for cubeless
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       0.0, // Not available from EngineStructDoubleAction
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Save to analysis table (will update if exists, insert if not)
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// determineBestCubeAction determines the best cube action from analysis
// Takes into account that opponent will play optimally (choose min equity for the player)
func (d *Database) determineBestCubeAction(analysis *xgparser.CubeAnalysis) string {
	cubefulNoDouble := analysis.CubefulNoDouble
	cubefulDoubleTake := analysis.CubefulDoubleTake
	cubefulDoublePass := analysis.CubefulDoublePass

	// If player doubles, opponent will choose the action that minimizes player's equity
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	// Player's best choice is max(NoDouble, effectiveDoubleEquity)
	if effectiveDoubleEquity > cubefulNoDouble {
		if cubefulDoubleTake <= cubefulDoublePass {
			return "Double, Take"
		}
		return "Double, Pass"
	}
	return "No Double"
}

// enginePriority returns a sort priority for analysis engines.
// XG gets priority 0 (first), GNUbg gets 1, unknown/empty gets 2.
func enginePriority(engine string) int {
	switch strings.ToLower(engine) {
	case "xg":
		return 0
	case "gnubg":
		return 1
	default:
		return 2
	}
}

// sortCubeAnalysesByEngine sorts cube analyses so XG comes first, then GNUbg, then others.
func sortCubeAnalysesByEngine(analyses []DoublingCubeAnalysis) {
	sort.SliceStable(analyses, func(i, j int) bool {
		return enginePriority(analyses[i].AnalysisEngine) < enginePriority(analyses[j].AnalysisEngine)
	})
}

// mergeCheckerMoves merges two sets of checker moves, avoiding duplicates
// Moves are considered duplicates if they have the same move string
// Returns moves sorted by equity (highest first), with XG engine preferred as tiebreaker
func mergeCheckerMoves(existing, incoming []CheckerMove) []CheckerMove {
	// Use a map to track unique moves by their move string
	moveMap := make(map[string]CheckerMove)

	// Add existing moves to the map
	for _, m := range existing {
		moveMap[m.Move] = m
	}

	// Add incoming moves, prefer incoming if there's a conflict (newer analysis)
	for _, m := range incoming {
		if existingMove, exists := moveMap[m.Move]; exists {
			// If the incoming move has the same depth or higher quality analysis, use it
			// Otherwise keep the existing one
			if m.AnalysisDepth >= existingMove.AnalysisDepth {
				moveMap[m.Move] = m
			}
		} else {
			moveMap[m.Move] = m
		}
	}

	// Convert map to slice
	result := make([]CheckerMove, 0, len(moveMap))
	for _, m := range moveMap {
		result = append(result, m)
	}

	// Sort by equity (highest first), with XG engine preferred as tiebreaker
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Equity != result[j].Equity {
			return result[i].Equity > result[j].Equity
		}
		return enginePriority(result[i].AnalysisEngine) < enginePriority(result[j].AnalysisEngine)
	})

	// Recalculate equity errors relative to the best move
	if len(result) > 0 {
		bestEquity := result[0].Equity
		for i := range result {
			result[i].Index = i
			if i == 0 {
				result[i].EquityError = nil
			} else {
				diff := bestEquity - result[i].Equity
				result[i].EquityError = &diff
			}
		}
	}

	return result
}

// mergePlayedMoves merges played moves/cube actions, avoiding duplicates
func mergePlayedMoves(existing, incoming []string) []string {
	// Use a map to track unique moves
	moveSet := make(map[string]bool)

	for _, m := range existing {
		if m != "" {
			moveSet[normalizeMove(m)] = true
		}
	}

	for _, m := range incoming {
		if m != "" {
			moveSet[normalizeMove(m)] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(moveSet))
	for m := range moveSet {
		result = append(result, m)
	}

	sort.Strings(result)
	return result
}

// normalizeMove normalizes a move string for comparison
// "5/2 5/4" and "5/4 5/2" are the same move but in different order
func normalizeMove(move string) string {
	parts := strings.Fields(move)
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

// saveAnalysisInTx saves a PositionAnalysis within a transaction, merging with existing analysis if present
func (d *Database) saveAnalysisInTx(tx *sql.Tx, positionID int64, analysis PositionAnalysis) error {
	// Ensure the positionID is set in the analysis
	analysis.PositionID = int(positionID)

	// Update last modified date
	analysis.LastModifiedDate = time.Now()

	// Check if an analysis already exists for the given position ID
	var existingID int64
	var existingAnalysisJSON string
	err := tx.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisJSON)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existingID > 0 {
		// Parse existing analysis
		var existingAnalysis PositionAnalysis
		err = json.Unmarshal([]byte(existingAnalysisJSON), &existingAnalysis)
		if err != nil {
			return err
		}

		// Preserve the existing creation date
		analysis.CreationDate = existingAnalysis.CreationDate

		// Merge checker analysis if both exist
		if existingAnalysis.CheckerAnalysis != nil && analysis.CheckerAnalysis != nil {
			analysis.CheckerAnalysis.Moves = mergeCheckerMoves(
				existingAnalysis.CheckerAnalysis.Moves,
				analysis.CheckerAnalysis.Moves,
			)
		} else if existingAnalysis.CheckerAnalysis != nil && analysis.CheckerAnalysis == nil {
			// Keep existing checker analysis if new one is nil
			analysis.CheckerAnalysis = existingAnalysis.CheckerAnalysis
		}

		// Merge doubling cube analysis - keep all engine analyses in AllCubeAnalyses
		if existingAnalysis.DoublingCubeAnalysis != nil && analysis.DoublingCubeAnalysis != nil {
			// Both have cube analysis - check if they're from different engines
			existingEngine := existingAnalysis.DoublingCubeAnalysis.AnalysisEngine
			incomingEngine := analysis.DoublingCubeAnalysis.AnalysisEngine

			if existingEngine != incomingEngine && existingEngine != "" && incomingEngine != "" {
				// Different engines: build AllCubeAnalyses array with both
				allCube := make([]DoublingCubeAnalysis, 0)
				// Add existing engine's cube analyses
				if len(existingAnalysis.AllCubeAnalyses) > 0 {
					allCube = append(allCube, existingAnalysis.AllCubeAnalyses...)
				} else {
					allCube = append(allCube, *existingAnalysis.DoublingCubeAnalysis)
				}
				// Add incoming if not already present for this engine
				hasIncoming := false
				for _, ca := range allCube {
					if ca.AnalysisEngine == incomingEngine {
						hasIncoming = true
						break
					}
				}
				if !hasIncoming {
					allCube = append(allCube, *analysis.DoublingCubeAnalysis)
				}
				sortCubeAnalysesByEngine(allCube)
				analysis.AllCubeAnalyses = allCube
				// Primary DoublingCubeAnalysis stays as the incoming one
			} else {
				// Same engine or empty engine - keep incoming, preserve AllCubeAnalyses
				if len(existingAnalysis.AllCubeAnalyses) > 0 {
					analysis.AllCubeAnalyses = existingAnalysis.AllCubeAnalyses
				}
			}
		} else if existingAnalysis.DoublingCubeAnalysis != nil && analysis.DoublingCubeAnalysis == nil {
			analysis.DoublingCubeAnalysis = existingAnalysis.DoublingCubeAnalysis
			analysis.AllCubeAnalyses = existingAnalysis.AllCubeAnalyses
		}

		// Merge played moves (support both old single field and new array field)
		existingPlayedMoves := existingAnalysis.PlayedMoves
		if existingAnalysis.PlayedMove != "" && len(existingPlayedMoves) == 0 {
			existingPlayedMoves = []string{existingAnalysis.PlayedMove}
		}
		incomingPlayedMoves := analysis.PlayedMoves
		if analysis.PlayedMove != "" && len(incomingPlayedMoves) == 0 {
			incomingPlayedMoves = []string{analysis.PlayedMove}
		}
		analysis.PlayedMoves = mergePlayedMoves(existingPlayedMoves, incomingPlayedMoves)

		// Merge played cube actions
		existingCubeActions := existingAnalysis.PlayedCubeActions
		if existingAnalysis.PlayedCubeAction != "" && len(existingCubeActions) == 0 {
			existingCubeActions = []string{existingAnalysis.PlayedCubeAction}
		}
		incomingCubeActions := analysis.PlayedCubeActions
		if analysis.PlayedCubeAction != "" && len(incomingCubeActions) == 0 {
			incomingCubeActions = []string{analysis.PlayedCubeAction}
		}
		analysis.PlayedCubeActions = mergePlayedMoves(existingCubeActions, incomingCubeActions)

		// Clear deprecated single fields after merging
		analysis.PlayedMove = ""
		analysis.PlayedCubeAction = ""

		// Sort checker moves by equity after merging
		if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
			sort.Slice(analysis.CheckerAnalysis.Moves, func(i, j int) bool {
				return analysis.CheckerAnalysis.Moves[i].Equity > analysis.CheckerAnalysis.Moves[j].Equity
			})
			// Recalculate indices and equity errors
			bestEquity := analysis.CheckerAnalysis.Moves[0].Equity
			for i := range analysis.CheckerAnalysis.Moves {
				analysis.CheckerAnalysis.Moves[i].Index = i
				if i == 0 {
					analysis.CheckerAnalysis.Moves[i].EquityError = nil
				} else {
					diff := bestEquity - analysis.CheckerAnalysis.Moves[i].Equity
					analysis.CheckerAnalysis.Moves[i].EquityError = &diff
				}
			}
		}

		// Update the existing analysis
		analysisJSON, err := json.Marshal(analysis)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE analysis SET data = ? WHERE id = ?`, string(analysisJSON), existingID)
		if err != nil {
			return err
		}
	} else {
		// Set creation date if not already set
		if analysis.CreationDate.IsZero() {
			analysis.CreationDate = time.Now()
		}

		// Convert single played move to array if needed
		if analysis.PlayedMove != "" && len(analysis.PlayedMoves) == 0 {
			analysis.PlayedMoves = []string{analysis.PlayedMove}
			analysis.PlayedMove = ""
		}
		if analysis.PlayedCubeAction != "" && len(analysis.PlayedCubeActions) == 0 {
			analysis.PlayedCubeActions = []string{analysis.PlayedCubeAction}
			analysis.PlayedCubeAction = ""
		}

		// Sort checker moves by equity
		if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
			sort.Slice(analysis.CheckerAnalysis.Moves, func(i, j int) bool {
				return analysis.CheckerAnalysis.Moves[i].Equity > analysis.CheckerAnalysis.Moves[j].Equity
			})
			// Recalculate indices and equity errors
			bestEquity := analysis.CheckerAnalysis.Moves[0].Equity
			for i := range analysis.CheckerAnalysis.Moves {
				analysis.CheckerAnalysis.Moves[i].Index = i
				if i == 0 {
					analysis.CheckerAnalysis.Moves[i].EquityError = nil
				} else {
					diff := bestEquity - analysis.CheckerAnalysis.Moves[i].Equity
					analysis.CheckerAnalysis.Moves[i].EquityError = &diff
				}
			}
		}

		// Insert a new analysis
		analysisJSON, err := json.Marshal(analysis)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, string(analysisJSON))
		if err != nil {
			return err
		}
	}

	return nil
}

// inferMoveMultipliers analyzes a partial move array and the position difference
// to infer the correct number of repetitions for each move.
// XG sometimes stores only one instance of a move even when multiple checkers make the same move.
// This function expands the move array to include all repetitions.
// Returns the expanded move array with correct multipliers.
func inferMoveMultipliers(partialMove [8]int32, initialPos, finalPos *xgparser.Position, activePlayer int32) [8]int32 {
	if initialPos == nil || finalPos == nil {
		return partialMove
	}

	// First, count how many moves are explicitly in the input
	// and count occurrences of each unique (from,to) pair
	type moveSpec struct {
		from int32
		to   int32
	}
	moveCount := make(map[moveSpec]int32)
	totalInputMoves := 0

	for i := 0; i < 8; i += 2 {
		from := partialMove[i]
		to := partialMove[i+1]
		if from == -1 {
			break
		}
		// Handle implicit bear-off
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		moveCount[moveSpec{from, to}]++
		totalInputMoves++
	}

	if totalInputMoves == 0 {
		return partialMove
	}

	// If we already have multiple moves in input (not a compact representation),
	// just return the input as-is - no inference needed
	// XG uses compact representation only for doublets where same move repeats
	if totalInputMoves > 1 {
		// Check if all moves are the same (compact doublet notation)
		allSame := true
		var firstMove moveSpec
		first := true
		for ms := range moveCount {
			if first {
				firstMove = ms
				first = false
			} else if ms != firstMove {
				allSame = false
				break
			}
		}
		// If moves are different, no inference needed - return as-is
		if !allSame {
			return partialMove
		}
	}

	// At this point, either:
	// 1. We have a single move that might need expansion (e.g., [8,3,-1,-1,-1,-1,-1,-1] for 8/3(4))
	// 2. We have multiple identical moves (already explicit, no expansion needed)

	// Get the unique moves preserving order
	var uniqueMoves []moveSpec
	seen := make(map[moveSpec]bool)
	for i := 0; i < 8; i += 2 {
		from := partialMove[i]
		to := partialMove[i+1]
		if from == -1 {
			break
		}
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		ms := moveSpec{from, to}
		if !seen[ms] {
			seen[ms] = true
			uniqueMoves = append(uniqueMoves, ms)
		}
	}

	// If we have multiple identical moves in input, they're already explicit
	if len(uniqueMoves) == 1 && moveCount[uniqueMoves[0]] > 1 {
		return partialMove
	}

	// Calculate how many checkers left each source point
	// netChange[point] = initialCheckers - finalCheckers (positive = net loss of checkers)
	netChange := make(map[int32]int32)

	for _, ms := range uniqueMoves {
		if ms.from == 25 {
			netChange[25] = int32(initialPos.Checkers[25]) - int32(finalPos.Checkers[25])
		} else if ms.from >= 1 && ms.from <= 24 {
			netChange[ms.from] = int32(initialPos.Checkers[ms.from]) - int32(finalPos.Checkers[ms.from])
		}
		// Track destination changes too
		if ms.to >= 1 && ms.to <= 24 {
			netChange[ms.to] = int32(initialPos.Checkers[ms.to]) - int32(finalPos.Checkers[ms.to])
		}
	}

	// Build a flow model: how many checkers move from each source
	// Track arriving checkers for intermediate points
	arriving := make(map[int32]int32) // checkers arriving at each point

	var expandedMove [8]int32
	for i := range expandedMove {
		expandedMove[i] = -1
	}
	moveIndex := 0

	// Process moves in order
	for _, ms := range uniqueMoves {
		var count int32 = 1

		if ms.to == -2 {
			// Bear-off: count = net checkers that left this point + checkers arriving from other points
			count = netChange[ms.from] + arriving[ms.from]
		} else if ms.to >= 1 && ms.to <= 24 {
			// Move to board point
			// Count = how many checkers left the source AND didn't come back
			// For most cases, this is simply the net change of the source
			// But we need to also account for checker flow

			// Net change at source tells us how many checkers total left
			srcLoss := netChange[ms.from]

			// If source also receives checkers (from another move), we need less moves
			srcReceive := arriving[ms.from]

			// The move count is the source loss minus any checkers that arrived
			count = srcLoss - srcReceive
			if count <= 0 {
				count = 1
			}

			// After this move, these checkers arrive at the destination
			arriving[ms.to] += count
		}

		// Cap at 4 (maximum for doubles) or remaining slots
		maxMoves := int32(4)
		remainingSlots := int32((8 - moveIndex) / 2)
		if count > maxMoves {
			count = maxMoves
		}
		if count > remainingSlots {
			count = remainingSlots
		}
		if count < 1 {
			count = 1
		}

		for j := int32(0); j < count && moveIndex < 8; j++ {
			expandedMove[moveIndex] = ms.from
			expandedMove[moveIndex+1] = ms.to
			moveIndex += 2
		}
	}

	return expandedMove
}

// translateAnalysisDepth converts XG analysis depth codes to human-readable strings
// XG depth codes:
//   - 0-9 = (N+1)-ply search depth (XG stores ply-1 internally)
//   - 998-1000 = Book moves (different footnote references)
//   - 1001 = XG Roller (neural network)
//   - 1002 = XG Roller++ (extended neural network analysis)
func translateAnalysisDepth(depth int) string {
	switch {
	case depth >= 0 && depth <= 9:
		return fmt.Sprintf("%d-ply", depth+1)
	case depth >= 998 && depth <= 1000:
		return "Book"
	case depth == 1001:
		return "XG Roller"
	case depth == 1002:
		return "XG Roller++"
	default:
		return fmt.Sprintf("%d", depth)
	}
}

// convertXGMoveToString converts XG move format to readable string
// XG move format: [from, to, from, to, ...] where:
//   - 1-24 are board points (from active player's perspective)
//   - 25 is the bar
//   - -2 is bear off
//   - -1 is unused/end of move
//
// Moves in XG files and analysis are stored from the player on roll's perspective.
// They should be displayed as-is without mirroring, per standard backgammon notation.
func (d *Database) convertXGMoveToString(playedMove [8]int32, activePlayer int32) string {
	// Note: activePlayer is kept for API compatibility but moves don't need transformation
	// Moves are always from the roller's perspective (24 = furthest from home, 1 = closest)
	_ = activePlayer // unused but kept for signature compatibility

	// Parse raw moves into from/to pairs
	var fromPts []int32
	var toPts []int32
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		// Check for end of move marker (-1 when from is also -1)
		if from == -1 {
			break
		}
		// Handle implicit bear-off: XG sometimes encodes bear-off as to=-1 or to<=0
		// when the calculated destination (from - die) would be <= 0
		// This happens when bearing off from the home board (points 1-6)
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2 // Convert to explicit bear-off
		}
		// Skip invalid point values (should be 1-24 for points, 25 for bar, -2 for bear off)
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}
		fromPts = append(fromPts, from)
		toPts = append(toPts, to)
	}

	if len(fromPts) == 0 {
		return "Cannot Move"
	}

	// Try to merge consecutive checker slides (same checker moving through points)
	// E.g., 24/23 23/22 22/21 21/20 -> 24/20 when dice are all same
	fromPts, toPts = d.mergeSlides(fromPts, toPts)

	// Create sortable items
	type moveItem struct {
		from int32
		to   int32
	}
	items := make([]moveItem, len(fromPts))
	for i := range fromPts {
		items[i] = moveItem{from: fromPts[i], to: toPts[i]}
	}

	// Sort moves by 'from' point descending (standard backgammon notation)
	// bar (25) comes first, then higher points before lower points
	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	// Format each move as string
	formatPoint := func(p int32) string {
		if p == 25 {
			return "bar"
		} else if p == -2 {
			return "off"
		} else if p >= 1 && p <= 24 {
			return fmt.Sprintf("%d", p)
		}
		return fmt.Sprintf("?%d", p)
	}

	// Build move string, grouping identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		// Count consecutive identical moves
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
			} else {
				break
			}
		}
		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s(%d)", formatPoint(item.from), formatPoint(item.to), count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s", formatPoint(item.from), formatPoint(item.to)))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// convertXGMoveToStringWithHits converts XG move format to readable string with hit indicators (*)
// It uses the initial position to detect when a blot is hit
// Moves are displayed from the player on roll's perspective (standard notation).
// activePlayer: XG encoding: 1 = Player 1 (X), -1 = Player 2 (O)
func (d *Database) convertXGMoveToStringWithHits(playedMove [8]int32, initialPos *xgparser.Position, activePlayer int32) string {
	if initialPos == nil {
		return d.convertXGMoveToString(playedMove, activePlayer)
	}

	// Note: No mirroring needed - moves are always from the roller's perspective

	// Create a mutable copy of the position to track changes as we process moves
	// XG position format: Checkers[1-24] are points 1-24 (1-based indexing)
	// [0]=opponent's bar, [25]=player's bar
	// Positive values = player's checkers, negative = opponent's checkers
	positionCopy := make([]int8, 26)
	copy(positionCopy, initialPos.Checkers[:])

	// Parse raw moves into from/to pairs and track hits
	var items []xgMoveWithHit
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		// Check for end of move marker (-1 when from is also -1)
		if from == -1 {
			break
		}
		// Handle implicit bear-off: XG sometimes encodes bear-off as to=-1 or to<=0
		// when the calculated destination (from - die) would be <= 0
		// This happens when bearing off from the home board (points 1-6)
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2 // Convert to explicit bear-off
		}
		// Skip invalid point values (should be 1-24 for points, 25 for bar, -2 for bear off)
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}

		// Check if this move hits a blot
		// The destination point must have exactly one opponent checker (negative value in XG format)
		isHit := false
		if to >= 1 && to <= 24 {
			// Position.Checkers uses 1-based indexing: Checkers[1] = point 1, Checkers[24] = point 24
			if positionCopy[to] == -1 {
				isHit = true
				// Update position: opponent checker goes to bar
				positionCopy[to] = 0
			}
		}

		// Update position: move our checker
		if from >= 1 && from <= 24 {
			// Position.Checkers uses 1-based indexing
			if positionCopy[from] > 0 {
				positionCopy[from]--
			}
		} else if from == 25 {
			// From bar - player's bar is at index 25
			if positionCopy[25] > 0 {
				positionCopy[25]--
			}
		}

		if to >= 1 && to <= 24 {
			// Position.Checkers uses 1-based indexing
			positionCopy[to]++
		}

		// Store directly - no conversion needed since moves are already in roller's perspective
		items = append(items, xgMoveWithHit{from: from, to: to, isHit: isHit})
	}

	if len(items) == 0 {
		return "Cannot Move"
	}

	// Try to merge consecutive checker slides - but preserve hit info for the final move
	// For slides, only the last move in a chain can be a hit
	items = d.mergeSlidesWithHits(items)

	// Sort moves by 'from' point descending (standard backgammon notation)
	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	// Format each move as string
	formatPoint := func(p int32) string {
		if p == 25 {
			return "bar"
		} else if p == -2 {
			return "off"
		} else if p >= 1 && p <= 24 {
			return fmt.Sprintf("%d", p)
		}
		return fmt.Sprintf("?%d", p)
	}

	// Build move string, grouping identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		allHits := item.isHit
		// Count consecutive identical moves
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
				allHits = allHits && items[j].isHit
			} else {
				break
			}
		}

		hitMarker := ""
		if item.isHit || allHits {
			hitMarker = "*"
		}

		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s%s(%d)", formatPoint(item.from), formatPoint(item.to), hitMarker, count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s%s", formatPoint(item.from), formatPoint(item.to), hitMarker))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// mergeSlidesWithHits merges consecutive moves of the same checker, preserving hit info
// For example: 14/12 12/8 becomes 14/8, but only if 12 was just a waypoint (not hit)
// If there was a hit at the intermediate point, we keep both moves to show the hit
func (d *Database) mergeSlidesWithHits(items []xgMoveWithHit) []xgMoveWithHit {
	if len(items) <= 1 {
		return items
	}

	// Try to merge chains: if move[i].to == move[j].from and move[i] is not a hit,
	// they can be merged (the intermediate point was just a waypoint)
	result := make([]xgMoveWithHit, 0, len(items))
	used := make([]bool, len(items))

	for i := 0; i < len(items); i++ {
		if used[i] {
			continue
		}

		// Start a chain from this item
		chainFrom := items[i].from
		chainTo := items[i].to
		chainHit := items[i].isHit
		used[i] = true

		// Only extend if the current segment doesn't end with a hit
		// (we want to show hits, so don't merge past them)
		if !chainHit {
			// Extend chain forward: find items where from == chainTo
			for changed := true; changed; {
				changed = false
				for j := 0; j < len(items); j++ {
					if used[j] {
						continue
					}
					if items[j].from == chainTo {
						chainTo = items[j].to
						chainHit = items[j].isHit
						used[j] = true
						changed = true
						// If this new segment has a hit, stop extending
						if chainHit {
							break
						}
					}
				}
				if chainHit {
					break
				}
			}
		}

		result = append(result, xgMoveWithHit{from: chainFrom, to: chainTo, isHit: chainHit})
	}

	return result
}

// xgMoveWithHit represents a single move in XG format with hit information
type xgMoveWithHit struct {
	from  int32
	to    int32
	isHit bool
}

// xgMoveItem is unused but kept for documentation
type xgMoveItem struct {
	from int32
	to   int32
}

// mergeSlides merges consecutive moves of the same checker
// For example: 14/12 12/8 becomes 14/8
func (d *Database) mergeSlides(fromPts, toPts []int32) ([]int32, []int32) {
	if len(fromPts) <= 1 {
		return fromPts, toPts
	}

	// Count how many times each destination point is used
	// If multiple moves end at the same point, we should NOT merge through that point
	// because it means different checkers are moving, not the same checker sliding
	toCount := make(map[int32]int)
	for _, t := range toPts {
		toCount[t]++
	}

	// Also count how many times each point is a source
	fromCount := make(map[int32]int)
	for _, f := range fromPts {
		fromCount[f]++
	}

	// Try to merge chains: if move[i].to == move[j].from, they can be merged
	// BUT only if that intermediate point appears exactly once as a destination AND once as a source
	resultFrom := make([]int32, 0, len(fromPts))
	resultTo := make([]int32, 0, len(toPts))
	used := make([]bool, len(fromPts))

	for i := 0; i < len(fromPts); i++ {
		if used[i] {
			continue
		}

		// Start a chain from this item
		chainFrom := fromPts[i]
		chainTo := toPts[i]
		used[i] = true

		// Extend chain forward: find items where from == chainTo
		// Only merge if the intermediate point is not used by multiple checkers
		for changed := true; changed; {
			changed = false
			for j := 0; j < len(fromPts); j++ {
				if used[j] {
					continue
				}
				if fromPts[j] == chainTo {
					// Check if this is a valid merge (same checker moving)
					// Don't merge if chainTo is a destination for multiple moves
					// or if chainTo is a source for multiple moves
					// This indicates different checkers
					if toCount[chainTo] > 1 || fromCount[chainTo] > 1 {
						continue // Don't merge - different checkers
					}
					chainTo = toPts[j]
					used[j] = true
					changed = true
				}
			}
		}

		resultFrom = append(resultFrom, chainFrom)
		resultTo = append(resultTo, chainTo)
	}

	return resultFrom, resultTo
}

// convertCubeAction converts cube action code to string
func (d *Database) convertCubeAction(action int32) string {
	switch action {
	case 0:
		return "No Double"
	case 1:
		return "Double"
	case 2:
		return "Take"
	case 3:
		return "Pass"
	default:
		return fmt.Sprintf("Unknown(%d)", action)
	}
}

// ErrDuplicateMatch is returned when attempting to import a match that already exists
var ErrDuplicateMatch = fmt.Errorf("duplicate match: this match has already been imported")

// ComputeMatchHash generates a unique hash for a match based on full match transcription
// This is used to detect duplicate imports - includes all moves and decisions
func ComputeMatchHash(match *xgparser.Match) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2Name))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, match.Metadata.MatchLength))

	// Include full game transcription
	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|",
			gameIdx, game.InitialScore[0], game.InitialScore[1], game.Winner, game.PointsWon))

		// Include all moves in the game
		for moveIdx, move := range game.Moves {
			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, move.MoveType))

			if move.CheckerMove != nil {
				// Include dice and played move
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%v|",
					move.CheckerMove.Dice[0], move.CheckerMove.Dice[1],
					move.CheckerMove.PlayedMove))
			}

			if move.CubeMove != nil {
				// Include cube action details
				hashBuilder.WriteString(fmt.Sprintf("c%d|", move.CubeMove.CubeAction))
			}
		}
	}

	// Compute SHA256 hash of the full transcription
	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// computeMatchHashFromStoredData computes a hash for existing matches in the database
// This is used during migration when we don't have access to the original XG file
func computeMatchHashFromStoredData(db *sql.DB, matchID int64, p1Name, p2Name string, matchLength int32) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(p1Name))
	p2 := strings.TrimSpace(strings.ToLower(p2Name))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, matchLength))

	// Query all games for this match
	gameRows, err := db.Query(`
		SELECT id, game_number, initial_score_1, initial_score_2, winner, points_won 
		FROM game WHERE match_id = ? ORDER BY game_number`, matchID)
	if err != nil {
		// Fallback to simple hash
		hash := sha256.Sum256([]byte(hashBuilder.String()))
		return hex.EncodeToString(hash[:])
	}
	defer gameRows.Close()

	for gameRows.Next() {
		var gameID int64
		var gameNum, initScore1, initScore2, winner, pointsWon int32
		if err := gameRows.Scan(&gameID, &gameNum, &initScore1, &initScore2, &winner, &pointsWon); err != nil {
			continue
		}

		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|", gameNum, initScore1, initScore2, winner, pointsWon))

		// Query all moves for this game
		moveRows, err := db.Query(`
			SELECT move_number, move_type, dice_1, dice_2, checker_move, cube_action 
			FROM move WHERE game_id = ? ORDER BY move_number`, gameID)
		if err != nil {
			continue
		}

		for moveRows.Next() {
			var moveNum int32
			var moveType string
			var dice1, dice2 int32
			var checkerMove, cubeAction sql.NullString
			if err := moveRows.Scan(&moveNum, &moveType, &dice1, &dice2, &checkerMove, &cubeAction); err != nil {
				continue
			}

			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveNum, moveType))
			if moveType == "checker" && checkerMove.Valid {
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%s|", dice1, dice2, checkerMove.String))
			}
			if moveType == "cube" && cubeAction.Valid {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", cubeAction.String))
			}
		}
		if err := moveRows.Err(); err != nil {
			return ""
		}
		moveRows.Close()
	}
	if err := gameRows.Err(); err != nil {
		return ""
	}

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// CheckMatchExists checks if a match with the given hash already exists in the database
// Returns the existing match ID if found, 0 otherwise
func (d *Database) CheckMatchExists(matchHash string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM match WHERE match_hash = ?`, matchHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error checking for duplicate match: %w", err)
	}
	return existingID, nil
}

// CheckMatchExistsLocked is the same as CheckMatchExists but doesn't acquire the lock
// Use this when you already hold the database lock
func (d *Database) checkMatchExistsLocked(matchHash string) (int64, error) {
	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM match WHERE match_hash = ?`, matchHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error checking for duplicate match: %w", err)
	}
	return existingID, nil
}

// ============================================================================
// GnuBG / Jellyfish import functions (SGF, MAT, TXT formats)
// ============================================================================

// ImportGnuBGMatchFromText imports a match from clipboard/string content in MAT/TXT format
// using the gnubgparser library. Only MAT/TXT format is supported (no SGF).
func (d *Database) ImportGnuBGMatchFromText(content string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	gnuMatch, err := gnubgparser.ParseMAT(strings.NewReader(content))
	if err != nil {
		return 0, fmt.Errorf("failed to parse match text: %w", err)
	}

	return d.importGnuBGMatchInternal(gnuMatch, "clipboard", false)
}

// ImportGnuBGMatch imports a match from a GnuBG file (SGF, MAT, or TXT format)
// using the gnubgparser library. SGF files include full analysis data,
// while MAT/TXT files contain only moves (no analysis).
func (d *Database) ImportGnuBGMatch(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Determine format from extension and parse accordingly
	ext := strings.ToLower(filepath.Ext(filePath))
	var gnuMatch *gnubgparser.Match
	var err error

	// isSGF indicates whether moves use absolute coordinates (Player 0's system)
	// SGF: moves are in absolute coords, MoveString is in letter format (needs conversion)
	// MAT/TXT: moves are in player-relative coords, MoveString is already human-readable
	isSGF := ext == ".sgf"

	switch ext {
	case ".sgf":
		gnuMatch, err = gnubgparser.ParseSGFFile(filePath)
	case ".mat", ".txt":
		gnuMatch, err = gnubgparser.ParseMATFile(filePath)
	default:
		return 0, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to parse file: %w", err)
	}

	return d.importGnuBGMatchInternal(gnuMatch, filePath, isSGF)
}

// importGnuBGMatchInternal is the shared implementation for importing a parsed GnuBG match.
// isSGF indicates SGF format where moves use absolute coordinates.
func (d *Database) importGnuBGMatchInternal(gnuMatch *gnubgparser.Match, filePath string, isSGF bool) (int64, error) {
	// Parse match date
	var matchDate time.Time
	if gnuMatch.Metadata.Date != "" {
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			"2006/01/02",
			"01/02/2006",
			"January 2, 2006",
			time.RFC3339,
		} {
			if t, parseErr := time.Parse(layout, gnuMatch.Metadata.Date); parseErr == nil {
				matchDate = t
				break
			}
		}
	}
	if matchDate.IsZero() {
		matchDate = time.Now()
	}

	// Compute match hash for duplicate detection
	matchHash := ComputeGnuBGMatchHash(gnuMatch)

	// Compute canonical hash (format-independent)
	canonicalHash := ComputeCanonicalMatchHashFromGnuBG(gnuMatch)

	// Check if this match already exists (same format)
	existingMatchID, err := d.checkMatchExistsLocked(matchHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for duplicate match: %w", err)
	}
	if existingMatchID > 0 {
		return 0, ErrDuplicateMatch
	}

	// Check for canonical duplicate (same match from different format)
	canonicalMatchID, err := d.checkCanonicalMatchExistsLocked(canonicalHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for canonical duplicate: %w", err)
	}
	isCanonicalDuplicate := canonicalMatchID > 0

	// Begin transaction for atomic import
	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Insert match metadata or reuse existing canonical match
	var matchID int64
	if isCanonicalDuplicate {
		matchID = canonicalMatchID
		fmt.Printf("Canonical duplicate detected - reusing match ID %d, importing new analysis only\n", matchID)
	} else {
		result, err := tx.Exec(`
			INSERT INTO match (player1_name, player2_name, event, location, round,
			                   match_length, match_date, file_path, game_count, match_hash, canonical_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, gnuMatch.Metadata.Player1, gnuMatch.Metadata.Player2,
			gnuMatch.Metadata.Event, gnuMatch.Metadata.Place, gnuMatch.Metadata.Round,
			gnuMatch.Metadata.MatchLength, matchDate, filePath, len(gnuMatch.Games), matchHash, canonicalHash)

		if err != nil {
			return 0, fmt.Errorf("failed to insert match: %w", err)
		}

		matchID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get match ID: %w", err)
		}

		// Auto-link tournament from event metadata
		eventName := strings.TrimSpace(gnuMatch.Metadata.Event)
		if eventName != "" {
			var tournamentID int64
			err2 := tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, eventName).Scan(&tournamentID)
			if err2 != nil {
				res2, err3 := tx.Exec(`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, eventName)
				if err3 == nil {
					tournamentID, err = res2.LastInsertId()
					if err != nil {
						return 0, fmt.Errorf("failed to get last insert ID: %w", err)
					}
				}
			}
			if tournamentID > 0 {
				_, err = tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID)
				if err != nil {
					fmt.Printf("Warning: failed to link match to tournament: %v\n", err)
				}
			}
		}
	}

	// Build a position cache for deduplication
	positionCache := make(map[string]int64) // map[positionJSON]positionID
	semanticCache := make(map[string]int64) // map[semanticKey]positionID

	// Load existing positions into cache
	existingRows, err := tx.Query(`SELECT id, state FROM position`)
	if err != nil {
		return 0, fmt.Errorf("failed to load existing positions: %w", err)
	}

	for existingRows.Next() {
		var existingID int64
		var existingStateJSON string
		if err := existingRows.Scan(&existingID, &existingStateJSON); err != nil {
			continue
		}

		var existingPosition Position
		if err := json.Unmarshal([]byte(existingStateJSON), &existingPosition); err != nil {
			continue
		}

		normalizedPosition := existingPosition.NormalizeForStorage()
		normalizedPosition.ID = 0
		normalizedJSON, err := json.Marshal(normalizedPosition)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		positionCache[string(normalizedJSON)] = existingID
		semanticCache[positionSemanticKey(&normalizedPosition)] = existingID
	}
	if err := existingRows.Err(); err != nil {
		return 0, err
	}
	existingRows.Close()

	fmt.Printf("Loaded %d existing positions into cache for GnuBG import\n", len(positionCache))

	if isCanonicalDuplicate {
		// Canonical duplicate: import analysis to existing positions, create genuinely new ones
		for gameIdx, game := range gnuMatch.Games {
			game.GameNumber = gameIdx + 1
			currentBoard := initStandardGnuBGPosition()

			for i := range game.Moves {
				moveRec := &game.Moves[i]

				switch string(moveRec.Type) {
				case "setboard":
					if moveRec.Position != nil {
						currentBoard = *moveRec.Position
					}
					continue
				case "setdice", "setcube", "setcubepos":
					if moveRec.Type == "setcube" {
						currentBoard.CubeValue = moveRec.CubeValue
					}
					if moveRec.Type == "setcubepos" {
						currentBoard.CubeOwner = moveRec.CubeOwner
					}
					continue
				}

				if moveRec.Position == nil {
					posCopy := currentBoard
					moveRec.Position = &posCopy
				}

				switch moveRec.Type {
				case "move":
					if moveRec.Analysis != nil && len(moveRec.Analysis.Moves) > 0 {
						pos, err := d.createPositionFromGnuBG(moveRec.Position, &game, gnuMatch.Metadata.MatchLength)
						if err != nil {
							continue
						}
						pos.PlayerOnRoll = moveRec.Player
						pos.DecisionType = CheckerAction
						pos.Dice = [2]int{moveRec.Dice[0], moveRec.Dice[1]}

						posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, positionCache, semanticCache)
						if err != nil {
							continue
						}

						var checkerMoveStr string
						if isSGF {
							checkerMoveStr = convertGnuBGMoveToString(moveRec.Move, moveRec.Player)
						} else {
							checkerMoveStr = moveRec.MoveString
							if checkerMoveStr == "" {
								checkerMoveStr = convertPlayerRelativeMoveToString(moveRec.Move)
							}
						}
						err = d.saveGnuBGCheckerAnalysisToPositionInTx(tx, posID, moveRec.Analysis, moveRec.Player, checkerMoveStr, isSGF)
						if err != nil {
							fmt.Printf("Warning: failed to save analysis for canonical duplicate: %v\n", err)
						}
					}
					if moveRec.CubeAnalysis != nil {
						pos, err := d.createPositionFromGnuBG(moveRec.Position, &game, gnuMatch.Metadata.MatchLength)
						if err == nil {
							pos.PlayerOnRoll = moveRec.Player
							pos.DecisionType = CheckerAction
							pos.Dice = [2]int{moveRec.Dice[0], moveRec.Dice[1]}
							posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, positionCache, semanticCache)
							if err == nil {
								// Convert MWC to EMG for match play (copy to avoid mutating original)
								cubeAnalysis := *moveRec.CubeAnalysis
								if gnuMatch.Metadata.MatchLength > 0 && moveRec.Position != nil {
									convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], moveRec.Player, moveRec.Position.CubeValue, gnuMatch.Metadata.MatchLength)
								}
								d.saveGnuBGCubeAnalysisForCheckerPositionInTx(tx, posID, &cubeAnalysis)
							}
						}
					}

				case "double":
					// Only import cube analysis for "double" entries (skip take/drop which are redundant)
					if moveRec.CubeAnalysis != nil {
						pos, err := d.createPositionFromGnuBG(moveRec.Position, &game, gnuMatch.Metadata.MatchLength)
						if err != nil {
							continue
						}
						pos.PlayerOnRoll = moveRec.Player
						pos.DecisionType = CubeAction
						pos.Dice = [2]int{0, 0}

						posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, positionCache, semanticCache)
						if err != nil {
							continue
						}

						// Determine played action by looking at the opponent's response
						cubeAction := "Double/Pass" // default
						for j := i + 1; j < len(game.Moves); j++ {
							nextType := string(game.Moves[j].Type)
							if nextType == "take" {
								cubeAction = "Double/Take"
								break
							} else if nextType == "drop" {
								cubeAction = "Double/Pass"
								break
							} else if nextType != "setboard" && nextType != "setdice" && nextType != "setcube" && nextType != "setcubepos" {
								break
							}
						}
						// Convert MWC to EMG for match play (copy to avoid mutating original)
						cubeAnalysis := *moveRec.CubeAnalysis
						if gnuMatch.Metadata.MatchLength > 0 && moveRec.Position != nil {
							convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], moveRec.Player, moveRec.Position.CubeValue, gnuMatch.Metadata.MatchLength)
						}
						err = d.saveGnuBGCubeAnalysisToPositionInTx(tx, posID, &cubeAnalysis, cubeAction)
						if err != nil {
							fmt.Printf("Warning: failed to save cube analysis for canonical duplicate: %v\n", err)
						}
					}
				case "take", "drop":
					// Skip — cube analysis was already saved on the "double" entry
				}

				// Update board state
				switch string(moveRec.Type) {
				case "move":
					applyGnuBGCheckerMove(&currentBoard, moveRec, isSGF)
				case "take":
					if currentBoard.CubeValue == 0 {
						currentBoard.CubeValue = 2
					} else {
						currentBoard.CubeValue *= 2
					}
					currentBoard.CubeOwner = moveRec.Player
				}
			}
		}
	} else {
		// Normal import path
		for gameIdx, game := range gnuMatch.Games {
			game.GameNumber = gameIdx + 1
			gameID, err := d.importGnuBGGame(tx, matchID, &game)
			if err != nil {
				return 0, fmt.Errorf("failed to import game %d: %w", game.GameNumber, err)
			}

			currentBoard := initStandardGnuBGPosition()

			moveNumber := int32(0)
			for i := range game.Moves {
				moveRec := &game.Moves[i]

				switch string(moveRec.Type) {
				case "setboard":
					if moveRec.Position != nil {
						currentBoard = *moveRec.Position
					}
					continue
				case "setdice":
					continue
				case "setcube":
					currentBoard.CubeValue = moveRec.CubeValue
					continue
				case "setcubepos":
					currentBoard.CubeOwner = moveRec.CubeOwner
					continue
				}

				if moveRec.Position == nil {
					posCopy := currentBoard
					moveRec.Position = &posCopy
				}

				// For "double" moves, determine the actual cube action
				// by looking ahead to the opponent's response (take/drop)
				cubeAction := ""
				if string(moveRec.Type) == "double" {
					cubeAction = "Double/Pass" // default if no response found
					for j := i + 1; j < len(game.Moves); j++ {
						nextType := string(game.Moves[j].Type)
						if nextType == "take" {
							cubeAction = "Double/Take"
							break
						} else if nextType == "drop" {
							cubeAction = "Double/Pass"
							break
						} else if nextType != "setboard" && nextType != "setdice" && nextType != "setcube" && nextType != "setcubepos" {
							break // unexpected move type, stop looking
						}
					}
				}

				err := d.importGnuBGMove(tx, gameID, moveNumber, moveRec, &game, gnuMatch.Metadata.MatchLength, positionCache, isSGF, cubeAction)
				if err != nil {
					fmt.Printf("Warning: failed to import move %d in game %d: %v\n", moveNumber, game.GameNumber, err)
					moveNumber++
					continue
				}

				switch string(moveRec.Type) {
				case "move":
					applyGnuBGCheckerMove(&currentBoard, moveRec, isSGF)
				case "take":
					if currentBoard.CubeValue == 0 {
						currentBoard.CubeValue = 2
					} else {
						currentBoard.CubeValue *= 2
					}
					currentBoard.CubeOwner = moveRec.Player
				}

				// take/drop don't increment moveNumber (they're part of the double action)
				if string(moveRec.Type) != "take" && string(moveRec.Type) != "drop" {
					moveNumber++
				}
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully imported GnuBG match %d with %d games from %s\n", matchID, len(gnuMatch.Games), filePath)
	return matchID, nil
}

// importGnuBGGame inserts a game record from gnubgparser data and returns its ID
func (d *Database) importGnuBGGame(tx *sql.Tx, matchID int64, game *gnubgparser.Game) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2,
		                  winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, matchID, game.GameNumber, game.Score[0], game.Score[1],
		game.Winner, game.Points, len(game.Moves))

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// importGnuBGMove imports a single move record from gnubgparser data
// isSGF indicates SGF format where moves use absolute coordinates (Player 0's system)
func (d *Database) importGnuBGMove(tx *sql.Tx, gameID int64, moveNumber int32, moveRec *gnubgparser.MoveRecord, game *gnubgparser.Game, matchLength int, positionCache map[string]int64, isSGF bool, cubeAction string) error {
	switch moveRec.Type {
	case "move":
		return d.importGnuBGCheckerMove(tx, gameID, moveNumber, moveRec, game, matchLength, positionCache, isSGF)
	case "double":
		// cubeAction is determined by the caller based on the opponent's response
		// ("Double/Take" or "Double/Pass")
		return d.importGnuBGCubeMove(tx, gameID, moveNumber, moveRec, game, matchLength, positionCache, cubeAction, isSGF)
	case "take", "drop":
		// Skip take/drop as separate entries — the "double" entry already captures
		// the full cube decision (like XG's single "Double/Pass" or "Double/Take")
		return nil
	case "resign":
		// Skip resign moves - they don't produce positions
		return nil
	default:
		// Skip unknown move types
		return nil
	}
}

// importGnuBGCheckerMove handles importing a checker move from gnubgparser
// isSGF indicates SGF format where moves use absolute coordinates and MoveString is in letter format
func (d *Database) importGnuBGCheckerMove(tx *sql.Tx, gameID int64, moveNumber int32, moveRec *gnubgparser.MoveRecord, game *gnubgparser.Game, matchLength int, positionCache map[string]int64, isSGF bool) error {
	player := moveRec.Player // 0 or 1, maps directly to blunderDB

	// Convert player to XG-style encoding for DB storage consistency
	// The DB player column uses XG encoding (1=Player1, -1=Player2) so that
	// GetMatchMovePositions can use convertXGPlayerToBlunderDB uniformly.
	dbPlayer := convertBlunderDBPlayerToXG(player)

	// Get move string
	var checkerMoveStr string
	if isSGF {
		// For SGF files, Move[8]int is in absolute coordinates (Player 0's system).
		// Always compute from Move array since MoveString is in letter format.
		checkerMoveStr = convertGnuBGMoveToString(moveRec.Move, moveRec.Player)
	} else {
		// For MAT/TXT files, MoveString is already in human-readable notation.
		checkerMoveStr = moveRec.MoveString
		if checkerMoveStr == "" {
			// Move[8]int is in player-relative coordinates for MAT/TXT
			checkerMoveStr = convertPlayerRelativeMoveToString(moveRec.Move)
		}
	}

	// If position is available, create and save it
	var positionID int64
	if moveRec.Position != nil {
		pos, err := d.createPositionFromGnuBG(moveRec.Position, game, matchLength)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		pos.PlayerOnRoll = player
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{moveRec.Dice[0], moveRec.Dice[1]}

		posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID
	}

	// Save move record
	var moveResult sql.Result
	var err error
	if positionID > 0 {
		moveResult, err = tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, dbPlayer,
			moveRec.Dice[0], moveRec.Dice[1], checkerMoveStr)
	} else {
		moveResult, err = tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, NULL, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", dbPlayer,
			moveRec.Dice[0], moveRec.Dice[1], checkerMoveStr)
	}
	if err != nil {
		return err
	}

	moveID, err := moveResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Save analysis if available (SGF files only)
	if moveRec.Analysis != nil && len(moveRec.Analysis.Moves) > 0 && positionID > 0 {
		// Save to move_analysis table
		for _, moveOpt := range moveRec.Analysis.Moves {
			err = d.saveGnuBGMoveAnalysisInTx(tx, moveID, &moveOpt)
			if err != nil {
				fmt.Printf("Warning: failed to save checker analysis: %v\n", err)
			}
		}

		// Save to position analysis table (for UI compatibility)
		err = d.saveGnuBGCheckerAnalysisToPositionInTx(tx, positionID, moveRec.Analysis, moveRec.Player, checkerMoveStr, isSGF)
		if err != nil {
			fmt.Printf("Warning: failed to save position analysis: %v\n", err)
		}
	}

	// Save cube analysis if available on a checker move position
	if moveRec.CubeAnalysis != nil && positionID > 0 {
		// Convert MWC to EMG for match play (copy to avoid mutating original)
		cubeAnalysis := *moveRec.CubeAnalysis
		if matchLength > 0 && moveRec.Position != nil {
			convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], player, moveRec.Position.CubeValue, matchLength)
		}
		err = d.saveGnuBGCubeAnalysisForCheckerPositionInTx(tx, positionID, &cubeAnalysis)
		if err != nil {
			fmt.Printf("Warning: failed to save cube analysis for checker position: %v\n", err)
		}
	}

	return nil
}

// importGnuBGCubeMove handles importing a cube move (double/take/drop) from gnubgparser
func (d *Database) importGnuBGCubeMove(tx *sql.Tx, gameID int64, moveNumber int32, moveRec *gnubgparser.MoveRecord, game *gnubgparser.Game, matchLength int, positionCache map[string]int64, cubeAction string, isSGF bool) error {
	player := moveRec.Player // 0 or 1

	// Convert player to XG-style encoding for DB storage consistency
	dbPlayer := convertBlunderDBPlayerToXG(player)

	var positionID int64
	if moveRec.Position != nil {
		pos, err := d.createPositionFromGnuBG(moveRec.Position, game, matchLength)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		pos.PlayerOnRoll = player
		pos.DecisionType = CubeAction
		pos.Dice = [2]int{0, 0} // No dice for cube decisions

		posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID
	}

	// Save move record
	var moveResult sql.Result
	var err error
	if positionID > 0 {
		moveResult, err = tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, cube_action)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "cube", positionID, dbPlayer, 0, 0, cubeAction)
	} else {
		moveResult, err = tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, cube_action)
			VALUES (?, ?, ?, NULL, ?, ?, ?, ?)
		`, gameID, moveNumber, "cube", dbPlayer, 0, 0, cubeAction)
	}
	if err != nil {
		return err
	}

	moveID, err := moveResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Save cube analysis if available (SGF files only)
	if moveRec.CubeAnalysis != nil && positionID > 0 {
		// Convert MWC to EMG for match play (copy to avoid mutating original)
		cubeAnalysis := *moveRec.CubeAnalysis
		if matchLength > 0 && moveRec.Position != nil {
			convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], player, moveRec.Position.CubeValue, matchLength)
		}

		// Save to move_analysis table
		err = d.saveGnuBGCubeMoveAnalysisInTx(tx, moveID, &cubeAnalysis)
		if err != nil {
			fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
		}

		// Save to position analysis table (for UI compatibility)
		err = d.saveGnuBGCubeAnalysisToPositionInTx(tx, positionID, &cubeAnalysis, cubeAction)
		if err != nil {
			fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
		}
	}

	return nil
}

// initStandardGnuBGPosition returns a gnubgparser.Position set to the standard
// backgammon starting position. Used when SGF files don't include explicit setboard events.
//
// In gnuBG's player-relative encoding (0=ace point, 23=24-point):
//   - Each player has: 2@pt23, 5@pt12, 3@pt7, 5@pt5 (15 checkers total)
func initStandardGnuBGPosition() gnubgparser.Position {
	var pos gnubgparser.Position
	pos.CubeValue = 1
	pos.CubeOwner = -1 // center

	// Standard starting position (same for both players from their own perspective)
	for p := 0; p < 2; p++ {
		pos.Board[p][23] = 2 // 24-point: 2 checkers
		pos.Board[p][12] = 5 // 13-point: 5 checkers
		pos.Board[p][7] = 3  // 8-point: 3 checkers
		pos.Board[p][5] = 5  // 6-point: 5 checkers
	}

	return pos
}

// applyGnuBGCheckerMove updates a gnubgparser board state after a checker move.
//
// When isAbsoluteCoords is true (SGF format):
//
//	Move[8]int uses absolute coordinates (Player 0's system: 0=1pt, 23=24pt, 24=bar, 25=off).
//	For Player 1, indices must be mirrored to reach the player-relative board.
//
// When isAbsoluteCoords is false (MAT/TXT format):
//
//	Move[8]int uses player-relative coordinates (from player's perspective:
//	0=ace/home, 23=24pt, 24=bar, -1=off). No mirroring needed.
//
// Board[p] always uses player-relative coords (0=ace/home, 23=far, 24=bar).
func applyGnuBGCheckerMove(board *gnubgparser.Position, moveRec *gnubgparser.MoveRecord, isAbsoluteCoords bool) {
	player := moveRec.Player
	opponent := 1 - player

	for i := 0; i < 8; i += 2 {
		from := moveRec.Move[i]
		to := moveRec.Move[i+1]
		if from == -1 {
			break
		}

		var fromBoard, toBoard, opponentBoard int
		var isBearOff bool

		if isAbsoluteCoords {
			// SGF: absolute coordinates — mirror for Player 1
			fromBoard = from
			if player == 1 && from != 24 {
				fromBoard = 23 - from
			}

			isBearOff = (to == 25)

			if !isBearOff {
				toBoard = to
				if player == 1 {
					toBoard = 23 - to
				}
				// Opponent's board index for the same physical point
				if player == 0 {
					opponentBoard = 23 - to
				} else {
					opponentBoard = to
				}
			}
		} else {
			// MAT/TXT: player-relative coordinates — no mirroring needed
			fromBoard = from // already in player's perspective

			isBearOff = (to == -1)

			if !isBearOff {
				toBoard = to // already in player's perspective
				// Opponent sees the mirror of this physical point
				opponentBoard = 23 - to
			}
		}

		// Remove checker from source point
		if fromBoard >= 0 && fromBoard <= 24 {
			board.Board[player][fromBoard]--
		}

		// If bearing off, checker leaves the board entirely
		if isBearOff {
			continue
		}

		// Check for hit at destination
		if opponentBoard >= 0 && opponentBoard <= 23 {
			if board.Board[opponent][opponentBoard] == 1 {
				// Hit: send opponent's checker to the bar
				board.Board[opponent][opponentBoard] = 0
				board.Board[opponent][24]++
			}
		}

		// Place checker at destination
		if toBoard >= 0 && toBoard <= 24 {
			board.Board[player][toBoard]++
		}
	}
}

// createPositionFromGnuBG converts a gnubgparser.Position to a blunderDB Position
//
// gnubgparser board encoding (player-relative):
//   - Board[player][0-23]: board points from player's perspective (0=ace/home, 23=far)
//   - Board[player][24]: checkers on bar
//
// blunderDB board encoding (absolute):
//   - Points[0]: Player 2's bar (White)
//   - Points[1-24]: board points (standard numbering)
//   - Points[25]: Player 1's bar (Black)
//   - Color 0 = Player 1 (Black, moves 24→1), Color 1 = Player 2 (White, moves 1→24)
//
// Mapping:
//   - Board[0][i] (player 0/Black): blunderDB point (i+1), Color 0
//   - Board[1][i] (player 1/White): blunderDB point (24-i), Color 1
//   - Board[0][24] (player 0 bar): blunderDB Points[25]
//   - Board[1][24] (player 1 bar): blunderDB Points[0]
func (d *Database) createPositionFromGnuBG(gnubgPos *gnubgparser.Position, game *gnubgparser.Game, matchLength int) (*Position, error) {
	// Calculate away scores
	// blunderDB stores scores as "points away from winning"
	awayScore1 := matchLength - game.Score[0]
	awayScore2 := matchLength - game.Score[1]

	// Handle unlimited/money match (matchLength == 0)
	if matchLength == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Convert cube value from actual (1,2,4,8...) to exponent (0,1,2,3...)
	cubeValue := 0
	if gnubgPos.CubeValue > 0 {
		for v := gnubgPos.CubeValue; v > 1; v >>= 1 {
			cubeValue++
		}
	}

	// Cube owner: gnubgparser uses -1=center, 0=player0, 1=player1 (same as blunderDB)
	cubeOwner := gnubgPos.CubeOwner

	pos := &Position{
		PlayerOnRoll: gnubgPos.OnRoll, // Will be overridden from move context
		DecisionType: CheckerAction,   // Will be overridden from move context
		Score:        [2]int{awayScore1, awayScore2},
		Cube: Cube{
			Value: cubeValue,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0}, // Will be set from move data
	}

	// Initialize all points as empty
	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = Point{Checkers: 0, Color: -1}
	}

	// Place Player 0's (Black) checkers
	for pt := 0; pt < 25; pt++ {
		count := gnubgPos.Board[0][pt]
		if count > 0 {
			if pt == 24 {
				// Bar: Player 0's bar → blunderDB index 25
				pos.Board.Points[25] = Point{Checkers: count, Color: 0}
			} else {
				// Board point: gnubg pt → blunderDB pt+1
				pos.Board.Points[pt+1] = Point{Checkers: count, Color: 0}
			}
		}
	}

	// Place Player 1's (White) checkers
	for pt := 0; pt < 25; pt++ {
		count := gnubgPos.Board[1][pt]
		if count > 0 {
			if pt == 24 {
				// Bar: Player 1's bar → blunderDB index 0
				pos.Board.Points[0] = Point{Checkers: count, Color: 1}
			} else {
				// Board point: gnubg pt (from player 1's view) → blunderDB (24-pt)
				pos.Board.Points[24-pt] = Point{Checkers: count, Color: 1}
			}
		}
	}

	// Calculate bearoff (15 checkers total per player minus those on the board)
	player1Total := 0
	player2Total := 0
	for i := 0; i < 26; i++ {
		if pos.Board.Points[i].Color == 0 {
			player1Total += pos.Board.Points[i].Checkers
		} else if pos.Board.Points[i].Color == 1 {
			player2Total += pos.Board.Points[i].Checkers
		}
	}
	pos.Board.Bearoff = [2]int{15 - player1Total, 15 - player2Total}

	return pos, nil
}

// convertGnuBGMoveToString converts a gnubgparser Move[8]int to standard notation.
// This function handles moves in ABSOLUTE coordinates (SGF format).
// Move encoding: 0-23 = board points (absolute), 24 = bar, 25 = off, -1 = unused
// For player 0: gnubg point i → standard point (i+1)
// For player 1: gnubg point i → standard point (24-i)
func convertGnuBGMoveToString(move [8]int, player int) string {
	formatPoint := func(pt int, p int) string {
		if pt == 24 {
			return "bar"
		}
		if pt == 25 {
			return "off"
		}
		if p == 0 {
			return fmt.Sprintf("%d", pt+1) // Player 0: 0→1, 23→24
		}
		return fmt.Sprintf("%d", 24-pt) // Player 1: 0→24, 23→1
	}

	return formatGnuBGMoveItems(move, player, formatPoint)
}

// convertPlayerRelativeMoveToString converts a player-relative Move[8]int to standard notation.
// This function handles moves in PLAYER-RELATIVE coordinates (MAT/TXT format).
// Move encoding: 0-23 = board points (from player's perspective), 24 = bar, -1 = off
// For both players: point i → standard point (i+1)
func convertPlayerRelativeMoveToString(move [8]int) string {
	formatPoint := func(pt int, _ int) string {
		if pt == 24 {
			return "bar"
		}
		if pt == -1 {
			return "off"
		}
		return fmt.Sprintf("%d", pt+1) // Player-relative: 0→1, 23→24
	}

	return formatGnuBGMoveItems(move, 0, formatPoint)
}

// formatGnuBGMoveItems is a helper that formats move items using a point formatter.
func formatGnuBGMoveItems(move [8]int, player int, formatPoint func(int, int) string) string {

	type moveItem struct {
		from string
		to   string
	}

	var items []moveItem
	for i := 0; i < 8; i += 2 {
		from := move[i]
		to := move[i+1]
		if from == -1 {
			break
		}
		items = append(items, moveItem{
			from: formatPoint(from, player),
			to:   formatPoint(to, player),
		})
	}

	if len(items) == 0 {
		return "Cannot Move"
	}

	// Sort by 'from' point descending (standard notation)
	sort.Slice(items, func(i, j int) bool {
		// Parse to int for comparison
		fi, _ := strconv.Atoi(items[i].from)
		fj, _ := strconv.Atoi(items[j].from)
		if items[i].from == "bar" {
			fi = 25
		}
		if items[j].from == "bar" {
			fj = 25
		}
		return fi > fj
	})

	// Group identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
			} else {
				break
			}
		}
		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s(%d)", item.from, item.to, count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s", item.from, item.to))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// saveGnuBGMoveAnalysisInTx saves a gnubgparser MoveOption to the move_analysis table
func (d *Database) saveGnuBGMoveAnalysisInTx(tx *sql.Tx, moveID int64, moveOpt *gnubgparser.MoveOption) error {
	// gnubgparser rates are 0-1 fractions; convert to percentages
	player1WinRate := float64(moveOpt.Player1WinRate) * 100.0
	player2WinRate := float64(moveOpt.Player2WinRate) * 100.0

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "checker", translateGnuBGAnalysisDepth(moveOpt.AnalysisDepth),
		moveOpt.Equity, 0.0,
		player1WinRate, float64(moveOpt.Player1GammonRate)*100.0, float64(moveOpt.Player1BackgammonRate)*100.0,
		player2WinRate, float64(moveOpt.Player2GammonRate)*100.0, float64(moveOpt.Player2BackgammonRate)*100.0)

	return err
}

// ============================================================================
// GNUbg Match Equity Table (MET) — Kazaross-XG2 (GNUbg default) with Zadeh fallback
//
// GNUbg loads the Kazaross-XG2 explicit MET by default (met/Kazaross-XG2.xml).
// This table was generated using XG rollouts to 9pts, GNUbg Supremo full rollouts
// to 15pts, and extended to 25pts by projecting take points.
//
// For entries beyond the 25×25 explicit table (matches > 25 points), GNUbg uses
// the Zadeh model (N. Zadeh, Management Science 23, 986, 1977) as a fallback.
// We replicate this by computing Zadeh first (full 64×64), then overlaying
// the Kazaross-XG2 explicit values for indices 0-24.
//
// gnuBGPreCrawfordMET[i][j] = player 0's MWC when player 0 needs i+1 pts, player 1 needs j+1 pts.
// gnuBGPostCrawfordMET[n] = trailer's MWC when trailer needs n+1 pts and leader needs 1 pt.
//
// Antisymmetry: gnuBGPreCrawfordMET[i][j] + gnuBGPreCrawfordMET[j][i] = 1.0
// This means MET[myAway-1][theirAway-1] gives "my" MWC for either player.
// ============================================================================

const (
	gnuBGMaxScore     = 64
	gnuBGMaxCubeLevel = 7
)

// d3Array is a 3D array type used during Zadeh MET computation.
// Heap-allocated to avoid ~900KB stack pressure.
// Uses float32 to match GNUbg's native precision exactly.
type d3Array [gnuBGMaxScore][gnuBGMaxScore][gnuBGMaxCubeLevel]float32

// MET tables use float32 internally to match GNUbg's C `float` type exactly.
// The accumulated precision of float32 arithmetic in the Zadeh iteration
// produces MET values that match GNUbg's, ensuring correct equity conversions.
var (
	gnuBGPreCrawfordMET  [gnuBGMaxScore][gnuBGMaxScore]float32
	gnuBGPostCrawfordMET [gnuBGMaxScore]float32
)

// kazarossXG2PreCrawford is the Kazaross-XG2 pre-Crawford Match Equity Table (25×25).
// This is GNUbg's DEFAULT MET (loaded from met/Kazaross-XG2.xml).
// Generated using XG rollouts to 9pts, GNUbg Supremo full rollouts to 15pts,
// extended to 25pts by projecting take points.
// Index [i][j] = player 0's MWC when player 0 needs i+1 pts, player 1 needs j+1 pts.
var kazarossXG2PreCrawford = [25][25]float32{
	{0.50000, 0.67736, 0.75076, 0.81436, 0.84179, 0.88731, 0.90724, 0.93250, 0.94402, 0.959275, 0.966442, 0.975534, 0.979845, 0.985273, 0.987893, 0.99114, 0.99273, 0.99467, 0.99563, 0.99679, 0.99737, 0.99807, 0.99842, 0.99884, 0.99905},
	{0.32264, 0.50000, 0.59947, 0.66870, 0.74359, 0.79940, 0.84225, 0.87539, 0.90197, 0.923034, 0.939311, 0.952470, 0.962495, 0.970701, 0.976887, 0.98196, 0.98580, 0.98893, 0.99129, 0.99322, 0.99466, 0.99585, 0.99675, 0.99746, 0.99802},
	{0.24924, 0.40053, 0.50000, 0.57150, 0.64795, 0.71123, 0.76209, 0.80468, 0.84017, 0.870638, 0.894417, 0.914831, 0.930702, 0.944426, 0.954931, 0.96399, 0.97093, 0.97687, 0.98139, 0.98522, 0.98814, 0.99062, 0.99248, 0.99407, 0.99527},
	{0.18564, 0.33130, 0.42850, 0.50000, 0.57732, 0.64285, 0.69924, 0.74577, 0.78799, 0.824059, 0.853955, 0.879141, 0.900233, 0.918040, 0.932657, 0.94495, 0.95499, 0.96341, 0.97021, 0.97589, 0.98044, 0.98422, 0.98726, 0.98975, 0.99174},
	{0.15821, 0.25641, 0.35205, 0.42268, 0.50000, 0.56635, 0.62638, 0.67786, 0.72540, 0.767055, 0.802732, 0.833654, 0.859934, 0.882866, 0.902013, 0.91847, 0.93223, 0.94397, 0.95367, 0.96189, 0.96864, 0.97432, 0.97896, 0.98283, 0.98600},
	{0.11269, 0.20060, 0.28877, 0.35715, 0.43365, 0.50000, 0.56261, 0.61636, 0.66787, 0.713057, 0.753427, 0.788634, 0.819569, 0.846648, 0.869999, 0.89021, 0.90756, 0.92246, 0.93508, 0.94583, 0.95488, 0.96254, 0.96894, 0.97432, 0.97879},
	{0.09276, 0.15775, 0.23791, 0.30076, 0.37362, 0.43739, 0.50000, 0.55480, 0.60854, 0.656283, 0.700209, 0.739054, 0.774121, 0.805203, 0.832566, 0.85659, 0.87761, 0.89591, 0.91171, 0.92535, 0.93702, 0.94703, 0.95553, 0.96276, 0.96887},
	{0.06750, 0.12461, 0.19532, 0.25423, 0.32214, 0.38364, 0.44520, 0.50000, 0.55442, 0.603718, 0.649899, 0.691356, 0.729447, 0.763593, 0.794397, 0.82158, 0.84578, 0.86714, 0.88589, 0.90230, 0.91658, 0.92898, 0.93968, 0.94891, 0.95682},
	{0.05598, 0.09803, 0.15983, 0.21201, 0.27460, 0.33213, 0.39146, 0.44558, 0.50000, 0.550196, 0.597926, 0.641481, 0.682119, 0.718927, 0.752814, 0.78301, 0.81037, 0.83483, 0.85662, 0.87591, 0.89294, 0.90791, 0.92098, 0.93240, 0.94230},
	{0.040725, 0.076966, 0.129362, 0.175941, 0.232945, 0.286943, 0.343717, 0.396282, 0.449804, 0.500000, 0.548547, 0.593459, 0.635880, 0.674830, 0.711113, 0.74371, 0.77375, 0.80093, 0.82543, 0.84741, 0.86703, 0.88448, 0.89991, 0.91353, 0.92550},
	{0.033558, 0.060689, 0.105583, 0.146045, 0.197268, 0.246573, 0.299791, 0.350101, 0.402074, 0.451453, 0.500000, 0.545552, 0.589242, 0.629736, 0.667927, 0.70303, 0.73530, 0.76494, 0.79198, 0.81648, 0.83862, 0.85849, 0.87629, 0.89214, 0.90622},
	{0.024466, 0.047530, 0.085169, 0.120859, 0.166346, 0.211366, 0.260946, 0.308644, 0.358519, 0.406541, 0.454448, 0.500000, 0.544068, 0.585701, 0.625259, 0.66178, 0.69610, 0.72778, 0.75703, 0.78381, 0.80826, 0.83044, 0.85051, 0.86856, 0.88476},
	{0.020155, 0.037505, 0.069298, 0.099767, 0.140066, 0.180431, 0.225879, 0.270553, 0.317881, 0.364120, 0.410758, 0.455932, 0.500000, 0.541943, 0.582545, 0.62036, 0.65619, 0.68966, 0.72081, 0.74963, 0.77619, 0.80054, 0.82276, 0.84295, 0.86123},
	{0.014727, 0.029299, 0.055574, 0.081960, 0.117134, 0.153352, 0.194797, 0.236407, 0.281073, 0.325170, 0.370264, 0.414299, 0.458057, 0.500000, 0.540750, 0.57942, 0.61634, 0.65117, 0.68391, 0.71448, 0.74290, 0.76917, 0.79339, 0.81559, 0.83586},
	{0.012107, 0.023113, 0.045069, 0.067343, 0.097987, 0.130001, 0.167434, 0.205603, 0.247186, 0.288887, 0.332073, 0.374741, 0.417455, 0.459250, 0.500000, 0.53916, 0.57679, 0.61261, 0.64659, 0.67859, 0.70862, 0.73664, 0.76265, 0.78669, 0.80883},
	{0.00886, 0.01804, 0.03601, 0.05505, 0.08153, 0.10979, 0.14341, 0.17842, 0.21699, 0.25629, 0.29697, 0.33822, 0.37964, 0.42058, 0.46084, 0.50000, 0.53796, 0.57441, 0.60929, 0.64241, 0.67376, 0.70323, 0.73084, 0.75657, 0.78046},
	{0.00727, 0.01420, 0.02907, 0.04501, 0.06777, 0.09244, 0.12239, 0.15422, 0.18963, 0.22625, 0.26470, 0.30390, 0.34381, 0.38366, 0.42321, 0.46204, 0.50000, 0.53676, 0.57222, 0.60618, 0.63856, 0.66925, 0.69822, 0.72542, 0.75087},
	{0.00533, 0.01107, 0.02313, 0.03659, 0.05603, 0.07754, 0.10409, 0.13286, 0.16517, 0.19907, 0.23506, 0.27222, 0.31034, 0.34883, 0.38739, 0.42559, 0.46324, 0.50000, 0.53574, 0.57023, 0.60336, 0.63501, 0.66510, 0.69356, 0.72038},
	{0.00437, 0.00871, 0.01861, 0.02979, 0.04633, 0.06492, 0.08829, 0.11411, 0.14338, 0.17457, 0.20802, 0.24297, 0.27919, 0.31609, 0.35341, 0.39071, 0.42778, 0.46426, 0.50000, 0.53475, 0.56838, 0.60073, 0.63171, 0.66122, 0.68921},
	{0.00321, 0.00678, 0.01478, 0.02411, 0.03811, 0.05417, 0.07465, 0.09770, 0.12409, 0.15259, 0.18352, 0.21619, 0.25037, 0.28552, 0.32141, 0.35759, 0.39382, 0.42977, 0.46525, 0.50000, 0.53387, 0.56667, 0.59830, 0.62864, 0.65760},
	{0.00263, 0.00534, 0.01186, 0.01956, 0.03136, 0.04512, 0.06298, 0.08342, 0.10706, 0.13297, 0.16138, 0.19174, 0.22381, 0.25710, 0.29138, 0.32624, 0.36144, 0.39664, 0.43162, 0.46613, 0.50000, 0.53303, 0.56508, 0.59603, 0.62576},
	{0.00193, 0.00415, 0.00938, 0.01578, 0.02568, 0.03746, 0.05297, 0.07102, 0.09209, 0.11552, 0.14151, 0.16956, 0.19946, 0.23083, 0.26336, 0.29677, 0.33075, 0.36499, 0.39927, 0.43333, 0.46697, 0.50000, 0.53226, 0.56360, 0.59391},
	{0.00158, 0.00325, 0.00752, 0.01274, 0.02104, 0.03106, 0.04447, 0.06032, 0.07902, 0.10009, 0.12371, 0.14949, 0.17724, 0.20661, 0.23735, 0.26916, 0.30178, 0.33490, 0.36829, 0.40170, 0.43492, 0.46774, 0.50000, 0.53153, 0.56221},
	{0.00116, 0.00254, 0.00593, 0.01025, 0.01717, 0.02568, 0.03724, 0.05109, 0.06760, 0.08647, 0.10786, 0.13144, 0.15705, 0.18441, 0.21331, 0.24343, 0.27458, 0.30644, 0.33878, 0.37136, 0.40397, 0.43640, 0.46847, 0.50000, 0.53086},
	{0.00095, 0.00198, 0.00473, 0.00826, 0.01400, 0.02121, 0.03113, 0.04318, 0.05770, 0.07450, 0.09378, 0.11524, 0.13877, 0.16414, 0.19117, 0.21954, 0.24913, 0.27962, 0.31079, 0.34240, 0.37424, 0.40609, 0.43779, 0.46914, 0.50000},
}

// kazarossXG2PostCrawford is the Kazaross-XG2 post-Crawford MET (24 entries, indices 0-23).
// Index i = trailer's MWC when trailer needs i+1 pts and leader needs 1 pt.
// Entry for 1-away (index 0) = 0.5 (the leader wins with probability 1 minus this).
// GNUbg copies entries 0..nLength-2 from the XML, then extends using the Zadeh formula.
var kazarossXG2PostCrawford = [24]float32{
	0.500000, 0.48803, 0.32264, 0.31002, 0.19012, 0.18072,
	0.11559, 0.10906, 0.06953, 0.065161, 0.042069, 0.039060,
	0.025371, 0.023428, 0.015304, 0.014050, 0.009240, 0.008420,
	0.005560, 0.005050, 0.003360, 0.003030, 0.002030, 0.001820,
}

func init() {
	gnuBGInitPostCrawfordMET()
	gnuBGInitPreCrawfordMET()
	// Overlay Kazaross-XG2 values (GNUbg's default MET) onto the Zadeh-computed tables.
	// For matches ≤ 25 points this gives exact GNUbg values; beyond 25 uses Zadeh as fallback.
	gnuBGOverlayKazarossXG2()
}

// gnuBGOverlayKazarossXG2 overlays the Kazaross-XG2 explicit table values onto
// the Zadeh-computed MET arrays. GNUbg's default is Kazaross-XG2 (met/Kazaross-XG2.xml),
// not Zadeh. The explicit table covers matches up to 25 points; beyond that,
// the Zadeh values remain as a reasonable fallback.
func gnuBGOverlayKazarossXG2() {
	// Pre-Crawford: copy 25×25 explicit values
	for i := 0; i < 25; i++ {
		for j := 0; j < 25; j++ {
			gnuBGPreCrawfordMET[i][j] = kazarossXG2PreCrawford[i][j]
		}
	}

	// Post-Crawford: copy 24 explicit entries (GNUbg copies 0..nLength-2 = 0..23)
	for i := 0; i < 24; i++ {
		gnuBGPostCrawfordMET[i] = kazarossXG2PostCrawford[i]
	}

	// Re-compute 1-away row/column using the new post-Crawford values
	// This matches GNUbg's InitMatchEquity behavior where the pre-Crawford
	// 1-away entries are independent of the Zadeh computation.
	// Not needed since they are already set from the explicit table.
}

// gnuBGGetMETEntry returns gnuBGPreCrawfordMET[i][j] with boundary handling.
// Mirrors the GET_MET macro: i<0 → 1.0, j<0 → 0.0.
func gnuBGGetMETEntry(i, j int) float32 {
	if i < 0 {
		return 1.0
	}
	if j < 0 {
		return 0.0
	}
	return gnuBGPreCrawfordMET[i][j]
}

// gnuBGGetCubePrimeValue mirrors GetCubePrimeValue from matchequity.c.
// Returns 2*nCubeValue if automatic double applies, otherwise nCubeValue.
func gnuBGGetCubePrimeValue(i, j, nCubeValue int) int {
	if i < 2*nCubeValue && j >= 2*nCubeValue {
		return 2 * nCubeValue
	}
	return nCubeValue
}

// gnuBGInitPostCrawfordMET computes the post-Crawford MET using Zadeh's formula.
// Default parameters: gammon-rate-trailer=0.25, free-drop-2-away=0.015, free-drop-4-away=0.004.
func gnuBGInitPostCrawfordMET() {
	rG := float32(0.25)
	rFD2 := float32(0.015)
	rFD4 := float32(0.004)

	for i := 0; i < gnuBGMaxScore; i++ {
		pc4 := float32(1.0)
		if i-4 >= 0 {
			pc4 = gnuBGPostCrawfordMET[i-4]
		}
		pc2 := float32(1.0)
		if i-2 >= 0 {
			pc2 = gnuBGPostCrawfordMET[i-2]
		}
		gnuBGPostCrawfordMET[i] = rG*0.5*pc4 + (1.0-rG)*0.5*pc2

		if i == 1 {
			gnuBGPostCrawfordMET[i] -= rFD2
		}
		if i == 3 {
			gnuBGPostCrawfordMET[i] -= rFD4
		}
	}
}

// gnuBGInitPreCrawfordMET computes the pre-Crawford MET using Zadeh's formula.
// Default parameters: gammon-rate-leader=0.25, gammon-rate-trailer=0.15, delta=0.08, deltabar=0.06.
// This is a faithful translation of initMETZadeh() from GNUbg's matchequity.c.
func gnuBGInitPreCrawfordMET() {
	rG1 := float32(0.25)
	rG2 := float32(0.15)
	rDelta := float32(0.08)
	rDeltaBar := float32(0.06)

	pc := &gnuBGPostCrawfordMET
	met := &gnuBGPreCrawfordMET
	getMET := gnuBGGetMETEntry
	getCPV := gnuBGGetCubePrimeValue

	// Heap-allocate cube efficiency arrays
	d1 := new(d3Array)
	d2 := new(d3Array)
	d1bar := new(d3Array)
	d2bar := new(d3Array)

	// 1-away, n-away match equities (Crawford game row/column)
	for i := 0; i < gnuBGMaxScore; i++ {
		pcI2 := float32(1.0)
		if i-2 >= 0 {
			pcI2 = pc[i-2]
		}
		pcI1 := float32(1.0)
		if i-1 >= 0 {
			pcI1 = pc[i-1]
		}
		met[i][0] = rG1*0.5*pcI2 + (1.0-rG1)*0.5*pcI1
		met[0][i] = 1.0 - met[i][0]
	}

	// Fill the rest of the MET using Zadeh's iterative cube-adjusted formula
	for i := 0; i < gnuBGMaxScore; i++ {
		for j := 0; j <= i; j++ {
			for nCube := gnuBGMaxCubeLevel - 1; nCube >= 0; nCube-- {
				nCubeValue := 1 << nCube

				// --- D1bar ---
				nCPV := getCPV(i, j, nCubeValue)
				num := getMET(i-nCubeValue, j) -
					rG2*getMET(i, j-4*nCPV) -
					(1.0-rG2)*getMET(i, j-2*nCPV)
				den := rG1*getMET(i-4*nCPV, j) +
					(1.0-rG1)*getMET(i-2*nCPV, j) -
					rG2*getMET(i, j-4*nCPV) -
					(1.0-rG2)*getMET(i, j-2*nCPV)
				d1bar[i][j][nCube] = num / den

				if i != j {
					nCPV2 := getCPV(j, i, nCubeValue)
					numJI := getMET(j-nCubeValue, i) -
						rG2*getMET(j, i-4*nCPV2) -
						(1.0-rG2)*getMET(j, i-2*nCPV2)
					denJI := rG1*getMET(j-4*nCPV2, i) +
						(1.0-rG1)*getMET(j-2*nCPV2, i) -
						rG2*getMET(j, i-4*nCPV2) -
						(1.0-rG2)*getMET(j, i-2*nCPV2)
					d1bar[j][i][nCube] = numJI / denJI
				}

				// --- D2bar ---
				nCPV = getCPV(j, i, nCubeValue)
				num = getMET(j-nCubeValue, i) -
					rG2*getMET(j, i-4*nCPV) -
					(1.0-rG2)*getMET(j, i-2*nCPV)
				den = rG1*getMET(j-4*nCPV, i) +
					(1.0-rG1)*getMET(j-2*nCPV, i) -
					rG2*getMET(j, i-4*nCPV) -
					(1.0-rG2)*getMET(j, i-2*nCPV)
				d2bar[i][j][nCube] = num / den

				if i != j {
					nCPV2 := getCPV(i, j, nCubeValue)
					numJI := getMET(i-nCubeValue, j) -
						rG2*getMET(i, j-4*nCPV2) -
						(1.0-rG2)*getMET(i, j-2*nCPV2)
					denJI := rG1*getMET(i-4*nCPV2, j) +
						(1.0-rG1)*getMET(i-2*nCPV2, j) -
						rG2*getMET(i, j-4*nCPV2) -
						(1.0-rG2)*getMET(i, j-2*nCPV2)
					d2bar[j][i][nCube] = numJI / denJI
				}

				// --- D1 (cube efficiency adjusted) ---
				if i < 2*nCubeValue || j < 2*nCubeValue {
					d1[i][j][nCube] = d1bar[i][j][nCube]
					if i != j {
						d1[j][i][nCube] = d1bar[j][i][nCube]
					}
				} else {
					d1[i][j][nCube] = 1.0 + (d2[i][j][nCube+1]+rDelta)*(d1bar[i][j][nCube]-1.0)
					if i != j {
						d1[j][i][nCube] = 1.0 + (d2[j][i][nCube+1]+rDelta)*(d1bar[j][i][nCube]-1.0)
					}
				}

				// --- D2 (cube efficiency adjusted) ---
				if i < 2*nCubeValue || j < 2*nCubeValue {
					d2[i][j][nCube] = d2bar[i][j][nCube]
					if i != j {
						d2[j][i][nCube] = d2bar[j][i][nCube]
					}
				} else {
					d2[i][j][nCube] = 1.0 + (d1[i][j][nCube+1]+rDelta)*(d2bar[i][j][nCube]-1.0)
					if i != j {
						d2[j][i][nCube] = 1.0 + (d1[j][i][nCube+1]+rDelta)*(d2bar[j][i][nCube]-1.0)
					}
				}

				// --- Compute MET entry at cube level 0 ---
				if nCube == 0 && i > 0 && j > 0 {
					met[i][j] = ((d2[i][j][0]+rDeltaBar-0.5)*getMET(i-1, j) +
						(d1[i][j][0]+rDeltaBar-0.5)*getMET(i, j-1)) /
						(d1[i][j][0] + rDeltaBar + d2[i][j][0] + rDeltaBar - 1.0)
					if i != j {
						met[j][i] = 1.0 - met[i][j]
					}
				}
			}
		}
	}
}

// gnuBGGetME mirrors GNUbg's getME() function from matchequity.c.
// Returns the match winning chance from fPlayer's perspective after fWhoWins
// wins nPoints from the current match state.
//
// Parameters mirror the C function:
//   - score0, score1: current match scores for player 0 and player 1
//   - matchTo: match length
//   - fPlayer: whose perspective (0 or 1) to return MWC for
//   - nPoints: points won (typically cube value)
//   - fWhoWins: which player wins (0 or 1)
//   - fCrawford: whether the current game is Crawford
func gnuBGGetME(score0, score1, matchTo, fPlayer, nPoints, fWhoWins int, fCrawford bool) float64 {
	// Compute post-game "away" scores (0-indexed: n=0 means 1-away)
	notWhoWins := 0
	if fWhoWins == 0 {
		notWhoWins = 1
	}
	n0 := matchTo - (score0 + notWhoWins*nPoints) - 1
	n1 := matchTo - (score1 + fWhoWins*nPoints) - 1

	// Check if either player has won the match
	if n0 < 0 {
		// Player 0 has won
		if fPlayer != 0 {
			return 0.0
		}
		return 1.0
	}
	if n1 < 0 {
		// Player 1 has won
		if fPlayer != 0 {
			return 1.0
		}
		return 0.0
	}

	// Crawford / post-Crawford handling
	if fCrawford || matchTo-score0 == 1 || matchTo-score1 == 1 {
		if n0 == 0 {
			// Player 0 at 1-away after game
			if fPlayer != 0 {
				return float64(gnuBGPostCrawfordMET[n1])
			}
			return float64(1.0 - gnuBGPostCrawfordMET[n1])
		}
		// Player 1 must be at or near match point
		if fPlayer != 0 {
			return float64(1.0 - gnuBGPostCrawfordMET[n0])
		}
		return float64(gnuBGPostCrawfordMET[n0])
	}

	// Normal pre-Crawford lookup
	if fPlayer != 0 {
		return float64(1.0 - gnuBGPreCrawfordMET[n0][n1])
	}
	return float64(gnuBGPreCrawfordMET[n0][n1])
}

// convertGnuBGCubeMWCToEMG converts GNUbg cubeful equity values from Match Winning
// Chances (MWC, 0.0-1.0 scale) to Equivalent Money Game equity (EMG) for match play.
//
// This is a faithful implementation of GNUbg's mwc2eq() function from eval.c.
// The conversion uses the match equity table to compute MWC reference points:
//
//	rMwcWin  = getME(score0, score1, matchTo, fMove, cube, fMove, ...)    // MWC if "I" win
//	rMwcLose = getME(score0, score1, matchTo, fMove, cube, !fMove, ...)   // MWC if "I" lose
//	EMG = (2*MWC - (rMwcWin + rMwcLose)) / (rMwcWin - rMwcLose)
//
// This linear mapping ensures EMG=+1 when MWC=rMwcWin and EMG=-1 when MWC=rMwcLose.
//
// Parameters:
//   - score0, score1: absolute match scores for player 0 and player 1
//   - fMove: which player (0 or 1) is making the cube decision
//   - cubeValue: current cube value
//   - matchLength: match length (0 for money game)
func convertGnuBGCubeMWCToEMG(analysis *gnubgparser.CubeAnalysis, score0, score1, fMove, cubeValue, matchLength int) {
	if matchLength <= 0 || analysis == nil {
		return // Money game or nil analysis — no conversion needed
	}

	// For cube decisions, the game is never Crawford (cube is dead in Crawford games)
	fCrawford := false

	// MWC reference points: what happens if I win/lose the current game at this cube level
	// Use float32 throughout to match GNUbg's C float arithmetic exactly.
	// GNUbg reads DA values as float (via "(float) g_ascii_strtod"), stores MET as float,
	// and computes mwc2eq entirely in float. Using float64 would introduce subtle differences
	// that can flip BestAction at decision boundaries.
	mwcWin := float32(gnuBGGetME(score0, score1, matchLength, fMove, cubeValue, fMove, fCrawford))
	mwcLose := float32(gnuBGGetME(score0, score1, matchLength, fMove, cubeValue, 1-fMove, fCrawford))

	denom := mwcWin - mwcLose
	if denom < 1e-7 && denom > -1e-7 {
		// Degenerate case (e.g. dead cube) — keep raw values, set DP=1.0
		analysis.CubefulDoublePass = 1.0
		return
	}

	sum := mwcWin + mwcLose

	// Truncate input MWC to float32 to match GNUbg's "(float) g_ascii_strtod" behavior
	ndMwc := float32(analysis.CubefulNoDouble)
	dtMwc := float32(analysis.CubefulDoubleTake)

	// Convert ND and DT from MWC to EMG using GNUbg's mwc2eq formula (in float32)
	analysis.CubefulNoDouble = float64((2.0*ndMwc - sum) / denom)
	analysis.CubefulDoubleTake = float64((2.0*dtMwc - sum) / denom)
	analysis.CubefulDoublePass = 1.0 // DP is always 1.0 in EMG (by definition of the mapping)

	// Recompute BestAction with the converted EMG values
	effectiveDouble := analysis.CubefulDoubleTake
	if analysis.CubefulDoublePass < analysis.CubefulDoubleTake {
		effectiveDouble = analysis.CubefulDoublePass
	}
	if effectiveDouble > analysis.CubefulNoDouble {
		if analysis.CubefulDoubleTake <= analysis.CubefulDoublePass {
			analysis.BestAction = "Double, Take"
		} else {
			analysis.BestAction = "Double, Pass"
		}
	} else {
		analysis.BestAction = "No Double"
	}
}

// saveGnuBGCubeMoveAnalysisInTx saves gnubgparser CubeAnalysis to the move_analysis table
func (d *Database) saveGnuBGCubeMoveAnalysisInTx(tx *sql.Tx, moveID int64, analysis *gnubgparser.CubeAnalysis) error {
	player1WinRate := float64(analysis.Player1WinRate) * 100.0
	player2WinRate := float64(analysis.Player2WinRate) * 100.0

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "cube", translateGnuBGAnalysisDepth(analysis.AnalysisDepth),
		analysis.CubefulNoDouble, 0.0,
		player1WinRate, float64(analysis.Player1GammonRate)*100.0, float64(analysis.Player1BackgammonRate)*100.0,
		player2WinRate, float64(analysis.Player2GammonRate)*100.0, float64(analysis.Player2BackgammonRate)*100.0)

	return err
}

// saveGnuBGCheckerAnalysisToPositionInTx converts gnubgparser MoveAnalysis to PositionAnalysis and saves it
// isSGF indicates SGF format where Move[8]int is in absolute coordinates and MoveString is in letter format
func (d *Database) saveGnuBGCheckerAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analysis *gnubgparser.MoveAnalysis, player int, playedMoveStr string, isSGF bool) error {
	if analysis == nil || len(analysis.Moves) == 0 {
		return nil
	}

	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "GNU Backgammon",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	// Build checker moves list
	checkerMoves := make([]CheckerMove, 0, len(analysis.Moves))
	for i, moveOpt := range analysis.Moves {
		var moveStr string
		if isSGF {
			// For SGF, always convert from Move[8]int (absolute coords) to numeric notation
			moveStr = convertGnuBGMoveToString(moveOpt.Move, player)
		} else {
			// For MAT/TXT, use the existing human-readable MoveString
			moveStr = moveOpt.MoveString
			if moveStr == "" {
				// Move[8]int is in player-relative coordinates for MAT/TXT
				moveStr = convertPlayerRelativeMoveToString(moveOpt.Move)
			}
		}

		var equityError *float64
		if i > 0 {
			diff := analysis.Moves[0].Equity - moveOpt.Equity
			equityError = &diff
		}

		checkerMove := CheckerMove{
			Index:                    i,
			AnalysisDepth:            translateGnuBGAnalysisDepth(moveOpt.AnalysisDepth),
			AnalysisEngine:           "GNUbg",
			Move:                     moveStr,
			Equity:                   moveOpt.Equity,
			EquityError:              equityError,
			PlayerWinChance:          float64(moveOpt.Player1WinRate) * 100.0,
			PlayerGammonChance:       float64(moveOpt.Player1GammonRate) * 100.0,
			PlayerBackgammonChance:   float64(moveOpt.Player1BackgammonRate) * 100.0,
			OpponentWinChance:        float64(moveOpt.Player2WinRate) * 100.0,
			OpponentGammonChance:     float64(moveOpt.Player2GammonRate) * 100.0,
			OpponentBackgammonChance: float64(moveOpt.Player2BackgammonRate) * 100.0,
		}
		checkerMoves = append(checkerMoves, checkerMove)
	}

	posAnalysis.CheckerAnalysis = &CheckerAnalysis{
		Moves: checkerMoves,
	}

	// Set played move
	if playedMoveStr != "" {
		posAnalysis.PlayedMoves = []string{playedMoveStr}
	}

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveGnuBGCubeAnalysisToPositionInTx converts gnubgparser CubeAnalysis to PositionAnalysis and saves it
func (d *Database) saveGnuBGCubeAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analysis *gnubgparser.CubeAnalysis, playedCubeAction string) error {
	if analysis == nil {
		return nil
	}

	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "GNU Backgammon",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubefulNoDouble := analysis.CubefulNoDouble
	cubefulDoubleTake := analysis.CubefulDoubleTake
	cubefulDoublePass := analysis.CubefulDoublePass

	// Calculate best equity considering opponent's optimal response
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateGnuBGAnalysisDepth(analysis.AnalysisDepth),
		AnalysisEngine:            "GNUbg",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BackgammonRate) * 100.0,
		OpponentWinChances:        float64(analysis.Player2WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BackgammonRate) * 100.0,
		CubelessNoDoubleEquity:    analysis.CubelessEquity,
		CubelessDoubleEquity:      analysis.CubelessEquity,
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Set played cube action
	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveGnuBGCubeAnalysisForCheckerPositionInTx saves cube analysis to a checker position
// This allows displaying cube info when pressing 'd' on a checker decision
func (d *Database) saveGnuBGCubeAnalysisForCheckerPositionInTx(tx *sql.Tx, positionID int64, analysis *gnubgparser.CubeAnalysis) error {
	if analysis == nil {
		return nil
	}

	// Build a PositionAnalysis with just the cube analysis and let saveAnalysisInTx handle merging
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "GNU Backgammon",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubefulNoDouble := analysis.CubefulNoDouble
	cubefulDoubleTake := analysis.CubefulDoubleTake
	cubefulDoublePass := analysis.CubefulDoublePass

	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateGnuBGAnalysisDepth(analysis.AnalysisDepth),
		AnalysisEngine:            "GNUbg",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BackgammonRate) * 100.0,
		OpponentWinChances:        float64(analysis.Player2WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BackgammonRate) * 100.0,
		CubelessNoDoubleEquity:    analysis.CubelessEquity,
		CubelessDoubleEquity:      analysis.CubelessEquity,
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// translateGnuBGAnalysisDepth converts gnuBG analysis depth to a human-readable string
// gnuBG uses ply levels: 0=0-ply (contact/race evaluation), 1=1-ply, 2=2-ply, etc.
func translateGnuBGAnalysisDepth(depth int) string {
	if depth >= 0 {
		return fmt.Sprintf("%d-ply", depth)
	}
	return fmt.Sprintf("%d", depth)
}

// ComputeGnuBGMatchHash generates a unique hash for a gnubgparser match
// Used to detect duplicate imports
func ComputeGnuBGMatchHash(match *gnubgparser.Match) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, match.Metadata.MatchLength))

	// Include full game transcription
	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|",
			gameIdx, game.Score[0], game.Score[1], game.Winner, game.Points))

		// Include all moves in the game
		for moveIdx, moveRec := range game.Moves {
			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, string(moveRec.Type)))

			if moveRec.Type == "move" {
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%s|",
					moveRec.Dice[0], moveRec.Dice[1], moveRec.MoveString))
			} else if moveRec.Type == "double" || moveRec.Type == "take" || moveRec.Type == "drop" {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", string(moveRec.Type)))
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// maxCanonicalDicePerGame limits how many dice per game are included in the canonical hash.
// Different file formats (XG, SGF, MAT) handle end-of-game dice differently:
// XG records game-ending rolls that SGF/MAT may omit, and MAT can diverge
// in later moves due to parser edge cases. Using only the first N dice avoids
// these differences while providing strong match identification.
// 10 dice per game * 21 possible outcomes * 7+ games = astronomically low collision probability.
const maxCanonicalDicePerGame = 10

// ComputeCanonicalMatchHashFromXG computes a format-independent match hash from XG data.
// This hash uses only the first N dice per game (physical events identical across all export
// formats) plus normalized player names, match length, and game count, making it identical
// whether the match was imported from XG, GnuBG (SGF/MAT), or BGBlitz (BGF) format.
func ComputeCanonicalMatchHashFromXG(match *xgparser.Match) string {
	var hashBuilder strings.Builder

	// Normalized player names (sorted alphabetically for consistency)
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2Name))
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	hashBuilder.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, match.Metadata.MatchLength, len(match.Games)))

	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d|", gameIdx))
		diceCount := 0
		for _, move := range game.Moves {
			if diceCount >= maxCanonicalDicePerGame {
				break
			}
			if move.MoveType == "checker" && move.CheckerMove != nil {
				d1 := move.CheckerMove.Dice[0]
				d2 := move.CheckerMove.Dice[1]
				if d1 > d2 {
					d1, d2 = d2, d1
				}
				hashBuilder.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
				diceCount++
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// ComputeCanonicalMatchHashFromGnuBG computes a format-independent match hash from GnuBG data.
// Must produce the same hash as ComputeCanonicalMatchHashFromXG for the same match.
// Uses only the first N dice per game for cross-format compatibility.
func ComputeCanonicalMatchHashFromGnuBG(match *gnubgparser.Match) string {
	var hashBuilder strings.Builder

	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2))
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	hashBuilder.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, match.Metadata.MatchLength, len(match.Games)))

	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d|", gameIdx))
		diceCount := 0
		for _, moveRec := range game.Moves {
			if diceCount >= maxCanonicalDicePerGame {
				break
			}
			if moveRec.Type == "move" {
				d1 := moveRec.Dice[0]
				d2 := moveRec.Dice[1]
				if d1 > d2 {
					d1, d2 = d2, d1
				}
				hashBuilder.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
				diceCount++
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// checkCanonicalMatchExistsLocked checks if a match with the given canonical hash already exists
// Returns the existing match ID if found, 0 otherwise
func (d *Database) checkCanonicalMatchExistsLocked(canonicalHash string) (int64, error) {
	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM match WHERE canonical_hash = ?`, canonicalHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error checking for canonical match: %w", err)
	}
	return existingID, nil
}

// ============================================================================
// BGBlitz BGF import functions
// ============================================================================

// ImportBGFMatch imports a match from a BGBlitz BGF file using the bgfparser library.
// BGF files contain full match data including moves, analysis, and cube decisions.
func (d *Database) ImportBGFMatch(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Parse the BGF file
	bgfMatch, err := bgfparser.ParseBGF(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGF file: %w", err)
	}

	if bgfMatch.Data == nil {
		return 0, fmt.Errorf("BGF file contains no match data")
	}

	data := bgfMatch.Data

	// Extract match metadata
	nameGreen := bgfGetString(data, "nameGreen")
	nameRed := bgfGetString(data, "nameRed")
	matchLen := bgfGetInt(data, "matchlen")
	event := bgfGetString(data, "event")
	location := bgfGetString(data, "location")
	round := bgfGetString(data, "round")
	dateStr := bgfGetString(data, "date")

	// Parse match date
	var matchDate time.Time
	if dateStr != "" {
		for _, layout := range []string{
			"Jan 2, 2006",
			"2006-01-02 15:04:05",
			"2006-01-02",
			"January 2, 2006",
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, dateStr); err == nil {
				matchDate = t
				break
			}
		}
	}
	if matchDate.IsZero() {
		matchDate = time.Now()
	}

	// Extract games
	gamesData, ok := data["games"].([]interface{})
	if !ok || len(gamesData) == 0 {
		return 0, fmt.Errorf("BGF file contains no games")
	}

	// Compute match hash for duplicate detection
	matchHash := ComputeBGFMatchHash(bgfMatch)

	// Compute canonical hash (format-independent) for cross-format duplicate detection
	canonicalHash := ComputeCanonicalMatchHashFromBGF(bgfMatch)

	// Check if this exact match already exists (same format)
	existingMatchID, err := d.checkMatchExistsLocked(matchHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for duplicate match: %w", err)
	}
	if existingMatchID > 0 {
		return 0, ErrDuplicateMatch
	}

	// Check if same match was imported from a different format (canonical duplicate)
	canonicalMatchID, err := d.checkCanonicalMatchExistsLocked(canonicalHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for canonical duplicate: %w", err)
	}
	isCanonicalDuplicate := canonicalMatchID > 0

	// Begin transaction for atomic import
	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Insert match metadata or reuse existing canonical match
	var matchID int64
	if isCanonicalDuplicate {
		matchID = canonicalMatchID
		fmt.Printf("Canonical duplicate detected - reusing match ID %d, importing new analysis only\n", matchID)
	} else {
		result, err := tx.Exec(`
			INSERT INTO match (player1_name, player2_name, event, location, round,
			                   match_length, match_date, file_path, game_count, match_hash, canonical_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, nameGreen, nameRed, event, location, round,
			matchLen, matchDate, filePath, len(gamesData), matchHash, canonicalHash)
		if err != nil {
			return 0, fmt.Errorf("failed to insert match: %w", err)
		}

		matchID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get match ID: %w", err)
		}

		// Auto-link tournament from event metadata
		eventName := strings.TrimSpace(event)
		if eventName != "" {
			var tournamentID int64
			err2 := tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, eventName).Scan(&tournamentID)
			if err2 != nil {
				res2, err3 := tx.Exec(`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, eventName)
				if err3 == nil {
					tournamentID, err = res2.LastInsertId()
					if err != nil {
						return 0, fmt.Errorf("failed to get last insert ID: %w", err)
					}
				}
			}
			if tournamentID > 0 {
				_, err = tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID)
				if err != nil {
					fmt.Printf("Warning: failed to link match to tournament: %v\n", err)
				}
			}
		}
	}

	// Build a position cache for deduplication
	positionCache := make(map[string]int64)
	semanticCache := make(map[string]int64) // map[semanticKey]positionID

	// Load existing positions into cache
	existingRows, err := tx.Query(`SELECT id, state FROM position`)
	if err != nil {
		return 0, fmt.Errorf("failed to load existing positions: %w", err)
	}

	for existingRows.Next() {
		var existingID int64
		var existingStateJSON string
		if err := existingRows.Scan(&existingID, &existingStateJSON); err != nil {
			continue
		}
		var existingPosition Position
		if err := json.Unmarshal([]byte(existingStateJSON), &existingPosition); err != nil {
			continue
		}
		normalizedPosition := existingPosition.NormalizeForStorage()
		normalizedPosition.ID = 0
		normalizedJSON, err := json.Marshal(normalizedPosition)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		positionCache[string(normalizedJSON)] = existingID
		semanticCache[positionSemanticKey(&normalizedPosition)] = existingID
	}
	if err := existingRows.Err(); err != nil {
		return 0, err
	}
	existingRows.Close()

	fmt.Printf("Loaded %d existing positions into cache for BGF import\n", len(positionCache))

	// Process each game
	for gameIdx, gameRaw := range gamesData {
		gameData, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract game metadata
		scoreGreen := bgfGetInt(gameData, "scoreGreen")
		scoreRed := bgfGetInt(gameData, "scoreRed")
		isCrawford := bgfGetBool(gameData, "isCrawford")
		wonPoints := bgfGetInt(gameData, "wonPoints")

		// Get moves
		movesData, ok := gameData["moves"].([]interface{})
		if !ok {
			continue
		}

		// Determine game winner from wonPoints and final positions
		// Winner is determined by who won the game
		winner := int32(0) // Will be computed from match final scores

		if !isCanonicalDuplicate {
			// Insert game record
			gameResult, err := tx.Exec(`
				INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2,
				                  winner, points_won, move_count)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, matchID, gameIdx+1, scoreGreen, scoreRed,
				winner, wonPoints, len(movesData))
			if err != nil {
				return 0, fmt.Errorf("failed to insert game %d: %w", gameIdx+1, err)
			}

			gameID, err := gameResult.LastInsertId()
			if err != nil {
				return 0, fmt.Errorf("failed to get game ID: %w", err)
			}

			// Process moves - build board state as we go
			boardState := bgfInitBoardFromGame(gameData)
			cubeValue := 1
			cubeOwner := -1 // center

			moveNumber := int32(0)
			pendingCubeDouble := false // tracks if previous move was a cube double encoded as amove
			for moveIdx, moveRaw := range movesData {
				moveData, ok := moveRaw.(map[string]interface{})
				if !ok {
					continue
				}

				mtype := bgfGetString(moveData, "type")
				player := bgfGetInt(moveData, "player") // -1 = Green, 1 = Red

				switch mtype {
				case "amove":
					// Check if this is a cube action encoded as amove
					// (BGBlitz uses amove with from=[-1,-1,-1,-1] and green=7 for cube actions)
					fromArr := bgfGetIntArray(moveData, "from")
					if fromArr[0] == -1 {
						if pendingCubeDouble {
							// This is the response to a pending cube double (take/pass)
							pendingCubeDouble = false
							equity := bgfGetMap(moveData, "equity")
							if equity != nil {
								cd := bgfGetMap(equity, "cubeDecision")
								if cd != nil && bgfGetBool(cd, "hasAccepted") {
									// Take - update cube state
									cubeValue *= 2
									if player == -1 {
										cubeOwner = 0 // Green takes
									} else {
										cubeOwner = 1 // Red takes
									}
								}
								// Pass: no cube state update needed (game ends)
							}
							// Don't increment moveNumber (response is part of the double action)
							continue
						}

						// This is a cube double
						pendingCubeDouble = true
						cubeAction := "Double/Take" // default
						// Look ahead for the response
						for j := moveIdx + 1; j < len(movesData); j++ {
							nextMove, ok := movesData[j].(map[string]interface{})
							if !ok {
								continue
							}
							nextFrom := bgfGetIntArray(nextMove, "from")
							if nextFrom[0] == -1 {
								eq := bgfGetMap(nextMove, "equity")
								if eq != nil {
									cd := bgfGetMap(eq, "cubeDecision")
									if cd != nil && !bgfGetBool(cd, "hasAccepted") {
										cubeAction = "Double/Pass"
									}
								}
								break
							}
							break
						}

						err := d.importBGFCubeMove(tx, gameID, moveNumber, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, positionCache, cubeAction)
						if err != nil {
							fmt.Printf("Warning: failed to import BGF cube move in game %d: %v\n", gameIdx+1, err)
						}
						moveNumber++
						continue
					}

					// Normal checker move
					err := d.importBGFCheckerMove(tx, gameID, moveNumber, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, positionCache)
					if err != nil {
						fmt.Printf("Warning: failed to import BGF move %d in game %d: %v\n", moveIdx, gameIdx+1, err)
					}
					// Update board state - skip when green=7 (unplayable die marker)
					// BGBlitz uses green=7 to indicate that one die couldn't be played.
					// The from/to data in these moves represents analysis recommendations,
					// not actual game moves, so applying them corrupts the board state.
					greenDie := bgfGetInt(moveData, "green")
					if greenDie != 7 {
						bgfApplyCheckerMove(&boardState, moveData, player)
					}
					moveNumber++

				case "adouble":
					// Cube double - find the response (take/pass)
					cubeAction := "Double/Pass"
					for j := moveIdx + 1; j < len(movesData); j++ {
						nextMove, ok := movesData[j].(map[string]interface{})
						if !ok {
							continue
						}
						nextType := bgfGetString(nextMove, "type")
						if nextType == "atake" {
							cubeAction = "Double/Take"
							break
						} else if nextType == "apass" {
							cubeAction = "Double/Pass"
							break
						} else if nextType != "amove" {
							break
						}
					}

					err := d.importBGFCubeMove(tx, gameID, moveNumber, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, positionCache, cubeAction)
					if err != nil {
						fmt.Printf("Warning: failed to import BGF cube move in game %d: %v\n", gameIdx+1, err)
					}
					moveNumber++

				case "atake":
					// Take - update cube state
					if cubeValue == 1 {
						cubeValue = 2
					} else {
						cubeValue *= 2
					}
					// The taker becomes the cube owner
					if player == -1 {
						cubeOwner = 0 // Green
					} else {
						cubeOwner = 1 // Red
					}
					// Don't increment moveNumber (part of the double action)

				case "apass":
					// Pass/Drop - game ends, don't need to update state
					// Don't increment moveNumber

				default:
					// Skip unknown move types
					continue
				}
			}
		} else {
			// Canonical duplicate: only import analysis to existing positions
			boardState := bgfInitBoardFromGame(gameData)
			cubeValue := 1
			cubeOwner := -1
			pendingCubeDouble2 := false

			for moveIdx, moveRaw := range movesData {
				moveData, ok := moveRaw.(map[string]interface{})
				if !ok {
					continue
				}

				mtype := bgfGetString(moveData, "type")
				player := bgfGetInt(moveData, "player")

				switch mtype {
				case "amove":
					// Check if this is a cube action encoded as amove
					fromArr := bgfGetIntArray(moveData, "from")
					if fromArr[0] == -1 {
						if pendingCubeDouble2 {
							pendingCubeDouble2 = false
							equity := bgfGetMap(moveData, "equity")
							if equity != nil {
								cd := bgfGetMap(equity, "cubeDecision")
								if cd != nil && bgfGetBool(cd, "hasAccepted") {
									cubeValue *= 2
									if player == -1 {
										cubeOwner = 0
									} else {
										cubeOwner = 1
									}
								}
							}
							continue
						}
						pendingCubeDouble2 = true
						cubeAction := "Double/Take"
						for j := moveIdx + 1; j < len(movesData); j++ {
							nextMove, ok := movesData[j].(map[string]interface{})
							if !ok {
								continue
							}
							nextFrom := bgfGetIntArray(nextMove, "from")
							if nextFrom[0] == -1 {
								eq := bgfGetMap(nextMove, "equity")
								if eq != nil {
									cd := bgfGetMap(eq, "cubeDecision")
									if cd != nil && !bgfGetBool(cd, "hasAccepted") {
										cubeAction = "Double/Pass"
									}
								}
								break
							}
							break
						}
						d.importBGFCubeAnalysisOnly(tx, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, positionCache, semanticCache, cubeAction)
						continue
					}

					d.importBGFCheckerAnalysisOnly(tx, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, positionCache, semanticCache)
					// Skip board update for green=7 moves (unplayable die - analysis data only)
					greenDie := bgfGetInt(moveData, "green")
					if greenDie != 7 {
						bgfApplyCheckerMove(&boardState, moveData, player)
					}

				case "adouble":
					cubeAction := "Double/Pass"
					for j := moveIdx + 1; j < len(movesData); j++ {
						nextMove, ok := movesData[j].(map[string]interface{})
						if !ok {
							continue
						}
						nextType := bgfGetString(nextMove, "type")
						if nextType == "atake" {
							cubeAction = "Double/Take"
							break
						} else if nextType == "apass" {
							cubeAction = "Double/Pass"
							break
						}
					}
					d.importBGFCubeAnalysisOnly(tx, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, positionCache, semanticCache, cubeAction)

				case "atake":
					if cubeValue == 1 {
						cubeValue = 2
					} else {
						cubeValue *= 2
					}
					if player == -1 {
						cubeOwner = 0
					} else {
						cubeOwner = 1
					}
				}
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully imported BGF match %d with %d games from %s\n", matchID, len(gamesData), filePath)
	return matchID, nil
}

// importBGFCheckerMove imports a single checker move from a BGF file
func (d *Database) importBGFCheckerMove(tx *sql.Tx, gameID int64, moveNumber int32, moveData map[string]interface{}, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue int, cubeOwner int, isCrawford bool, positionCache map[string]int64) error {
	player := bgfGetInt(moveData, "player") // -1 = Green, 1 = Red

	// Convert BGF player to blunderDB player encoding
	// BGF: -1 = Green (first player), 1 = Red (second player)
	// blunderDB: 0 = Player 1 (Green/Black), 1 = Player 2 (Red/White)
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	// Get dice
	dieGreen := bgfGetInt(moveData, "green")
	dieRed := bgfGetInt(moveData, "red")
	die1 := dieGreen
	die2 := dieRed

	// Handle impossible dice values (green=7 appears in BGBlitz for cube actions)
	// Cube actions with from[0]==-1 are now handled in the main loop, so this
	// should only fire for edge cases. Try to infer dice from the from/to arrays.
	if die1 > 6 || die2 > 6 || die1 < 1 || die2 < 1 {
		fromArr := bgfGetIntArray(moveData, "from")
		toArr := bgfGetIntArray(moveData, "to")
		if fromArr[0] == -1 {
			// No checker move at all - skip
			return nil
		}
		// Infer dice from the move sub-moves
		var diceUsed []int
		for j := 0; j < 4; j++ {
			if fromArr[j] == -1 {
				break
			}
			f := fromArr[j]
			t := toArr[j]
			if f == 25 {
				// From bar: die = destination point
				diceUsed = append(diceUsed, t)
			} else if t == 0 {
				// Bear off: die >= distance from point to off
				diceUsed = append(diceUsed, f)
			} else {
				diff := f - t
				if diff < 0 {
					diff = -diff
				}
				diceUsed = append(diceUsed, diff)
			}
		}
		if len(diceUsed) >= 2 {
			die1 = diceUsed[0]
			die2 = diceUsed[1]
		} else if len(diceUsed) == 1 {
			// Single sub-move: one die was used, the other couldn't be played
			if die1 >= 1 && die1 <= 6 {
				die2 = diceUsed[0]
			} else if die2 >= 1 && die2 <= 6 {
				die1 = diceUsed[0]
			} else {
				die1 = diceUsed[0]
				die2 = diceUsed[0] // best guess
			}
		}
	}

	// Create board position from current state
	pos := d.createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = CheckerAction
	pos.Dice = [2]int{die1, die2}

	// Save position
	posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
	if err != nil {
		return fmt.Errorf("failed to save position: %w", err)
	}

	// Convert move to string notation
	checkerMoveStr := bgfConvertMoveToString(moveData, player)

	// Convert player to XG-style encoding for DB storage consistency
	dbPlayer := convertBlunderDBPlayerToXG(blunderDBPlayer)

	// Save move record
	moveResult, err := tx.Exec(`
		INSERT INTO move (game_id, move_number, move_type, position_id, player,
		                  dice_1, dice_2, checker_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, moveNumber, "checker", posID, dbPlayer, die1, die2, checkerMoveStr)
	if err != nil {
		return err
	}

	moveID, err := moveResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Save analysis if available
	moveAnalysis, ok := moveData["moveAnalysis"].([]interface{})
	if ok && len(moveAnalysis) > 0 {
		// Save to move_analysis table (first/played move)
		for _, maRaw := range moveAnalysis {
			maData, ok := maRaw.(map[string]interface{})
			if !ok {
				continue
			}
			if bgfGetBool(maData, "played") {
				err = d.saveBGFMoveAnalysisInTx(tx, moveID, maData)
				if err != nil {
					fmt.Printf("Warning: failed to save BGF move analysis: %v\n", err)
				}
				break // Only save the played move to move_analysis
			}
		}

		// Save to position analysis table (all moves for UI compatibility)
		err = d.saveBGFCheckerAnalysisToPositionInTx(tx, posID, moveAnalysis, blunderDBPlayer, checkerMoveStr)
		if err != nil {
			fmt.Printf("Warning: failed to save BGF position analysis: %v\n", err)
		}
	}

	// Save cube analysis from the equity field if present on a checker move
	equity := bgfGetMap(moveData, "equity")
	if equity != nil {
		cubeDecision := bgfGetMap(equity, "cubeDecision")
		if cubeDecision != nil {
			stateOnMove := bgfGetString(cubeDecision, "stateOnMove")
			if stateOnMove != "" {
				err = d.saveBGFCubeAnalysisForCheckerPositionInTx(tx, posID, equity, cubeDecision)
				if err != nil {
					fmt.Printf("Warning: failed to save cube analysis for checker position: %v\n", err)
				}
			}
		}
	}

	return nil
}

// importBGFCubeMove imports a cube double/take/pass move from a BGF file
func (d *Database) importBGFCubeMove(tx *sql.Tx, gameID int64, moveNumber int32, moveData map[string]interface{}, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue int, cubeOwner int, isCrawford bool, positionCache map[string]int64, cubeAction string) error {
	player := bgfGetInt(moveData, "player")
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	// Create position from current board state
	pos := d.createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = CubeAction
	pos.Dice = [2]int{0, 0}

	// Save position
	posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
	if err != nil {
		return fmt.Errorf("failed to save position: %w", err)
	}

	// Convert player to XG-style encoding for DB storage consistency
	dbPlayer := convertBlunderDBPlayerToXG(blunderDBPlayer)

	// Save move record
	moveResult, err := tx.Exec(`
		INSERT INTO move (game_id, move_number, move_type, position_id, player,
		                  dice_1, dice_2, cube_action)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, moveNumber, "cube", posID, dbPlayer, 0, 0, cubeAction)
	if err != nil {
		return err
	}

	moveID, err := moveResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Save cube analysis from equity field
	equity := bgfGetMap(moveData, "equity")
	if equity != nil {
		cubeDecision := bgfGetMap(equity, "cubeDecision")
		if cubeDecision != nil {
			err = d.saveBGFCubeMoveAnalysisInTx(tx, moveID, equity, cubeDecision)
			if err != nil {
				fmt.Printf("Warning: failed to save BGF cube analysis: %v\n", err)
			}

			err = d.saveBGFCubeAnalysisToPositionInTx(tx, posID, equity, cubeDecision, cubeAction)
			if err != nil {
				fmt.Printf("Warning: failed to save BGF position cube analysis: %v\n", err)
			}
		}
	}

	return nil
}

// importBGFCheckerAnalysisOnly imports only analysis for a canonical duplicate
func (d *Database) importBGFCheckerAnalysisOnly(tx *sql.Tx, moveData map[string]interface{}, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue int, cubeOwner int, isCrawford bool, positionCache map[string]int64, semanticCache map[string]int64) {
	player := bgfGetInt(moveData, "player")
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	dieGreen := bgfGetInt(moveData, "green")
	dieRed := bgfGetInt(moveData, "red")

	pos := d.createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = CheckerAction
	pos.Dice = [2]int{dieGreen, dieRed}

	posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, positionCache, semanticCache)
	if err != nil {
		return
	}

	moveAnalysis, ok := moveData["moveAnalysis"].([]interface{})
	if ok && len(moveAnalysis) > 0 {
		checkerMoveStr := bgfConvertMoveToString(moveData, player)
		d.saveBGFCheckerAnalysisToPositionInTx(tx, posID, moveAnalysis, blunderDBPlayer, checkerMoveStr)
	}

	equity := bgfGetMap(moveData, "equity")
	if equity != nil {
		cubeDecision := bgfGetMap(equity, "cubeDecision")
		if cubeDecision != nil && bgfGetString(cubeDecision, "stateOnMove") != "" {
			d.saveBGFCubeAnalysisForCheckerPositionInTx(tx, posID, equity, cubeDecision)
		}
	}
}

// importBGFCubeAnalysisOnly imports only cube analysis for a canonical duplicate
func (d *Database) importBGFCubeAnalysisOnly(tx *sql.Tx, moveData map[string]interface{}, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue int, cubeOwner int, isCrawford bool, positionCache map[string]int64, semanticCache map[string]int64, cubeAction string) {
	player := bgfGetInt(moveData, "player")
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	pos := d.createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = CubeAction
	pos.Dice = [2]int{0, 0}

	posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, positionCache, semanticCache)
	if err != nil {
		return
	}

	equity := bgfGetMap(moveData, "equity")
	if equity != nil {
		cubeDecision := bgfGetMap(equity, "cubeDecision")
		if cubeDecision != nil {
			d.saveBGFCubeAnalysisToPositionInTx(tx, posID, equity, cubeDecision, cubeAction)
		}
	}
}

// createPositionFromBGF creates a blunderDB Position from BGF board state
func (d *Database) createPositionFromBGF(boardState [28]int, gameData map[string]interface{}, matchLen int, cubeValue int, cubeOwner int, isCrawford bool) *Position {
	scoreGreen := bgfGetInt(gameData, "scoreGreen")
	scoreRed := bgfGetInt(gameData, "scoreRed")

	// Calculate away scores (points away from winning)
	awayScore1 := matchLen - scoreGreen
	awayScore2 := matchLen - scoreRed

	if matchLen == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Convert cube value to exponent for blunderDB (2^n representation)
	cubeExponent := 0
	if cubeValue > 0 {
		for v := cubeValue; v > 1; v >>= 1 {
			cubeExponent++
		}
	}

	pos := &Position{
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
		Score:        [2]int{awayScore1, awayScore2},
		Cube: Cube{
			Value: cubeExponent,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0},
	}

	// Determine Jacoby/Beaver from match settings
	if matchLen == 0 {
		// Money game - check match-level settings
		// (Jacoby/Beaver are only relevant in money games)
	}

	// Initialize all points as empty
	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = Point{Checkers: 0, Color: -1}
	}

	// Convert BGF board encoding to blunderDB:
	// BGF board is indexed from Green's far side (24-point) to near side (1-point):
	//   BGF index 0 = Green's 24-point, BGF index 23 = Green's 1-point
	//   So: BGF index i corresponds to board point (24-i)
	// BGF index 24: Green's bar, index 25: Red's bar
	// BGF index 26: Green's borne off, index 27: Red's borne off
	// Positive = Green checkers, Negative = Red checkers
	//
	// blunderDB:
	// - Color 0 = Player 1 (Green) moves 24→1 (same as BGF Green)
	// - Color 1 = Player 2 (Red) moves 1→24
	// - Index 0 = Player 2's bar (Red/White), Index 25 = Player 1's bar (Green/Black)
	// - Index 1-24 = Points 1-24

	// Map board points (BGF index i → blunderDB point 24-i)
	for i := 0; i < 24; i++ {
		count := boardState[i]
		blunderDBPoint := 24 - i // BGF index 0 = point 24, index 23 = point 1
		if count > 0 {
			// Green checkers (positive)
			pos.Board.Points[blunderDBPoint] = Point{Checkers: count, Color: 0} // Color 0 = Green/Player 1
		} else if count < 0 {
			// Red checkers (negative)
			pos.Board.Points[blunderDBPoint] = Point{Checkers: -count, Color: 1} // Color 1 = Red/Player 2
		}
	}

	// Map bar: BGF index 24 = Green's bar → blunderDB index 25
	if boardState[24] > 0 {
		pos.Board.Points[25] = Point{Checkers: boardState[24], Color: 0}
	}
	// BGF index 25 = Red's bar → blunderDB index 0
	if boardState[25] < 0 {
		pos.Board.Points[0] = Point{Checkers: -boardState[25], Color: 1}
	} else if boardState[25] > 0 {
		// Red bar is stored as positive in some encodings
		pos.Board.Points[0] = Point{Checkers: boardState[25], Color: 1}
	}

	// Calculate bearoff
	player1Total := 0
	player2Total := 0
	for i := 0; i < 26; i++ {
		if pos.Board.Points[i].Color == 0 {
			player1Total += pos.Board.Points[i].Checkers
		} else if pos.Board.Points[i].Color == 1 {
			player2Total += pos.Board.Points[i].Checkers
		}
	}
	pos.Board.Bearoff = [2]int{15 - player1Total, 15 - player2Total}

	return pos
}

// bgfInitBoardFromGame extracts the initial board position from a BGF game
func bgfInitBoardFromGame(gameData map[string]interface{}) [28]int {
	var board [28]int

	initial, ok := gameData["initial"].(map[string]interface{})
	if !ok {
		// Return standard starting position
		board = [28]int{2, 0, 0, 0, 0, -5, 0, -3, 0, 0, 0, 5, -5, 0, 0, 0, 3, 0, 5, 0, 0, 0, 0, -2, 0, 0, 0, 0}
		return board
	}

	points, ok := initial["points"].([]interface{})
	if !ok || len(points) < 28 {
		board = [28]int{2, 0, 0, 0, 0, -5, 0, -3, 0, 0, 0, 5, -5, 0, 0, 0, 3, 0, 5, 0, 0, 0, 0, -2, 0, 0, 0, 0}
		return board
	}

	for i := 0; i < 28 && i < len(points); i++ {
		board[i] = bgfToInt(points[i])
	}

	return board
}

// bgfApplyCheckerMove updates the board state after a BGF checker move.
// BGF from/to use 1-based point numbering from the active player's perspective:
//   - Points 1-24 = board positions (1 = player's 1-point)
//   - 25 = bar (from only)
//   - 0 = bear off (to only)
//
// Board state uses 0-based Green's perspective: indices 0-23 = points 1-24,
// 24 = Green's bar, 25 = Red's bar, 26 = Green off, 27 = Red off.
func bgfApplyCheckerMove(boardState *[28]int, moveData map[string]interface{}, player int) {
	fromArr := bgfGetIntArray(moveData, "from")
	toArr := bgfGetIntArray(moveData, "to")

	for i := 0; i < 4; i++ {
		from := fromArr[i]
		to := toArr[i]
		if from == -1 {
			break // No more submoves
		}

		if player == -1 {
			// Green moves: Green's point N maps to board index (24-N)
			// BGF board: index 0 = Green's 24-point, index 23 = Green's 1-point
			// Green moves in decreasing direction (24→1→off)
			var fromIdx int
			if from == 25 {
				fromIdx = 24 // Green's bar
			} else {
				fromIdx = 24 - from // Green's point N → board index (24-N)
			}

			// Remove checker from source
			boardState[fromIdx]--

			if to == 0 {
				// Bear off
				boardState[26]++
			} else {
				toIdx := 24 - to // Green's point N → board index (24-N)
				// Check for hit
				if boardState[toIdx] < 0 {
					// Hit Red checker - move it to Red's bar
					boardState[25] += boardState[toIdx] // boardState[toIdx] is negative, so this decrements
					boardState[toIdx] = 0
				}
				boardState[toIdx]++
			}
		} else {
			// Red moves: Red's point N maps to board index (N-1)
			// Red's point N = Green's point (25-N) = board index 24-(25-N) = N-1
			// Red moves in increasing direction (from Green's perspective)
			var fromIdx int
			if from == 25 {
				fromIdx = 25 // Red's bar
			} else {
				fromIdx = from - 1 // Red's point N → board index (N-1)
			}

			// Remove checker from source (Red checkers are negative)
			boardState[fromIdx]++

			if to == 0 {
				// Bear off
				boardState[27]--
			} else {
				toIdx := to - 1 // Red's point N → board index (N-1)
				// Check for hit
				if boardState[toIdx] > 0 {
					// Hit Green checker - move it to Green's bar
					boardState[24] += boardState[toIdx]
					boardState[toIdx] = 0
				}
				boardState[toIdx]--
			}
		}
	}
}

// bgfConvertMoveToString converts BGF move from/to arrays to standard notation.
// BGF from/to are 1-based from the active player's perspective (25=bar, 0=off).
func bgfConvertMoveToString(moveData map[string]interface{}, player int) string {
	fromArr := bgfGetIntArray(moveData, "from")
	toArr := bgfGetIntArray(moveData, "to")

	if fromArr[0] == -1 {
		return "" // No move (fanned/dance)
	}

	type submove struct {
		from int
		to   int
	}

	moves := make([]submove, 0, 4)
	for i := 0; i < 4; i++ {
		if fromArr[i] == -1 {
			break
		}
		// from/to are already 1-based from player perspective
		from := fromArr[i]
		to := toArr[i]

		// from=25 means bar, keep as 25
		// to=0 means bear off, keep as 0

		moves = append(moves, submove{from, to})
	}

	if len(moves) == 0 {
		return ""
	}

	// Sort by source point descending
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].from > moves[j].from
	})

	// Group identical moves
	var parts []string
	i := 0
	for i < len(moves) {
		count := 1
		for i+count < len(moves) && moves[i+count].from == moves[i].from && moves[i+count].to == moves[i].to {
			count++
		}

		fromStr := fmt.Sprintf("%d", moves[i].from)
		if moves[i].from == 25 {
			fromStr = "bar"
		}

		toStr := fmt.Sprintf("%d", moves[i].to)
		if moves[i].to == 0 {
			toStr = "off"
		}

		if count > 1 {
			parts = append(parts, fmt.Sprintf("%s/%s(%d)", fromStr, toStr, count))
		} else {
			parts = append(parts, fmt.Sprintf("%s/%s", fromStr, toStr))
		}

		i += count
	}

	return strings.Join(parts, " ")
}

// saveBGFMoveAnalysisInTx saves BGF move analysis to the move_analysis table
func (d *Database) saveBGFMoveAnalysisInTx(tx *sql.Tx, moveID int64, maData map[string]interface{}) error {
	eq := bgfGetMap(maData, "eq")
	if eq == nil {
		return nil
	}

	ply := bgfGetInt(maData, "ply")

	playerWin := bgfGetFloat(eq, "myWins") * 100.0
	playerGammon := bgfGetFloat(eq, "myGammon") * 100.0
	playerBg := bgfGetFloat(eq, "myBackGammon") * 100.0
	opponentWin := bgfGetFloat(eq, "oppWins") * 100.0
	opponentGammon := bgfGetFloat(eq, "oppGammon") * 100.0
	opponentBg := bgfGetFloat(eq, "oppBackGammon") * 100.0

	equity := bgfGetFloat(eq, "emg")
	if !bgfGetBool(eq, "hasEMG") {
		equity = bgfGetFloat(eq, "equity")
	}

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "checker", translateBGFAnalysisDepth(ply),
		equity, 0.0,
		playerWin, playerGammon, playerBg,
		opponentWin, opponentGammon, opponentBg)

	return err
}

// saveBGFCheckerAnalysisToPositionInTx saves BGF checker analysis to position analysis table
func (d *Database) saveBGFCheckerAnalysisToPositionInTx(tx *sql.Tx, positionID int64, moveAnalysis []interface{}, blunderDBPlayer int, playedMoveStr string) error {
	if len(moveAnalysis) == 0 {
		return nil
	}

	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "BGBlitz",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	checkerMoves := make([]CheckerMove, 0, len(moveAnalysis))
	var bestEquity float64

	for i, maRaw := range moveAnalysis {
		maData, ok := maRaw.(map[string]interface{})
		if !ok {
			continue
		}

		eq := bgfGetMap(maData, "eq")
		if eq == nil {
			continue
		}

		ply := bgfGetInt(maData, "ply")
		played := bgfGetBool(maData, "played")

		equity := bgfGetFloat(eq, "emg")
		if !bgfGetBool(eq, "hasEMG") {
			equity = bgfGetFloat(eq, "equity")
		}

		if i == 0 {
			bestEquity = equity
		}

		// Convert move from analysis
		moveStr := ""
		moveInfo := bgfGetMap(maData, "move")
		if moveInfo != nil {
			moveStr = bgfConvertAnalysisMoveToString(moveInfo)
		}

		var equityError *float64
		if i > 0 {
			diff := bestEquity - equity
			equityError = &diff
		}

		checkerMove := CheckerMove{
			Index:                    i,
			AnalysisDepth:            translateBGFAnalysisDepth(ply),
			AnalysisEngine:           "BGBlitz",
			Move:                     moveStr,
			Equity:                   equity,
			EquityError:              equityError,
			PlayerWinChance:          bgfGetFloat(eq, "myWins") * 100.0,
			PlayerGammonChance:       bgfGetFloat(eq, "myGammon") * 100.0,
			PlayerBackgammonChance:   bgfGetFloat(eq, "myBackGammon") * 100.0,
			OpponentWinChance:        bgfGetFloat(eq, "oppWins") * 100.0,
			OpponentGammonChance:     bgfGetFloat(eq, "oppGammon") * 100.0,
			OpponentBackgammonChance: bgfGetFloat(eq, "oppBackGammon") * 100.0,
		}
		checkerMoves = append(checkerMoves, checkerMove)

		_ = played
	}

	posAnalysis.CheckerAnalysis = &CheckerAnalysis{
		Moves: checkerMoves,
	}

	if playedMoveStr != "" {
		posAnalysis.PlayedMoves = []string{playedMoveStr}
	}

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveBGFCubeMoveAnalysisInTx saves BGF cube analysis to move_analysis table
func (d *Database) saveBGFCubeMoveAnalysisInTx(tx *sql.Tx, moveID int64, equity map[string]interface{}, cubeDecision map[string]interface{}) error {
	playerWin := bgfGetFloat(equity, "myWins") * 100.0
	playerGammon := bgfGetFloat(equity, "myGammon") * 100.0
	playerBg := bgfGetFloat(equity, "myBackGammon") * 100.0
	opponentWin := bgfGetFloat(equity, "oppWins") * 100.0
	opponentGammon := bgfGetFloat(equity, "oppGammon") * 100.0
	opponentBg := bgfGetFloat(equity, "oppBackGammon") * 100.0

	eqNoDouble := bgfGetFloat(cubeDecision, "eqNoDouble")

	ply := 0 // Cube analysis doesn't have a ply field in BGF, use the move level
	if pVal, ok := equity["ply"]; ok {
		ply = bgfToInt(pVal)
	}

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "cube", translateBGFAnalysisDepth(ply),
		eqNoDouble, 0.0,
		playerWin, playerGammon, playerBg,
		opponentWin, opponentGammon, opponentBg)

	return err
}

// saveBGFCubeAnalysisToPositionInTx saves BGF cube analysis to position analysis table
func (d *Database) saveBGFCubeAnalysisToPositionInTx(tx *sql.Tx, positionID int64, equity map[string]interface{}, cubeDecision map[string]interface{}, playedCubeAction string) error {
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "BGBlitz",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubefulNoDouble := bgfGetFloat(cubeDecision, "eqNoDouble")
	cubefulDoubleTake := bgfGetFloat(cubeDecision, "eqDoubleTake")
	cubefulDoublePass := bgfGetFloat(cubeDecision, "eqDoublePass")

	cubelessEquity := bgfGetFloat(cubeDecision, "eqCubeLess")
	_ = bgfGetFloat(cubeDecision, "eqCubeFul") // cubefulEquity available but not directly used

	// Calculate best equity
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	// Also derive from BGF stateOnMove/stateOther
	stateOnMove := bgfGetString(cubeDecision, "stateOnMove")
	stateOther := bgfGetString(cubeDecision, "stateOther")
	if stateOnMove == "DOUBLE" || stateOnMove == "REDOUBLE" {
		if stateOther == "ACCEPT" {
			bestAction = "Double, Take"
		} else if stateOther == "REJECT" {
			bestAction = "Double, Pass"
		}
	} else if stateOnMove == "TO_GOOD" {
		bestAction = "No Double"
	} else if stateOnMove == "NO_DOUBLE" {
		bestAction = "No Double"
	}

	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             "2-ply", // BGBlitz default
		AnalysisEngine:            "BGBlitz",
		PlayerWinChances:          bgfGetFloat(equity, "myWins") * 100.0,
		PlayerGammonChances:       bgfGetFloat(equity, "myGammon") * 100.0,
		PlayerBackgammonChances:   bgfGetFloat(equity, "myBackGammon") * 100.0,
		OpponentWinChances:        bgfGetFloat(equity, "oppWins") * 100.0,
		OpponentGammonChances:     bgfGetFloat(equity, "oppGammon") * 100.0,
		OpponentBackgammonChances: bgfGetFloat(equity, "oppBackGammon") * 100.0,
		CubelessNoDoubleEquity:    cubelessEquity,
		CubelessDoubleEquity:      cubelessEquity,
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       0.0,
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveBGFCubeAnalysisForCheckerPositionInTx saves cube analysis from equity field to a checker position
func (d *Database) saveBGFCubeAnalysisForCheckerPositionInTx(tx *sql.Tx, positionID int64, equity map[string]interface{}, cubeDecision map[string]interface{}) error {
	// Try to load existing analysis for this position
	var existingAnalysisJSON string
	var existingID int64
	err := tx.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisJSON)

	var posAnalysis PositionAnalysis
	if err == nil && existingID > 0 {
		err = json.Unmarshal([]byte(existingAnalysisJSON), &posAnalysis)
		if err != nil {
			return err
		}
	} else {
		posAnalysis = PositionAnalysis{
			PositionID:            int(positionID),
			AnalysisType:          "CheckerMove",
			AnalysisEngineVersion: "BGBlitz",
			CreationDate:          time.Now(),
		}
	}

	posAnalysis.LastModifiedDate = time.Now()

	cubefulNoDouble := bgfGetFloat(cubeDecision, "eqNoDouble")
	cubefulDoubleTake := bgfGetFloat(cubeDecision, "eqDoubleTake")
	cubefulDoublePass := bgfGetFloat(cubeDecision, "eqDoublePass")
	cubelessEquity := bgfGetFloat(cubeDecision, "eqCubeLess")

	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             "2-ply",
		AnalysisEngine:            "BGBlitz",
		PlayerWinChances:          bgfGetFloat(equity, "myWins") * 100.0,
		PlayerGammonChances:       bgfGetFloat(equity, "myGammon") * 100.0,
		PlayerBackgammonChances:   bgfGetFloat(equity, "myBackGammon") * 100.0,
		OpponentWinChances:        bgfGetFloat(equity, "oppWins") * 100.0,
		OpponentGammonChances:     bgfGetFloat(equity, "oppGammon") * 100.0,
		OpponentBackgammonChances: bgfGetFloat(equity, "oppBackGammon") * 100.0,
		CubelessNoDoubleEquity:    cubelessEquity,
		CubelessDoubleEquity:      cubelessEquity,
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       0.0,
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// bgfConvertAnalysisMoveToString converts a BGF move from analysis entry to string notation
// bgfConvertAnalysisMoveToString converts a BGF move from analysis entry to string notation.
// BGF from/to are 1-based from the active player's perspective (25=bar, 0=off).
func bgfConvertAnalysisMoveToString(moveInfo map[string]interface{}) string {
	fromArr := bgfGetIntArray(moveInfo, "from")
	toArr := bgfGetIntArray(moveInfo, "to")

	if fromArr[0] == -1 {
		return ""
	}

	type submove struct {
		from int
		to   int
	}

	moves := make([]submove, 0, 4)
	for i := 0; i < 4; i++ {
		if fromArr[i] == -1 {
			break
		}
		// from/to are already 1-based from player perspective
		from := fromArr[i]
		to := toArr[i]

		moves = append(moves, submove{from, to})
	}

	if len(moves) == 0 {
		return ""
	}

	sort.Slice(moves, func(i, j int) bool {
		return moves[i].from > moves[j].from
	})

	var parts []string
	i := 0
	for i < len(moves) {
		count := 1
		for i+count < len(moves) && moves[i+count].from == moves[i].from && moves[i+count].to == moves[i].to {
			count++
		}

		fromStr := fmt.Sprintf("%d", moves[i].from)
		if moves[i].from == 25 {
			fromStr = "bar"
		}
		toStr := fmt.Sprintf("%d", moves[i].to)
		if moves[i].to == 0 {
			toStr = "off"
		}

		if count > 1 {
			parts = append(parts, fmt.Sprintf("%s/%s(%d)", fromStr, toStr, count))
		} else {
			parts = append(parts, fmt.Sprintf("%s/%s", fromStr, toStr))
		}
		i += count
	}

	return strings.Join(parts, " ")
}

// translateBGFAnalysisDepth converts BGF ply level to human-readable string
func translateBGFAnalysisDepth(ply int) string {
	if ply > 0 {
		return fmt.Sprintf("%d-ply", ply)
	}
	return "0-ply"
}

// ComputeBGFMatchHash generates a unique hash for a BGF match for duplicate detection
func ComputeBGFMatchHash(match *bgfparser.Match) string {
	var hashBuilder strings.Builder

	data := match.Data
	p1 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameGreen")))
	p2 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameRed")))
	matchLen := bgfGetInt(data, "matchlen")
	hashBuilder.WriteString(fmt.Sprintf("bgf:%s|%s|%d|", p1, p2, matchLen))

	gamesData, _ := data["games"].([]interface{})
	for gameIdx, gameRaw := range gamesData {
		g, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d|",
			gameIdx, bgfGetInt(g, "scoreGreen"), bgfGetInt(g, "scoreRed"), bgfGetInt(g, "wonPoints")))

		movesData, _ := g["moves"].([]interface{})
		for moveIdx, moveRaw := range movesData {
			m, ok := moveRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mtype := bgfGetString(m, "type")
			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, mtype))
			if mtype == "amove" {
				d1 := bgfGetInt(m, "green")
				d2 := bgfGetInt(m, "red")
				hashBuilder.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
			} else if mtype == "adouble" || mtype == "atake" || mtype == "apass" {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", mtype))
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// ComputeCanonicalMatchHashFromBGF computes a format-independent match hash from BGF data.
// Must produce the same hash as ComputeCanonicalMatchHashFromXG for the same match.
// Uses only the first N dice per game for cross-format compatibility.
func ComputeCanonicalMatchHashFromBGF(match *bgfparser.Match) string {
	var hashBuilder strings.Builder

	data := match.Data
	p1 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameGreen")))
	p2 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameRed")))
	matchLen := bgfGetInt(data, "matchlen")

	if p1 > p2 {
		p1, p2 = p2, p1
	}

	gamesData, _ := data["games"].([]interface{})
	hashBuilder.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, matchLen, len(gamesData)))

	for gameIdx, gameRaw := range gamesData {
		g, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		hashBuilder.WriteString(fmt.Sprintf("g%d|", gameIdx))

		diceCount := 0
		movesData, _ := g["moves"].([]interface{})
		for _, moveRaw := range movesData {
			if diceCount >= maxCanonicalDicePerGame {
				break
			}
			m, ok := moveRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mtype := bgfGetString(m, "type")
			if mtype == "amove" {
				// Skip cube actions encoded as amove (from[0] == -1)
				fromArr := bgfGetIntArray(m, "from")
				if len(fromArr) > 0 && fromArr[0] == -1 {
					continue
				}
				d1 := bgfGetInt(m, "green")
				d2 := bgfGetInt(m, "red")
				if d1 > d2 {
					d1, d2 = d2, d1
				}
				hashBuilder.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
				diceCount++
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// ImportBGFPosition imports a single BGBlitz position from a TXT file
func (d *Database) ImportBGFPosition(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pos, err := bgfparser.ParseTXT(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGBlitz position file: %w", err)
	}

	return d.saveBGFPositionWithAnalysis(pos)
}

// ImportBGFPositionFromText imports a BGBlitz position from text content (clipboard/string)
func (d *Database) ImportBGFPositionFromText(content string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pos, err := bgfparser.ParseTXTFromReader(strings.NewReader(content))
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGBlitz position text: %w", err)
	}

	return d.saveBGFPositionWithAnalysis(pos)
}

// saveBGFPositionWithAnalysis converts a bgfparser.Position to blunderDB Position and saves it
func (d *Database) saveBGFPositionWithAnalysis(bgfPos *bgfparser.Position) (int64, error) {
	// Convert bgfparser.Position to blunderDB Position
	pos := d.convertBGFTextPosition(bgfPos)

	// Save position to database (inline, since caller already holds the mutex)
	normalizedPosition := pos.NormalizeForStorage()
	positionJSON, err := json.Marshal(normalizedPosition)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal position: %w", err)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
	if err != nil {
		return 0, fmt.Errorf("failed to insert position: %w", err)
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get position ID: %w", err)
	}

	normalizedPosition.ID = positionID
	positionJSON, err = json.Marshal(normalizedPosition)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal position with ID: %w", err)
	}
	_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSON), positionID)
	if err != nil {
		return 0, fmt.Errorf("failed to update position state: %w", err)
	}

	// Save checker evaluation analysis if available
	if len(bgfPos.Evaluations) > 0 {
		posAnalysis := PositionAnalysis{
			PositionID:            int(positionID),
			XGID:                  bgfPos.XGID,
			Player1:               bgfPos.PlayerX,
			Player2:               bgfPos.PlayerO,
			AnalysisType:          "CheckerMove",
			AnalysisEngineVersion: "BGBlitz",
			CreationDate:          time.Now(),
			LastModifiedDate:      time.Now(),
		}

		checkerMoves := make([]CheckerMove, 0, len(bgfPos.Evaluations))
		for i, eval := range bgfPos.Evaluations {
			var equityError *float64
			if i > 0 {
				diff := bgfPos.Evaluations[0].Equity - eval.Equity
				equityError = &diff
			}

			checkerMove := CheckerMove{
				Index:                    i,
				AnalysisDepth:            "2-ply", // BGBlitz TXT files don't specify ply, default to 2-ply
				AnalysisEngine:           "BGBlitz",
				Move:                     eval.Move,
				Equity:                   eval.Equity,
				EquityError:              equityError,
				PlayerWinChance:          eval.Win * 100.0,
				PlayerGammonChance:       eval.WinG * 100.0,
				PlayerBackgammonChance:   eval.WinBG * 100.0,
				OpponentWinChance:        (1.0 - eval.Win) * 100.0,
				OpponentGammonChance:     eval.LoseG * 100.0,
				OpponentBackgammonChance: eval.LoseBG * 100.0,
			}
			checkerMoves = append(checkerMoves, checkerMove)
		}

		posAnalysis.CheckerAnalysis = &CheckerAnalysis{
			Moves: checkerMoves,
		}

		analysisJSON, err := json.Marshal(posAnalysis)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal checker analysis: %w", err)
		}
		_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, string(analysisJSON))
		if err != nil {
			return 0, fmt.Errorf("failed to save checker analysis for BGBlitz position: %w", err)
		}
	}

	// Save cube decision analysis if available
	if len(bgfPos.CubeDecisions) > 0 {
		posAnalysis := PositionAnalysis{
			PositionID:            int(positionID),
			XGID:                  bgfPos.XGID,
			Player1:               bgfPos.PlayerX,
			Player2:               bgfPos.PlayerO,
			AnalysisType:          "DoublingCube",
			AnalysisEngineVersion: "BGBlitz",
			CreationDate:          time.Now(),
			LastModifiedDate:      time.Now(),
		}

		// Find the best action using multilingual classifier
		var noDouble, doubleTake, doublePass *bgfparser.CubeDecision
		for i := range bgfPos.CubeDecisions {
			cd := &bgfPos.CubeDecisions[i]
			switch classifyBGFCubeAction(cd.Action) {
			case "nodbl":
				noDouble = cd
			case "take":
				doubleTake = cd
			case "pass":
				doublePass = cd
			}
		}

		cubefulNoDouble := 0.0
		cubefulDoubleTake := 0.0
		cubefulDoublePass := 1.0 // Default pass equity

		if noDouble != nil {
			cubefulNoDouble = noDouble.EMG
		}
		if doubleTake != nil {
			cubefulDoubleTake = doubleTake.EMG
		}
		if doublePass != nil {
			cubefulDoublePass = doublePass.EMG
		}

		effectiveDoubleEquity := cubefulDoubleTake
		if cubefulDoublePass < cubefulDoubleTake {
			effectiveDoubleEquity = cubefulDoublePass
		}

		bestEquity := cubefulNoDouble
		bestAction := "No Double"
		if effectiveDoubleEquity > cubefulNoDouble {
			bestEquity = effectiveDoubleEquity
			if cubefulDoubleTake <= cubefulDoublePass {
				bestAction = "Double, Take"
			} else {
				bestAction = "Double, Pass"
			}
		}

		cubeAnalysis := DoublingCubeAnalysis{
			AnalysisDepth:           "2-ply",
			AnalysisEngine:          "BGBlitz",
			CubelessNoDoubleEquity:  bgfPos.CubelessEquity,
			CubelessDoubleEquity:    bgfPos.CubelessEquity,
			CubefulNoDoubleEquity:   cubefulNoDouble,
			CubefulNoDoubleError:    cubefulNoDouble - bestEquity,
			CubefulDoubleTakeEquity: cubefulDoubleTake,
			CubefulDoubleTakeError:  cubefulDoubleTake - bestEquity,
			CubefulDoublePassEquity: cubefulDoublePass,
			CubefulDoublePassError:  cubefulDoublePass - bestEquity,
			BestCubeAction:          bestAction,
		}

		posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

		analysisJSON, err := json.Marshal(posAnalysis)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal cube analysis: %w", err)
		}
		_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, string(analysisJSON))
		if err != nil {
			return 0, fmt.Errorf("failed to save cube analysis for BGBlitz position: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit position with analysis: %w", err)
	}
	return positionID, nil
}

// convertBGFTextPosition converts a bgfparser.Position from TXT format to blunderDB Position
func (d *Database) convertBGFTextPosition(bgfPos *bgfparser.Position) *Position {
	pos := &Position{
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}

	// Convert board from bgfparser encoding to blunderDB
	// bgfparser Board[26]:
	//   Index 0: (unused or bar-like)
	//   Index 1-24: Points 1-24 (positive=X/Green, negative=O/Red)
	//   Index 25: (unused or bar-like)
	// blunderDB:
	//   Index 0: Player 2's bar (Red/White)
	//   Index 1-24: Points 1-24
	//   Index 25: Player 1's bar (Green/Black)

	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = Point{Checkers: 0, Color: -1}
	}

	// Map points 1-24
	for i := 1; i <= 24; i++ {
		count := bgfPos.Board[i]
		if count > 0 {
			pos.Board.Points[i] = Point{Checkers: count, Color: 0} // Green = Color 0
		} else if count < 0 {
			pos.Board.Points[i] = Point{Checkers: -count, Color: 1} // Red = Color 1
		}
	}

	// Map bars from OnBar map
	if bgfPos.OnBar != nil {
		if xBar, ok := bgfPos.OnBar["X"]; ok && xBar > 0 {
			pos.Board.Points[25] = Point{Checkers: xBar, Color: 0} // Green bar
		}
		if oBar, ok := bgfPos.OnBar["O"]; ok && oBar > 0 {
			pos.Board.Points[0] = Point{Checkers: oBar, Color: 1} // Red bar
		}
	}

	// Calculate bearoff
	player1Total := 0
	player2Total := 0
	for i := 0; i < 26; i++ {
		if pos.Board.Points[i].Color == 0 {
			player1Total += pos.Board.Points[i].Checkers
		} else if pos.Board.Points[i].Color == 1 {
			player2Total += pos.Board.Points[i].Checkers
		}
	}
	pos.Board.Bearoff = [2]int{15 - player1Total, 15 - player2Total}

	// Set player on roll
	if bgfPos.OnRoll == "O" {
		pos.PlayerOnRoll = 1
	} else {
		pos.PlayerOnRoll = 0
	}

	// Set dice
	pos.Dice = [2]int{bgfPos.Dice[0], bgfPos.Dice[1]}

	// Set cube
	cubeExponent := 0
	if bgfPos.CubeValue > 0 {
		for v := bgfPos.CubeValue; v > 1; v >>= 1 {
			cubeExponent++
		}
	}
	pos.Cube.Value = cubeExponent

	switch bgfPos.CubeOwner {
	case "X":
		pos.Cube.Owner = 0 // Green owns
	case "O":
		pos.Cube.Owner = 1 // Red owns
	default:
		pos.Cube.Owner = -1 // Center
	}

	// Set scores (away scores)
	if bgfPos.MatchLength > 0 {
		pos.Score = [2]int{bgfPos.MatchLength - bgfPos.ScoreX, bgfPos.MatchLength - bgfPos.ScoreO}
	} else {
		pos.Score = [2]int{-1, -1} // Unlimited
	}

	// Decision type based on available analysis
	if len(bgfPos.CubeDecisions) > 0 && len(bgfPos.Evaluations) == 0 {
		pos.DecisionType = CubeAction
		pos.Dice = [2]int{0, 0}
	}

	return pos
}

// ============================================================================
// BGF helper functions for extracting typed values from map[string]interface{}
// ============================================================================

// bgfPlayerToBlunderDB converts BGF player encoding to blunderDB encoding
// BGF: -1 = Green (first player), 1 = Red (second player)
// blunderDB: 0 = Player 1 (Green/Black), 1 = Player 2 (Red/White)
func bgfPlayerToBlunderDB(bgfPlayer int) int {
	if bgfPlayer == -1 {
		return 0 // Green = Player 1
	}
	return 1 // Red = Player 2
}

func bgfGetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// classifyBGFCubeAction classifies a cube decision action string into one of
// "nodbl" (No Double / No Redouble), "take" (Double/Take, Redouble/Take),
// or "pass" (Double/Pass, Redouble/Pass).
// Handles multilingual action strings from BGBlitz text export (EN, FR, DE, JP, etc.).
func classifyBGFCubeAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))

	// No Double / No Redouble patterns
	noDoublePatterns := []string{
		"no double", "no redouble", // English
		"pas de double", "pas de redouble", // French
		"kein doppel", "kein redoppel", // German
		"\u30c0\u30d6\u30eb\u305b\u305a", // Japanese: ダブルせず
	}
	for _, p := range noDoublePatterns {
		if strings.Contains(action, p) {
			return "nodbl"
		}
	}

	// Double/Take patterns (check before pass since "take" is more specific)
	takePatterns := []string{
		"take", "accept", // English
		"prendre", "accepter", // French
		"annehmen",           // German
		"\u53d7\u3051\u308b", // Japanese: 受ける
	}
	for _, p := range takePatterns {
		if strings.Contains(action, p) {
			return "take"
		}
	}

	// Double/Pass patterns
	passPatterns := []string{
		"pass", "reject", "decline", // English
		"refuser",            // French
		"ablehnen",           // German
		"\u964d\u308a\u308b", // Japanese: 降りる
	}
	for _, p := range passPatterns {
		if strings.Contains(action, p) {
			return "pass"
		}
	}

	// Fallback: if the action contains a separator ("/"), it's likely take or pass.
	// If no separator, it's likely no double.
	if !strings.Contains(action, "/") {
		return "nodbl"
	}

	return "unknown"
}

func bgfGetInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		return bgfToInt(v)
	}
	return 0
}

func bgfGetFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		return bgfToFloat(v)
	}
	return 0.0
}

func bgfGetBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func bgfGetMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]interface{}); ok {
			return sub
		}
	}
	return nil
}

func bgfGetIntArray(m map[string]interface{}, key string) [4]int {
	var result [4]int
	for i := range result {
		result[i] = -1
	}
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			for i := 0; i < 4 && i < len(arr); i++ {
				result[i] = bgfToInt(arr[i])
			}
		}
	}
	return result
}

func bgfToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	}
	return 0
}

func bgfToFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0.0
}

// GetAllMatches returns all matches from the database
func (d *Database) GetAllMatches() ([]Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT m.id, m.player1_name, m.player2_name, m.event, m.location, m.round, 
		       m.match_length, m.match_date, m.import_date, m.file_path, m.game_count,
		       m.tournament_id, COALESCE(t.name, '') as tournament_name,
		       COALESCE(m.last_visited_position, -1) as last_visited_position
		FROM match m
		LEFT JOIN tournament t ON m.tournament_id = t.id
		ORDER BY CASE WHEN m.match_date IS NULL OR m.match_date = '' OR m.match_date = '0001-01-01T00:00:00Z' THEN m.import_date ELSE m.match_date END DESC
	`)
	if err != nil {
		fmt.Println("Error loading matches:", err)
		return nil, err
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		err := rows.Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
			&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount,
			&m.TournamentID, &m.TournamentName, &m.LastVisitedPosition)
		if err != nil {
			fmt.Println("Error scanning match:", err)
			continue
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}

// GetMatchByID returns a specific match by ID
func (d *Database) GetMatchByID(matchID int64) (*Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var m Match
	err := d.db.QueryRow(`
		SELECT id, player1_name, player2_name, event, location, round,
		       match_length, match_date, import_date, file_path, game_count,
		       COALESCE(last_visited_position, -1) as last_visited_position
		FROM match
		WHERE id = ?
	`, matchID).Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
		&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount, &m.LastVisitedPosition)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("match not found")
		}
		fmt.Println("Error loading match:", err)
		return nil, err
	}

	return &m, nil
}

// SaveLastVisitedPosition saves the last visited position index for a match
func (d *Database) SaveLastVisitedPosition(matchID int64, positionIndex int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`UPDATE match SET last_visited_position = ? WHERE id = ?`, positionIndex, matchID)
	if err != nil {
		fmt.Println("Error saving last visited position:", err)
		return err
	}
	return nil
}

// GetLastVisitedMatch returns the most recently visited match (match with highest last_visited_position != -1)
// If no match has been visited, returns the most recent match (first in date order)
func (d *Database) GetLastVisitedMatch() (*Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var m Match
	// First try to find a match that has been visited (last_visited_position >= 0)
	err := d.db.QueryRow(`
		SELECT m.id, m.player1_name, m.player2_name, m.event, m.location, m.round,
		       m.match_length, m.match_date, m.import_date, m.file_path, m.game_count,
		       m.tournament_id, COALESCE(t.name, '') as tournament_name,
		       COALESCE(m.last_visited_position, -1) as last_visited_position
		FROM match m
		LEFT JOIN tournament t ON m.tournament_id = t.id
		WHERE m.last_visited_position >= 0
		ORDER BY m.import_date DESC
		LIMIT 1
	`).Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
		&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount,
		&m.TournamentID, &m.TournamentName, &m.LastVisitedPosition)

	if err == nil {
		return &m, nil
	}

	if err != sql.ErrNoRows {
		fmt.Println("Error finding last visited match:", err)
		return nil, err
	}

	// No visited match found, return the most recent match
	err = d.db.QueryRow(`
		SELECT m.id, m.player1_name, m.player2_name, m.event, m.location, m.round,
		       m.match_length, m.match_date, m.import_date, m.file_path, m.game_count,
		       m.tournament_id, COALESCE(t.name, '') as tournament_name,
		       COALESCE(m.last_visited_position, -1) as last_visited_position
		FROM match m
		LEFT JOIN tournament t ON m.tournament_id = t.id
		ORDER BY CASE WHEN m.match_date IS NULL OR m.match_date = '' OR m.match_date = '0001-01-01T00:00:00Z' THEN m.import_date ELSE m.match_date END DESC
		LIMIT 1
	`).Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
		&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount,
		&m.TournamentID, &m.TournamentName, &m.LastVisitedPosition)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no matches in database")
		}
		fmt.Println("Error finding most recent match:", err)
		return nil, err
	}

	return &m, nil
}

// GetGamesByMatch returns all games in a match
func (d *Database) GetGamesByMatch(matchID int64) ([]Game, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT id, match_id, game_number, initial_score_1, initial_score_2,
		       winner, points_won, move_count
		FROM game
		WHERE match_id = ?
		ORDER BY game_number ASC
	`, matchID)
	if err != nil {
		fmt.Println("Error loading games:", err)
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		var score1, score2 int32
		err := rows.Scan(&g.ID, &g.MatchID, &g.GameNumber, &score1, &score2,
			&g.Winner, &g.PointsWon, &g.MoveCount)
		if err != nil {
			fmt.Println("Error scanning game:", err)
			continue
		}
		g.InitialScore = [2]int32{score1, score2}
		games = append(games, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return games, nil
}

// GetMovesByGame returns all moves in a game
func (d *Database) GetMovesByGame(gameID int64) ([]Move, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT id, game_id, move_number, move_type, position_id, player,
		       dice_1, dice_2, checker_move, cube_action
		FROM move
		WHERE game_id = ?
		ORDER BY move_number ASC
	`, gameID)
	if err != nil {
		fmt.Println("Error loading moves:", err)
		return nil, err
	}
	defer rows.Close()

	var moves []Move
	for rows.Next() {
		var m Move
		var dice1, dice2 int32
		var checkerMove, cubeAction sql.NullString
		err := rows.Scan(&m.ID, &m.GameID, &m.MoveNumber, &m.MoveType, &m.PositionID,
			&m.Player, &dice1, &dice2, &checkerMove, &cubeAction)
		if err != nil {
			fmt.Println("Error scanning move:", err)
			continue
		}
		m.Dice = [2]int32{dice1, dice2}
		if checkerMove.Valid {
			m.CheckerMove = checkerMove.String
		}
		if cubeAction.Valid {
			m.CubeAction = cubeAction.String
		}
		moves = append(moves, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return moves, nil
}

// DeleteMatch deletes a match and all associated games, moves, and analysis
func (d *Database) DeleteMatch(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Collect position IDs referenced by this match's moves before cascade delete
	rows, err := tx.Query(`
		SELECT DISTINCT m.position_id 
		FROM move m
		INNER JOIN game g ON m.game_id = g.id
		WHERE g.match_id = ? AND m.position_id IS NOT NULL
	`, matchID)
	if err != nil {
		return fmt.Errorf("error collecting position IDs: %w", err)
	}
	var positionIDs []int64
	for rows.Next() {
		var pid int64
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return fmt.Errorf("error scanning position ID: %w", err)
		}
		positionIDs = append(positionIDs, pid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating position IDs: %w", err)
	}

	// Foreign key constraints will cascade delete to game, move, and move_analysis
	_, err = tx.Exec(`DELETE FROM match WHERE id = ?`, matchID)
	if err != nil {
		return fmt.Errorf("error deleting match: %w", err)
	}

	// Delete orphaned positions that are no longer referenced by any move
	// and not part of any collection
	for _, pid := range positionIDs {
		var refCount int
		err := tx.QueryRow(`
			SELECT COUNT(*) FROM (
				SELECT position_id FROM move WHERE position_id = ?
				UNION ALL
				SELECT position_id FROM collection_position WHERE position_id = ?
			)
		`, pid, pid).Scan(&refCount)
		if err != nil {
			return fmt.Errorf("error checking position references for ID %d: %w", pid, err)
		}
		if refCount == 0 {
			// Position is orphaned — delete it (cascades to analysis and comment)
			_, err = tx.Exec(`DELETE FROM position WHERE id = ?`, pid)
			if err != nil {
				return fmt.Errorf("error deleting orphaned position %d: %w", pid, err)
			}
		}
	}

	return tx.Commit()
}

// GetMatchMovePositions returns all positions from a match in chronological order
// Positions are returned as they were stored (from player on roll POV)
// The frontend is responsible for mirroring display if needed
func (d *Database) GetMatchMovePositions(matchID int64) ([]MatchMovePosition, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Get match info for player names
	var player1Name, player2Name string
	err := d.db.QueryRow(`
		SELECT player1_name, player2_name 
		FROM match 
		WHERE id = ?
	`, matchID).Scan(&player1Name, &player2Name)
	if err != nil {
		return nil, fmt.Errorf("match not found: %w", err)
	}

	// Get all moves across all games in chronological order
	// Join with game table to get game number and position table to get position data
	rows, err := d.db.Query(`
		SELECT 
			m.id as move_id,
			m.game_id,
			g.game_number,
			m.move_number,
			m.move_type,
			m.player,
			m.position_id,
			p.state as position_state,
			COALESCE(m.checker_move, '') as checker_move,
			COALESCE(m.cube_action, '') as cube_action
		FROM move m
		INNER JOIN game g ON m.game_id = g.id
		INNER JOIN position p ON m.position_id = p.id
		WHERE g.match_id = ?
		ORDER BY g.game_number ASC, m.move_number ASC
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to query moves: %w", err)
	}
	defer rows.Close()

	var movePositions []MatchMovePosition
	for rows.Next() {
		var moveID, gameID, positionID int64
		var gameNumber, moveNumber, player int32
		var moveType, positionState, checkerMove, cubeAction string

		err := rows.Scan(&moveID, &gameID, &gameNumber, &moveNumber, &moveType, &player, &positionID, &positionState, &checkerMove, &cubeAction)
		if err != nil {
			fmt.Printf("Error scanning move: %v\n", err)
			continue
		}

		// Unmarshal position
		var position Position
		err = json.Unmarshal([]byte(positionState), &position)
		if err != nil {
			fmt.Printf("Error unmarshalling position: %v\n", err)
			continue
		}

		// Convert player from XG encoding (-1, 1) to blunderDB encoding (0, 1)
		playerBlunderDB := convertXGPlayerToBlunderDB(player)

		movePos := MatchMovePosition{
			Position:     position,
			MoveID:       moveID,
			GameID:       gameID,
			GameNumber:   gameNumber,
			MoveNumber:   moveNumber,
			MoveType:     moveType,
			PlayerOnRoll: int32(playerBlunderDB), // Now 0 or 1
			Player1Name:  player1Name,
			Player2Name:  player2Name,
			CheckerMove:  checkerMove,
			CubeAction:   cubeAction,
		}

		movePositions = append(movePositions, movePos)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movePositions, nil
}

// GetDatabaseStats returns statistics about the database
func (d *Database) GetDatabaseStats() (map[string]interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	stats := make(map[string]interface{})

	// Count positions
	var posCount int64
	err := d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount)
	if err != nil {
		return nil, err
	}
	stats["position_count"] = posCount

	// Count analyses
	var analysisCount int64
	err = d.db.QueryRow(`SELECT COUNT(*) FROM analysis`).Scan(&analysisCount)
	if err != nil {
		return nil, err
	}
	stats["analysis_count"] = analysisCount

	// Count matches
	var matchCount int64
	err = d.db.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&matchCount)
	if err != nil {
		// Table might not exist in older databases
		stats["match_count"] = int64(0)
	} else {
		stats["match_count"] = matchCount
	}

	// Count games
	var gameCount int64
	err = d.db.QueryRow(`SELECT COUNT(*) FROM game`).Scan(&gameCount)
	if err != nil {
		stats["game_count"] = int64(0)
	} else {
		stats["game_count"] = gameCount
	}

	// Count moves
	var moveCount int64
	err = d.db.QueryRow(`SELECT COUNT(*) FROM move`).Scan(&moveCount)
	if err != nil {
		stats["move_count"] = int64(0)
	} else {
		stats["move_count"] = moveCount
	}

	return stats, nil
}

// Collection represents a collection of positions
type Collection struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	SortOrder     int    `json:"sortOrder"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	PositionCount int    `json:"positionCount"`
}

// CollectionPosition represents a position in a collection with its order
type CollectionPosition struct {
	ID           int64    `json:"id"`
	CollectionID int64    `json:"collectionId"`
	PositionID   int64    `json:"positionId"`
	SortOrder    int      `json:"sortOrder"`
	AddedAt      string   `json:"addedAt"`
	Position     Position `json:"position"`
}

// CreateCollection creates a new collection
func (d *Database) CreateCollection(name string, description string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}

	// Get the max sort_order
	var maxOrder int
	err := d.db.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM collection`).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	result, err := d.db.Exec(`
		INSERT INTO collection (name, description, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, name, description, maxOrder+1)
	if err != nil {
		fmt.Println("Error creating collection:", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetAllCollections returns all collections with their position counts
func (d *Database) GetAllCollections() ([]Collection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			c.id,
			c.name,
			COALESCE(c.description, ''),
			c.sort_order,
			c.created_at,
			c.updated_at,
			COUNT(cp.id) as position_count
		FROM collection c
		LEFT JOIN collection_position cp ON c.id = cp.collection_id
		GROUP BY c.id
		ORDER BY c.sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt, &c.PositionCount)
		if err != nil {
			continue
		}
		collections = append(collections, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return collections, nil
}

// UpdateCollection updates a collection's name and description
func (d *Database) UpdateCollection(id int64, name string, description string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`
		UPDATE collection SET name = ?, description = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, description, id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteCollection deletes a collection and all its position associations
func (d *Database) DeleteCollection(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`DELETE FROM collection WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return nil
}

// ReorderCollections updates the sort order of all collections
func (d *Database) ReorderCollections(collectionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for i, id := range collectionIDs {
		_, err := tx.Exec(`UPDATE collection SET sort_order = ? WHERE id = ?`, i, id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// AddPositionToCollection adds a position to a collection
func (d *Database) AddPositionToCollection(collectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the max sort_order for this collection
	var maxOrder int
	err = tx.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = ?`, collectionID).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	_, err = tx.Exec(`
		INSERT OR IGNORE INTO collection_position (collection_id, position_id, sort_order, added_at)
		VALUES (?, ?, ?, datetime('now'))
	`, collectionID, positionID, maxOrder+1)
	if err != nil {
		return err
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AddPositionsToCollection adds multiple positions to a collection
func (d *Database) AddPositionsToCollection(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Get the max sort_order for this collection
	var maxOrder int
	err := d.db.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = ?`, collectionID).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for i, positionID := range positionIDs {
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO collection_position (collection_id, position_id, sort_order, added_at)
			VALUES (?, ?, ?, datetime('now'))
		`, collectionID, positionID, maxOrder+1+i)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// RemovePositionFromCollection removes a position from a collection
func (d *Database) RemovePositionFromCollection(collectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM collection_position 
		WHERE collection_id = ? AND position_id = ?
	`, collectionID, positionID)
	if err != nil {
		return err
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RemovePositionsFromCollection removes multiple positions from a collection
func (d *Database) RemovePositionsFromCollection(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, positionID := range positionIDs {
		_, err = tx.Exec(`
			DELETE FROM collection_position 
			WHERE collection_id = ? AND position_id = ?
		`, collectionID, positionID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GetPositionIndexMap returns a map of position ID to its 1-based index in the database
func (d *Database) GetPositionIndexMap() (map[int64]int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`SELECT id FROM position ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]int)
	index := 1
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		result[id] = index
		index++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetCollectionPositions returns all positions in a collection
func (d *Database) GetCollectionPositions(collectionID int64) ([]Position, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT p.id, p.state
		FROM position p
		INNER JOIN collection_position cp ON p.id = cp.position_id
		WHERE cp.collection_id = ?
		ORDER BY cp.sort_order ASC
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var id int64
		var state string
		err := rows.Scan(&id, &state)
		if err != nil {
			continue
		}

		var position Position
		err = json.Unmarshal([]byte(state), &position)
		if err != nil {
			continue
		}
		position.ID = id
		positions = append(positions, position)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return positions, nil
}

// ReorderCollectionPositions updates the sort order of positions within a collection
func (d *Database) ReorderCollectionPositions(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for i, positionID := range positionIDs {
		_, err := tx.Exec(`
			UPDATE collection_position SET sort_order = ?
			WHERE collection_id = ? AND position_id = ?
		`, i, collectionID, positionID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// MovePositionBetweenCollections moves a position from one collection to another
func (d *Database) MovePositionBetweenCollections(fromCollectionID int64, toCollectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// Remove from source collection
	_, err = tx.Exec(`
		DELETE FROM collection_position 
		WHERE collection_id = ? AND position_id = ?
	`, fromCollectionID, positionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Get max sort_order in destination collection
	var maxOrder int
	err = tx.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = ?`, toCollectionID).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	// Add to destination collection
	_, err = tx.Exec(`
		INSERT INTO collection_position (collection_id, position_id, sort_order, added_at)
		VALUES (?, ?, ?, datetime('now'))
	`, toCollectionID, positionID, maxOrder+1)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Update both collections' updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id IN (?, ?)`, fromCollectionID, toCollectionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// CopyPositionToCollection copies a position to a collection (position can be in multiple collections)
func (d *Database) CopyPositionToCollection(toCollectionID int64, positionID int64) error {
	return d.AddPositionToCollection(toCollectionID, positionID)
}

// GetCollectionByID returns a collection by its ID
func (d *Database) GetCollectionByID(id int64) (*Collection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var c Collection
	err := d.db.QueryRow(`
		SELECT 
			c.id,
			c.name,
			COALESCE(c.description, ''),
			c.sort_order,
			c.created_at,
			c.updated_at,
			COUNT(cp.id) as position_count
		FROM collection c
		LEFT JOIN collection_position cp ON c.id = cp.collection_id
		WHERE c.id = ?
		GROUP BY c.id
	`, id).Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt, &c.PositionCount)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// GetPositionCollections returns all collections that contain a specific position
func (d *Database) GetPositionCollections(positionID int64) ([]Collection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			c.id,
			c.name,
			COALESCE(c.description, ''),
			c.sort_order,
			c.created_at,
			c.updated_at
		FROM collection c
		INNER JOIN collection_position cp ON c.id = cp.collection_id
		WHERE cp.position_id = ?
		ORDER BY c.sort_order ASC
	`, positionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}
		collections = append(collections, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return collections, nil
}

// ExportCollections exports specific collections to a database file
func (d *Database) ExportCollections(exportPath string, collectionIDs []int64, metadata map[string]string, includeAnalysis bool, includeComments bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Collect all unique position IDs from selected collections
	positionIDsMap := make(map[int64]bool)
	for _, collectionID := range collectionIDs {
		rows, err := d.db.Query(`SELECT position_id FROM collection_position WHERE collection_id = ?`, collectionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var posID int64
			if err := rows.Scan(&posID); err == nil {
				positionIDsMap[posID] = true
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// Convert map to slice
	var positionIDs []int64
	for id := range positionIDsMap {
		positionIDs = append(positionIDs, id)
	}

	// Delete the export file if it already exists
	if _, err := os.Stat(exportPath); err == nil {
		if err := os.Remove(exportPath); err != nil {
			return fmt.Errorf("cannot remove existing export file: %v", err)
		}
	}

	// Create export database
	exportDB, err := sql.Open("sqlite", exportPath)
	if err != nil {
		return err
	}
	defer exportDB.Close()

	// Create schema
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER UNIQUE,
			data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS comment (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER UNIQUE,
			text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS filter_library (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			command TEXT,
			edit_position TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			position TEXT,
			timestamp INTEGER
		)
	`)
	if err != nil {
		return err
	}

	// Create match-related tables (for v1.4.0 compatibility)
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS match (
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
			match_hash TEXT,
			last_visited_position INTEGER DEFAULT -1
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`CREATE TABLE IF NOT EXISTS game (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id INTEGER,
		game_number INTEGER,
		initial_score_1 INTEGER,
		initial_score_2 INTEGER,
		winner INTEGER,
		points_won INTEGER,
		move_count INTEGER DEFAULT 0,
		FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
	)`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`CREATE TABLE IF NOT EXISTS move (
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
	)`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`CREATE TABLE IF NOT EXISTS move_analysis (
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
	)`)
	if err != nil {
		return err
	}

	// Create collection tables
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS collection (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS collection_position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL,
			position_id INTEGER NOT NULL,
			sort_order INTEGER DEFAULT 0,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(collection_id, position_id)
		)
	`)
	if err != nil {
		return err
	}

	// Export positions and create ID mapping
	oldToNewID := make(map[int64]int64)
	for _, posID := range positionIDs {
		var state string
		err := d.db.QueryRow(`SELECT state FROM position WHERE id = ?`, posID).Scan(&state)
		if err != nil {
			continue
		}

		result, err := exportDB.Exec(`INSERT INTO position (state) VALUES (?)`, state)
		if err != nil {
			continue
		}
		newID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		oldToNewID[posID] = newID

		// Export analysis if requested
		if includeAnalysis {
			var analysisData string
			err := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, posID).Scan(&analysisData)
			if err == nil {
				_, _ = exportDB.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newID, analysisData)
			}
		}

		// Export comments if requested
		if includeComments {
			var commentText string
			err := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, posID).Scan(&commentText)
			if err == nil {
				_, _ = exportDB.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newID, commentText)
			}
		}
	}

	// Export collections and their position mappings
	collectionIDMapping := make(map[int64]int64)
	for _, collectionID := range collectionIDs {
		var name, description string
		var sortOrder int
		var createdAt, updatedAt string
		err := d.db.QueryRow(`SELECT name, COALESCE(description, ''), sort_order, created_at, updated_at FROM collection WHERE id = ?`, collectionID).
			Scan(&name, &description, &sortOrder, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		result, err := exportDB.Exec(`INSERT INTO collection (name, description, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			name, description, sortOrder, createdAt, updatedAt)
		if err != nil {
			continue
		}
		newCollectionID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		collectionIDMapping[collectionID] = newCollectionID

		// Export collection_position mappings
		rows, err := d.db.Query(`SELECT position_id, sort_order, added_at FROM collection_position WHERE collection_id = ?`, collectionID)
		if err != nil {
			continue
		}
		for rows.Next() {
			var oldPosID int64
			var sortOrder int
			var addedAt string
			if err := rows.Scan(&oldPosID, &sortOrder, &addedAt); err != nil {
				continue
			}
			if newPosID, ok := oldToNewID[oldPosID]; ok {
				_, _ = exportDB.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order, added_at) VALUES (?, ?, ?, ?)`,
					newCollectionID, newPosID, sortOrder, addedAt)
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// Export metadata
	_, err = exportDB.Exec(`INSERT INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		return err
	}

	for key, value := range metadata {
		_, _ = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
	}

	return nil
}

// ========== Tournament Functions ==========

// CreateTournament creates a new tournament
func (d *Database) CreateTournament(name string, date string, location string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}

	// Get the max sort_order
	var maxOrder int
	err := d.db.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM tournament`).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	result, err := d.db.Exec(`
		INSERT INTO tournament (name, date, location, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`, name, date, location, maxOrder+1)
	if err != nil {
		fmt.Println("Error creating tournament:", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetAllTournaments returns all tournaments with their match counts
func (d *Database) GetAllTournaments() ([]Tournament, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			t.id,
			t.name,
			COALESCE(t.date, ''),
			COALESCE(t.location, ''),
			t.sort_order,
			t.created_at,
			t.updated_at,
			COUNT(m.id) as match_count
		FROM tournament t
		LEFT JOIN match m ON t.id = m.tournament_id
		GROUP BY t.id
		ORDER BY t.date DESC, t.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tournaments []Tournament
	for rows.Next() {
		var t Tournament
		err := rows.Scan(&t.ID, &t.Name, &t.Date, &t.Location, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt, &t.MatchCount)
		if err != nil {
			continue
		}
		tournaments = append(tournaments, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tournaments, nil
}

// UpdateTournament updates a tournament's details
func (d *Database) UpdateTournament(id int64, name string, date string, location string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`
		UPDATE tournament SET name = ?, date = ?, location = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, date, location, id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteTournament deletes a tournament (matches are unlinked, not deleted)
func (d *Database) DeleteTournament(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Unlink matches from this tournament
	_, err = tx.Exec(`UPDATE match SET tournament_id = NULL WHERE tournament_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete the tournament
	_, err = tx.Exec(`DELETE FROM tournament WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AddMatchToTournament adds a match to a tournament
func (d *Database) AddMatchToTournament(tournamentID int64, matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID)
	if err != nil {
		return err
	}

	// Update tournament's updated_at
	_, err = tx.Exec(`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveMatchFromTournament removes a match from a tournament
func (d *Database) RemoveMatchFromTournament(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`UPDATE match SET tournament_id = NULL WHERE id = ?`, matchID)
	return err
}

// SetMatchTournamentByName assigns a match to a tournament by name.
// If tournamentName is empty, the match is unlinked from any tournament.
// If no tournament with that name exists, one is created.
func (d *Database) SetMatchTournamentByName(matchID int64, tournamentName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	name := strings.TrimSpace(tournamentName)
	if name == "" {
		_, err := d.db.Exec(`UPDATE match SET tournament_id = NULL WHERE id = ?`, matchID)
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Look for existing tournament with that name
	var tournamentID int64
	err = tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, name).Scan(&tournamentID)
	if err != nil {
		// Create new tournament
		res, err2 := tx.Exec(`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, name)
		if err2 != nil {
			return err2
		}
		tournamentID, err2 = res.LastInsertId()
		if err2 != nil {
			return err2
		}
	}

	_, err = tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateMatch updates editable metadata for a match (player names and date).
// matchDate should be an empty string or a date string parseable by time.Parse ("2006-01-02").
func (d *Database) UpdateMatch(matchID int64, player1Name, player2Name, matchDate string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	var dateVal interface{}
	if matchDate != "" {
		t, err := time.Parse("2006-01-02", matchDate)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
		dateVal = t
	} else {
		dateVal = nil
	}

	_, err := d.db.Exec(
		`UPDATE match SET player1_name = ?, player2_name = ?, match_date = ? WHERE id = ?`,
		strings.TrimSpace(player1Name),
		strings.TrimSpace(player2Name),
		dateVal,
		matchID,
	)
	return err
}

// SwapMatchPlayers swaps the two players in a match: player1 becomes player2 and vice versa.
// This updates player names, game scores, game winners, and move player assignments.
func (d *Database) SwapMatchPlayers(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Swap player1_name and player2_name in the match table
	_, err = tx.Exec(`
		UPDATE match 
		SET player1_name = player2_name, player2_name = player1_name
		WHERE id = ?
	`, matchID)
	if err != nil {
		return fmt.Errorf("failed to swap player names: %w", err)
	}

	// 2. Swap initial_score_1/initial_score_2 and flip winner in the game table
	_, err = tx.Exec(`
		UPDATE game
		SET initial_score_1 = initial_score_2,
		    initial_score_2 = initial_score_1,
		    winner = -winner
		WHERE match_id = ?
	`, matchID)
	if err != nil {
		return fmt.Errorf("failed to swap game scores/winner: %w", err)
	}

	// 3. Flip player in the move table (XG encoding: 1 → -1, -1 → 1)
	_, err = tx.Exec(`
		UPDATE move
		SET player = -player
		WHERE game_id IN (SELECT id FROM game WHERE match_id = ?)
	`, matchID)
	if err != nil {
		return fmt.Errorf("failed to swap move players: %w", err)
	}

	// 4. Update position state JSON to swap scores and cube owner
	// Positions are stored normalized (player_on_roll = 0), but Score and Cube.Owner
	// reflect the original player assignment and must be swapped.
	posRows, err := tx.Query(`
		SELECT DISTINCT p.id, p.state
		FROM position p
		INNER JOIN move m ON m.position_id = p.id
		INNER JOIN game g ON m.game_id = g.id
		WHERE g.match_id = ?
	`, matchID)
	if err != nil {
		return fmt.Errorf("failed to query positions for swap: %w", err)
	}
	defer posRows.Close()

	for posRows.Next() {
		var posID int64
		var stateJSON string
		if err := posRows.Scan(&posID, &stateJSON); err != nil {
			return fmt.Errorf("failed to scan position %d: %w", posID, err)
		}

		var pos Position
		if err := json.Unmarshal([]byte(stateJSON), &pos); err != nil {
			return fmt.Errorf("failed to unmarshal position %d: %w", posID, err)
		}

		// Swap scores
		pos.Score[0], pos.Score[1] = pos.Score[1], pos.Score[0]

		// Flip cube owner (if not centered)
		if pos.Cube.Owner != None {
			pos.Cube.Owner = 1 - pos.Cube.Owner
		}

		updatedJSON, err := json.Marshal(pos)
		if err != nil {
			return fmt.Errorf("failed to marshal updated position %d: %w", posID, err)
		}

		_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(updatedJSON), posID)
		if err != nil {
			return fmt.Errorf("failed to update position %d: %w", posID, err)
		}
	}
	if err := posRows.Err(); err != nil {
		return fmt.Errorf("error iterating positions for swap: %w", err)
	}

	return tx.Commit()
}

// GetTournamentMatches returns all matches in a tournament
func (d *Database) GetTournamentMatches(tournamentID int64) ([]Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			id, player1_name, player2_name, event, location, round, 
			match_length, match_date, import_date, file_path, game_count, tournament_id,
			COALESCE(last_visited_position, -1) as last_visited_position
		FROM match 
		WHERE tournament_id = ?
		ORDER BY match_date DESC
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		var tournamentID sql.NullInt64
		err := rows.Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
			&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount, &tournamentID, &m.LastVisitedPosition)
		if err != nil {
			continue
		}
		if tournamentID.Valid {
			tid := tournamentID.Int64
			m.TournamentID = &tid
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}

// GetMatchTournament returns the tournament a match belongs to (if any)
func (d *Database) GetMatchTournament(matchID int64) (*Tournament, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var tournamentID sql.NullInt64
	err := d.db.QueryRow(`SELECT tournament_id FROM match WHERE id = ?`, matchID).Scan(&tournamentID)
	if err != nil {
		return nil, err
	}

	if !tournamentID.Valid {
		return nil, nil // Match is not in any tournament
	}

	var t Tournament
	err = d.db.QueryRow(`
		SELECT id, name, COALESCE(date, ''), COALESCE(location, ''), sort_order, created_at, updated_at
		FROM tournament WHERE id = ?
	`, tournamentID.Int64).Scan(&t.ID, &t.Name, &t.Date, &t.Location, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// ExportTournaments exports specific tournaments and their matches to a database file
func (d *Database) ExportTournaments(exportPath string, tournamentIDs []int64, metadata map[string]string, includeAnalysis bool, includeComments bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Collect all match IDs from selected tournaments
	var matchIDs []int64
	for _, tournamentID := range tournamentIDs {
		rows, err := d.db.Query(`SELECT id FROM match WHERE tournament_id = ?`, tournamentID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var matchID int64
			if err := rows.Scan(&matchID); err == nil {
				matchIDs = append(matchIDs, matchID)
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// Collect all unique position IDs from matches
	positionIDsMap := make(map[int64]bool)
	for _, matchID := range matchIDs {
		rows, err := d.db.Query(`
			SELECT DISTINCT m.position_id 
			FROM move m
			JOIN game g ON m.game_id = g.id
			WHERE g.match_id = ? AND m.position_id IS NOT NULL
		`, matchID)
		if err != nil {
			continue
		}
		for rows.Next() {
			var posID int64
			if err := rows.Scan(&posID); err == nil {
				positionIDsMap[posID] = true
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// Convert map to slice
	var positionIDs []int64
	for id := range positionIDsMap {
		positionIDs = append(positionIDs, id)
	}

	// Delete the export file if it already exists
	if _, err := os.Stat(exportPath); err == nil {
		if err := os.Remove(exportPath); err != nil {
			return fmt.Errorf("cannot remove existing export file: %v", err)
		}
	}

	// Create export database
	exportDB, err := sql.Open("sqlite", exportPath)
	if err != nil {
		return err
	}
	defer exportDB.Close()

	// Create schema (same as SetupDatabase but simplified)
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS analysis (
			id INTEGER PRIMARY KEY,
			position_id INTEGER,
			data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS comment (
			id INTEGER PRIMARY KEY,
			position_id INTEGER,
			text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS tournament (
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
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS match (
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
			match_hash TEXT,
			tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL,
			last_visited_position INTEGER DEFAULT -1
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER,
			game_number INTEGER,
			initial_score_1 INTEGER,
			initial_score_2 INTEGER,
			winner INTEGER,
			points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS move (
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
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS move_analysis (
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
		)
	`)
	if err != nil {
		return err
	}

	// Export positions
	oldToNewID := make(map[int64]int64)
	for _, posID := range positionIDs {
		var state string
		err := d.db.QueryRow(`SELECT state FROM position WHERE id = ?`, posID).Scan(&state)
		if err != nil {
			continue
		}

		result, err := exportDB.Exec(`INSERT INTO position (state) VALUES (?)`, state)
		if err != nil {
			continue
		}
		newID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		oldToNewID[posID] = newID

		// Export analysis if requested
		if includeAnalysis {
			var analysisData string
			err := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, posID).Scan(&analysisData)
			if err == nil {
				_, _ = exportDB.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newID, analysisData)
			}
		}

		// Export comments if requested
		if includeComments {
			var commentText string
			err := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, posID).Scan(&commentText)
			if err == nil {
				_, _ = exportDB.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newID, commentText)
			}
		}
	}

	// Export tournaments
	tournamentIDMapping := make(map[int64]int64)
	for _, tournamentID := range tournamentIDs {
		var name, date, location string
		var sortOrder int
		var createdAt, updatedAt string
		err := d.db.QueryRow(`SELECT name, COALESCE(date, ''), COALESCE(location, ''), sort_order, created_at, updated_at FROM tournament WHERE id = ?`, tournamentID).
			Scan(&name, &date, &location, &sortOrder, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		result, err := exportDB.Exec(`INSERT INTO tournament (name, date, location, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			name, date, location, sortOrder, createdAt, updatedAt)
		if err != nil {
			continue
		}
		newTournamentID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		tournamentIDMapping[tournamentID] = newTournamentID
	}

	// Export matches and their games/moves
	matchIDMapping := make(map[int64]int64)
	for _, matchID := range matchIDs {
		var player1Name, player2Name, event, location, round, filePath string
		var matchLength, gameCount int
		var matchDate, importDate string
		var matchHash sql.NullString
		var srcTournamentID sql.NullInt64

		err := d.db.QueryRow(`
			SELECT player1_name, player2_name, event, location, round, match_length, 
			       match_date, import_date, file_path, game_count, match_hash, tournament_id 
			FROM match WHERE id = ?`, matchID).
			Scan(&player1Name, &player2Name, &event, &location, &round, &matchLength,
				&matchDate, &importDate, &filePath, &gameCount, &matchHash, &srcTournamentID)
		if err != nil {
			continue
		}

		var newTournamentID sql.NullInt64
		if srcTournamentID.Valid {
			if newID, ok := tournamentIDMapping[srcTournamentID.Int64]; ok {
				newTournamentID = sql.NullInt64{Int64: newID, Valid: true}
			}
		}

		result, err := exportDB.Exec(`
			INSERT INTO match (player1_name, player2_name, event, location, round, match_length, 
			                   match_date, import_date, file_path, game_count, match_hash, tournament_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			player1Name, player2Name, event, location, round, matchLength,
			matchDate, importDate, filePath, gameCount, matchHash, newTournamentID)
		if err != nil {
			continue
		}
		newMatchID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		matchIDMapping[matchID] = newMatchID

		// Export games
		gameRows, err := d.db.Query(`SELECT id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count FROM game WHERE match_id = ?`, matchID)
		if err != nil {
			continue
		}

		gameIDMapping := make(map[int64]int64)
		for gameRows.Next() {
			var gameID int64
			var gameNumber, initialScore1, initialScore2, winner, pointsWon, moveCount int
			if err := gameRows.Scan(&gameID, &gameNumber, &initialScore1, &initialScore2, &winner, &pointsWon, &moveCount); err != nil {
				continue
			}

			result, err := exportDB.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				newMatchID, gameNumber, initialScore1, initialScore2, winner, pointsWon, moveCount)
			if err != nil {
				continue
			}
			newGameID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}
			gameIDMapping[gameID] = newGameID
		}
		if err := gameRows.Err(); err != nil {
			return err
		}
		gameRows.Close()

		// Export moves for each game
		for oldGameID, newGameID := range gameIDMapping {
			moveRows, err := d.db.Query(`SELECT id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action FROM move WHERE game_id = ?`, oldGameID)
			if err != nil {
				continue
			}

			for moveRows.Next() {
				var moveID int64
				var moveNumber, player, dice1, dice2 int
				var moveType, checkerMove, cubeAction string
				var oldPosID sql.NullInt64
				if err := moveRows.Scan(&moveID, &moveNumber, &moveType, &oldPosID, &player, &dice1, &dice2, &checkerMove, &cubeAction); err != nil {
					continue
				}

				var newPosID sql.NullInt64
				if oldPosID.Valid {
					if newID, ok := oldToNewID[oldPosID.Int64]; ok {
						newPosID = sql.NullInt64{Int64: newID, Valid: true}
					}
				}

				result, err := exportDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					newGameID, moveNumber, moveType, newPosID, player, dice1, dice2, checkerMove, cubeAction)
				if err != nil {
					continue
				}
				newMoveID, err := result.LastInsertId()
				if err != nil {
					return fmt.Errorf("failed to get last insert ID: %w", err)
				}

				// Export move_analysis
				analysisRows, err := d.db.Query(`SELECT analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate FROM move_analysis WHERE move_id = ?`, moveID)
				if err != nil {
					continue
				}
				for analysisRows.Next() {
					var analysisType, depth string
					var equity, equityError, winRate, gammonRate, bgRate, oppWinRate, oppGammonRate, oppBgRate float64
					if err := analysisRows.Scan(&analysisType, &depth, &equity, &equityError, &winRate, &gammonRate, &bgRate, &oppWinRate, &oppGammonRate, &oppBgRate); err != nil {
						continue
					}
					_, _ = exportDB.Exec(`INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
						newMoveID, analysisType, depth, equity, equityError, winRate, gammonRate, bgRate, oppWinRate, oppGammonRate, oppBgRate)
				}
				if err := analysisRows.Err(); err != nil {
					return err
				}
				analysisRows.Close()
			}
			if err := moveRows.Err(); err != nil {
				return err
			}
			moveRows.Close()
		}
	}

	// Export metadata
	_, err = exportDB.Exec(`INSERT INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		return err
	}

	for key, value := range metadata {
		_, _ = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
	}

	return nil
}
