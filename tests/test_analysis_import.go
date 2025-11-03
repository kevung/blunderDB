package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// Inline minimal Database structure for testing
type Database struct {
	db *sql.DB
}

func NewDatabase() *Database {
	return &Database{}
}

func (d *Database) SetupDatabase(path string) error {
	var err error
	d.db, err = sql.Open("sqlite", path)
	if err != nil {
		return err
	}

	// Basic table creation (simplified)
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		);
		CREATE TABLE IF NOT EXISTS analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER,
			data TEXT,
			FOREIGN KEY (position_id) REFERENCES position(id)
		);
		CREATE TABLE IF NOT EXISTS move_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			move_id INTEGER,
			data TEXT
		);
	`)
	return err
}

// Minimal structures needed for testing
type PositionAnalysis struct {
	PositionID            int                       `json:"position_id"`
	AnalysisType          string                    `json:"analysis_type"`
	AnalysisEngineVersion string                    `json:"analysis_engine_version"`
	CreationDate          time.Time                 `json:"creation_date"`
	LastModifiedDate      time.Time                 `json:"last_modified_date"`
	CheckerAnalysis       *CheckerAnalysisData      `json:"checker_analysis,omitempty"`
	DoublingCubeAnalysis  *DoublingCubeAnalysisData `json:"doubling_cube_analysis,omitempty"`
}

type CheckerAnalysisData struct {
	Moves []CheckerMoveData `json:"moves"`
}

type CheckerMoveData struct {
	Index                    int      `json:"index"`
	AnalysisDepth            string   `json:"analysis_depth"`
	Move                     string   `json:"move"`
	Equity                   float64  `json:"equity"`
	EquityError              *float64 `json:"equity_error,omitempty"`
	PlayerWinChance          float64  `json:"player_win_chance"`
	PlayerGammonChance       float64  `json:"player_gammon_chance"`
	PlayerBackgammonChance   float64  `json:"player_backgammon_chance"`
	OpponentWinChance        float64  `json:"opponent_win_chance"`
	OpponentGammonChance     float64  `json:"opponent_gammon_chance"`
	OpponentBackgammonChance float64  `json:"opponent_backgammon_chance"`
}

type DoublingCubeAnalysisData struct {
	AnalysisDepth             string  `json:"analysis_depth"`
	BestCubeAction            string  `json:"best_cube_action"`
	PlayerWinChances          float64 `json:"player_win_chances"`
	PlayerGammonChances       float64 `json:"player_gammon_chances"`
	PlayerBackgammonChances   float64 `json:"player_backgammon_chances"`
	OpponentWinChances        float64 `json:"opponent_win_chances"`
	OpponentGammonChances     float64 `json:"opponent_gammon_chances"`
	OpponentBackgammonChances float64 `json:"opponent_backgammon_chances"`
	CubelessNoDoubleEquity    float64 `json:"cubeless_no_double_equity"`
	CubelessDoubleEquity      float64 `json:"cubeless_double_equity"`
	CubefulNoDoubleEquity     float64 `json:"cubeful_no_double_equity"`
	CubefulNoDoubleError      float64 `json:"cubeful_no_double_error"`
	CubefulDoubleTakeEquity   float64 `json:"cubeful_double_take_equity"`
	CubefulDoubleTakeError    float64 `json:"cubeful_double_take_error"`
	CubefulDoublePassEquity   float64 `json:"cubeful_double_pass_equity"`
	CubefulDoublePassError    float64 `json:"cubeful_double_pass_error"`
	WrongPassPercentage       float64 `json:"wrong_pass_percentage"`
	WrongTakePercentage       float64 `json:"wrong_take_percentage"`
}

func main() {
	// Create a test database
	dbPath := "test_analysis.db"

	// Remove old test database if exists
	os.Remove(dbPath)

	// Create database instance
	db := NewDatabase()
	err := db.SetupDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}

	// Import the test XG match
	fmt.Println("Importing test.xg...")
	_, err = db.ImportXGMatch("test.xg")
	if err != nil {
		log.Fatalf("Failed to import match: %v", err)
	}

	// Query analysis table
	fmt.Println("\nQuerying analysis table...")
	rows, err := db.db.Query(`
		SELECT a.id, a.position_id, p.decision_type, a.data 
		FROM analysis a
		JOIN position p ON a.position_id = p.id
		ORDER BY a.id
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("Failed to query analysis: %v", err)
	}
	defer rows.Close()

	analysisCount := 0
	checkerCount := 0
	cubeCount := 0

	for rows.Next() {
		var id, positionID int64
		var decisionType string
		var dataJSON string

		err = rows.Scan(&id, &positionID, &decisionType, &dataJSON)
		if err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		analysisCount++

		// Parse the analysis data
		var analysis PositionAnalysis
		err = json.Unmarshal([]byte(dataJSON), &analysis)
		if err != nil {
			log.Printf("Warning: Failed to unmarshal analysis %d: %v", id, err)
			continue
		}

		fmt.Printf("\nAnalysis ID: %d, Position ID: %d\n", id, positionID)
		fmt.Printf("  Decision Type: %s\n", decisionType)
		fmt.Printf("  Analysis Type: %s\n", analysis.AnalysisType)
		fmt.Printf("  Engine: %s\n", analysis.AnalysisEngineVersion)

		if analysis.CheckerAnalysis != nil {
			checkerCount++
			fmt.Printf("  Checker Analysis: %d moves\n", len(analysis.CheckerAnalysis.Moves))
			if len(analysis.CheckerAnalysis.Moves) > 0 {
				bestMove := analysis.CheckerAnalysis.Moves[0]
				fmt.Printf("    Best Move: %s\n", bestMove.Move)
				fmt.Printf("    Equity: %.4f\n", bestMove.Equity)
				fmt.Printf("    Win%%: %.2f%%\n", bestMove.PlayerWinChance)
			}
		}

		if analysis.DoublingCubeAnalysis != nil {
			cubeCount++
			fmt.Printf("  Cube Analysis:\n")
			fmt.Printf("    Best Action: %s\n", analysis.DoublingCubeAnalysis.BestCubeAction)
			fmt.Printf("    No Double Equity: %.4f\n", analysis.DoublingCubeAnalysis.CubefulNoDoubleEquity)
			fmt.Printf("    Double/Take Equity: %.4f\n", analysis.DoublingCubeAnalysis.CubefulDoubleTakeEquity)
			fmt.Printf("    Double/Pass Equity: %.4f\n", analysis.DoublingCubeAnalysis.CubefulDoublePassEquity)
		}
	}

	if err = rows.Err(); err != nil {
		log.Fatalf("Row iteration error: %v", err)
	}

	// Count total positions
	var totalPositions int
	err = db.db.QueryRow("SELECT COUNT(*) FROM position").Scan(&totalPositions)
	if err != nil {
		log.Fatalf("Failed to count positions: %v", err)
	}

	// Count total analyses
	var totalAnalyses int
	err = db.db.QueryRow("SELECT COUNT(*) FROM analysis").Scan(&totalAnalyses)
	if err != nil {
		log.Fatalf("Failed to count analyses: %v", err)
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total Positions: %d\n", totalPositions)
	fmt.Printf("Total Analyses: %d\n", totalAnalyses)
	fmt.Printf("Checker Analyses: %d\n", checkerCount)
	fmt.Printf("Cube Analyses: %d\n", cubeCount)

	// Verify move_analysis table also has data
	var moveAnalyses int
	err = db.db.QueryRow("SELECT COUNT(*) FROM move_analysis").Scan(&moveAnalyses)
	if err == nil {
		fmt.Printf("Move Analyses: %d\n", moveAnalyses)
	}

	fmt.Println("\n✓ Test completed successfully!")
}
