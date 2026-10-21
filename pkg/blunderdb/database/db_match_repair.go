package database

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchRepair is one line of the repair listing: a match whose stored moves
// differ from its source file, or whose source cannot be used (Error).
type MatchRepair struct {
	ingest.RepairReport
	Player1Name string `json:"player1_name"`
	Player2Name string `json:"player2_name"`
	Error       string `json:"error,omitempty"`
}

// RepairMatch rewrites a match's games, moves, positions and analyses from
// its source file, keeping everything the user brought to it
// (ingest.RepairMatch). With dryRun nothing is written and the report says
// what would change. CancelImport stops it.
func (d *Database) RepairMatch(matchID int64, dryRun bool) (ingest.RepairReport, error) {
	ctx, done := d.beginCancellableImport()
	defer done()
	return d.repairMatch(ctx, matchID, dryRun)
}

func (d *Database) repairMatch(ctx context.Context, matchID int64, dryRun bool) (ingest.RepairReport, error) {
	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return ingest.RepairReport{}, fmt.Errorf("no database is currently open")
	}
	m, err := d.store.Matches().Get(ctx, "", matchID)
	dir := d.libraryDir(ctx)
	d.mu.RUnlock()
	if err != nil {
		return ingest.RepairReport{MatchID: matchID}, err
	}

	// The file is read without the lock: the transaction checks that it
	// still holds the match.
	graph, err := ingest.MapMatchSource(m.FilePath, dir)
	if err != nil {
		return ingest.RepairReport{MatchID: matchID, FilePath: m.FilePath}, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return ingest.RepairReport{}, fmt.Errorf("no database is currently open")
	}
	return ingest.RepairMatch(ctx, d.store, "", matchID, graph, dryRun)
}

// FindMatchesToRepair dry-runs RepairMatch on every match that names a
// source file and returns those that differ from it, and those whose source
// is missing or no longer holds them. Nothing is written. CancelImport stops
// it.
func (d *Database) FindMatchesToRepair() ([]MatchRepair, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("no database is currently open")
	}
	var matches []MatchRepair
	for m, err := range d.store.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			d.mu.RUnlock()
			return nil, err
		}
		if m.FilePath == "" {
			continue
		}
		matches = append(matches, MatchRepair{
			RepairReport: ingest.RepairReport{MatchID: m.ID, FilePath: m.FilePath},
			Player1Name:  m.Player1Name, Player2Name: m.Player2Name,
		})
	}
	d.mu.RUnlock()

	var out []MatchRepair
	for _, mr := range matches {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		rep, err := d.repairMatch(ctx, mr.MatchID, true)
		if err != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			mr.Error = err.Error()
			out = append(out, mr)
			continue
		}
		if rep.Differs() {
			mr.RepairReport = rep
			out = append(out, mr)
		}
	}
	return out, nil
}

// libraryDir is the directory of the open library file, "" for an in-memory
// one or a library that is no local file: where a relative source path is
// looked for first. Callers hold d.mu.
func (d *Database) libraryDir(ctx context.Context) string {
	local, ok := d.store.(storage.LocalFile)
	if !ok {
		return ""
	}
	file, err := local.FilePath(ctx)
	if err != nil || file == "" {
		return ""
	}
	return filepath.Dir(file)
}
