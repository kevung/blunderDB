package database

import (
	"fmt"
	"log/slog"
)

// Collection represents a collection of positions
type Collection struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	SortOrder     int    `json:"sortOrder"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	PositionCount int    `json:"positionCount"`
}

// CollectionPosition represents a position in a collection with its order
type CollectionPosition struct {
	ID           int64    `json:"id"`
	CollectionID int64    `json:"collectionId"`
	PositionID   int64    `json:"positionId"`
	SortOrder    int      `json:"sortOrder"`
	AddedAt      string   `json:"addedAt"`
	Position     Position `json:"position"`
}

// CreateCollection creates a new collection
func (d *Database) CreateCollection(name string, description string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}

	// Get the max sort_order
	var maxOrder int
	err := d.db.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM collection`).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	result, err := d.db.Exec(`
		INSERT INTO collection (name, description, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, name, description, maxOrder+1)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetAllCollections returns all collections with their position counts
func (d *Database) GetAllCollections() ([]Collection, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			c.id,
			c.name,
			COALESCE(c.description, ''),
			c.sort_order,
			COALESCE(strftime('%Y-%m-%d %H:%M:%S', c.created_at), ''),
			COALESCE(strftime('%Y-%m-%d %H:%M:%S', c.updated_at), ''),
			COUNT(cp.id) as position_count
		FROM collection c
		LEFT JOIN collection_position cp ON c.id = cp.collection_id
		GROUP BY c.id
		ORDER BY c.sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt, &c.PositionCount)
		if err != nil {
			slog.Warn("scanning collection", "err", err)
			continue
		}
		collections = append(collections, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return collections, nil
}

// UpdateCollection updates a collection's name and description
func (d *Database) UpdateCollection(id int64, name string, description string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`
		UPDATE collection SET name = ?, description = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, description, id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteCollection deletes a collection and all its position associations
func (d *Database) DeleteCollection(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`DELETE FROM collection WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return nil
}

// ReorderCollections updates the sort order of all collections
func (d *Database) ReorderCollections(collectionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for i, id := range collectionIDs {
		_, err := tx.Exec(`UPDATE collection SET sort_order = ? WHERE id = ?`, i, id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// AddPositionToCollection adds a position to a collection
func (d *Database) AddPositionToCollection(collectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the max sort_order for this collection
	var maxOrder int
	err = tx.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = ?`, collectionID).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	_, err = tx.Exec(`
		INSERT OR IGNORE INTO collection_position (collection_id, position_id, sort_order, added_at)
		VALUES (?, ?, ?, datetime('now'))
	`, collectionID, positionID, maxOrder+1)
	if err != nil {
		return err
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AddPositionsToCollection adds multiple positions to a collection
func (d *Database) AddPositionsToCollection(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Get the max sort_order for this collection
	var maxOrder int
	err := d.db.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = ?`, collectionID).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for i, positionID := range positionIDs {
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO collection_position (collection_id, position_id, sort_order, added_at)
			VALUES (?, ?, ?, datetime('now'))
		`, collectionID, positionID, maxOrder+1+i)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// RemovePositionFromCollection removes a position from a collection
func (d *Database) RemovePositionFromCollection(collectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM collection_position 
		WHERE collection_id = ? AND position_id = ?
	`, collectionID, positionID)
	if err != nil {
		return err
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RemovePositionsFromCollection removes multiple positions from a collection
func (d *Database) RemovePositionsFromCollection(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, positionID := range positionIDs {
		_, err = tx.Exec(`
			DELETE FROM collection_position 
			WHERE collection_id = ? AND position_id = ?
		`, collectionID, positionID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GetPositionIndexMap returns a map of position ID to its 1-based index in the database
func (d *Database) GetPositionIndexMap() (map[int64]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`SELECT id FROM position ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]int)
	index := 1
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			slog.Warn("scanning position id", "err", err)
			continue
		}
		result[id] = index
		index++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetCollectionPositions returns all positions in a collection
func (d *Database) GetCollectionPositions(collectionID int64) ([]Position, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT `+positionSelectColsP+`
		FROM position p
		INNER JOIN collection_position cp ON p.id = cp.position_id
		WHERE cp.collection_id = ?
		ORDER BY cp.sort_order ASC
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		position, err := scanPositionRow(rows)
		if err != nil {
			slog.Warn("scanning collection position", "collectionID", collectionID, "err", err)
			continue
		}
		positions = append(positions, position)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return positions, nil
}

