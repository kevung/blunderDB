package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// schemaStatements is the full v2.7.0 DDL for a fresh database, in dependency
// order. It is the same table/index set the Database wrapper's SetupDatabase
// builds; the wrapper delegates to Bootstrap so the two never drift.
var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS position (
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
	)`,
	`CREATE TABLE IF NOT EXISTS analysis (
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
	)`,
	`CREATE TABLE IF NOT EXISTS comment (
		id INTEGER PRIMARY KEY,
		position_id INTEGER,
		text TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		modified_at DATETIME,
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS command_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		command TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS filter_library (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		command TEXT,
		edit_position TEXT,
		exclude_position TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS search_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		command TEXT,
		position TEXT,
		exclude_position TEXT,
		timestamp INTEGER
	)`,
	`CREATE TABLE IF NOT EXISTS match (
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
	)`,
	`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`,
	`CREATE TABLE IF NOT EXISTS game (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id INTEGER,
		game_number INTEGER,
		initial_score_1 INTEGER,
		initial_score_2 INTEGER,
		winner INTEGER,
		points_won INTEGER,
		move_count INTEGER DEFAULT 0,
		FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS move (
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
	)`,
	`CREATE TABLE IF NOT EXISTS move_analysis (
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
	)`,
	`CREATE TABLE IF NOT EXISTS collection (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS collection_position (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		collection_id INTEGER NOT NULL,
		position_id INTEGER NOT NULL,
		sort_order INTEGER DEFAULT 0,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
		UNIQUE(collection_id, position_id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_collection_position_collection ON collection_position(collection_id)`,
	`CREATE TABLE IF NOT EXISTS tournament (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		date TEXT,
		location TEXT,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`,
	`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`,
	`ALTER TABLE match ADD COLUMN canonical_hash TEXT`,
	`ALTER TABLE match ADD COLUMN comment TEXT DEFAULT ''`,
	`ALTER TABLE match ADD COLUMN tournament_sort_order INTEGER DEFAULT 0`,
	`ALTER TABLE tournament ADD COLUMN comment TEXT DEFAULT ''`,
	`CREATE TABLE IF NOT EXISTS anki_deck (
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
	)`,
	`CREATE TABLE IF NOT EXISTS anki_card (
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
	)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_card_deck ON anki_card(deck_id)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_card_due ON anki_card(deck_id, due)`,
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
	`CREATE        INDEX IF NOT EXISTS idx_analysis_is_close_cube  ON analysis(is_close_cube) WHERE is_close_cube = 1`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_match_canonical         ON match(canonical_hash)`,
	`CREATE        INDEX IF NOT EXISTS idx_move_position           ON move(position_id)`,
	`CREATE        INDEX IF NOT EXISTS idx_move_game               ON move(game_id)`,
	`CREATE        INDEX IF NOT EXISTS idx_game_match              ON game(match_id)`,
}

// Bootstrap creates the full v2.7.0 schema on a fresh database and records the
// schema version. It is run by Open for an empty database and by the Database
// wrapper's SetupDatabase. It assumes an empty database: the ALTER TABLE
// statements would fail on a database that already has those columns.
func Bootstrap(ctx context.Context, db *sql.DB) error {
	for _, stmt := range schemaStatements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sqlite: bootstrap schema: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx,
		`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', ?)`,
		domain.DatabaseVersion); err != nil {
		return fmt.Errorf("sqlite: bootstrap version: %w", err)
	}
	return nil
}

// isFreshDB reports whether db has no schema yet (no metadata table).
func isFreshDB(ctx context.Context, db *sql.DB) (bool, error) {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name='metadata'`).Scan(&name)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("sqlite: probe schema: %w", err)
	}
	return false, nil
}
