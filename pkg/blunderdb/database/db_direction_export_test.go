package database

import (
	"path/filepath"
	"testing"
)

// A Direction travels with its tournament.
//
// What these tests hold: an exported database, reopened elsewhere, finds its directed
// tournaments in their exact state; an export of chosen tournaments carries only THEIR
// directions.

// exportTo writes the database to a new file and opens it as a second Database, the way a
// colleague would.
func exportDirectionTo(t *testing.T, d *Database, opts ExportOptions) *Database {
	t.Helper()
	if opts.ExportPath == "" {
		opts.ExportPath = filepath.Join(t.TempDir(), "export.db")
	}
	if _, err := d.ExportDatabase(opts); err != nil {
		t.Fatalf("export: %v", err)
	}
	other := newTestDB(t)
	if err := other.OpenDatabase(opts.ExportPath); err != nil {
		t.Fatalf("reopening the export: %v", err)
	}
	return other
}

// launchARound launches whatever is launchable; the draw comes first in a Swiss, so "launch
// all" may have to be asked twice.
func launchARound(t *testing.T, d *Database, tID int64) {
	t.Helper()
	for i := 0; i < 4; i++ {
		v, err := d.ConfirmAllProposals(tID)
		if err != nil {
			t.Fatal(err)
		}
		if len(v.Running) > 0 {
			return
		}
	}
	t.Fatal("no match running after four attempts")
}

func TestDirectionExport_TravelsWithItsTournament(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 8)
	launchARound(t, d, tID)
	before, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}

	// Exporting tournaments is an explicit choice of the producer's — the modal has a toggle
	// for it, off by default — and a Direction travels with the tournament that was chosen.
	other := exportDirectionTo(t, d, ExportOptions{
		AllPositions: true, IncludeMatches: true, TournamentIDs: []int64{tID},
	})
	rows, err := other.ListDirections()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("%d directions in the export, expected 1", len(rows))
	}
	after, err := other.GetDirection(rows[0].TournamentID)
	if err != nil {
		t.Fatal(err)
	}
	// The exact state: the same journal replays to the same tournament.
	if after.EventCount != before.EventCount {
		t.Errorf("%d events exported, %d in the original", after.EventCount, before.EventCount)
	}
	if len(after.Players) != len(before.Players) || len(after.Running) != len(before.Running) {
		t.Errorf("the replayed state differs: %d players / %d running, was %d / %d",
			len(after.Players), len(after.Running), len(before.Players), len(before.Running))
	}
	if after.State != before.State || after.Config.Name != before.Config.Name {
		t.Errorf("state %q / name %q, was %q / %q", after.State, after.Config.Name, before.State, before.Config.Name)
	}

	// The display folder does NOT travel: it is a path on the producer's machine.
	if after.OutputDir != "" {
		t.Errorf("the output folder followed the export: %q", after.OutputDir)
	}
}

// An export of chosen tournaments carries only THEIR directions.
func TestDirectionExport_OnlyTheChosenTournaments(t *testing.T) {
	d := newTestDB(t)
	kept := startedDirection(t, d, 8)
	launchARound(t, d, kept)

	left, err := d.CreateTournament("Open de Paris", "2026-10-10", "Paris")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Paris","tables":{"count":8},"phases":[{"kind":"swiss_lives","length":7,"lives":2}]}`
	if err := d.CreateDirection(left, cfg, 7); err != nil {
		t.Fatal(err)
	}

	other := exportDirectionTo(t, d, ExportOptions{TournamentIDs: []int64{kept}, IncludeMatches: true})
	rows, err := other.ListDirections()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("%d directions exported, expected only the chosen one: %+v", len(rows), rows)
	}
	v, err := other.GetDirection(rows[0].TournamentID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Config.Name != "Open de Lyon" {
		t.Errorf("the wrong tournament's direction travelled: %q", v.Config.Name)
	}

	// And a tournament the producer did NOT choose leaves its direction behind, with nothing
	// of it in the file at all.
	names, err := other.GetAllTournaments()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if n.Name == "Open de Paris" {
			t.Error("a tournament that was not chosen travelled")
		}
	}
}

// A database with no direction exports without one, and without an error: the hook must be a
// no-op on the ordinary case, which is every export anyone has ever run.
func TestDirectionExport_NothingToCarryIsNotAnError(t *testing.T) {
	d := newTestDB(t)
	if _, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon"); err != nil {
		t.Fatal(err)
	}
	other := exportDirectionTo(t, d, ExportOptions{AllPositions: true, IncludeMatches: true})
	rows, err := other.ListDirections()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("%d directions in an export that had none", len(rows))
	}
}
