package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync" // Import sync package
	"sync/atomic"
	"time"

	"github.com/kevung/xgparser/xgparser"
	_ "modernc.org/sqlite"
)

type Database struct {
	db              *sql.DB
	mu              sync.Mutex // Add a mutex to the Database struct
	importCancelled int32      // Flag to cancel ongoing import (atomic)
}

func NewDatabase() *Database {
	return &Database{}
}

func (d *Database) SetupDatabase(path string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	if d.db != nil {
		d.db.Close() // Close the currently opened database
	}

	// Open the database using string path
	var err error
	d.db, err = sql.Open("sqlite", path)
	if err != nil {
		fmt.Println("Error opening database:", err)
		return err
	}

	// Erase any content in the database
	_, err = d.db.Exec(`
		PRAGMA writable_schema = 1;
		DELETE FROM sqlite_master WHERE type IN ('table', 'index', 'trigger');
		PRAGMA writable_schema = 0;
		VACUUM;
		PRAGMA INTEGRITY_CHECK;
	`)
	if err != nil {
		fmt.Println("Error erasing database content:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS position (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            state TEXT
        )
    `)
	if err != nil {
		fmt.Println("Error creating position table:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS analysis (
            id INTEGER PRIMARY KEY,
            position_id INTEGER,
            data JSON,
            FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
        )
    `)
	if err != nil {
		fmt.Println("Error creating analysis table:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS comment (
            id INTEGER PRIMARY KEY,
            position_id INTEGER,
            text TEXT,
            FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
        )
    `)
	if err != nil {
		fmt.Println("Error creating comment table:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS metadata (
            key TEXT PRIMARY KEY,
            value TEXT
        )
    `)
	if err != nil {
		fmt.Println("Error creating metadata table:", err)
		return err
	}

	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS command_history (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            command TEXT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		fmt.Println("Error creating command_history table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS filter_library (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			command TEXT,
			edit_position TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating filter_library table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			position TEXT,
			timestamp INTEGER
		)
	`)
	if err != nil {
		fmt.Println("Error creating search_history table:", err)
		return err
	}

	// Create match-related tables for XG import (v1.4.0)
	_, err = d.db.Exec(`
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
			match_hash TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating match table:", err)
		return err
	}

	// Create index on match_hash for fast duplicate detection
	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
	if err != nil {
		fmt.Println("Error creating match_hash index:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER,
			game_number INTEGER,
			initial_score_1 INTEGER,
			initial_score_2 INTEGER,
			winner INTEGER,
			points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		fmt.Println("Error creating game table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS move (
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
		)
	`)
	if err != nil {
		fmt.Println("Error creating move table:", err)
		return err
	}

	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS move_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			move_id INTEGER,
			analysis_type TEXT,
			depth TEXT,
			equity REAL,
			equity_error REAL,
			win_rate REAL,
			gammon_rate REAL,
			backgammon_rate REAL,
			opponent_win_rate REAL,
			opponent_gammon_rate REAL,
			opponent_backgammon_rate REAL,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		fmt.Println("Error creating move_analysis table:", err)
		return err
	}

	// Insert or update the database version
	_, err = d.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		fmt.Println("Error inserting database version:", err)
		return err
	}

	return nil
}

func (d *Database) OpenDatabase(path string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	if d.db != nil {
		d.db.Close() // Close the currently opened database
	}

	// Open the database using string path
	var err error
	d.db, err = sql.Open("sqlite", path)
	if err != nil {
		fmt.Println("Error opening database:", err)
		return err
	}

	// Check the database version
	var dbVersion string
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	// Check if the required tables exist based on the database version
	requiredTables := []string{"position", "analysis", "comment", "metadata"}
	if dbVersion >= "1.1.0" {
		requiredTables = append(requiredTables, "command_history")
	}
	if dbVersion >= "1.2.0" {
		requiredTables = append(requiredTables, "filter_library")
	}
	if dbVersion >= "1.3.0" {
		requiredTables = append(requiredTables, "search_history")
	}
	if dbVersion >= "1.4.0" {
		requiredTables = append(requiredTables, "match", "game", "move", "move_analysis")
	}

	// Auto-migrate from 1.2.0 to 1.3.0
	if dbVersion == "1.2.0" {
		var searchHistoryExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='search_history'`).Scan(&searchHistoryExists)
		if err == sql.ErrNoRows {
			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS search_history (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					command TEXT,
					position TEXT,
					timestamp INTEGER
				)
			`)
			if err != nil {
				fmt.Println("Error creating search_history table:", err)
				return err
			}
			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.3.0")
			if err != nil {
				fmt.Println("Error updating database version:", err)
				return err
			}
			dbVersion = "1.3.0"
			fmt.Println("Database automatically upgraded from 1.2.0 to 1.3.0")
		}
	}

	// Auto-migrate from 1.3.0 to 1.4.0
	if dbVersion == "1.3.0" {
		var matchExists string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&matchExists)
		if err == sql.ErrNoRows {
			// Create match-related tables
			_, err = d.db.Exec(`
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
					match_hash TEXT
				)
			`)
			if err != nil {
				fmt.Println("Error creating match table:", err)
				return err
			}

			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS game (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					match_id INTEGER,
					game_number INTEGER,
					initial_score_1 INTEGER,
					initial_score_2 INTEGER,
					winner INTEGER,
					points_won INTEGER,
					move_count INTEGER DEFAULT 0,
					FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				fmt.Println("Error creating game table:", err)
				return err
			}

			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS move (
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
				)
			`)
			if err != nil {
				fmt.Println("Error creating move table:", err)
				return err
			}

			_, err = d.db.Exec(`
				CREATE TABLE IF NOT EXISTS move_analysis (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					move_id INTEGER,
					analysis_type TEXT,
					depth TEXT,
					equity REAL,
					equity_error REAL,
					win_rate REAL,
					gammon_rate REAL,
					backgammon_rate REAL,
					opponent_win_rate REAL,
					opponent_gammon_rate REAL,
					opponent_backgammon_rate REAL,
					FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				fmt.Println("Error creating move_analysis table:", err)
				return err
			}

			// Create index on match_hash for fast duplicate detection
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
			if err != nil {
				fmt.Println("Error creating match_hash index:", err)
				return err
			}

			_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.4.0")
			if err != nil {
				fmt.Println("Error updating database version:", err)
				return err
			}
			dbVersion = "1.4.0"
			fmt.Println("Database automatically upgraded from 1.3.0 to 1.4.0")
		}
	}

	// Add match_hash column to existing v1.4.0 databases if it doesn't exist
	if dbVersion == "1.4.0" {
		// Check if match_hash column exists
		var colInfo string
		err = d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='match'`).Scan(&colInfo)
		if err == nil && !strings.Contains(colInfo, "match_hash") {
			// Add match_hash column
			_, err = d.db.Exec(`ALTER TABLE match ADD COLUMN match_hash TEXT`)
			if err != nil {
				fmt.Println("Error adding match_hash column:", err)
				return err
			}

			// Create index on match_hash
			_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_match_hash ON match(match_hash)`)
			if err != nil {
				fmt.Println("Error creating match_hash index:", err)
				return err
			}

			// Populate match_hash for existing matches using stored data
			// This uses a fallback hash based on stored moves since we don't have the original file
			matchRows, err := d.db.Query(`SELECT id, player1_name, player2_name, match_length FROM match`)
			if err == nil {
				defer matchRows.Close()
				for matchRows.Next() {
					var matchID int64
					var p1Name, p2Name string
					var matchLength int32
					if err := matchRows.Scan(&matchID, &p1Name, &p2Name, &matchLength); err != nil {
						continue
					}
					hash := computeMatchHashFromStoredData(d.db, matchID, p1Name, p2Name, matchLength)
					_, _ = d.db.Exec(`UPDATE match SET match_hash = ? WHERE id = ?`, hash, matchID)
				}
			}

			fmt.Println("Added match_hash column and populated existing matches")
		}
	}

	for _, table := range requiredTables {
		var tableName string
		err = d.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&tableName)
		if err != nil {
			fmt.Printf("Error checking table %s: %v\n", table, err)
			return err
		}
		if tableName != table {
			return fmt.Errorf("required table %s does not exist", table)
		}
	}

	// Check if the required metadata keys exist
	requiredKeys := []string{"database_version"}
	for _, key := range requiredKeys {
		var value string
		err = d.db.QueryRow(`SELECT value FROM metadata WHERE key=?`, key).Scan(&value)
		if err != nil {
			fmt.Printf("Error checking metadata key %s: %v\n", key, err)
			return err
		}
		if value == "" {
			return fmt.Errorf("required metadata key %s does not exist", key)
		}
	}

	return nil
}

func (d *Database) CheckVersion(databaseVersion string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
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
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return "", err
	}
	return dbVersion, nil
}

func (d *Database) PositionExists(position Position) (map[string]interface{}, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Create a copy of the position without the ID field inside the state
	positionCopy := position
	positionCopy.ID = 0

	positionJSON, err := json.Marshal(positionCopy)
	if err != nil {
		fmt.Println("Error marshalling position:", err)
		return nil, err
	}

	rows, err := d.db.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error querying positions:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stateJSON string
		var positionID int64
		if err = rows.Scan(&positionID, &stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			return nil, err
		}

		var existingPosition Position
		if err = json.Unmarshal([]byte(stateJSON), &existingPosition); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			return nil, err
		}

		// Compare the positions excluding the ID field inside the state
		existingPosition.ID = 0
		existingPositionJSON, err := json.Marshal(existingPosition)
		if err != nil {
			fmt.Println("Error marshalling existing position:", err)
			return nil, err
		}

		if string(positionJSON) == string(existingPositionJSON) {
			return map[string]interface{}{"id": positionID, "exists": true}, nil
		}
	}

	return map[string]interface{}{"id": 0, "exists": false}, nil
}

func (d *Database) SavePosition(position *Position) (int64, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	positionJSON, err := json.Marshal(position)
	if err != nil {
		fmt.Println("Error marshalling position:", err)
		return 0, err
	}

	result, err := d.db.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
	if err != nil {
		fmt.Println("Error inserting position:", err)
		return 0, err
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		fmt.Println("Error getting last insert ID:", err)
		return 0, err
	}

	position.ID = positionID // Update the position ID

	// Update the state with the new ID
	positionJSON, err = json.Marshal(position)
	if err != nil {
		fmt.Println("Error marshalling position with ID:", err)
		return 0, err
	}

	_, err = d.db.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSON), positionID)
	if err != nil {
		fmt.Println("Error updating position with ID:", err)
		return 0, err
	}

	return positionID, nil
}

func (d *Database) UpdatePosition(position Position) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	positionJSON, err := json.Marshal(position)
	if err != nil {
		fmt.Println("Error marshalling position:", err)
		return err
	}

	_, err = d.db.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSON), position.ID)
	if err != nil {
		fmt.Println("Error updating position:", err)
		return err
	}

	return nil
}

func (d *Database) SaveAnalysis(positionID int64, analysis PositionAnalysis) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Ensure the positionID is set in the analysis
	analysis.PositionID = int(positionID)

	// Update last modified date
	analysis.LastModifiedDate = time.Now()

	// Check if an analysis already exists for the given position ID
	var existingID int64
	var existingAnalysisJSON string
	err := d.db.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisJSON)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error querying analysis:", err)
		return err
	}

	if existingID > 0 {
		// Preserve the existing creation date
		var existingAnalysis PositionAnalysis
		err = json.Unmarshal([]byte(existingAnalysisJSON), &existingAnalysis)
		if err != nil {
			fmt.Println("Error unmarshalling existing analysis:", err)
			return err
		}
		analysis.CreationDate = existingAnalysis.CreationDate

		// Update the existing analysis
		analysisJSON, err := json.Marshal(analysis)
		if err != nil {
			fmt.Println("Error marshalling analysis:", err)
			return err
		}
		_, err = d.db.Exec(`UPDATE analysis SET data = ? WHERE id = ?`, string(analysisJSON), existingID)
		if err != nil {
			fmt.Println("Error updating analysis:", err)
			return err
		}
	} else {
		// Set creation date if not already set
		if analysis.CreationDate.IsZero() {
			analysis.CreationDate = time.Now()
		}

		// Insert a new analysis
		analysisJSON, err := json.Marshal(analysis)
		if err != nil {
			fmt.Println("Error marshalling analysis:", err)
			return err
		}
		_, err = d.db.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, string(analysisJSON))
		if err != nil {
			fmt.Println("Error inserting analysis:", err)
			return err
		}
	}

	return nil
}

func (d *Database) LoadPosition(id int) (*Position, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var stateJSON string

	err := d.db.QueryRow(`SELECT state from position WHERE id = ?`, id).Scan(&stateJSON)
	if err != nil {
		fmt.Println("Error loading position:", err)
		return nil, err
	}

	var state Position
	err = json.Unmarshal([]byte(stateJSON), &state)
	if err != nil {
		fmt.Println("Error unmarshalling position:", err)
		return nil, err
	}

	return &state, nil
}

func (d *Database) LoadAnalysis(positionID int64) (*PositionAnalysis, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var analysisJSON string
	err := d.db.QueryRow(`SELECT data from analysis WHERE position_id = ?`, positionID).Scan(&analysisJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		fmt.Println("Error loading analysis:", err)
		return nil, err
	}

	var analysis PositionAnalysis
	err = json.Unmarshal([]byte(analysisJSON), &analysis)
	if err != nil {
		fmt.Println("Error unmarshalling analysis:", err)
		return nil, err
	}

	// Try to load the played move if this position is from a match
	var checkerMove sql.NullString
	var cubeAction sql.NullString
	err = d.db.QueryRow(`
		SELECT checker_move, cube_action 
		FROM move 
		WHERE position_id = ?
		LIMIT 1
	`, positionID).Scan(&checkerMove, &cubeAction)

	if err == nil {
		// Found a move record for this position
		if checkerMove.Valid {
			analysis.PlayedMove = checkerMove.String
		}
		if cubeAction.Valid {
			analysis.PlayedCubeAction = cubeAction.String
		}
	}
	// If no move found, that's okay - this position might not be from a match

	return &analysis, nil
}

func (d *Database) LoadAllPositions() ([]Position, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	rows, err := d.db.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error loading positions:", err)
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			return nil, err
		}

		var position Position
		if err = json.Unmarshal([]byte(stateJSON), &position); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			return nil, err
		}
		position.ID = id // Ensure the ID is set
		positions = append(positions, position)
	}

	if len(positions) == 0 {
		fmt.Println("No positions found, returning empty array.")
	}

	fmt.Println("Loaded positions:", positions)
	return positions, nil
}

func (d *Database) DeletePosition(positionID int64) error {

	// Delete the associated analysis first
	err := d.DeleteAnalysis(positionID)
	if err != nil {
		fmt.Println("Error deleting associated analysis:", err)
		return err
	}

	// Delete the associated comment
	err = d.DeleteComment(positionID)
	if err != nil {
		fmt.Println("Error deleting associated comment:", err)
		return err
	}

	d.mu.Lock() // Lock the mutex

	// Delete the position
	_, err = d.db.Exec(`DELETE FROM position WHERE id = ?`, positionID)
	if err != nil {
		fmt.Println("Error deleting position:", err)
		return err
	}

	// Check if the database is empty
	var count int
	err = d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&count)
	if err != nil {
		fmt.Println("Error counting positions:", err)
		return err
	}
	if count == 0 {
		fmt.Println("Database is empty.")
	}

	d.mu.Unlock() // Unlock the mutex when the function returns

	return nil
}

func (d *Database) DeleteAnalysis(positionID int64) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	_, err := d.db.Exec(`DELETE FROM analysis WHERE position_id = ?`, positionID)
	if err != nil {
		fmt.Println("Error deleting analysis:", err)
		return err
	}
	return nil
}

func (d *Database) DeleteComment(positionID int64) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	_, err := d.db.Exec(`DELETE FROM comment WHERE position_id = ?`, positionID)
	if err != nil {
		fmt.Println("Error deleting comment:", err)
		return err
	}
	return nil
}

// SaveComment saves a comment for a given position ID
func (d *Database) SaveComment(positionID int64, text string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Check if a comment already exists for the given position ID
	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM comment WHERE position_id = ?`, positionID).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error querying comment:", err)
		return err
	}

	if existingID > 0 {
		// Update the existing comment
		_, err = d.db.Exec(`UPDATE comment SET text = ? WHERE id = ?`, text, existingID)
		if err != nil {
			fmt.Println("Error updating comment:", err)
			return err
		}
	} else {
		// Insert a new comment
		_, err = d.db.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, positionID, text)
		if err != nil {
			fmt.Println("Error inserting comment:", err)
			return err
		}
	}

	return nil
}

