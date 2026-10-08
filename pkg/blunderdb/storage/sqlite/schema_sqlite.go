package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// schemaStatements is the full DDL for a fresh database at the current
// domain.DatabaseVersion, in dependency order, and the ONLY current-schema
// DDL: Bootstrap runs it on a fresh database, EnsureSchema diffs an existing
// one against a reference built from it. The migrations in
// database/db_migration*.go keep their own historical DDL;
// database/schema_parity_test.go diffs the paths.
var schemaStatements = []string{
	// The CHECK constraints below bind a FRESH database only: SQLite adds no
	// constraint through ALTER TABLE, and rebuilding large tables on upgrade
	// is not worth it. Existing databases are judged by `blunderdb verify`
	// (database/db_verify.go, CheckConstraints). A NULL passes a CHECK, which
	// the nullable scalar columns need.
	//
	// zobrist_hash is deliberately NOT NOT NULL, though a NULL hash is a
	// defect (a UNIQUE index tolerates any number of NULLs): EnsureSchema
	// could not ALTER such a column into an old file (no default),
	// repairPositionsWithoutScalars (db_schema.go) must be able to find and
	// fix NULL rows on open, and the constraint would bind new files only.
	// CheckConstraints states the rule for every database instead.
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
		-- The board, 28 signed bytes (engine.EncodeBoardState, ADR-0071).
		-- Declared TEXT so that a migrated library and a fresh one share one
		-- declaration: TEXT affinity stores a BLOB as it is. Rows older than
		-- 2.31.0 the migration could not read keep their text;
		-- engine.DecodeBoardCompact reads every form.
		state             TEXT    NOT NULL,
		is_cube_response  INTEGER NOT NULL DEFAULT 0,
		-- Provenance: set when the position entered the database on its own
		-- rather than inside a match. Sticky — see ADR-0001.
		individually_imported INTEGER NOT NULL DEFAULT 0,
		flagged INTEGER NOT NULL DEFAULT 0,
		-- Date of the earliest match that reaches this position, in Unix
		-- seconds UTC (ADR-0071, derived by sqlite.UnixFromMatchDateSQL);
		-- NULL when no match reaches it. Denormalised so a date filter reads
		-- one indexed column instead of joining move, game and match; the
		-- match store keeps it true when a move is written, a match date is
		-- edited or a match is deleted.
		match_date INTEGER,
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
		-- Action code (domain.ActionCode, ADR-0071); read through
		-- sqlshared.ActionLabelSQL.
		best_cube_action            INTEGER,
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
		-- Provenance of the verdict (engine.AnalysisProvenance): the
		-- engine label, the depth as domain.AnalysisDepthRank, and the blob's
		-- CreationDate in Unix seconds UTC (ADR-0071). NULL analysis_engine
		-- means "not derived yet": a row written before the column, or by a
		-- path that writes the blob alone;
		-- the open-time pass (database.backfillAnalysisProvenance) fills it.
		analysis_engine             TEXT,
		analysis_depth              INTEGER,
		creation_date               INTEGER,
		-- The match equity table the verdict was computed with (ADR-0068):
		-- the id of a match_equity_table row (whose digest names the table),
		-- NULL for the built-in Kazaross-XG2. An analysis whose table differs
		-- from the library's current one is shown as "different MET" and
		-- left out of the comparisons. An integer rather than the digest
		-- itself: eight bytes in place of a 64-character hex string.
		-- Deleting the table falls back to the built-in one.
		met_id                      INTEGER REFERENCES match_equity_table(id) ON DELETE SET NULL,
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
		-- The person behind the comment, free text ('' when unknown), beside
		-- origin, which only says which program carried it: a coach and a
		-- student annotating one shared library are two authors of origin
		-- 'user'. A position may carry several comments.
		author TEXT NOT NULL DEFAULT '',
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
	// key and per scope. metadata is database infrastructure (schema version,
	// issuance) and holds no per-tenant data.
	`CREATE TABLE IF NOT EXISTS session_state (
		scope TEXT NOT NULL DEFAULT '',
		key   TEXT NOT NULL,
		value TEXT,
		PRIMARY KEY (scope, key)
	)`,
	// One row per import the user launched: the matches it wrote point back
	// at it, and it holds the counts the end-of-import report shows. Deleting
	// a batch never deletes its matches — hence ON DELETE SET NULL there.
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
	// One row per file a batch met: what it was (path, size, mtime,
	// SHA-256) and what it gave. outcome is new | duplicate | enriched |
	// error; match_id is the new match, or the match that covers a duplicate;
	// error holds the message. A resumed batch skips a file already listed
	// with the same path, size and mtime. Import data, never a reading mark
	// (ADR-0007).
	`CREATE TABLE IF NOT EXISTS import_batch_file (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		batch_id INTEGER NOT NULL,
		path     TEXT    NOT NULL,
		size     INTEGER NOT NULL DEFAULT 0,
		mtime    DATETIME,
		sha256   TEXT    NOT NULL DEFAULT '',
		outcome  TEXT    NOT NULL DEFAULT '',
		match_id INTEGER,
		error    TEXT    NOT NULL DEFAULT '',
		FOREIGN KEY(batch_id) REFERENCES import_batch(id) ON DELETE CASCADE,
		FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE SET NULL
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
		-- Who signs match.comment (domain.Match.CommentAuthor), '' when
		-- unknown.
		comment_author TEXT NOT NULL DEFAULT '',
		tournament_sort_order INTEGER DEFAULT 0,
		import_batch_id INTEGER REFERENCES import_batch(id) ON DELETE SET NULL,
		-- The Slot this Match fills in its Tournament's Direction (ADR-0047): the
		-- Nicomaque match id ("M12"), empty when the Match fills no Slot. A Match
		-- fills at most one Slot and a Slot carries at most one Match, which the
		-- unique index below enforces per tournament.
		direction_match_id TEXT DEFAULT '',
		-- Length, initial score and the dice of every game, in order: the
		-- same match under other player names has the same dice_hash, which
		-- match_hash and canonical_hash (both over the names) cannot see.
		-- NULL until computed.
		dice_hash TEXT,
		-- What the source file says of the players and the session, NULL or
		-- '' when it says nothing: XG rating and experience per seat, the
		-- transcriber, the Jacoby and Beaver rules of a money session, and
		-- the program version that wrote the analyses.
		player1_elo REAL,
		player2_elo REAL,
		player1_experience INTEGER,
		player2_experience INTEGER,
		transcriber TEXT DEFAULT '',
		has_jacoby INTEGER,
		has_beaver INTEGER,
		engine_version TEXT DEFAULT ''
	)`,
	`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`,
	`CREATE INDEX IF NOT EXISTS idx_match_dice_hash ON match(dice_hash) WHERE dice_hash IS NOT NULL`,
	// The per-match, per-seat tallies the corpus statistics read instead of
	// re-aggregating move × analysis on every request, with the arithmetic
	// of sqlshared/stats.go: decisions counted by countedExpr, error_mp the
	// sum of their statsErrExpr in millipoints, pr = 500 × error_mp / 1000 /
	// decisions (NULL without a decision), blunders at the library's
	// threshold (a threshold change recomputes the table), luck_mp the sum of
	// move.luck_mp over the luck_rolls rolls that carry one. analysis_engine
	// and analysis_depth are the provenance most of the seat's decisions
	// carry. checker_error_mp / cube_error_mp split error_mp by decision
	// type, errors counts the decisions at the error threshold. The Snowie
	// rate's parts ignore countedExpr: snowie_error_mp is every analysed
	// decision's error, snowie_moves the analysed checker decisions, and
	// checker_moves every checker decision with a position, analysed or not
	// (the players table's denominator). Those six are NULL on a row an
	// earlier 2.30.0 build wrote, which the open recomputes. A derived table:
	// always recomputable from the moves, and it holds no position.
	`CREATE TABLE IF NOT EXISTS match_stats (
		match_id INTEGER NOT NULL REFERENCES match(id) ON DELETE CASCADE,
		seat INTEGER NOT NULL CHECK (seat IN (1, 2)),
		decisions INTEGER NOT NULL DEFAULT 0,
		checker_decisions INTEGER NOT NULL DEFAULT 0,
		cube_decisions INTEGER NOT NULL DEFAULT 0,
		error_mp INTEGER NOT NULL DEFAULT 0,
		pr REAL,
		luck_mp INTEGER NOT NULL DEFAULT 0,
		luck_rolls INTEGER NOT NULL DEFAULT 0,
		blunders INTEGER NOT NULL DEFAULT 0,
		analysis_engine TEXT,
		analysis_depth INTEGER,
		computed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		checker_error_mp INTEGER,
		cube_error_mp INTEGER,
		errors INTEGER,
		snowie_error_mp INTEGER,
		snowie_moves INTEGER,
		checker_moves INTEGER,
		PRIMARY KEY (match_id, seat)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_match_stats_pr ON match_stats(pr) WHERE pr IS NOT NULL`,
	// The breakdowns of a seat's counted decisions (sqlshared/match_stats_cells.go
	// states the kinds and the arithmetic): one row per decision type, table
	// of the analysis (met_id, 0 for the built-in one), kind and value(s) of
	// its dimension (-1 for NULL), with the decisions, their error sum and
	// largest error, the blunders at the library's threshold and the sum of
	// their MWC losses over the mwc_decisions the table values; positions
	// counts a phase cell's private positions. Written and
	// dropped with the seat's match_stats row; the cascade reads the index on
	// (match_id, seat). The rows are clustered (WITHOUT ROWID) by kind, then
	// by the dimension's values: the stats read one kind at a time and sum it
	// by those values, so the read takes only that kind's rows and groups
	// them in key order without sorting. A key led by the match would make
	// each kind's read go over every cell of every match, and a rowid table
	// would add a table seek per cell. The DDL text of this table and of
	// match_stats_position is compared at every open (ddlShape): any edit of
	// it, even cosmetic, drops both tables and recomputes every match once.
	`CREATE TABLE IF NOT EXISTS match_stats_cell (
		match_id INTEGER NOT NULL,
		seat INTEGER NOT NULL,
		decision_type INTEGER NOT NULL,
		met_id INTEGER NOT NULL,
		kind INTEGER NOT NULL,
		k1 INTEGER NOT NULL,
		k2 INTEGER NOT NULL,
		decisions INTEGER NOT NULL,
		error_mp INTEGER NOT NULL,
		max_error_mp INTEGER NOT NULL,
		blunders INTEGER NOT NULL,
		mwc_loss REAL NOT NULL,
		mwc_decisions INTEGER NOT NULL,
		positions INTEGER NOT NULL,
		PRIMARY KEY (kind, k1, k2, match_id, seat, decision_type, met_id),
		FOREIGN KEY (match_id, seat) REFERENCES match_stats(match_id, seat) ON DELETE CASCADE
	) WITHOUT ROWID`,
	`CREATE INDEX IF NOT EXISTS idx_match_stats_cell_match ON match_stats_cell(match_id, seat)`,
	// The shared positions a seat's counted decisions reach — those a move of
	// another seat or match reaches too — for the one figure no sum of cells
	// gives: how many distinct positions a selection holds. The fill reads it
	// by position to tell which seats hold a position private.
	`CREATE TABLE IF NOT EXISTS match_stats_position (
		match_id INTEGER NOT NULL,
		seat INTEGER NOT NULL,
		position_id INTEGER NOT NULL,
		decision_type INTEGER NOT NULL,
		met_id INTEGER NOT NULL,
		PRIMARY KEY (match_id, seat, position_id, met_id),
		FOREIGN KEY (match_id, seat) REFERENCES match_stats(match_id, seat) ON DELETE CASCADE
	) WITHOUT ROWID`,
	`CREATE INDEX IF NOT EXISTS idx_match_stats_position_position ON match_stats_position(position_id)`,
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
		-- move_type and cube_action are action codes (domain.ActionCode,
		-- ADR-0071); read through sqlshared.ActionLabelSQL.
		move_type INTEGER,
		position_id INTEGER,
		player INTEGER,
		dice_1 INTEGER,
		dice_2 INTEGER,
		checker_move TEXT,
		cube_action INTEGER,
		luck_mp INTEGER,
		-- The equity this play gave up against the best one, in non-negative
		-- millipoints (domain.Move.ErrorMP), stored so that a statistic over
		-- the plays of one player reads a column instead of decoding every
		-- analysis. NULL means unscored: no analysis, a play absent from the
		-- candidates, or a row older than the column that the resumable pass
		-- (MatchStore.ScoreMoves) has not reached yet.
		error_mp INTEGER,
		-- The error and the close-cube flag the statistics count this
		-- decision by (sqlshared/played_decisions.go): the analysis's
		-- cube_error / best_move_equity_error and is_close_cube, scored
		-- against this move's own play, not the first one the position saw.
		-- A NULL error is unscored (no analysis, or a play no candidate
		-- names). Rewritten with every analysis write of the position.
		decision_error_mp INTEGER,
		is_close_cube INTEGER NOT NULL DEFAULT 0,
		-- How long the player took over the decision, in milliseconds
		-- (domain.Move.DecisionMS / CubeDecisionMS, ADR-0073). NULL is
		-- unknown, never zero: only a Duel measures it.
		decision_ms INTEGER,
		cube_decision_ms INTEGER,
		FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL
	)`,
	// A transcription is a DRAFT (ADR-0045): one opaque JSON document, plus
	// the columns the library list shows. The document carries its OWN
	// format_version, so its shape changes never need a DatabaseVersion
	// migration. match_id is the Match it produced, if any; ON DELETE SET NULL
	// so deleting the match never destroys the typing behind it. revision
	// counts the row's writes, the version a gesture must name (ADR-0057).
	`CREATE TABLE IF NOT EXISTS transcription (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		format_version TEXT NOT NULL,
		match_id INTEGER REFERENCES match(id) ON DELETE SET NULL,
		label TEXT DEFAULT '',
		document TEXT NOT NULL,
		revision INTEGER NOT NULL DEFAULT 1
	)`,
	`CREATE INDEX IF NOT EXISTS idx_transcription_match ON transcription(match_id)`,
	// A Duel is the DRAFT of a match played here (ADR-0072 rule 10): one
	// opaque JSON document rewritten after every Action, like a transcription.
	// dice_seed is its own column, written at creation and never again: every
	// roll is computed from it, and it leaves the Arbiter only with the Match.
	// is_open says the Duel is being played, its clocks running: in the row,
	// so every process on the library sees the same Duels open.
	`CREATE TABLE IF NOT EXISTS duel (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		format_version TEXT NOT NULL,
		label TEXT NOT NULL DEFAULT '',
		document TEXT NOT NULL,
		dice_seed TEXT NOT NULL,
		revision INTEGER NOT NULL DEFAULT 1,
		is_open INTEGER NOT NULL DEFAULT 0
	)`,
	// The origin of a Match played here (ADR-0072 rule 10): its Start (an
	// XGID, '' for the opening position), the revealed dice seed, and whether
	// it was stopped before its end or lost on time, and, when a Bot played,
	// its level and the gammonNet tag whose policy it played (bot_engine); the
	// Bots external Sides declared, as JSON (declared_bots, '' for none); the
	// contributions to a combined seed, as JSON (contributions, '' for none). A
	// Match without a row was not played here. A table of its own so that the
	// match row, which every importer writes, stays as it is.
	`CREATE TABLE IF NOT EXISTS match_origin (
		match_id INTEGER PRIMARY KEY REFERENCES match(id) ON DELETE CASCADE,
		start TEXT NOT NULL DEFAULT '',
		dice_seed TEXT NOT NULL,
		stopped_early INTEGER NOT NULL DEFAULT 0,
		over_time INTEGER NOT NULL DEFAULT 0,
		bot_level TEXT NOT NULL DEFAULT '',
		cadence TEXT NOT NULL DEFAULT '',
		bot_engine TEXT NOT NULL DEFAULT '',
		declared_bots TEXT NOT NULL DEFAULT '',
		contributions TEXT NOT NULL DEFAULT ''
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
	// The trash: a SNAPSHOT of what was deleted, not a deleted_at column, so
	// no live query, statistic or retention predicate has to know about it
	// (ADR-0036).
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
	// The Training journal: two tables rather than a metadata JSON key,
	// because the per-number detail grows without bound (ADR-0040 rule 6).
	`CREATE TABLE IF NOT EXISTS training_session (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		-- scores | pips | bearoff | evaluation | decision
		exercise TEXT NOT NULL,
		-- pool | board | library (ADR-0041 rule 2); empty when the exercise
		-- has only one source.
		seed_source TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		numbers_asked INTEGER NOT NULL DEFAULT 0,
		faults INTEGER NOT NULL DEFAULT 0,
		-- How many of the numbers carried a signed deviation, and the mean of
		-- their absolute values. The count is not redundant with the mean: a
		-- declared exercise produces none at all, and a question that ran out
		-- of time produces none either — averaging its missing answer as a
		-- zero error would flatter the mean.
		deviations INTEGER NOT NULL DEFAULT 0,
		mean_deviation REAL NOT NULL DEFAULT 0,
		median_ms INTEGER NOT NULL DEFAULT 0,
		-- The session PR, for the Decision exercise only; 0 elsewhere, where
		-- no equity error exists to rate.
		pr REAL NOT NULL DEFAULT 0
	)`,
	`CREATE INDEX IF NOT EXISTS idx_training_session_exercise ON training_session(exercise, id)`,
	`CREATE TABLE IF NOT EXISTS training_item (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL,
		-- The kind of number, never the face it was asked from: the same cell
		-- of the same table seen from either side is one weakness, not two
		-- (ADR-0040 rule 4). "tp4.last", "gv2", "pips.bottom", …
		number_type TEXT NOT NULL,
		wrong INTEGER NOT NULL DEFAULT 0,
		has_deviation INTEGER NOT NULL DEFAULT 0,
		deviation REAL NOT NULL DEFAULT 0,
		-- The position a decision question was asked on, the answer given
		-- (the move or cube action as the quiz renders it) and its cost in
		-- millipoints; NULL, '' and NULL for the number exercises. SET NULL,
		-- not a hold: a quiz answer does not keep a position its match no
		-- longer reaches (positionIsHeldSQL is unchanged).
		position_id INTEGER REFERENCES position(id) ON DELETE SET NULL,
		answer TEXT NOT NULL DEFAULT '',
		error_mp INTEGER,
		FOREIGN KEY(session_id) REFERENCES training_session(id) ON DELETE CASCADE
	)`,
	`CREATE INDEX IF NOT EXISTS idx_training_item_session ON training_item(session_id)`,
	`CREATE INDEX IF NOT EXISTS idx_training_item_type ON training_item(number_type)`,
	`CREATE INDEX IF NOT EXISTS idx_training_item_position ON training_item(position_id) WHERE position_id IS NOT NULL`,
	// A Rencontre is the room several directed Tournaments share (ADR-0056):
	// its tables, and the output folder of its wall page. What happens in the
	// room — a table out of service, a break — is not stored here: it is an
	// event in the log of every member Direction.
	`CREATE TABLE IF NOT EXISTS rencontre (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		starts_on TEXT DEFAULT '',
		ends_on TEXT DEFAULT '',
		tables INTEGER NOT NULL DEFAULT 0,
		output_dir TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS tournament (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		date TEXT,
		location TEXT,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		comment TEXT DEFAULT '',
		-- The Rencontre this Tournament plays in, if any. Deleting the
		-- Rencontre detaches its Tournaments; it never deletes one.
		rencontre_id INTEGER REFERENCES rencontre(id) ON DELETE SET NULL,
		-- The rooms this Tournament may play in within its Rencontre, as a
		-- JSON array of room labels; NULL plays on every table. A fact of
		-- the membership, cleared with it on detachment (ADR-0058 §5).
		rencontre_rooms TEXT
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tournament_rencontre ON tournament(rencontre_id)`,
	// The properties of a table (ADR-0058): one row per table that has any,
	// owned by the Rencontre whose events share it or by a Tournament run on
	// its own, never both. The number is the table's identity; a room is only
	// the label several rows share. assigned_to is a JSON array of names.
	`CREATE TABLE IF NOT EXISTS table_setting (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		rencontre_id INTEGER REFERENCES rencontre(id) ON DELETE CASCADE,
		tournament_id INTEGER REFERENCES tournament(id) ON DELETE CASCADE,
		number INTEGER NOT NULL CHECK (number > 0),
		name TEXT NOT NULL DEFAULT '',
		room TEXT NOT NULL DEFAULT '',
		reserved INTEGER NOT NULL DEFAULT 0,
		assigned_to TEXT NOT NULL DEFAULT '[]',
		CHECK ((rencontre_id IS NULL) <> (tournament_id IS NULL))
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_table_setting_rencontre ON table_setting(rencontre_id, number) WHERE rencontre_id IS NOT NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_table_setting_tournament ON table_setting(tournament_id, number) WHERE tournament_id IS NOT NULL`,
	// Other spellings of one player or one event: alias → the
	// canonical name the import, the stats and the search read instead.
	// Names, not ids: a match stores its players and its event as text.
	`CREATE TABLE IF NOT EXISTS player_alias (
		alias      TEXT PRIMARY KEY,
		canonical  TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_player_alias_canonical ON player_alias(canonical)`,
	`CREATE TABLE IF NOT EXISTS event_alias (
		alias      TEXT PRIMARY KEY,
		canonical  TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_event_alias_canonical ON event_alias(canonical)`,
	// A Lesson (ADR-0066): an ordered sequence of Steps, each a text that may
	// show a Collection, a Position, both or neither. A deleted Collection or
	// Position leaves the Step and its text; deleting the Lesson takes its
	// Steps. A Position a Step shows is held (positionIsHeldSQL).
	`CREATE TABLE IF NOT EXISTS lesson (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS lesson_step (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lesson_id INTEGER NOT NULL REFERENCES lesson(id) ON DELETE CASCADE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		title TEXT NOT NULL DEFAULT '',
		text TEXT NOT NULL DEFAULT '',
		collection_id INTEGER REFERENCES collection(id) ON DELETE SET NULL,
		position_id INTEGER REFERENCES position(id) ON DELETE SET NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_lesson_step_lesson ON lesson_step(lesson_id, sort_order)`,
	`CREATE INDEX IF NOT EXISTS idx_lesson_step_position ON lesson_step(position_id)`,
	// The student's progress through a Lesson (ADR-0069): one row per Step
	// marked done, written only by the explicit "step done" gesture, never by
	// reading, opening or importing. A table of its own so that no exporter,
	// which copies lesson and lesson_step, ever carries it.
	`CREATE TABLE IF NOT EXISTS lesson_progress (
		lesson_step_id INTEGER PRIMARY KEY REFERENCES lesson_step(id) ON DELETE CASCADE,
		done_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	// The positions the user marked "studied" in the library-wide study
	// backlog: written only by that gesture, reversible, never exported (a
	// table of its own so that no exporter carries it). Not a retention
	// reason: marking a position studied keeps nothing alive.
	`CREATE TABLE IF NOT EXISTS study_mark (
		position_id INTEGER PRIMARY KEY REFERENCES position(id) ON DELETE CASCADE,
		marked_at INTEGER NOT NULL
	)`,
	// The action labels the fixed list of domain.ActionCode lacks, each under
	// a code from domain.FirstRegisteredActionCode up (ADR-0071).
	`CREATE TABLE IF NOT EXISTS action_label (
		code  INTEGER PRIMARY KEY,
		label TEXT NOT NULL UNIQUE
	)`,
	// The match equity tables of the library (ADR-0068): each imported from
	// a gnubg .xml file, identified by the digest of its values. At most one
	// is current; none current means the built-in Kazaross-XG2.
	`CREATE TABLE IF NOT EXISTS match_equity_table (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		digest TEXT NOT NULL UNIQUE,
		source TEXT NOT NULL,
		is_current INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	// The Direction of a Tournament (ADR-0047): everything the tournament
	// director decided while running it. One row per directed Tournament, plus
	// one row per event in direction_event.
	//
	// `config` holds the configuration only while the Direction is in
	// preparation; the first launched match freezes it into the `created` event
	// and the column stops being read. The derived state — standings, brackets,
	// pairings, what to do next — is NEVER stored: it is replayed from the
	// events at every open, which is what makes a power cut cost nothing.
	`CREATE TABLE IF NOT EXISTS direction (
		tournament_id INTEGER PRIMARY KEY REFERENCES tournament(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		format_version INTEGER NOT NULL DEFAULT 1,
		engine_version TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL DEFAULT 'draft',
		config TEXT NOT NULL DEFAULT '',
		output_dir TEXT DEFAULT ''
	)`,
	// The events themselves: APPEND-ONLY. A wrong result is corrected by a later
	// correction event, a match launched by mistake by a later cancellation —
	// never by rewriting a row (ADR-0047). Nothing here is ever UPDATEd or
	// DELETEd outside the cascade from a deleted Tournament.
	`CREATE TABLE IF NOT EXISTS direction_event (
		tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
		seq INTEGER NOT NULL,
		kind TEXT NOT NULL,
		time DATETIME NOT NULL,
		payload TEXT NOT NULL,
		PRIMARY KEY (tournament_id, seq)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_direction_event_tournament ON direction_event(tournament_id, seq)`,
	// The two people behind a doubles Participant (ADR-0056 §4). The engine
	// knows one Participant named "A / B"; the Directory, the availability
	// check and the entry form need the two persons, each with a club and a
	// rating. seat is 0 or 1, in the order the pair was entered.
	`CREATE TABLE IF NOT EXISTS direction_pair_member (
		tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
		player_id TEXT NOT NULL,
		seat INTEGER NOT NULL,
		name TEXT NOT NULL,
		club TEXT NOT NULL DEFAULT '',
		rating REAL NOT NULL DEFAULT 0,
		PRIMARY KEY (tournament_id, player_id, seat)
	)`,
	// A Slot carries at most one Match: the pair (tournament, slot) is unique
	// wherever a slot is actually named. The partial index leaves the empty
	// string free, which is what every Match that fills no Slot carries.
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_match_direction_slot ON match(tournament_id, direction_match_id) WHERE direction_match_id <> ''`,
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
	// A card asks about a position OR a score (ADR-0042): `kind` says which,
	// `key` names it (the position id as text, or the unordered score "3:5").
	// position_id is NULLABLE, NULL for a score card rather than a dangling 0.
	// Older databases are rebuilt by the 2.23.0 migration: EnsureSchema cannot
	// relax a constraint.
	`CREATE TABLE IF NOT EXISTS anki_card (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		deck_id INTEGER NOT NULL,
		kind TEXT NOT NULL DEFAULT 'position',
		key TEXT NOT NULL DEFAULT '',
		position_id INTEGER,
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
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
	)`,
	// kind/key and a nullable position_id, as anki_card: a review can be of
	// a score.
	`CREATE TABLE IF NOT EXISTS anki_review_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_id INTEGER NOT NULL,
		deck_id INTEGER NOT NULL,
		kind TEXT NOT NULL DEFAULT 'position',
		key TEXT NOT NULL DEFAULT '',
		position_id INTEGER,
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
	// A card's identity within its deck, stated as an INDEX and not as a table
	// constraint on purpose: EnsureSchema builds indexes on an existing
	// database and cannot add a constraint, and this is the index the sync's
	// "ON CONFLICT (deck_id, kind, key)" needs in order to be a no-op rather
	// than a duplicate.
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_anki_card_identity ON anki_card(deck_id, kind, key)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_card_due ON anki_card(deck_id, due)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_review_log_card ON anki_review_log(card_id, reviewed_at)`,
	`CREATE INDEX IF NOT EXISTS idx_anki_review_log_deck ON anki_review_log(deck_id, reviewed_at)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_position_zobrist        ON position(zobrist_hash)`,
	// No index leads with decision_type: it splits the library in
	// two, and a filter on it ran slower through such an index than through
	// the table on a 15.6 M-position library. Only the take/pass side of
	// is_cube_response is selective.
	`CREATE        INDEX IF NOT EXISTS idx_position_cube_take      ON position(is_cube_response) WHERE is_cube_response = 1`,
	`CREATE        INDEX IF NOT EXISTS idx_position_individual     ON position(individually_imported) WHERE individually_imported = 1`,
	`CREATE        INDEX IF NOT EXISTS idx_position_flagged        ON position(flagged) WHERE flagged = 1`,
	`CREATE        INDEX IF NOT EXISTS idx_position_pip_diff       ON position(pip_diff)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_dice           ON position(dice_1, dice_2)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_off            ON position(off_1, off_2)`,
	// No idx_position_score: it was a strict prefix of this one;
	// ensureAllTablesExist (db_schema.go) drops it from existing databases.
	`CREATE        INDEX IF NOT EXISTS idx_position_score_cube     ON position(match_length, score_1, score_2, cube_value)`,
	// One analysis per position, enforced: the upsert in analyses_sqlite.go
	// needs a UNIQUE target for ON CONFLICT, and concurrent saves must not
	// insert two rows. The 2.18.0 migration deduplicates older databases.
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_analysis_position       ON analysis(position_id)`,
	// Covering index for the win/gammon search's `p.id IN (SELECT position_id
	// FROM analysis WHERE …)`: with position_id third, the subquery never
	// touches the table. A new name because EnsureSchema builds indexes by
	// name; ensureAllTablesExist (db_schema.go) drops the superseded
	// idx_analysis_win_gammon and idx_analysis_win1 (a strict prefix).
	`CREATE        INDEX IF NOT EXISTS idx_analysis_win_gammon_covering ON analysis(player1_win_rate, player1_gammon_rate, position_id)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_cube_error     ON analysis(cube_error)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_move_error     ON analysis(best_move_equity_error)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_is_forced      ON analysis(is_forced) WHERE is_forced = 1`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_is_close_cube  ON analysis(is_close_cube) WHERE is_close_cube = 1`,
	// The comment-presence filter (`co`/`xco`) probes this table once per search
	// with an EXISTS subquery; without the index that is a full comment scan.
	`CREATE        INDEX IF NOT EXISTS idx_comment_position        ON comment(position_id, origin)`,
	// Range filters that would otherwise scan the table.
	`CREATE        INDEX IF NOT EXISTS idx_position_back_checkers_1 ON position(back_checkers_1)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_back_checkers_2 ON position(back_checkers_2)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_pip_1          ON position(pip_1)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_no_contact     ON position(no_contact) WHERE no_contact = 1`,
	// game_phase leads, so a phase filter alone still reads it; off_1 follows
	// because the race filter is the one combined with a bear-off count, and
	// without it the planner sorted every off_1 candidate for a first page.
	`CREATE        INDEX IF NOT EXISTS idx_position_phase_off      ON position(game_phase, off_1)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_game_type      ON position(game_type)`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_backgammon1    ON analysis(player1_backgammon_rate)`,
	// The player-2 twin of idx_analysis_win_gammon_covering, for the same
	// IN-subquery (sqlshared/search.go).
	`CREATE        INDEX IF NOT EXISTS idx_analysis_win_gammon2_covering ON analysis(player2_win_rate, player2_gammon_rate, position_id)`,
	// No index on analysis_engine or analysis_depth: their filters run inside
	// a correlated EXISTS on position_id (search) or on a join reached by
	// move (stats), and a library rarely holds more than one engine. Only the
	// provenance backfill looks rows up by a NULL engine, and the rows it is
	// looking for are the only ones this partial index holds — it is empty
	// once the pass is over.
	`CREATE        INDEX IF NOT EXISTS idx_analysis_provenance_pending ON analysis(id) WHERE analysis_engine IS NULL`,
	// The few analyses computed with an imported table: the stats read them to
	// leave out what another table valued (ADR-0068).
	`CREATE        INDEX IF NOT EXISTS idx_analysis_met            ON analysis(met_id) WHERE met_id IS NOT NULL`,
	`CREATE        INDEX IF NOT EXISTS idx_analysis_creation_date  ON analysis(creation_date)`,
	`CREATE        INDEX IF NOT EXISTS idx_position_match_date     ON position(match_date)`,
	`CREATE        INDEX IF NOT EXISTS idx_import_batch_file_batch ON import_batch_file(batch_id)`,
	`CREATE        INDEX IF NOT EXISTS idx_import_batch_file_path  ON import_batch_file(path, size, mtime)`,
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

// EnsureSchema adds to an existing database the tables, columns and indexes
// it lacks; nothing is renamed or retyped, and nothing is dropped but the
// derived breakdown tables of match_stats when their layout differs from the
// schema's, which are recreated and recomputed (reclusterDerivedTables).
// Idempotent; the Database wrapper runs it on every open, after the
// migration chain.
//
// What is missing is found against a reference database built in memory
// from schemaStatements, so there is no second column list to keep in step.
//
// Tables are created strictly. Columns and indexes are best-effort and
// logged (a non-constant default, a UNIQUE index over violating rows): such a
// database must still open, and CheckSchema reports what was left out.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	ref, err := referenceSchema(ctx)
	if err != nil {
		return err
	}
	if err := reclusterDerivedTables(ctx, db, ref); err != nil {
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
	have, err := indexNames(ctx, db)
	if err != nil {
		return err
	}
	for _, stmt := range ref.indexes {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			slog.Warn("sqlite: cannot ensure index", "stmt", stmt, "err", err)
			continue
		}
		// An index missing from an existing database is the trace of a bulk
		// import cut before it rebuilt what it had dropped: say so.
		if m := indexStmt.FindStringSubmatch(stmt); m != nil && len(have) > 0 && !have[m[1]] {
			slog.Warn("sqlite: rebuilt missing index", "index", m[1])
		}
	}
	return nil
}

// derivedClusteredTables are match_stats's breakdowns. Their layout is
// part of how fast the stats read them, and they hold nothing match_stats
// cannot recompute.
var derivedClusteredTables = []string{"match_stats_cell", "match_stats_position"}

// reclusterDerivedTables drops the breakdown tables when one was created
// with another layout than the schema's (a rowid, another key order), and
// match_stats with them, since a seat row without its cells would be read as
// complete. The table creation that follows recreates them, and the next
// fill recomputes every match. A table cannot change its key in place; the
// data being derived, recomputing it is the simplest rebuild, and the check
// costs two catalogue reads once the layout is right.
func reclusterDerivedTables(ctx context.Context, db *sql.DB, ref *reference) error {
	// A library a newer build wrote is opened as it is: its layout may be
	// that build's, and rebuilding it would make each build recompute the
	// other's tables at every alternation.
	var version string
	if err := db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&version); err == nil &&
		newerThanBuild(version) {
		return nil
	}
	stale := false
	for _, name := range derivedClusteredTables {
		var ddl string
		err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&ddl)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("sqlite: read %s layout: %w", name, err)
		}
		for _, t := range ref.tables {
			if t.name == name && ddlShape(ddl) != ddlShape(t.createSQL) {
				stale = true
			}
		}
	}
	if !stale {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: recluster match stats: %w", err)
	}
	defer tx.Rollback() // a no-op after Commit
	stmts := []string{}
	for _, name := range derivedClusteredTables {
		stmts = append(stmts, `DROP TABLE IF EXISTS `+name)
	}
	stmts = append(stmts, `DELETE FROM match_stats`)
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sqlite: recluster match stats (%s): %w", stmt, err)
		}
	}
	return tx.Commit()
}

