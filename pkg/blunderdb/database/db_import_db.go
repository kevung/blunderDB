package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// ErrImportCancelled is returned by CommitImportDatabase when the user cancels
// via CancelImport — an outcome asked for, not a failure. The returned error
// also wraps the context's own, so errors.Is matches either.
var ErrImportCancelled = errors.New("import cancelled by user")

// wrapImportCancelled builds the error CommitImportDatabase returns once
// ctx.Err() is non-nil, wrapping both errors. Separate so it is testable
// without racing a real CancelImport against a commit.
func wrapImportCancelled(ctxErr error) error {
	return fmt.Errorf("%w: %w", ErrImportCancelled, ctxErr)
}

// sourcePositionScalarColumns is the SELECT-list fragment for the scalar
// columns a compact-state row needs (decision type, dice, cube, score,
// jacoby, beaver), qualified with alias (e.g. "p." or ""), selected alongside
// id/state so decodeSourcePosition needs no query per compact row. A pre-2.2.0
// database holds no compact row, so its NULL stand-ins are never read; they
// only keep the SELECT valid.
func sourcePositionScalarColumns(importDB *sql.DB, alias string) string {
	if !queryable(importDB, `SELECT decision_type FROM position LIMIT 1`) {
		return "NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL"
	}
	cols := []string{"decision_type", "player_on_roll", "dice_1", "dice_2",
		"cube_value", "cube_owner", "score_1", "score_2", "has_jacoby", "has_beaver"}
	qualified := make([]string, len(cols))
	for i, c := range cols {
		qualified[i] = alias + c
	}
	return strings.Join(qualified, ", ")
}

// sourcePositionQuery selects (id, state, the ten scalar columns
// sourcePositionScalarColumns lists, individually_imported) from the
// database being imported.
//
// Before 2.13.0 the flag is derived exactly as the 2.12.0→2.13.0 migration
// derives it (no move reaches the position), so migrating in place and
// importing elsewhere agree on provenance.
func sourcePositionQuery(importDB *sql.DB) string {
	if queryable(importDB, `SELECT individually_imported FROM position LIMIT 1`) {
		return `SELECT id, state, ` + sourcePositionScalarColumns(importDB, "") + `, individually_imported FROM position`
	}
	if queryable(importDB, `SELECT 1 FROM move LIMIT 1`) {
		slog.Debug("import database predates individually_imported; deriving it from the move graph")
		return `SELECT p.id, p.state, ` + sourcePositionScalarColumns(importDB, "p.") + `,
			       NOT EXISTS (SELECT 1 FROM move m WHERE m.position_id = p.id)
			FROM position p`
	}
	// No move table at all: the database holds no matches, so every position in
	// it stands on its own. This is the same rule as the derivation above, taken
	// to its limit.
	slog.Debug("import database has no move table; all its positions are individually imported")
	return `SELECT id, state, ` + sourcePositionScalarColumns(importDB, "") + `, 1 FROM position`
}

// queryable reports whether q runs against db — used to probe for a column or a
// table that older schema versions may not have.
func queryable(db *sql.DB, q string) bool {
	var dummy int
	err := db.QueryRow(q).Scan(&dummy)
	return err == nil || err == sql.ErrNoRows
}

// checkImportableVersion allows a database to be imported into another when
// its schema major version is the same or older: the importer reads the source
// with the current code, which knows every past major and none of the future
// ones. Compared numerically (parseVersion): as strings, "10" sorts before "2".
func checkImportableVersion(importVersion, currentVersion string) error {
	iv, err := parseVersion(importVersion)
	if err != nil {
		return fmt.Errorf("import database version %q: %w", importVersion, err)
	}
	cv, err := parseVersion(currentVersion)
	if err != nil {
		return fmt.Errorf("current database version %q: %w", currentVersion, err)
	}
	if iv[0] > cv[0] {
		return fmt.Errorf("cannot import from a newer major database version (import: %s, current: %s)", importVersion, currentVersion)
	}
	return nil
}

