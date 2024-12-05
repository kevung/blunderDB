package main

import (
	"database/sql"
	"encoding/json"
	"fmt"

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
	return nil
}

func (d *Database) PositionExists(position Position) (int64, bool, error) {
	positionJSON, err := json.Marshal(position)
	if err != nil {
		fmt.Println("Error marshalling position:", err)
		return 0, false, err
	}

	var id int64
	err = d.db.QueryRow(`SELECT id FROM position WHERE state = ?`, string(positionJSON)).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		fmt.Println("Error querying position:", err)
		return 0, false, err
	}

	return id, true, nil
}

func (d *Database) SavePosition(position Position) (int64, error) {
	positionID, exists, err := d.PositionExists(position)
	if err != nil {
		fmt.Println("Error checking if position exists:", err)
		return 0, err
	}

	positionJSON, err := json.Marshal(position)
	if err != nil {
		fmt.Println("Error marshalling position:", err)
		return 0, err
	}

	if exists {
		_, err = d.db.Exec(`UPDATE position SET state = ? WHERE id = ?`, string(positionJSON), positionID)
		if err != nil {
			fmt.Println("Error updating position:", err)
			return 0, err
		}
	} else {
		result, err := d.db.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
		if err != nil {
			fmt.Println("Error inserting position:", err)
			return 0, err
		}

		positionID, err = result.LastInsertId()
		if err != nil {
			fmt.Println("Error getting last insert ID:", err)
			return 0, err
		}
	}

	return positionID, nil
}

func (d *Database) SaveAnalysis(positionID int64, analysis PositionAnalysis) error {
	analysisJSON, err := json.Marshal(analysis)
	if err != nil {
		fmt.Println("Error marshalling analysis:", err)
		return err
	}

	// Check if an analysis already exists for the given position ID
	var existingID int64
	err = d.db.QueryRow(`SELECT id FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error querying analysis:", err)
		return err
	}

	if existingID > 0 {
		// Update the existing analysis
		_, err = d.db.Exec(`UPDATE analysis SET data = ? WHERE id = ?`, string(analysisJSON), existingID)
		if err != nil {
			fmt.Println("Error updating analysis:", err)
			return err
		}
	} else {
		// Insert a new analysis
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

func (d *Database) LoadAnalysis(positionID int) (*PositionAnalysis, error) {
	var analysisJSON string

	err := d.db.QueryRow(`SELECT data from analysis WHERE position_id = ?`, positionID).Scan(&analysisJSON)
	if err != nil {
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
	rows, err := d.db.Query(`SELECT state FROM position`)
	if err != nil {
		fmt.Println("Error loading positions:", err)
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var stateJSON string
		if err := rows.Scan(&stateJSON); err != nil {
			fmt.Println("Error scanning position:", err)
			return nil, err
		}

		var position Position
		if err := json.Unmarshal([]byte(stateJSON), &position); err != nil {
			fmt.Println("Error unmarshalling position:", err)
			return nil, err
		}
		positions = append(positions, position)
	}

	fmt.Println("Loaded positions:", positions)
	return positions, nil
}

func (d *Database) LoadAllAnalyses() ([]PositionAnalysis, error) {
	rows, err := d.db.Query(`SELECT data FROM analysis`)
	if err != nil {
		fmt.Println("Error loading analyses:", err)
		return nil, err
	}
	defer rows.Close()

	var analyses []PositionAnalysis
	for rows.Next() {
		var analysisJSON string
		if err := rows.Scan(&analysisJSON); err != nil {
			fmt.Println("Error scanning analysis:", err)
			return nil, err
		}

		var analysis PositionAnalysis
		if err := json.Unmarshal([]byte(analysisJSON), &analysis); err != nil {
			fmt.Println("Error unmarshalling analysis:", err)
			return nil, err
		}
		analyses = append(analyses, analysis)
	}

	fmt.Println("Loaded analyses:", analyses)
	return analyses, nil
}