// newerThanBuild reports whether the schema version v is past
// domain.DatabaseVersion; a version that does not parse is not.
func newerThanBuild(v string) bool {
	var a, b [3]int
	if _, err := fmt.Sscanf(v, "%d.%d.%d", &a[0], &a[1], &a[2]); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(domain.DatabaseVersion, "%d.%d.%d", &b[0], &b[1], &b[2]); err != nil {
		return false
	}
	return slices.Compare(a[:], b[:]) > 0
}

// ddlShape is a CREATE TABLE statement up to case, spacing and IF NOT
// EXISTS, which SQLite drops from the text it keeps.
func ddlShape(ddl string) string {
	s := strings.ToUpper(strings.Join(strings.Fields(ddl), ""))
	return strings.Replace(s, "IFNOTEXISTS", "", 1)
}

// indexNames lists the named indexes db holds.
func indexNames(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'index'`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list indexes: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

// CreateTableSQL returns the fresh-database DDL of one table, as
// schemaStatements states it. It exists for the rare migration that has to
// REBUILD a table rather than add to it — SQLite relaxes no constraint through
// ALTER TABLE — so that the rebuilt table is built from the one schema and not
// from a copy of it going stale in a migration file.
func CreateTableSQL(name string) (string, error) {
	prefix := "CREATE TABLE IF NOT EXISTS " + name + " ("
	for _, stmt := range schemaStatements {
		if strings.HasPrefix(stmt, prefix) {
			return stmt, nil
		}
	}
	return "", fmt.Errorf("sqlite: no table %q in the schema", name)
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
