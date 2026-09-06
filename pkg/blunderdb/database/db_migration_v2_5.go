package database

// Migration steps 2.5.0 → DatabaseVersion: the analysis/position flag columns
// with their backfills (is_forced, is_close_cube, is_cube_response, the
// Zobrist rehash), then the small DDL steps (exclusion structures, scopes,
// Anki review log, provenance, flags, luck). See db_migration.go for the
// registry that runs them and the rules a step follows (no version stamp,
// re-runnable).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// migrate_2_5_0_to_2_6_0 adds the is_close_cube column to analysis and backfills
// cube positions using the gnuBG isCloseCubedecision predicate (eval.c:5088):
//
//	rDouble = min(DoubleTakeEquity, 1.0)
//	isClose = (OptimalEquity - rDouble) < 0.16
//
// Take/Pass positions always get is_close_cube = 1 (cube was already offered).
func (d *Database) migrate_2_5_0_to_2_6_0(ctx context.Context) error {
	if err := d.addColumn("analysis", "is_close_cube INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("migrate 2.6.0 add column: %w", err)
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
		// A Stmt bound to the transaction is a resource of its own; one per
		// batch loop, closed with the loop, not one per row.
		txUpdate := tx.Stmt(updateStmt)
		defer txUpdate.Close()

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
				WHERE p.decision_type = 1 AND a.id > ?
				ORDER BY a.id LIMIT ?`, []any{lastID, batchSize}, func(rows *sql.Rows) {
				var r row
				if err := rows.Scan(&r.id, &r.data); err == nil {
					batch = append(batch, r)
				}
			})
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("migrate 2.6.0 rows: %w", err)
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
					if _, err := txUpdate.Exec(r.id); err != nil {
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
		return nil
	}

	fixes, err := d.cubeHashFixes()
	if err != nil {
		return err
	}
	if len(fixes) == 0 {
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

	slog.Info("migration 2.7.0 rehashed cube positions", "positions_rehashed", len(fixes))
	return nil
}

// hashFix is a position whose zobrist_hash the 2.7.0 step rewrites.
type hashFix struct {
	id      int64
	newHash int64
}

// cubeHashFixes recomputes the Zobrist hash of every position with
// cube_value >= 1 from its stored state and columns. Unlike the batch
// backfills, a row that cannot be scanned is a hard error here: a hash left
// stale would keep colliding.
func (d *Database) cubeHashFixes() ([]hashFix, error) {
	rows, err := d.db.Query(`SELECT id, state, decision_type, player_on_roll, dice_1, dice_2, cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver FROM position WHERE cube_value >= 1`)
	if err != nil {
		return nil, fmt.Errorf("migrate 2.7.0 query positions: %w", err)
	}
	defer rows.Close()

	var fixes []hashFix
	for rows.Next() {
		var id int64
		var state string
		var decisionType, playerOnRoll, dice1, dice2, cubeValue, cubeOwner, score1, score2, hasJacoby, hasBeaver int
		if err := rows.Scan(&id, &state, &decisionType, &playerOnRoll, &dice1, &dice2, &cubeValue, &cubeOwner, &score1, &score2, &hasJacoby, &hasBeaver); err != nil {
			return nil, fmt.Errorf("migrate 2.7.0 scan: %w", err)
		}
		pos := reconstructPosition(id, state, decisionType, playerOnRoll, dice1, dice2, cubeValue, cubeOwner, score1, score2, hasJacoby, hasBeaver)
		newHash := engine.ZobristHash(&pos)
		fixes = append(fixes, hashFix{id: id, newHash: int64(newHash)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate 2.7.0 rows positions: %w", err)
	}
	return fixes, nil
}

// migrate_2_7_0_to_2_8_0 adds an exclude_position column to search_history and
// filter_library so the "Sauf" (exclusion structure) of a search can be persisted
// and restored on replay. The column is nullable; existing rows keep NULL (no
// exclusion structure).
func (d *Database) migrate_2_7_0_to_2_8_0(_ context.Context) error {
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

	return nil
}

// migrate_2_8_0_to_2_9_0 adds a scope column to command_history, search_history
// and filter_library. SQLite previously ignored the scope argument on these
// stores (the GUI/CLI always uses the empty scope); the column lets a
// multi-tenant SQLite-server isolate each tenant's command/search history and
// saved filters, mirroring the tenant_id scoping the PostgreSQL backend already
// has. Existing rows default to the empty scope so GUI/CLI data is unchanged.
func (d *Database) migrate_2_8_0_to_2_9_0(_ context.Context) error {
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

	return nil
}

// migrate_2_9_0_to_2_10_0 adds the is_cube_response column to position and
// backfills it: a cube position (decision_type = 1) is flagged 1 when any of its
// recorded played cube actions is a take/pass response (engine.IsResponseCubeAction),
// as opposed to a doubling decision (Double / No Double / Redouble). This lets the
// search filter distinguish "double/no-double" from "take/pass" cube decisions.
func (d *Database) migrate_2_9_0_to_2_10_0(ctx context.Context) error {
	if err := d.addColumn("position", "is_cube_response INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("migrate 2.10.0 add column: %w", err)
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
		// A Stmt bound to the transaction is a resource of its own; one per
		// batch loop, closed with the loop, not one per row.
		txUpdate := tx.Stmt(updateStmt)
		defer txUpdate.Close()

		const batchSize = 1000
		var lastID int64
		done := 0

		for {
			if err := ctx.Err(); err != nil {
				tx.Rollback()
				return err
			}

			var batch []int64
			err := d.forEachRow(`
				SELECT id FROM position
				WHERE decision_type = 1 AND id > ?
				ORDER BY id LIMIT ?`, []any{lastID, batchSize}, func(rows *sql.Rows) {
				var id int64
				if err := rows.Scan(&id); err == nil {
					batch = append(batch, id)
				}
			})
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("migrate 2.10.0 rows: %w", err)
			}

			if len(batch) == 0 {
				break
			}

			for _, id := range batch {
				lastID = id

				isResp, err := playedCubeResponse(lookupActions, id)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("migrate 2.10.0 actions rows: %w", err)
				}

				if isResp {
					if _, err := txUpdate.Exec(id); err != nil {
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

	return nil
}

// playedCubeResponse reports whether ANY of the played cube actions recorded
// for a position is a take/pass (OR semantics across matches for a deduped
// position). lookupActions selects the non-empty cube actions of a position
// id; a lookup that fails to run counts as no response, an iteration error is
// returned.
func playedCubeResponse(lookupActions *sql.Stmt, positionID int64) (bool, error) {
	actRows, err := lookupActions.Query(positionID)
	if err != nil {
		return false, nil
	}
	defer actRows.Close()
	isResp := false
	for actRows.Next() {
		var ca string
		if err := actRows.Scan(&ca); err == nil && engine.IsResponseCubeAction(ca) {
			isResp = true
		}
	}
	return isResp, actRows.Err()
}

// migrate_2_10_0_to_2_11_0 adds the anki_review_log table, an append-only
// journal of every spaced-repetition review (rating + FSRS outcome). It powers
// retention/streak statistics, the review heatmap and a faithful undo of the
// last review. The table is also (re)created by ensureAllTablesExist, so a
// missing table here is not fatal; this step exists so the chain records the
// version bump on an existing user database.
func (d *Database) migrate_2_10_0_to_2_11_0(_ context.Context) error {
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

	return nil
}

// migrate_2_11_0_to_2_12_0 extends anki_card with suspend/bury state. A
// suspended card is excluded from review indefinitely; a buried card is hidden
// until buried_until passes (typically the next day). The columns are also
// (re)added by ensureAllTablesExist, so a column that already exists here is
// not fatal; this step exists so the chain records the version bump on an
// existing user database.
func (d *Database) migrate_2_11_0_to_2_12_0(_ context.Context) error {
	for _, stmt := range []string{
		`ALTER TABLE anki_card ADD COLUMN suspended INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE anki_card ADD COLUMN buried_until DATETIME`,
	} {
		_, _ = d.db.Exec(stmt) // ignore error: column may already exist
	}

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
func (d *Database) migrate_2_12_0_to_2_13_0(_ context.Context) error {
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

	return nil
}

// migrate_2_13_0_to_2_14_0 adds position.flagged (docs/adr/0006): the mark the
// user put on a position in the tool the match came from — today only eXtreme
// Gammon, which records it per move.
//
// There is deliberately NO backfill. Unlike individually_imported, which could
// be reconstructed from the move graph, nothing in an existing database records
// a source-file flag: the information only exists in the .xg files themselves.
// Existing positions therefore start unflagged and gain the mark when their
// match is imported again — which is why ingest applies flags even to an exact
// duplicate that is otherwise skipped.
func (d *Database) migrate_2_13_0_to_2_14_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE position ADD COLUMN flagged INTEGER NOT NULL DEFAULT 0`) // may already exist

	if _, err := d.db.Exec(
		`CREATE INDEX IF NOT EXISTS idx_position_flagged ON position(flagged) WHERE flagged = 1`); err != nil {
		return fmt.Errorf("migrate 2.14.0 create index: %w", err)
	}

	return nil
}

