package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
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
			if err := ctx.Err(); err != nil {
				return err
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

// migrate_2_4_0_to_2_5_0 adds the is_forced column to analysis and backfills
// checker positions where exactly one legal move was stored (len(Moves) == 1).
//
// A position is "forced" in the gnuBG sense when there is only one legal checker
// play (pmr->ml.cMoves == 1 in gnubg/analysis.c:458). blunderDB detects this by
// checking that CheckerAnalysis.Moves has exactly one entry.
func (d *Database) migrate_2_4_0_to_2_5_0(ctx context.Context) error {
	// Add the column (idempotent: no-op if it already exists after a partial run).
	if _, err := d.db.Exec(`ALTER TABLE analysis ADD COLUMN is_forced INTEGER NOT NULL DEFAULT 0`); err != nil {
		// SQLite returns "duplicate column name" on repeat; ignore that specific error.
		if err.Error() != `duplicate column name: is_forced` {
			return fmt.Errorf("migrate 2.5.0 add column: %w", err)
		}
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

			rows, err := d.db.Query(`
				SELECT a.id, a.data
				FROM analysis a
				JOIN position p ON p.id = a.position_id
				WHERE p.decision_type = 0 AND a.id > ?
				ORDER BY a.id LIMIT ?`, lastID, batchSize)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("migrate 2.5.0 query: %w", err)
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

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.5.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.4.0", "to", "2.5.0")
	return nil
}

// migrate_2_5_0_to_2_6_0 adds the is_close_cube column to analysis and backfills
// cube positions using the gnuBG isCloseCubedecision predicate (eval.c:5088):
//
//	rDouble = min(DoubleTakeEquity, 1.0)
//	isClose = (OptimalEquity - rDouble) < 0.16
//
// Take/Pass positions always get is_close_cube = 1 (cube was already offered).
func (d *Database) migrate_2_5_0_to_2_6_0(ctx context.Context) error {
	if _, err := d.db.Exec(`ALTER TABLE analysis ADD COLUMN is_close_cube INTEGER NOT NULL DEFAULT 0`); err != nil {
		if err.Error() != `duplicate column name: is_close_cube` {
			return fmt.Errorf("migrate 2.6.0 add column: %w", err)
		}
	}

	if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_analysis_is_close_cube ON analysis(is_close_cube) WHERE is_close_cube = 1`); err != nil {
		return fmt.Errorf("migrate 2.6.0 create index: %w", err)
	}

	// Backfill all cube positions (decision_type = 1).
	var total int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM analysis a JOIN position p ON p.id = a.position_id WHERE p.decision_type = 1`).Scan(&total)

	if total > 0 {
		updateStmt, err := d.db.Prepare(`UPDATE analysis SET is_close_cube = 1 WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("migrate 2.6.0 prepare update: %w", err)
		}
		defer updateStmt.Close()

		// Also look up played cube action from the move table for Take/Pass detection.
		lookupAction, err := d.db.Prepare(`
			SELECT COALESCE(mv.cube_action, '')
			FROM move mv
			JOIN analysis a ON a.position_id = mv.position_id
			WHERE a.id = ? AND mv.cube_action IS NOT NULL AND mv.cube_action != ''
			LIMIT 1`)
		if err != nil {
			return fmt.Errorf("migrate 2.6.0 prepare action lookup: %w", err)
		}
		defer lookupAction.Close()

		tx, err := d.db.Begin()
		if err != nil {
			return fmt.Errorf("migrate 2.6.0 begin tx: %w", err)
		}

		const batchSize = 1000
		var lastID int64
		done := 0

		for {
			if err := ctx.Err(); err != nil {
				tx.Rollback()
				return err
			}

			rows, err := d.db.Query(`
				SELECT a.id, a.data
				FROM analysis a
				JOIN position p ON p.id = a.position_id
				WHERE p.decision_type = 1 AND a.id > ?
				ORDER BY a.id LIMIT ?`, lastID, batchSize)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("migrate 2.6.0 query: %w", err)
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
				ana, err := decodeAnalysisFromStorage(r.data)
				if err != nil {
					done++
					continue
				}

				// Determine the played cube action from the move table if not in blob.
				playedAction := ""
				if len(ana.PlayedCubeActions) > 0 {
					playedAction = ana.PlayedCubeActions[0]
				} else if ana.PlayedCubeAction != "" {
					playedAction = ana.PlayedCubeAction
				}
				if playedAction == "" {
					var ca string
					if err := lookupAction.QueryRow(r.id).Scan(&ca); err == nil {
						playedAction = ca
					}
				}

				if computeIsCloseCube(ana.DoublingCubeAnalysis, playedAction) == 1 {
					if _, err := tx.Stmt(updateStmt).Exec(r.id); err != nil {
						tx.Rollback()
						return fmt.Errorf("migrate 2.6.0 update: %w", err)
					}
				}
				done++
				if done%500 == 0 {
					d.emitMigrationProgress("is_close_cube_backfill", done, total)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate 2.6.0 commit: %w", err)
		}
		d.emitMigrationProgress("is_close_cube_backfill", total, total)
	}

	_, _ = d.db.Exec(`ANALYZE`)

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.6.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.5.0", "to", "2.6.0")
	return nil
}

// migrate_2_9_0_to_2_10_0 adds the is_cube_response column to position and
// backfills it: a cube position (decision_type = 1) is flagged 1 when any of its
// recorded played cube actions is a take/pass response (engine.IsResponseCubeAction),
// as opposed to a doubling decision (Double / No Double / Redouble). This lets the
// search filter distinguish "double/no-double" from "take/pass" cube decisions.
func (d *Database) migrate_2_9_0_to_2_10_0(ctx context.Context) error {
	if _, err := d.db.Exec(`ALTER TABLE position ADD COLUMN is_cube_response INTEGER NOT NULL DEFAULT 0`); err != nil {
		if err.Error() != `duplicate column name: is_cube_response` {
			return fmt.Errorf("migrate 2.10.0 add column: %w", err)
		}
	}

	if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_position_cube_response ON position(decision_type, is_cube_response)`); err != nil {
		// On minimal legacy schemas decision_type may not exist yet; the v2
		// column backfill (ensureAllTablesExist) re-creates this index afterwards.
		slog.Debug("migrate 2.10.0 deferring cube_response index", "err", err)
	}

	// Backfill all cube positions (decision_type = 1) from the move table.
	var total int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM position WHERE decision_type = 1`).Scan(&total)

	if total > 0 {
		updateStmt, err := d.db.Prepare(`UPDATE position SET is_cube_response = 1 WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("migrate 2.10.0 prepare update: %w", err)
		}
		defer updateStmt.Close()

		// Look up the played cube actions for a position from the move table.
		lookupActions, err := d.db.Prepare(`
			SELECT COALESCE(cube_action, '')
			FROM move
			WHERE position_id = ? AND cube_action IS NOT NULL AND cube_action != ''`)
		if err != nil {
			return fmt.Errorf("migrate 2.10.0 prepare action lookup: %w", err)
		}
		defer lookupActions.Close()

		tx, err := d.db.Begin()
		if err != nil {
			return fmt.Errorf("migrate 2.10.0 begin tx: %w", err)
		}

		const batchSize = 1000
		var lastID int64
		done := 0

		for {
			if err := ctx.Err(); err != nil {
				tx.Rollback()
				return err
			}

			rows, err := d.db.Query(`
				SELECT id FROM position
				WHERE decision_type = 1 AND id > ?
				ORDER BY id LIMIT ?`, lastID, batchSize)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("migrate 2.10.0 query: %w", err)
			}
			var batch []int64
			for rows.Next() {
				var id int64
				if err := rows.Scan(&id); err == nil {
					batch = append(batch, id)
				}
			}
			rows.Close()

			if len(batch) == 0 {
				break
			}

			for _, id := range batch {
				lastID = id

				// A position is a response if ANY of its played cube actions is a
				// take/pass (OR semantics across matches for a deduped position).
				isResp := false
				actRows, err := lookupActions.Query(id)
				if err == nil {
					for actRows.Next() {
						var ca string
						if err := actRows.Scan(&ca); err == nil && engine.IsResponseCubeAction(ca) {
							isResp = true
						}
					}
					actRows.Close()
				}

				if isResp {
					if _, err := tx.Stmt(updateStmt).Exec(id); err != nil {
						tx.Rollback()
						return fmt.Errorf("migrate 2.10.0 update: %w", err)
					}
				}
				done++
				if done%500 == 0 {
					d.emitMigrationProgress("is_cube_response_backfill", done, total)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate 2.10.0 commit: %w", err)
		}
		d.emitMigrationProgress("is_cube_response_backfill", total, total)
	}

	_, _ = d.db.Exec(`ANALYZE`)

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.10.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.9.0", "to", "2.10.0")
	return nil
}

// migrate_2_6_0_to_2_7_0 fixes Zobrist hashes that were computed with the wrong
// cube-value convention. The ZobristHash function received Cube.Value as an EXPONENT
// (0=cube@1, 1=cube@2, 2=cube@4, …) but passed it to cubeValueIndex() which expects
// the ACTUAL cube value (1, 2, 4, 8, …). For exponent=0 both functions return index 0,
// so those hashes are correct. For exponent >= 1 the old index was floor(log2(exp))
// instead of exp, producing wrong (and sometimes colliding) hashes.
//
// Fix: recompute the ZobristHash from the stored position state for every position
// with cube_value >= 1. This is idempotent: already-correct hashes are unchanged
// (recomputing with the fixed function returns the same correct value).
func (d *Database) migrate_2_6_0_to_2_7_0(ctx context.Context) error {
	// If the position table doesn't yet have a zobrist_hash column (possible when
	// migrating from a very old schema that was never at 2.0.0), skip the
	// hash-patching step — the hashes will be computed correctly on first use.
	var colCount int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('position') WHERE name='zobrist_hash'`).Scan(&colCount)
	if colCount == 0 {
		if _, err := d.db.Exec(`UPDATE metadata SET value='2.7.0' WHERE key='database_version'`); err != nil {
			return fmt.Errorf("migrate 2.7.0 version bump: %w", err)
		}
		slog.Info("database upgraded", "from", "2.6.0", "to", "2.7.0", "positions_rehashed", 0)
		return nil
	}

	rows, err := d.db.Query(`SELECT id, state, decision_type, player_on_roll, dice_1, dice_2, cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver FROM position WHERE cube_value >= 1`)
	if err != nil {
		return fmt.Errorf("migrate 2.7.0 query positions: %w", err)
	}

	type fix struct {
		id      int64
		newHash int64
	}
	var fixes []fix
	for rows.Next() {
		var id int64
		var state string
		var decisionType, playerOnRoll, dice1, dice2, cubeValue, cubeOwner, score1, score2, hasJacoby, hasBeaver int
		if err := rows.Scan(&id, &state, &decisionType, &playerOnRoll, &dice1, &dice2, &cubeValue, &cubeOwner, &score1, &score2, &hasJacoby, &hasBeaver); err != nil {
			rows.Close()
			return fmt.Errorf("migrate 2.7.0 scan: %w", err)
		}
		pos := reconstructPosition(id, state, decisionType, playerOnRoll, dice1, dice2, cubeValue, cubeOwner, score1, score2, hasJacoby, hasBeaver)
		newHash := engine.ZobristHash(&pos)
		fixes = append(fixes, fix{id: id, newHash: int64(newHash)})
	}
	rows.Close()

	if len(fixes) == 0 {
		if _, err := d.db.Exec(`UPDATE metadata SET value='2.7.0' WHERE key='database_version'`); err != nil {
			return fmt.Errorf("migrate 2.7.0 version bump: %w", err)
		}
		slog.Info("database upgraded", "from", "2.6.0", "to", "2.7.0", "positions_rehashed", 0)
		return nil
	}

	// Drop the unique index before bulk-updating hashes to avoid transient
	// uniqueness violations (two rows may temporarily have the same hash during
	// the update pass).
	if _, err := d.db.Exec(`DROP INDEX IF EXISTS idx_position_zobrist`); err != nil {
		return fmt.Errorf("migrate 2.7.0 drop index: %w", err)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("migrate 2.7.0 begin tx: %w", err)
	}

	stmt, err := tx.Prepare(`UPDATE position SET zobrist_hash = ? WHERE id = ?`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("migrate 2.7.0 prepare update: %w", err)
	}
	defer stmt.Close()

	for _, f := range fixes {
		if err := ctx.Err(); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := stmt.Exec(f.newHash, f.id); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrate 2.7.0 update id %d: %w", f.id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate 2.7.0 commit: %w", err)
	}

	// Recreate the unique index. If there are genuine hash collisions after the
	// fix (astronomically unlikely), this will fail — treat as a hard error.
	if _, err := d.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_position_zobrist ON position(zobrist_hash)`); err != nil {
		return fmt.Errorf("migrate 2.7.0 recreate unique index: %w", err)
	}

	_, _ = d.db.Exec(`ANALYZE`)

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.7.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate 2.7.0 version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.6.0", "to", "2.7.0", "positions_rehashed", len(fixes))
	return nil
}

