package direction

import (
	"slices"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func twoRooms() []domain.TableSetting {
	return []domain.TableSetting{
		{Number: 1, Room: "A"}, {Number: 2, Room: "A", Name: "Stream", Reserved: true},
		{Number: 3, Room: "B", AssignedTo: []string{"Alice"}}, {Number: 4, Room: "B", AssignedTo: []string{"Bob"}},
	}
}

func TestTablePlanAllowedAndClosed(t *testing.T) {
	all := TablePlan{Count: 4, Settings: twoRooms()}
	for n := 1; n <= 4; n++ {
		if !all.Allowed(n) {
			t.Errorf("unrestricted event refused table %d", n)
		}
	}
	if got := all.Closed(); !slices.Equal(got, []int{2, 3, 4}) {
		t.Errorf("Closed() unrestricted = %v, want reserved and kept tables [2 3 4]", got)
	}
	b := TablePlan{Count: 4, Settings: twoRooms(), Rooms: []string{"B"}}
	if b.Allowed(1) || !b.Allowed(3) {
		t.Errorf("event in room B: Allowed(1)=%v Allowed(3)=%v", b.Allowed(1), b.Allowed(3))
	}
	if got := b.Closed(); !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Errorf("Closed() in room B = %v", got)
	}
	// A Rencontre where no table carries a room has a single room: a restriction is moot.
	plain := TablePlan{Count: 3, Settings: []domain.TableSetting{{Number: 2, Name: "Stream"}}, Rooms: []string{"B"}}
	if !plain.Allowed(1) || len(plain.Closed()) != 0 {
		t.Errorf("roomless plan: Allowed(1)=%v Closed=%v", plain.Allowed(1), plain.Closed())
	}
	if got := RoomNames(twoRooms()); !slices.Equal(got, []string{"A", "B"}) {
		t.Errorf("RoomNames = %v", got)
	}
}

func TestAssignKept(t *testing.T) {
	p := TablePlan{Count: 6, Settings: twoRooms()}
	names := func(id tournoi.PlayerID) []string {
		return map[tournoi.PlayerID][]string{"a": {"Alice"}, "b": {"Bob"}, "c": {"Carol"}, "d": {"Dan"}, "p": {"Eve", "Bob"}}[id]
	}
	start := func(a, b string, table int, reason tournoi.ReasonCode) tournoi.Action {
		return tournoi.Action{Kind: tournoi.ActStartMatch, A: tournoi.PlayerID(a), B: tournoi.PlayerID(b), Table: table, Reason: reason}
	}
	tables := tournoi.Tables{Count: 6}

	t.Run("a holder plays on their free table", func(t *testing.T) {
		got := AssignKept([]tournoi.Action{start("c", "a", 1, "")}, p, tables, nil, names)
		if got[0].Table != 3 {
			t.Errorf("Alice's match on table %d, want 3", got[0].Table)
		}
	})
	t.Run("two holders meet on the smaller table", func(t *testing.T) {
		got := AssignKept([]tournoi.Action{start("b", "a", 1, "")}, p, tables, nil, names)
		if got[0].Table != 3 {
			t.Errorf("Alice v Bob on table %d, want 3", got[0].Table)
		}
	})
	t.Run("a taken table leaves the engine's", func(t *testing.T) {
		got := AssignKept([]tournoi.Action{start("a", "c", 1, "")}, p, tables, map[int]bool{3: true}, names)
		if got[0].Table != 1 {
			t.Errorf("Alice's match on table %d, want the engine's 1", got[0].Table)
		}
	})
	t.Run("a waiting holder gets their table", func(t *testing.T) {
		got := AssignKept([]tournoi.Action{start("d", "a", 0, tournoi.ReasonWaitingTable)}, p, tables, nil, names)
		if got[0].Table != 3 || got[0].Reason != tournoi.ReasonNone {
			t.Errorf("got table %d reason %q, want 3 and no wait", got[0].Table, got[0].Reason)
		}
	})
	t.Run("a pair is concerned through one member, and no table is given twice", func(t *testing.T) {
		got := AssignKept([]tournoi.Action{start("b", "c", 1, ""), start("p", "d", 2, "")}, p, tables, nil, names)
		if got[0].Table != 4 || got[1].Table != 2 {
			t.Errorf("tables %d and %d, want 4 then the engine's 2", got[0].Table, got[1].Table)
		}
	})
	t.Run("outside the event's rooms the table is not given", func(t *testing.T) {
		inA := TablePlan{Count: 6, Settings: twoRooms(), Rooms: []string{"A"}}
		got := AssignKept([]tournoi.Action{start("a", "c", 1, "")}, inA, tables, nil, names)
		if got[0].Table != 1 {
			t.Errorf("Alice's match on table %d, want 1", got[0].Table)
		}
	})
	t.Run("a table a holder leaves goes to a match waiting in the batch", func(t *testing.T) {
		got := AssignKept([]tournoi.Action{start("a", "c", 1, ""), start("d", "x", 0, tournoi.ReasonWaitingTable)}, p, tables, nil, names)
		if got[0].Table != 3 || got[1].Table != 1 || got[1].Reason != tournoi.ReasonNone {
			t.Errorf("tables %d and %d (reason %q), want 3 then the freed 1 with no wait", got[0].Table, got[1].Table, got[1].Reason)
		}
	})
	t.Run("a freed table keeps its section's reservation", func(t *testing.T) {
		only := tournoi.Tables{Count: 6, Reserved: []tournoi.TableRule{{Table: 1, Section: "Dames"}}}
		got := AssignKept([]tournoi.Action{start("a", "c", 1, ""), start("d", "x", 0, tournoi.ReasonWaitingTable)}, p, only, nil, names)
		if got[1].Table != 0 || got[1].Reason != tournoi.ReasonWaitingTable {
			t.Errorf("waiting match of another section got table %d", got[1].Table)
		}
	})
	t.Run("a player held elsewhere is left alone", func(t *testing.T) {
		got := AssignKept([]tournoi.Action{start("a", "c", 0, tournoi.ReasonPlayerBusy)}, p, tables, nil, names)
		if got[0].Table != 0 {
			t.Errorf("busy proposal got table %d", got[0].Table)
		}
	})
}

func TestNameTablesNamesProposals(t *testing.T) {
	page := `<li class="t"><span class="num">Table 5</span>x</li>` +
		`<ol class="actions"><li>Table 5 — Ronde 1 : A v B</li><li class="alerte">Table 5 — C v D</li><li>Table 50 — E v F</li></ol>`
	got := NameTables(page, "Table", []domain.TableSetting{{Number: 5, Name: "<Stream>"}})
	for _, want := range []string{
		`<span class="num">Table 5 — &lt;Stream&gt;</span>`,
		`<li>Table 5 — &lt;Stream&gt; — Ronde 1`,
		`<li class="alerte">Table 5 — &lt;Stream&gt; — C v D`,
		`<li>Table 50 — E v F`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}
