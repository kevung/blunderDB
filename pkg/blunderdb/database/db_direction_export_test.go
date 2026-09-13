package database

import (
	"path/filepath"
	"strings"
	"testing"
)

// A Direction travels with its tournament (issue #396).
//
// What these tests hold: an exported database, reopened elsewhere, finds its directed
// tournaments in their exact state; an export of chosen tournaments carries only THEIR
// directions; and the addition is by allow-list, so a column nobody listed does not travel.

// exportTo writes the database to a new file and opens it as a second Database, the way a
// colleague would.
func exportDirectionTo(t *testing.T, d *Database, opts ExportOptions) *Database {
	t.Helper()
	if opts.ExportPath == "" {
		opts.ExportPath = filepath.Join(t.TempDir(), "export.db")
	}
	if err := d.ExportDatabase(opts); err != nil {
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

// The allow-list is explicit: every column named here is a column someone decided to send.
// A column added to `direction` and not added here does not travel — which is the point.
func TestDirectionExport_TheAllowListIsExplicit(t *testing.T) {
	d := newTestDB(t)
	columns := map[string]bool{}
	rows, err := d.db.Query(`SELECT name FROM pragma_table_info('direction')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, c := range exportedDirectionColumns {
		if !columns[c] {
			t.Errorf("the allow-list names %q, which the table does not have", c)
		}
	}
	// output_dir exists and is deliberately NOT in the list: a path on the producer's machine
	// has no meaning on the recipient's.
	if !columns["output_dir"] {
		t.Error("output_dir is expected to exist, so that leaving it out is a decision")
	}
	for _, c := range exportedDirectionColumns {
		if c == "output_dir" {
			t.Error("output_dir must not travel")
		}
	}
	// And the columns the table has that nobody listed: they are named here so that adding a
	// column is a decision taken in this file rather than a silence.
	var unlisted []string
	listed := map[string]bool{}
	for _, c := range exportedDirectionColumns {
		listed[c] = true
	}
	for c := range columns {
		if !listed[c] {
			unlisted = append(unlisted, c)
		}
	}
	if strings.Join(unlisted, ",") != "output_dir" {
		t.Errorf("columns outside the allow-list: %v — add them to it, or to this test's expectation, on purpose", unlisted)
	}

	// The journal's own allow-list, held the same way: an event carries these five columns and
	// no others, and the list is the schema's.
	eventColumns := map[string]bool{}
	erows, err := d.db.Query(`SELECT name FROM pragma_table_info('direction_event')`)
	if err != nil {
		t.Fatal(err)
	}
	defer erows.Close()
	for erows.Next() {
		var name string
		if err := erows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		eventColumns[name] = true
	}
	if err := erows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(eventColumns) != len(exportedEventColumns) {
		t.Errorf("direction_event has %d columns and the allow-list names %d", len(eventColumns), len(exportedEventColumns))
	}
	for _, c := range exportedEventColumns {
		if !eventColumns[c] {
			t.Errorf("the journal's allow-list names %q, which the table does not have", c)
		}
	}
}
