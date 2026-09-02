package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ========== Tournament Functions ==========

// CreateTournament creates a new tournament
func (d *Database) CreateTournament(name string, date string, location string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}

	// Get the max sort_order
	var maxOrder int
	err := d.db.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM tournament`).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	result, err := d.db.Exec(`
		INSERT INTO tournament (name, date, location, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`, name, date, location, maxOrder+1)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetAllTournaments returns all tournaments with their match counts
func (d *Database) GetAllTournaments() ([]Tournament, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			t.id,
			t.name,
			COALESCE(t.date, ''),
			COALESCE(t.location, ''),
			t.sort_order,
			t.created_at,
			t.updated_at,
			COUNT(m.id) as match_count,
			COALESCE(t.comment, '')
		FROM tournament t
		LEFT JOIN match m ON t.id = m.tournament_id
		GROUP BY t.id
		ORDER BY t.date DESC, t.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tournaments []Tournament
	for rows.Next() {
		var t Tournament
		err := rows.Scan(&t.ID, &t.Name, &t.Date, &t.Location, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt, &t.MatchCount, &t.Comment)
		if err != nil {
			slog.Warn("scanning tournament", "err", err)
			continue
		}
		tournaments = append(tournaments, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := d.applyTournamentBadges(tournaments); err != nil {
		// Non-fatal
		_ = err
	}

	return tournaments, nil
}

// UpdateTournament updates a tournament's details
func (d *Database) UpdateTournament(id int64, name string, date string, location string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`
		UPDATE tournament SET name = ?, date = ?, location = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, date, location, id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteTournament deletes a tournament (matches are unlinked, not deleted)
func (d *Database) DeleteTournament(id int64) error {
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

	// Unlink matches from this tournament
	_, err = tx.Exec(`UPDATE match SET tournament_id = NULL WHERE tournament_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete the tournament
	_, err = tx.Exec(`DELETE FROM tournament WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AddMatchToTournament adds a match to a tournament
func (d *Database) AddMatchToTournament(tournamentID int64, matchID int64) error {
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

	// Get the max sort order for this tournament
	var maxOrder int
	err = tx.QueryRow(`SELECT COALESCE(MAX(tournament_sort_order), -1) FROM match WHERE tournament_id = ?`, tournamentID).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	_, err = tx.Exec(`UPDATE match SET tournament_id = ?, tournament_sort_order = ? WHERE id = ?`, tournamentID, maxOrder+1, matchID)
	if err != nil {
		return err
	}

	// Update tournament's updated_at
	_, err = tx.Exec(`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveMatchFromTournament removes a match from a tournament
func (d *Database) RemoveMatchFromTournament(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`UPDATE match SET tournament_id = NULL, tournament_sort_order = 0 WHERE id = ?`, matchID)
	return err
}

// UpdateMatchComment updates the comment of a match
func (d *Database) UpdateMatchComment(matchID int64, comment string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`UPDATE match SET comment = ? WHERE id = ?`, comment, matchID)
	return err
}

// UpdateTournamentComment updates the comment of a tournament
func (d *Database) UpdateTournamentComment(tournamentID int64, comment string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`UPDATE tournament SET comment = ?, updated_at = datetime('now') WHERE id = ?`, comment, tournamentID)
	return err
}

// ReorderTournamentMatches sets the sort order for matches in a tournament.
// matchIDs should be in the desired order.
func (d *Database) ReorderTournamentMatches(tournamentID int64, matchIDs []int64) error {
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

	for i, matchID := range matchIDs {
		_, err := tx.Exec(`UPDATE match SET tournament_sort_order = ? WHERE id = ? AND tournament_id = ?`, i, matchID, tournamentID)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SetMatchTournamentByName assigns a match to a tournament by name.
// If tournamentName is empty, the match is unlinked from any tournament.
// If no tournament with that name exists, one is created.
func (d *Database) SetMatchTournamentByName(matchID int64, tournamentName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	name := strings.TrimSpace(tournamentName)
	if name == "" {
		_, err := d.db.Exec(`UPDATE match SET tournament_id = NULL WHERE id = ?`, matchID)
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Look for existing tournament with that name
	var tournamentID int64
	err = tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, name).Scan(&tournamentID)
	if err != nil {
		// Create new tournament
		res, err2 := tx.Exec(`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, name)
		if err2 != nil {
			return err2
		}
		tournamentID, err2 = res.LastInsertId()
		if err2 != nil {
			return err2
		}
	}

	_, err = tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// GetTournamentMatches returns all matches in a tournament
func (d *Database) GetTournamentMatches(tournamentID int64) ([]Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			id, player1_name, player2_name, event, location, round, 
			match_length, match_date, import_date, file_path, game_count, tournament_id,
			COALESCE(last_visited_position, -1) as last_visited_position,
			COALESCE(comment, '') as comment,
			COALESCE(tournament_sort_order, 0) as tournament_sort_order
		FROM match 
		WHERE tournament_id = ?
		ORDER BY tournament_sort_order ASC, match_date DESC
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		var tournamentID sql.NullInt64
		err := rows.Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
			&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount, &tournamentID, &m.LastVisitedPosition,
			&m.Comment, &m.TournamentSortOrder)
		if err != nil {
			// tournamentID here would be the loop's own (still zero-value,
			// unscanned) sql.NullInt64, not the tournament this query is
			// scoped to (the function parameter of the same name) — the log
			// line only names the query, not that value, to avoid printing a
			// misleading id.
			slog.Warn("scanning match for GetTournamentMatches", "err", err)
			continue
		}
		if tournamentID.Valid {
			tid := tournamentID.Int64
			m.TournamentID = &tid
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := d.applyMatchBadges(matches); err != nil {
		// Non-fatal: log but continue without stats
		_ = err
	}

	return matches, nil
}

// GetMatchTournament returns the tournament a match belongs to (if any)
func (d *Database) GetMatchTournament(matchID int64) (*Tournament, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var tournamentID sql.NullInt64
	err := d.db.QueryRow(`SELECT tournament_id FROM match WHERE id = ?`, matchID).Scan(&tournamentID)
	if err != nil {
		return nil, err
	}

	if !tournamentID.Valid {
		return nil, nil // Match is not in any tournament
	}

	var t Tournament
	err = d.db.QueryRow(`
		SELECT id, name, COALESCE(date, ''), COALESCE(location, ''), sort_order, created_at, updated_at
		FROM tournament WHERE id = ?
	`, tournamentID.Int64).Scan(&t.ID, &t.Name, &t.Date, &t.Location, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// ExportTournaments exports specific tournaments — their matches, games,
// moves and move analyses — and the positions those matches reached, to a
// database file. watermark and watermarkNote mirror ExportDatabase's
// Watermark/WatermarkNote: an empty watermark means the export carries none.
// The work is ingest.ExportSQLite, the same exporter ExportDatabase and
// ExportCollections run.
func (d *Database) ExportTournaments(exportPath string, tournamentIDs []int64, metadata map[string]string, includeAnalysis bool, includeComments bool, watermark string, watermarkNote string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	watermarkDocument, err := sealWatermark(watermark, watermarkNote)
	if err != nil {
		return fmt.Errorf("cannot sign the watermark: %w", err)
	}

	_, err = ingest.ExportSQLite(context.Background(), d.store, "", exportPath, ingest.ExportOptions{
		Format: ingest.FormatSQLite,
		Selection: ingest.Selection{
			TournamentIDs:     tournamentIDs,
			TournamentMatches: true,
			MatchPositions:    true,
		},
		Analysis:  includeAnalysis,
		Comments:  includeComments,
		Metadata:  metadata,
		Watermark: watermarkDocument,
	})
	return err
}
