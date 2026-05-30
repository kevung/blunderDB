package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// SaveCommand saves a command to the command_history table
func (d *Database) SaveCommand(command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.1.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion < "1.1.0" {
		return fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	_, err = d.db.Exec(`INSERT INTO command_history (command) VALUES (?)`, command)
	if err != nil {
		return err
	}

	// Keep only the last 1000 entries to prevent unbounded growth
	_, _ = d.db.Exec(`
		DELETE FROM command_history
		WHERE id NOT IN (
			SELECT id FROM command_history
			ORDER BY timestamp DESC
			LIMIT 1000
		)
	`)

	return nil
}

// LoadCommandHistory loads the command history from the command_history table
func (d *Database) LoadCommandHistory() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check if the database version is 1.1.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return nil, err
	}

	if dbVersion < "1.1.0" {
		return nil, fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	rows, err := d.db.Query(`SELECT command FROM command_history ORDER BY timestamp ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []string
	for rows.Next() {
		var command string
		if err = rows.Scan(&command); err != nil {
			return nil, err
		}
		history = append(history, command)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return history, nil
}

func (d *Database) ClearCommandHistory() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.1.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion < "1.1.0" {
		return fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	_, err = d.db.Exec(`DELETE FROM command_history`)
	if err != nil {
		return err
	}
	return nil
}

// SearchHistory represents a search history entry
type SearchHistory struct {
	ID              int    `json:"id"`
	Command         string `json:"command"`
	Position        string `json:"position"`
	ExcludePosition string `json:"excludePosition"`
	Timestamp       int64  `json:"timestamp"`
}

// SaveSearchHistory saves a search command, its include position and its optional
// exclude ("Sauf") position to the search_history table.
func (d *Database) SaveSearchHistory(command string, position string, excludePosition string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

	// Insert the search history entry
	_, err := d.db.Exec(`INSERT INTO search_history (command, position, exclude_position, timestamp) VALUES (?, ?, ?, ?)`,
		command, position, excludePosition, time.Now().UnixMilli())
	if err != nil {
		return err
	}

	// Keep only the last 100 entries
	_, err = d.db.Exec(`
		DELETE FROM search_history 
		WHERE id NOT IN (
			SELECT id FROM search_history 
			ORDER BY timestamp DESC 
			LIMIT 100
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

// LoadSearchHistory loads the search history from the search_history table
func (d *Database) LoadSearchHistory() ([]SearchHistory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("database is not opened")
	}

	rows, err := d.db.Query(`SELECT id, command, position, exclude_position, timestamp FROM search_history ORDER BY timestamp DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []SearchHistory
	for rows.Next() {
		var entry SearchHistory
		var excludePosition sql.NullString
		err := rows.Scan(&entry.ID, &entry.Command, &entry.Position, &excludePosition, &entry.Timestamp)
		if err != nil {
			return nil, err
		}
		entry.ExcludePosition = excludePosition.String
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return history, nil
}

// DeleteSearchHistoryEntry deletes a search history entry by timestamp
func (d *Database) DeleteSearchHistoryEntry(timestamp int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

	_, err := d.db.Exec(`DELETE FROM search_history WHERE timestamp = ?`, timestamp)
	if err != nil {
		return err
	}

	return nil
}

// SessionState represents the last session state for restoring when reopening a database
type SessionState struct {
	LastSearchCommand  string  `json:"lastSearchCommand"`  // The last search command executed
	LastSearchPosition string  `json:"lastSearchPosition"` // The position used for the last search (JSON)
	LastPositionIndex  int     `json:"lastPositionIndex"`  // The index of the last viewed position in results
	LastPositionIDs    []int64 `json:"lastPositionIds"`    // The list of position IDs from the last search
	HasActiveSearch    bool    `json:"hasActiveSearch"`    // Whether there was an active search session
	ViewsJSON          string  `json:"viewsJSON"`          // Serialized view tabs state
}

// SaveSessionState saves the current session state to the metadata table
func (d *Database) SaveSessionState(state SessionState) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

	// Serialize position IDs to JSON
	positionIDsJSON, err := json.Marshal(state.LastPositionIDs)
	if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Save each session state field as a metadata entry
	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_last_search_command', ?)`, state.LastSearchCommand)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_last_search_position', ?)`, state.LastSearchPosition)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_last_position_index', ?)`, strconv.Itoa(state.LastPositionIndex))
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_last_position_ids', ?)`, string(positionIDsJSON))
	if err != nil {
		return err
	}

	hasActiveSearchStr := "false"
	if state.HasActiveSearch {
		hasActiveSearchStr = "true"
	}
	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_has_active_search', ?)`, hasActiveSearchStr)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('session_views', ?)`, state.ViewsJSON)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// LoadSessionState loads the last session state from the metadata table
func (d *Database) LoadSessionState() (*SessionState, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("database is not opened")
	}

	state := &SessionState{}

	// Load last search command
	var lastSearchCommand sql.NullString
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_last_search_command'`).Scan(&lastSearchCommand)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if lastSearchCommand.Valid {
		state.LastSearchCommand = lastSearchCommand.String
	}

	// Load last search position
	var lastSearchPosition sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_last_search_position'`).Scan(&lastSearchPosition)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if lastSearchPosition.Valid {
		state.LastSearchPosition = lastSearchPosition.String
	}

	// Load last position index
	var lastPositionIndex sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_last_position_index'`).Scan(&lastPositionIndex)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if lastPositionIndex.Valid {
		index, parseErr := strconv.Atoi(lastPositionIndex.String)
		if parseErr == nil {
			state.LastPositionIndex = index
		}
	}

	// Load last position IDs
	var lastPositionIDs sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_last_position_ids'`).Scan(&lastPositionIDs)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if lastPositionIDs.Valid && lastPositionIDs.String != "" {
		var ids []int64
		if parseErr := json.Unmarshal([]byte(lastPositionIDs.String), &ids); parseErr == nil {
			state.LastPositionIDs = ids
		}
	}

	// Load has active search flag
	var hasActiveSearch sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_has_active_search'`).Scan(&hasActiveSearch)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if hasActiveSearch.Valid {
		state.HasActiveSearch = hasActiveSearch.String == "true"
	}

	// Load views JSON
	var viewsJSON sql.NullString
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'session_views'`).Scan(&viewsJSON)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if viewsJSON.Valid {
		state.ViewsJSON = viewsJSON.String
	}

	return state, nil
}

// ClearSessionState clears the session state from the metadata table
func (d *Database) ClearSessionState() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

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
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, key := range sessionKeys {
		_, err := tx.Exec(`DELETE FROM metadata WHERE key = ?`, key)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *Database) Migrate_1_0_0_to_1_1_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion != "1.0.0" {
		return fmt.Errorf("database version is not 1.0.0, current version: %s", dbVersion)
	}

	// Create the command_history table
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

	// Update the database version to 1.1.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.1.0")
	if err != nil {
		return err
	}

	slog.Info("database migrated", "from", "1.0.0", "to", "1.1.0")
	return nil
}

func (d *Database) Migrate_1_1_0_to_1_2_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion != "1.1.0" {
		return fmt.Errorf("database version is not 1.1.0, current version: %s", dbVersion)
	}

	// Create the filter_library table
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

	// Update the database version to 1.2.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.2.0")
	if err != nil {
		return err
	}

	slog.Info("database migrated", "from", "1.1.0", "to", "1.2.0")
	return nil
}

func (d *Database) Migrate_1_2_0_to_1_3_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion != "1.2.0" {
		return fmt.Errorf("database version is not 1.2.0, current version: %s", dbVersion)
	}

	// Create the search_history table
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

	// Update the database version to 1.3.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.3.0")
	if err != nil {
		return err
	}

	slog.Info("database migrated", "from", "1.2.0", "to", "1.3.0")
	return nil
}

func (d *Database) SaveFilter(name, command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	// Check if a filter with the same name already exists
	var existingID int64
	err = d.db.QueryRow(`SELECT id FROM filter_library WHERE name = ?`, name).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingID > 0 {
		return fmt.Errorf("filter name already exists")
	}

	_, err = d.db.Exec(`INSERT INTO filter_library (name, command) VALUES (?, ?)`, name, command)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) UpdateFilter(id int64, name, command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	result, err := d.db.Exec(`UPDATE filter_library SET name = ?, command = ? WHERE id = ?`, name, command, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("filter with id %d not found", id)
	}
	return nil
}

func (d *Database) DeleteFilter(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	result, err := d.db.Exec(`DELETE FROM filter_library WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("filter with id %d not found", id)
	}
	return nil
}

