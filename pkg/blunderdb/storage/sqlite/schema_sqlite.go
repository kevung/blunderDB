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
		state             TEXT    NOT NULL,
		is_cube_response  INTEGER NOT NULL DEFAULT 0,
		-- Provenance: set when the position entered the database on its own
		-- rather than inside a match. Sticky — see ADR-0001.
		individually_imported INTEGER NOT NULL DEFAULT 0,
		flagged INTEGER NOT NULL DEFAULT 0
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
		FOREIGN KEY(card_id) REFERENCES anki_card(id) ON DELETE CASCADE
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
	`CREATE        INDEX IF NOT EXISTS idx_analysis_position       ON analysis(position_id)`,
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
	`CREATE        INDEX IF NOT EXISTS idx_comment_position        ON comment(position_id)`,
	// Range filters that previously had no supporting index (full scans).
	`CREATE        INDEX IF NOT EXISTS idx_position_back_checkers_1 ON position(back_checkers_1)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_back_checkers_2 ON position(back_checkers_2)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_pip_1          ON position(pip_1)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_no_contact     ON position(no_contact) WHERE no_contact = 1`,
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
// existed.
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
	tables  []referenceTable
	indexes []string
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
