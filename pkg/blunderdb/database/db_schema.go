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

// ensureAllTablesExist brings an existing database up to the current schema
// on every open, after the migration chain has run. The schema itself —
// tables, columns, indexes — lives in one place, storage/sqlite's
// schemaStatements, and sqlite.EnsureSchema derives what this database is
// missing from it; nothing here names a column. What remains here is history:
// the data repairs and index drops that only a database which lived through
// an earlier version can need.
func (d *Database) ensureAllTablesExist() error {
	ctx := context.Background()

	// Import writes an empty canonical hash as NULL (matches_sqlite.go
	// nullableString); older imports stored '' and two of those would collide
	// under idx_match_canonical (UNIQUE), which EnsureSchema builds below.
	// Normalise first so the index can be built; on a database that genuinely
	// holds two matches with the same canonical hash it still cannot, and the
	// database keeps working unindexed (FindByHash is a plain SELECT). Before
	// 1.7.0 there is no match table yet, and the statement fails harmlessly.
	_, _ = d.db.ExecContext(ctx, `UPDATE match SET canonical_hash = NULL WHERE canonical_hash = ''`)

	// Indexes the fresh schema no longer declares (E3 redundancy pass and the
	// fiche-05 covering index): each is a strict column prefix of an index the
	// schema still has, so nothing plans differently without it. Dropping is
	// perf-only — no query names an index — hence no DatabaseVersion bump.
	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_position_score`,
		`DROP INDEX IF EXISTS idx_analysis_win_gammon`,
		`DROP INDEX IF EXISTS idx_analysis_win1`,
	} {
		_, _ = d.db.ExecContext(ctx, stmt)
	}

	if err := sqlite.EnsureSchema(ctx, d.db); err != nil {
		return err
	}

	// Collections created before their timestamps had defaults carry NULLs
	// that the GUI cannot sort by.
	_, _ = d.db.ExecContext(ctx, `UPDATE collection SET created_at = datetime('now') WHERE created_at IS NULL OR created_at = ''`)
	_, _ = d.db.ExecContext(ctx, `UPDATE collection SET updated_at = datetime('now') WHERE updated_at IS NULL OR updated_at = ''`)
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
	type defective struct {
		pos   Position
		state string
	}
	var todo []defective
	if err := func() error {
		rows, err := d.db.QueryContext(ctx, `SELECT `+positionSelectCols+`
			FROM position WHERE zobrist_hash IS NULL ORDER BY id`)
		if err != nil {
			return fmt.Errorf("listing positions without scalar columns: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var state string
			pos, err := scanPositionRowWithState(rows, &state)
			if err != nil {
				return fmt.Errorf("scanning position without scalar columns: %w", err)
			}
			todo = append(todo, defective{pos, state})
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