func (d *Database) LoadFilters() ([]map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return nil, err
	}

	if dbVersion < "1.2.0" {
		return nil, fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	rows, err := d.db.Query(`SELECT id, name, command FROM filter_library`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, command string
		if err = rows.Scan(&id, &name, &command); err != nil {
			return nil, err
		}
		filters = append(filters, map[string]interface{}{
			"id":      id,
			"name":    name,
			"command": command,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return filters, nil
}

func (d *Database) SaveEditPosition(filterName, editPosition string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	// Check if a filter with the same name already exists
	var existingID int64
	err = d.db.QueryRow(`SELECT id FROM filter_library WHERE name = ?`, filterName).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingID > 0 {
		_, err = d.db.Exec(`UPDATE filter_library SET edit_position = ? WHERE id = ?`, editPosition, existingID)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("filter name does not exist")
	}

	return nil
}

func (d *Database) LoadEditPosition(filterName string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		return "", err
	}

	if dbVersion < "1.2.0" {
		return "", fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	var editPosition string
	err = d.db.QueryRow(`SELECT edit_position FROM filter_library WHERE name = ?`, filterName).Scan(&editPosition)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No edit position found
		}
		return "", err
	}
	return editPosition, nil
}

// SaveExcludePosition stores the "Sauf" (exclusion) structure of a saved filter.
// Mirrors SaveEditPosition but targets the exclude_position column.
func (d *Database) SaveExcludePosition(filterName, excludePosition string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM filter_library WHERE name = ?`, filterName).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if existingID > 0 {
		if _, err = d.db.Exec(`UPDATE filter_library SET exclude_position = ? WHERE id = ?`, excludePosition, existingID); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("filter name does not exist")
	}
	return nil
}

// LoadExcludePosition returns the "Sauf" (exclusion) structure of a saved filter,
// or "" when the filter has none.
func (d *Database) LoadExcludePosition(filterName string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var excludePosition sql.NullString
	err := d.db.QueryRow(`SELECT exclude_position FROM filter_library WHERE name = ?`, filterName).Scan(&excludePosition)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return excludePosition.String, nil
}
