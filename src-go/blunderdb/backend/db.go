package backend

import (
    "database/sql"
    "encoding/json"
    _ "modernc.org/sqlite"
)

func SetupDatabase() (*sql.DB, error) {
    db, err := sql.Open("sqlite", "backgammon.db")
    if err != nil {
        return nil, err
    }

    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS game_state (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            state TEXT
        )
    `)
    if err != nil {
        return nil, err
    }
    return db, nil
}

func SaveGameState(db *sql.DB, state GameState) error {
    stateJSON, err := json.Marshal(state)
    if err != nil {
        return err
    }

    _, err = db.Exec(`INSERT INTO game_state (state) VALUES (?)`,
        string(stateJSON))
    return err
}

func LoadGameState(db *sql.DB, id int) (*GameState, error) {
    var stateJSON string

    err := db.QueryRow(`SELECT state from game_state WHERE id = ?`, id).Scan(
        &stateJSON)
    if err != nil {
        return nil, err
    }

    var state GameState
    err = json.Unmarshal([]byte(stateJSON), &state)
    if err != nil {
        return nil, err
    }

    return &state, nil
}
