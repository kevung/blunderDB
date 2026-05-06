package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

type Database struct {
	db                *sql.DB
	mu                sync.RWMutex                        // RWMutex allows concurrent reads
	importCancelled   int32                               // Flag to cancel ongoing import (atomic)
	migrationProgress func(phase string, done, total int) // optional progress callback (GUI only)
}

func NewDatabase() *Database {
	return &Database{}
}

// WAL journal mode is skipped for in-memory databases (":memory:")
// because WAL requires a real filesystem.
func (d *Database) applyPragmas(path string) error {
	if path != ":memory:" {
		var mode string
		if err := d.db.QueryRow(`PRAGMA journal_mode = WAL`).Scan(&mode); err != nil {
			return fmt.Errorf("PRAGMA journal_mode=WAL: %w", err)
		}
	}
	pragmas := []string{
		`PRAGMA synchronous  = NORMAL`,
		`PRAGMA cache_size   = -65536`,
		`PRAGMA temp_store   = MEMORY`,
		`PRAGMA mmap_size    = 268435456`,
		`PRAGMA foreign_keys = ON`,
	}
	for _, p := range pragmas {
		if _, err := d.db.Exec(p); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}
	return nil
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
		return err
	}

	// Apply performance and safety PRAGMAs (includes foreign_keys=ON)
	if err = d.applyPragmas(path); err != nil {
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
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS position (
            id                INTEGER PRIMARY KEY AUTOINCREMENT,
            zobrist_hash      INTEGER,
            decision_type     INTEGER,
            player_on_roll    INTEGER,
            dice_1            INTEGER,
            dice_2            INTEGER,
            cube_value        INTEGER,
            cube_owner        INTEGER,
            score_1           INTEGER,
            score_2           INTEGER,
            match_length      INTEGER,
            has_jacoby        INTEGER,
            has_beaver        INTEGER,
            pip_1             INTEGER,
            pip_2             INTEGER,
            pip_diff          INTEGER,
            off_1             INTEGER,
            off_2             INTEGER,
            back_checkers_1   INTEGER,
            back_checkers_2   INTEGER,
            no_contact        INTEGER,
            occupancy_1       INTEGER,
            occupancy_2       INTEGER,
            point_mask_1      INTEGER,
            point_mask_2      INTEGER,
            state             TEXT    NOT NULL
        )
    `)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS analysis (
            id                          INTEGER PRIMARY KEY,
            position_id                 INTEGER,
            data                        JSON,
            best_cube_action            TEXT,
            cube_error                  INTEGER,
            best_move_equity_error      INTEGER,
            player1_win_rate            INTEGER,
            player1_gammon_rate         INTEGER,
            player1_backgammon_rate     INTEGER,
            player2_win_rate            INTEGER,
            player2_gammon_rate         INTEGER,
            player2_backgammon_rate     INTEGER,
            is_forced                   INTEGER NOT NULL DEFAULT 0,
            FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
        )
    `)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS comment (
            id INTEGER PRIMARY KEY,
            position_id INTEGER,
            text TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            modified_at DATETIME,
            FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
        )
    `)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS metadata (
            key TEXT PRIMARY KEY,
            value TEXT
        )
    `)
	if err != nil {
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
		return err
	}

	// Create index on match_hash for fast duplicate detection
	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
	if err != nil {
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
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS move_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			move_id INTEGER,
			analysis_type TEXT,
			depth TEXT,
			equity INTEGER,
			equity_error INTEGER,
			win_rate INTEGER,
			gammon_rate INTEGER,
			backgammon_rate INTEGER,
			opponent_win_rate INTEGER,
			opponent_gammon_rate INTEGER,
			opponent_backgammon_rate INTEGER,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
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
		return err
	}

	// Create index for faster collection lookups
	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_collection_position_collection ON collection_position(collection_id)`)
	if err != nil {
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
		return err
	}

	// Add tournament_id column to match table if it doesn't exist
	d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)

	// Add last_visited_position column to match table if it doesn't exist (v1.7.0)
	d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)

	// Add canonical_hash column to match table if it doesn't exist
	// canonical_hash is format-independent (same match imported from XG and SGF will have the same canonical_hash)
	d.db.Exec(`ALTER TABLE match ADD COLUMN canonical_hash TEXT`)

	// Add comment column to match table
	d.db.Exec(`ALTER TABLE match ADD COLUMN comment TEXT DEFAULT ''`)

	// Add tournament_sort_order column to match table (ordering within a tournament)
	d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_sort_order INTEGER DEFAULT 0`)

	// Add comment column to tournament table
	d.db.Exec(`ALTER TABLE tournament ADD COLUMN comment TEXT DEFAULT ''`)

	// Create Anki spaced repetition tables (v1.8.0)
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS anki_deck (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			source_type TEXT NOT NULL DEFAULT 'collection',
			source_id INTEGER DEFAULT 0,
			source_command TEXT DEFAULT '',
			request_retention REAL DEFAULT 0.9,
			maximum_interval REAL DEFAULT 36500,
			enable_fuzz INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS anki_card (
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
			FOREIGN KEY(deck_id) REFERENCES anki_deck(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(deck_id, position_id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_anki_card_deck ON anki_card(deck_id)`)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_anki_card_due ON anki_card(deck_id, due)`)
	if err != nil {
		return err
	}

	// v2.0.0 indexes — position search acceleration
	v2indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_position_zobrist        ON position(zobrist_hash)`,
		`CREATE        INDEX IF NOT EXISTS idx_position_decision_pip   ON position(decision_type, pip_diff)`,
		`CREATE        INDEX IF NOT EXISTS idx_position_decision_dice  ON position(decision_type, dice_1, dice_2)`,
		`CREATE        INDEX IF NOT EXISTS idx_position_pip_diff       ON position(pip_diff)`,
		`CREATE        INDEX IF NOT EXISTS idx_position_dice           ON position(dice_1, dice_2)`,
		`CREATE        INDEX IF NOT EXISTS idx_position_off            ON position(off_1, off_2)`,
		`CREATE        INDEX IF NOT EXISTS idx_position_score          ON position(match_length, score_1, score_2)`,
		`CREATE        INDEX IF NOT EXISTS idx_position_score_cube     ON position(match_length, score_1, score_2, cube_value)`,
		`CREATE        INDEX IF NOT EXISTS idx_analysis_position       ON analysis(position_id)`,
		`CREATE        INDEX IF NOT EXISTS idx_analysis_win_gammon     ON analysis(player1_win_rate, player1_gammon_rate)`,
		`CREATE        INDEX IF NOT EXISTS idx_analysis_win1           ON analysis(player1_win_rate)`,
		`CREATE        INDEX IF NOT EXISTS idx_analysis_cube_error     ON analysis(cube_error)`,
		`CREATE        INDEX IF NOT EXISTS idx_analysis_move_error     ON analysis(best_move_equity_error)`,
		`CREATE        INDEX IF NOT EXISTS idx_analysis_is_forced      ON analysis(is_forced) WHERE is_forced = 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_match_canonical         ON match(canonical_hash)`,
		`CREATE        INDEX IF NOT EXISTS idx_move_position           ON move(position_id)`,
		`CREATE        INDEX IF NOT EXISTS idx_move_game               ON move(game_id)`,
		`CREATE        INDEX IF NOT EXISTS idx_game_match              ON game(match_id)`,
	}
	for _, idx := range v2indexes {
		if _, err = d.db.Exec(idx); err != nil {
			return err
		}
	}

	// Insert or update the database version
	_, err = d.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
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
		return err
	}

	// Apply performance and safety PRAGMAs (includes foreign_keys=ON)
	if err = d.applyPragmas(path); err != nil {
		return err
	}

	// Check the database version
	var dbVersion string
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
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
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.1.0")
			if err != nil {
				return err
			}
			dbVersion = "1.1.0"
			slog.Info("database upgraded", "from", "1.0.0", "to", "1.1.0")
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
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.2.0")
			if err != nil {
				return err
			}
			dbVersion = "1.2.0"
			slog.Info("database upgraded", "from", "1.1.0", "to", "1.2.0")
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
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.3.0")
			if err != nil {
				return err
			}
			dbVersion = "1.3.0"
			slog.Info("database upgraded", "from", "1.2.0", "to", "1.3.0")
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
				return err
			}

			// Create index on match_hash for fast duplicate detection
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
			if err != nil {
				return err
			}

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.4.0")
			if err != nil {
				return err
			}
			dbVersion = "1.4.0"
			slog.Info("database upgraded", "from", "1.3.0", "to", "1.4.0")
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
				return err
			}

			// Create index on match_hash
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
			if err != nil {
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

			slog.Info("added match_hash column and populated existing matches")
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
				return err
			}

			// Create index for faster collection lookups
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_collection_position_collection ON collection_position(collection_id)`)
			if err != nil {
				return err
			}

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.5.0")
			if err != nil {
				return err
			}
			dbVersion = "1.5.0"
			slog.Info("database upgraded", "from", "1.4.0", "to", "1.5.0")
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
				return err
			}

			// Add tournament_id column to match table
			d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.6.0")
			if err != nil {
				return err
			}
			dbVersion = "1.6.0"
			slog.Info("database upgraded", "from", "1.5.0", "to", "1.6.0")
		}
	}

	// Auto-migrate from 1.6.0 to 1.7.0
	if dbVersion == "1.6.0" {
		// Add last_visited_position column to match table
		d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)

		_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.7.0")
		if err != nil {
			return err
		}
		dbVersion = "1.7.0"
		slog.Info("database upgraded", "from", "1.6.0", "to", "1.7.0")
	}

	// Auto-migrate from 1.7.0 to 1.8.0
	if dbVersion == "1.7.0" {
		// Add created_at column to comment table
		d.db.Exec(`ALTER TABLE comment ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP`)
		// Backfill existing rows that have NULL created_at
		d.db.Exec(`UPDATE comment SET created_at = CURRENT_TIMESTAMP WHERE created_at IS NULL`)

		_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.8.0")
		if err != nil {
			return err
		}
		dbVersion = "1.8.0"
		slog.Info("database upgraded", "from", "1.7.0", "to", "1.8.0")
	}

	// Auto-migrate from 1.8.0 to 1.9.0
	if dbVersion == "1.8.0" {
		// Add modified_at column to comment table
		d.db.Exec(`ALTER TABLE comment ADD COLUMN modified_at DATETIME`)

		_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.9.0")
		if err != nil {
			return err
		}
		dbVersion = "1.9.0"
		slog.Info("database upgraded", "from", "1.8.0", "to", "1.9.0")
	}

	// Auto-migrate from 1.9.0 to 2.0.0
	// Backfills new scalar columns, deduplicates positions, and creates indexes.
	if dbVersion == "1.9.0" {
		if err := d.migrate_1_9_0_to_2_0_0(); err != nil {
			return fmt.Errorf("migration 1.9.0→2.0.0 failed: %w", err)
		}
		dbVersion = "2.0.0"
	}

	// Auto-migrate from 2.0.0 to 2.1.0
	// Converts analysis/move_analysis REAL columns to integer-scaled INTEGER values
	// and re-rounds all JSON blobs for compact storage.
	if dbVersion == "2.0.0" {
		if err := d.migrate_2_0_0_to_2_1_0(); err != nil {
			return fmt.Errorf("migration 2.0.0→2.1.0 failed: %w", err)
		}
		dbVersion = "2.1.0"
	}

	// Auto-migrate from 2.1.0 to 2.2.0
	// Compacts position.state from full Position JSON to board-only array,
	// reducing storage by ~85-90% on the state column. Also prunes command history.
	if dbVersion == "2.1.0" {
		if err := d.migrate_2_1_0_to_2_2_0(); err != nil {
			return fmt.Errorf("migration 2.1.0→2.2.0 failed: %w", err)
		}
		dbVersion = "2.2.0"
	}

	// Auto-migrate from 2.2.0 to 2.3.0
	// Compresses analysis.data JSON blobs with zlib, reducing analysis storage
	// by ~60-80%. Scalar filter columns are unchanged.
	if dbVersion == "2.2.0" {
		if err := d.migrate_2_2_0_to_2_3_0(); err != nil {
			return fmt.Errorf("migration 2.2.0→2.3.0 failed: %w", err)
		}
		dbVersion = "2.3.0"
	}

	// Auto-migrate from 2.3.0 to 2.4.0
	// Repairs best_move_equity_error for positions where PlayedMoves was missing
	// from the analysis JSON blob in earlier migrations, by looking up move.checker_move.
	if dbVersion == "2.3.0" {
		if err := d.migrate_2_3_0_to_2_4_0(); err != nil {
			return fmt.Errorf("migration 2.3.0→2.4.0 failed: %w", err)
		}
		dbVersion = "2.4.0"
	}

	// Auto-migrate from 2.4.0 to 2.5.0
	// Adds is_forced column to analysis; backfills checker positions with exactly
	// one legal move (len(CheckerAnalysis.Moves) == 1).
	if dbVersion == "2.4.0" {
		if err := d.migrate_2_4_0_to_2_5_0(); err != nil {
			return fmt.Errorf("migration 2.4.0→2.5.0 failed: %w", err)
		}
		dbVersion = "2.5.0"
	}

	// Ensure all required tables and columns exist.
	// This repairs databases that were migrated through versions that skipped
	// creating some tables (e.g. filter_library was missing from some migration paths).
	if err := d.ensureAllTablesExist(); err != nil {
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