// LoadComment loads a comment for a given position ID
func (d *Database) LoadComment(positionID int64) (string, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var text string
	err := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, positionID).Scan(&text)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No comment found
		}
		fmt.Println("Error loading comment:", err)
		return "", err
	}
	return text, nil
}

func (d *Database) LoadPositionsByFilters(
	filter Position,
	includeCube bool,
	includeScore bool,
	pipCountFilter string,
	winRateFilter string,
	gammonRateFilter string,
	backgammonRateFilter string,
	player2WinRateFilter string,
	player2GammonRateFilter string,
	player2BackgammonRateFilter string,
	player1CheckerOffFilter string,
	player2CheckerOffFilter string,
	player1BackCheckerFilter string,
	player2BackCheckerFilter string,
	player1CheckerInZoneFilter string,
	player2CheckerInZoneFilter string,
	searchText string,
	player1AbsolutePipCountFilter string,
	equityFilter string,
	decisionTypeFilter bool,
	diceRollFilter bool,
	movePatternFilter string,
	dateFilter string,
	player1OutfieldBlotFilter string,
	player2OutfieldBlotFilter string,
	player1JanBlotFilter string,
	player2JanBlotFilter string,
	noContactFilter bool,
	mirrorFilter bool,
) ([]Position, error) {
	d.mu.Lock()
	rows, err := d.db.Query(`SELECT id, state FROM position`)
	d.mu.Unlock()
	if err != nil {
		fmt.Println("Error loading positions:", err)
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			return nil, err
		}

		var position Position
		if err = json.Unmarshal([]byte(stateJSON), &position); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			return nil, err
		}
		position.ID = id // Ensure the ID is set

		fmt.Printf("Checking position ID: %d\n", position.ID) // Add logging

		// Function to check if a position matches all filters
		matchesFilters := func(pos Position) bool {
			return pos.MatchesCheckerPosition(filter) &&
				(!includeCube || pos.MatchesCubePosition(filter)) &&
				(!includeScore || pos.MatchesScorePosition(filter)) &&
				(!decisionTypeFilter || pos.MatchesDecisionType(filter)) &&
				(pipCountFilter == "" || pos.MatchesPipCountFilter(pipCountFilter)) &&
				(winRateFilter == "" || pos.MatchesWinRate(winRateFilter, d)) &&
				(gammonRateFilter == "" || pos.MatchesGammonRate(gammonRateFilter, d)) &&
				(backgammonRateFilter == "" || pos.MatchesBackgammonRate(backgammonRateFilter, d)) &&
				(player2WinRateFilter == "" || pos.MatchesPlayer2WinRate(player2WinRateFilter, d)) &&
				(player2GammonRateFilter == "" || pos.MatchesPlayer2GammonRate(player2GammonRateFilter, d)) &&
				(player2BackgammonRateFilter == "" || pos.MatchesPlayer2BackgammonRate(player2BackgammonRateFilter, d)) &&
				(player1CheckerOffFilter == "" || pos.MatchesPlayer1CheckerOff(player1CheckerOffFilter)) &&
				(player2CheckerOffFilter == "" || pos.MatchesPlayer2CheckerOff(player2CheckerOffFilter)) &&
				(player1BackCheckerFilter == "" || pos.MatchesPlayer1BackChecker(player1BackCheckerFilter)) &&
				(player2BackCheckerFilter == "" || pos.MatchesPlayer2BackChecker(player2BackCheckerFilter)) &&
				(player1CheckerInZoneFilter == "" || pos.MatchesPlayer1CheckerInZone(player1CheckerInZoneFilter)) &&
				(player2CheckerInZoneFilter == "" || pos.MatchesPlayer2CheckerInZone(player2CheckerInZoneFilter)) &&
				(searchText == "" || pos.MatchesSearchText(searchText, d)) &&
				(player1AbsolutePipCountFilter == "" || pos.MatchesPlayer1AbsolutePipCount(player1AbsolutePipCountFilter)) &&
				(equityFilter == "" || pos.MatchesEquityFilter(equityFilter, d)) &&
				(!diceRollFilter || pos.MatchesDiceRoll(filter)) &&
				(dateFilter == "" || pos.MatchesDateFilter(dateFilter, d)) &&
				(player1OutfieldBlotFilter == "" || pos.MatchesPlayer1OutfieldBlot(player1OutfieldBlotFilter)) &&
				(player2OutfieldBlotFilter == "" || pos.MatchesPlayer2OutfieldBlot(player2OutfieldBlotFilter)) &&
				(player1JanBlotFilter == "" || pos.MatchesPlayer1JanBlot(player1JanBlotFilter)) &&
				(player2JanBlotFilter == "" || pos.MatchesPlayer2JanBlot(player2JanBlotFilter)) &&
				(!noContactFilter || pos.MatchesNoContact())
		}

		// Check the original position
		if matchesFilters(position) {
			if movePatternFilter != "" {
				fmt.Printf("Checking move pattern filter: %s for position ID: %d\n", movePatternFilter, position.ID) // Add logging
				if position.MatchesMovePattern(movePatternFilter, d) {
					positions = append(positions, position)
				}
			} else {
				positions = append(positions, position)
			}
		} else if mirrorFilter {
			mirroredPosition := position.Mirror()
			if matchesFilters(mirroredPosition) {
				if movePatternFilter != "" {
					fmt.Printf("Checking move pattern filter: %s for mirrored position ID: %d\n", movePatternFilter, mirroredPosition.ID) // Add logging
					if mirroredPosition.MatchesMovePattern(movePatternFilter, d) {
						positions = append(positions, mirroredPosition)
					}
				} else {
					positions = append(positions, mirroredPosition)
				}
			}
		}
	}

	fmt.Println("Loaded positions by filters:", positions)
	return positions, nil
}

func (p *Position) MatchesDecisionType(filter Position) bool {
	return p.DecisionType == filter.DecisionType && p.PlayerOnRoll == filter.PlayerOnRoll
}

func (p *Position) MatchesSearchText(searchText string, d *Database) bool {
	comment, err := d.LoadComment(p.ID)
	if err != nil {
		fmt.Printf("Error loading comment for position ID: %d, error: %v\n", p.ID, err)
		return false
	}

	// Extract the keyword from the raw search text filter
	searchTextMatch := strings.Trim(searchText, ` t"'`)
	searchTextArray := strings.Split(strings.ToLower(searchTextMatch), ";")
	comment = strings.ToLower(comment)
	for _, text := range searchTextArray {
		if strings.Contains(comment, text) {
			return true
		}
	}
	return false
}

// Add MatchesPlayer1CheckerOff method to Position type
func (p *Position) MatchesPlayer1CheckerOff(filter string) bool {
	checkersOff := p.Board.Bearoff[0]

	if strings.HasPrefix(filter, "o>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersOff >= value
	} else if strings.HasPrefix(filter, "o<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersOff <= value
	} else if strings.HasPrefix(filter, "o") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'ox' means 'ox,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersOff >= minValue && checkersOff <= maxValue
	}
	return false
}

// Add MatchesPlayer2CheckerOff method to Position type
func (p *Position) MatchesPlayer2CheckerOff(filter string) bool {
	checkersOff := p.Board.Bearoff[1]

	if strings.HasPrefix(filter, "O>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersOff >= value
	} else if strings.HasPrefix(filter, "O<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersOff <= value
	} else if strings.HasPrefix(filter, "O") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Ox' means 'Ox,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersOff >= minValue && checkersOff <= maxValue
	}
	return false
}

// Add MatchesPlayer2BackgammonRate method to Position type
func (p *Position) MatchesPlayer2BackgammonRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var backgammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		backgammonRate = analysis.DoublingCubeAnalysis.OpponentBackgammonChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 2 Backgammon Rate: %f\n", p.ID, backgammonRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		backgammonRate = analysis.CheckerAnalysis.Moves[0].OpponentBackgammonChance
		fmt.Printf("Position ID: %d, Checker decision, Player 2 Backgammon Rate: %f\n", p.ID, backgammonRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no backgammon rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "B>") && !strings.HasPrefix(filter, "BO>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate >= value
	} else if strings.HasPrefix(filter, "B<") && !strings.HasPrefix(filter, "BO<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate <= value
	} else if strings.HasPrefix(filter, "B") && !strings.HasPrefix(filter, "BO") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backgammonRate >= minValue && backgammonRate <= maxValue
	}
	return false
}

// Add MatchesPlayer2GammonRate method to Position type
func (p *Position) MatchesPlayer2GammonRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var gammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		gammonRate = analysis.DoublingCubeAnalysis.OpponentGammonChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 2 Gammon Rate: %f\n", p.ID, gammonRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		gammonRate = analysis.CheckerAnalysis.Moves[0].OpponentGammonChance
		fmt.Printf("Position ID: %d, Checker decision, Player 2 Gammon Rate: %f\n", p.ID, gammonRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no gammon rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "G>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return gammonRate >= value
	} else if strings.HasPrefix(filter, "G<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return gammonRate <= value
	} else if strings.HasPrefix(filter, "G") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return gammonRate >= minValue && gammonRate <= maxValue
	}
	return false
}

// Add MatchesScorePosition method to Position type
func (p *Position) MatchesScorePosition(filter Position) bool {
	return p.Score[0] == filter.Score[0] && p.Score[1] == filter.Score[1]
}

// Add MatchesCubePosition method to Position type
func (p *Position) MatchesCubePosition(filter Position) bool {
	return p.Cube.Value == filter.Cube.Value && p.Cube.Owner == filter.Cube.Owner
}

// Add MatchesPipCountFilter method to Position type
func (p *Position) MatchesPipCountFilter(filter string) bool {
	pipCountDiff := p.PipCountDifference()
	player1PipCount, player2PipCount := p.ComputePipCounts()
	fmt.Printf("Checking pip count filter: %s, Player 1 Pip Count: %d, Player 2 Pip Count: %d, Pip count difference: %d\n", filter, player1PipCount, player2PipCount, pipCountDiff)
	if strings.HasPrefix(filter, "p>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return pipCountDiff >= value
	} else if strings.HasPrefix(filter, "p<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return pipCountDiff <= value
	} else if strings.HasPrefix(filter, "p") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return pipCountDiff >= minValue && pipCountDiff <= maxValue
	}
	return false
}

// Add MatchesWinRate method to Position type
func (p *Position) MatchesWinRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var winRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		winRate = analysis.DoublingCubeAnalysis.PlayerWinChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 1 Win Rate: %f\n", p.ID, winRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		winRate = analysis.CheckerAnalysis.Moves[0].PlayerWinChance
		fmt.Printf("Position ID: %d, Checker decision, Player 1 Win Rate: %f\n", p.ID, winRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no win rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "w>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return winRate >= value
	} else if strings.HasPrefix(filter, "w<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return winRate <= value
	} else if strings.HasPrefix(filter, "w") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return winRate >= minValue && winRate <= maxValue
	}
	return false
}

// Add MatchesPlayer2WinRate method to Position type
func (p *Position) MatchesPlayer2WinRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var winRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		winRate = analysis.DoublingCubeAnalysis.OpponentWinChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 2 Win Rate: %f\n", p.ID, winRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		winRate = analysis.CheckerAnalysis.Moves[0].OpponentWinChance
		fmt.Printf("Position ID: %d, Checker decision, Player 2 Win Rate: %f\n", p.ID, winRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no win rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "W>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return winRate >= value
	} else if strings.HasPrefix(filter, "W<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return winRate <= value
	} else if strings.HasPrefix(filter, "W") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return winRate >= minValue && winRate <= maxValue
	}
	return false
}

// Add MatchesGammonRate method to Position type
func (p *Position) MatchesGammonRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var gammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		gammonRate = analysis.DoublingCubeAnalysis.PlayerGammonChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 1 Gammon Rate: %f\n", p.ID, gammonRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		gammonRate = analysis.CheckerAnalysis.Moves[0].PlayerGammonChance
		fmt.Printf("Position ID: %d, Checker decision, Player 1 Gammon Rate: %f\n", p.ID, gammonRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no gammon rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "g>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return gammonRate >= value
	} else if strings.HasPrefix(filter, "g<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return gammonRate <= value
	} else if strings.HasPrefix(filter, "g") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return gammonRate >= minValue && gammonRate <= maxValue
	}
	return false
}

// Add MatchesBackgammonRate method to Position type
func (p *Position) MatchesBackgammonRate(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var backgammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		backgammonRate = analysis.DoublingCubeAnalysis.PlayerBackgammonChances
		fmt.Printf("Position ID: %d, Doubling decision, Player 1 Backgammon Rate: %f\n", p.ID, backgammonRate)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		backgammonRate = analysis.CheckerAnalysis.Moves[0].PlayerBackgammonChance
		fmt.Printf("Position ID: %d, Checker decision, Player 1 Backgammon Rate: %f\n", p.ID, backgammonRate)
	} else {
		fmt.Printf("Excluding position ID: %d due to no backgammon rate found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "b>") && !strings.HasPrefix(filter, "bo>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate >= value
	} else if strings.HasPrefix(filter, "b<") && !strings.HasPrefix(filter, "bo<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate <= value
	} else if strings.HasPrefix(filter, "b") && !strings.HasPrefix(filter, "bo") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backgammonRate >= minValue && backgammonRate <= maxValue
	}
	return false
}

// Add PipCountDifference method to Position type
func (p *Position) PipCountDifference() int {
	player1PipCount, player2PipCount := p.ComputePipCounts()
	return player1PipCount - player2PipCount
}

// Add ComputePipCounts method to Position type
func (p *Position) ComputePipCounts() (int, int) {
	player1PipCount := 0
	player2PipCount := 0

	for i, point := range p.Board.Points {
		if point.Color == 0 {
			player1PipCount += point.Checkers * i
		} else if point.Color == 1 {
			player2PipCount += point.Checkers * (25 - i)
		}
	}

	return player1PipCount, player2PipCount
}

// Add MatchesPlayer1BackChecker method to Position type with logging
func (p *Position) MatchesPlayer1BackChecker(filter string) bool {
	fmt.Printf("MatchesPlayer1BackChecker called with filter: %s\n", filter) // Add logging

	backCheckers := 0
	for i := 19; i <= 24; i++ {
		if p.Board.Points[i].Color == 0 {
			backCheckers += p.Board.Points[i].Checkers
		}
	}
	fmt.Printf("Checking back checkers filter: %s, Player 1 Back Checkers: %d\n", filter, backCheckers)

	if strings.HasPrefix(filter, "k>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backCheckers >= value
	} else if strings.HasPrefix(filter, "k<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backCheckers <= value
	} else if strings.HasPrefix(filter, "k") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'kx' means 'kx,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backCheckers >= minValue && backCheckers <= maxValue
	}
	return false
}

// Add MatchesPlayer2BackChecker method to Position type with logging
func (p *Position) MatchesPlayer2BackChecker(filter string) bool {
	fmt.Printf("MatchesPlayer2BackChecker called with filter: %s\n", filter) // Add logging

	backCheckers := 0
	for i := 1; i <= 6; i++ {
		if p.Board.Points[i].Color == 1 {
			backCheckers += p.Board.Points[i].Checkers
		}
	}
	fmt.Printf("Checking back checkers filter: %s, Player 2 Back Checkers: %d\n", filter, backCheckers)

	if strings.HasPrefix(filter, "K>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backCheckers >= value
	} else if strings.HasPrefix(filter, "K<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backCheckers <= value
	} else if strings.HasPrefix(filter, "K") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Kx' means 'Kx,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backCheckers >= minValue && backCheckers <= maxValue
	}
	return false
}

// Add MatchesPlayer1CheckerInZone method to Position type with logging
func (p *Position) MatchesPlayer1CheckerInZone(filter string) bool {
	fmt.Printf("MatchesPlayer1CheckerInZone called with filter: %s\n", filter) // Add logging

	checkersInZone := 0
	for i := 0; i <= 12; i++ {
		if p.Board.Points[i].Color == 0 {
			checkersInZone += p.Board.Points[i].Checkers
		}
	}
	fmt.Printf("Checking checkers in zone filter: %s, Player 1 Checkers in Zone: %d\n", filter, checkersInZone)

	if strings.HasPrefix(filter, "z>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersInZone >= value
	} else if strings.HasPrefix(filter, "z<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersInZone <= value
	} else if strings.HasPrefix(filter, "z") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'zx' means 'zx,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersInZone >= minValue && checkersInZone <= maxValue
	}
	return false
}