// migrate_2_14_0_to_2_15_0 adds move.luck_mp (docs/adr/0010): the luck of a
// roll, in signed millipoints of equity, as the analysing tool computed it.
//
// The column is NULLable and there is deliberately NO backfill or default:
// zero is a real value (a neutral roll), so an existing row must read "unknown"
// rather than "neutral". blunderDB cannot recompute luck either — it has no
// evaluation engine — and unlike the denormalised analysis columns repaired in
// #115, nothing on disk holds it: the stored analysis JSON never carried luck.
// Existing rolls therefore stay unknown until their match is imported again
// from the source file.
//
// It lands on move rather than position because a Position is deduplicated
// across matches while the luck of a roll belongs to one occurrence of it.
func (d *Database) migrate_2_14_0_to_2_15_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE move ADD COLUMN luck_mp INTEGER`) // may already exist

	return nil
}

// migrate_2_15_0_to_2_16_0 adds anki_deck.session_limit (ADR-0026 rule 2).
//
// Nullable with no default, so every existing deck comes out of the migration
// with NO limit and behaves exactly as before: a setting that is introduced
// must never change how existing data behaves.
func (d *Database) migrate_2_15_0_to_2_16_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE anki_deck ADD COLUMN session_limit INTEGER`) // may already exist

	return nil
}

