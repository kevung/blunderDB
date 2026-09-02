package database

// Migration steps 1.0.0 → 2.0.0. See db_migration.go for the registry that
// runs them and the rules a step follows (no version stamp, re-runnable).
//
// The 1.0.0 → 1.6.0 steps each create tables, and only run when the table
// they create is absent: a database that already holds it returns
// errStepNotApplicable and the chain stops there, as it always has. The
// tables are also (re)created by ensureAllTablesExist after the chain.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// migrate_1_0_0_to_1_1_0 adds the command_history table.
func (d *Database) migrate_1_0_0_to_1_1_0(_ context.Context) error {
	if !d.tableAbsent("command_history") {
		return errStepNotApplicable
	}
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// migrate_1_1_0_to_1_2_0 adds the filter_library table.
func (d *Database) migrate_1_1_0_to_1_2_0(_ context.Context) error {
	if !d.tableAbsent("filter_library") {
		return errStepNotApplicable
	}
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS filter_library (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			command TEXT,
			edit_position TEXT
		)
	`)
	return err
}

// migrate_1_2_0_to_1_3_0 adds the search_history table.
func (d *Database) migrate_1_2_0_to_1_3_0(_ context.Context) error {
	if !d.tableAbsent("search_history") {
		return errStepNotApplicable
	}
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			position TEXT,
			timestamp INTEGER
		)
	`)
	return err
}