// Add MatchesPlayer2CheckerInZone method to Position type with logging
func (p *Position) MatchesPlayer2CheckerInZone(filter string) bool {
	fmt.Printf("MatchesPlayer2CheckerInZone called with filter: %s\n", filter) // Add logging

	checkersInZone := 0
	for i := 13; i <= 25; i++ {
		if p.Board.Points[i].Color == 1 {
			checkersInZone += p.Board.Points[i].Checkers
		}
	}
	fmt.Printf("Checking checkers in zone filter: %s, Player 2 Checkers in Zone: %d\n", filter, checkersInZone)

	if strings.HasPrefix(filter, "Z>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersInZone >= value
	} else if strings.HasPrefix(filter, "Z<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return checkersInZone <= value
	} else if strings.HasPrefix(filter, "Z") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			values = append(values, values[0]) // Handle case where 'Zx' means 'Zx,x'
		}
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return checkersInZone >= minValue && checkersInZone <= maxValue
	}
	return false
}

// Add MatchesPlayer1AbsolutePipCount method to Position type
func (p *Position) MatchesPlayer1AbsolutePipCount(filter string) bool {
	player1PipCount, _ := p.ComputePipCounts()

	if strings.HasPrefix(filter, "P>") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return player1PipCount >= value
	} else if strings.HasPrefix(filter, "P<") {
		value, err := strconv.Atoi(filter[2:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return player1PipCount <= value
	} else if strings.HasPrefix(filter, "P") {
		values := strings.Split(filter[1:], ",")
		if len(values) == 1 {
			value, err := strconv.Atoi(values[0])
			if err != nil {
				fmt.Printf("Error parsing filter value: %s\n", values[0])
				return false
			}
			return player1PipCount == value
		} else if len(values) == 2 {
			value1, err1 := strconv.Atoi(values[0])
			value2, err2 := strconv.Atoi(values[1])
			if err1 != nil || err2 != nil {
				fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
				return false
			}
			minValue := value1
			maxValue := value2
			if value1 > value2 {
				minValue = value2
				maxValue = value1
			}
			return player1PipCount >= minValue && player1PipCount <= maxValue
		}
	}
	return false
}

// Add MatchesEquityFilter method to Position type with detailed logging
func (p *Position) MatchesEquityFilter(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	var equity float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		equity = analysis.DoublingCubeAnalysis.CubefulNoDoubleEquity
		fmt.Printf("Position ID: %d, Doubling decision, Equity: %f\n", p.ID, equity)
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		equity = analysis.CheckerAnalysis.Moves[0].Equity
		fmt.Printf("Position ID: %d, Checker decision, Equity: %f\n", p.ID, equity)
	} else {
		fmt.Printf("Excluding position ID: %d due to no equity found\n", p.ID)
		return false
	}

	if strings.HasPrefix(filter, "e>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		value /= 1000 // Convert millipoints to points
		fmt.Printf("Equity filter condition: >, value: %f\n", value)
		return equity >= value
	} else if strings.HasPrefix(filter, "e<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		value /= 1000 // Convert millipoints to points
		fmt.Printf("Equity filter condition: <, value: %f\n", value)
		return equity <= value
	} else if strings.HasPrefix(filter, "e") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		value1 /= 1000 // Convert millipoints to points
		value2 /= 1000 // Convert millipoints to points
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		fmt.Printf("Equity filter condition: BETWEEN, values: %f, %f\n", minValue, maxValue)
		return equity >= minValue && equity <= maxValue
	}
	return false
}

// Add MatchesDiceRoll method to Position type
func (p *Position) MatchesDiceRoll(filter Position) bool {
	dice := fmt.Sprintf("%d%d", p.Dice[0], p.Dice[1])
	reverseDice := fmt.Sprintf("%d%d", p.Dice[1], p.Dice[0])
	filterDice := fmt.Sprintf("%d%d", filter.Dice[0], filter.Dice[1])
	return (dice == filterDice || reverseDice == filterDice) && p.PlayerOnRoll == filter.PlayerOnRoll && p.DecisionType == filter.DecisionType
}

func (p *Position) MatchesMovePattern(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	// Extract the move pattern from the raw string
	movePatternMatch := strings.Trim(filter, `m"'`)
	movePatterns := strings.Split(strings.ToLower(movePatternMatch), ";")

	if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		move := strings.ToLower(analysis.CheckerAnalysis.Moves[0].Move)
		for _, pattern := range movePatterns {
			if strings.Contains(move, pattern) {
				fmt.Printf("Position ID: %d, Checker decision, Move: %s, Filter: %s\n", p.ID, move, pattern) // Add logging
				return true
			}
		}
	} else if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		for _, pattern := range movePatterns {
			switch pattern {
			case "nd":
				if analysis.DoublingCubeAnalysis.CubefulNoDoubleError == 0 {
					fmt.Printf("Position ID: %d, Doubling decision, No Double Error: %f, Filter: %s\n", p.ID, analysis.DoublingCubeAnalysis.CubefulNoDoubleError, pattern) // Add logging
					return true
				}
			case "dt":
				if analysis.DoublingCubeAnalysis.CubefulDoubleTakeError == 0 {
					fmt.Printf("Position ID: %d, Doubling decision, Double Take Error: %f, Filter: %s\n", p.ID, analysis.DoublingCubeAnalysis.CubefulDoubleTakeError, pattern) // Add logging
					return true
				}
			case "dp":
				if analysis.DoublingCubeAnalysis.CubefulDoublePassError == 0 {
					fmt.Printf("Position ID: %d, Doubling decision, Double Pass Error: %f, Filter: %s\n", p.ID, analysis.DoublingCubeAnalysis.CubefulDoublePassError, pattern) // Add logging
					return true
				}
			}
		}
	}
	fmt.Printf("Position ID: %d does not match move pattern filter: %s\n", p.ID, filter) // Add logging
	return false
}

func (d *Database) GetDatabaseVersion() (string, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	return DatabaseVersion, nil
}

func (d *Database) LoadMetadata() (map[string]string, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	rows, err := d.db.Query(`SELECT key, value FROM metadata WHERE key IN ('user', 'description', 'dateOfCreation', 'database_version')`)
	if err != nil {
		fmt.Println("Error loading metadata:", err)
		return nil, err
	}
	defer rows.Close()

	metadata := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
			fmt.Println("Error scanning metadata:", err)
			return nil, err
		}
		metadata[key] = value
	}

	return metadata, nil
}

func (d *Database) SaveMetadata(metadata map[string]string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	for key, value := range metadata {
		_, err := d.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
		if err != nil {
			fmt.Println("Error saving metadata:", err)
			return err
		}
	}
	return nil
}

// Add MatchesDateFilter method to Position type
func (p *Position) MatchesDateFilter(filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		fmt.Printf("Excluding position ID: %d due to error: %v\n", p.ID, err)
		return false
	}

	creationDate := analysis.CreationDate
	fmt.Printf("Position ID: %d, Creation Date: %s\n", p.ID, creationDate)

	if strings.HasPrefix(filter, "T>") {
		dateStr := filter[2:]
		date, err := time.ParseInLocation("2006/01/02", dateStr, creationDate.Location())
		if err != nil {
			fmt.Printf("Error parsing date filter value: %s\n", dateStr)
			return false
		}
		fmt.Printf("Filter: T>, Date: %s\n", date)
		match := creationDate.After(date) || creationDate.Equal(date)
		fmt.Printf("Position ID: %d, Matches: %v\n", p.ID, match)
		return match
	} else if strings.HasPrefix(filter, "T<") {
		dateStr := filter[2:]
		date, err := time.ParseInLocation("2006/01/02", dateStr, creationDate.Location())
		if err != nil {
			fmt.Printf("Error parsing date filter value: %s\n", dateStr)
			return false
		}
		date = date.Add(24 * time.Hour).Add(-1 * time.Second) // Include the entire day
		fmt.Printf("Filter: T<, Date: %s\n", date)
		match := creationDate.Before(date)
		fmt.Printf("Position ID: %d, Matches: %v\n", p.ID, match)
		return match
	} else if strings.HasPrefix(filter, "T") {
		dateRange := strings.Split(filter[1:], ",")
		if len(dateRange) != 2 {
			fmt.Printf("Error parsing date range filter values: %s\n", filter[1:])
			return false
		}
		startDate, err1 := time.ParseInLocation("2006/01/02", dateRange[0], creationDate.Location())
		endDate, err2 := time.ParseInLocation("2006/01/02", dateRange[1], creationDate.Location())
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing date range filter values: %s, %s\n", dateRange[0], dateRange[1])
			return false
		}
		if startDate.After(endDate) {
			startDate, endDate = endDate, startDate // Swap to ensure correct order
		}
		endDate = endDate.Add(24 * time.Hour).Add(-1 * time.Second) // Include the entire day
		fmt.Printf("Filter: T, Start Date: %s, End Date: %s\n", startDate, endDate)
		match := (creationDate.After(startDate) || creationDate.Equal(startDate)) && (creationDate.Before(endDate) || creationDate.Equal(endDate))
		fmt.Printf("Position ID: %d, Matches: %v\n", p.ID, match)
		return match
	}
	return false
}

// Add MatchesPlayer1OutfieldBlot method to Position type
func (p *Position) MatchesPlayer1OutfieldBlot(filter string) bool {
	outfieldBlots := 0
	for i := 7; i <= 18; i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers == 1 {
			outfieldBlots++
		}
	}

	if strings.HasPrefix(filter, "bo>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return outfieldBlots >= value
	} else if strings.HasPrefix(filter, "bo<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return outfieldBlots <= value
	} else if strings.HasPrefix(filter, "bo") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[2:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return outfieldBlots >= minValue && outfieldBlots <= maxValue
	}
	return false
}

// Add MatchesPlayer2OutfieldBlot method to Position type
func (p *Position) MatchesPlayer2OutfieldBlot(filter string) bool {
	opponentOutfieldBlots := 0
	for i := 7; i <= 18; i++ {
		if p.Board.Points[i].Color == 1 && p.Board.Points[i].Checkers == 1 {
			opponentOutfieldBlots++
		}
	}

	if strings.HasPrefix(filter, "BO>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return opponentOutfieldBlots >= value
	} else if strings.HasPrefix(filter, "BO<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return opponentOutfieldBlots <= value
	} else if strings.HasPrefix(filter, "BO") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[2:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return opponentOutfieldBlots >= minValue && opponentOutfieldBlots <= maxValue
	}
	return false
}

// Add MatchesPlayer1JanBlot method to Position type
func (p *Position) MatchesPlayer1JanBlot(filter string) bool {
	janBlots := 0
	for i := 1; i <= 6; i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers == 1 {
			janBlots++
		}
	}

	if strings.HasPrefix(filter, "bj>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return janBlots >= value
	} else if strings.HasPrefix(filter, "bj<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return janBlots <= value
	} else if strings.HasPrefix(filter, "bj") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[2:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return janBlots >= minValue && janBlots <= maxValue
	}
	return false
}

// Add MatchesPlayer2JanBlot method to Position type
func (p *Position) MatchesPlayer2JanBlot(filter string) bool {
	opponentJanBlots := 0
	for i := 19; i <= 24; i++ {
		if p.Board.Points[i].Color == 1 && p.Board.Points[i].Checkers == 1 {
			opponentJanBlots++
		}
	}

	if strings.HasPrefix(filter, "BJ>") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return opponentJanBlots >= value
	} else if strings.HasPrefix(filter, "BJ<") {
		value, err := strconv.Atoi(filter[3:])
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[3:])
			return false
		}
		return opponentJanBlots <= value
	} else if strings.HasPrefix(filter, "BJ") {
		values := strings.Split(filter[2:], ",")
		if len(values) != 2 {
			fmt.Printf("Error parsing filter values: %s\n", filter[2:])
			return false
		}
		value1, err1 := strconv.Atoi(values[0])
		value2, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Error parsing filter values: %s, %s\n", values[0], values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return opponentJanBlots >= minValue && opponentJanBlots <= maxValue
	}
	return false
}

// Add MatchesNoContact method to Position type
func (p *Position) MatchesNoContact() bool {
	var furthestPlayerChecker, furthestOpponentChecker int

	// Initialize to invalid indices
	furthestPlayerChecker = -1
	furthestOpponentChecker = 26

	for i := 0; i < len(p.Board.Points); i++ {
		if p.Board.Points[i].Color == 0 && p.Board.Points[i].Checkers > 0 {
			furthestPlayerChecker = i
		}
		if p.Board.Points[25-i].Color == 1 && p.Board.Points[25-i].Checkers > 0 {
			furthestOpponentChecker = 25 - i
		}
	}

	// Compare indices to determine if there is no contact
	return furthestPlayerChecker < furthestOpponentChecker
}

func (p *Position) MatchesMirrorPosition(filter Position) bool {
	mirroredPosition := p.Mirror()
	return p.MatchesCheckerPosition(filter) || mirroredPosition.MatchesCheckerPosition(filter)
}

// Mirror creates a mirrored version of the current Position.
// It reverses the board points, swaps the bearoff positions,
// changes the player on roll, swaps the scores, and changes the cube owner.
// Returns the mirrored Position.
func (p *Position) Mirror() Position {
	mirrored := *p
	for i, point := range p.Board.Points {
		mirrored.Board.Points[25-i] = Point{
			Color:    point.Color,
			Checkers: point.Checkers,
		}
		if point.Color != -1 {
			mirrored.Board.Points[25-i].Color = 1 - point.Color
		}
	}
	mirrored.Board.Bearoff[0], mirrored.Board.Bearoff[1] = p.Board.Bearoff[1], p.Board.Bearoff[0]
	mirrored.PlayerOnRoll = 1 - p.PlayerOnRoll
	mirrored.Score[0], mirrored.Score[1] = p.Score[1], p.Score[0]
	if p.Cube.Owner != -1 {
		mirrored.Cube.Owner = 1 - p.Cube.Owner
	}
	return mirrored
}

// SaveCommand saves a command to the command_history table
func (d *Database) SaveCommand(command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.1.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.1.0" {
		return fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	_, err = d.db.Exec(`INSERT INTO command_history (command) VALUES (?)`, command)
	if err != nil {
		fmt.Println("Error saving command:", err)
		return err
	}
	return nil
}

// LoadCommandHistory loads the command history from the command_history table
func (d *Database) LoadCommandHistory() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.1.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return nil, err
	}

	if dbVersion < "1.1.0" {
		return nil, fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	rows, err := d.db.Query(`SELECT command FROM command_history ORDER BY timestamp ASC`)
	if err != nil {
		fmt.Println("Error loading command history:", err)
		return nil, err
	}
	defer rows.Close()

	var history []string
	for rows.Next() {
		var command string
		if err = rows.Scan(&command); err != nil {
			fmt.Println("Error scanning command:", err)
			return nil, err
		}
		history = append(history, command)
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
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.1.0" {
		return fmt.Errorf("database version is lower than 1.1.0, current version: %s", dbVersion)
	}

	_, err = d.db.Exec(`DELETE FROM command_history`)
	if err != nil {
		fmt.Println("Error clearing command history:", err)
		return err
	}
	return nil
}

// SearchHistory represents a search history entry
type SearchHistory struct {
	ID        int    `json:"id"`
	Command   string `json:"command"`
	Position  string `json:"position"`
	Timestamp int64  `json:"timestamp"`
}

// SaveSearchHistory saves a search command and position to the search_history table
func (d *Database) SaveSearchHistory(command string, position string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database is not opened")
	}

	// Insert the search history entry
	_, err := d.db.Exec(`INSERT INTO search_history (command, position, timestamp) VALUES (?, ?, ?)`,
		command, position, time.Now().UnixMilli())
	if err != nil {
		fmt.Println("Error saving search history:", err)
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
		fmt.Println("Error pruning search history:", err)
		return err
	}

	return nil
}

// LoadSearchHistory loads the search history from the search_history table
func (d *Database) LoadSearchHistory() ([]SearchHistory, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("database is not opened")
	}

	rows, err := d.db.Query(`SELECT id, command, position, timestamp FROM search_history ORDER BY timestamp DESC LIMIT 100`)
	if err != nil {
		fmt.Println("Error loading search history:", err)
		return nil, err
	}
	defer rows.Close()

	var history []SearchHistory
	for rows.Next() {
		var entry SearchHistory
		err := rows.Scan(&entry.ID, &entry.Command, &entry.Position, &entry.Timestamp)
		if err != nil {
			fmt.Println("Error scanning search history row:", err)
			return nil, err
		}
		history = append(history, entry)
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
		fmt.Println("Error deleting search history entry:", err)
		return err
	}

	return nil
}

