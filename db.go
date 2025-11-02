package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync" // Import sync package
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	db              *sql.DB
	mu              sync.Mutex // Add a mutex to the Database struct
	importCancelled bool       // Flag to cancel ongoing import
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
	d.mu.Lock()
	defer d.mu.Unlock()
	d.importCancelled = true
	fmt.Println("Import cancellation requested")
}

// isImportCancelled checks if import has been cancelled (internal method, no lock needed as it's called within locked context)
func (d *Database) isImportCancelled() bool {
	return d.importCancelled
}

// resetImportCancellation resets the cancellation flag (internal method)
func (d *Database) resetImportCancellation() {
	d.importCancelled = false
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
