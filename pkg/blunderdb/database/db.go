package database

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

type Database struct {
	db                *sql.DB
	mu                sync.RWMutex                        // RWMutex allows concurrent reads
	cancelMu          sync.Mutex                          // guards importCancel (held briefly, never with mu)
	importCancel      context.CancelFunc                  // cancels the in-flight import/migration; nil when idle
	migrationProgress func(phase string, done, total int) // optional progress callback (GUI only)
	store             *sqlite.Storage                     // SQLite Storage backend, wraps db (P2)
}

// beginCancellableImport creates a fresh cancellable context for an import or
// migration and registers its cancel func so CancelImport can abort it from
// another goroutine (e.g. the Wails frontend) while the operation holds d.mu.
// cancelMu — never d.mu — guards the registration, so CancelImport never blocks
// on the running import. The returned done func must be deferred: it clears the
// registration and releases the context's resources.
func (d *Database) beginCancellableImport() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	d.cancelMu.Lock()
	d.importCancel = cancel
	d.cancelMu.Unlock()
	return ctx, func() {
		d.cancelMu.Lock()
		d.importCancel = nil
		d.cancelMu.Unlock()
		cancel()
	}
}

// CancelImport aborts any in-flight import or migration started through the
// Database wrapper. It is bound to the Wails frontend. No-op when idle.
func (d *Database) CancelImport() {
	d.cancelMu.Lock()
	cancel := d.importCancel
	d.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// rebuildStore (re)creates the SQLite Storage that wraps the current *sql.DB.
// It must be called after SetupDatabase/OpenDatabase replace d.db. The Storage
// borrows the handle (sqlite.New): d.db stays owned by this Database.
func (d *Database) rebuildStore() {
	d.store = sqlite.New(d.db)
}

func NewDatabase() *Database {
	return &Database{}
}

// Conn returns the underlying *sql.DB handle. It is exposed for callers
// outside the database package that need to run maintenance statements or
// raw queries (CLI maintenance, tests). It may be nil before Setup/Open.
func (d *Database) Conn() *sql.DB {
	return d.db
}

// Close closes the underlying connection and clears it. It is safe to call
// when the connection is already nil or closed.
func (d *Database) Close() error {
	if d.db == nil {
		return nil
	}
	err := d.db.Close()
	d.db = nil
	return err
}

// applyPragmas applies the exact same PRAGMA set as the standalone SQLite
// Storage backend (D9). WAL journal mode is skipped for in-memory databases
// (":memory:") because WAL requires a real filesystem.
func (d *Database) applyPragmas(path string) error {
	return sqlite.ApplyPragmas(d.db, path)
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

	// Size the connection pool. Critical for ":memory:": each pooled
	// connection is a SEPARATE empty in-memory database, so concurrent reads
	// (RWMutex allows multiple readers) would otherwise hit a fresh connection
	// with no schema -> "no such table". ConfigurePool pins ":memory:" to a
	// single connection; file-backed DBs are allowed to grow.
	sqlite.ConfigurePool(d.db, path)

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
            state             TEXT    NOT NULL,
            is_cube_response  INTEGER NOT NULL DEFAULT 0
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
            is_close_cube               INTEGER NOT NULL DEFAULT 0,
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
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            scope TEXT NOT NULL DEFAULT ''
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
			edit_position TEXT,
			exclude_position TEXT,
			scope TEXT NOT NULL DEFAULT ''
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
			exclude_position TEXT,
			timestamp INTEGER,
			scope TEXT NOT NULL DEFAULT ''
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
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)

	// Add last_visited_position column to match table if it doesn't exist (v1.7.0)
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)

	// Add canonical_hash column to match table if it doesn't exist
	// canonical_hash is format-independent (same match imported from XG and SGF will have the same canonical_hash)
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN canonical_hash TEXT`)

	// Add comment column to match table
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN comment TEXT DEFAULT ''`)

	// Add tournament_sort_order column to match table (ordering within a tournament)
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_sort_order INTEGER DEFAULT 0`)

	// Add comment column to tournament table
	_, _ = d.db.Exec(`ALTER TABLE tournament ADD COLUMN comment TEXT DEFAULT ''`)

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
		`CREATE        INDEX IF NOT EXISTS idx_position_cube_response  ON position(decision_type, is_cube_response)`,
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
		`CREATE        INDEX IF NOT EXISTS idx_analysis_is_close_cube  ON analysis(is_close_cube) WHERE is_close_cube = 1`,
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

	d.rebuildStore()
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

	// Size the connection pool. Critical for ":memory:": each pooled
	// connection is a SEPARATE empty in-memory database, so concurrent reads
	// (RWMutex allows multiple readers) would otherwise hit a fresh connection
	// with no schema -> "no such table". ConfigurePool pins ":memory:" to a
	// single connection; file-backed DBs are allowed to grow.
	sqlite.ConfigurePool(d.db, path)

	// Apply performance and safety PRAGMAs (includes foreign_keys=ON)
	if err = d.applyPragmas(path); err != nil {
		return err
	}

	migCtx, migDone := d.beginCancellableImport()
	defer migDone()
	if err := d.runMigrationChain(migCtx); err != nil {
		return err
	}

	d.ensureSearchStats()

	d.rebuildStore()
	return nil
}

// ensureSearchStats runs a one-time ANALYZE when the opened database has no
// query-planner statistics yet (sqlite_stat1 absent or empty). Without stats
// SQLite mis-estimates selectivity for non-selective search filters — e.g. a
// "win rate > 55% AND gammon > 20%" search that matches most rows is planned as
// a single-column analysis-index scan followed by a TEMP B-TREE sort on p.id,
// instead of scanning position in primary-key order (no sort). A full ANALYZE
// fixes the plan (~4x on that case in the tournois benchmark); the stats persist
// in the file, so later opens — and migrated databases, which already ANALYZE —
// skip this. Non-fatal: search still works with stale/absent stats.
func (d *Database) ensureSearchStats() {
	if d.db == nil {
		return
	}
	var n int
	// Errors (e.g. sqlite_stat1 does not exist yet) count as "no stats".
	if err := d.db.QueryRow(`SELECT count(*) FROM sqlite_stat1`).Scan(&n); err == nil && n > 0 {
		return
	}
	if _, err := d.db.Exec(`ANALYZE`); err != nil {
		slog.Warn("ANALYZE for search statistics failed", "err", err)
	}
}