func (d *Database) Migrate_1_0_0_to_1_1_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
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
		fmt.Println("Error creating command_history table:", err)
		return err
	}

	// Update the database version to 1.1.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.1.0")
	if err != nil {
		fmt.Println("Error updating database version:", err)
		return err
	}

	fmt.Println("Database successfully migrated from version 1.0.0 to 1.1.0")
	return nil
}

func (d *Database) Migrate_1_1_0_to_1_2_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
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
		fmt.Println("Error creating filter_library table:", err)
		return err
	}

	// Update the database version to 1.2.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.2.0")
	if err != nil {
		fmt.Println("Error updating database version:", err)
		return err
	}

	fmt.Println("Database successfully migrated from version 1.1.0 to 1.2.0")
	return nil
}

func (d *Database) Migrate_1_2_0_to_1_3_0() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check current database version
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
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
		fmt.Println("Error creating search_history table:", err)
		return err
	}

	// Update the database version to 1.3.0
	_, err = d.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, "1.3.0")
	if err != nil {
		fmt.Println("Error updating database version:", err)
		return err
	}

	fmt.Println("Database successfully migrated from version 1.2.0 to 1.3.0")
	return nil
}

func (d *Database) SaveFilter(name, command string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	// Check if a filter with the same name already exists
	var existingID int64
	err = d.db.QueryRow(`SELECT id FROM filter_library WHERE name = ?`, name).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error checking existing filter:", err)
		return err
	}
	if existingID > 0 {
		return fmt.Errorf("filter name already exists")
	}

	_, err = d.db.Exec(`INSERT INTO filter_library (name, command) VALUES (?, ?)`, name, command)
	if err != nil {
		fmt.Println("Error saving filter:", err)
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
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	_, err = d.db.Exec(`UPDATE filter_library SET name = ?, command = ? WHERE id = ?`, name, command, id)
	if err != nil {
		fmt.Println("Error updating filter:", err)
		return err
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
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	_, err = d.db.Exec(`DELETE FROM filter_library WHERE id = ?`, id)
	if err != nil {
		fmt.Println("Error deleting filter:", err)
		return err
	}
	return nil
}

func (d *Database) LoadFilters() ([]map[string]interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return nil, err
	}

	if dbVersion < "1.2.0" {
		return nil, fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	rows, err := d.db.Query(`SELECT id, name, command FROM filter_library`)
	if err != nil {
		fmt.Println("Error loading filters:", err)
		return nil, err
	}
	defer rows.Close()

	var filters []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, command string
		if err = rows.Scan(&id, &name, &command); err != nil {
			fmt.Println("Error scanning filter:", err)
			return nil, err
		}
		filters = append(filters, map[string]interface{}{
			"id":      id,
			"name":    name,
			"command": command,
		})
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
		fmt.Println("Error querying database version:", err)
		return err
	}

	if dbVersion < "1.2.0" {
		return fmt.Errorf("database version is lower than 1.2.0, current version: %s", dbVersion)
	}

	// Check if a filter with the same name already exists
	var existingID int64
	err = d.db.QueryRow(`SELECT id FROM filter_library WHERE name = ?`, filterName).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error checking existing filter:", err)
		return err
	}
	if existingID > 0 {
		_, err = d.db.Exec(`UPDATE filter_library SET edit_position = ? WHERE id = ?`, editPosition, existingID)
		if err != nil {
			fmt.Println("Error updating edit position:", err)
			return err
		}
	} else {
		return fmt.Errorf("filter name does not exist")
	}

	return nil
}

func (d *Database) LoadEditPosition(filterName string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if the database version is 1.2.0 or higher
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
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
		fmt.Println("Error loading edit position:", err)
		return "", err
	}
	return editPosition, nil
}

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
		fmt.Println("Error opening import database:", err)
		return nil, err
	}
	defer importDB.Close()

	// Check the import database version
	var importDBVersion string
	err = importDB.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&importDBVersion)
	if err != nil {
		fmt.Println("Error querying import database version:", err)
		return nil, fmt.Errorf("import database is invalid or missing version information")
	}

	// Check the current database version
	var currentDBVersion string
	err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&currentDBVersion)
	if err != nil {
		fmt.Println("Error querying current database version:", err)
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
		fmt.Println("Error counting positions:", err)
		return nil, err
	}

	// OPTIMIZATION: Build a hash map of all current positions ONCE
	// This converts O(n²) to O(n) complexity
	currentPositionsMap := make(map[string]int64) // map[positionJSON]positionID
	currentRows, err := d.db.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error querying current database positions:", err)
		return nil, err
	}

	for currentRows.Next() {
		var currentID int64
		var currentStateJSON string
		if err = currentRows.Scan(&currentID, &currentStateJSON); err != nil {
			continue
		}

		var currentPosition Position
		if err = json.Unmarshal([]byte(currentStateJSON), &currentPosition); err != nil {
			continue
		}

		// Reset ID for comparison
		currentPosition.ID = 0
		currentPositionJSON, _ := json.Marshal(currentPosition)
		currentPositionsMap[string(currentPositionJSON)] = currentID
	}
	currentRows.Close()

	fmt.Printf("Built index of %d positions in current database\n", len(currentPositionsMap))

	// Analyze what would happen
	rows, err := importDB.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error loading positions from import database:", err)
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
			fmt.Println("Error scanning position:", err)
			positionsToSkip++
			continue
		}

		var importPosition Position
		if err = json.Unmarshal([]byte(stateJSON), &importPosition); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			positionsToSkip++
			continue
		}

		// Reset ID for existence check
		importPosition.ID = 0
		importPositionJSON, _ := json.Marshal(importPosition)

		// OPTIMIZATION: O(1) hash map lookup instead of nested loop
		existingPositionID, existsInCurrent := currentPositionsMap[string(importPositionJSON)]

		if existsInCurrent {
			// Check if there's actually something to merge
			hasNewData := false

			// Check for analysis to merge
			var importAnalysisJSON string
			err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisJSON)
			if err == nil {
				var existingAnalysisJSON string
				existingErr := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, existingPositionID).Scan(&existingAnalysisJSON)

				if existingErr == sql.ErrNoRows {
					// New analysis to add
					hasNewData = true
				} else if existingErr == nil {
					// Check if import has better analysis
					var existingAnalysis PositionAnalysis
					var importAnalysis PositionAnalysis
					json.Unmarshal([]byte(existingAnalysisJSON), &existingAnalysis)
					json.Unmarshal([]byte(importAnalysisJSON), &importAnalysis)

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

	result := map[string]interface{}{
		"toAdd":      positionsToAdd,
		"toMerge":    positionsToMerge,
		"toSkip":     positionsToSkip,
		"total":      totalPositions,
		"importPath": importPath,
	}

	fmt.Printf("Import analysis: %d to add, %d to merge, %d to skip out of %d total\n", positionsToAdd, positionsToMerge, positionsToSkip, totalPositions)
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
		fmt.Println("Error starting transaction:", err)
		return nil, err
	}

	// Ensure rollback on error or cancellation
	defer func() {
		if err != nil || d.isImportCancelled() {
			tx.Rollback()
			if d.isImportCancelled() {
				fmt.Println("Transaction rolled back due to user cancellation")
			} else {
				fmt.Println("Transaction rolled back due to error")
			}
		}
	}()

	// Open the import database
	importDB, err := sql.Open("sqlite", importPath)
	if err != nil {
		fmt.Println("Error opening import database:", err)
		return nil, err
	}
	defer importDB.Close()

	// Check the import database version
	var importDBVersion string
	err = importDB.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&importDBVersion)
	if err != nil {
		fmt.Println("Error querying import database version:", err)
		return nil, fmt.Errorf("import database is invalid or missing version information")
	}

	// Check the current database version
	var currentDBVersion string
	err = tx.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&currentDBVersion)
	if err != nil {
		fmt.Println("Error querying current database version:", err)
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
		fmt.Println("Error counting positions:", err)
		return nil, err
	}

	// OPTIMIZATION: Build a hash map of all current positions ONCE
	// This converts O(n²) to O(n) complexity
	currentPositionsMap := make(map[string]int64) // map[positionJSON]positionID
	currentRows, err := tx.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error querying current database positions:", err)
		return nil, err
	}

	for currentRows.Next() {
		var currentID int64
		var currentStateJSON string
		if err = currentRows.Scan(&currentID, &currentStateJSON); err != nil {
			continue
		}

		var currentPosition Position
		if err = json.Unmarshal([]byte(currentStateJSON), &currentPosition); err != nil {
			continue
		}

		// Reset ID for comparison
		currentPosition.ID = 0
		currentPositionJSON, _ := json.Marshal(currentPosition)
		currentPositionsMap[string(currentPositionJSON)] = currentID
	}
	currentRows.Close()

	fmt.Printf("Built index of %d positions in current database for commit\n", len(currentPositionsMap))

	// Load all positions from the import database
	rows, err := importDB.Query(`SELECT id, state FROM position`)
	if err != nil {
		fmt.Println("Error loading positions from import database:", err)
		return nil, err
	}
	defer rows.Close()

	var positionsAdded int
	var positionsMerged int
	var positionsSkipped int

	for rows.Next() {
		// Check for cancellation
		if d.isImportCancelled() {
			fmt.Println("Import cancelled by user during processing")
			return nil, fmt.Errorf("import cancelled by user")
		}

		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			continue
		}

		var importPosition Position
		if err = json.Unmarshal([]byte(stateJSON), &importPosition); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			continue
		}

		// Reset ID for existence check
		importPosition.ID = 0
		importPositionJSON, _ := json.Marshal(importPosition)

		// OPTIMIZATION: O(1) hash map lookup instead of nested loop
		existingPositionID, existsInCurrent := currentPositionsMap[string(importPositionJSON)]

		if existsInCurrent {
			// Track if we actually merge anything
			hasMerged := false

			// Merge analysis if it exists
			var importAnalysisJSON string
			err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisJSON)

			if err == nil {
				// Load existing analysis from current database (using transaction)
				var existingAnalysisJSON string
				existingErr := tx.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, existingPositionID).Scan(&existingAnalysisJSON)

				if existingErr == sql.ErrNoRows {
					// No existing analysis, insert the imported one
					_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, existingPositionID, importAnalysisJSON)
					if err != nil {
						fmt.Printf("Error inserting analysis for position %d: %v\n", existingPositionID, err)
					} else {
						hasMerged = true
					}
				} else if existingErr == nil {
					// Both have analysis - keep the existing one unless it's empty
					var existingAnalysis PositionAnalysis
					var importAnalysis PositionAnalysis

					json.Unmarshal([]byte(existingAnalysisJSON), &existingAnalysis)
					json.Unmarshal([]byte(importAnalysisJSON), &importAnalysis)

					// If import has analysis but existing doesn't, use import
					if existingAnalysis.AnalysisType == "" && importAnalysis.AnalysisType != "" {
						_, err = tx.Exec(`UPDATE analysis SET data = ? WHERE position_id = ?`, importAnalysisJSON, existingPositionID)
						if err != nil {
							fmt.Printf("Error updating analysis for position %d: %v\n", existingPositionID, err)
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
						fmt.Printf("Error inserting comment for position %d: %v\n", existingPositionID, err)
					} else {
						hasMerged = true
					}
				} else if existingErr == nil {
					// Merge comments - only add if not already present
					if trimmedImport != "" && !strings.Contains(trimmedExisting, trimmedImport) {
						mergedComment := trimmedExisting
						if trimmedExisting != "" {
							mergedComment = trimmedExisting + "\n\n" + trimmedImport
						} else {
							mergedComment = trimmedImport
						}
						_, err = tx.Exec(`UPDATE comment SET text = ? WHERE position_id = ?`, mergedComment, existingPositionID)
						if err != nil {
							fmt.Printf("Error updating comment for position %d: %v\n", existingPositionID, err)
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
			result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, stateJSON)
			if err != nil {
				fmt.Println("Error inserting position:", err)
				positionsSkipped++
				continue
			}

			newPositionID, err := result.LastInsertId()
			if err != nil {
				fmt.Println("Error getting last insert ID:", err)
				positionsSkipped++
				continue
			}

			// Update the position ID in the state JSON
			importPosition.ID = newPositionID
			updatedStateJSON, _ := json.Marshal(importPosition)
			_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(updatedStateJSON), newPositionID)
			if err != nil {
				fmt.Println("Error updating position with ID:", err)
			}

			// Copy analysis if it exists
			var importAnalysisJSON string
			err = importDB.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, id).Scan(&importAnalysisJSON)
			if err == nil {
				// Update position_id in the analysis JSON
				var analysis PositionAnalysis
				json.Unmarshal([]byte(importAnalysisJSON), &analysis)
				analysis.PositionID = int(newPositionID)
				updatedAnalysisJSON, _ := json.Marshal(analysis)

				_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newPositionID, string(updatedAnalysisJSON))
				if err != nil {
					fmt.Printf("Error inserting analysis for new position %d: %v\n", newPositionID, err)
				}
			}

			// Copy comment if it exists
			var importComment string
			err = importDB.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, id).Scan(&importComment)
			if err == nil && importComment != "" {
				_, err = tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newPositionID, importComment)
				if err != nil {
					fmt.Printf("Error inserting comment for new position %d: %v\n", newPositionID, err)
				}
			}

			positionsAdded++
		}
	}

	// Final check for cancellation before committing
	if d.isImportCancelled() {
		fmt.Println("Import cancelled by user before commit")
		return nil, fmt.Errorf("import cancelled by user")
	}

	// Commit the transaction - this makes all changes atomic
	err = tx.Commit()
	if err != nil {
		fmt.Println("Error committing transaction:", err)
		return nil, err
	}

	result := map[string]interface{}{
		"added":   positionsAdded,
		"merged":  positionsMerged,
		"skipped": positionsSkipped,
		"total":   totalPositions,
	}

	fmt.Printf("Import committed: %d added, %d merged, %d skipped out of %d total\n", positionsAdded, positionsMerged, positionsSkipped, totalPositions)
	return result, nil
}

