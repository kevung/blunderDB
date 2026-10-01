package database

import "testing"

// The search index feeds the director's palette: every event, player, table and running match of
// the room in one call, whichever event they belong to.
func TestRencontreSearchIndex_RoomWide(t *testing.T) {
	d := newTestDB(t)
	a := startedDirection(t, d, 16)
	b := startedDirectionNamed(t, d, 16, "Joueuse ")
	r, err := d.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 14)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{a, b} {
		if _, err := d.AttachToRencontre(id, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	ma := runningMatch(t, d, a)
	mb := runningMatch(t, d, b)

	idx, err := d.RencontreSearchIndex(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	var playingBeforeFree = true
	seenFree := false
	for _, e := range idx {
		count[e.Kind]++
		if e.Kind == "player" {
			if e.State != "playing" {
				seenFree = true
			} else if seenFree {
				playingBeforeFree = false
			}
		}
	}
	if count["epreuve"] != 2 || count["player"] != 32 || count["table"] != 14 {
		t.Errorf("index counts = %v, want 2 events, 32 players, 14 tables", count)
	}
	if count["match"] < 2 {
		t.Errorf("index lists %d running matches, want at least the two started", count["match"])
	}
	if !playingBeforeFree {
		t.Error("players at a table must come before the free ones")
	}
	for _, m := range []struct {
		tid int64
		a   string
		tab int
	}{{a, "Joueur " + string(ma.A), ma.Table}, {b, "Joueuse " + string(mb.A), mb.Table}} {
		found := false
		for _, e := range idx {
			if e.Kind == "player" && e.TournamentID == m.tid && e.Name == m.a {
				found = e.State == "playing" && e.Table == m.tab && e.Opponent != ""
			}
		}
		if !found {
			t.Errorf("player %q of event %d is not indexed as playing at table %d with an opponent", m.a, m.tid, m.tab)
		}
	}
	for _, e := range idx {
		if e.Kind == "table" && e.Table == ma.Table && (e.TournamentID != a || e.State != "running" || e.Name == "") {
			t.Errorf("table %d = %+v, want occupied by event %d", ma.Table, e, a)
		}
	}
}

// A Direction outside any Rencontre is searched alone, over its own tables.
func TestDirectionSearchIndex_LoneEvent(t *testing.T) {
	d := newTestDB(t)
	a := startedDirection(t, d, 8)
	idx, err := d.DirectionSearchIndex(a)
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	for _, e := range idx {
		count[e.Kind]++
	}
	if count["epreuve"] != 1 || count["player"] != 8 || count["table"] == 0 {
		t.Errorf("index counts = %v, want 1 event, 8 players and its tables", count)
	}
}