// decodeSourcePosition reconstructs a Position from a source row. Full-JSON
// state is self-describing; compact state holds only the board, so dice,
// score, cube and decision type come from the row's scalar columns (dt … hb,
// see sourcePositionScalarColumns). Zeroing them would make
// positionIdentityJSON miss the existing row and duplicate every position.
// A corrupt row with compact state but NULL columns degrades to a board-only
// Position rather than aborting the import.
func decodeSourcePosition(state string, dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64) (Position, error) {
	var pos Position
	if !isCompactState(state) {
		if err := json.Unmarshal([]byte(state), &pos); err != nil {
			return Position{}, err
		}
		return pos, nil
	}
	pos.Board = decodeBoardCompact(state)
	pos.DecisionType = int(dt.Int64)
	pos.PlayerOnRoll = int(por.Int64)
	pos.Dice = [2]int{int(d1.Int64), int(d2.Int64)}
	pos.Cube = Cube{Owner: int(co.Int64), Value: int(cv.Int64)}
	pos.Score = [2]int{int(s1.Int64), int(s2.Int64)}
	pos.HasJacoby = int(hj.Int64)
	pos.HasBeaver = int(hb.Int64)
	return pos, nil
}

// queryer is satisfied by both *sql.DB (the import source, and
// AnalyzeImportDatabase's read-only current database) and *sql.Tx
// (CommitImportDatabase's write transaction) — loadJoinedCommentText runs
// against either.
type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// loadJoinedCommentText returns a position's full comment text: every row, in
// a stable order, joined as storage/sqlshared's loadCommentText joins them. A
// position can carry several comment rows; comparing one arbitrary row
// misjudges what is "already present".
func loadJoinedCommentText(db queryer, positionID int64) (string, error) {
	rows, err := db.Query(`SELECT text FROM comment WHERE position_id = ? AND text != '' ORDER BY id ASC`, positionID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return "", err
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return strings.Join(parts, "\n\n"), nil
}

// importedComment is one comment row of a database being imported.
type importedComment struct{ text, author string }

// sourceCommentsSigned reports whether the database being imported records who
// wrote each comment; one written before comments were signed does not.
func sourceCommentsSigned(db *sql.DB) bool {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('comment') WHERE name = 'author'`).Scan(&n)
	return err == nil && n > 0
}

// loadImportedComments returns a source position's non-empty comment rows in
// id order, each with the author the source records: an import copies the
// producer's signature and never signs in the importer's name (ADR-0007).
func loadImportedComments(db queryer, positionID int64, signed bool) ([]importedComment, error) {
	author := `''`
	if signed {
		author = `COALESCE(author, '')`
	}
	rows, err := db.Query(`SELECT text, `+author+` FROM comment WHERE position_id = ? AND text != '' ORDER BY id ASC`, positionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []importedComment
	for rows.Next() {
		var c importedComment
		if err := rows.Scan(&c.text, &c.author); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AnalyzeImportDatabase analyzes what would be imported without making changes
func (d *Database) AnalyzeImportDatabase(importPath string) (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check that the current database is open
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	// Open the import database
	importDB, err := openExistingSQLite(importPath)
	if err != nil {
		return nil, err
	}
	defer importDB.Close()

	// Check the import database version
	var importDBVersion string
	err = importDB.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&importDBVersion)
	if err != nil {
		return nil, fmt.Errorf("import database is invalid or missing version information")
	}

	// Check the current database version
	var currentDBVersion string
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&currentDBVersion)
	if err != nil {
		return nil, err
	}

	if err := checkImportableVersion(importDBVersion, currentDBVersion); err != nil {
		return nil, err
	}

	// Count total positions to import
	var totalPositions int
	err = importDB.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&totalPositions)
	if err != nil {
		return nil, err
	}

	// OPTIMIZATION: Build a hash map of all current positions ONCE
	// This converts O(n²) to O(n) complexity
	currentPositionsMap, err := positionIdentityIndex(d.db)
	if err != nil {
		return nil, err
	}

	slog.Debug("built position index", "count", len(currentPositionsMap))

	// Analyze what would happen. Unlike sourcePositionQuery, no
	// individually_imported: this pass does not need it.
	rows, err := importDB.Query(`SELECT id, state, ` + sourcePositionScalarColumns(importDB, "") + ` FROM position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positionsToAdd int
	var positionsToMerge int
	var positionsToSkip int

	for rows.Next() {
		var id int64
		var stateJSON string
		var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
		if err = rows.Scan(&id, &stateJSON, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb); err != nil {
			slog.Warn("scanning position", "err", err)
			positionsToSkip++
			continue
		}

		importPosition, decErr := decodeSourcePosition(stateJSON, dt, por, d1, d2, cv, co, s1, s2, hj, hb)
		if decErr != nil {
			slog.Warn("unmarshalling position", "err", decErr)
			positionsToSkip++
			continue
		}

		// Assigned, not declared: the rollback defer reads this very err.
		var importPositionJSON string
		if importPositionJSON, err = positionIdentityJSON(importPosition); err != nil {
			return nil, err
		}

		// OPTIMIZATION: O(1) hash map lookup instead of nested loop
		existingPositionID, existsInCurrent := currentPositionsMap[importPositionJSON]
		if !existsInCurrent {
			existingPositionID, existsInCurrent, err = heldByZobrist(d.store.Positions(), &importPosition)
			if err != nil {
				return nil, err
			}
		}

		if existsInCurrent {
			// Check if there's actually something to merge
			hasNewData := false

			// Check for analysis to merge
			var importAnalysisData []byte
			err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisData)
			if err == nil {
				var existingAnalysisData []byte
				existingErr := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, existingPositionID).Scan(&existingAnalysisData)

				if existingErr == sql.ErrNoRows {
					// New analysis to add
					hasNewData = true
				} else if existingErr == nil {
					existingAnalysis, _ := decodeAnalysisFromStorage(existingAnalysisData)
					importAnalysis, _ := decodeAnalysisFromStorage(importAnalysisData)
					if _, changed := domain.MergeImportedAnalysis(&existingAnalysis, &importAnalysis); changed {
						hasNewData = true
					}
				}
			}

			// Check for comments to merge, joining every row on both sides.
			importComment, err := loadJoinedCommentText(importDB, id)
			if err != nil {
				return nil, err
			}
			if importComment != "" {
				existingComment, err := loadJoinedCommentText(d.db, existingPositionID)
				if err != nil {
					return nil, err
				}

				trimmedImport := strings.TrimSpace(importComment)
				trimmedExisting := strings.TrimSpace(existingComment)

				if trimmedExisting == "" && trimmedImport != "" {
					// New comment to add
					hasNewData = true
				} else if trimmedImport != "" && !strings.Contains(trimmedExisting, trimmedImport) {
					// Comment text to merge
					hasNewData = true
				}
			}

			if hasNewData {
				positionsToMerge++
			} else {
				positionsToSkip++
			}
		} else {
			positionsToAdd++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"toAdd":      positionsToAdd,
		"toMerge":    positionsToMerge,
		"toSkip":     positionsToSkip,
		"total":      totalPositions,
		"importPath": importPath,
	}

	slog.Info("import analysis", "toAdd", positionsToAdd, "toMerge", positionsToMerge, "toSkip", positionsToSkip, "total", totalPositions)
	return result, nil
}

