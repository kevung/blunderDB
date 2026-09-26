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

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
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

// migrate_2_6_0_to_2_7_0 recomputes the Zobrist hash of every position with
// cube_value >= 1: those were hashed with Cube.Value (a log2 exponent) passed
// where cubeValueIndex expects the actual cube value, giving wrong and
// sometimes colliding hashes. Exponent 0 was unaffected. Idempotent.
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

// migrate_2_7_0_to_2_8_0 adds a nullable exclude_position column to
// search_history and filter_library, persisting a search's "Sauf" structure.
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
// and filter_library so a multi-tenant SQLite daemon isolates each tenant, as
// tenant_id does on PostgreSQL. Existing rows get the empty scope the GUI/CLI use.
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

// migrate_2_9_0_to_2_10_0 adds position.is_cube_response and backfills it: a
// cube position is flagged when any played cube action is a take/pass
// (engine.IsResponseCubeAction) rather than a doubling decision.
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

// playedCubeResponse reports whether ANY played cube action of a position is a
// take/pass (OR across matches of a deduped position). A lookup that fails to
// run counts as no response; an iteration error is returned.
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

// migrate_2_10_0_to_2_11_0 adds anki_review_log, the append-only journal of
// every review (rating + FSRS outcome). ensureAllTablesExist also creates it;
// this step keeps the chain continuous.
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

// migrate_2_11_0_to_2_12_0 adds anki_card's suspend/bury state (a buried card
// is hidden until buried_until). ensureAllTablesExist also adds the columns, so
// one that already exists is not fatal.
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
// The backfill is a one-shot reconstruction with two accepted error classes:
// false positives (an "enrich" import creates no move row, so its positions
// look individual) and false negatives (an individual import that preceded a
// match holding the same position is unrecoverable). From here on the flag is
// written at import time and is exact.
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

// migrate_2_13_0_to_2_14_0 adds position.flagged (ADR-0006): the mark set in
// the source tool (only eXtreme Gammon records one). No backfill: the flag
// exists only in the .xg files, so positions gain it on re-import — which is
// why ingest applies flags even to an otherwise skipped exact duplicate.
func (d *Database) migrate_2_13_0_to_2_14_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE position ADD COLUMN flagged INTEGER NOT NULL DEFAULT 0`) // may already exist

	if _, err := d.db.Exec(
		`CREATE INDEX IF NOT EXISTS idx_position_flagged ON position(flagged) WHERE flagged = 1`); err != nil {
		return fmt.Errorf("migrate 2.14.0 create index: %w", err)
	}

	return nil
}

// migrate_2_14_0_to_2_15_0 adds move.luck_mp (ADR-0010): a roll's luck in
// signed millipoints, as the analysing tool computed it. NULLable, no default,
// no backfill: zero means a neutral roll, so old rows must read "unknown", and
// nothing on disk holds the value. It lives on move because a Position is
// deduplicated while luck belongs to one occurrence of the roll.
func (d *Database) migrate_2_14_0_to_2_15_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE move ADD COLUMN luck_mp INTEGER`) // may already exist

	return nil
}

// migrate_2_15_0_to_2_16_0 adds anki_deck.session_limit (ADR-0026 rule 2),
// nullable with no default so existing decks keep behaving as before.
func (d *Database) migrate_2_15_0_to_2_16_0(_ context.Context) error {
	_, _ = d.db.Exec(`ALTER TABLE anki_deck ADD COLUMN session_limit INTEGER`) // may already exist

	return nil
}

// migrate_2_16_0_to_2_17_0 moves the UI session state from six metadata rows
// (`session_*`, `<scope>:session_*`) into session_state(scope, key, value).
// metadata is global by design, so a daemon reading it leaked every tenant's
// session to the others. Copy-then-delete in one transaction, re-runnable; the
// desktop's rows keep the empty scope.
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

// migrate_2_17_0_to_2_18_0 retires Jacoby/beaver from the position identity
// (ADR-0028) and makes analysis(position_id) UNIQUE.
func (d *Database) migrate_2_17_0_to_2_18_0(ctx context.Context) error {
	if err := d.retireRuleFlagsFromZobrist(ctx); err != nil {
		return err
	}
	return d.enforceOneAnalysisPerPosition(ctx)
}

