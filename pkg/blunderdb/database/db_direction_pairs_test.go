package database

import (
	"context"
	"strings"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

func doublesDirection(t *testing.T, d *Database) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Doubles", "2026-10-04", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := tournoi.Config{Name: "Doubles", Tables: tournoi.Tables{Count: 8},
		Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindBracket, Length: 5}}}
	if _, err := direction.Create(context.Background(), d.DirectionStore(), tID, cfg, 3, time.Now()); err != nil {
		t.Fatal(err)
	}
	return tID
}

// S3's doubles replayed with real pairs (ADR-0056 §4): each pair is one Participant "A / B"
// whose rating is the mean of the two, correctable; its Matches carry "A / B"; the Directory
// lists the persons and never a pair.
func TestDoublesPairsAreTwoPersons(t *testing.T) {
	d := newTestDB(t)
	tID := doublesDirection(t, d)
	var last *DirectionView
	for i := 0; i < 16; i++ {
		a := PairMember{Name: "Ana " + string(rune('A'+i)), Club: "BC Ourcq", Rating: 4}
		b := PairMember{Name: "Bea " + string(rune('A'+i)), Club: "Cercle du Rhône", Rating: 6}
		blob := `[{"name":"` + a.Name + `","club":"` + a.Club + `","rating":4},{"name":"` + b.Name + `","club":"` + b.Club + `","rating":6}]`
		v, err := d.AddPair(tID, blob, 0)
		if err != nil {
			t.Fatalf("AddPair %d: %v", i, err)
		}
		last = v
	}
	if len(last.Players) != 16 || len(last.Pairs) != 16 {
		t.Fatalf("%d participants, %d pairs; want 16 of each", len(last.Players), len(last.Pairs))
	}
	first := last.Players[0]
	if first.Name != "Ana A / Bea A" || first.Rating != 5 || first.Club != "BC Ourcq / Cercle du Rhône" {
		t.Errorf("first pair = %+v, want the derived label, the mean rating and both clubs", first)
	}

	// The director corrects the rating; the members stay, the identifier too.
	v, err := d.UpdatePair(tID, string(first.ID), `[{"name":"Ana A","club":"BC Ourcq","rating":4},{"name":"Bea A","club":"BC Ourcq","rating":6}]`, 4.5)
	if err != nil {
		t.Fatal(err)
	}
	if p := v.Players[0]; p.ID != first.ID || p.Rating != 4.5 || p.Club != "BC Ourcq" {
		t.Errorf("corrected pair = %+v", p)
	}

	// Launched, a Match carries "A / B" on each side.
	// The first confirmation draws the bracket, the second launches its matches.
	for range 2 {
		if v, err = d.ConfirmAllProposals(tID); err != nil {
			t.Fatal(err)
		}
	}
	if len(v.Running) == 0 {
		t.Fatal("no match launched")
	}
	cells, err := d.TableGrid(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cells {
		if c.MatchID != "" && (!strings.Contains(c.AName, " / ") || !strings.Contains(c.BName, " / ")) {
			t.Errorf("table %d: %q – %q, want a pair on each side", c.Table, c.AName, c.BName)
		}
	}

	// The Directory holds 32 persons and no "A / B" line.
	dir, err := d.Directory()
	if err != nil {
		t.Fatal(err)
	}
	if len(dir) != 32 {
		t.Errorf("directory: %d lines, want the 32 persons", len(dir))
	}
	for _, e := range dir {
		if strings.Contains(e.Name, " / ") {
			t.Errorf("directory line %q is a pair", e.Name)
		}
	}
}

func TestPairNeedsTwoNamedPersons(t *testing.T) {
	d := newTestDB(t)
	tID := doublesDirection(t, d)
	for _, blob := range []string{`[{"name":"Solo"}]`, `[{"name":"A"},{"name":" "}]`} {
		if _, err := d.AddPair(tID, blob, 0); err == nil {
			t.Errorf("AddPair(%s) accepted", blob)
		}
	}
	if v, _ := d.GetDirection(tID); len(v.Players) != 0 {
		t.Errorf("a refused pair left %d participant(s)", len(v.Players))
	}
}