// ReorderCollectionPositions updates the sort order of positions within a collection
func (d *Database) ReorderCollectionPositions(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for i, positionID := range positionIDs {
		_, err := tx.Exec(`
			UPDATE collection_position SET sort_order = ?
			WHERE collection_id = ? AND position_id = ?
		`, i, collectionID, positionID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Update collection's updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// MovePositionBetweenCollections moves a position from one collection to another
func (d *Database) MovePositionBetweenCollections(fromCollectionID int64, toCollectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// Remove from source collection
	_, err = tx.Exec(`
		DELETE FROM collection_position 
		WHERE collection_id = ? AND position_id = ?
	`, fromCollectionID, positionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Get max sort_order in destination collection
	var maxOrder int
	err = tx.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = ?`, toCollectionID).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	// Add to destination collection
	_, err = tx.Exec(`
		INSERT INTO collection_position (collection_id, position_id, sort_order, added_at)
		VALUES (?, ?, ?, datetime('now'))
	`, toCollectionID, positionID, maxOrder+1)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Update both collections' updated_at
	_, err = tx.Exec(`UPDATE collection SET updated_at = datetime('now') WHERE id IN (?, ?)`, fromCollectionID, toCollectionID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// CopyPositionToCollection copies a position to a collection (position can be in multiple collections)
func (d *Database) CopyPositionToCollection(toCollectionID int64, positionID int64) error {
	return d.AddPositionToCollection(toCollectionID, positionID)
}

// GetCollectionByID returns a collection by its ID
func (d *Database) GetCollectionByID(id int64) (*Collection, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var c Collection
	err := d.db.QueryRow(`
		SELECT 
			c.id,
			c.name,
			COALESCE(c.description, ''),
			c.sort_order,
			COALESCE(strftime('%Y-%m-%d %H:%M:%S', c.created_at), ''),
			COALESCE(strftime('%Y-%m-%d %H:%M:%S', c.updated_at), ''),
			COUNT(cp.id) as position_count
		FROM collection c
		LEFT JOIN collection_position cp ON c.id = cp.collection_id
		WHERE c.id = ?
		GROUP BY c.id
	`, id).Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt, &c.PositionCount)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// GetPositionCollections returns all collections that contain a specific position
func (d *Database) GetPositionCollections(positionID int64) ([]Collection, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			c.id,
			c.name,
			COALESCE(c.description, ''),
			c.sort_order,
			COALESCE(strftime('%Y-%m-%d %H:%M:%S', c.created_at), ''),
			COALESCE(strftime('%Y-%m-%d %H:%M:%S', c.updated_at), '')
		FROM collection c
		INNER JOIN collection_position cp ON c.id = cp.collection_id
		WHERE cp.position_id = ?
		ORDER BY c.sort_order ASC
	`, positionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			slog.Warn("scanning collection for position", "positionID", positionID, "err", err)
			continue
		}
		collections = append(collections, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return collections, nil
}

// ExportCollections exports specific collections to a database file. watermark
// and watermarkNote mirror ExportDatabase's Watermark/WatermarkNote: an empty
// watermark means the export carries none. See db_export_position.go for the
// schema and position-writing code shared with the other two export paths.
func (d *Database) ExportCollections(exportPath string, collectionIDs []int64, metadata map[string]string, includeAnalysis bool, includeComments bool, watermark string, watermarkNote string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Collect all unique position IDs from selected collections
	positionIDsMap := make(map[int64]bool)
	for _, collectionID := range collectionIDs {
		rows, err := d.db.Query(`SELECT position_id FROM collection_position WHERE collection_id = ?`, collectionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var posID int64
			if err := rows.Scan(&posID); err != nil {
				slog.Warn("scanning collection_position id for export", "collectionID", collectionID, "err", err)
				continue
			}
			positionIDsMap[posID] = true
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// Convert map to slice
	var positionIDs []int64
	for id := range positionIDsMap {
		positionIDs = append(positionIDs, id)
	}

	exportDB, err := newExportDB(exportPath)
	if err != nil {
		return err
	}
	defer exportDB.Close()

	if err := writeExportMetadata(exportDB, metadata, watermark, watermarkNote); err != nil {
		return err
	}

	// Read every position and (if requested) its analysis/comment in one batched
	// statement each, instead of the N+1 per-position SELECTs this used to run —
	// the same helpers ExportDatabase uses (db_export.go).
	positions, err := d.positionsByIDsLocked(positionIDs)
	if err != nil {
		return fmt.Errorf("cannot read the positions to export: %w", err)
	}
	var analysisByPosition map[int64][]byte
	if includeAnalysis {
		if analysisByPosition, err = d.analysisForPositions(positionIDs); err != nil {
			return fmt.Errorf("cannot read analyses to export: %w", err)
		}
	}
	var commentByPosition map[int64]string
	if includeComments {
		if commentByPosition, err = d.commentsForPositions(positionIDs); err != nil {
			return fmt.Errorf("cannot read comments to export: %w", err)
		}
	}

	tx, err := exportDB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	insertPosition, err := tx.Prepare(exportPositionInsertSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare the position insert: %w", err)
	}
	defer insertPosition.Close()
	lookupPosition, err := tx.Prepare(exportPositionLookupSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare the position lookup: %w", err)
	}
	defer lookupPosition.Close()

	// Export positions and create ID mapping
	oldToNewID := make(map[int64]int64, len(positions))
	for _, pos := range positions {
		posID := pos.ID
		newID, insErr := insertExportPosition(insertPosition, lookupPosition, pos)
		if insErr != nil {
			slog.Warn("inserting position into collection export database", "positionID", posID, "err", insErr)
			continue
		}
		oldToNewID[posID] = newID

		if includeAnalysis {
			if data, ok := analysisByPosition[posID]; ok {
				// Decompress for export compatibility (the export's analysis.data holds
				// plain JSON, like ExportDatabase's).
				jsonData, decErr := decompressAnalysisData(data)
				if decErr != nil {
					slog.Warn("decompressing analysis for collection export", "positionID", posID, "err", decErr)
				} else if _, insErr := tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newID, string(jsonData)); insErr != nil {
					slog.Warn("inserting analysis into collection export database", "positionID", posID, "err", insErr)
				}
			}
		}

		if includeComments {
			if text := commentByPosition[posID]; text != "" {
				if _, insErr := tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newID, text); insErr != nil {
					slog.Warn("inserting comment into collection export database", "positionID", posID, "err", insErr)
				}
			}
		}
	}

	// Export collections and their position mappings
	for _, collectionID := range collectionIDs {
		var name, description string
		var sortOrder int
		var createdAt, updatedAt string
		cErr := d.db.QueryRow(`SELECT name, COALESCE(description, ''), sort_order, created_at, updated_at FROM collection WHERE id = ?`, collectionID).
			Scan(&name, &description, &sortOrder, &createdAt, &updatedAt)
		if cErr != nil {
			slog.Warn("reading collection for export", "collectionID", collectionID, "err", cErr)
			continue
		}

		result, insErr := tx.Exec(`INSERT INTO collection (name, description, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			name, description, sortOrder, createdAt, updatedAt)
		if insErr != nil {
			slog.Warn("inserting collection into export database", "collectionID", collectionID, "err", insErr)
			continue
		}
		newCollectionID, idErr := result.LastInsertId()
		if idErr != nil {
			err = fmt.Errorf("failed to get last insert ID: %w", idErr)
			return err
		}

		// Export collection_position mappings
		rows, qErr := d.db.Query(`SELECT position_id, sort_order, added_at FROM collection_position WHERE collection_id = ?`, collectionID)
		if qErr != nil {
			slog.Warn("querying collection_position for export", "collectionID", collectionID, "err", qErr)
			continue
		}
		for rows.Next() {
			var oldPosID int64
			var cpSortOrder int
			var addedAt string
			if sErr := rows.Scan(&oldPosID, &cpSortOrder, &addedAt); sErr != nil {
				slog.Warn("scanning collection_position for export", "collectionID", collectionID, "err", sErr)
				continue
			}
			if newPosID, ok := oldToNewID[oldPosID]; ok {
				if _, insErr := tx.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order, added_at) VALUES (?, ?, ?, ?)`,
					newCollectionID, newPosID, cpSortOrder, addedAt); insErr != nil {
					slog.Warn("inserting collection_position into export database", "collectionID", collectionID, "err", insErr)
				}
			}
		}
		if rErr := rows.Err(); rErr != nil {
			rows.Close()
			err = rErr
			return err
		}
		rows.Close()
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// CollectionCoverage reports, for every collection, how many of its positions are part of
// the given selection.
//
// The export writes a collection's membership only for positions it is actually exporting,
// so a collection whose positions are not all in the selection arrives truncated. That used
// to happen in silence: the recipient believed they had the collection and had a fragment of
// it. This lets the export screen say so before anything is written.
func (d *Database) CollectionCoverage(positionIDs []int64) (map[int64]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	selected := make(map[int64]bool, len(positionIDs))
	for _, id := range positionIDs {
		selected[id] = true
	}

	rows, err := d.db.Query(`SELECT collection_id, position_id FROM collection_position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coverage := make(map[int64]int)
	for rows.Next() {
		var collectionID, positionID int64
		if err := rows.Scan(&collectionID, &positionID); err != nil {
			return nil, err
		}
		if _, ok := coverage[collectionID]; !ok {
			coverage[collectionID] = 0
		}
		if selected[positionID] {
			coverage[collectionID]++
		}
	}
	return coverage, rows.Err()
}