// migrate_2_16_0_to_2_17_0 moves the UI session state out of metadata and
// into its own table, session_state(scope, key, value) (issue #156).
//
// Until 2.16.0 the session (last search, last position, open views) was six
// metadata rows — `session_*` for the desktop's empty scope, `<scope>:session_*`
// for a tenant of a multi-tenant SQLite daemon. metadata is database
// infrastructure (schema version, issuance) and is global by design; keeping a
// tenant's rows there meant the daemon's metadata.load handed every tenant
// the session of every other one, so the route is gone and the rows move to a
// table that carries the scope as a column, the way command_history,
// search_history and filter_library already do.
//
// The move is a copy-then-delete inside one transaction, re-runnable: the
// table is created IF NOT EXISTS and the copy is an INSERT OR REPLACE. The
// desktop's rows keep the empty scope, so a database migrated on the desktop
// reopens on the same search and the same views.
func (d *Database) migrate_2_16_0_to_2_17_0(_ context.Context) error {
	// The six keys as 2.16.0's sqlshared.SessionStore wrote them. Listed here
	// rather than imported: a migration describes the past, and matching the
	// exact keys (rather than a `session_%` pattern) leaves any other row of
	// metadata alone.
	sessionKeys := []string{
		"session_last_search_command",
		"session_last_search_position",
		"session_last_position_index",
		"session_last_position_ids",
		"session_has_active_search",
		"session_views",
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("migrate 2.17.0: begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS session_state (
		scope TEXT NOT NULL DEFAULT '',
		key   TEXT NOT NULL,
		value TEXT,
		PRIMARY KEY (scope, key)
	)`); err != nil {
		return fmt.Errorf("migrate 2.17.0: create session_state: %w", err)
	}

	for _, key := range sessionKeys {
		// Unprefixed: the desktop's (empty) scope.
		if _, err := tx.Exec(`INSERT OR REPLACE INTO session_state (scope, key, value)
			SELECT '', key, value FROM metadata WHERE key = ?`, key); err != nil {
			return fmt.Errorf("migrate 2.17.0: move %s: %w", key, err)
		}
		// Prefixed "<scope>:<key>": the scope is everything before the suffix,
		// whatever characters it holds.
		suffix := ":" + key
		if _, err := tx.Exec(`INSERT OR REPLACE INTO session_state (scope, key, value)
			SELECT substr(key, 1, length(key) - ?), ?, value FROM metadata
			WHERE length(key) > ? AND substr(key, -?) = ?`,
			len(suffix), key, len(suffix), len(suffix), suffix); err != nil {
			return fmt.Errorf("migrate 2.17.0: move scoped %s: %w", key, err)
		}
		if _, err := tx.Exec(`DELETE FROM metadata WHERE key = ? OR (length(key) > ? AND substr(key, -?) = ?)`,
			key, len(suffix), len(suffix), suffix); err != nil {
			return fmt.Errorf("migrate 2.17.0: delete %s from metadata: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate 2.17.0: commit: %w", err)
	}
	return nil
}

// migrate_2_17_0_to_2_18_0 is the 2.18.0 wave: one schema version for the
// three repairs of lot B that all needed a bump, applied in one open rather
// than in three successive upgrades of the same file.
//
//   - retireRuleFlagsFromZobrist — Jacoby and beaver leave the position
//     identity (ADR-0028, issue #171).
//   - enforceOneAnalysisPerPosition — analysis(position_id) becomes UNIQUE
//     (issue #173).
func (d *Database) migrate_2_17_0_to_2_18_0(ctx context.Context) error {
	if err := d.retireRuleFlagsFromZobrist(ctx); err != nil {
		return err
	}
	return d.enforceOneAnalysisPerPosition(ctx)
}

// enforceOneAnalysisPerPosition prepares the UNIQUE index on
// analysis(position_id) the fresh schema now declares (issue #173).
//
// analysisStore.Save used to SELECT an existing row and then INSERT or UPDATE.
// Two saves racing on the same position both read "no row" and both inserted;
// Load then read `SELECT data FROM analysis WHERE position_id = ?` and took
// whichever row the planner reached first, so a position could show an
// analysis that had been superseded — with no way to tell from the outside.
// Save is a single upsert now, which needs the index to exist: an ON CONFLICT
// target must name a UNIQUE constraint.
//
// The rows already there are deduplicated first, keeping the HIGHEST id per
// position — the last one written, which is the one Save meant to leave. Then
// the old non-unique index of the same name is dropped, and EnsureSchema (which
// runs right after the chain) builds the UNIQUE one in its place: an index is
// not retyped by `CREATE ... IF NOT EXISTS` under a name that already exists.
//
// Idempotent, unlike its sibling above: on a second pass there is nothing left
// to delete and no index left to drop.
func (d *Database) enforceOneAnalysisPerPosition(ctx context.Context) error {
	switch ok, err := d.columnExists("analysis", "position_id"); {
	case err != nil:
		return fmt.Errorf("migrate 2.18.0: %w", err)
	case !ok:
		return nil
	}

	res, err := d.db.ExecContext(ctx,
		`DELETE FROM analysis WHERE id NOT IN (SELECT MAX(id) FROM analysis GROUP BY position_id)`)
	if err != nil {
		return fmt.Errorf("migrate 2.18.0: deduplicate analyses: %w", err)
	}
	if dropped, err := res.RowsAffected(); err == nil && dropped > 0 {
		slog.Info("dropped superseded analysis rows before making analysis(position_id) unique",
			"dropped", dropped)
	}

	if _, err := d.db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_analysis_position`); err != nil {
		return fmt.Errorf("migrate 2.18.0: drop the non-unique analysis index: %w", err)
	}
	return nil
}