// migrate_2_7_0_to_2_8_0 adds an exclude_position column to search_history and
// filter_library so the "Sauf" (exclusion structure) of a search can be persisted
// and restored on replay. The column is nullable; existing rows keep NULL (no
// exclusion structure).
func (d *Database) migrate_2_7_0_to_2_8_0() error {
	for _, stmt := range []struct{ table, col string }{
		{"search_history", "exclude_position"},
		{"filter_library", "exclude_position"},
	} {
		if _, err := d.db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s TEXT`, stmt.table, stmt.col)); err != nil {
			// Tolerate a duplicate column (idempotent retry) and a missing table:
			// ensureAllTablesExist runs after migrations and (re)creates absent
			// tables with the exclude_position column already present.
			msg := err.Error()
			if !strings.Contains(msg, `duplicate column name: `+stmt.col) && !strings.Contains(msg, `no such table: `+stmt.table) {
				return fmt.Errorf("migrate 2.8.0 add column: %w", err)
			}
		}
	}

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.8.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate 2.8.0 version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.7.0", "to", "2.8.0")
	return nil
}

// migrate_2_8_0_to_2_9_0 adds a scope column to command_history, search_history
// and filter_library. SQLite previously ignored the scope argument on these
// stores (the GUI/CLI always uses the empty scope); the column lets a
// multi-tenant SQLite-server isolate each tenant's command/search history and
// saved filters, mirroring the tenant_id scoping the PostgreSQL backend already
// has. Existing rows default to the empty scope so GUI/CLI data is unchanged.
func (d *Database) migrate_2_8_0_to_2_9_0() error {
	for _, table := range []string{"command_history", "search_history", "filter_library"} {
		if _, err := d.db.Exec(fmt.Sprintf(
			`ALTER TABLE %s ADD COLUMN scope TEXT NOT NULL DEFAULT ''`, table)); err != nil {
			// Tolerate an idempotent retry (duplicate column) and a missing table
			// (ensureAllTablesExist recreates it with the scope column present).
			msg := err.Error()
			if !strings.Contains(msg, `duplicate column name: scope`) && !strings.Contains(msg, `no such table: `+table) {
				return fmt.Errorf("migrate 2.9.0 add scope to %s: %w", table, err)
			}
		}
	}
	for _, idx := range []string{
		`CREATE INDEX IF NOT EXISTS idx_command_history_scope ON command_history(scope, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_search_history_scope  ON search_history(scope, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_filter_library_scope_name ON filter_library(scope, name)`,
	} {
		if _, err := d.db.Exec(idx); err != nil {
			// A missing table is recreated (with the scope column) by
			// ensureAllTablesExist after migrations; the index is a perf-only
			// optimisation, so a missing table here is not fatal.
			if !strings.Contains(err.Error(), `no such table`) {
				return fmt.Errorf("migrate 2.9.0 create index: %w", err)
			}
		}
	}

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.9.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate 2.9.0 version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.8.0", "to", "2.9.0")
	return nil
}

// migrate_2_10_0_to_2_11_0 adds the anki_review_log table, an append-only
// journal of every spaced-repetition review (rating + FSRS outcome). It powers
// retention/streak statistics, the review heatmap and a faithful undo of the
// last review. The table is also (re)created by ensureAllTablesExist, so a
// missing table here is not fatal; this step exists so the version bump is
// recorded on an existing user database.
func (d *Database) migrate_2_10_0_to_2_11_0() error {
	if _, err := d.db.Exec(`
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
	`); err != nil {
		return fmt.Errorf("migrate 2.11.0 create anki_review_log: %w", err)
	}

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.11.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate 2.11.0 version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.10.0", "to", "2.11.0")
	return nil
}

// migrate_2_11_0_to_2_12_0 extends anki_card with suspend/bury state. A
// suspended card is excluded from review indefinitely; a buried card is hidden
// until buried_until passes (typically the next day). The columns are also
// (re)added by ensureAllTablesExist, so a column that already exists here is
// not fatal; this step exists so the version bump is recorded on an existing
// user database.
func (d *Database) migrate_2_11_0_to_2_12_0() error {
	for _, stmt := range []string{
		`ALTER TABLE anki_card ADD COLUMN suspended INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE anki_card ADD COLUMN buried_until DATETIME`,
	} {
		_, _ = d.db.Exec(stmt) // ignore error: column may already exist
	}

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.12.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate 2.12.0 version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.11.0", "to", "2.12.0")
	return nil
}

// migrate_2_12_0_to_2_13_0 adds position.individually_imported (ADR-0001) and
// backfills it from the only signal an existing database carries: a position
// reachable from no move never came from a match.
//
// The backfill is a one-shot reconstruction of history, not a definition. It
// has two known error classes, both accepted deliberately:
//   - False positives: positions created by a cross-format "enrich" import
//     (ingest.WriteMatch skips move creation when enriching) are match-sourced
//     yet have no move row, so they are marked individual here.
//   - False negatives: a position that was individually imported *before* a
//     match that also contained it is indistinguishable from a plain match
//     position. That information does not exist in the database and cannot be
//     recovered.
//
// From here on the flag is written at import time and is exact.
func (d *Database) migrate_2_12_0_to_2_13_0() error {
	_, _ = d.db.Exec(`ALTER TABLE position ADD COLUMN individually_imported INTEGER NOT NULL DEFAULT 0`) // may already exist

	if _, err := d.db.Exec(`
		UPDATE position SET individually_imported = 1
		WHERE NOT EXISTS (SELECT 1 FROM move WHERE move.position_id = position.id)`); err != nil {
		return fmt.Errorf("migrate 2.13.0 backfill individually_imported: %w", err)
	}

	if _, err := d.db.Exec(
		`CREATE INDEX IF NOT EXISTS idx_position_individual ON position(individually_imported) WHERE individually_imported = 1`); err != nil {
		return fmt.Errorf("migrate 2.13.0 create index: %w", err)
	}

	if _, err := d.db.Exec(`UPDATE metadata SET value='2.13.0' WHERE key='database_version'`); err != nil {
		return fmt.Errorf("migrate 2.13.0 version bump: %w", err)
	}

	slog.Info("database upgraded", "from", "2.12.0", "to", "2.13.0")
	return nil
}

// runMigrationChain reads the recorded schema version and applies the
// sequential upgrade steps up to the current DatabaseVersion, then verifies
// the expected tables and metadata keys exist. It is shared by the GUI/CLI
// Database wrapper (OpenDatabase) and, via the registered storage migrator
// (migrate_hook.go), by the headless storage backend opening a pre-existing
// database. The caller must hold d.mu when d is a shared instance; the
// storage path uses a transient Database, so no lock is needed there.
func (d *Database) runMigrationChain(ctx context.Context) error {
	var err error
	// Check the database version
	var dbVersion string
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	// Auto-migrate from 1.0.0 to 1.1.0
	if dbVersion == "1.0.0" {
		var cmdHistExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='command_history'`).Scan(&cmdHistExists)
		if err == sql.ErrNoRows {
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS command_history (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					command TEXT,
					timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
				)
			`)
			if err != nil {
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.1.0")
			if err != nil {
				return err
			}
			dbVersion = "1.1.0"
			slog.Info("database upgraded", "from", "1.0.0", "to", "1.1.0")
		}
	}

	// Auto-migrate from 1.1.0 to 1.2.0
	if dbVersion == "1.1.0" {
		var filterLibExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='filter_library'`).Scan(&filterLibExists)
		if err == sql.ErrNoRows {
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS filter_library (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT,
					command TEXT,
					edit_position TEXT
				)
			`)
			if err != nil {
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.2.0")
			if err != nil {
				return err
			}
			dbVersion = "1.2.0"
			slog.Info("database upgraded", "from", "1.1.0", "to", "1.2.0")
		}
	}

	// Auto-migrate from 1.2.0 to 1.3.0
	if dbVersion == "1.2.0" {
		var searchHistoryExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='search_history'`).Scan(&searchHistoryExists)
		if err == sql.ErrNoRows {
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS search_history (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					command TEXT,
					position TEXT,
					timestamp INTEGER
				)
			`)
			if err != nil {
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.3.0")
			if err != nil {
				return err
			}
			dbVersion = "1.3.0"
			slog.Info("database upgraded", "from", "1.2.0", "to", "1.3.0")
		}
	}

	// Auto-migrate from 1.3.0 to 1.4.0
	if dbVersion == "1.3.0" {
		var matchExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&matchExists)
		if err == sql.ErrNoRows {
			// Create match-related tables
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
					equity REAL,
					equity_error REAL,
					win_rate REAL,
					gammon_rate REAL,
					backgammon_rate REAL,
					opponent_win_rate REAL,
					opponent_gammon_rate REAL,
					opponent_backgammon_rate REAL,
					FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
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

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.4.0")
			if err != nil {
				return err
			}
			dbVersion = "1.4.0"
			slog.Info("database upgraded", "from", "1.3.0", "to", "1.4.0")
		}
	}

	// Add match_hash column to existing v1.4.0 databases if it doesn't exist
	if dbVersion == "1.4.0" {
		// Check if match_hash column exists
		var colInfo string
		err = d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&colInfo)
		if err == nil && !strings.Contains(colInfo, "match_hash") {
			// Add match_hash column
			_, err = d.db.Exec(`ALTER TABLE match ADD COLUMN match_hash TEXT`)
			if err != nil {
				return err
			}

			// Create index on match_hash
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
			if err != nil {
				return err
			}

			// Populate match_hash for existing matches using stored data
			// This uses a fallback hash based on stored moves since we don't have the original file
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
		}
	}

	// Auto-migrate from 1.4.0 to 1.5.0
	if dbVersion == "1.4.0" {
		var collectionExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='collection'`).Scan(&collectionExists)
		if err == sql.ErrNoRows {
			// Create collection-related tables
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

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.5.0")
			if err != nil {
				return err
			}
			dbVersion = "1.5.0"
			slog.Info("database upgraded", "from", "1.4.0", "to", "1.5.0")
		}
	}

	// Auto-migrate from 1.5.0 to 1.6.0
	if dbVersion == "1.5.0" {
		var tournamentExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='tournament'`).Scan(&tournamentExists)
		if err == sql.ErrNoRows {
			// Create tournament table
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

			// Add tournament_id column to match table
			_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL`)

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.6.0")
			if err != nil {
				return err
			}
			dbVersion = "1.6.0"
			slog.Info("database upgraded", "from", "1.5.0", "to", "1.6.0")
		}
	}

	// Auto-migrate from 1.6.0 to 1.7.0
	if dbVersion == "1.6.0" {
		// Add last_visited_position column to match table
		_, _ = d.db.Exec(`ALTER TABLE match ADD COLUMN last_visited_position INTEGER DEFAULT -1`)

		_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.7.0")
		if err != nil {
			return err
		}
		dbVersion = "1.7.0"
		slog.Info("database upgraded", "from", "1.6.0", "to", "1.7.0")
	}

	// Auto-migrate from 1.7.0 to 1.8.0
	if dbVersion == "1.7.0" {
		// Add created_at column to comment table
		_, _ = d.db.Exec(`ALTER TABLE comment ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP`)
		// Backfill existing rows that have NULL created_at
		_, _ = d.db.Exec(`UPDATE comment SET created_at = CURRENT_TIMESTAMP WHERE created_at IS NULL`)

		_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.8.0")
		if err != nil {
			return err
		}
		dbVersion = "1.8.0"
		slog.Info("database upgraded", "from", "1.7.0", "to", "1.8.0")
	}

	// Auto-migrate from 1.8.0 to 1.9.0
	if dbVersion == "1.8.0" {
		// Add modified_at column to comment table
		_, _ = d.db.Exec(`ALTER TABLE comment ADD COLUMN modified_at DATETIME`)

		_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.9.0")
		if err != nil {
			return err
		}
		dbVersion = "1.9.0"
		slog.Info("database upgraded", "from", "1.8.0", "to", "1.9.0")
	}

	// Auto-migrate from 1.9.0 to 2.0.0
	// Backfills new scalar columns, deduplicates positions, and creates indexes.
	if dbVersion == "1.9.0" {
		if err := d.migrate_1_9_0_to_2_0_0(ctx); err != nil {
			return fmt.Errorf("migration 1.9.0→2.0.0 failed: %w", err)
		}
		dbVersion = "2.0.0"
	}

	// Auto-migrate from 2.0.0 to 2.1.0
	// Converts analysis/move_analysis REAL columns to integer-scaled INTEGER values
	// and re-rounds all JSON blobs for compact storage.
	if dbVersion == "2.0.0" {
		if err := d.migrate_2_0_0_to_2_1_0(); err != nil {
			return fmt.Errorf("migration 2.0.0→2.1.0 failed: %w", err)
		}
		dbVersion = "2.1.0"
	}

	// Auto-migrate from 2.1.0 to 2.2.0
	// Compacts position.state from full Position JSON to board-only array,
	// reducing storage by ~85-90% on the state column. Also prunes command history.
	if dbVersion == "2.1.0" {
		if err := d.migrate_2_1_0_to_2_2_0(ctx); err != nil {
			return fmt.Errorf("migration 2.1.0→2.2.0 failed: %w", err)
		}
		dbVersion = "2.2.0"
	}

	// Auto-migrate from 2.2.0 to 2.3.0
	// Compresses analysis.data JSON blobs with zlib, reducing analysis storage
	// by ~60-80%. Scalar filter columns are unchanged.
	if dbVersion == "2.2.0" {
		if err := d.migrate_2_2_0_to_2_3_0(ctx); err != nil {
			return fmt.Errorf("migration 2.2.0→2.3.0 failed: %w", err)
		}
		dbVersion = "2.3.0"
	}

	// Auto-migrate from 2.3.0 to 2.4.0
	// Repairs best_move_equity_error for positions where PlayedMoves was missing
	// from the analysis JSON blob in earlier migrations, by looking up move.checker_move.
	if dbVersion == "2.3.0" {
		if err := d.migrate_2_3_0_to_2_4_0(ctx); err != nil {
			return fmt.Errorf("migration 2.3.0→2.4.0 failed: %w", err)
		}
		dbVersion = "2.4.0"
	}

	// Auto-migrate from 2.4.0 to 2.5.0
	// Adds is_forced column to analysis; backfills checker positions with exactly
	// one legal move (len(CheckerAnalysis.Moves) == 1).
	if dbVersion == "2.4.0" {
		if err := d.migrate_2_4_0_to_2_5_0(ctx); err != nil {
			return fmt.Errorf("migration 2.4.0→2.5.0 failed: %w", err)
		}
		dbVersion = "2.5.0"
	}

	// Auto-migrate from 2.5.0 to 2.6.0
	// Adds is_close_cube column to analysis; backfills cube positions using the
	// gnuBG isCloseCubedecision predicate (threshold 0.16 equity, eval.c:5088).
	if dbVersion == "2.5.0" {
		if err := d.migrate_2_5_0_to_2_6_0(ctx); err != nil {
			return fmt.Errorf("migration 2.5.0→2.6.0 failed: %w", err)
		}
		dbVersion = "2.6.0"
	}

	// Auto-migrate from 2.6.0 to 2.7.0
	// Recomputes zobrist_hash for positions with cube_value >= 1 to fix a bug
	// where cubeValueIndex() was called with an exponent instead of actual cube
	// value, causing cube=0 (initial) and cube=1 (2-cube) to hash identically.
	if dbVersion == "2.6.0" {
		if err := d.migrate_2_6_0_to_2_7_0(ctx); err != nil {
			return fmt.Errorf("migration 2.6.0→2.7.0 failed: %w", err)
		}
		dbVersion = "2.7.0"
	}

	// Auto-migrate from 2.7.0 to 2.8.0
	// Adds exclude_position column to search_history and filter_library to persist
	// the "Sauf" (exclusion structure) of a search.
	if dbVersion == "2.7.0" {
		if err := d.migrate_2_7_0_to_2_8_0(); err != nil {
			return fmt.Errorf("migration 2.7.0→2.8.0 failed: %w", err)
		}
		dbVersion = "2.8.0"
	}

	// Auto-migrate from 2.8.0 to 2.9.0
	// Adds a scope column to command_history, search_history and filter_library
	// so a multi-tenant SQLite-server isolates per-tenant history/filters.
	if dbVersion == "2.8.0" {
		if err := d.migrate_2_8_0_to_2_9_0(); err != nil {
			return fmt.Errorf("migration 2.8.0→2.9.0 failed: %w", err)
		}
		dbVersion = "2.9.0"
	}

	// Auto-migrate from 2.9.0 to 2.10.0
	// Adds is_cube_response column to position; backfills cube positions by
	// detecting take/pass responses from the move table (engine.IsResponseCubeAction).
	if dbVersion == "2.9.0" {
		if err := d.migrate_2_9_0_to_2_10_0(ctx); err != nil {
			return fmt.Errorf("migration 2.9.0→2.10.0 failed: %w", err)
		}
		dbVersion = "2.10.0"
	}

	// Auto-migrate from 2.10.0 to 2.11.0
	// Adds the anki_review_log table (append-only review journal).
	if dbVersion == "2.10.0" {
		if err := d.migrate_2_10_0_to_2_11_0(); err != nil {
			return fmt.Errorf("migration 2.10.0→2.11.0 failed: %w", err)
		}
		dbVersion = "2.11.0"
	}

	// Auto-migrate from 2.11.0 to 2.12.0
	// Adds anki_card suspend/bury columns.
	if dbVersion == "2.11.0" {
		if err := d.migrate_2_11_0_to_2_12_0(); err != nil {
			return fmt.Errorf("migration 2.11.0→2.12.0 failed: %w", err)
		}
		dbVersion = "2.12.0"
	}

	// Auto-migrate from 2.12.0 to 2.13.0
	// Adds position.individually_imported and backfills it from the move graph.
	if dbVersion == "2.12.0" {
		if err := d.migrate_2_12_0_to_2_13_0(); err != nil {
			return fmt.Errorf("migration 2.12.0→2.13.0 failed: %w", err)
		}
		dbVersion = "2.13.0"
	}

	// Ensure all required tables and columns exist.
	// This repairs databases that were migrated through versions that skipped
	// creating some tables (e.g. filter_library was missing from some migration paths).
	if err := d.ensureAllTablesExist(); err != nil {
		return err
	}

	// Build required tables list based on the FINAL dbVersion (after all migrations)
	requiredTables := []string{"position", "analysis", "comment", "metadata"}
	if dbVersion >= "1.1.0" {
		requiredTables = append(requiredTables, "command_history")
	}
	if dbVersion >= "1.2.0" {
		requiredTables = append(requiredTables, "filter_library")
	}
	if dbVersion >= "1.3.0" {
		requiredTables = append(requiredTables, "search_history")
	}
	if dbVersion >= "1.4.0" {
		requiredTables = append(requiredTables, "match", "game", "move", "move_analysis")
	}
	if dbVersion >= "1.5.0" {
		requiredTables = append(requiredTables, "collection", "collection_position")
	}
	if dbVersion >= "1.6.0" {
		requiredTables = append(requiredTables, "tournament")
	}

	for _, table := range requiredTables {
		var tableName string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&tableName)
		if err != nil {
			return err
		}
		if tableName != table {
			return fmt.Errorf("required table %s does not exist", table)
		}
	}

	// Check if the required metadata keys exist
	requiredKeys := []string{"database_version"}
	for _, key := range requiredKeys {
		var value string
		err = d.db.QueryRow(`SELECT value FROM metadata WHERE key=?`, key).Scan(&value)
		if err != nil {
			return err
		}
		if value == "" {
			return fmt.Errorf("required metadata key %s does not exist", key)
		}
	}
	return nil
}
