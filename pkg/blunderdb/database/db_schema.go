package database

import (
	"fmt"
	"strings"
)

// ensureAllTablesExist creates any missing tables and columns that should exist
// at the current database version. This repairs databases that were migrated
// through code paths that skipped creating some schema elements.
func (d *Database) ensureAllTablesExist() error {
	// v1.1.0: command_history
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			scope TEXT NOT NULL DEFAULT ''
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
			edit_position TEXT,
			exclude_position TEXT,
			scope TEXT NOT NULL DEFAULT ''
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
			exclude_position TEXT,
			timestamp INTEGER,
			scope TEXT NOT NULL DEFAULT ''
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
			luck_mp INTEGER,
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

	// Ensure collection timestamps are not NULL (repair old databases)
	_, _ = d.db.Exec(`UPDATE collection SET created_at = datetime('now') WHERE created_at IS NULL OR created_at = ''`)
	_, _ = d.db.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE updated_at IS NULL OR updated_at = ''`)

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
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)
	// last_visited_position (v1.7.0)
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)
	// canonical_hash (v1.7.0)
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN canonical_hash TEXT`)

	// match comment
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN comment TEXT DEFAULT ''`)
	// tournament_sort_order (ordering within a tournament)
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_sort_order INTEGER DEFAULT 0`)
	// tournament comment
	_, _ = d.db.Exec(`ALTER TABLE tournament ADD COLUMN comment TEXT DEFAULT ''`)

	// Ensure columns added in later versions exist on comment table
	// created_at (v1.8.0)
	_, _ = d.db.Exec(`ALTER TABLE comment ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP`)
	// modified_at (v1.9.0)
	_, _ = d.db.Exec(`ALTER TABLE comment ADD COLUMN modified_at DATETIME`)

	// v1.8.0: anki_deck, anki_card
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
		return fmt.Errorf("error ensuring anki_deck table: %w", err)
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
			suspended INTEGER NOT NULL DEFAULT 0,
			buried_until DATETIME,
			FOREIGN KEY(deck_id) REFERENCES anki_deck(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(deck_id, position_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring anki_card table: %w", err)
	}

	// v2.12.0: ensure anki_card suspend/bury columns exist on databases created
	// before they were added (ALTER TABLE is a no-op if the column exists).
	for _, stmt := range []string{
		`ALTER TABLE anki_card ADD COLUMN suspended INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE anki_card ADD COLUMN buried_until DATETIME`,
	} {
		_, _ = d.db.Exec(stmt) // ignore error: column may already exist
	}

	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_anki_card_deck ON anki_card(deck_id)`)
	if err != nil {
		return fmt.Errorf("error ensuring anki_card deck index: %w", err)
	}

	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_anki_card_due ON anki_card(deck_id, due)`)
	if err != nil {
		return fmt.Errorf("error ensuring anki_card due index: %w", err)
	}

	// v2.11.0: anki_review_log (append-only review journal)
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS anki_review_log (
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
			reviewed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(card_id) REFERENCES anki_card(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("error ensuring anki_review_log table: %w", err)
	}

	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_anki_review_log_card ON anki_review_log(card_id, reviewed_at)`)
	if err != nil {
		return fmt.Errorf("error ensuring anki_review_log card index: %w", err)
	}

	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_anki_review_log_deck ON anki_review_log(deck_id, reviewed_at)`)
	if err != nil {
		return fmt.Errorf("error ensuring anki_review_log deck index: %w", err)
	}

	// v2.0.0: ensure new position/analysis columns exist (ALTER TABLE is a no-op if column already exists)
	newPositionCols := []string{
		`ALTER TABLE position ADD COLUMN zobrist_hash    INTEGER`,
		`ALTER TABLE position ADD COLUMN decision_type  INTEGER`,
		`ALTER TABLE position ADD COLUMN player_on_roll INTEGER`,
		`ALTER TABLE position ADD COLUMN dice_1         INTEGER`,
		`ALTER TABLE position ADD COLUMN dice_2         INTEGER`,
		`ALTER TABLE position ADD COLUMN cube_value     INTEGER`,
		`ALTER TABLE position ADD COLUMN cube_owner     INTEGER`,
		`ALTER TABLE position ADD COLUMN score_1        INTEGER`,
		`ALTER TABLE position ADD COLUMN score_2        INTEGER`,
		`ALTER TABLE position ADD COLUMN match_length   INTEGER`,
		`ALTER TABLE position ADD COLUMN has_jacoby     INTEGER`,
		`ALTER TABLE position ADD COLUMN has_beaver     INTEGER`,
		`ALTER TABLE position ADD COLUMN pip_1          INTEGER`,
		`ALTER TABLE position ADD COLUMN pip_2          INTEGER`,
		`ALTER TABLE position ADD COLUMN pip_diff       INTEGER`,
		`ALTER TABLE position ADD COLUMN off_1          INTEGER`,
		`ALTER TABLE position ADD COLUMN off_2          INTEGER`,
		`ALTER TABLE position ADD COLUMN back_checkers_1 INTEGER`,
		`ALTER TABLE position ADD COLUMN back_checkers_2 INTEGER`,
		`ALTER TABLE position ADD COLUMN no_contact     INTEGER`,
		`ALTER TABLE position ADD COLUMN occupancy_1    INTEGER`,
		`ALTER TABLE position ADD COLUMN occupancy_2    INTEGER`,
		`ALTER TABLE position ADD COLUMN point_mask_1   INTEGER`,
		`ALTER TABLE position ADD COLUMN point_mask_2   INTEGER`,
		`ALTER TABLE position ADD COLUMN is_cube_response INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE position ADD COLUMN individually_imported INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE position ADD COLUMN flagged INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range newPositionCols {
		_, _ = d.db.Exec(stmt) // ignore error: column may already exist
	}

	// Nullable on purpose: NULL means "luck unknown for this roll", which is
	// not the same as a neutral roll (see ADR-0010), so no DEFAULT here.
	newMoveCols := []string{
		`ALTER TABLE move ADD COLUMN luck_mp INTEGER`,
	}
	for _, stmt := range newMoveCols {
		_, _ = d.db.Exec(stmt) // ignore error: column may already exist
	}

	// Declared INTEGER to match the fresh schema (schema_sqlite.go). The
	// 1.9.0→2.0.0 migration declared these REAL and databases it upgraded keep
	// that; the difference is affinity-only (a fractional value is stored REAL
	// under either) and cannot be changed in place. This list only fires on a
	// database that somehow lacks the columns, so it follows the fresh DDL.
	newAnalysisCols := []string{
		`ALTER TABLE analysis ADD COLUMN best_cube_action        TEXT`,
		`ALTER TABLE analysis ADD COLUMN cube_error              INTEGER`,
		`ALTER TABLE analysis ADD COLUMN best_move_equity_error  INTEGER`,
		`ALTER TABLE analysis ADD COLUMN player1_win_rate        INTEGER`,
		`ALTER TABLE analysis ADD COLUMN player1_gammon_rate     INTEGER`,
		`ALTER TABLE analysis ADD COLUMN player1_backgammon_rate INTEGER`,
		`ALTER TABLE analysis ADD COLUMN player2_win_rate        INTEGER`,
		`ALTER TABLE analysis ADD COLUMN player2_gammon_rate     INTEGER`,
		`ALTER TABLE analysis ADD COLUMN player2_backgammon_rate INTEGER`,
	}
	for _, stmt := range newAnalysisCols {
		_, _ = d.db.Exec(stmt) // ignore error: column may already exist
	}

	// v2.0.0 indexes — non-unique ones are safe to add to existing DBs
	v2indexesSafe := []string{
		`CREATE INDEX IF NOT EXISTS idx_position_decision_pip   ON position(decision_type, pip_diff)`,
		`CREATE INDEX IF NOT EXISTS idx_position_decision_dice  ON position(decision_type, dice_1, dice_2)`,
		`CREATE INDEX IF NOT EXISTS idx_position_cube_response  ON position(decision_type, is_cube_response)`,
		`CREATE INDEX IF NOT EXISTS idx_position_pip_diff       ON position(pip_diff)`,
		`CREATE INDEX IF NOT EXISTS idx_position_dice           ON position(dice_1, dice_2)`,
		`CREATE INDEX IF NOT EXISTS idx_position_off            ON position(off_1, off_2)`,
		// idx_position_score (E3, index redundancy pass): dropped from this list
		// — a strict column prefix of idx_position_score_cube below, so any seek
		// it could serve, the wider index serves identically (verified by
		// EXPLAIN QUERY PLAN). Existing databases that still carry it lose it
		// in legacyIndexes below.
		`CREATE INDEX IF NOT EXISTS idx_position_score_cube     ON position(match_length, score_1, score_2, cube_value)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_position       ON analysis(position_id)`,
		// Covering index for the win/gammon combo search (fiche-05 T3) — see the
		// long comment on its twin in schema_sqlite.go. This list runs on every
		// open of an existing database (ensureAllTablesExist, called
		// unconditionally at the end of runMigrationChain), so listing the new
		// name here is what actually gets already-created databases the index —
		// no separate migration or VACUUM needed. idx_analysis_win_gammon
		// (2-column) and idx_analysis_win1 (E3: a strict prefix of the covering
		// index, same reasoning as idx_position_score above) are dropped from
		// existing databases in legacyIndexes below.
		`CREATE INDEX IF NOT EXISTS idx_analysis_win_gammon_covering ON analysis(player1_win_rate, player1_gammon_rate, position_id)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_cube_error     ON analysis(cube_error)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_move_error     ON analysis(best_move_equity_error)`,
		// The comment-presence filter (`co`/`xco`) probes this table once per
		// search with an EXISTS subquery; without the index that is a full
		// comment scan. Non-unique, so it is safe on existing DBs and needs no
		// DatabaseVersion bump — nothing schema-visible changes.
		`CREATE INDEX IF NOT EXISTS idx_comment_position        ON comment(position_id)`,
		// Range filters that previously had no supporting index (full scans).
		`CREATE INDEX IF NOT EXISTS idx_position_back_checkers_1 ON position(back_checkers_1)`,
		`CREATE INDEX IF NOT EXISTS idx_position_back_checkers_2 ON position(back_checkers_2)`,
		`CREATE INDEX IF NOT EXISTS idx_position_pip_1          ON position(pip_1)`,
		`CREATE INDEX IF NOT EXISTS idx_position_no_contact     ON position(no_contact) WHERE no_contact = 1`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_backgammon1    ON analysis(player1_backgammon_rate)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_win2           ON analysis(player2_win_rate)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_gammon2        ON analysis(player2_gammon_rate)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_backgammon2    ON analysis(player2_backgammon_rate)`,
		`CREATE INDEX IF NOT EXISTS idx_move_position           ON move(position_id)`,
		`CREATE INDEX IF NOT EXISTS idx_move_game               ON move(game_id)`,
		`CREATE INDEX IF NOT EXISTS idx_game_match              ON game(match_id)`,
		// v2.9.0 per-scope history/filter indexes. The 2.8.0→2.9.0 migration
		// creates them, but a database created fresh by an older SetupDatabase
		// (which never listed them) went without; the headless daemon serving
		// such a file paid a full scan per tenant lookup.
		`CREATE INDEX IF NOT EXISTS idx_command_history_scope   ON command_history(scope, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_search_history_scope    ON search_history(scope, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_filter_library_scope_name ON filter_library(scope, name)`,
	}
	for _, idx := range v2indexesSafe {
		_, _ = d.db.Exec(idx) // ignore error: index may already exist or column may be NULL
	}

	// Indexes the fresh schema no longer declares (E3 redundancy pass and the
	// fiche-05 covering index): each is a strict column prefix of an index in
	// the list above, so nothing plans differently without it. Dropping is
	// perf-only — no query names an index — hence no DatabaseVersion bump.
	legacyIndexes := []string{
		`DROP INDEX IF EXISTS idx_position_score`,
		`DROP INDEX IF EXISTS idx_analysis_win_gammon`,
		`DROP INDEX IF EXISTS idx_analysis_win1`,
	}
	for _, stmt := range legacyIndexes {
		_, _ = d.db.Exec(stmt)
	}

	// idx_position_zobrist (UNIQUE) is only safe after the 2.0.0→2.1.0 dedup
	// migration, which creates it. idx_match_canonical (UNIQUE) was never
	// created for migrated databases, so cross-format duplicate detection ran
	// unindexed there. Import writes an empty canonical hash as NULL
	// (matches_sqlite.go nullableString); older imports stored '' and two of
	// those would collide, so normalise first. Best-effort: a database that
	// genuinely holds two matches with the same canonical hash keeps working
	// without the index (FindByHash is a plain SELECT), it just stays unindexed.
	_, _ = d.db.Exec(`UPDATE match SET canonical_hash = NULL WHERE canonical_hash = ''`)
	_, _ = d.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_match_canonical ON match(canonical_hash)`)

	return nil
}

func (d *Database) CheckVersion(databaseVersion string) error {
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
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
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return "", err
	}
	return dbVersion, nil
}

func (d *Database) GetDatabaseVersion() (string, error) {
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	return DatabaseVersion, nil
}

func (d *Database) LoadMetadata() (map[string]string, error) {
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	rows, err := d.db.Query(`SELECT key, value FROM metadata WHERE key IN ('user', 'description', 'dateOfCreation', 'database_version')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadata := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
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
			return err
		}
	}
	return tx.Commit()
}
