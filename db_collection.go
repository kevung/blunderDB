package main

import (
	"database/sql"
	"fmt"
	"os"
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
	d.mu.Lock()
	defer d.mu.Unlock()

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
	d.mu.Lock()
	defer d.mu.Unlock()

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
	d.mu.Lock()
	defer d.mu.Unlock()

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
	d.mu.Lock()
	defer d.mu.Unlock()

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
	d.mu.Lock()
	defer d.mu.Unlock()

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
			continue
		}
		collections = append(collections, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return collections, nil
}

// ExportCollections exports specific collections to a database file
func (d *Database) ExportCollections(exportPath string, collectionIDs []int64, metadata map[string]string, includeAnalysis bool, includeComments bool) error {
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
			if err := rows.Scan(&posID); err == nil {
				positionIDsMap[posID] = true
			}
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

	// Delete the export file if it already exists
	if _, err := os.Stat(exportPath); err == nil {
		if err := os.Remove(exportPath); err != nil {
			return fmt.Errorf("cannot remove existing export file: %v", err)
		}
	}

	// Create export database
	exportDB, err := sql.Open("sqlite", exportPath)
	if err != nil {
		return err
	}
	defer exportDB.Close()

	// Create schema
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER UNIQUE,
			data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS comment (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER UNIQUE,
			text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
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

	_, err = exportDB.Exec(`
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

	// Create match-related tables (for v1.4.0 compatibility)
	_, err = exportDB.Exec(`
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
			match_hash TEXT,
			last_visited_position INTEGER DEFAULT -1
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`CREATE TABLE IF NOT EXISTS game (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id INTEGER,
		game_number INTEGER,
		initial_score_1 INTEGER,
		initial_score_2 INTEGER,
		winner INTEGER,
		points_won INTEGER,
		move_count INTEGER DEFAULT 0,
		FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
	)`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`CREATE TABLE IF NOT EXISTS move (
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
	)`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`CREATE TABLE IF NOT EXISTS move_analysis (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		move_id INTEGER,
		analysis_type TEXT,
		depth TEXT,
		equity INTEGER,
		equity_error INTEGER,
		win_rate INTEGER,
		gammon_rate INTEGER,
		backgammon_rate INTEGER,
		opponent_win_rate INTEGER,
		opponent_gammon_rate INTEGER,
		opponent_backgammon_rate INTEGER,
		FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
	)`)
	if err != nil {
		return err
	}

	// Create collection tables
	_, err = exportDB.Exec(`
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

	_, err = exportDB.Exec(`
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

	// Export positions and create ID mapping
	oldToNewID := make(map[int64]int64)
	for _, posID := range positionIDs {
		pos, err := d.loadPositionByIDUnlocked(posID)
		if err != nil {
			continue
		}

		result, err := exportDB.Exec(`INSERT INTO position (state) VALUES (?)`, fullPositionJSON(pos))
		if err != nil {
			continue
		}
		newID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		oldToNewID[posID] = newID

		// Export analysis if requested
		if includeAnalysis {
			var analysisData []byte
			err := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, posID).Scan(&analysisData)
			if err == nil {
				// Decompress for export compatibility
				jsonData, _ := decompressAnalysisData(analysisData)
				_, _ = exportDB.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newID, string(jsonData))
			}
		}

		// Export comments if requested
		if includeComments {
			var commentText string
			err := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, posID).Scan(&commentText)
			if err == nil {
				_, _ = exportDB.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newID, commentText)
			}
		}
	}

	// Export collections and their position mappings
	collectionIDMapping := make(map[int64]int64)
	for _, collectionID := range collectionIDs {
		var name, description string
		var sortOrder int
		var createdAt, updatedAt string
		err := d.db.QueryRow(`SELECT name, COALESCE(description, ''), sort_order, created_at, updated_at FROM collection WHERE id = ?`, collectionID).
			Scan(&name, &description, &sortOrder, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		result, err := exportDB.Exec(`INSERT INTO collection (name, description, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			name, description, sortOrder, createdAt, updatedAt)
		if err != nil {
			continue
		}
		newCollectionID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		collectionIDMapping[collectionID] = newCollectionID

		// Export collection_position mappings
		rows, err := d.db.Query(`SELECT position_id, sort_order, added_at FROM collection_position WHERE collection_id = ?`, collectionID)
		if err != nil {
			continue
		}
		for rows.Next() {
			var oldPosID int64
			var sortOrder int
			var addedAt string
			if err := rows.Scan(&oldPosID, &sortOrder, &addedAt); err != nil {
				continue
			}
			if newPosID, ok := oldToNewID[oldPosID]; ok {
				_, _ = exportDB.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order, added_at) VALUES (?, ?, ?, ?)`,
					newCollectionID, newPosID, sortOrder, addedAt)
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// Export metadata
	_, err = exportDB.Exec(`INSERT INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		return err
	}

	for key, value := range metadata {
		_, _ = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
	}

	return nil
}
