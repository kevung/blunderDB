package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

func NewDatabase() *Database {
	return &Database{}
}

func (d *Database) SetupDatabase(path string) error {
	// Open the database using string path
	var err error
	d.db, err = sql.Open("sqlite", path)
	if err != nil {
		fmt.Println("Error opening database:", err)
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

	// Insert or update the database version
	_, err = d.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		fmt.Println("Error inserting database version:", err)
		return err
	}

	return nil
}

func (d *Database) OpenDatabase(path string) error {
	// Open the database using string path
	var err error
	d.db, err = sql.Open("sqlite", path)
	if err != nil {
		fmt.Println("Error opening database:", err)
		return err
	}

	// Check if the required tables exist
	requiredTables := []string{"position", "analysis", "comment", "metadata"}
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
	var dbVersion string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&dbVersion)
	if err != nil {
		fmt.Println("Error querying database version:", err)
		return "", err
	}
	return dbVersion, nil
}

func (d *Database) PositionExists(position Position) (map[string]interface{}, error) {
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
	fmt.Printf("Loading analysis for position ID: %d\n", positionID) // Add logging

	var analysisJSON string
	err := d.db.QueryRow(`SELECT data from analysis WHERE position_id = ?`, positionID).Scan(&analysisJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("No analysis found for position ID: %d\n", positionID) // Add logging
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

	return nil
}

func (d *Database) DeleteAnalysis(positionID int64) error {
	_, err := d.db.Exec(`DELETE FROM analysis WHERE position_id = ?`, positionID)
	if err != nil {
		fmt.Println("Error deleting analysis:", err)
		return err
	}
	return nil
}

func (d *Database) DeleteComment(positionID int64) error {
	_, err := d.db.Exec(`DELETE FROM comment WHERE position_id = ?`, positionID)
	if err != nil {
		fmt.Println("Error deleting comment:", err)
		return err
	}
	return nil
}

// SaveComment saves a comment for a given position ID
func (d *Database) SaveComment(positionID int64, text string) error {
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

func (d *Database) LoadPositionsByFilters(filter Position, includeCube bool, includeScore bool, pipCountFilter string, winRateFilter string, gammonRateFilter string, backgammonRateFilter string, player2WinRateFilter string, player2GammonRateFilter string, player2BackgammonRateFilter string, player1CheckerOffFilter string, player2CheckerOffFilter string, player1BackCheckerFilter string, player2BackCheckerFilter string, player1CheckerInZoneFilter string, player2CheckerInZoneFilter string, searchText string, player1AbsolutePipCountFilter string, equityFilter string, decisionTypeFilter bool, diceRollFilter bool, movePatternFilter string) ([]Position, error) {
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

		fmt.Printf("Checking position ID: %d\n", position.ID) // Add logging

		if position.MatchesCheckerPosition(filter) &&
			(!includeCube || position.MatchesCubePosition(filter)) &&
			(!includeScore || position.MatchesScorePosition(filter)) &&
			(!decisionTypeFilter || position.MatchesDecisionType(filter)) &&
			(pipCountFilter == "" || position.MatchesPipCountFilter(pipCountFilter)) &&
			(winRateFilter == "" || position.MatchesWinRate(winRateFilter, d)) &&
			(gammonRateFilter == "" || position.MatchesGammonRate(gammonRateFilter, d)) &&
			(backgammonRateFilter == "" || position.MatchesBackgammonRate(backgammonRateFilter, d)) &&
			(player2WinRateFilter == "" || position.MatchesPlayer2WinRate(player2WinRateFilter, d)) &&
			(player2GammonRateFilter == "" || position.MatchesPlayer2GammonRate(player2GammonRateFilter, d)) &&
			(player2BackgammonRateFilter == "" || position.MatchesPlayer2BackgammonRate(player2BackgammonRateFilter, d)) &&
			(player1CheckerOffFilter == "" || position.MatchesPlayer1CheckerOff(player1CheckerOffFilter)) &&
			(player2CheckerOffFilter == "" || position.MatchesPlayer2CheckerOff(player2CheckerOffFilter)) &&
			(player1BackCheckerFilter == "" || position.MatchesPlayer1BackChecker(player1BackCheckerFilter)) &&
			(player2BackCheckerFilter == "" || position.MatchesPlayer2BackChecker(player2BackCheckerFilter)) &&
			(player1CheckerInZoneFilter == "" || position.MatchesPlayer1CheckerInZone(player1CheckerInZoneFilter)) &&
			(player2CheckerInZoneFilter == "" || position.MatchesPlayer2CheckerInZone(player2CheckerInZoneFilter)) &&
			(searchText == "" || position.MatchesSearchText(searchText, d)) &&
			(player1AbsolutePipCountFilter == "" || position.MatchesPlayer1AbsolutePipCount(player1AbsolutePipCountFilter)) &&
			(equityFilter == "" || position.MatchesEquityFilter(equityFilter, d)) &&
			(!diceRollFilter || position.MatchesDiceRoll(filter)) {
			if movePatternFilter != "" {
				fmt.Printf("Checking move pattern filter: %s for position ID: %d\n", movePatternFilter, position.ID) // Add logging
				if position.MatchesMovePattern(movePatternFilter, d) {
					positions = append(positions, position)
				}
			} else {
				positions = append(positions, position)
			}
		}
	}

	fmt.Println("Loaded positions by filters:", positions)
	return positions, nil
}

// Add MatchesDecisionType method to Position type
func (p *Position) MatchesDecisionType(filter Position) bool {
	return p.DecisionType == filter.DecisionType && p.PlayerOnRoll == filter.PlayerOnRoll
}

func (p *Position) MatchesSearchText(searchText string, d *Database) bool {
	comment, err := d.LoadComment(p.ID)
	if err != nil {
		fmt.Printf("Error loading comment for position ID: %d, error: %v\n", p.ID, err)
		return false
	}
	searchTextArray := strings.Split(strings.ToLower(searchText), ";")
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

	if strings.HasPrefix(filter, "B>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate >= value
	} else if strings.HasPrefix(filter, "B<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate <= value
	} else if strings.HasPrefix(filter, "B") {
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

	if strings.HasPrefix(filter, "b>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate >= value
	} else if strings.HasPrefix(filter, "b<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			fmt.Printf("Error parsing filter value: %s\n", filter[2:])
			return false
		}
		return backgammonRate <= value
	} else if strings.HasPrefix(filter, "b") {
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
	for i := 14; i <= 25; i++ {
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
	for i := 0; i <= 11; i++ {
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

	if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		move := strings.ToLower(analysis.CheckerAnalysis.Moves[0].Move)
		movePatterns := strings.Split(strings.ToLower(filter), ";")
		for _, pattern := range movePatterns {
			if strings.Contains(move, pattern) {
				fmt.Printf("Position ID: %d, Checker decision, Move: %s, Filter: %s\n", p.ID, move, pattern) // Add logging
				return true
			}
		}
	} else if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		movePatterns := strings.Split(strings.ToLower(filter), ";")
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
	return DatabaseVersion, nil
}

func colorName(color int) string {
	switch color {
	case Black:
		return "Black"
	case White:
		return "White"
	default:
		return "None"
	}
}

func (d *Database) LoadMetadata() (map[string]string, error) {
	rows, err := d.db.Query(`SELECT key, value FROM metadata WHERE key IN ('user', 'description', 'dateOfCreation')`)
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
	for key, value := range metadata {
		_, err := d.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
		if err != nil {
			fmt.Println("Error saving metadata:", err)
			return err
		}
	}
	return nil
}
