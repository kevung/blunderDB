package database

import (
	"math"
	"path/filepath"
	"testing"
)

// TestSwapMatchPlayersDelegates checks the legacy Desktop path swaps a match's
// players by delegating to the storage layer. Importing a real .mat, swapping, and reading back must show
// the names exchanged with no error.
func TestSwapMatchPlayersDelegates(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	matchID, err := db.ImportGnuBGMatch("testdata/test.mat")
	if err != nil {
		t.Fatalf("import test.mat: %v", err)
	}

	before := matchByID(t, db, matchID)
	p1, p2 := before.Player1Name, before.Player2Name
	if p1 == "" || p2 == "" || p1 == p2 {
		t.Fatalf("fixture needs two distinct named players, got %q / %q", p1, p2)
	}

	if err := db.SwapMatchPlayers(matchID); err != nil {
		t.Fatalf("SwapMatchPlayers: %v", err)
	}

	after := matchByID(t, db, matchID)
	if after.Player1Name != p2 || after.Player2Name != p1 {
		t.Errorf("names not swapped: got %q / %q, want %q / %q",
			after.Player1Name, after.Player2Name, p2, p1)
	}
}

func matchByID(t *testing.T, db *Database, id int64) Match {
	t.Helper()
	matches, err := db.GetAllMatches()
	if err != nil {
		t.Fatalf("GetAllMatches: %v", err)
	}
	for _, m := range matches {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("match %d not found", id)
	return Match{}
}

// TestSwapMatchPlayers_RealMatchKeepsAnalyses holds the seat swap on an XG
// match analysed at a match score: every analysis stays, each person keeps
// their PR, MWC loss and decision count under the other seat, the row the
// list re-reads (GetMatchByID) carries the badges as they now are, and a
// second swap gives the match back.
func TestSwapMatchPlayers_RealMatchKeepsAnalyses(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "swap.db")); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.ImportXGMatch(filepath.Join("testdata", "HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg")); err != nil {
		t.Fatal(err)
	}
	matches, err := d.GetAllMatches()
	if err != nil || len(matches) != 1 {
		t.Fatalf("%d matches, %v", len(matches), err)
	}
	id := matches[0].ID

	counts := func() (analyses, positions int) {
		t.Helper()
		if err := d.db.QueryRow(`SELECT (SELECT COUNT(*) FROM analysis), (SELECT COUNT(*) FROM position)`).
			Scan(&analyses, &positions); err != nil {
			t.Fatal(err)
		}
		return
	}
	type seat struct {
		name      string
		pr, mwc   float64
		l7        float64
		decisions int
	}
	read := func() [2]seat {
		t.Helper()
		m, err := d.GetMatchByID(id)
		if err != nil {
			t.Fatal(err)
		}
		detail, err := d.GetMatchDetailStats(id)
		if err != nil {
			t.Fatal(err)
		}
		return [2]seat{
			{m.Player1Name, m.PR, m.MWCLoss, m.MWC7.Loss, detail.Player1.TotalDecisions},
			{m.Player2Name, m.PR2, m.MWCLoss2, m.MWC7P2.Loss, detail.Player2.TotalDecisions},
		}
	}
	near := func(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(math.Abs(a), math.Abs(b)) }

	analyses, positions := counts()
	before := read()
	if before[0].decisions == 0 || before[1].decisions == 0 || before[0].pr == 0 {
		t.Fatalf("nothing to compare: %+v", before)
	}

	if err := d.SwapMatchPlayers(id); err != nil {
		t.Fatal(err)
	}
	if a, p := counts(); a != analyses || p != positions {
		t.Errorf("after the swap: %d analyses, %d positions; was %d, %d", a, p, analyses, positions)
	}
	after := read()
	for i := range 2 {
		got, want := after[i], before[1-i]
		if got.name != want.name || got.decisions != want.decisions ||
			!near(got.pr, want.pr) || !near(got.mwc, want.mwc) || !near(got.l7, want.l7) {
			t.Errorf("seat %d after the swap: %+v, want seat %d before: %+v", i+1, got, 2-i, want)
		}
	}

	if err := d.SwapMatchPlayers(id); err != nil {
		t.Fatal(err)
	}
	if again := read(); again != before {
		t.Errorf("two swaps: %+v, want %+v", again, before)
	}
}
