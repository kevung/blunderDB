package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// schemaStatements is the full DDL for a fresh database at the current
// domain.DatabaseVersion, in dependency order. It is the ONLY schema DDL in
// the code base: Bootstrap runs it on a fresh database (the Database
// wrapper's SetupDatabase delegates to it, and so does every export file),
// and EnsureSchema derives from it — by introspecting a reference database
// built with it — what an existing database is missing on open (the Database
// wrapper's ensureAllTablesExist). The version-by-version migrations in
// database/db_migration*.go keep their own historical DDL: they describe what
// each past version looked like, not what the current one is. The parity test
// in database/schema_parity_test.go diffs the paths.
var schemaStatements = []string{
	// The CHECK constraints below are stated by a FRESH database only: SQLite
	// adds no constraint through ALTER TABLE, and rebuilding a table holding
	// hundreds of thousands of positions on upgrade would be a long,
	// disk-hungry operation to enforce what the writing code already
	// guarantees. An existing database is judged against them by
	// `blunderdb verify` instead (database/db_verify.go, CheckConstraints).
	// A NULL passes a CHECK — unknown, not violated — which is what the
	// nullable scalar columns need.
	//
	// zobrist_hash is deliberately NOT declared NOT NULL, though a row without
	// a hash is precisely the defect issue #173 set out to close (a nullable
	// column under a UNIQUE index tolerates any number of NULLs, which is how
	// the native-.db importer once slipped duplicates past idx_position_zobrist).
	// Three things stand in the way of the constraint. EnsureSchema adds a
	// missing column by ALTER TABLE, and SQLite refuses a NOT NULL column with
	// no default — so an old file that lacks the column entirely would never
	// receive it and could no longer be opened at all. repairPositionsWithoutScalars
	// (db_schema.go) exists to FIND rows with a NULL hash and fix them, and runs
	// on every open. And the constraint would hold only in files created after
	// 2.18.0, since it cannot reach the others. The rule is therefore stated
	// where it can be told the truth about every database — CheckConstraints,
	// reported by `blunderdb verify` — and the repair pass remains the remedy.
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
		-- Session cube ceiling (2.20.0, issue #271): the log2 exponent the
		-- XGID's tenth field carries, 0 when the source stated no ceiling.
		-- Reported next to the verdict, never folded into the Zobrist hash —
		-- same reasoning as has_jacoby/has_beaver (ADR-0028).
		max_cube          INTEGER NOT NULL DEFAULT 0,
		pip_1             INTEGER,
		pip_2             INTEGER,
		pip_diff          INTEGER,
		off_1             INTEGER,
		off_2             INTEGER,
		back_checkers_1   INTEGER,
		back_checkers_2   INTEGER,
		no_contact        INTEGER,
		-- Derived phase label (2.19.0, issue #264): 0 unknown, 1 opening,
		-- 2 middlegame, 3 race, 4 bearoff. Written by every path that writes
		-- a position, recomputed by the repair pass, never edited.
		game_phase        INTEGER NOT NULL DEFAULT 0,
		-- Derived plan of play (2.20.0, issue #291): 0 unknown, then the
		-- domain.GameType constants. Like game_phase it is written by every
		-- path that writes a position, recomputed by the repair pass, and
		-- never edited. Unlike it, it names the plan of the SIDE ON ROLL.
		game_type         INTEGER NOT NULL DEFAULT 0,
		occupancy_1       INTEGER,
		occupancy_2       INTEGER,
		point_mask_1      INTEGER,
		point_mask_2      INTEGER,
		state             TEXT    NOT NULL,
		is_cube_response  INTEGER NOT NULL DEFAULT 0,
		-- Provenance: set when the position entered the database on its own
		-- rather than inside a match. Sticky — see ADR-0001.
		individually_imported INTEGER NOT NULL DEFAULT 0,
		flagged INTEGER NOT NULL DEFAULT 0,
		CHECK (dice_1 BETWEEN 0 AND 6),
		CHECK (dice_2 BETWEEN 0 AND 6),
		-- cube_value is the EXPONENT (0 = cube at 1), never negative.
		CHECK (cube_value >= 0),
		CHECK (pip_1 >= 0),
		CHECK (pip_2 >= 0),
		-- off_1/off_2 are the borne-off counts, at most fifteen checkers.
		CHECK (off_1 BETWEEN 0 AND 15),
		CHECK (off_2 BETWEEN 0 AND 15)
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
		-- Who wrote this comment (2.19.0, issue #263): 'user', 'xg', 'gnubg',
		-- 'bgf' or 'unknown'. The default is deliberately 'unknown' and not
		-- 'user': a row that predates the column has no provenance, and
		-- claiming the user wrote it would make the purge spare positions it
		-- has always dropped.
		origin TEXT NOT NULL DEFAULT 'unknown',
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS command_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		command TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		scope TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS filter_library (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		command TEXT,
		edit_position TEXT,
		exclude_position TEXT,
		scope TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS search_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		command TEXT,
		position TEXT,
		exclude_position TEXT,
		timestamp INTEGER,
		scope TEXT NOT NULL DEFAULT ''
	)`,
	// UI session state (last search, last position, open views), one row per
	// key and per scope. It lived in metadata as '<scope>:session_*' rows
	// until 2.16.0; metadata is database infrastructure (schema version,
	// issuance) and holds no per-tenant data since 2.17.0 (issue #156).
	`CREATE TABLE IF NOT EXISTS session_state (
		scope TEXT NOT NULL DEFAULT '',
		key   TEXT NOT NULL,
		value TEXT,
		PRIMARY KEY (scope, key)
	)`,
	// One row per import the user launched (2.19.0, issue #257). It is what
	// lets the end-of-import report say "this file", rather than "the
	// database": the matches an import wrote point back at their batch, and
	// the batch holds the counts the report shows. Deleting a batch never
	// deletes its matches — hence ON DELETE SET NULL on the match side.
	`CREATE TABLE IF NOT EXISTS import_batch (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		started_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME,
		-- What was imported: a file path, a folder, or a short label for a
		-- paste. Shown to the user verbatim, so never normalised here.
		source TEXT NOT NULL DEFAULT '',
		-- xg | gnubg | bgf | mat | db | position | mixed
		format TEXT NOT NULL DEFAULT '',
		-- The report's figures as a JSON object (see domain.ImportBatchCounts).
		-- JSON rather than columns because the report gains figures over time
		-- and each one would otherwise be a schema bump.
		counts TEXT NOT NULL DEFAULT '{}'
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
		match_hash TEXT,
		tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL,
		last_visited_position INTEGER DEFAULT -1,
		canonical_hash TEXT,
		comment TEXT DEFAULT '',
		tournament_sort_order INTEGER DEFAULT 0,
		import_batch_id INTEGER REFERENCES import_batch(id) ON DELETE SET NULL
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
		luck_mp INTEGER,
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
		-- A LIVING collection (2.20.0, issue #282): a search query, in the
		-- grammar the command bar speaks, re-evaluated every time the
		-- collection is opened. Empty is the ordinary case — a hand-made list
		-- whose membership lives in collection_position.
		filter_query TEXT NOT NULL DEFAULT '',
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
	// The trash (2.19.0, issue #285): a SNAPSHOT of what was deleted, not a
	// deleted_at column on each table. See docs/adr/0036. A row here is dead
	// weight the live queries never see, which is the whole point: none of the
	// fifty search filters, none of the statistics, and neither backend's
	// retention predicate had to learn about it.
	`CREATE TABLE IF NOT EXISTS trash (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		-- position | collection | comment | anki_card
		kind TEXT NOT NULL,
		-- What to show in the trash list ("Position 412", a collection name…).
		label TEXT NOT NULL DEFAULT '',
		-- Everything needed to put it back, as JSON. A position snapshot holds
		-- its board and its Zobrist hash: restoring re-Saves it, so it comes
		-- back on the SAME row as before if nothing else took the hash, and
		-- merges into the existing row if something did.
		payload TEXT NOT NULL,
		deleted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_trash_deleted_at ON trash(deleted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_trash_kind ON trash(kind, deleted_at)`,
	`CREATE TABLE IF NOT EXISTS tournament (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		date TEXT,
		location TEXT,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		comment TEXT DEFAULT ''
	)`,
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
		session_limit INTEGER,
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
		suspended INTEGER NOT NULL DEFAULT 0,
		buried_until DATETIME,
		FOREIGN KEY(deck_id) REFERENCES anki_deck(id) ON DELETE CASCADE,
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
		UNIQUE(deck_id, position_id)
	)`,
	`CREATE TABLE IF NOT EXISTS anki_review_log (
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
		FOREIGN KEY(card_id) REFERENCES anki_card(id) ON DELETE CASCADE,
		-- deck_id and position_id were plain integers until 2.18.0 (issue #185):
		-- the journal named a deck and a position with nothing to say they had
		-- to exist. They do cascade in practice — the card is deleted with
		-- either, and the log row with the card — but "in practice" is what the
		-- orphans of issue #157 were made of. SQLite adds no foreign key to a
		-- table that already exists, so an upgraded database keeps the two
		-- unconstrained columns and "blunderdb verify" counts what dangles.
		FOREIGN KEY(deck_id) REFERENCES anki_deck(id) ON DELETE CASCADE,
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
		-- The four FSRS grades. anki.ScheduleNext refuses anything else
		-- (storage.ErrInvalid); go-fsrs indexed its weights with the value.
		CHECK (rating BETWEEN 1 AND 4)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_card_deck ON anki_card(deck_id)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_card_due ON anki_card(deck_id, due)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_review_log_card ON anki_review_log(card_id, reviewed_at)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_review_log_deck ON anki_review_log(deck_id, reviewed_at)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_position_zobrist        ON position(zobrist_hash)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_decision_pip   ON position(decision_type, pip_diff)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_decision_dice  ON position(decision_type, dice_1, dice_2)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_cube_response  ON position(decision_type, is_cube_response)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_individual     ON position(individually_imported) WHERE individually_imported = 1`,
	`CREATE        INDEX IF NOT EXISTS idx_position_flagged        ON position(flagged) WHERE flagged = 1`,
	`CREATE        INDEX IF NOT EXISTS idx_position_pip_diff       ON position(pip_diff)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_dice           ON position(dice_1, dice_2)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_off            ON position(off_1, off_2)`,
	// idx_position_score (match_length, score_1, score_2) is a strict column
	// prefix of idx_position_score_cube below and is dropped here (E3, index
	// redundancy pass): any seek idx_position_score could serve, the wider
	// index serves identically. ensureAllTablesExist (db_schema.go) drops it
	// from existing databases on open.
	`CREATE        INDEX IF NOT EXISTS idx_position_score_cube     ON position(match_length, score_1, score_2, cube_value)`,
	// One analysis per position, enforced rather than assumed: Save used to
	// SELECT then INSERT-or-UPDATE, so two concurrent saves inserted two rows
	// and Load took whichever the planner reached first. The index is what
	// makes the upsert in analyses_sqlite.go possible at all — an ON CONFLICT
	// target must name a UNIQUE constraint. It replaces a non-unique index of
	// the same name; the 2.18.0 migration deduplicates and drops the old one
	// so EnsureSchema builds this one in its place.
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_analysis_position       ON analysis(position_id)`,
	// Covering index for the win/gammon combo search (fiche-05 T3): the query
	// narrows via `p.id IN (SELECT position_id FROM analysis WHERE
	// player1_win_rate … AND player1_gammon_rate …)`, and with position_id as
	// the index's third column that subquery is answered from the index alone
	// (no analysis-table row lookup). Supersedes the old 2-column
	// idx_analysis_win_gammon — a new name because a 3rd-column addition to an
	// existing index does not retroactively appear in already-created SQLite
	// indexes; EnsureSchema (re)creates this index by name on every open of an
	// existing database, so no VACUUM/migration is needed to pick it up.
	// idx_analysis_win_gammon is simply no longer created here, and
	// ensureAllTablesExist (db_schema.go) drops it from existing databases on
	// open. idx_analysis_win1 (player1_win_rate alone), a strict prefix of
	// this covering index, is dropped here too (E3): any WinRateFilter-only
	// search plans identically against the wider index (verified by EXPLAIN
	// QUERY PLAN — same `SEARCH analysis USING COVERING INDEX
	// idx_analysis_win_gammon_covering (player1_win_rate>?)` with or without
	// idx_analysis_win1 present).
	`CREATE        INDEX IF NOT EXISTS idx_analysis_win_gammon_covering ON analysis(player1_win_rate, player1_gammon_rate, position_id)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_cube_error     ON analysis(cube_error)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_move_error     ON analysis(best_move_equity_error)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_is_forced      ON analysis(is_forced) WHERE is_forced = 1`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_is_close_cube  ON analysis(is_close_cube) WHERE is_close_cube = 1`,
	// The comment-presence filter (`co`/`xco`) probes this table once per search
	// with an EXISTS subquery; without the index that is a full comment scan.
	`CREATE        INDEX IF NOT EXISTS idx_comment_position        ON comment(position_id, origin)`,
	// Range filters that previously had no supporting index (full scans).
	`CREATE        INDEX IF NOT EXISTS idx_position_back_checkers_1 ON position(back_checkers_1)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_back_checkers_2 ON position(back_checkers_2)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_pip_1          ON position(pip_1)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_no_contact     ON position(no_contact) WHERE no_contact = 1`,
	`CREATE        INDEX IF NOT EXISTS idx_position_game_phase     ON position(game_phase)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_game_type      ON position(game_type)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_backgammon1    ON analysis(player1_backgammon_rate)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_win2           ON analysis(player2_win_rate)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_gammon2        ON analysis(player2_gammon_rate)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_backgammon2    ON analysis(player2_backgammon_rate)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_match_canonical         ON match(canonical_hash)`,
	`CREATE        INDEX IF NOT EXISTS idx_move_position           ON move(position_id)`,
	`CREATE        INDEX IF NOT EXISTS idx_move_game               ON move(game_id)`,
	`CREATE        INDEX IF NOT EXISTS idx_game_match              ON game(match_id)`,
	`CREATE        INDEX IF NOT EXISTS idx_command_history_scope   ON command_history(scope, timestamp)`,
	`CREATE        INDEX IF NOT EXISTS idx_search_history_scope    ON search_history(scope, timestamp)`,
	`CREATE        INDEX IF NOT EXISTS idx_filter_library_scope_name ON filter_library(scope, name)`,
}

// Bootstrap creates the full schema at domain.DatabaseVersion on a fresh
// database and records the schema version. It is run by Open for an empty
// database and by the Database wrapper's SetupDatabase. Every statement is
// CREATE ... IF NOT EXISTS, so running it on a database that already has the
// schema changes nothing but the version stamp.
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

// EnsureSchema brings an existing database up to the current schema without
// touching what it already has: tables it lacks are created, columns it lacks
// are added, indexes it lacks are built. Nothing is dropped, renamed or
// retyped. It is idempotent and cheap on a database that is already current,
// and it is what the Database wrapper runs on every open, after the migration
// chain — the chain describes the past, this describes the present.
//
// There is deliberately no second list of columns to keep in step with the
// CREATE TABLE statements above: the columns an existing database is missing
// are found by comparing it against a reference database built, in memory,
// from schemaStatements. That is what makes the DDL single-sourced — a column
// added to a CREATE TABLE above reaches existing databases without anyone
// remembering to write its ALTER TABLE twin.
//
// Tables are created strictly (a failure is returned). Columns and indexes
// are best-effort and logged: SQLite cannot add a column whose default is not
// a constant, and a UNIQUE index cannot be built over rows that violate it
// (idx_position_zobrist before the 2.1.0 dedup, idx_match_canonical before
// the empty hashes are normalised — the caller does that and calls again).
// A database in that state must still open; it worked before the index
// existed. What was logged and left out is what CheckSchema reports, so
// `blunderdb verify` shows it where a log line would go unread.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	ref, err := referenceSchema(ctx)
	if err != nil {
		return err
	}
	for _, t := range ref.tables {
		if _, err := db.ExecContext(ctx, t.createSQL); err != nil {
			return fmt.Errorf("sqlite: ensure table %s: %w", t.name, err)
		}
	}
	for _, t := range ref.tables {
		have, err := columnNames(ctx, db, t.name)
		if err != nil {
			return err
		}
		for _, c := range t.columns {
			if have[c.name] {
				continue
			}
			if err := addColumn(ctx, db, t.name, c); err != nil {
				slog.Warn("sqlite: cannot add missing column", "table", t.name, "column", c.name, "err", err)
			}
		}
	}
	for _, stmt := range ref.indexes {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			slog.Warn("sqlite: cannot ensure index", "stmt", stmt, "err", err)
		}
	}
	return nil
}

// SchemaDrift is what a database lacks against the reference schema
// (schemaStatements): the tables, columns and indexes it should have and does
// not. Only absences are listed — the schema only grows, so an element the
// reference does not name (an index a past version built, a column a user
// added) is not drift. Elements are named as the reference names them:
// tables and indexes by name, columns as table.column.
type SchemaDrift struct {
	MissingTables  []string `json:"missing_tables"`
	MissingColumns []string `json:"missing_columns"`
	MissingIndexes []string `json:"missing_indexes"`
}

// Count is the number of missing elements of every kind.
func (d SchemaDrift) Count() int {
	return len(d.MissingTables) + len(d.MissingColumns) + len(d.MissingIndexes)
}

// CheckSchema diffs db against the reference schema and reports what it
// lacks. It reads only. It is EnsureSchema's audit: EnsureSchema adds what
// it can and logs what it cannot (a UNIQUE index over rows that violate it,
// say), and this is how `blunderdb verify` surfaces those leftovers — a
// column or index missing here is one that some query will fail to name.
func CheckSchema(ctx context.Context, db *sql.DB) (SchemaDrift, error) {
	var drift SchemaDrift
	ref, err := referenceSchema(ctx)
	if err != nil {
		return drift, err
	}
	tables, err := objectNames(ctx, db, "table")
	if err != nil {
		return drift, err
	}
	for _, t := range ref.tables {
		if !tables[t.name] {
			drift.MissingTables = append(drift.MissingTables, t.name)
			continue
		}
		have, err := columnNames(ctx, db, t.name)
		if err != nil {
			return drift, err
		}
		for _, c := range t.columns {
			if !have[c.name] {
				drift.MissingColumns = append(drift.MissingColumns, t.name+"."+c.name)
			}
		}
	}
	indexes, err := objectNames(ctx, db, "index")
	if err != nil {
		return drift, err
	}
	for _, name := range ref.indexNames {
		if !indexes[name] {
			drift.MissingIndexes = append(drift.MissingIndexes, name)
		}
	}
	return drift, nil
}

// objectNames returns the set of names sqlite_master holds of that type
// ("table", "index"), the automatic ones (sqlite_autoindex_*, sqlite_stat*)
// included — they never collide with a reference name.
func objectNames(ctx context.Context, db *sql.DB, typ string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = ?`, typ)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list %ss: %w", typ, err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("sqlite: list %ss: %w", typ, err)
		}
		out[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: list %ss: %w", typ, err)
	}
	return out, nil
}

// addColumn adds c to table. SQLite refuses to add a column whose default is
// not a constant (DEFAULT CURRENT_TIMESTAMP, say); rather than leave the column
// out — every SELECT naming it would then fail — it is added without its
// default, which only costs new rows the automatic value, and said so.
func addColumn(ctx context.Context, db *sql.DB, table string, c referenceColumn) error {
	_, err := db.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN `+c.definition)
	if err == nil || c.defaultClause == "" {
		return err
	}
	withoutDefault := strings.Replace(c.definition, " "+c.defaultClause, "", 1)
	if _, retryErr := db.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN `+withoutDefault); retryErr != nil {
		return err
	}
	slog.Warn("sqlite: column added without its default", "table", table, "column", c.name, "default", c.defaultClause, "err", err)
	return nil
}

// referenceTable is one table of the reference schema: the statement that
// creates it and, for every non-primary-key column, the definition an ALTER
// TABLE ADD COLUMN needs to reproduce it.
type referenceTable struct {
	name      string
	createSQL string
	columns   []referenceColumn
}

type referenceColumn struct {
	name          string
	definition    string // `name TYPE [NOT NULL] [DEFAULT x] [REFERENCES t(c) ON DELETE …]`
	defaultClause string // `DEFAULT x` when the column has one, else ""
}

type reference struct {
	tables     []referenceTable
	indexes    []string // the CREATE INDEX statements, in schema order
	indexNames []string // the names those statements declare, in schema order
}

// referenceSchema builds the current schema in memory and reads it back.
// pragma_table_info gives every column's declared type, NOT NULL and default
// exactly as written; pragma_foreign_key_list gives the REFERENCES clause a
// column carries. Together they reconstruct the ADD COLUMN definition without
// parsing the CREATE TABLE text.
func referenceSchema(ctx context.Context) (*reference, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("sqlite: open reference schema: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := Bootstrap(ctx, db); err != nil {
		return nil, err
	}

	ref := &reference{}
	for _, stmt := range schemaStatements {
		switch {
		case strings.HasPrefix(stmt, "CREATE TABLE IF NOT EXISTS "):
			name := strings.Fields(stmt[len("CREATE TABLE IF NOT EXISTS "):])[0]
			name = strings.TrimSuffix(name, "(")
			ref.tables = append(ref.tables, referenceTable{name: name, createSQL: stmt})
		case strings.HasPrefix(stmt, "CREATE INDEX IF NOT EXISTS "),
			strings.HasPrefix(stmt, "CREATE UNIQUE INDEX IF NOT EXISTS "),
			strings.HasPrefix(stmt, "CREATE        INDEX IF NOT EXISTS "):
			ref.indexes = append(ref.indexes, stmt)
			// CREATE [UNIQUE] INDEX IF NOT EXISTS <name> ON ...
			afterExists := stmt[strings.Index(stmt, " EXISTS ")+len(" EXISTS "):]
			ref.indexNames = append(ref.indexNames, strings.Fields(afterExists)[0])
		}
	}
	for i := range ref.tables {
		cols, err := referenceColumns(ctx, db, ref.tables[i].name)
		if err != nil {
			return nil, err
		}
		ref.tables[i].columns = cols
	}
	return ref, nil
}

func referenceColumns(ctx context.Context, db *sql.DB, table string) ([]referenceColumn, error) {
	refs, err := foreignKeyClauses(ctx, db, table)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT name, type, "notnull", dflt_value, pk FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		return nil, fmt.Errorf("sqlite: reference columns of %s: %w", table, err)
	}
	defer rows.Close()
	var out []referenceColumn
	for rows.Next() {
		var name, typ string
		var dflt sql.NullString
		var notnull, pk int
		if err := rows.Scan(&name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("sqlite: reference columns of %s: %w", table, err)
		}
		if pk != 0 {
			continue // a primary key is never added after the fact
		}
		def := name + " " + typ
		if notnull != 0 {
			def += " NOT NULL"
		}
		col := referenceColumn{name: name}
		if dflt.Valid {
			col.defaultClause = "DEFAULT " + dflt.String
			def += " " + col.defaultClause
		}
		if clause, ok := refs[name]; ok {
			def += " " + clause
		}
		col.definition = def
		out = append(out, col)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: reference columns of %s: %w", table, err)
	}
	return out, nil
}

// foreignKeyClauses returns, per column of table, the REFERENCES clause it
// declares (only single-column keys — the schema has no composite ones).
func foreignKeyClauses(ctx context.Context, db *sql.DB, table string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT "table", "from", COALESCE("to", ''), on_update, on_delete FROM pragma_foreign_key_list(?)`, table)
	if err != nil {
		return nil, fmt.Errorf("sqlite: reference foreign keys of %s: %w", table, err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var ref, from, to, onUpdate, onDelete string
		if err := rows.Scan(&ref, &from, &to, &onUpdate, &onDelete); err != nil {
			return nil, fmt.Errorf("sqlite: reference foreign keys of %s: %w", table, err)
		}
		clause := "REFERENCES " + ref + "(" + to + ")"
		if onUpdate != "NO ACTION" {
			clause += " ON UPDATE " + onUpdate
		}
		if onDelete != "NO ACTION" {
			clause += " ON DELETE " + onDelete
		}
		out[from] = clause
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: reference foreign keys of %s: %w", table, err)
	}
	return out, nil
}

// columnNames returns the set of columns table currently has.
func columnNames(ctx context.Context, db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, fmt.Errorf("sqlite: columns of %s: %w", table, err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("sqlite: columns of %s: %w", table, err)
		}
		out[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: columns of %s: %w", table, err)
	}
	return out, nil
}
