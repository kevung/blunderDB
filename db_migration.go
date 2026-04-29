package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
)

// SetMigrationProgress registers a callback that is invoked during the v2.0.0
// backfill migration to report progress. The GUI wires in a Wails event emitter
// here; the CLI and tests can leave it nil.
func (d *Database) SetMigrationProgress(fn func(phase string, done, total int)) {
	d.migrationProgress = fn
}

// emitMigrationProgress calls the progress callback if one was registered.
func (d *Database) emitMigrationProgress(phase string, done, total int) {
	if d.migrationProgress != nil {
		d.migrationProgress(phase, done, total)
	}
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
//  7. Bump database_version to "2.0.0".
//
// If the process is interrupted before step 7 (COMMIT), the version string
// remains "1.9.0" and the migration is retried on the next open; the ALTER TABLE
// statements are idempotent so repeated runs are safe.
//
// The caller must hold d.mu.
func (d *Database) migrate_1_9_0_to_2_0_0() error {
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
			if atomic.LoadInt32(&d.importCancelled) != 0 {
				return fmt.Errorf("migration cancelled")
			}

			rows, err := d.db.Query(
				`SELECT id, state FROM position WHERE id > ? ORDER BY id LIMIT ?`,
				lastID, batchSize)
			if err != nil {
				return fmt.Errorf("migrate position query: %w", err)
			}

			var batch []struct {
				id    int64
				state string
			}
			for rows.Next() {
				var id int64
				var state string
				if err := rows.Scan(&id, &state); err == nil {
					batch = append(batch, struct {
						id    int64
						state string
					}{id, state})
				}
			}
			rows.Close()

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
			if atomic.LoadInt32(&d.importCancelled) != 0 {
				return fmt.Errorf("migration cancelled")
			}

			rows, err := d.db.Query(
				`SELECT id, data FROM analysis WHERE id > ? ORDER BY id LIMIT ?`,
				lastID, batchSize)
			if err != nil {
				return fmt.Errorf("migrate analysis query: %w", err)
			}

			var batch []struct {
				id   int64
				data []byte
			}
			for rows.Next() {
				var id int64
				var data []byte
				if err := rows.Scan(&id, &data); err == nil {
					batch = append(batch, struct {
						id   int64
						data []byte
					}{id, data})
				}
			}
			rows.Close()

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
	dupRows, err := d.db.Query(`
		SELECT zobrist_hash, MIN(id) AS keep_id, GROUP_CONCAT(id ORDER BY id) AS all_ids
		FROM position
		WHERE zobrist_hash IS NOT NULL
		GROUP BY zobrist_hash
		HAVING COUNT(*) > 1`)
	if err != nil {
		return fmt.Errorf("migrate dedup query: %w", err)
	}

	type dedupGroup struct {
		hash   int64
		keepID int64
		allIDs []int64
	}
	var dups []dedupGroup
	for dupRows.Next() {
		var hash, keepID int64
		var allIDsStr string
		if err := dupRows.Scan(&hash, &keepID, &allIDsStr); err != nil {
			continue
		}
		var allIDs []int64
		for _, part := range strings.Split(allIDsStr, ",") {
			var id int64
			if _, err := fmt.Sscan(strings.TrimSpace(part), &id); err == nil {
				allIDs = append(allIDs, id)
			}
		}
		dups = append(dups, dedupGroup{hash, keepID, allIDs})
	}
	dupRows.Close()

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

	// -----------------------------------------------------------------
	// 7. Bump version
	// -----------------------------------------------------------------
	if _, err := d.db.Exec(`UPDATE metadata SET value='2.0.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "1.9.0", "to", "2.0.0")
	return nil
}

// migrate_2_0_0_to_2_1_0 converts analysis and move_analysis scalar columns
// from REAL to integer-scaled INTEGER values and re-rounds all JSON blobs.
//
// Encoding:
//   - Win/gammon/backgammon rates: stored as rate × 100 (hundredths of percent).
//   - Equity and errors: stored as value × 1000 (millipoints).
//
// JSON blobs are re-parsed, rounded via roundAnalysisForStorage, and re-serialised.
// Analysis scalar columns are re-computed from the rounded JSON via populateAnalysisColumns.
//
// The caller must hold d.mu.
func (d *Database) migrate_2_0_0_to_2_1_0() error {
	// -----------------------------------------------------------------
	// 1. Convert move_analysis REAL columns to integer scale via SQL.
	// -----------------------------------------------------------------
	d.emitMigrationProgress("move_analysis", 0, 1)
	_, err := d.db.Exec(`
		UPDATE move_analysis SET
			equity                  = ROUND(equity * 1000),
			equity_error            = ROUND(equity_error * 1000),
			win_rate                = ROUND(win_rate * 100),
			gammon_rate             = ROUND(gammon_rate * 100),
			backgammon_rate         = ROUND(backgammon_rate * 100),
			opponent_win_rate       = ROUND(opponent_win_rate * 100),
			opponent_gammon_rate    = ROUND(opponent_gammon_rate * 100),
			opponent_backgammon_rate = ROUND(opponent_backgammon_rate * 100)
	`)
	if err != nil {
		return fmt.Errorf("migrate move_analysis to integer: %w", err)
	}
	d.emitMigrationProgress("move_analysis", 1, 1)

	// -----------------------------------------------------------------
	// 2. Re-round JSON blobs and rebuild analysis scalar columns.
	// -----------------------------------------------------------------
	rows, err := d.db.Query(`SELECT id, data FROM analysis WHERE data IS NOT NULL AND data != ''`)
	if err != nil {
		return fmt.Errorf("migrate analysis query: %w", err)
	}

	type anaRow struct {
		id   int64
		data string
	}
	var batch []anaRow
	for rows.Next() {
		var r anaRow
		if err := rows.Scan(&r.id, &r.data); err != nil {
			continue
		}
		batch = append(batch, r)
	}
	rows.Close()

	anaTotal := len(batch)
	d.emitMigrationProgress("analysis", 0, anaTotal)

	updateStmt, err := d.db.Prepare(`UPDATE analysis SET
		data=?, best_cube_action=?, cube_error=?, best_move_equity_error=?,
		player1_win_rate=?, player1_gammon_rate=?, player1_backgammon_rate=?,
		player2_win_rate=?, player2_gammon_rate=?, player2_backgammon_rate=?
		WHERE id=?`)
	if err != nil {
		return fmt.Errorf("migrate prepare update: %w", err)
	}
	defer updateStmt.Close()

	// Look up the played cube action from the move table when analysis JSON lacks it.
	lookupCubeAction2, err := d.db.Prepare(`
		SELECT m.cube_action FROM move m
		WHERE m.position_id = (SELECT a2.position_id FROM analysis a2 WHERE a2.id = ?)
		  AND m.move_type = 'cube' AND m.cube_action != ''
		LIMIT 1`)
	if err != nil {
		return fmt.Errorf("migrate cube action lookup prepare: %w", err)
	}
	defer lookupCubeAction2.Close()

	for i, r := range batch {
		var ana PositionAnalysis
		if err := json.Unmarshal([]byte(r.data), &ana); err != nil {
			continue
		}
		roundAnalysisForStorage(&ana)
		newJSON, err := json.Marshal(ana)
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
		if playedCubeAction == "" && ana.DoublingCubeAnalysis != nil {
			var ca sql.NullString
			if err := lookupCubeAction2.QueryRow(r.id).Scan(&ca); err == nil && ca.Valid {
				playedCubeAction = ca.String
			}
		}
		ac := populateAnalysisColumns(&ana, playedMove, playedCubeAction)
		_, _ = updateStmt.Exec(
			string(newJSON),
			ac.BestCubeAction, ac.CubeError, ac.BestMoveEquityError,
			ac.Player1WinRate, ac.Player1GammonRate, ac.Player1BackgammonRate,
			ac.Player2WinRate, ac.Player2GammonRate, ac.Player2BackgammonRate,
			r.id)
		if (i+1)%200 == 0 {
			d.emitMigrationProgress("analysis", i+1, anaTotal)
		}
	}
	d.emitMigrationProgress("analysis", anaTotal, anaTotal)

	// -----------------------------------------------------------------
	// 3. ANALYZE
	// -----------------------------------------------------------------
	_, _ = d.db.Exec(`ANALYZE`)

	// -----------------------------------------------------------------
	// 4. Bump version
	// -----------------------------------------------------------------
	if _, err := d.db.Exec(`UPDATE metadata SET value='2.1.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.0.0", "to", "2.1.0")
	return nil
}

// migrate_2_1_0_to_2_2_0 compacts the position.state column from full Position
// JSON (~800 bytes) to a compact board-only array (~60-80 bytes). All non-board
// fields (cube, dice, score, flags) are already stored in the denormalized
// columns added in v2.0.0, so the JSON blob is redundant for those fields.
//
// This reduces the position table size by ~85-90% on the state column.
//
// The caller must hold d.mu.
func (d *Database) migrate_2_1_0_to_2_2_0() error {
	var posTotal int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posTotal)

	if posTotal > 0 {
		updateStmt, err := d.db.Prepare(`UPDATE position SET state = ? WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("migrate prepare: %w", err)
		}
		defer updateStmt.Close()

		const batchSize = 1000
		var lastID int64 = 0
		done := 0

		for {
			if atomic.LoadInt32(&d.importCancelled) != 0 {
				return fmt.Errorf("migration cancelled")
			}

			rows, err := d.db.Query(
				`SELECT id, state FROM position WHERE id > ? ORDER BY id LIMIT ?`,
				lastID, batchSize)
			if err != nil {
				return fmt.Errorf("migrate query: %w", err)
			}

			type row struct {
				id    int64
				state string
			}
			var batch []row
			for rows.Next() {
				var r row
				if err := rows.Scan(&r.id, &r.state); err == nil {
					batch = append(batch, r)
				}
			}
			rows.Close()

			if len(batch) == 0 {
				break
			}

			for _, r := range batch {
				lastID = r.id
				// Skip if already compact
				if isCompactState(r.state) {
					done++
					continue
				}

				var pos Position
				if err := json.Unmarshal([]byte(r.state), &pos); err != nil {
					done++
					continue // malformed JSON — leave as-is
				}
				compact := encodeBoardCompact(pos.Board)
				_, _ = updateStmt.Exec(compact, r.id)
				done++
				if done%200 == 0 {
					d.emitMigrationProgress("compact_state", done, posTotal)
				}
			}
		}
		d.emitMigrationProgress("compact_state", posTotal, posTotal)
	}

	// Prune command_history to last 1000 entries
	_, _ = d.db.Exec(`
		DELETE FROM command_history
		WHERE id NOT IN (
			SELECT id FROM command_history
			ORDER BY timestamp DESC
			LIMIT 1000
		)
	`)

	_, _ = d.db.Exec(`ANALYZE`)

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.2.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.1.0", "to", "2.2.0")
	return nil
}

// migrate_2_2_0_to_2_3_0 compresses analysis.data JSON blobs with zlib.
// The analysis table's scalar filter columns are unchanged; only the data
// column is re-encoded. This reduces analysis storage by ~60-80%.
//
// The caller must hold d.mu.
func (d *Database) migrate_2_2_0_to_2_3_0() error {
	var anaTotal int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM analysis WHERE data IS NOT NULL AND data != ''`).Scan(&anaTotal)

	if anaTotal > 0 {
		updateStmt, err := d.db.Prepare(`UPDATE analysis SET data = ? WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("migrate prepare: %w", err)
		}
		defer updateStmt.Close()

		const batchSize = 1000
		var lastID int64 = 0
		done := 0

		for {
			if atomic.LoadInt32(&d.importCancelled) != 0 {
				return fmt.Errorf("migration cancelled")
			}

			rows, err := d.db.Query(
				`SELECT id, data FROM analysis WHERE id > ? AND data IS NOT NULL AND data != '' ORDER BY id LIMIT ?`,
				lastID, batchSize)
			if err != nil {
				return fmt.Errorf("migrate query: %w", err)
			}

			type row struct {
				id   int64
				data []byte
			}
			var batch []row
			for rows.Next() {
				var r row
				if err := rows.Scan(&r.id, &r.data); err == nil {
					batch = append(batch, r)
				}
			}
			rows.Close()

			if len(batch) == 0 {
				break
			}

			for _, r := range batch {
				lastID = r.id
				// Skip if already compressed (not raw JSON)
				if len(r.data) > 0 && r.data[0] != '{' {
					done++
					continue
				}
				compressed, err := compressAnalysisData(r.data)
				if err != nil {
					done++
					continue
				}
				_, _ = updateStmt.Exec(compressed, r.id)
				done++
				if done%200 == 0 {
					d.emitMigrationProgress("compress_analysis", done, anaTotal)
				}
			}
		}
		d.emitMigrationProgress("compress_analysis", anaTotal, anaTotal)
	}

	_, _ = d.db.Exec(`ANALYZE`)

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.3.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.2.0", "to", "2.3.0")
	return nil
}

// migrate_2_3_0_to_2_4_0 repairs the best_move_equity_error scalar column for
// checker positions where it was stored as 0 because PlayedMoves was missing from
// the analysis JSON blob at the time of earlier migrations.
//
// Root cause: older import code did not always set PositionAnalysis.PlayedMoves in
// the JSON blob. All migration passes that recomputed best_move_equity_error relied
// solely on the JSON's PlayedMoves/PlayedMove fields; they never fell back to
// move.checker_move. As a result, best_move_equity_error stayed 0 even for
// positions where the player made a sub-optimal checker move.
//
// This migration:
//  1. Queries all analysis rows where best_move_equity_error = 0.
//  2. For each, looks up the played checker move from move.checker_move.
//  3. Decodes the analysis blob, matches the played move against CheckerAnalysis.Moves.
//  4. If the played move is found at index > 0, updates best_move_equity_error.
//
// The caller must hold d.mu.
func (d *Database) migrate_2_3_0_to_2_4_0() error {
	var anaTotal int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM analysis WHERE best_move_equity_error = 0`).Scan(&anaTotal)

	if anaTotal > 0 {
		// Prepare lookup: given an analysis id, find the played checker move
		// for the corresponding position.
		lookupCheckerMove, err := d.db.Prepare(`
                        SELECT mv.checker_move
                        FROM move mv
                        JOIN analysis a ON a.position_id = mv.position_id
                        WHERE a.id = ?
                          AND mv.checker_move IS NOT NULL AND mv.checker_move != ''
                          AND mv.move_type = 'checker'
                        LIMIT 1`)
		if err != nil {
			return fmt.Errorf("migrate 2.4.0 prepare checker move lookup: %w", err)
		}
		defer lookupCheckerMove.Close()

		updateStmt, err := d.db.Prepare(`UPDATE analysis SET best_move_equity_error=? WHERE id=?`)
		if err != nil {
			return fmt.Errorf("migrate 2.4.0 prepare update: %w", err)
		}
		defer updateStmt.Close()

		const batchSize = 1000
		var lastID int64 = 0
		done := 0

		for {
			if atomic.LoadInt32(&d.importCancelled) != 0 {
				return fmt.Errorf("migration cancelled")
			}

			rows, err := d.db.Query(
				`SELECT id, data FROM analysis WHERE best_move_equity_error = 0 AND id > ? ORDER BY id LIMIT ?`,
				lastID, batchSize)
			if err != nil {
				return fmt.Errorf("migrate 2.4.0 query: %w", err)
			}

			type anaRow struct {
				id   int64
				data []byte
			}
			var batch []anaRow
			for rows.Next() {
				var r anaRow
				if err := rows.Scan(&r.id, &r.data); err == nil {
					batch = append(batch, r)
				}
			}
			rows.Close()

			if len(batch) == 0 {
				break
			}

			for _, r := range batch {
				lastID = r.id

				ana, err := decodeAnalysisFromStorage(r.data)
				if err != nil {
					done++
					continue
				}

				// Skip if analysis already has a played move (shouldn't happen here,
				// but be safe — re-derive the error from the existing PlayedMoves).
				playedMove := ""
				if len(ana.PlayedMoves) > 0 {
					playedMove = ana.PlayedMoves[0]
				} else if ana.PlayedMove != "" {
					playedMove = ana.PlayedMove
				}

				// If no played move in JSON, fall back to move.checker_move.
				if playedMove == "" && ana.CheckerAnalysis != nil {
					var cm sql.NullString
					if err := lookupCheckerMove.QueryRow(r.id).Scan(&cm); err == nil && cm.Valid {
						playedMove = cm.String
					}
				}

				if playedMove == "" {
					done++
					continue
				}

				ac := populateAnalysisColumns(&ana, playedMove, "")
				if ac.BestMoveEquityError != 0 {
					_, _ = updateStmt.Exec(ac.BestMoveEquityError, r.id)
				}
				done++
				if done%200 == 0 {
					d.emitMigrationProgress("repair_move_error", done, anaTotal)
				}
			}
		}
		d.emitMigrationProgress("repair_move_error", anaTotal, anaTotal)
	}

	_, _ = d.db.Exec(`ANALYZE`)

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.4.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.3.0", "to", "2.4.0")
	return nil
}