// importRun is the state CommitImportDatabase shares with the per-position
// steps it delegates: one transaction on the target, the read-only source, and
// the maps that tie source ids to target ids.
type importRun struct {
	ctx      context.Context
	tx       *sql.Tx
	stx      storage.Tx
	importDB *sql.DB
	carrier  *mets.Carrier
	srcMET   map[int64]int64
	signed   bool
	targetOf map[int64]int64
}

// mergeExisting folds a source position the target already holds into it:
// provenance flag, analysis, then comments. It reports whether anything changed.
func (r *importRun) mergeExisting(id, existingPositionID int64, sourceIndividual bool) (bool, error) {
	ctx, tx, stx, importDB, carrier, srcMET, signed := r.ctx, r.tx, r.stx, r.importDB, r.carrier, r.srcMET, r.signed
	r.targetOf[id] = existingPositionID
	var err error
	hasMerged := false

	// Provenance is sticky (ADR-0001): an individually-imported source
	// position raises the flag on the position we already hold, and a
	// source position that was not individually imported never lowers it.
	if sourceIndividual {
		if _, err := tx.Exec(
			`UPDATE position SET individually_imported = 1
			 WHERE id = ? AND individually_imported = 0`, existingPositionID); err != nil {
			slog.Warn("marking position individually imported", "positionID", existingPositionID, "err", err)
		}
	}

	// Merge analysis if it exists
	var importAnalysisData []byte
	err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisData)

	if err == nil {
		// Load existing analysis from current database (using transaction)
		var existingAnalysisData []byte
		existingErr := tx.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, existingPositionID).Scan(&existingAnalysisData)

		if existingErr == sql.ErrNoRows {
			// No existing analysis, insert the imported one (re-compress for current format)
			recompressed, compErr := recompressAnalysisData(importAnalysisData)
			if compErr != nil {
				recompressed = importAnalysisData
			}
			_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, existingPositionID, recompressed)
			if err != nil {
				slog.Warn("inserting analysis for position", "positionID", existingPositionID, "err", err)
			} else {
				hasMerged = true
				if err := carryTable(ctx, stx, carrier, srcMET[id], existingPositionID); err != nil {
					_ = tx.Rollback()
					return false, err
				}
			}
		} else if existingErr == nil {
			existingAnalysis, _ := decodeAnalysisFromStorage(existingAnalysisData)
			importAnalysis, _ := decodeAnalysisFromStorage(importAnalysisData)
			existingSide, importedSide, metErr := mergeSides(ctx, stx, carrier, srcMET[id], existingPositionID, &existingAnalysis, &importAnalysis)
			if metErr != nil {
				_ = tx.Rollback()
				return false, metErr
			}
			if merged, changed := domain.MergeImportedAnalysis(&existingAnalysis, &importAnalysis); changed {
				var metID any
				if met := mets.AfterMerge(merged, existingSide, importedSide); met != 0 {
					metID = met
				}
				written, err := saveMergedAnalysis(tx, existingPositionID, merged, metID)
				if err != nil {
					return false, err
				}
				if written {
					hasMerged = true
				}
			}
		}
	}

	// Merge comments: each imported row is appended, with its
	// author, when the target's joined text does not already contain
	// it, as ingest.DBImporter does: existing rows are never
	// rewritten.
	importComments, err := loadImportedComments(importDB, id, signed)
	if err != nil {
		slog.Warn("reading import comment", "positionID", id, "err", err)
	} else if len(importComments) > 0 {
		existingComment, err := loadJoinedCommentText(tx, existingPositionID)
		if err != nil {
			slog.Warn("reading existing comment", "positionID", existingPositionID, "err", err)
		}
		for _, c := range importComments {
			trimmed := strings.TrimSpace(c.text)
			if err != nil || trimmed == "" || strings.Contains(existingComment, trimmed) {
				continue
			}
			if _, ierr := tx.Exec(`INSERT INTO comment (position_id, text, author) VALUES (?, ?, ?)`, existingPositionID, trimmed, c.author); ierr != nil {
				slog.Warn("inserting comment for position", "positionID", existingPositionID, "err", ierr)
				continue
			}
			hasMerged = true
			if existingComment == "" {
				existingComment = trimmed
			} else {
				existingComment += "\n\n" + trimmed
			}
		}
	}
	return hasMerged, nil
}

