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
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
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

		importPositionJSON, err := positionIdentityJSON(importPosition)
		if err != nil {
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
					// Check if import has better analysis
					existingAnalysis, _ := decodeAnalysisFromStorage(existingAnalysisData)
					importAnalysis, _ := decodeAnalysisFromStorage(importAnalysisData)

					if existingAnalysis.AnalysisType == "" && importAnalysis.AnalysisType != "" {
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

		importPositionJSON, err := positionIdentityJSON(importPosition)
		if err != nil {
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
			// Track if we actually merge anything
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
					}
				} else if existingErr == nil {
					// Both have analysis - keep the existing one unless it's empty
					existingAnalysis, _ := decodeAnalysisFromStorage(existingAnalysisData)
					importAnalysis, _ := decodeAnalysisFromStorage(importAnalysisData)

					// If import has analysis but existing doesn't, use import
					if existingAnalysis.AnalysisType == "" && importAnalysis.AnalysisType != "" {
						recompressed, compErr := recompressAnalysisData(importAnalysisData)
						if compErr != nil {
							recompressed = importAnalysisData
						}
						_, err = tx.Exec(`UPDATE analysis SET data = ? WHERE position_id = ?`, recompressed, existingPositionID)
						if err != nil {
							slog.Warn("updating analysis for position", "positionID", existingPositionID, "err", err)
						} else {
							hasMerged = true
						}
					}
				}
			}

			// Merge comments, joining every row on both sides. The imported
			// text is appended as a new row when not already contained, as
			// ingest.DBImporter does: existing rows are never rewritten.
			importComment, err := loadJoinedCommentText(importDB, id)
			if err != nil {
				slog.Warn("reading import comment", "positionID", id, "err", err)
			} else if trimmedImport := strings.TrimSpace(importComment); trimmedImport != "" {
				existingComment, err := loadJoinedCommentText(tx, existingPositionID)
				if err != nil {
					slog.Warn("reading existing comment", "positionID", existingPositionID, "err", err)
				} else if !strings.Contains(existingComment, trimmedImport) {
					if _, err := tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, existingPositionID, trimmedImport); err != nil {
						slog.Warn("inserting comment for position", "positionID", existingPositionID, "err", err)
					} else {
						hasMerged = true
					}
				}
			}

			if hasMerged {
				positionsMerged++
			} else {
				positionsSkipped++
			}
		} else {
			// New position: write through Save, which computes the Zobrist
			// hash and scalar search columns — a raw INSERT leaves them NULL,
			// hiding the row from filters and from its own next import. Save
			// ORs IndividuallyImported into the stored flag (ADR-0001).
			importPosition.IndividuallyImported = sourceIndividual
			newPositionID, err := stx.Positions().Save(ctx, "", &importPosition)
			if err != nil {
				slog.Warn("inserting position", "err", err)
				positionsSkipped++
				continue
			}

			// Copy analysis if it exists
			var importAnalysisData []byte
			err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisData)
			if err == nil {
				// Update position_id in the analysis JSON
				analysis, _ := decodeAnalysisFromStorage(importAnalysisData)
				analysis.PositionID = int(newPositionID)
				updatedAnalysisData, err := encodeAnalysisForStorage(&analysis)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal analysis: %w", err)
				}

				_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newPositionID, updatedAnalysisData)
				if err != nil {
					slog.Warn("inserting analysis for new position", "positionID", newPositionID, "err", err)
				}
			}

			// Copy comment if it exists
			var importComment string
			err = importDB.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, id).Scan(&importComment)
			if err == nil && importComment != "" {
				_, err = tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newPositionID, importComment)
				if err != nil {
					slog.Warn("inserting comment for new position", "positionID", newPositionID, "err", err)
				}
			}

			positionsAdded++
		}
	}
	if err := rows.Err(); err != nil {
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

	result := map[string]interface{}{
		"added":   positionsAdded,
		"merged":  positionsMerged,
		"skipped": positionsSkipped,
		"total":   totalPositions,
	}

	slog.Info("import committed", "added", positionsAdded, "merged", positionsMerged, "skipped", positionsSkipped, "total", totalPositions)
	return result, nil
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