// retireRuleFlagsFromZobrist takes the Jacoby and beaver flags out of the
// position identity (ADR-0028, issue #171).
//
// Until 2.18.0 engine.ZobristHash folded has_jacoby and has_beaver into the
// hash. They are rules of the *session*, not facts of the board, and no import
// format but an XGID carries them — every file importer leaves both at 0. The
// same money position pasted from an XGID (Jacoby set) and imported from a .xg
// file (Jacoby clear) therefore hashed differently and landed on two rows, with
// the analyses, comments and collection memberships of one position split
// across both. That is invariant no. 1 broken; this step repairs the databases
// that lived through it.
//
// A Zobrist hash is a XOR of keys, so undoing a fold is XORing the same key
// back in (engine.RetiredFlagDelta): no board is decoded and no state is read,
// only the two flag columns the row already carries. The rows that carry
// neither flag — nearly all of them — keep the hash they were stored under and
// are not touched at all.
//
// Rehashing can bring two rows onto one hash: those two rows were always the
// same position and are merged, the OLDER row (lowest id) kept, exactly as
// mergePositionInto folds the duplicates repairPositionsWithoutScalars finds.
// The candidates release their hash first (set to NULL, which the UNIQUE index
// tolerates any number of) and take their new one in id order, so a collision
// is always judged against the hash the other row will END with, never against
// the one it is about to leave.
//
// One transaction, and — uniquely in this chain — a step that is NOT
// idempotent: XOR is its own inverse, so running it twice would put the flag
// keys back. What makes that safe is atomicity, not re-runnability. The
// conversion either commits whole or rolls back whole, and runMigrationChain
// stamps 2.18.0 only after it returns nil; an interrupted upgrade therefore
// leaves a file that is still entirely at 2.17.0 and is replayed from the
// start. Nothing here reads the file to decide whether it applies — the
// recorded version alone says so, as every step in this chain does.
func (d *Database) retireRuleFlagsFromZobrist(ctx context.Context) error {
	// A real 2.17.0 file has all three columns; the migration-chain fixtures
	// stamp a 2.x version onto a 1.x table and leave the scalar columns to
	// ensureAllTablesExist, which runs after the chain. Nothing to convert
	// where there is no hash to convert.
	for _, column := range []string{"zobrist_hash", "has_jacoby", "has_beaver"} {
		switch ok, err := d.columnExists("position", column); {
		case err != nil:
			return fmt.Errorf("migrate 2.18.0: %w", err)
		case !ok:
			return nil
		}
	}

	type candidate struct {
		id      int64
		newHash int64
	}

	var todo []candidate
	if err := func() error {
		rows, err := d.db.QueryContext(ctx, `SELECT id, zobrist_hash,
				COALESCE(has_jacoby, 0), COALESCE(has_beaver, 0)
			FROM position
			WHERE zobrist_hash IS NOT NULL AND (COALESCE(has_jacoby, 0) <> 0 OR COALESCE(has_beaver, 0) <> 0)
			ORDER BY id`)
		if err != nil {
			return fmt.Errorf("migrate 2.18.0: list positions carrying a rule flag: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id, hash int64
			var jacoby, beaver int
			if err := rows.Scan(&id, &hash, &jacoby, &beaver); err != nil {
				return fmt.Errorf("migrate 2.18.0: scan position: %w", err)
			}
			// int64 ↔ uint64 is a reinterpretation of the same 64 bits: the
			// hash is stored in SQLite's signed INTEGER and XOR does not care.
			todo = append(todo, candidate{id, int64(uint64(hash) ^ engine.RetiredFlagDelta(jacoby, beaver))})
		}
		return rows.Err()
	}(); err != nil {
		return err
	}
	if len(todo) == 0 {
		return nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate 2.18.0: begin: %w", err)
	}
	defer tx.Rollback()

	// Release every candidate's hash before any is reassigned. A UNIQUE index
	// holds as many NULLs as it likes, so the table now states, for every row,
	// the hash it will end with — or nothing at all.
	for _, c := range todo {
		if _, err := tx.ExecContext(ctx, `UPDATE position SET zobrist_hash = NULL WHERE id = ?`, c.id); err != nil {
			return fmt.Errorf("migrate 2.18.0: release hash of position %d: %w", c.id, err)
		}
	}

	var rehashed, merged int
	for i, c := range todo {
		if err := ctx.Err(); err != nil {
			return err
		}
		var other int64
		switch err := tx.QueryRowContext(ctx,
			`SELECT id FROM position WHERE zobrist_hash = ?`, c.newHash).Scan(&other); {
		case errors.Is(err, sql.ErrNoRows):
			other = 0
		case err != nil:
			return fmt.Errorf("migrate 2.18.0: probe hash of position %d: %w", c.id, err)
		}
		switch {
		case other == 0:
			if _, err := tx.ExecContext(ctx,
				`UPDATE position SET zobrist_hash = ? WHERE id = ?`, c.newHash, c.id); err != nil {
				return fmt.Errorf("migrate 2.18.0: rehash position %d: %w", c.id, err)
			}
			rehashed++
		case other < c.id:
			// The row already holding this hash is the older one: it keeps it.
			if err := mergePositionInto(ctx, tx, other, c.id); err != nil {
				return err
			}
			merged++
		default:
			// The candidate is the older row: it takes the hash, and the row
			// that held it is folded in first so the index is free.
			if err := mergePositionInto(ctx, tx, c.id, other); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE position SET zobrist_hash = ? WHERE id = ?`, c.newHash, c.id); err != nil {
				return fmt.Errorf("migrate 2.18.0: rehash position %d: %w", c.id, err)
			}
			merged++
		}
		d.emitMigrationProgress("zobrist_rule_flags", i+1, len(todo))
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate 2.18.0: commit: %w", err)
	}
	slog.Info("Jacoby and beaver left the position identity",
		"rehashed", rehashed, "merged", merged)
	return nil
}

// migrate_2_18_0_to_2_19_0 is the 2.19.0 wave: the four schema changes the
// product lot asks for, applied in one open rather than in four successive
// upgrades of the same file (tasks/plan-amelioration-2026-09b, lot I).
//
//   - position.game_phase — the derived phase label (issue #264, ADR-0035).
//   - comment.origin — who wrote a comment (issue #263).
//   - import_batch + match.import_batch_id — the end-of-import report's
//     unit of account (issue #257).
//   - trash — the snapshot table undo restores from (issue #285, ADR-0036).
//
// The columns and tables themselves are added by EnsureSchema, which derives
// what is missing from schemaStatements and runs right AFTER the chain. So
// this step has nothing to create — and cannot do the one thing that does need
// writing, the game_phase backfill, because the column it would write does not
// exist yet. It records the request instead; runMigrationChain honours it once
// EnsureSchema has been through.
//
// The backfill cannot be an UPDATE ... SET game_phase = <expression> either:
// the classification reads the board out of the compact `state` encoding.
func (d *Database) migrate_2_18_0_to_2_19_0(context.Context) error {
	d.pendingPhaseBackfill = true
	return nil
}
