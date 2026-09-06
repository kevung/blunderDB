package cli

import (
	"encoding/csv"
	"strings"
	"testing"
)

// The CSV columns are a CONTRACT (#280, fiche I.24): a notebook or a script
// written against these names must keep working. These tests pin the header of
// each export exactly, so a rename or a reorder fails here rather than in
// somebody's notebook six months later.
//
// A column ADDED at the end is a deliberate change and updates the list below;
// a column renamed or moved should have to argue for itself in a diff.

func TestExportPositionsCSV(t *testing.T) {
	cli := setupCLI(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	out := captureStdout(t, func() {
		if err := cli.exportPositionsCSV(0); err != nil {
			t.Fatalf("exportPositionsCSV: %v", err)
		}
	})
	rows := parseCSV(t, out)
	assertHeader(t, rows, positionColumns)
	if len(rows) < 2 {
		t.Fatal("the export carries no position")
	}
	// The XGID column is the one a reader pastes back into blunderDB or into
	// XG: an export whose identifiers are empty is a table nobody can act on.
	if !strings.Contains(rows[1][1], ":") {
		t.Errorf("position row carries no XGID: %q", rows[1][1])
	}
}

func TestExportMovesCSV(t *testing.T) {
	cli := setupCLI(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	out := captureStdout(t, func() {
		if err := cli.exportMovesCSV(0); err != nil {
			t.Fatalf("exportMovesCSV: %v", err)
		}
	})
	rows := parseCSV(t, out)
	assertHeader(t, rows, moveColumns)
	if len(rows) < 2 {
		t.Fatal("the export carries no move")
	}
	// The match is repeated on every row on purpose: a table that has to be
	// joined before it says anything is a table nobody uses.
	if rows[1][5] == "" && rows[1][6] == "" {
		t.Error("a move row names neither player")
	}
}

func TestExportAnalysesCSV(t *testing.T) {
	cli := setupCLI(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	out := captureStdout(t, func() {
		if err := cli.exportAnalysesCSV(0); err != nil {
			t.Fatalf("exportAnalysesCSV: %v", err)
		}
	})
	rows := parseCSV(t, out)
	assertHeader(t, rows, analysisColumns)
	if len(rows) < 2 {
		t.Fatal("the export carries no analysis though the match came with one")
	}
}

// A limit the caller passes is honoured; the export is otherwise whole. The
// "otherwise whole" half is what `list`'s own default of 10 would have broken
// silently — see exportLimit.
func TestExportRespectsAnExplicitLimit(t *testing.T) {
	cli := setupCLI(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	whole := parseCSV(t, captureStdout(t, func() {
		if err := cli.exportPositionsCSV(0); err != nil {
			t.Fatalf("exportPositionsCSV: %v", err)
		}
	}))
	short := parseCSV(t, captureStdout(t, func() {
		if err := cli.exportPositionsCSV(5); err != nil {
			t.Fatalf("exportPositionsCSV(5): %v", err)
		}
	}))
	if len(short) != 6 { // header + five rows
		t.Fatalf("limit 5 gave %d line(s) including the header", len(short))
	}
	if len(whole) <= len(short) {
		t.Fatalf("the unlimited export (%d lines) is no larger than the limited one (%d)", len(whole), len(short))
	}
}

func parseCSV(t *testing.T, out string) [][]string {
	t.Helper()
	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("the export is not valid CSV: %v", err)
	}
	return rows
}

func assertHeader(t *testing.T, rows [][]string, want []string) {
	t.Helper()
	if len(rows) == 0 {
		t.Fatal("the export has no header")
	}
	if strings.Join(rows[0], ",") != strings.Join(want, ",") {
		t.Errorf("header drifted:\n got %v\nwant %v", rows[0], want)
	}
}