// enforceOneAnalysisPerPosition prepares the UNIQUE index on
// analysis(position_id) that Save's upsert needs as its ON CONFLICT target;
// racing saves could otherwise leave two rows and Load read either. Duplicates
// are removed keeping the HIGHEST id (the last written), then the old
// non-unique index is dropped so EnsureSchema can recreate it UNIQUE —
// `CREATE ... IF NOT EXISTS` never retypes an existing name. Idempotent.
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

// retireRuleFlagsFromZobrist takes has_jacoby/has_beaver out of the position
// identity (ADR-0028): session rules that only an XGID sets split one position
// across two rows.
//
// Undoing a XOR fold is XORing the key back (engine.RetiredFlagDelta), from the
// flag columns alone; rows with neither flag are untouched. Rows that collide
// after rehashing were always one position and are merged, keeping the lowest
// id (as mergePositionInto does). Candidates first release their hash (NULL)
// and take the new one in id order, so a collision is judged against the other
// row's FINAL hash.
//
// NOT idempotent (XOR is its own inverse): safety comes from atomicity. One
// transaction, and runMigrationChain stamps 2.18.0 only on success, so an
// interrupted upgrade is replayed from 2.17.0.
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

// migrate_2_18_0_to_2_19_0 covers position.game_phase (ADR-0035),
// comment.origin, import_batch + match.import_batch_id, and trash (ADR-0036).
// EnsureSchema creates them AFTER the chain, so the game_phase backfill (which
// must decode the compact `state`, not a SQL expression) is only requested
// here; runMigrationChain runs it after EnsureSchema.
func (d *Database) migrate_2_18_0_to_2_19_0(context.Context) error {
	d.pendingPhaseBackfill = true
	return nil
}

// migrate_2_19_0_to_2_20_0 covers position.max_cube, the cube ceiling as the
// log2 exponent of the XGID's tenth field. EnsureSchema adds the column; the
// step keeps the chain continuous (TestMigrationSteps_ContinuousChain). No
// backfill: 0 means "no ceiling stated", true of every older row.
func (d *Database) migrate_2_19_0_to_2_20_0(context.Context) error {
	return nil
}

// migrate_2_20_0_to_2_21_0 covers the transcription table (ADR-0045): an
// opaque JSON draft with its OWN format_version. EnsureSchema creates it; the
// step keeps the chain continuous. Nothing to backfill: a draft is not
// derivable from a saved match (ADR-0045 §2).
func (d *Database) migrate_2_20_0_to_2_21_0(context.Context) error {
	return nil
}

// migrate_2_21_0_to_2_22_0 covers training_session / training_item, the
// Training journal (ADR-0040 rule 6). EnsureSchema creates them; the step keeps
// the chain continuous. The old per-session summary in `metadata` is not
// imported: it lacks the per-number detail the journal exists for.
func (d *Database) migrate_2_21_0_to_2_22_0(context.Context) error {
	return nil
}

// migrate_2_22_0_to_2_23_0 covers anki_card/anki_review_log kind + key
// (ADR-0042): a `position` card keyed by id, or a `score` card keyed by the
// unordered score ("3:5") with a NULL position_id. EnsureSchema adds the
// columns; filling old keys and relaxing position_id's NOT NULL (which ALTER
// TABLE cannot do) are deferred to repairAnkiCardKinds after the schema pass.
func (d *Database) migrate_2_22_0_to_2_23_0(context.Context) error {
	d.pendingAnkiCardKinds = true
	return nil
}

// repairAnkiCardKinds brings the two anki tables of an existing database into
// the 2.23.0 shape: every card that was there before is a `position` card with
// its id as key (nothing old changes meaning, ADR-0042), and position_id
// becomes nullable so a score card can hold nothing there rather than a 0
// pointing at no row.
//
// Idempotent: only empty keys are filled, and a table is rebuilt only while
// its position_id is still NOT NULL.
func (d *Database) repairAnkiCardKinds(ctx context.Context) error {
	for _, table := range []string{"anki_card", "anki_review_log"} {
		if _, err := d.db.ExecContext(ctx,
			`UPDATE `+table+` SET kind = ?, key = CAST(position_id AS TEXT)
			 WHERE key = '' AND position_id IS NOT NULL`, domain.AnkiKindPosition); err != nil {
			return fmt.Errorf("%s: backfilling the key: %w", table, err)
		}
		notNull, err := columnIsNotNull(ctx, d.db, table, "position_id")
		if err != nil {
			return err
		}
		if !notNull {
			continue
		}
		if err := rebuildTable(ctx, d.db, table); err != nil {
			return err
		}
	}
	return nil
}