// CancelImport sets the flag to cancel any ongoing import operation
func (d *Database) CancelImport() {
	atomic.StoreInt32(&d.importCancelled, 1)
	fmt.Println("Import cancellation requested")
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

// ExportDatabase creates a new database file containing the current selection of positions
// with their analysis and comments
func (d *Database) ExportDatabase(exportPath string, positions []Position, metadata map[string]string, includeAnalysis bool, includeComments bool, includeFilterLibrary bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check that the current database is open
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Delete the export file if it already exists
	if _, err := os.Stat(exportPath); err == nil {
		// File exists, remove it
		if err := os.Remove(exportPath); err != nil {
			return fmt.Errorf("cannot remove existing export file: %v", err)
		}
		fmt.Printf("Removed existing export file: %s\n", exportPath)
	}

	// Create a new database for export
	exportDB, err := sql.Open("sqlite", exportPath)
	if err != nil {
		fmt.Println("Error creating export database:", err)
		return err
	}
	defer exportDB.Close()

	// Create the schema for the export database
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating position table in export database:", err)
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
		fmt.Println("Error creating analysis table in export database:", err)
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
		fmt.Println("Error creating comment table in export database:", err)
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		fmt.Println("Error creating metadata table in export database:", err)
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
		fmt.Println("Error creating command_history table in export database:", err)
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
		fmt.Println("Error creating filter_library table in export database:", err)
		return err
	}

	// Insert database version
	_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		fmt.Println("Error inserting database version in export database:", err)
		return err
	}

	// Insert metadata (user, description, dateOfCreation)
	for key, value := range metadata {
		if value != "" {
			_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
			if err != nil {
				fmt.Printf("Error inserting metadata %s in export database: %v\n", key, err)
			}
		}
	}

	// If dateOfCreation is not provided, set it to current date
	if metadata["dateOfCreation"] == "" {
		currentDate := time.Now().Format("2006-01-02")
		_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('dateOfCreation', ?)`, currentDate)
		if err != nil {
			fmt.Println("Error inserting default creation date in export database:", err)
		}
	}

	// Begin transaction for export
	tx, err := exportDB.Begin()
	if err != nil {
		fmt.Println("Error starting transaction for export:", err)
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			fmt.Println("Transaction rolled back due to error during export")
		}
	}()

	// Export all positions with their analysis and comments
	idMapping := make(map[int64]int64) // map old position ID to new position ID

	for _, position := range positions {
		oldPositionID := position.ID

		// Reset the ID for the new database
		position.ID = 0

		// Marshal the position
		positionJSON, err := json.Marshal(position)
		if err != nil {
			fmt.Printf("Error marshalling position %d: %v\n", oldPositionID, err)
			continue
		}

		// Insert the position into the export database
		result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
		if err != nil {
			fmt.Printf("Error inserting position %d into export database: %v\n", oldPositionID, err)
			continue
		}

		newPositionID, err := result.LastInsertId()
		if err != nil {
			fmt.Printf("Error getting last insert ID for position %d: %v\n", oldPositionID, err)
			continue
		}

		// Update the position ID in the state JSON
		position.ID = newPositionID
		updatedPositionJSON, _ := json.Marshal(position)
		_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(updatedPositionJSON), newPositionID)
		if err != nil {
			fmt.Printf("Error updating position %d with new ID: %v\n", newPositionID, err)
		}

		// Store the ID mapping
		idMapping[oldPositionID] = newPositionID

		// Export analysis if it exists and if includeAnalysis is true
		if includeAnalysis {
			var analysisJSON string
			analysisErr := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, oldPositionID).Scan(&analysisJSON)
			if analysisErr == nil {
				// Update position_id in the analysis JSON
				var analysis PositionAnalysis
				if unmarshalErr := json.Unmarshal([]byte(analysisJSON), &analysis); unmarshalErr == nil {
					analysis.PositionID = int(newPositionID)
					updatedAnalysisJSON, _ := json.Marshal(analysis)

					if _, insertErr := tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newPositionID, string(updatedAnalysisJSON)); insertErr != nil {
						fmt.Printf("Error inserting analysis for position %d (old ID: %d): %v\n", newPositionID, oldPositionID, insertErr)
					}
				}
			} else if analysisErr != sql.ErrNoRows {
				fmt.Printf("Error querying analysis for position %d: %v\n", oldPositionID, analysisErr)
			}
		}

		// Export comment if it exists and if includeComments is true
		if includeComments {
			var comment string
			commentErr := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, oldPositionID).Scan(&comment)
			if commentErr == nil && comment != "" {
				if _, insertErr := tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newPositionID, comment); insertErr != nil {
					fmt.Printf("Error inserting comment for position %d (old ID: %d): %v\n", newPositionID, oldPositionID, insertErr)
				}
			} else if commentErr != nil && commentErr != sql.ErrNoRows {
				fmt.Printf("Error querying comment for position %d: %v\n", oldPositionID, commentErr)
			}
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		fmt.Println("Error committing transaction for export:", err)
		return err
	}

	// Export filter library if includeFilterLibrary is true
	if includeFilterLibrary {
		rows, err := d.db.Query(`SELECT name, command, edit_position FROM filter_library`)
		if err == nil {
			defer rows.Close()

			for rows.Next() {
				var name, command, editPosition string
				err := rows.Scan(&name, &command, &editPosition)
				if err == nil {
					_, err = exportDB.Exec(`INSERT INTO filter_library (name, command, edit_position) VALUES (?, ?, ?)`, name, command, editPosition)
					if err != nil {
						fmt.Printf("Error inserting filter library entry '%s': %v\n", name, err)
					}
				}
			}
		}
	}

	fmt.Printf("Successfully exported %d positions to %s\n", len(positions), exportPath)
	return nil
}

// DeleteFile is a helper function to delete a file
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}

// Match import and management functions

// Import XG match file using xgparser library
// This function uses raw segment parsing to capture complete cube action information
func (d *Database) ImportXGMatch(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Parse the XG file using raw segments for complete data
	imp := xgparser.NewImport(filePath)
	segments, err := imp.GetFileSegments()
	if err != nil {
		return 0, fmt.Errorf("failed to get file segments: %w", err)
	}

	// Also parse lightweight structure for metadata
	match, err := xgparser.ParseXG(segments)
	if err != nil {
		fmt.Printf("Error parsing XG file: %v\n", err)
		return 0, fmt.Errorf("failed to parse XG file: %w", err)
	}

	// Parse raw records for complete cube information
	rawCubeInfo := make(map[string]*RawCubeAction) // key: "game_cubeIdx"
	for _, seg := range segments {
		if seg.Type == xgparser.SegmentXGGameFile {
			records, _ := xgparser.ParseGameFile(seg.Data, -1)
			gameNum := int32(0)
			cubeIdx := 0
			for _, rec := range records {
				switch r := rec.(type) {
				case *xgparser.HeaderGameEntry:
					gameNum = r.GameNumber
					cubeIdx = 0
				case *xgparser.CubeEntry:
					if r.Double != -2 { // Skip initial positions
						key := fmt.Sprintf("%d_%d", gameNum, cubeIdx)
						rawCubeInfo[key] = &RawCubeAction{
							Double:   r.Double,
							Take:     r.Take,
							ActiveP:  r.ActiveP,
							CubeB:    r.CubeB,
							Position: r.Position,
							Doubled:  r.Doubled,
						}
						cubeIdx++
					}
				}
			}
		}
	}

	// Parse match date
	var matchDate time.Time
	if match.Metadata.DateTime != "" {
		// Try to parse various date formats
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, match.Metadata.DateTime); err == nil {
				matchDate = t
				break
			}
		}
	}
	if matchDate.IsZero() {
		matchDate = time.Now()
	}

	// Compute match hash for duplicate detection (includes full match transcription)
	matchHash := ComputeMatchHash(match)

	// Check if this match already exists
	existingMatchID, err := d.checkMatchExistsLocked(matchHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for duplicate match: %w", err)
	}
	if existingMatchID > 0 {
		return 0, ErrDuplicateMatch
	}

	// Begin transaction for atomic import
	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Insert match metadata (including match_hash)
	result, err := tx.Exec(`
		INSERT INTO match (player1_name, player2_name, event, location, round, 
		                   match_length, match_date, file_path, game_count, match_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, match.Metadata.Player1Name, match.Metadata.Player2Name,
		match.Metadata.Event, match.Metadata.Location, match.Metadata.Round,
		match.Metadata.MatchLength, matchDate, filePath, len(match.Games), matchHash)

	if err != nil {
		return 0, fmt.Errorf("failed to insert match: %w", err)
	}

	matchID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get match ID: %w", err)
	}

	// Build a position cache for deduplication
	positionCache := make(map[string]int64) // map[positionJSON]positionID

	// Load existing positions into cache
	existingRows, err := tx.Query(`SELECT id, state FROM position`)
	if err != nil {
		return 0, fmt.Errorf("failed to load existing positions: %w", err)
	}

	for existingRows.Next() {
		var existingID int64
		var existingStateJSON string
		if err := existingRows.Scan(&existingID, &existingStateJSON); err != nil {
			continue
		}

		var existingPosition Position
		if err := json.Unmarshal([]byte(existingStateJSON), &existingPosition); err != nil {
			continue
		}

		// Normalize for comparison
		existingPosition.ID = 0
		normalizedJSON, _ := json.Marshal(existingPosition)
		positionCache[string(normalizedJSON)] = existingID
	}
	existingRows.Close()

	fmt.Printf("Loaded %d existing positions into cache\n", len(positionCache))

	// Import each game
	for gameIdx, game := range match.Games {
		gameID, err := d.importGame(tx, matchID, &game)
		if err != nil {
			return 0, fmt.Errorf("failed to import game %d: %w", game.GameNumber, err)
		}

		// Track cube index for raw data lookup
		// Each turn in XG has a CubeEntry followed by a MoveEntry
		// cubeIdx tracks which CubeEntry we're at (reset per game)
		cubeIdx := 0

		// Track the last cube analysis to associate with checker moves
		var lastCubeAnalysis *RawCubeAction

		// Import moves for this game
		for moveIdx, move := range game.Moves {
			var rawCube *RawCubeAction

			if move.MoveType == "cube" {
				// This is a cube decision, look up the raw data
				key := fmt.Sprintf("%d_%d", gameIdx+1, cubeIdx)
				if rc, ok := rawCubeInfo[key]; ok {
					rawCube = rc
					lastCubeAnalysis = rc // Remember for the following checker move
				}
				cubeIdx++
			} else if move.MoveType == "checker" {
				// For checker moves, the cube analysis comes from the preceding CubeEntry
				// Pass the last cube analysis so it can be saved with the checker position
				rawCube = lastCubeAnalysis
				lastCubeAnalysis = nil // Clear after use (one cube analysis per checker move)
			}

			err := d.importMoveWithCacheAndRawCube(tx, gameID, int32(moveIdx), &move, &game, int32(match.Metadata.MatchLength), positionCache, rawCube)
			if err != nil {
				return 0, fmt.Errorf("failed to import move %d in game %d: %w", moveIdx, game.GameNumber, err)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully imported match %d with %d games from %s\n", matchID, len(match.Games), filePath)
	return matchID, nil
}

// RawCubeAction stores raw cube action data from XG file
type RawCubeAction struct {
	Double   int32
	Take     int32
	ActiveP  int32
	CubeB    int32
	Position [26]int8
	Doubled  *xgparser.EngineStructDoubleAction // Full cube analysis data
}

// importGame inserts a game and returns its ID
func (d *Database) importGame(tx *sql.Tx, matchID int64, game *xgparser.Game) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2,
		                  winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, matchID, game.GameNumber, game.InitialScore[0], game.InitialScore[1],
		game.Winner, game.PointsWon, len(game.Moves))

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// importMoveWithCache imports a move using a position cache for deduplication
func (d *Database) importMoveWithCache(tx *sql.Tx, gameID int64, moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, positionCache map[string]int64) error {
	var positionID int64
	var player int32
	var dice [2]int32
	var checkerMoveStr string
	var cubeActionStr string

	if move.MoveType == "checker" && move.CheckerMove != nil {
		// Create position from checker move
		pos, err := d.createPositionFromXG(move.CheckerMove.Position, game, matchLength, moveNumber, move.CheckerMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Set position-specific attributes from move
		// Convert XG player encoding (-1, 1) to blunderDB encoding (0, 1)
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{int(move.CheckerMove.Dice[0]), int(move.CheckerMove.Dice[1])}

		// Save position with cache
		posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CheckerMove.ActivePlayer
		dice = move.CheckerMove.Dice

		// Convert move notation with hit detection for consistency with analysis display
		checkerMoveStr = d.convertXGMoveToStringWithHits(move.CheckerMove.PlayedMove, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, player, dice[0], dice[1], checkerMoveStr)
		if err != nil {
			return err
		}

		moveID, _ := moveResult.LastInsertId()

		// Save analysis if available
		if len(move.CheckerMove.Analysis) > 0 {
			// Save to move_analysis table
			for _, analysis := range move.CheckerMove.Analysis {
				err = d.saveMoveAnalysisInTx(tx, moveID, &analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save checker analysis: %v\n", err)
				}
			}

			// Also save to position analysis table (for UI compatibility)
			err = d.saveCheckerAnalysisToPositionInTx(tx, positionID, move.CheckerMove.Analysis, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer, &move.CheckerMove.PlayedMove)
			if err != nil {
				fmt.Printf("Warning: failed to save position analysis: %v\n", err)
			}
		}

	} else if move.MoveType == "cube" && move.CubeMove != nil {
		// Create position from cube move
		pos, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Set position-specific attributes from move
		// Convert XG player encoding (-1, 1) to blunderDB encoding (0, 1)
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
		pos.DecisionType = CubeAction
		pos.Dice = [2]int{0, 0} // No dice for cube decisions

		// Save position with cache
		posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CubeMove.ActivePlayer
		cubeActionStr = d.convertCubeAction(move.CubeMove.CubeAction)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, cube_action)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "cube", positionID, player, 0, 0, cubeActionStr)
		if err != nil {
			return err
		}

		moveID, _ := moveResult.LastInsertId()

		// Save cube analysis if available
		if move.CubeMove.Analysis != nil {
			// Save to move_analysis table
			err = d.saveCubeAnalysisInTx(tx, moveID, move.CubeMove.Analysis)
			if err != nil {
				fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
			}

			// Also save to position analysis table (for UI compatibility)
			err = d.saveCubeAnalysisToPositionInTx(tx, positionID, move.CubeMove.Analysis)
			if err != nil {
				fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
			}
		}
	}

	return nil
}

