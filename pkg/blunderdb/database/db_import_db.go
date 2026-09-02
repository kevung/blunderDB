package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// sourcePositionQuery selects (id, state, individually_imported) from the
// database being imported.
//
// Databases older than 2.13.0 have no individually_imported column, so the flag
// is derived there the same way the 2.12.0→2.13.0 migration derives it: a
// position reachable from no move never came from a match. Using the same rule
// in both places is what keeps the two routes consistent — migrating a database
// in place and importing it into another must not disagree about which of its
// positions were individually imported.
func sourcePositionQuery(importDB *sql.DB) string {
	if queryable(importDB, `SELECT individually_imported FROM position LIMIT 1`) {
		return `SELECT id, state, individually_imported FROM position`
	}
	if queryable(importDB, `SELECT 1 FROM move LIMIT 1`) {
		slog.Debug("import database predates individually_imported; deriving it from the move graph")
		return `SELECT p.id, p.state,
			       NOT EXISTS (SELECT 1 FROM move m WHERE m.position_id = p.id)
			FROM position p`
	}
	// No move table at all: the database holds no matches, so every position in
	// it stands on its own. This is the same rule as the derivation above, taken
	// to its limit.
	slog.Debug("import database has no move table; all its positions are individually imported")
	return `SELECT id, state, 1 FROM position`
}

// queryable reports whether q runs against db — used to probe for a column or a
// table that older schema versions may not have.
func queryable(db *sql.DB, q string) bool {
	var dummy int
	err := db.QueryRow(q).Scan(&dummy)
	return err == nil || err == sql.ErrNoRows
}