// columnIsNotNull reports whether a column of an existing table carries NOT
// NULL, read off PRAGMA table_info.
func columnIsNotNull(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return false, fmt.Errorf("%s: reading the columns: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, typ        string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return false, fmt.Errorf("%s: reading the columns: %w", table, err)
		}
		if name == column {
			return notNull != 0, nil
		}
	}
	return false, rows.Err()
}

// rebuildTable replaces a table with the one schemaStatements declares today,
// carrying every column the two shapes share across. It is SQLite's documented
// twelve-step ALTER (create, copy, drop, rename), and it is the only way to
// relax a constraint on a table that already exists.
//
// Foreign keys are OFF for the duration, on a PINNED connection (a PRAGMA
// reaches one pooled connection only): dropping the old table with them on
// would cascade its children away. The copy names its columns, so a newer
// column takes its default instead of shifting the copy. Affordable on the
// small anki tables, not on `position` (see schemaStatements).
func rebuildTable(ctx context.Context, db *sql.DB, table string) error {
	createSQL, err := sqlite.CreateTableSQL(table)
	if err != nil {
		return err
	}
	tmp := table + "_rebuild"
	createTmp := strings.Replace(createSQL, "CREATE TABLE IF NOT EXISTS "+table+" (",
		"CREATE TABLE "+tmp+" (", 1)

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("%s: rebuilding: %w", table, err)
	}
	defer conn.Close()
	// Outside any transaction, and undone before the connection goes back to
	// the pool — see the doc comment.
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("%s: rebuilding: %w", table, err)
	}
	defer func() { _, _ = conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`) }()

	if _, err := conn.ExecContext(ctx, `DROP TABLE IF EXISTS `+tmp); err != nil {
		return fmt.Errorf("%s: rebuilding: %w", table, err)
	}
	if _, err := conn.ExecContext(ctx, createTmp); err != nil {
		return fmt.Errorf("%s: rebuilding: %w", table, err)
	}
	shared, err := sharedColumns(ctx, conn, table, tmp)
	if err != nil {
		return err
	}
	cols := strings.Join(shared, ", ")

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: rebuilding: %w", table, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO `+tmp+` (`+cols+`) SELECT `+cols+` FROM `+table); err != nil {
		return fmt.Errorf("%s: rebuilding, copying the rows: %w", table, err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE `+table); err != nil {
		return fmt.Errorf("%s: rebuilding, dropping the old table: %w", table, err)
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE `+tmp+` RENAME TO `+table); err != nil {
		return fmt.Errorf("%s: rebuilding, renaming: %w", table, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: rebuilding: %w", table, err)
	}
	slog.Info("anki table rebuilt so a card can be something other than a position", "table", table)
	return nil
}

// sharedColumns lists the columns two tables have in common, in the order the
// destination declares them.
func sharedColumns(ctx context.Context, conn *sql.Conn, src, dst string) ([]string, error) {
	names := func(table string) (map[string]bool, []string, error) {
		rows, err := conn.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: reading the columns: %w", table, err)
		}
		defer rows.Close()
		set := map[string]bool{}
		var order []string
		for rows.Next() {
			var (
				cid, notNull, pk int
				name, typ        string
				dflt             sql.NullString
			)
			if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
				return nil, nil, fmt.Errorf("%s: reading the columns: %w", table, err)
			}
			set[name] = true
			order = append(order, name)
		}
		return set, order, rows.Err()
	}
	srcSet, _, err := names(src)
	if err != nil {
		return nil, err
	}
	_, dstOrder, err := names(dst)
	if err != nil {
		return nil, err
	}
	var shared []string
	for _, c := range dstOrder {
		if srcSet[c] {
			shared = append(shared, c)
		}
	}
	return shared, nil
}

// migrate_2_23_0_to_2_24_0 covers direction / direction_event (ADR-0047: the
// director's decisions, append-only, derived state never stored) and
// match.direction_match_id, the Slot a Match fills. EnsureSchema creates them;
// the step keeps the chain continuous. No backfill is possible: an imported
// Tournament was never directed, and attaching a match is always deliberate.
func (d *Database) migrate_2_23_0_to_2_24_0(context.Context) error {
	return nil
}