// importMoveWithCacheAndRawCube imports a move with raw cube data for complete action info
func (d *Database) importMoveWithCacheAndRawCube(tx *sql.Tx, gameID int64, moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, positionCache map[string]int64, rawCube *RawCubeAction) error {
	var positionID int64
	var player int32
	var dice [2]int32
	var checkerMoveStr string
	var cubeActionStr string

	if move.MoveType == "checker" && move.CheckerMove != nil {
		// Create position from checker move
		pos, err := d.createPositionFromXG(move.CheckerMove.Position, game, matchLength, moveNumber, move.CheckerMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Set position-specific attributes from move
		// Convert XG player encoding (-1, 1) to blunderDB encoding (0, 1)
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{int(move.CheckerMove.Dice[0]), int(move.CheckerMove.Dice[1])}

		// Save position with cache
		posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CheckerMove.ActivePlayer
		dice = move.CheckerMove.Dice

		// Convert move notation with hit detection for consistency with analysis display
		checkerMoveStr = d.convertXGMoveToStringWithHits(move.CheckerMove.PlayedMove, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, player, dice[0], dice[1], checkerMoveStr)
		if err != nil {
			return err
		}

		moveID, _ := moveResult.LastInsertId()

		// Save analysis if available
		if len(move.CheckerMove.Analysis) > 0 {
			for _, analysis := range move.CheckerMove.Analysis {
				err = d.saveMoveAnalysisInTx(tx, moveID, &analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save checker analysis: %v\n", err)
				}
			}

			err = d.saveCheckerAnalysisToPositionInTx(tx, positionID, move.CheckerMove.Analysis, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer, &move.CheckerMove.PlayedMove)
			if err != nil {
				fmt.Printf("Warning: failed to save position analysis: %v\n", err)
			}
		}

		// Save cube analysis for this checker position (from the preceding CubeEntry)
		// This allows displaying cube info when pressing 'd' on a checker decision
		if rawCube != nil && rawCube.Doubled != nil {
			err = d.saveCubeAnalysisForCheckerPositionInTx(tx, positionID, rawCube)
			if err != nil {
				fmt.Printf("Warning: failed to save cube analysis for checker position: %v\n", err)
			}
		}

	} else if move.MoveType == "cube" && move.CubeMove != nil {
		// Handle explicit cube decisions
		// Only create positions for EXPLICIT cube actions (when a player actually doubles)
		// Skip implicit "No Double" decisions

		// Check if this is an explicit cube action
		isExplicitCubeAction := false
		if rawCube != nil {
			// Explicit action: Double was offered (Double == 1)
			isExplicitCubeAction = (rawCube.Double == 1)
		} else {
			// Fallback: check CubeAction field
			isExplicitCubeAction = (move.CubeMove.CubeAction != 0) // 0 = No Double
		}

		if !isExplicitCubeAction {
			// Skip implicit "No Double" decisions - don't create position
			return nil
		}

		if rawCube != nil && rawCube.Double == 1 && rawCube.Take == 1 {
			// DOUBLE/TAKE scenario: Create two positions

			// Position 1: Doubling decision (before the double)
			// The player on roll decides whether to double
			pos1, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
			if err != nil {
				return fmt.Errorf("failed to create doubling position: %w", err)
			}
			pos1.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
			pos1.DecisionType = CubeAction
			pos1.Dice = [2]int{0, 0}

			// Save position 1 (doubling decision)
			posID1, err := d.savePositionInTxWithCache(tx, pos1, positionCache)
			if err != nil {
				return fmt.Errorf("failed to save doubling position: %w", err)
			}

			player = move.CubeMove.ActivePlayer

			// Save move 1: Double
			moveResult1, err := tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", posID1, player, 0, 0, "Double")
			if err != nil {
				return err
			}
			moveID1, _ := moveResult1.LastInsertId()

			// Save analysis to first position
			if move.CubeMove.Analysis != nil {
				err = d.saveCubeAnalysisInTx(tx, moveID1, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
				}
				err = d.saveCubeAnalysisToPositionInTx(tx, posID1, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
				}
			}

			// Position 2: Take/Pass decision (after the double)
			// The opponent decides whether to take or pass
			// Note: We show the position BEFORE the take decision is executed,
			// so the cube is still in the center at doubled value.
			// The cube ownership will be reflected in the NEXT checker move position.
			pos2 := *pos1                           // Clone the position
			pos2.Cube.Value++                       // Double the cube (increment exponent: 0→1, 1→2, etc.)
			opponentPlayer := 1 - pos1.PlayerOnRoll // Opponent player (blunderDB encoding)
			pos2.PlayerOnRoll = opponentPlayer      // Opponent decides whether to take
			pos2.Cube.Owner = -1                    // Cube still in center (decision not yet executed)

			// Save position 2 (take decision)
			posID2, err := d.savePositionInTxWithCache(tx, &pos2, positionCache)
			if err != nil {
				return fmt.Errorf("failed to save take position: %w", err)
			}
			positionID = posID2 // Use second position as the reference

			// Convert opponent from blunderDB (0,1) back to XG encoding (1,-1) for move table
			opponentPlayerXG := int32(1)
			if opponentPlayer == 0 {
				opponentPlayerXG = 1
			} else {
				opponentPlayerXG = -1
			}

			// Save move 2: Take
			_, err = tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", posID2, opponentPlayerXG, 0, 0, "Take")
			if err != nil {
				return err
			}

		} else {
			// Single cube action (Double without Take, or other actions)
			pos, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
			if err != nil {
				return fmt.Errorf("failed to create position: %w", err)
			}

			pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
			pos.DecisionType = CubeAction
			pos.Dice = [2]int{0, 0}

			// Save position
			posID, err := d.savePositionInTxWithCache(tx, pos, positionCache)
			if err != nil {
				return fmt.Errorf("failed to save position: %w", err)
			}
			positionID = posID
			player = move.CubeMove.ActivePlayer

			// Determine cube action string
			if rawCube != nil {
				cubeActionStr = d.convertRawCubeAction(rawCube.Double, rawCube.Take)
			} else {
				cubeActionStr = d.convertCubeAction(move.CubeMove.CubeAction)
			}

			// Save move
			moveResult, err := tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", positionID, player, 0, 0, cubeActionStr)
			if err != nil {
				return err
			}

			moveID, _ := moveResult.LastInsertId()

			// Save cube analysis
			if move.CubeMove.Analysis != nil {
				err = d.saveCubeAnalysisInTx(tx, moveID, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
				}

				err = d.saveCubeAnalysisToPositionInTx(tx, positionID, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
				}
			}
		}
	}

	return nil
}

// convertRawCubeAction converts raw Double/Take values to action string
func (d *Database) convertRawCubeAction(double, take int32) string {
	// XG cube action encoding:
	// Double=0, Take=-1: No Double
	// Double=1, Take=1: Double, Take
	// Double=1, Take=-1: Double/Pass (opponent passed the double)
	// Double=-2: Initial position (should be filtered before)

	if double == 0 {
		return "No Double"
	} else if double == 1 {
		if take == 1 {
			return "Double/Take"
		} else {
			return "Double/Pass"
		}
	}
	return fmt.Sprintf("Unknown(D=%d,T=%d)", double, take)
}

// importMove imports a move, creates associated position, and stores analysis (deprecated - use importMoveWithCache)
func (d *Database) importMove(tx *sql.Tx, gameID int64, moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32) error {
	var positionID int64
	var player int32
	var dice [2]int32
	var checkerMoveStr string
	var cubeActionStr string

	if move.MoveType == "checker" && move.CheckerMove != nil {
		// Create position from checker move
		pos, err := d.createPositionFromXG(move.CheckerMove.Position, game, matchLength, moveNumber, move.CheckerMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Save position
		posID, err := d.savePositionInTx(tx, pos)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CheckerMove.ActivePlayer
		dice = move.CheckerMove.Dice

		// Convert move notation with hit detection for consistency with analysis display
		checkerMoveStr = d.convertXGMoveToStringWithHits(move.CheckerMove.PlayedMove, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, player, dice[0], dice[1], checkerMoveStr)
		if err != nil {
			return err
		}

		moveID, _ := moveResult.LastInsertId()

		// Save analysis if available
		if len(move.CheckerMove.Analysis) > 0 {
			for _, analysis := range move.CheckerMove.Analysis {
				err = d.saveMoveAnalysisInTx(tx, moveID, &analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save checker analysis: %v\n", err)
				}
			}
		}

	} else if move.MoveType == "cube" && move.CubeMove != nil {
		// Create position from cube move
		pos, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Save position
		posID, err := d.savePositionInTx(tx, pos)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CubeMove.ActivePlayer
		cubeActionStr = d.convertCubeAction(move.CubeMove.CubeAction)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, cube_action)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "cube", positionID, player, 0, 0, cubeActionStr)
		if err != nil {
			return err
		}

		moveID, _ := moveResult.LastInsertId()

		// Save cube analysis if available
		if move.CubeMove.Analysis != nil {
			err = d.saveCubeAnalysisInTx(tx, moveID, move.CubeMove.Analysis)
			if err != nil {
				fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
			}
		}
	}

	return nil
}

// createPositionFromXG converts xgparser.Position to blunderDB Position
// activePlayer indicates which XG player (-1 or 1) is on roll in this position
func (d *Database) createPositionFromXG(xgPos xgparser.Position, game *xgparser.Game, matchLength int32, moveNum int32, activePlayer int32) (*Position, error) {
	// Convert XG player encoding to blunderDB encoding
	// XG: -1 = Player 1, 1 = Player 2
	// blunderDB: 0 = Player 1, 1 = Player 2
	activePlayerBlunderDB := convertXGPlayerToBlunderDB(activePlayer)
	opponentPlayerBlunderDB := 1 - activePlayerBlunderDB

	// Calculate away scores from current scores
	// In blunderDB, scores are "points away from winning"
	// game.InitialScore contains current scores (e.g., 2-3 in a 7-point match)
	// We need to convert to away scores (e.g., 5-away, 4-away)
	awayScore1 := int(matchLength) - int(game.InitialScore[0])
	awayScore2 := int(matchLength) - int(game.InitialScore[1])

	// Handle unlimited match (matchLength == 0)
	if matchLength == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Map XG cube position to blunderDB format
	// XG CubePos is RELATIVE to active player:
	//   CubePos = 0: Center (no owner)
	//   CubePos = 1: Active player owns the cube
	//   CubePos = -1: Opponent owns the cube
	// blunderDB uses absolute encoding:
	//   -1 = center (no owner)
	//   0 = Player 1 owns (bottom, black)
	//   1 = Player 2 owns (top, white)
	cubeOwner := -1 // Default: center (no owner)
	if xgPos.CubePos == 1 {
		// Active player owns the cube
		cubeOwner = activePlayerBlunderDB
	} else if xgPos.CubePos == -1 {
		// Opponent owns the cube
		cubeOwner = opponentPlayerBlunderDB
	}
	// CubePos == 0 means center, cubeOwner stays -1

	// Convert cube value from XG (direct value 1,2,4,8...) to blunderDB (exponent 0,1,2,3...)
	// blunderDB displays 2^value, so we need log2(xgCube)
	cubeValue := 0
	if xgPos.Cube > 0 {
		// Calculate log2 of cube value
		for v := int(xgPos.Cube); v > 1; v >>= 1 {
			cubeValue++
		}
	}

	pos := &Position{
		PlayerOnRoll: 0, // Will be set from move context
		DecisionType: 0, // Checker decision by default
		Score:        [2]int{awayScore1, awayScore2},
		Cube: Cube{
			Value: cubeValue,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0}, // Will be set from move data
	}

	// Convert checkers from XG format to blunderDB format
	// XG format: index 0-23 are points 1-24, index 24=bar, 25=opponent bar
	// Positive values = active player's checkers, negative = opponent's checkers
	//
	// In blunderDB:
	// - Color 0 = Player 1 (always at bottom, black, moves 24→1) - NEVER CHANGES
	// - Color 1 = Player 2 (always at top, white, moves 1→24) - NEVER CHANGES
	// - Points 1-24 with indices 1-24 in the array
	// - Index 0 = Player 2's bar (white), Index 25 = Player 1's bar (black)
	//
	// XG stores positions from the active player's perspective:
	// - Positive checkers = active player's checkers
	// - Negative checkers = opponent's checkers
	// - Point numbering is from active player's perspective
	//
	// Player mapping:
	// - XG Player 0 = blunderDB Player 1 (Color 0, bottom, black)
	// - XG Player 1 = blunderDB Player 2 (Color 1, top, white)
	//
	// Strategy:
	// 1. Determine which player owns the checkers based on sign AND activePlayer
	// 2. Assign colors based on OWNER, not on sign
	// 3. Mirror positions if activePlayer == 1 (since XG encodes from active player's view)
	for i := 0; i < 26; i++ {
		checkerCount := xgPos.Checkers[i]

		// Determine WHERE to place them (calculate targetIndex for ALL points, even empty)
		// XG uses 1-based indexing: index 1-24 = points 1-24, index 0 = opponent bar, index 25 = active bar
		// blunderDB also uses same: index 1-24 = points 1-24, index 0 = P2 bar, index 25 = P1 bar
		targetIndex := i
		if activePlayerBlunderDB == 1 {
			// Player 2's perspective, need to mirror to Player 1's perspective
			if i >= 1 && i <= 24 {
				// XG index i = Player 2's point i → Player 1's point (25 - i) → same index (25 - i)
				targetIndex = 25 - i
			} else if i == 0 {
				// Opponent's bar from Player 2's view = Player 1's bar
				targetIndex = 25
			} else if i == 25 {
				// Active player's bar (Player 2) → Player 2's bar
				targetIndex = 0
			}
		} else {
			// Player 1's perspective - direct mapping (XG index = blunderDB index)
			// XG index 1-24 = Player 1's points 1-24 → blunderDB index 1-24 (same)
			// XG index 0 = opponent bar (Player 2) → blunderDB index 0 (Player 2's bar)
			// XG index 25 = active bar (Player 1) → blunderDB index 25 (Player 1's bar)
			// No transformation needed: targetIndex = i
		}

		if checkerCount == 0 {
			pos.Board.Points[targetIndex] = Point{Checkers: 0, Color: -1}
			continue
		}

		// Determine WHO owns these checkers
		var ownerColor int // 0=Player1(black), 1=Player2(white)
		if checkerCount > 0 {
			// Positive = active player
			ownerColor = activePlayerBlunderDB
		} else {
			// Negative = opponent
			ownerColor = opponentPlayerBlunderDB
		}

		// Place checkers with FIXED color based on owner
		pos.Board.Points[targetIndex] = Point{
			Checkers: int(abs(checkerCount)),
			Color:    ownerColor,
		}
	}

	// Calculate bearoff (checkers borne off)
	player1Total := 0
	player2Total := 0
	for i := 0; i < 26; i++ {
		if pos.Board.Points[i].Color == 0 {
			player1Total += pos.Board.Points[i].Checkers
		} else if pos.Board.Points[i].Color == 1 {
			player2Total += pos.Board.Points[i].Checkers
		}
	}
	pos.Board.Bearoff = [2]int{15 - player1Total, 15 - player2Total}

	return pos, nil
}

// Helper function for absolute value
func abs(x int8) int {
	if x < 0 {
		return int(-x)
	}
	return int(x)
}

// convertXGPlayerToBlunderDB converts XG player encoding to blunderDB encoding
// XG: -1 = Player 1, 1 = Player 2
// blunderDB: 0 = Player 1, 1 = Player 2
func convertXGPlayerToBlunderDB(xgPlayer int32) int {
	// XG uses 1 for first player, -1 for second player
	// blunderDB uses 0 for Player 1 (black, bottom), 1 for Player 2 (white, top)
	if xgPlayer == 1 {
		return 0 // Player 1
	}
	return 1 // Player 2
}

// savePositionInTxWithCache saves a position within a transaction using a cache for deduplication
func (d *Database) savePositionInTxWithCache(tx *sql.Tx, position *Position, positionCache map[string]int64) (int64, error) {
	// Normalize position for comparison (exclude ID)
	positionCopy := *position
	positionCopy.ID = 0

	positionJSON, err := json.Marshal(positionCopy)
	if err != nil {
		return 0, err
	}

	// Check cache first
	if existingID, exists := positionCache[string(positionJSON)]; exists {
		return existingID, nil
	}

	// Position doesn't exist, create new one
	result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
	if err != nil {
		return 0, err
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Update position with ID
	position.ID = positionID
	positionJSONWithID, _ := json.Marshal(position)
	_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSONWithID), positionID)
	if err != nil {
		return 0, err
	}

	// Add to cache for future lookups
	positionCache[string(positionJSON)] = positionID

	return positionID, nil
}

// savePositionInTx saves a position within a transaction, checking for duplicates first
func (d *Database) savePositionInTx(tx *sql.Tx, position *Position) (int64, error) {
	// First check if this position already exists
	positionCopy := *position
	positionCopy.ID = 0

	positionJSON, err := json.Marshal(positionCopy)
	if err != nil {
		return 0, err
	}

	// Query existing positions to check for duplicates
	rows, err := tx.Query(`SELECT id, state FROM position`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var stateJSON string
		var positionID int64
		if err = rows.Scan(&positionID, &stateJSON); err != nil {
			continue
		}

		var existingPosition Position
		if err = json.Unmarshal([]byte(stateJSON), &existingPosition); err != nil {
			continue
		}

		// Compare positions excluding the ID field
		existingPosition.ID = 0
		existingPositionJSON, err := json.Marshal(existingPosition)
		if err != nil {
			continue
		}

		if string(positionJSON) == string(existingPositionJSON) {
			// Position already exists, return existing ID
			return positionID, nil
		}
	}

	// Position doesn't exist, create new one
	result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
	if err != nil {
		return 0, err
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Update position with ID
	position.ID = positionID
	positionJSON, _ = json.Marshal(position)
	_, err = tx.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSON), positionID)

	return positionID, err
}

// saveMoveAnalysisInTx saves checker move analysis within a transaction
func (d *Database) saveMoveAnalysisInTx(tx *sql.Tx, moveID int64, analysis *xgparser.CheckerAnalysis) error {
	// Calculate win rates (player1 is player on roll)
	player1WinRate := analysis.Player1WinRate * 100.0 // Convert to percentage
	player2WinRate := (1.0 - analysis.Player1WinRate) * 100.0

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "checker", translateAnalysisDepth(int(analysis.AnalysisDepth)),
		analysis.Equity, 0.0, // No separate equity error in CheckerAnalysis
		player1WinRate, analysis.Player1GammonRate*100.0, analysis.Player1BgRate*100.0,
		player2WinRate, analysis.Player2GammonRate*100.0, analysis.Player2BgRate*100.0)

	return err
}

// saveCubeAnalysisInTx saves cube analysis within a transaction
func (d *Database) saveCubeAnalysisInTx(tx *sql.Tx, moveID int64, analysis *xgparser.CubeAnalysis) error {
	// Calculate win rates
	player1WinRate := analysis.Player1WinRate * 100.0
	player2WinRate := (1.0 - analysis.Player1WinRate) * 100.0

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "cube", translateAnalysisDepth(int(analysis.AnalysisDepth)),
		analysis.CubefulNoDouble, 0.0,
		player1WinRate, analysis.Player1GammonRate*100.0, analysis.Player1BgRate*100.0,
		player2WinRate, analysis.Player2GammonRate*100.0, analysis.Player2BgRate*100.0)

	return err
}