// addNew writes a source position the target does not hold, with its analysis
// and comments. It reports false (and no error) when the position row itself
// could not be inserted.
func (r *importRun) addNew(id int64, importPosition *Position, sourceIndividual bool) (bool, error) {
	ctx, tx, stx, importDB, carrier, srcMET, signed := r.ctx, r.tx, r.stx, r.importDB, r.carrier, r.srcMET, r.signed
	importPosition.IndividuallyImported = sourceIndividual
	newPositionID, err := stx.Positions().Save(ctx, "", importPosition)
	if err != nil {
		slog.Warn("inserting position", "err", err)
		return false, nil
	}
	r.targetOf[id] = newPositionID

	// Copy analysis if it exists
	var importAnalysisData []byte
	err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisData)
	if err == nil {
		// Update position_id in the analysis JSON
		analysis, _ := decodeAnalysisFromStorage(importAnalysisData)
		analysis.PositionID = int(newPositionID)
		updatedAnalysisData, err := encodeAnalysisForStorage(&analysis)
		if err != nil {
			return false, fmt.Errorf("failed to marshal analysis: %w", err)
		}

		_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newPositionID, updatedAnalysisData)
		if err != nil {
			slog.Warn("inserting analysis for new position", "positionID", newPositionID, "err", err)
		} else if err := carryTable(ctx, stx, carrier, srcMET[id], newPositionID); err != nil {
			_ = tx.Rollback()
			return false, err
		}
	}

	// Copy every comment row, each with its author.
	importComments, err := loadImportedComments(importDB, id, signed)
	if err != nil {
		slog.Warn("reading import comment", "positionID", id, "err", err)
	}
	for _, c := range importComments {
		if _, err := tx.Exec(`INSERT INTO comment (position_id, text, author) VALUES (?, ?, ?)`, newPositionID, c.text, c.author); err != nil {
			slog.Warn("inserting comment for new position", "positionID", newPositionID, "err", err)
		}
	}

	return true, nil
}