// migrate_1_3_0_to_1_4_0 adds the match, game, move and move_analysis tables
// and the match_hash index used for duplicate detection.
func (d *Database) migrate_1_3_0_to_1_4_0(_ context.Context) error {
	if !d.tableAbsent("match") {
		return errStepNotApplicable
	}
	for _, stmt := range []string{
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
			equity REAL,
			equity_error REAL,
			win_rate REAL,
			gammon_rate REAL,
			backgammon_rate REAL,
			opponent_win_rate REAL,
			opponent_gammon_rate REAL,
			opponent_backgammon_rate REAL,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
		)`,
		// Index on match_hash for fast duplicate detection
		`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// migrate_1_4_0_to_1_5_0 first repairs a 1.4.0 database whose match table
// predates the match_hash column (adding and populating it), then adds the
// collection and collection_position tables.
func (d *Database) migrate_1_4_0_to_1_5_0(_ context.Context) error {
	if err := d.addMatchHashColumn(); err != nil {
		return err
	}

	if !d.tableAbsent("collection") {
		return errStepNotApplicable
	}
	for _, stmt := range []string{
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
		// Index for faster collection lookups
		`CREATE INDEX IF NOT EXISTS idx_collection_position_collection ON collection_position(collection_id)`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// addMatchHashColumn adds match.match_hash (and its index) to a 1.4.0 database
// created before the column existed, and fills it for the stored matches
// from a fallback hash of their stored data — the original files are gone.
// A match table that already names the column is left alone.
func (d *Database) addMatchHashColumn() error {
	var colInfo string
	err := d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&colInfo)
	if err != nil || strings.Contains(colInfo, "match_hash") {
		return nil
	}

	if _, err := d.db.Exec(`ALTER TABLE match ADD COLUMN match_hash TEXT`); err != nil {
		return err
	}
	if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`); err != nil {
		return err
	}

	// Populate match_hash for existing matches. A failing SELECT leaves the
	// column NULL, as it always has; only an iteration error is fatal.
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
	return nil
}

// migrate_1_5_0_to_1_6_0 adds the tournament table and match.tournament_id.
func (d *Database) migrate_1_5_0_to_1_6_0(_ context.Context) error {
	if !d.tableAbsent("tournament") {
		return errStepNotApplicable
	}
	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS tournament (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			date TEXT,
			location TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}
	// Add tournament_id column to match table
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)
	return nil
}

// migrate_1_6_0_to_1_7_0 adds match.last_visited_position.
func (d *Database) migrate_1_6_0_to_1_7_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)
	return nil
}

// migrate_1_7_0_to_1_8_0 adds comment.created_at and backfills it.
func (d *Database) migrate_1_7_0_to_1_8_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE comment ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP`)
	// Backfill existing rows that have NULL created_at
	_, _ = d.db.Exec(`UPDATE comment SET created_at = CURRENT_TIMESTAMP WHERE created_at IS NULL`)
	return nil
}

// migrate_1_8_0_to_1_9_0 adds comment.modified_at.
func (d *Database) migrate_1_8_0_to_1_9_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE comment ADD COLUMN modified_at DATETIME`)
	return nil
}

// migrate_1_9_0_to_2_0_0 performs the in-place backfill migration from the
// pre-2.0.0 schema (position.state only) to the v2.0.0 schema with scalar
// columns.  It runs inside a single transaction:
//
//  1. ALTER TABLE to add all new nullable columns (idempotent — duplicate-column
//     errors are silently ignored).
//  2. Backfill position rows in batches of 1000.
//  3. Backfill analysis rows in batches of 1000.
//  4. Deduplicate positions (same Zobrist hash) by re-pointing FK references
//     to the lowest id, then deleting orphans.
//  5. CREATE INDEX IF NOT EXISTS for every v2.0.0 index.
//  6. ANALYZE to refresh query-planner statistics.
//
// The chain stamps "2.0.0" once the step returns. If the process is
// interrupted before that, the version string remains "1.9.0" and the
// migration is retried on the next open; the ALTER TABLE statements are
// idempotent so repeated runs are safe.
//
// The caller must hold d.mu.
func (d *Database) migrate_1_9_0_to_2_0_0(ctx context.Context) error {
	// 1. ALTER TABLE — add new nullable columns (swallow "duplicate column")
	// -----------------------------------------------------------------
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
	}
	for _, stmt := range newPositionCols {
		_, _ = d.db.Exec(stmt) // duplicate column → silently ignored
	}

	newAnalysisCols := []string{
		`ALTER TABLE analysis ADD COLUMN best_cube_action        TEXT`,
		`ALTER TABLE analysis ADD COLUMN cube_error              REAL`,
		`ALTER TABLE analysis ADD COLUMN best_move_equity_error  REAL`,
		`ALTER TABLE analysis ADD COLUMN player1_win_rate        REAL`,
		`ALTER TABLE analysis ADD COLUMN player1_gammon_rate     REAL`,
		`ALTER TABLE analysis ADD COLUMN player1_backgammon_rate REAL`,
		`ALTER TABLE analysis ADD COLUMN player2_win_rate        REAL`,
		`ALTER TABLE analysis ADD COLUMN player2_gammon_rate     REAL`,
		`ALTER TABLE analysis ADD COLUMN player2_backgammon_rate REAL`,
	}
	for _, stmt := range newAnalysisCols {
		_, _ = d.db.Exec(stmt)
	}

	// -----------------------------------------------------------------
	// 2. Backfill position
	// -----------------------------------------------------------------
	var posTotal int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posTotal)

	if posTotal > 0 {
		updatePos, err := d.db.Prepare(`UPDATE position SET
			zobrist_hash=?, decision_type=?, player_on_roll=?, dice_1=?, dice_2=?,
			cube_value=?, cube_owner=?, score_1=?, score_2=?,
			has_jacoby=?, has_beaver=?,
			pip_1=?, pip_2=?, pip_diff=?, off_1=?, off_2=?,
			back_checkers_1=?, back_checkers_2=?, no_contact=?,
			occupancy_1=?, occupancy_2=?, point_mask_1=?, point_mask_2=?
			WHERE id=?`)
		if err != nil {
			return fmt.Errorf("migrate position prepare: %w", err)
		}
		defer updatePos.Close()

		const batchSize = 1000
		var lastID int64 = 0
		done := 0

		for {
			if err := ctx.Err(); err != nil {
				return err
			}

			var batch []struct {
				id    int64
				state string
			}
			err := d.forEachRow(
				`SELECT id, state FROM position WHERE id > ? ORDER BY id LIMIT ?`,
				[]any{lastID, batchSize}, func(rows *sql.Rows) {
					var id int64
					var state string
					if err := rows.Scan(&id, &state); err == nil {
						batch = append(batch, struct {
							id    int64
							state string
						}{id, state})
					}
				})
			if err != nil {
				return fmt.Errorf("migrate position rows: %w", err)
			}

			if len(batch) == 0 {
				break
			}

			for _, row := range batch {
				lastID = row.id
				var pos Position
				if err := json.Unmarshal([]byte(row.state), &pos); err != nil {
					continue // malformed JSON — leave columns NULL
				}
				c := populatePositionColumns(&pos)
				noContactInt := 0
				if c.NoContact {
					noContactInt = 1
				}
				_, _ = updatePos.Exec(
					int64(c.ZobristHash), c.DecisionType, 0 /*player_on_roll: always 0 after normalize*/, c.Dice1, c.Dice2,
					c.CubeValue, c.CubeOwner, c.Score1, c.Score2,
					c.HasJacoby, c.HasBeaver,
					c.Pip1, c.Pip2, c.PipDiff, c.Off1, c.Off2,
					c.BackCheckers1, c.BackCheckers2, noContactInt,
					int64(c.Occupancy1), int64(c.Occupancy2), int64(c.PointMask1), int64(c.PointMask2),
					row.id)
				done++
				if done%200 == 0 {
					d.emitMigrationProgress("position", done, posTotal)
				}
			}
		}
		d.emitMigrationProgress("position", posTotal, posTotal)
	}

	// -----------------------------------------------------------------
	// 3. Backfill analysis
	// -----------------------------------------------------------------
	var anaTotal int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM analysis`).Scan(&anaTotal)

	if anaTotal > 0 {
		updateAna, err := d.db.Prepare(`UPDATE analysis SET
			best_cube_action=?, cube_error=?, best_move_equity_error=?,
			player1_win_rate=?, player1_gammon_rate=?, player1_backgammon_rate=?,
			player2_win_rate=?, player2_gammon_rate=?, player2_backgammon_rate=?
			WHERE id=?`)
		if err != nil {
			return fmt.Errorf("migrate analysis prepare: %w", err)
		}
		defer updateAna.Close()

		// Prepare statement to look up the played cube action from the move table
		// when the analysis JSON doesn't have it.
		lookupCubeAction, err := d.db.Prepare(`
			SELECT m.cube_action FROM move m
			WHERE m.position_id = (SELECT a2.position_id FROM analysis a2 WHERE a2.id = ?)
			  AND m.move_type = 'cube' AND m.cube_action != ''
			LIMIT 1`)
		if err != nil {
			return fmt.Errorf("migrate cube action lookup prepare: %w", err)
		}
		defer lookupCubeAction.Close()

		const batchSize = 1000
		var lastID int64 = 0
		done := 0

		for {
			if err := ctx.Err(); err != nil {
				return err
			}

			var batch []struct {
				id   int64
				data []byte
			}
			err := d.forEachRow(
				`SELECT id, data FROM analysis WHERE id > ? ORDER BY id LIMIT ?`,
				[]any{lastID, batchSize}, func(rows *sql.Rows) {
					var id int64
					var data []byte
					if err := rows.Scan(&id, &data); err == nil {
						batch = append(batch, struct {
							id   int64
							data []byte
						}{id, data})
					}
				})
			if err != nil {
				return fmt.Errorf("migrate analysis rows: %w", err)
			}

			if len(batch) == 0 {
				break
			}

			for _, row := range batch {
				lastID = row.id
				ana, err := decodeAnalysisFromStorage(row.data)
				if err != nil {
					continue
				}
				playedMove := ""
				if len(ana.PlayedMoves) > 0 {
					playedMove = ana.PlayedMoves[0]
				} else if ana.PlayedMove != "" {
					playedMove = ana.PlayedMove
				}
				playedCubeAction := ""
				if len(ana.PlayedCubeActions) > 0 {
					playedCubeAction = ana.PlayedCubeActions[0]
				} else if ana.PlayedCubeAction != "" {
					playedCubeAction = ana.PlayedCubeAction
				}
				// If the analysis JSON doesn't have the played cube action but has
				// cube analysis, look it up from the move table.
				if playedCubeAction == "" && ana.DoublingCubeAnalysis != nil {
					var ca sql.NullString
					if err := lookupCubeAction.QueryRow(row.id).Scan(&ca); err == nil && ca.Valid {
						playedCubeAction = ca.String
					}
				}
				ac := populateAnalysisColumns(&ana, playedMove, playedCubeAction)
				_, _ = updateAna.Exec(
					ac.BestCubeAction, ac.CubeError, ac.BestMoveEquityError,
					ac.Player1WinRate, ac.Player1GammonRate, ac.Player1BackgammonRate,
					ac.Player2WinRate, ac.Player2GammonRate, ac.Player2BackgammonRate,
					row.id)
				done++
				if done%200 == 0 {
					d.emitMigrationProgress("analysis", done, anaTotal)
				}
			}
		}
		d.emitMigrationProgress("analysis", anaTotal, anaTotal)
	}

	// -----------------------------------------------------------------
	// 4. Dedup positions with the same Zobrist hash
	//    (should be rare — keep lowest id, remap FK references)
	// -----------------------------------------------------------------
	type dedupGroup struct {
		hash   int64
		keepID int64
		allIDs []int64
	}
	var dups []dedupGroup
	err := d.forEachRow(`
		SELECT zobrist_hash, MIN(id) AS keep_id, GROUP_CONCAT(id ORDER BY id) AS all_ids
		FROM position
		WHERE zobrist_hash IS NOT NULL
		GROUP BY zobrist_hash
		HAVING COUNT(*) > 1`, nil, func(dupRows *sql.Rows) {
		var hash, keepID int64
		var allIDsStr string
		if err := dupRows.Scan(&hash, &keepID, &allIDsStr); err != nil {
			return
		}
		var allIDs []int64
		for _, part := range strings.Split(allIDsStr, ",") {
			var id int64
			if _, err := fmt.Sscan(strings.TrimSpace(part), &id); err == nil {
				allIDs = append(allIDs, id)
			}
		}
		dups = append(dups, dedupGroup{hash, keepID, allIDs})
	})
	if err != nil {
		return fmt.Errorf("migrate dedup rows: %w", err)
	}

	mergedTotal := 0
	for _, g := range dups {
		for _, discardID := range g.allIDs {
			if discardID == g.keepID {
				continue
			}
			// Remap FK references
			_, _ = d.db.Exec(`UPDATE move               SET position_id=? WHERE position_id=?`, g.keepID, discardID)
			_, _ = d.db.Exec(`UPDATE collection_position SET position_id=? WHERE position_id=?`, g.keepID, discardID)
			_, _ = d.db.Exec(`UPDATE anki_card           SET position_id=? WHERE position_id=?`, g.keepID, discardID)
			// Delete orphan analysis + position
			_, _ = d.db.Exec(`DELETE FROM analysis WHERE position_id=?`, discardID)
			_, _ = d.db.Exec(`DELETE FROM position WHERE id=?`, discardID)
			mergedTotal++
		}
	}
	if mergedTotal > 0 {
		slog.Info("migration merged duplicate positions", "count", mergedTotal)
	}

	// -----------------------------------------------------------------
	// 5. Create indexes
	// -----------------------------------------------------------------
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
		`CREATE        INDEX IF NOT EXISTS idx_move_position           ON move(position_id)`,
		`CREATE        INDEX IF NOT EXISTS idx_move_game               ON move(game_id)`,
		`CREATE        INDEX IF NOT EXISTS idx_game_match              ON game(match_id)`,
	}
	for _, idx := range v2indexes {
		if _, err := d.db.Exec(idx); err != nil {
			// UNIQUE index may fail if dedup left residual NULLs; treat as non-fatal warning
			slog.Warn("migration index warning", "err", err)
		}
	}

	// -----------------------------------------------------------------
	// 6. ANALYZE
	// -----------------------------------------------------------------
	_, _ = d.db.Exec(`ANALYZE`)

	return nil
}
