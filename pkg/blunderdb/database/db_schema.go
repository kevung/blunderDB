package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
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

// repairPositionsWithoutScalars fills in the Zobrist hash and the scalar search
// columns of position rows that were stored without them, and folds the
// duplicates that omission let through back onto the row the hash index
// already holds.
//
// It repairs the damage of one bug: until 2026-09 the native .db importer
// (CommitImportDatabase) inserted a position the target did not hold yet with
// its state alone — no hash, no pip counts, no dice/score/cube columns. Such a
// row was invisible to every SQL filter (they read the columns, never the
// state), never took part in hash deduplication, and — because
// ReconstructPosition trusts the columns over the state — no longer matched
// itself on the next import of the same database, which inserted it again.
// The importer now writes through PositionStore.Save; this pass is for the
// databases that already carry those rows.
//
// It runs on every open, after the migration chain, and needs no schema
// version bump: nothing about the schema changes, the columns merely get the
// values they were always meant to hold. It is idempotent and cheap when there
// is nothing to do — a single probe on idx_position_zobrist finds no NULL
// row. A row whose state cannot be decoded is left alone (and reported), so
// a corrupted row cannot block the open; it will be reported again next time.
//
// Rows are repaired in one transaction: a crash mid-way leaves the database
// exactly as it was, and the next open picks the whole batch up again.
//
// The caller must hold d.mu.
func (d *Database) repairPositionsWithoutScalars(ctx context.Context) error {
	rows, err := d.db.QueryContext(ctx, `SELECT `+positionSelectCols+`
		FROM position WHERE zobrist_hash IS NULL ORDER BY id`)
	if err != nil {
		return fmt.Errorf("listing positions without scalar columns: %w", err)
	}
	type defective struct {
		pos   Position
		state string
	}
	var todo []defective
	for rows.Next() {
		var state string
		pos, err := scanPositionRowWithState(rows, &state)
		if err != nil {
			rows.Close()
			return fmt.Errorf("scanning position without scalar columns: %w", err)
		}
		todo = append(todo, defective{pos, state})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(todo) == 0 {
		return nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	positions := sqlite.WrapTx(tx).Positions()

	var repaired, merged, undecodable int
	for i, item := range todo {
		if err := ctx.Err(); err != nil {
			return err
		}
		pos := item.pos
		// The bug wrote the full position as JSON, and the NULL columns say
		// nothing; the state is the only faithful record of dice, score and
		// cube, so it — not the zeroed columns — is what gets re-read. A compact
		// state carries the board alone and has no better source to offer.
		if !isCompactState(item.state) {
			var decoded Position
			if err := json.Unmarshal([]byte(item.state), &decoded); err != nil {
				slog.Warn("position has no scalar columns and its state cannot be decoded; left as is", "id", pos.ID, "err", err)
				undecodable++
				continue
			}
			decoded.ID = pos.ID
			decoded.IndividuallyImported = pos.IndividuallyImported
			decoded.Flagged = pos.Flagged
			pos = decoded
		}
		// A board with no checker on it is not a position blunderDB ever
		// wrote — it is a placeholder (test fixtures, hand-edited files) — and
		// every such row would hash alike and be folded into one. Leave them.
		if boardIsEmpty(pos.Board) {
			slog.Warn("position has no scalar columns and an empty board; left as is", "id", pos.ID)
			undecodable++
			continue
		}
		norm := pos.NormalizeForStorage()
		norm.ID = pos.ID

		keepID, held, err := positions.Exists(ctx, "", engine.ZobristHash(&norm))
		if err != nil {
			return err
		}
		if held && keepID != norm.ID {
			if err := mergePositionInto(ctx, tx, keepID, norm.ID); err != nil {
				return err
			}
			merged++
		} else {
			if err := positions.Update(ctx, "", &norm); err != nil {
				return err
			}
			repaired++
		}
		d.emitMigrationProgress("position_scalars", i+1, len(todo))
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing position scalar repair: %w", err)
	}
	slog.Info("repaired positions stored without scalar columns", "repaired", repaired, "merged", merged, "undecodable", undecodable)
	return nil
}

// boardIsEmpty reports whether b carries no checker at all, on or off the board.
func boardIsEmpty(b Board) bool {
	for _, pt := range b.Points {
		if pt.Checkers > 0 {
			return false
		}
	}
	return b.Bearoff[0] == 0 && b.Bearoff[1] == 0
}

// scanPositionRowWithState is scanPositionRow that also hands back the raw
// state column, for the one caller that needs to re-read it.
func scanPositionRowWithState(rows *sql.Rows, state *string) (Position, error) {
	var id int64
	var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
	var individual, flagged sql.NullBool
	if err := rows.Scan(&id, state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &individual, &flagged); err != nil {
		return Position{}, err
	}
	pos := reconstructPosition(id, *state,
		int(dt.Int64), int(por.Int64), int(d1.Int64), int(d2.Int64),
		int(cv.Int64), int(co.Int64), int(s1.Int64), int(s2.Int64),
		int(hj.Int64), int(hb.Int64))
	pos.IndividuallyImported = individual.Bool
	pos.Flagged = flagged.Bool
	return pos, nil
}

// mergePositionInto moves everything attached to the duplicate position dupID
// onto keepID — the row the Zobrist index already holds — and deletes dupID.
// Match moves, collection memberships, Anki cards and comments follow the
// position; an analysis follows only when keepID has none (it is one per
// position, and the held row's own analysis wins); the sticky marks
// (individually_imported, flagged — ADR-0001, ADR-0006) are raised on keepID
// when dupID carried them and never lowered. Whatever cannot be re-pointed
// (a membership keepID already has) goes with dupID through ON DELETE CASCADE.
func mergePositionInto(ctx context.Context, tx *sql.Tx, keepID, dupID int64) error {
	for _, stmt := range []string{
		`UPDATE move SET position_id = ? WHERE position_id = ?`,
		`UPDATE OR IGNORE collection_position SET position_id = ? WHERE position_id = ?`,
		`UPDATE OR IGNORE anki_card SET position_id = ? WHERE position_id = ?`,
		`UPDATE anki_review_log SET position_id = ? WHERE position_id = ?`,
		`UPDATE comment SET position_id = ? WHERE position_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, keepID, dupID); err != nil {
			return fmt.Errorf("merging duplicate position %d into %d: %w", dupID, keepID, err)
		}
	}
	// The analysis blob names its position inside the JSON as well as in the
	// row, so it is re-encoded rather than merely re-pointed.
	var keepHasAnalysis bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM analysis WHERE position_id = ?)`, keepID).Scan(&keepHasAnalysis); err != nil {
		return fmt.Errorf("merging duplicate position %d into %d: %w", dupID, keepID, err)
	}
	if !keepHasAnalysis {
		var data []byte
		switch err := tx.QueryRowContext(ctx, `SELECT data FROM analysis WHERE position_id = ?`, dupID).Scan(&data); {
		case err == nil:
			analysis, err := decodeAnalysisFromStorage(data)
			if err != nil {
				return fmt.Errorf("merging duplicate position %d into %d: decode analysis: %w", dupID, keepID, err)
			}
			analysis.PositionID = int(keepID)
			encoded, err := encodeAnalysisForStorage(&analysis)
			if err != nil {
				return fmt.Errorf("merging duplicate position %d into %d: encode analysis: %w", dupID, keepID, err)
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE analysis SET position_id = ?, data = ? WHERE position_id = ?`, keepID, encoded, dupID); err != nil {
				return fmt.Errorf("merging duplicate position %d into %d: %w", dupID, keepID, err)
			}
		case errors.Is(err, sql.ErrNoRows):
		default:
			return fmt.Errorf("merging duplicate position %d into %d: %w", dupID, keepID, err)
		}
	}
	for _, mark := range []string{"individually_imported", "flagged"} {
		if _, err := tx.ExecContext(ctx,
			`UPDATE position SET `+mark+` = 1 WHERE id = ? AND `+mark+` = 0
			   AND (SELECT `+mark+` FROM position WHERE id = ?) = 1`, keepID, dupID); err != nil {
			return fmt.Errorf("merging duplicate position %d into %d: %w", dupID, keepID, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM position WHERE id = ?`, dupID); err != nil {
		return fmt.Errorf("deleting duplicate position %d: %w", dupID, err)
	}
	return nil
}