// CommitImportDatabase performs the actual import within a transaction (ACID)
func (d *Database) CommitImportDatabase(importPath string) (map[string]interface{}, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check that the current database is open
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	// Begin transaction for ACID compliance
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	// The same transaction seen through the Storage contract: brand-new
	// positions are written with PositionStore.Save, so they get their Zobrist
	// hash and scalar search columns like every other position (see the
	// "add it" branch below). Commit/Rollback stay on tx.
	stx := sqlite.WrapTx(tx)

	// Ensure rollback on error or cancellation
	defer func() {
		if err != nil || ctx.Err() != nil {
			tx.Rollback()
			if ctx.Err() != nil {
				slog.Info("transaction rolled back due to user cancellation")
			} else {
				slog.Warn("transaction rolled back due to error")
			}
		}
	}()

	// Open the import database
	importDB, err := openExistingSQLite(importPath)
	if err != nil {
		return nil, err
	}
	defer importDB.Close()
	signed := sourceCommentsSigned(importDB)

	// Check the import database version
	var importDBVersion string
	err = importDB.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&importDBVersion)
	if err != nil {
		return nil, fmt.Errorf("import database is invalid or missing version information")
	}

	// Check the current database version
	var currentDBVersion string
	err = tx.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&currentDBVersion)
	if err != nil {
		return nil, err
	}

	if err := checkImportableVersion(importDBVersion, currentDBVersion); err != nil {
		return nil, err
	}

	// First, count total positions to import
	var totalPositions int
	err = importDB.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&totalPositions)
	if err != nil {
		return nil, err
	}

	// The tables the source's analyses cite travel with them (ADR-0068,
	// rule 5). A failure to carry one fails the import: the analysis would be
	// read as valued with Kazaross-XG2.
	carrier, srcMET, err := readImportMETs(ctx, importDB, stx)
	if err != nil {
		return nil, err
	}

	// OPTIMIZATION: Build a hash map of all current positions ONCE
	// This converts O(n²) to O(n) complexity
	currentPositionsMap, err := positionIdentityIndex(tx)
	if err != nil {
		return nil, err
	}

	slog.Debug("built position index for commit", "count", len(currentPositionsMap))

	// Load all positions from the import database, carrying their provenance.
	rows, err := importDB.Query(sourcePositionQuery(importDB))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positionsAdded int
	var positionsMerged int
	var positionsSkipped int
	// targetOf maps each source position id to the id it holds here, for the
	// collections merged after the positions.
	targetOf := map[int64]int64{}
	run := &importRun{ctx: ctx, tx: tx, stx: stx, importDB: importDB, carrier: carrier, srcMET: srcMET, signed: signed, targetOf: targetOf}

	for rows.Next() {
		// Check for cancellation
		if err = ctx.Err(); err != nil {
			slog.Info("import cancelled by user during processing")
			return nil, wrapImportCancelled(err)
		}

		var id int64
		var stateJSON string
		var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
		var sourceIndividual bool
		if err = rows.Scan(&id, &stateJSON, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &sourceIndividual); err != nil {
			slog.Warn("scanning position", "err", err)
			continue
		}

		importPosition, decErr := decodeSourcePosition(stateJSON, dt, por, d1, d2, cv, co, s1, s2, hj, hb)
		if decErr != nil {
			slog.Warn("unmarshalling position", "err", decErr)
			continue
		}

		// Assigned, not declared: the rollback defer reads this very err.
		var importPositionJSON string
		if importPositionJSON, err = positionIdentityJSON(importPosition); err != nil {
			return nil, err
		}

		// OPTIMIZATION: O(1) hash map lookup instead of nested loop
		existingPositionID, existsInCurrent := currentPositionsMap[importPositionJSON]
		if !existsInCurrent {
			// Read through the transaction: a source that holds the same
			// position twice (pre-2.0.0 databases were never deduplicated) must
			// see the row this very import inserted a moment ago.
			existingPositionID, existsInCurrent, err = heldByZobrist(stx.Positions(), &importPosition)
			if err != nil {
				return nil, err
			}
		}

		if existsInCurrent {
			var changed bool
			if changed, err = run.mergeExisting(id, existingPositionID, sourceIndividual); err != nil {
				return nil, err
			}
			if changed {
				positionsMerged++
			} else {
				positionsSkipped++
			}
		} else {
			var added bool
			if added, err = run.addNew(id, &importPosition, sourceIndividual); err != nil {
				return nil, err
			}
			if added {
				positionsAdded++
			} else {
				positionsSkipped++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	srcCollections, err := readImportCollections(ctx, importDB)
	if err != nil {
		return nil, err
	}
	merged, err := ingest.MergeCollections(ctx, stx, "", srcCollections, targetOf)
	if err != nil {
		return nil, err
	}
	srcLessons, err := readImportLessons(ctx, importDB)
	if err != nil {
		return nil, err
	}
	lessons, err := ingest.MergeLessons(ctx, stx, "", srcLessons, srcCollections, targetOf)
	if err != nil {
		return nil, err
	}
	srcDecks, err := readImportDecks(ctx, importDB)
	if err != nil {
		return nil, err
	}
	decks, err := ingest.MergeDecks(ctx, stx, "", srcDecks, srcCollections, targetOf)
	if err != nil {
		return nil, err
	}

	// Final check for cancellation before committing
	if err = ctx.Err(); err != nil {
		slog.Info("import cancelled by user before commit")
		return nil, wrapImportCancelled(err)
	}

	// Commit the transaction - this makes all changes atomic
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	// The rows written above carry their blob alone; derive their provenance
	// now rather than on the next open.
	if err := d.backfillAnalysisProvenance(ctx); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"added":   positionsAdded,
		"merged":  positionsMerged,
		"skipped": positionsSkipped,
		"total":   totalPositions,
		// The same figures as ingest.Summary on the daemon's imports.db.
		"collections":              merged.Changed,
		"livingCollectionsSkipped": merged.LivingSkipped,
		"lessons":                  lessons,
		"decks":                    decks,
	}

	slog.Info("import committed", "added", positionsAdded, "merged", positionsMerged, "skipped", positionsSkipped, "total", totalPositions)
	return result, nil
}

// readImportCollections reads the source's collections through a read-only
// transaction seen as the Storage contract. A source from before collections
// existed has none.
func readImportCollections(ctx context.Context, importDB *sql.DB) ([]ingest.SourceCollection, error) {
	if !queryable(importDB, `SELECT id FROM collection LIMIT 0`) {
		return nil, nil
	}
	itx, err := importDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer itx.Rollback()
	return ingest.ReadSourceCollections(ctx, sqlite.WrapTx(itx), "")
}

// carryTable records that the imported analysis now at positionID was valued
// with the source table srcMET.
func carryTable(ctx context.Context, stx storage.Stores, carrier *mets.Carrier, srcMET, positionID int64) error {
	met, err := carrier.Target(ctx, srcMET)
	if err != nil || met == 0 {
		return err
	}
	return stx.MatchEquityTables().TagAnalyses(ctx, "", met, []int64{positionID})
}

// mergeSides describes the two analyses an import merges, each with the
// receiver's id of the table its verdict was valued with, read before the
// merge mutates either side: the merged verdict keeps the table of the side
// it came from.
func mergeSides(ctx context.Context, stx storage.Stores, carrier *mets.Carrier, srcMET, positionID int64, existing, imported *PositionAnalysis) (mets.Side, mets.Side, error) {
	existingMET, err := stx.MatchEquityTables().OfAnalysis(ctx, "", positionID)
	if err != nil {
		return mets.Side{}, mets.Side{}, err
	}
	importedMET, err := carrier.Target(ctx, srcMET)
	if err != nil {
		return mets.Side{}, mets.Side{}, err
	}
	return mets.SideOf(existing, existingMET), mets.SideOf(imported, importedMET), nil
}

// readImportMETs reads the source's tables and the table of each analysis
// citing one, and returns a carrier into the receiver. A source from before
// the tables (schema < 2.31.0) cites none.
func readImportMETs(ctx context.Context, importDB *sql.DB, dst storage.Stores) (*mets.Carrier, map[int64]int64, error) {
	ofPosition := map[int64]int64{}
	var tables []*domain.MatchEquityTable
	if sqlite.TableExists(ctx, importDB, "match_equity_table") {
		itx, err := importDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			return nil, nil, err
		}
		defer itx.Rollback()
		if tables, err = mets.ReadTables(ctx, sqlite.WrapTx(itx).MatchEquityTables(), ""); err != nil {
			return nil, nil, fmt.Errorf("reading the source's match equity tables: %w", err)
		}
		rows, err := itx.QueryContext(ctx, `SELECT position_id, met_id FROM analysis WHERE met_id IS NOT NULL`)
		if err != nil {
			return nil, nil, fmt.Errorf("reading the source analyses' match equity tables: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var pos, met int64
			if err := rows.Scan(&pos, &met); err != nil {
				return nil, nil, err
			}
			ofPosition[pos] = met
		}
		if err := rows.Err(); err != nil {
			return nil, nil, err
		}
	}
	return mets.NewCarrier("import", dst.MatchEquityTables(), "", tables), ofPosition, nil
}

// readImportLessons reads the source's Lessons like readImportCollections; a
// source from before Lessons existed has none.
func readImportLessons(ctx context.Context, importDB *sql.DB) ([]*domain.Lesson, error) {
	if !sqlite.TableExists(ctx, importDB, "lesson") {
		return nil, nil
	}
	itx, err := importDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer itx.Rollback()
	return ingest.ReadSourceLessons(ctx, sqlite.WrapTx(itx), "")
}

// readImportDecks reads the source's Anki decks like readImportCollections.
func readImportDecks(ctx context.Context, importDB *sql.DB) ([]ingest.SourceDeck, error) {
	if !sqlite.TableExists(ctx, importDB, "anki_deck") {
		return nil, nil
	}
	itx, err := importDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer itx.Rollback()
	return ingest.ReadSourceDecks(ctx, sqlite.WrapTx(itx), "")
}

// Deprecated: Use AnalyzeImportDatabase followed by CommitImportDatabase instead
func (d *Database) ImportDatabase(importPath string) (map[string]interface{}, error) {
	// This function is kept for backward compatibility but redirects to the new ACID approach
	return d.CommitImportDatabase(importPath)
}

// heldByZobrist reports whether the target already stores pos, by Zobrist
// hash as the store judges it. It backs up the byte-for-byte JSON identity map,
// which misses an un-normalised source position, so such a position still
// takes the merge branch (analysis, comment, provenance).
func heldByZobrist(positions storage.PositionStore, pos *Position) (int64, bool, error) {
	id, held, err := positions.Exists(context.Background(), "", engine.ZobristHash(pos))
	if err != nil {
		return 0, false, fmt.Errorf("checking whether the position is already held: %w", err)
	}
	return id, held, nil
}

// positionIdentityIndex maps every stored position's identity JSON to its id,
// so an import checks each incoming position against the current database in
// O(1) instead of a query per position. A row that fails to scan is skipped:
// it cannot be matched, and the import must not fail on it.
func positionIdentityIndex(q rowQuerier) (map[string]int64, error) {
	index := make(map[string]int64)
	err := forEachRow(q, `SELECT `+positionSelectCols+` FROM position`, nil, func(rows *sql.Rows) error {
		position, err := scanPositionRow(rows)
		if err != nil {
			return nil
		}
		identity, err := positionIdentityJSON(position)
		if err != nil {
			return err
		}
		index[identity] = position.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return index, nil
}

// saveMergedAnalysis writes an imported analysis merged into the stored one.
// An analysis that cannot be encoded or written is skipped (false, nil), as
// the rest of the import does; once it is written, the per-match figures of
// every match reaching the position are stale, and failing to drop them is
// an error: committing would leave them silently wrong.
func saveMergedAnalysis(tx *sql.Tx, positionID int64, merged *domain.PositionAnalysis, metID any) (bool, error) {
	encoded, err := encodeAnalysisForStorage(merged)
	if err != nil {
		slog.Warn("encoding merged analysis for position", "positionID", positionID, "err", err)
		return false, nil
	}
	if _, err := tx.Exec(`UPDATE analysis SET data = ?, analysis_engine = NULL, met_id = ? WHERE position_id = ?`, encoded, metID, positionID); err != nil {
		slog.Warn("updating analysis for position", "positionID", positionID, "err", err)
		return false, nil
	}
	if _, err := tx.Exec(sqlshared.InvalidateMatchStatsOfPositionsSQL+"(?)"+sqlshared.InvalidateMatchStatsOfPositionsSuffix, positionID); err != nil {
		return false, fmt.Errorf("invalidating match statistics of position %d: %w", positionID, err)
	}
	return true, nil
}
