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
        CREATE TABLE IF NOT EXISTS position (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            state TEXT
        )
    `)

    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS analysis (
            id INTEGER PRIMARY KEY,
            position_id INTEGER,
            data JSON,
            FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
        )
    `);


    if err != nil {
        return nil, err
    }
    return db, nil
}

func SavePosition(db *sql.DB, position Position) error {
    positionJSON, err := json.Marshal(position)
    if err != nil {
        return err
    }

    _, err = db.Exec(`INSERT INTO position (state) VALUES (?)`, string(positionJSON))
    return err
}

func LoadPosition(db *sql.DB, id int) (*Position, error) {
    var stateJSON string

    err := db.QueryRow(`SELECT state from position WHERE id = ?`, id).Scan(
        &stateJSON)
    if err != nil {
        return nil, err
    }

    var state Position
    err = json.Unmarshal([]byte(stateJSON), &state)
    if err != nil {
        return nil, err
    }

    return &state, nil
}