// saveCheckerAnalysisToPositionInTx converts XG checker analysis to PositionAnalysis and saves it
// playedMove is optional - if provided, it will be used as the source of truth for the first analysis
// (workaround for xgparser bug where analysis.Move may be incomplete for multi-submove bear-offs)
func (d *Database) saveCheckerAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analyses []xgparser.CheckerAnalysis, initialPosition *xgparser.Position, activePlayer int32, playedMove *[8]int32) error {
	if len(analyses) == 0 {
		return nil
	}

	// Create PositionAnalysis structure
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	// Build checker analysis with all moves
	checkerMoves := make([]CheckerMove, 0, len(analyses))
	for i, analysis := range analyses {
		// Convert move from [8]int8 to [8]int32 for convertXGMoveToString
		var move [8]int32

		// For the first analysis (index 0), use playedMove if available
		// This is a workaround for xgparser bug where analysis.Move may be incomplete
		// for multi-submove moves (e.g., 4/off 3/off might only have 4/off in analysis)
		if i == 0 && playedMove != nil {
			move = *playedMove
		} else {
			for j := 0; j < 8; j++ {
				move[j] = int32(analysis.Move[j])
			}
			// For other analyses, infer multipliers from position changes
			// XG stores moves compactly - e.g., 1/off(4) is stored as just 1/off once
			if initialPosition != nil {
				move = inferMoveMultipliers(move, initialPosition, &analysis.Position, activePlayer)
			}
		}

		// Use move string with hit detection if initial position is available
		var moveStr string
		if initialPosition != nil {
			moveStr = d.convertXGMoveToStringWithHits(move, initialPosition, activePlayer)
		} else {
			moveStr = d.convertXGMoveToString(move, activePlayer)
		}

		var equityError *float64
		if i > 0 {
			diff := float64(analyses[0].Equity - analysis.Equity)
			equityError = &diff
		}

		checkerMove := CheckerMove{
			Index:                    i,
			AnalysisDepth:            translateAnalysisDepth(int(analysis.AnalysisDepth)),
			Move:                     moveStr,
			Equity:                   float64(analysis.Equity),
			EquityError:              equityError,
			PlayerWinChance:          float64(analysis.Player1WinRate) * 100.0,
			PlayerGammonChance:       float64(analysis.Player1GammonRate) * 100.0,
			PlayerBackgammonChance:   float64(analysis.Player1BgRate) * 100.0,
			OpponentWinChance:        float64(1.0-analysis.Player1WinRate) * 100.0,
			OpponentGammonChance:     float64(analysis.Player2GammonRate) * 100.0,
			OpponentBackgammonChance: float64(analysis.Player2BgRate) * 100.0,
		}
		checkerMoves = append(checkerMoves, checkerMove)
	}

	posAnalysis.CheckerAnalysis = &CheckerAnalysis{
		Moves: checkerMoves,
	}

	// Save to analysis table
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveCubeAnalysisToPositionInTx converts XG cube analysis to PositionAnalysis and saves it
func (d *Database) saveCubeAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analysis *xgparser.CubeAnalysis) error {
	if analysis == nil {
		return nil
	}

	// Create PositionAnalysis structure
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubefulNoDouble := float64(analysis.CubefulNoDouble)
	cubefulDoubleTake := float64(analysis.CubefulDoubleTake)
	cubefulDoublePass := float64(analysis.CubefulDoublePass)

	// Calculate best equity considering opponent's optimal response
	// If player doubles, opponent will choose the action that minimizes player's equity
	// So effective double equity = min(DoubleTake, DoublePass)
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	// Player's best achievable equity = max(NoDouble, effectiveDoubleEquity)
	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		// Best action is "Double, Take" or "Double, Pass" depending on opponent's response
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	// Build doubling cube analysis
	// Error is negative when this decision loses equity vs best (current - best)
	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateAnalysisDepth(int(analysis.AnalysisDepth)),
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BgRate) * 100.0,
		OpponentWinChances:        float64(1.0-analysis.Player1WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BgRate) * 100.0,
		CubelessNoDoubleEquity:    float64(analysis.CubelessNoDouble),
		CubelessDoubleEquity:      float64(analysis.CubelessDouble),
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
		WrongTakePercentage:       0.0, // XG provides WrongPassTakePercent which covers both
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Save to analysis table
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveCubeAnalysisForCheckerPositionInTx saves cube analysis from a RawCubeAction to a checker position
// This is used to attach the cube decision analysis to checker moves (from the preceding CubeEntry)
// It merges the cube info with existing checker analysis if present
func (d *Database) saveCubeAnalysisForCheckerPositionInTx(tx *sql.Tx, positionID int64, rawCube *RawCubeAction) error {
	if rawCube == nil || rawCube.Doubled == nil {
		return nil
	}

	doubled := rawCube.Doubled

	// Try to load existing analysis for this position
	var existingAnalysisJSON string
	var existingID int64
	err := tx.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisJSON)

	var posAnalysis PositionAnalysis
	if err == nil && existingID > 0 {
		// Existing analysis found - merge cube info with it
		err = json.Unmarshal([]byte(existingAnalysisJSON), &posAnalysis)
		if err != nil {
			return err
		}
	} else {
		// No existing analysis - create new one
		posAnalysis = PositionAnalysis{
			PositionID:            int(positionID),
			AnalysisType:          "CheckerMove",
			AnalysisEngineVersion: "XG",
			CreationDate:          time.Now(),
		}
	}

	posAnalysis.LastModifiedDate = time.Now()

	// Extract data from EngineStructDoubleAction
	// XG Eval array mapping (from xgparser convertCubeEntry):
	// Eval[0] = opponent's backgammon rate
	// Eval[1] = opponent's gammon rate
	// Eval[2] = opponent's win rate (so player on roll's win rate = 1 - Eval[2])
	// Eval[4] = player on roll's gammon rate
	// Eval[5] = player on roll's backgammon rate
	// Eval[6] = cubeless equity
	cubefulNoDouble := float64(doubled.EquB)
	cubefulDoubleTake := float64(doubled.EquDouble)
	cubefulDoublePass := float64(doubled.EquDrop)

	// Win/gammon/bg rates from player on roll's perspective
	playerWin := (1.0 - float64(doubled.Eval[2])) * 100.0
	playerGammon := float64(doubled.Eval[4]) * 100.0
	playerBg := float64(doubled.Eval[5]) * 100.0
	opponentWin := float64(doubled.Eval[2]) * 100.0
	opponentGammon := float64(doubled.Eval[1]) * 100.0
	opponentBg := float64(doubled.Eval[0]) * 100.0

	// Calculate best equity considering opponent's optimal response
	// If player doubles, opponent will choose the action that minimizes player's equity
	// So effective double equity = min(DoubleTake, DoublePass)
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	// Player's best achievable equity = max(NoDouble, effectiveDoubleEquity)
	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		// Best action is "Double, Take" or "Double, Pass" depending on opponent's response
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	// Build doubling cube analysis
	// Error is negative when this decision loses equity vs best (current - best)
	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateAnalysisDepth(int(doubled.Level)),
		PlayerWinChances:          playerWin,
		PlayerGammonChances:       playerGammon,
		PlayerBackgammonChances:   playerBg,
		OpponentWinChances:        opponentWin,
		OpponentGammonChances:     opponentGammon,
		OpponentBackgammonChances: opponentBg,
		CubelessNoDoubleEquity:    float64(doubled.Eval[6]),
		CubelessDoubleEquity:      float64(doubled.Eval[6]), // Same as no double for cubeless
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       0.0, // Not available from EngineStructDoubleAction
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Save to analysis table (will update if exists, insert if not)
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// determineBestCubeAction determines the best cube action from analysis
// Takes into account that opponent will play optimally (choose min equity for the player)
func (d *Database) determineBestCubeAction(analysis *xgparser.CubeAnalysis) string {
	cubefulNoDouble := analysis.CubefulNoDouble
	cubefulDoubleTake := analysis.CubefulDoubleTake
	cubefulDoublePass := analysis.CubefulDoublePass

	// If player doubles, opponent will choose the action that minimizes player's equity
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	// Player's best choice is max(NoDouble, effectiveDoubleEquity)
	if effectiveDoubleEquity > cubefulNoDouble {
		if cubefulDoubleTake <= cubefulDoublePass {
			return "Double, Take"
		}
		return "Double, Pass"
	}
	return "No Double"
}