// decodeSourcePosition reconstructs a Position from a row of the database
// being imported. Full-JSON state (every pre-2.2.0 database, and every export
// before fiche-04) is self-describing: json.Unmarshal is enough. Compact state
// is not — it holds only the board — so the scalar columns that carry
// everything else (dice, score, cube, decision type) have to be read from the
// same row.
//
// Compact state is only ever produced by a database on the 2.2.0+ schema,
// which always has those columns (storage/sqlite.Bootstrap creates them
// unconditionally), so the lookup below cannot fail for a genuine compact
// export or an ordinary user database. It exists so a hand-built or corrupted
// fixture degrades to a board-only Position — losing dice/score identity, so
// it can be merged as "new" rather than aborting the whole import — instead
// of erroring out.
//
// Without this, importing one current-schema database into another (the
// "Import database" GUI feature, and fiche-04's own exports once they started
// writing compact state) silently duplicated every position: the decode used
// to zero every field but the board, so positionIdentityJSON never matched an
// existing row.
func decodeSourcePosition(importDB *sql.DB, id int64, state string) (Position, error) {
	var pos Position
	if !isCompactState(state) {
		if err := json.Unmarshal([]byte(state), &pos); err != nil {
			return Position{}, err
		}
		return pos, nil
	}
	pos.Board = decodeBoardCompact(state)
	var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
	err := importDB.QueryRow(`SELECT decision_type, player_on_roll, dice_1, dice_2,
		cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver
		FROM position WHERE id = ?`, id).
		Scan(&dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb)
	if err != nil {
		slog.Warn("reading scalar columns for a compact-state import position; identity will be board-only", "id", id, "err", err)
		return pos, nil
	}
	pos.DecisionType = int(dt.Int64)
	pos.PlayerOnRoll = int(por.Int64)
	pos.Dice = [2]int{int(d1.Int64), int(d2.Int64)}
	pos.Cube = Cube{Owner: int(co.Int64), Value: int(cv.Int64)}
	pos.Score = [2]int{int(s1.Int64), int(s2.Int64)}
	pos.HasJacoby = int(hj.Int64)
	pos.HasBeaver = int(hb.Int64)
	return pos, nil
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

	// Compare major versions - allow importing from same or lower version
	importMajor := strings.Split(importDBVersion, ".")[0]
	currentMajor := strings.Split(currentDBVersion, ".")[0]

	if importMajor > currentMajor {
		return nil, fmt.Errorf("cannot import from a newer major database version (import: %s, current: %s)", importDBVersion, currentDBVersion)
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

	// Analyze what would happen
	rows, err := importDB.Query(`SELECT id, state FROM position`)
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
		if err = rows.Scan(&id, &stateJSON); err != nil {
			slog.Warn("scanning position", "err", err)
			positionsToSkip++
			continue
		}

		importPosition, decErr := decodeSourcePosition(importDB, id, stateJSON)
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

			// Check for comments to merge
			var importComment string
			err = importDB.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, id).Scan(&importComment)
			if err == nil && importComment != "" {
				var existingComment string
				existingErr := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, existingPositionID).Scan(&existingComment)

				trimmedImport := strings.TrimSpace(importComment)
				trimmedExisting := strings.TrimSpace(existingComment)

				if existingErr == sql.ErrNoRows {
					// New comment to add
					hasNewData = true
				} else if existingErr == nil && trimmedImport != "" && !strings.Contains(trimmedExisting, trimmedImport) {
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

	// Compare major versions - allow importing from same or lower version
	importMajor := strings.Split(importDBVersion, ".")[0]
	currentMajor := strings.Split(currentDBVersion, ".")[0]

	if importMajor > currentMajor {
		return nil, fmt.Errorf("cannot import from a newer major database version (import: %s, current: %s)", importDBVersion, currentDBVersion)
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
			return nil, fmt.Errorf("import cancelled by user")
		}

		var id int64
		var stateJSON string
		var sourceIndividual bool
		if err = rows.Scan(&id, &stateJSON, &sourceIndividual); err != nil {
			slog.Warn("scanning position", "err", err)
			continue
		}

		importPosition, decErr := decodeSourcePosition(importDB, id, stateJSON)
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

			// Merge comments
			var importComment string
			err = importDB.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, id).Scan(&importComment)

			if err == nil && importComment != "" {
				var existingComment string
				existingErr := tx.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, existingPositionID).Scan(&existingComment)

				trimmedImport := strings.TrimSpace(importComment)
				trimmedExisting := strings.TrimSpace(existingComment)

				if existingErr == sql.ErrNoRows {
					// No existing comment, insert the imported one
					_, err = tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, existingPositionID, importComment)
					if err != nil {
						slog.Warn("inserting comment for position", "positionID", existingPositionID, "err", err)
					} else {
						hasMerged = true
					}
				} else if existingErr == nil {
					// Merge comments - only add if not already present
					if trimmedImport != "" && !strings.Contains(trimmedExisting, trimmedImport) {
						var mergedComment string
						if trimmedExisting != "" {
							mergedComment = trimmedExisting + "\n\n" + trimmedImport
						} else {
							mergedComment = trimmedImport
						}
						_, err = tx.Exec(`UPDATE comment SET text = ? WHERE position_id = ?`, mergedComment, existingPositionID)
						if err != nil {
							slog.Warn("updating comment for position", "positionID", existingPositionID, "err", err)
						} else {
							hasMerged = true
						}
					}
				}
			}

			if hasMerged {
				positionsMerged++
			} else {
				positionsSkipped++
			}
		} else {
			// Position doesn't exist: add it through the canonical write path,
			// inside the transaction. Save is what computes the Zobrist hash and
			// the scalar search columns (pip counts, back checkers, no-contact,
			// dice, score, cube…); a raw INSERT of the state used to leave them
			// all NULL, which hid the row from every SQL filter and — because
			// ReconstructPosition trusts the columns over the state — made it
			// fail to match itself on the next import of the same database.
			// Provenance rides along: Save ORs IndividuallyImported into the
			// stored flag (ADR-0001).
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
		return nil, fmt.Errorf("import cancelled by user")
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

// heldByZobrist reports whether the target already stores pos, judged the way
// the store itself judges it — by Zobrist hash. It backs up the JSON identity
// map the importer keys its lookup on: that map compares marshalled Positions
// byte for byte, so a source position that is the same position but was
// stored un-normalised (a hand-built fixture, a database written before
// NormalizeForStorage existed) would slip past it and be inserted again, only
// for Save's unique index to hand back the existing id. Asking the index first
// keeps the merge branch (analysis, comment, provenance) on that path too.
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
