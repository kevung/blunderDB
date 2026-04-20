package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// GetAllMatches returns all matches from the database
func (d *Database) GetAllMatches() ([]Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT m.id, m.player1_name, m.player2_name, m.event, m.location, m.round, 
		       m.match_length, m.match_date, m.import_date, m.file_path, m.game_count,
		       m.tournament_id, COALESCE(t.name, '') as tournament_name,
		       COALESCE(m.last_visited_position, -1) as last_visited_position,
		       COALESCE(m.comment, '') as comment
		FROM match m
		LEFT JOIN tournament t ON m.tournament_id = t.id
		ORDER BY CASE WHEN m.match_date IS NULL OR m.match_date = '' OR m.match_date = '0001-01-01T00:00:00Z' THEN m.import_date ELSE m.match_date END DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		err := rows.Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
			&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount,
			&m.TournamentID, &m.TournamentName, &m.LastVisitedPosition, &m.Comment)
		if err != nil {
			slog.Warn("scanning match", "err", err)
			continue
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
		       match_length, match_date, import_date, file_path, game_count,
		       COALESCE(last_visited_position, -1) as last_visited_position
		FROM match
		WHERE id = ?
	`, matchID).Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
		&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount, &m.LastVisitedPosition)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("match not found")
		}
		return nil, err
	}

	return &m, nil
}

// SaveLastVisitedPosition saves the last visited position index for a match
func (d *Database) SaveLastVisitedPosition(matchID int64, positionIndex int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`UPDATE match SET last_visited_position = ? WHERE id = ?`, positionIndex, matchID)
	if err != nil {
		return err
	}
	return nil
}

// GetLastVisitedMatch returns the most recently visited match (match with highest last_visited_position != -1)
// If no match has been visited, returns the most recent match (first in date order)
func (d *Database) GetLastVisitedMatch() (*Match, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var m Match
	// First try to find a match that has been visited (last_visited_position >= 0)
	err := d.db.QueryRow(`
		SELECT m.id, m.player1_name, m.player2_name, m.event, m.location, m.round,
		       m.match_length, m.match_date, m.import_date, m.file_path, m.game_count,
		       m.tournament_id, COALESCE(t.name, '') as tournament_name,
		       COALESCE(m.last_visited_position, -1) as last_visited_position
		FROM match m
		LEFT JOIN tournament t ON m.tournament_id = t.id
		WHERE m.last_visited_position >= 0
		ORDER BY m.import_date DESC
		LIMIT 1
	`).Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
		&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount,
		&m.TournamentID, &m.TournamentName, &m.LastVisitedPosition)

	if err == nil {
		return &m, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// No visited match found, return the most recent match
	err = d.db.QueryRow(`
		SELECT m.id, m.player1_name, m.player2_name, m.event, m.location, m.round,
		       m.match_length, m.match_date, m.import_date, m.file_path, m.game_count,
		       m.tournament_id, COALESCE(t.name, '') as tournament_name,
		       COALESCE(m.last_visited_position, -1) as last_visited_position
		FROM match m
		LEFT JOIN tournament t ON m.tournament_id = t.id
		ORDER BY CASE WHEN m.match_date IS NULL OR m.match_date = '' OR m.match_date = '0001-01-01T00:00:00Z' THEN m.import_date ELSE m.match_date END DESC
		LIMIT 1
	`).Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
		&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount,
		&m.TournamentID, &m.TournamentName, &m.LastVisitedPosition)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no matches in database")
		}
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
			slog.Warn("scanning game", "err", err)
			continue
		}
		g.InitialScore = [2]int32{score1, score2}
		games = append(games, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
			slog.Warn("scanning move", "err", err)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return moves, nil
}

// DeleteMatch deletes a match and all associated games, moves, and analysis
func (d *Database) DeleteMatch(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Collect position IDs referenced by this match's moves before cascade delete
	rows, err := tx.Query(`
		SELECT DISTINCT m.position_id 
		FROM move m
		INNER JOIN game g ON m.game_id = g.id
		WHERE g.match_id = ? AND m.position_id IS NOT NULL
	`, matchID)
	if err != nil {
		return fmt.Errorf("error collecting position IDs: %w", err)
	}
	var positionIDs []int64
	for rows.Next() {
		var pid int64
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return fmt.Errorf("error scanning position ID: %w", err)
		}
		positionIDs = append(positionIDs, pid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating position IDs: %w", err)
	}

	// Foreign key constraints will cascade delete to game, move, and move_analysis
	_, err = tx.Exec(`DELETE FROM match WHERE id = ?`, matchID)
	if err != nil {
		return fmt.Errorf("error deleting match: %w", err)
	}

	// Delete orphaned positions that are no longer referenced by any move
	// and not part of any collection
	for _, pid := range positionIDs {
		var refCount int
		err := tx.QueryRow(`
			SELECT COUNT(*) FROM (
				SELECT position_id FROM move WHERE position_id = ?
				UNION ALL
				SELECT position_id FROM collection_position WHERE position_id = ?
			)
		`, pid, pid).Scan(&refCount)
		if err != nil {
			return fmt.Errorf("error checking position references for ID %d: %w", pid, err)
		}
		if refCount == 0 {
			// Position is orphaned — delete it (cascades to analysis and comment)
			_, err = tx.Exec(`DELETE FROM position WHERE id = ?`, pid)
			if err != nil {
				return fmt.Errorf("error deleting orphaned position %d: %w", pid, err)
			}
		}
	}

	return tx.Commit()
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
			p.state as position_state,
			p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
			p.cube_value, p.cube_owner, p.score_1, p.score_2,
			p.has_jacoby, p.has_beaver,
			COALESCE(m.checker_move, '') as checker_move,
			COALESCE(m.cube_action, '') as cube_action
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
		var moveType, positionState, checkerMove, cubeAction string
		var pDT, pPOR, pD1, pD2, pCV, pCO, pS1, pS2, pHJ, pHB sql.NullInt64

		err := rows.Scan(&moveID, &gameID, &gameNumber, &moveNumber, &moveType, &player, &positionID, &positionState,
			&pDT, &pPOR, &pD1, &pD2, &pCV, &pCO, &pS1, &pS2, &pHJ, &pHB,
			&checkerMove, &cubeAction)
		if err != nil {
			slog.Warn("scanning move", "err", err)
			continue
		}

		// Reconstruct position from compact state + denormalized columns
		position := reconstructPosition(positionID, positionState,
			int(pDT.Int64), int(pPOR.Int64), int(pD1.Int64), int(pD2.Int64),
			int(pCV.Int64), int(pCO.Int64), int(pS1.Int64), int(pS2.Int64),
			int(pHJ.Int64), int(pHB.Int64))

		// Convert player from XG encoding (-1, 1) to blunderDB encoding (0, 1)
		playerBlunderDB := convertXGPlayerToBlunderDB(player)

		movePos := MatchMovePosition{
			Position:     position,
			MoveID:       moveID,
			GameID:       gameID,
			GameNumber:   gameNumber,
			MoveNumber:   moveNumber,
			MoveType:     moveType,
			PlayerOnRoll: int32(playerBlunderDB), // Now 0 or 1
			Player1Name:  player1Name,
			Player2Name:  player2Name,
			CheckerMove:  checkerMove,
			CubeAction:   cubeAction,
		}

		movePositions = append(movePositions, movePos)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

// UpdateMatch updates editable metadata for a match (player names and date).
// matchDate should be an empty string or a date string parseable by time.Parse ("2006-01-02").
func (d *Database) UpdateMatch(matchID int64, player1Name, player2Name, matchDate string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	var dateVal interface{}
	if matchDate != "" {
		t, err := time.Parse("2006-01-02", matchDate)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
		dateVal = t
	} else {
		dateVal = nil
	}

	_, err := d.db.Exec(
		`UPDATE match SET player1_name = ?, player2_name = ?, match_date = ? WHERE id = ?`,
		strings.TrimSpace(player1Name),
		strings.TrimSpace(player2Name),
		dateVal,
		matchID,
	)
	return err
}

// SwapMatchPlayers swaps the two players in a match: player1 becomes player2 and vice versa.
// This updates player names, game scores, game winners, and move player assignments.
func (d *Database) SwapMatchPlayers(matchID int64) error {
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

	// 1. Swap player1_name and player2_name in the match table
	_, err = tx.Exec(`
		UPDATE match 
		SET player1_name = player2_name, player2_name = player1_name
		WHERE id = ?
	`, matchID)
	if err != nil {
		return fmt.Errorf("failed to swap player names: %w", err)
	}

	// 2. Swap initial_score_1/initial_score_2 and flip winner in the game table
	_, err = tx.Exec(`
		UPDATE game
		SET initial_score_1 = initial_score_2,
		    initial_score_2 = initial_score_1,
		    winner = -winner
		WHERE match_id = ?
	`, matchID)
	if err != nil {
		return fmt.Errorf("failed to swap game scores/winner: %w", err)
	}

	// 3. Flip player in the move table (XG encoding: 1 → -1, -1 → 1)
	_, err = tx.Exec(`
		UPDATE move
		SET player = -player
		WHERE game_id IN (SELECT id FROM game WHERE match_id = ?)
	`, matchID)
	if err != nil {
		return fmt.Errorf("failed to swap move players: %w", err)
	}

	// 4. Update position denormalized columns to swap scores and cube owner.
	// The board (state) is unchanged; only the score_1/score_2 and cube_owner
	// columns need updating.
	_, err = tx.Exec(`
		UPDATE position SET
			score_1 = score_2, score_2 = score_1,
			cube_owner = CASE WHEN cube_owner = -1 THEN -1 WHEN cube_owner IS NULL THEN NULL ELSE 1 - cube_owner END
		WHERE id IN (
			SELECT DISTINCT m.position_id FROM move m
			INNER JOIN game g ON m.game_id = g.id
			WHERE g.match_id = ?
		)
	`, matchID)
	if err != nil {
		return fmt.Errorf("failed to swap position scores/cube: %w", err)
	}

	return tx.Commit()
}