// saveAnalysisInTx saves a PositionAnalysis within a transaction
func (d *Database) saveAnalysisInTx(tx *sql.Tx, positionID int64, analysis PositionAnalysis) error {
	// Ensure the positionID is set in the analysis
	analysis.PositionID = int(positionID)

	// Update last modified date
	analysis.LastModifiedDate = time.Now()

	// Check if an analysis already exists for the given position ID
	var existingID int64
	var existingAnalysisJSON string
	err := tx.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisJSON)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existingID > 0 {
		// Preserve the existing creation date
		var existingAnalysis PositionAnalysis
		err = json.Unmarshal([]byte(existingAnalysisJSON), &existingAnalysis)
		if err != nil {
			return err
		}
		analysis.CreationDate = existingAnalysis.CreationDate

		// Update the existing analysis
		analysisJSON, err := json.Marshal(analysis)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE analysis SET data = ? WHERE id = ?`, string(analysisJSON), existingID)
		if err != nil {
			return err
		}
	} else {
		// Set creation date if not already set
		if analysis.CreationDate.IsZero() {
			analysis.CreationDate = time.Now()
		}

		// Insert a new analysis
		analysisJSON, err := json.Marshal(analysis)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, string(analysisJSON))
		if err != nil {
			return err
		}
	}

	return nil
}

// inferMoveMultipliers analyzes a partial move array and the position difference
// to infer the correct number of repetitions for each move.
// XG sometimes stores only one instance of a move even when multiple checkers make the same move.
// This function expands the move array to include all repetitions.
// Returns the expanded move array with correct multipliers.
func inferMoveMultipliers(partialMove [8]int32, initialPos, finalPos *xgparser.Position, activePlayer int32) [8]int32 {
	if initialPos == nil || finalPos == nil {
		return partialMove
	}

	// Parse the partial move to get unique from/to pairs, preserving order
	type moveSpec struct {
		from int32
		to   int32
	}
	var uniqueMoves []moveSpec
	for i := 0; i < 8; i += 2 {
		from := partialMove[i]
		to := partialMove[i+1]
		if from == -1 {
			break
		}
		// Handle implicit bear-off
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		// Check if this move is already in our list
		found := false
		for _, m := range uniqueMoves {
			if m.from == from && m.to == to {
				found = true
				break
			}
		}
		if !found {
			uniqueMoves = append(uniqueMoves, moveSpec{from: from, to: to})
		}
	}

	if len(uniqueMoves) == 0 {
		return partialMove
	}

	// Calculate how many checkers left each source point
	// netChange[point] = initialCheckers - finalCheckers (positive = net loss of checkers)
	netChange := make(map[int32]int32)

	for _, ms := range uniqueMoves {
		if ms.from == 25 {
			netChange[25] = int32(initialPos.Checkers[25]) - int32(finalPos.Checkers[25])
		} else if ms.from >= 1 && ms.from <= 24 {
			netChange[ms.from] = int32(initialPos.Checkers[ms.from]) - int32(finalPos.Checkers[ms.from])
		}
		// Track destination changes too
		if ms.to >= 1 && ms.to <= 24 {
			netChange[ms.to] = int32(initialPos.Checkers[ms.to]) - int32(finalPos.Checkers[ms.to])
		}
	}

	// Build a flow model: how many checkers move from each source
	// Track arriving checkers for intermediate points
	arriving := make(map[int32]int32) // checkers arriving at each point

	var expandedMove [8]int32
	for i := range expandedMove {
		expandedMove[i] = -1
	}
	moveIndex := 0

	// Process moves in order
	for _, ms := range uniqueMoves {
		var count int32 = 1

		if ms.to == -2 {
			// Bear-off: count = net checkers that left this point + checkers arriving from other points
			count = netChange[ms.from] + arriving[ms.from]
		} else if ms.to >= 1 && ms.to <= 24 {
			// Move to board point
			// Count = how many checkers left the source AND didn't come back
			// For most cases, this is simply the net change of the source
			// But we need to also account for checker flow

			// Net change at source tells us how many checkers total left
			srcLoss := netChange[ms.from]

			// If source also receives checkers (from another move), we need less moves
			srcReceive := arriving[ms.from]

			// The move count is the source loss minus any checkers that arrived
			count = srcLoss - srcReceive
			if count <= 0 {
				count = 1
			}

			// After this move, these checkers arrive at the destination
			arriving[ms.to] += count
		}

		// Cap at 4 (maximum for doubles) or remaining slots
		maxMoves := int32(4)
		remainingSlots := int32((8 - moveIndex) / 2)
		if count > maxMoves {
			count = maxMoves
		}
		if count > remainingSlots {
			count = remainingSlots
		}
		if count < 1 {
			count = 1
		}

		for j := int32(0); j < count && moveIndex < 8; j++ {
			expandedMove[moveIndex] = ms.from
			expandedMove[moveIndex+1] = ms.to
			moveIndex += 2
		}
	}

	return expandedMove
}

// translateAnalysisDepth converts XG analysis depth codes to human-readable strings
// XG depth codes:
//   - 0-9 = (N+1)-ply search depth (XG stores ply-1 internally)
//   - 998-1000 = Book moves (different footnote references)
//   - 1001 = XG Roller (neural network)
//   - 1002 = XG Roller++ (extended neural network analysis)
func translateAnalysisDepth(depth int) string {
	switch {
	case depth >= 0 && depth <= 9:
		return fmt.Sprintf("%d-ply", depth+1)
	case depth >= 998 && depth <= 1000:
		return "Book"
	case depth == 1001:
		return "XG Roller"
	case depth == 1002:
		return "XG Roller++"
	default:
		return fmt.Sprintf("%d", depth)
	}
}

// convertXGMoveToString converts XG move format to readable string
// XG move format: [from, to, from, to, ...] where:
//   - 1-24 are board points (from active player's perspective)
//   - 25 is the bar
//   - -2 is bear off
//   - -1 is unused/end of move
//
// Moves in XG files and analysis are stored from the player on roll's perspective.
// They should be displayed as-is without mirroring, per standard backgammon notation.
func (d *Database) convertXGMoveToString(playedMove [8]int32, activePlayer int32) string {
	// Note: activePlayer is kept for API compatibility but moves don't need transformation
	// Moves are always from the roller's perspective (24 = furthest from home, 1 = closest)
	_ = activePlayer // unused but kept for signature compatibility

	// Parse raw moves into from/to pairs
	var fromPts []int32
	var toPts []int32
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		// Check for end of move marker (-1 when from is also -1)
		if from == -1 {
			break
		}
		// Handle implicit bear-off: XG sometimes encodes bear-off as to=-1 or to<=0
		// when the calculated destination (from - die) would be <= 0
		// This happens when bearing off from the home board (points 1-6)
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2 // Convert to explicit bear-off
		}
		// Skip invalid point values (should be 1-24 for points, 25 for bar, -2 for bear off)
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}
		fromPts = append(fromPts, from)
		toPts = append(toPts, to)
	}

	if len(fromPts) == 0 {
		return "Cannot Move"
	}

	// Try to merge consecutive checker slides (same checker moving through points)
	// E.g., 24/23 23/22 22/21 21/20 -> 24/20 when dice are all same
	fromPts, toPts = d.mergeSlides(fromPts, toPts)

	// Create sortable items
	type moveItem struct {
		from int32
		to   int32
	}
	items := make([]moveItem, len(fromPts))
	for i := range fromPts {
		items[i] = moveItem{from: fromPts[i], to: toPts[i]}
	}

	// Sort moves by 'from' point descending (standard backgammon notation)
	// bar (25) comes first, then higher points before lower points
	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	// Format each move as string
	formatPoint := func(p int32) string {
		if p == 25 {
			return "bar"
		} else if p == -2 {
			return "off"
		} else if p >= 1 && p <= 24 {
			return fmt.Sprintf("%d", p)
		}
		return fmt.Sprintf("?%d", p)
	}

	// Build move string, grouping identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		// Count consecutive identical moves
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
			} else {
				break
			}
		}
		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s(%d)", formatPoint(item.from), formatPoint(item.to), count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s", formatPoint(item.from), formatPoint(item.to)))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// convertXGMoveToStringWithHits converts XG move format to readable string with hit indicators (*)
// It uses the initial position to detect when a blot is hit
// Moves are displayed from the player on roll's perspective (standard notation).
// activePlayer: XG encoding: 1 = Player 1 (X), -1 = Player 2 (O)
func (d *Database) convertXGMoveToStringWithHits(playedMove [8]int32, initialPos *xgparser.Position, activePlayer int32) string {
	if initialPos == nil {
		return d.convertXGMoveToString(playedMove, activePlayer)
	}

	// Note: No mirroring needed - moves are always from the roller's perspective

	// Create a mutable copy of the position to track changes as we process moves
	// XG position format: Checkers[0-23] are points 1-24, [24]=player bar, [25]=opponent bar
	// Positive values = player's checkers, negative = opponent's checkers
	positionCopy := make([]int8, 26)
	copy(positionCopy, initialPos.Checkers[:])

	// Parse raw moves into from/to pairs and track hits
	var items []xgMoveWithHit
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		// Check for end of move marker (-1 when from is also -1)
		if from == -1 {
			break
		}
		// Handle implicit bear-off: XG sometimes encodes bear-off as to=-1 or to<=0
		// when the calculated destination (from - die) would be <= 0
		// This happens when bearing off from the home board (points 1-6)
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2 // Convert to explicit bear-off
		}
		// Skip invalid point values (should be 1-24 for points, 25 for bar, -2 for bear off)
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}

		// Check if this move hits a blot
		// The destination point must have exactly one opponent checker (negative value in XG format)
		isHit := false
		if to >= 1 && to <= 24 {
			// Convert to 0-based index (XG uses 0-23 for points 1-24)
			toIdx := to - 1
			if positionCopy[toIdx] == -1 {
				isHit = true
				// Update position: opponent checker goes to bar
				positionCopy[toIdx] = 0
			}
		}

		// Update position: move our checker
		if from >= 1 && from <= 24 {
			fromIdx := from - 1
			if positionCopy[fromIdx] > 0 {
				positionCopy[fromIdx]--
			}
		} else if from == 25 {
			// From bar
			if positionCopy[24] > 0 {
				positionCopy[24]--
			}
		}

		if to >= 1 && to <= 24 {
			toIdx := to - 1
			positionCopy[toIdx]++
		}

		// Store directly - no conversion needed since moves are already in roller's perspective
		items = append(items, xgMoveWithHit{from: from, to: to, isHit: isHit})
	}

	if len(items) == 0 {
		return "Cannot Move"
	}

	// Try to merge consecutive checker slides - but preserve hit info for the final move
	// For slides, only the last move in a chain can be a hit
	items = d.mergeSlidesWithHits(items)

	// Sort moves by 'from' point descending (standard backgammon notation)
	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	// Format each move as string
	formatPoint := func(p int32) string {
		if p == 25 {
			return "bar"
		} else if p == -2 {
			return "off"
		} else if p >= 1 && p <= 24 {
			return fmt.Sprintf("%d", p)
		}
		return fmt.Sprintf("?%d", p)
	}

	// Build move string, grouping identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		allHits := item.isHit
		// Count consecutive identical moves
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
				allHits = allHits && items[j].isHit
			} else {
				break
			}
		}

		hitMarker := ""
		if item.isHit || allHits {
			hitMarker = "*"
		}

		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s%s(%d)", formatPoint(item.from), formatPoint(item.to), hitMarker, count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s%s", formatPoint(item.from), formatPoint(item.to), hitMarker))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// mergeSlidesWithHits merges consecutive moves of the same checker, preserving hit info
// For example: 14/12 12/8 becomes 14/8, but only if 12 was just a waypoint (not hit)
// If there was a hit at the intermediate point, we keep both moves to show the hit
func (d *Database) mergeSlidesWithHits(items []xgMoveWithHit) []xgMoveWithHit {
	if len(items) <= 1 {
		return items
	}

	// Try to merge chains: if move[i].to == move[j].from and move[i] is not a hit,
	// they can be merged (the intermediate point was just a waypoint)
	result := make([]xgMoveWithHit, 0, len(items))
	used := make([]bool, len(items))

	for i := 0; i < len(items); i++ {
		if used[i] {
			continue
		}

		// Start a chain from this item
		chainFrom := items[i].from
		chainTo := items[i].to
		chainHit := items[i].isHit
		used[i] = true

		// Only extend if the current segment doesn't end with a hit
		// (we want to show hits, so don't merge past them)
		if !chainHit {
			// Extend chain forward: find items where from == chainTo
			for changed := true; changed; {
				changed = false
				for j := 0; j < len(items); j++ {
					if used[j] {
						continue
					}
					if items[j].from == chainTo {
						chainTo = items[j].to
						chainHit = items[j].isHit
						used[j] = true
						changed = true
						// If this new segment has a hit, stop extending
						if chainHit {
							break
						}
					}
				}
				if chainHit {
					break
				}
			}
		}

		result = append(result, xgMoveWithHit{from: chainFrom, to: chainTo, isHit: chainHit})
	}

	return result
}

// xgMoveWithHit represents a single move in XG format with hit information
type xgMoveWithHit struct {
	from  int32
	to    int32
	isHit bool
}

// xgMoveItem is unused but kept for documentation
type xgMoveItem struct {
	from int32
	to   int32
}

// mergeSlides merges consecutive moves of the same checker
// For example: 14/12 12/8 becomes 14/8
func (d *Database) mergeSlides(fromPts, toPts []int32) ([]int32, []int32) {
	if len(fromPts) <= 1 {
		return fromPts, toPts
	}

	// Count how many times each destination point is used
	// If multiple moves end at the same point, we should NOT merge through that point
	// because it means different checkers are moving, not the same checker sliding
	toCount := make(map[int32]int)
	for _, t := range toPts {
		toCount[t]++
	}

	// Also count how many times each point is a source
	fromCount := make(map[int32]int)
	for _, f := range fromPts {
		fromCount[f]++
	}

	// Try to merge chains: if move[i].to == move[j].from, they can be merged
	// BUT only if that intermediate point appears exactly once as a destination AND once as a source
	resultFrom := make([]int32, 0, len(fromPts))
	resultTo := make([]int32, 0, len(toPts))
	used := make([]bool, len(fromPts))

	for i := 0; i < len(fromPts); i++ {
		if used[i] {
			continue
		}

		// Start a chain from this item
		chainFrom := fromPts[i]
		chainTo := toPts[i]
		used[i] = true

		// Extend chain forward: find items where from == chainTo
		// Only merge if the intermediate point is not used by multiple checkers
		for changed := true; changed; {
			changed = false
			for j := 0; j < len(fromPts); j++ {
				if used[j] {
					continue
				}
				if fromPts[j] == chainTo {
					// Check if this is a valid merge (same checker moving)
					// Don't merge if chainTo is a destination for multiple moves
					// or if chainTo is a source for multiple moves
					// This indicates different checkers
					if toCount[chainTo] > 1 || fromCount[chainTo] > 1 {
						continue // Don't merge - different checkers
					}
					chainTo = toPts[j]
					used[j] = true
					changed = true
				}
			}
		}

		resultFrom = append(resultFrom, chainFrom)
		resultTo = append(resultTo, chainTo)
	}

	return resultFrom, resultTo
}

// convertCubeAction converts cube action code to string
func (d *Database) convertCubeAction(action int32) string {
	switch action {
	case 0:
		return "No Double"
	case 1:
		return "Double"
	case 2:
		return "Take"
	case 3:
		return "Pass"
	default:
		return fmt.Sprintf("Unknown(%d)", action)
	}
}

// ErrDuplicateMatch is returned when attempting to import a match that already exists
var ErrDuplicateMatch = fmt.Errorf("duplicate match: this match has already been imported")

// ComputeMatchHash generates a unique hash for a match based on full match transcription
// This is used to detect duplicate imports - includes all moves and decisions
func ComputeMatchHash(match *xgparser.Match) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2Name))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, match.Metadata.MatchLength))

	// Include full game transcription
	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|",
			gameIdx, game.InitialScore[0], game.InitialScore[1], game.Winner, game.PointsWon))

		// Include all moves in the game
		for moveIdx, move := range game.Moves {
			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, move.MoveType))

			if move.CheckerMove != nil {
				// Include dice and played move
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%v|",
					move.CheckerMove.Dice[0], move.CheckerMove.Dice[1],
					move.CheckerMove.PlayedMove))
			}

			if move.CubeMove != nil {
				// Include cube action details
				hashBuilder.WriteString(fmt.Sprintf("c%d|", move.CubeMove.CubeAction))
			}
		}
	}

	// Compute SHA256 hash of the full transcription
	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// computeMatchHashFromStoredData computes a hash for existing matches in the database
// This is used during migration when we don't have access to the original XG file
func computeMatchHashFromStoredData(db *sql.DB, matchID int64, p1Name, p2Name string, matchLength int32) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(p1Name))
	p2 := strings.TrimSpace(strings.ToLower(p2Name))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, matchLength))

	// Query all games for this match
	gameRows, err := db.Query(`
		SELECT id, game_number, initial_score_1, initial_score_2, winner, points_won 
		FROM game WHERE match_id = ? ORDER BY game_number`, matchID)
	if err != nil {
		// Fallback to simple hash
		hash := sha256.Sum256([]byte(hashBuilder.String()))
		return hex.EncodeToString(hash[:])
	}
	defer gameRows.Close()

	for gameRows.Next() {
		var gameID int64
		var gameNum, initScore1, initScore2, winner, pointsWon int32
		if err := gameRows.Scan(&gameID, &gameNum, &initScore1, &initScore2, &winner, &pointsWon); err != nil {
			continue
		}

		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|", gameNum, initScore1, initScore2, winner, pointsWon))

		// Query all moves for this game
		moveRows, err := db.Query(`
			SELECT move_number, move_type, dice_1, dice_2, checker_move, cube_action 
			FROM move WHERE game_id = ? ORDER BY move_number`, gameID)
		if err != nil {
			continue
		}

		for moveRows.Next() {
			var moveNum int32
			var moveType string
			var dice1, dice2 int32
			var checkerMove, cubeAction sql.NullString
			if err := moveRows.Scan(&moveNum, &moveType, &dice1, &dice2, &checkerMove, &cubeAction); err != nil {
				continue
			}

			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveNum, moveType))
			if moveType == "checker" && checkerMove.Valid {
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%s|", dice1, dice2, checkerMove.String))
			}
			if moveType == "cube" && cubeAction.Valid {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", cubeAction.String))
			}
		}
		moveRows.Close()
	}

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// CheckMatchExists checks if a match with the given hash already exists in the database
// Returns the existing match ID if found, 0 otherwise
func (d *Database) CheckMatchExists(matchHash string) (int64, error) {
	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM match WHERE match_hash = ?`, matchHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error checking for duplicate match: %w", err)
	}
	return existingID, nil
}

// CheckMatchExistsLocked is the same as CheckMatchExists but doesn't acquire the lock
// Use this when you already hold the database lock
func (d *Database) checkMatchExistsLocked(matchHash string) (int64, error) {
	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM match WHERE match_hash = ?`, matchHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error checking for duplicate match: %w", err)
	}
	return existingID, nil
}

// GetAllMatches returns all matches from the database
func (d *Database) GetAllMatches() ([]Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT id, player1_name, player2_name, event, location, round, 
		       match_length, match_date, import_date, file_path, game_count
		FROM match
		ORDER BY match_date DESC
	`)
	if err != nil {
		fmt.Println("Error loading matches:", err)
		return nil, err
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		err := rows.Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
			&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount)
		if err != nil {
			fmt.Println("Error scanning match:", err)
			continue
		}
		matches = append(matches, m)
	}

	return matches, nil
}

// GetMatchByID returns a specific match by ID
func (d *Database) GetMatchByID(matchID int64) (*Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var m Match
	err := d.db.QueryRow(`
		SELECT id, player1_name, player2_name, event, location, round,
		       match_length, match_date, import_date, file_path, game_count
		FROM match
		WHERE id = ?
	`, matchID).Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
		&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("match not found")
		}
		fmt.Println("Error loading match:", err)
		return nil, err
	}

	return &m, nil
}

// GetGamesByMatch returns all games in a match
func (d *Database) GetGamesByMatch(matchID int64) ([]Game, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT id, match_id, game_number, initial_score_1, initial_score_2,
		       winner, points_won, move_count
		FROM game
		WHERE match_id = ?
		ORDER BY game_number ASC
	`, matchID)
	if err != nil {
		fmt.Println("Error loading games:", err)
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		var score1, score2 int32
		err := rows.Scan(&g.ID, &g.MatchID, &g.GameNumber, &score1, &score2,
			&g.Winner, &g.PointsWon, &g.MoveCount)
		if err != nil {
			fmt.Println("Error scanning game:", err)
			continue
		}
		g.InitialScore = [2]int32{score1, score2}
		games = append(games, g)
	}

	return games, nil
}

// GetMovesByGame returns all moves in a game
func (d *Database) GetMovesByGame(gameID int64) ([]Move, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT id, game_id, move_number, move_type, position_id, player,
		       dice_1, dice_2, checker_move, cube_action
		FROM move
		WHERE game_id = ?
		ORDER BY move_number ASC
	`, gameID)
	if err != nil {
		fmt.Println("Error loading moves:", err)
		return nil, err
	}
	defer rows.Close()

	var moves []Move
	for rows.Next() {
		var m Move
		var dice1, dice2 int32
		var checkerMove, cubeAction sql.NullString
		err := rows.Scan(&m.ID, &m.GameID, &m.MoveNumber, &m.MoveType, &m.PositionID,
			&m.Player, &dice1, &dice2, &checkerMove, &cubeAction)
		if err != nil {
			fmt.Println("Error scanning move:", err)
			continue
		}
		m.Dice = [2]int32{dice1, dice2}
		if checkerMove.Valid {
			m.CheckerMove = checkerMove.String
		}
		if cubeAction.Valid {
			m.CubeAction = cubeAction.String
		}
		moves = append(moves, m)
	}

	return moves, nil
}

// DeleteMatch deletes a match and all associated games, moves, and analysis
func (d *Database) DeleteMatch(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Foreign key constraints will cascade delete to game, move, and move_analysis
	_, err := d.db.Exec(`DELETE FROM match WHERE id = ?`, matchID)
	if err != nil {
		fmt.Println("Error deleting match:", err)
		return err
	}

	return nil
}

// GetMatchMovePositions returns all positions from a match in chronological order
// Positions are returned as they were stored (from player on roll POV)
// The frontend is responsible for mirroring display if needed
func (d *Database) GetMatchMovePositions(matchID int64) ([]MatchMovePosition, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Get match info for player names
	var player1Name, player2Name string
	err := d.db.QueryRow(`
		SELECT player1_name, player2_name 
		FROM match 
		WHERE id = ?
	`, matchID).Scan(&player1Name, &player2Name)
	if err != nil {
		return nil, fmt.Errorf("match not found: %w", err)
	}

	// Get all moves across all games in chronological order
	// Join with game table to get game number and position table to get position data
	rows, err := d.db.Query(`
		SELECT 
			m.id as move_id,
			m.game_id,
			g.game_number,
			m.move_number,
			m.move_type,
			m.player,
			m.position_id,
			p.state as position_state
		FROM move m
		INNER JOIN game g ON m.game_id = g.id
		INNER JOIN position p ON m.position_id = p.id
		WHERE g.match_id = ?
		ORDER BY g.game_number ASC, m.move_number ASC
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to query moves: %w", err)
	}
	defer rows.Close()

	var movePositions []MatchMovePosition
	for rows.Next() {
		var moveID, gameID, positionID int64
		var gameNumber, moveNumber, player int32
		var moveType, positionState string

		err := rows.Scan(&moveID, &gameID, &gameNumber, &moveNumber, &moveType, &player, &positionID, &positionState)
		if err != nil {
			fmt.Printf("Error scanning move: %v\n", err)
			continue
		}

		// Unmarshal position
		var position Position
		err = json.Unmarshal([]byte(positionState), &position)
		if err != nil {
			fmt.Printf("Error unmarshalling position: %v\n", err)
			continue
		}

		movePos := MatchMovePosition{
			Position:     position,
			MoveID:       moveID,
			GameID:       gameID,
			GameNumber:   gameNumber,
			MoveNumber:   moveNumber,
			MoveType:     moveType,
			PlayerOnRoll: player,
			Player1Name:  player1Name,
			Player2Name:  player2Name,
		}

		movePositions = append(movePositions, movePos)
	}

	return movePositions, nil
}

// GetDatabaseStats returns statistics about the database
func (d *Database) GetDatabaseStats() (map[string]interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	stats := make(map[string]interface{})

	// Count positions
	var posCount int64
	err := d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount)
	if err != nil {
		return nil, err
	}
	stats["position_count"] = posCount

	// Count analyses
	var analysisCount int64
	err = d.db.QueryRow(`SELECT COUNT(*) FROM analysis`).Scan(&analysisCount)
	if err != nil {
		return nil, err
	}
	stats["analysis_count"] = analysisCount

	// Count matches
	var matchCount int64
	err = d.db.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&matchCount)
	if err != nil {
		// Table might not exist in older databases
		stats["match_count"] = int64(0)
	} else {
		stats["match_count"] = matchCount
	}

	// Count games
	var gameCount int64
	err = d.db.QueryRow(`SELECT COUNT(*) FROM game`).Scan(&gameCount)
	if err != nil {
		stats["game_count"] = int64(0)
	} else {
		stats["game_count"] = gameCount
	}

	// Count moves
	var moveCount int64
	err = d.db.QueryRow(`SELECT COUNT(*) FROM move`).Scan(&moveCount)
	if err != nil {
		stats["move_count"] = int64(0)
	} else {
		stats["move_count"] = moveCount
	}

	return stats, nil
}
