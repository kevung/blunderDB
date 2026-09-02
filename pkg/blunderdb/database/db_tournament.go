package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ========== Tournament Functions ==========
//
// Every tournament method below is an adapter over the Storage backend
// (d.store.Tournaments(), see storage.TournamentStore): it takes d.mu the way
// the GUI and CLI expect, then delegates with the wrapper's implicit scope.
// The SQL itself lives in storage/sqlite/tournaments_sqlite.go, held to the
// shared contract suite alongside the PostgreSQL backend. ExportTournaments,
// at the end of this file, is the one exception: it writes a match graph
// (games, moves, move analyses) and the positions those matches reached to a
// separate export file, via ingest.ExportSQLite — the same unified exporter
// ExportDatabase and ExportCollections run.

// CreateTournament creates a new tournament
func (d *Database) CreateTournament(name string, date string, location string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().Create(context.Background(), "", name, date, location)
}

// GetAllTournaments returns all tournaments with their match counts
func (d *Database) GetAllTournaments() ([]Tournament, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var tournaments []Tournament
	for t, err := range d.store.Tournaments().List(context.Background(), "") {
		if err != nil {
			return nil, err
		}
		tournaments = append(tournaments, *t)
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
	return d.store.Tournaments().Update(context.Background(), "", id, name, date, location)
}

// DeleteTournament deletes a tournament (matches are unlinked, not deleted)
func (d *Database) DeleteTournament(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().Delete(context.Background(), "", id)
}

// AddMatchToTournament adds a match to a tournament
func (d *Database) AddMatchToTournament(tournamentID int64, matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().AddMatch(context.Background(), "", tournamentID, matchID)
}

// RemoveMatchFromTournament removes a match from a tournament
func (d *Database) RemoveMatchFromTournament(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().RemoveMatch(context.Background(), "", matchID)
}

// UpdateMatchComment updates the comment of a match
func (d *Database) UpdateMatchComment(matchID int64, comment string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Matches().UpdateComment(context.Background(), "", matchID, comment)
}

// UpdateTournamentComment updates the comment of a tournament
func (d *Database) UpdateTournamentComment(tournamentID int64, comment string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().UpdateComment(context.Background(), "", tournamentID, comment)
}

// ReorderTournamentMatches sets the sort order for matches in a tournament.
// matchIDs should be in the desired order.
func (d *Database) ReorderTournamentMatches(tournamentID int64, matchIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().ReorderMatches(context.Background(), "", tournamentID, matchIDs)
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
	return d.store.Tournaments().SetMatchByName(context.Background(), "", matchID, tournamentName)
}

// GetTournamentMatches returns all matches in a tournament
func (d *Database) GetTournamentMatches(tournamentID int64) ([]Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var matches []Match
	for m, err := range d.store.Tournaments().Matches(context.Background(), "", tournamentID) {
		if err != nil {
			return nil, err
		}
		matches = append(matches, *m)
	}

	if err := d.applyMatchBadges(matches); err != nil {
		// Non-fatal: log but continue without stats
		_ = err
	}

	return matches, nil
}

// GetMatchTournament returns the tournament a match belongs to (if any).
// A match that exists but is not in any tournament yields (nil, nil), as does
// an unknown match id: the backend reports both as storage.ErrNotFound and the
// GUI treats "no tournament" as a plain absence, not an error.
func (d *Database) GetMatchTournament(matchID int64) (*Tournament, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	t, err := d.store.Tournaments().TournamentOf(context.Background(), "", matchID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	return t, err
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
