package database

// Migration steps 2.0.0 → 2.5.0: the storage-format rewrites that followed
// the 2.0.0 scalar columns (integer-scaled analysis values, compact board
// state, zlib-compressed analysis blobs, and the played-move error repair).
// See db_migration.go for the registry that runs them and the rules a step
// follows (no version stamp, re-runnable).

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

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
func (d *Database) migrate_2_0_0_to_2_1_0(_ context.Context) error {
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
	type anaRow struct {
		id   int64
		data string
	}
	var batch []anaRow
	err = d.forEachRow(`SELECT id, data FROM analysis WHERE data IS NOT NULL AND data != ''`, nil, func(rows *sql.Rows) {
		var r anaRow
		if err := rows.Scan(&r.id, &r.data); err != nil {
			return
		}
		batch = append(batch, r)
	})
	if err != nil {
		return fmt.Errorf("migrate analysis rows: %w", err)
	}

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
func (d *Database) migrate_2_1_0_to_2_2_0(ctx context.Context) error {
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
			if err := ctx.Err(); err != nil {
				return err
			}

			type row struct {
				id    int64
				state string
			}
			var batch []row
			err := d.forEachRow(
				`SELECT id, state FROM position WHERE id > ? ORDER BY id LIMIT ?`,
				[]any{lastID, batchSize}, func(rows *sql.Rows) {
					var r row
					if err := rows.Scan(&r.id, &r.state); err == nil {
						batch = append(batch, r)
					}
				})
			if err != nil {
				return fmt.Errorf("migrate 2.2.0 rows: %w", err)
			}

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

	return nil
}

// migrate_2_2_0_to_2_3_0 compresses analysis.data JSON blobs with zlib.
// The analysis table's scalar filter columns are unchanged; only the data
// column is re-encoded. This reduces analysis storage by ~60-80%.
//
// The caller must hold d.mu.
func (d *Database) migrate_2_2_0_to_2_3_0(ctx context.Context) error {
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
			if err := ctx.Err(); err != nil {
				return err
			}

			type row struct {
				id   int64
				data []byte
			}
			var batch []row
			err := d.forEachRow(
				`SELECT id, data FROM analysis WHERE id > ? AND data IS NOT NULL AND data != '' ORDER BY id LIMIT ?`,
				[]any{lastID, batchSize}, func(rows *sql.Rows) {
					var r row
					if err := rows.Scan(&r.id, &r.data); err == nil {
						batch = append(batch, r)
					}
				})
			if err != nil {
				return fmt.Errorf("migrate 2.3.0 rows: %w", err)
			}

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
func (d *Database) migrate_2_3_0_to_2_4_0(ctx context.Context) error {
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
			if err := ctx.Err(); err != nil {
				return err
			}

			type anaRow struct {
				id   int64
				data []byte
			}
			var batch []anaRow
			err := d.forEachRow(
				`SELECT id, data FROM analysis WHERE best_move_equity_error = 0 AND id > ? ORDER BY id LIMIT ?`,
				[]any{lastID, batchSize}, func(rows *sql.Rows) {
					var r anaRow
					if err := rows.Scan(&r.id, &r.data); err == nil {
						batch = append(batch, r)
					}
				})
			if err != nil {
				return fmt.Errorf("migrate 2.4.0 rows: %w", err)
			}

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

	return nil
}

// migrate_2_4_0_to_2_5_0 adds the is_forced column to analysis and backfills
// checker positions where exactly one legal move was stored (len(Moves) == 1).
//
// A position is "forced" in the gnuBG sense when there is only one legal checker
// play (pmr->ml.cMoves == 1 in gnubg/analysis.c:458). blunderDB detects this by
// checking that CheckerAnalysis.Moves has exactly one entry.
func (d *Database) migrate_2_4_0_to_2_5_0(ctx context.Context) error {
	// Add the column (idempotent: no-op if it already exists after a partial run).
	if err := d.addColumn(`ALTER TABLE analysis ADD COLUMN is_forced INTEGER NOT NULL DEFAULT 0`, "is_forced"); err != nil {
		return fmt.Errorf("migrate 2.5.0 add column: %w", err)
	}

	// Partial index: accelerates queries that filter on is_forced = 1.
	if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_analysis_is_forced ON analysis(is_forced) WHERE is_forced = 1`); err != nil {
		return fmt.Errorf("migrate 2.5.0 create index: %w", err)
	}

	// Backfill: find all checker analysis rows (position.decision_type = 0) and
	// set is_forced = 1 for those with exactly one candidate move.
	var total int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM analysis a JOIN position p ON p.id = a.position_id WHERE p.decision_type = 0`).Scan(&total)

	if total > 0 {
		updateStmt, err := d.db.Prepare(`UPDATE analysis SET is_forced = 1 WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("migrate 2.5.0 prepare update: %w", err)
		}
		defer updateStmt.Close()

		tx, err := d.db.Begin()
		if err != nil {
			return fmt.Errorf("migrate 2.5.0 begin tx: %w", err)
		}

		const batchSize = 1000
		var lastID int64
		done := 0

		for {
			if err := ctx.Err(); err != nil {
				tx.Rollback()
				return err
			}

			type row struct {
				id   int64
				data []byte
			}
			var batch []row
			err := d.forEachRow(`
				SELECT a.id, a.data
				FROM analysis a
				JOIN position p ON p.id = a.position_id
				WHERE p.decision_type = 0 AND a.id > ?
				ORDER BY a.id LIMIT ?`, []any{lastID, batchSize}, func(rows *sql.Rows) {
				var r row
				if err := rows.Scan(&r.id, &r.data); err == nil {
					batch = append(batch, r)
				}
			})
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("migrate 2.5.0 rows: %w", err)
			}

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
				if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) == 1 {
					if _, err := tx.Stmt(updateStmt).Exec(r.id); err != nil {
						tx.Rollback()
						return fmt.Errorf("migrate 2.5.0 update: %w", err)
					}
				}
				done++
				if done%500 == 0 {
					d.emitMigrationProgress("is_forced_backfill", done, total)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate 2.5.0 commit: %w", err)
		}
		d.emitMigrationProgress("is_forced_backfill", total, total)
	}

	_, _ = d.db.Exec(`ANALYZE`)

	return nil
}
