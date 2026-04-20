package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
)

// AnalyzeImportDatabase analyzes what would be imported without making changes
func (d *Database) AnalyzeImportDatabase(importPath string) (map[string]interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check that the current database is open
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	// Open the import database
	importDB, err := sql.Open("sqlite", importPath)
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
	currentPositionsMap := make(map[string]int64) // map[positionJSON]positionID
	currentRows, err := d.db.Query(`SELECT ` + positionSelectCols + ` FROM position`)
	if err != nil {
		return nil, err
	}

	for currentRows.Next() {
		currentPosition, err := scanPositionRow(currentRows)
		if err != nil {
			continue
		}
		positionID := currentPosition.ID

		// Reset ID for comparison
		currentPosition.ID = 0
		currentPositionJSON, err := json.Marshal(currentPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		currentPositionsMap[string(currentPositionJSON)] = positionID
	}
	if err := currentRows.Err(); err != nil {
		return nil, err
	}
	currentRows.Close()

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

		var importPosition Position
		if isCompactState(stateJSON) {
			importPosition.Board = decodeBoardCompact(stateJSON)
		} else if err = json.Unmarshal([]byte(stateJSON), &importPosition); err != nil {
			slog.Warn("unmarshalling position", "err", err)
			positionsToSkip++
			continue
		}

		// Reset ID for existence check
		importPosition.ID = 0
		importPositionJSON, err := json.Marshal(importPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}

		// OPTIMIZATION: O(1) hash map lookup instead of nested loop
		existingPositionID, existsInCurrent := currentPositionsMap[string(importPositionJSON)]

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
	d.mu.Lock()
	defer d.mu.Unlock()

	// Reset cancellation flag at start
	d.resetImportCancellation()

	// Check that the current database is open
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	// Begin transaction for ACID compliance
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	// Ensure rollback on error or cancellation
	defer func() {
		if err != nil || d.isImportCancelled() {
			tx.Rollback()
			if d.isImportCancelled() {
				slog.Info("transaction rolled back due to user cancellation")
			} else {
				slog.Warn("transaction rolled back due to error")
			}
		}
	}()

	// Open the import database
	importDB, err := sql.Open("sqlite", importPath)
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
	currentPositionsMap := make(map[string]int64) // map[positionJSON]positionID
	currentRows, err := tx.Query(`SELECT ` + positionSelectCols + ` FROM position`)
	if err != nil {
		return nil, err
	}

	for currentRows.Next() {
		currentPosition, err := scanPositionRow(currentRows)
		if err != nil {
			continue
		}
		positionID := currentPosition.ID

		// Reset ID for comparison
		currentPosition.ID = 0
		currentPositionJSON, err := json.Marshal(currentPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		currentPositionsMap[string(currentPositionJSON)] = positionID
	}
	if err := currentRows.Err(); err != nil {
		return nil, err
	}
	currentRows.Close()

	slog.Debug("built position index for commit", "count", len(currentPositionsMap))

	// Load all positions from the import database
	rows, err := importDB.Query(`SELECT id, state FROM position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positionsAdded int
	var positionsMerged int
	var positionsSkipped int

	for rows.Next() {
		// Check for cancellation
		if d.isImportCancelled() {
			slog.Info("import cancelled by user during processing")
			return nil, fmt.Errorf("import cancelled by user")
		}

		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			slog.Warn("scanning position", "err", err)
			continue
		}

		var importPosition Position
		if isCompactState(stateJSON) {
			importPosition.Board = decodeBoardCompact(stateJSON)
		} else if err = json.Unmarshal([]byte(stateJSON), &importPosition); err != nil {
			slog.Warn("unmarshalling position", "err", err)
			continue
		}

		// Reset ID for existence check
		importPosition.ID = 0
		importPositionJSON, err := json.Marshal(importPosition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}

		// OPTIMIZATION: O(1) hash map lookup instead of nested loop
		existingPositionID, existsInCurrent := currentPositionsMap[string(importPositionJSON)]

		if existsInCurrent {
			// Track if we actually merge anything
			hasMerged := false

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
			// Position doesn't exist, add it (using transaction)
			// Store as full JSON (import DB may not have denormalized columns)
			fullJSON := fullPositionJSON(importPosition)
			result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, fullJSON)
			if err != nil {
				slog.Warn("inserting position", "err", err)
				positionsSkipped++
				continue
			}

			newPositionID, err := result.LastInsertId()
			if err != nil {
				slog.Warn("getting last insert ID", "err", err)
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
	if d.isImportCancelled() {
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

// CancelImport sets the flag to cancel any ongoing import operation
func (d *Database) CancelImport() {
	atomic.StoreInt32(&d.importCancelled, 1)
	slog.Info("import cancellation requested")
}

// isImportCancelled checks if import has been cancelled (internal method, no lock needed as it's called within locked context)
func (d *Database) isImportCancelled() bool {
	return atomic.LoadInt32(&d.importCancelled) == 1
}

// resetImportCancellation resets the cancellation flag (internal method)
func (d *Database) resetImportCancellation() {
	atomic.StoreInt32(&d.importCancelled, 0)
}

// Deprecated: Use AnalyzeImportDatabase followed by CommitImportDatabase instead
func (d *Database) ImportDatabase(importPath string) (map[string]interface{}, error) {
	// This function is kept for backward compatibility but redirects to the new ACID approach
	return d.CommitImportDatabase(importPath)
}
