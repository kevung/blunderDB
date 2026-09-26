package direction

import (
	"context"
	"fmt"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// festival is S3 (tasks/nicomaque/simulation-2026-09/rapport/S3.md): a main event of 64, a speed
// of 32 and doubles of 16 pairs, three Directions in one room of 14 tables, 28 of the speed's
// players also in the main event, and the doubles' 32 members the other half of it.
type festival struct {
	dirs  []*Direction
	names []string
	// members are the persons behind each doubles Participant, per Direction.
	members []map[string][]string
	// ends is when each running match finishes, per Direction.
	ends []map[tournoi.MatchID]time.Time
}

func newFestival(t *testing.T) *festival {
	t.Helper()
	ctx := context.Background()
	store := newMemStore()
	start := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	room := tournoi.Tables{Count: 14}
	events := []struct {
		name    string
		length  int
		players []tournoi.Player
	}{
		{"principal", 7, festivalPlayers("p", 64, 0)},
		// The speed: 28 of the main event's players, and 4 from outside.
		{"speed", 3, append(festivalPlayers("p", 28, 0), festivalPlayers("x", 4, 0)...)},
		{"doubles", 5, festivalPlayers("d", 16, 0)},
	}
	f := &festival{}
	for i, e := range events {
		cfg := tournoi.Config{Name: e.name, Tables: room, MinPerPoint: 8,
			Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindBracket, Length: e.length}}}
		d, err := Create(ctx, store, int64(i+1), cfg, int64(11+i), start)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range e.players {
			if err := d.Enter(ctx, p, start); err != nil {
				t.Fatal(err)
			}
		}
		f.dirs = append(f.dirs, d)
		f.names = append(f.names, e.name)
		f.ends = append(f.ends, map[tournoi.MatchID]time.Time{})
		f.members = append(f.members, nil)
	}
	// Each pair is two players of the main event's lower half: the pair is busy as soon as one
	// of them plays there.
	pairs := map[string][]string{}
	for k := range 16 {
		pairs[fmt.Sprintf("d%02d", k+1)] = []string{fmt.Sprintf("p%02d", 33+2*k), fmt.Sprintf("p%02d", 34+2*k)}
	}
	f.members[2] = pairs
	return f
}

func festivalPlayers(prefix string, n, from int) []tournoi.Player {
	out := make([]tournoi.Player, n)
	for i := range out {
		id := fmt.Sprintf("%s%02d", prefix, from+i+1)
		out[i] = tournoi.Player{ID: tournoi.PlayerID(id), Name: id, Rating: float64(3 + i%9)}
	}
	return out
}

// play runs the weekend in five-minute steps, as a director confirming every proposal that
// carries a table would — holding no one back by hand — and returns how many times a table
// carried two matches of the room at once, and a person sat at two of them. seeRoom says whether
// each Direction is told what its sisters occupy: their tables and their players.
func (f *festival) play(t *testing.T, seeRoom bool) (collisions, doubled, launched int) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	for step := 0; step < 2000; step++ {
		now = now.Add(5 * time.Minute)
		// Results first: a table freed at this instant is free for this instant's proposals.
		for i, d := range f.dirs {
			for _, m := range d.State().Running() {
				if end, ok := f.ends[i][m.ID]; ok && !end.After(now) {
					ev := tournoi.ResultEvent(m.ID, m.A, m.Length, m.Length/2, now)
					if err := d.Apply(ctx, ev); err != nil {
						t.Fatalf("%s result %s: %v", f.names[i], m.ID, err)
					}
				}
			}
		}
		done := true
		for i, d := range f.dirs {
			if d.State().Finished {
				continue
			}
			done = false
			var others []*Direction
			var sisters []Sister
			for j, o := range f.dirs {
				if j != i {
					others = append(others, o)
					sisters = append(sisters, Sister{Name: f.names[j], Dir: o, Members: f.members[j]})
				}
			}
			ext := tournoi.External{}
			if seeRoom {
				ext.BusyTables = BusyTables(others...)
				ext.BusyPlayers = BusyPlayers(d.BusyIn(PlayingElsewhere(sisters...), f.members[i]))
			}
			for again := true; again; {
				again = false
				for _, a := range d.ProposeWith(now, ext) {
					if a.Kind == tournoi.ActDraw || a.Kind == tournoi.ActNextPhase || a.Kind == tournoi.ActBye {
						// A draw or a phase change reshapes what comes next: apply it, then ask again.
						ev, err := d.EventFor(a, now)
						if err != nil {
							t.Fatalf("%s: %v", f.names[i], err)
						}
						if err := d.Apply(ctx, ev); err != nil {
							t.Fatalf("%s: %v", f.names[i], err)
						}
						again = true
						break
					}
					switch {
					case a.Kind == tournoi.ActFinish:
						if err := d.Finish(ctx, now); err != nil {
							t.Fatal(err)
						}
						continue
					case a.Kind != tournoi.ActStartMatch, a.Table <= 0,
						a.Reason == tournoi.ReasonWaitingTable, a.Reason == tournoi.ReasonPlayerUnavailable,
						a.Reason == tournoi.ReasonPlayerBusy:
						continue
					}
					ev, err := d.EventFor(a, now)
					if err != nil {
						t.Fatalf("%s: %v", f.names[i], err)
					}
					if err := d.Apply(ctx, ev); err != nil {
						t.Fatalf("%s: %v", f.names[i], err)
					}
					f.ends[i][ev.MatchID] = now.Add(time.Duration(a.Length*8) * time.Minute)
					launched++
				}
			}
		}
		collisions += tableClashes(f.dirs)
		doubled += f.personClashes()
		if done {
			return collisions, doubled, launched
		}
	}
	for i, d := range f.dirs {
		t.Logf("%s finished=%v running=%d proposals=%+v", f.names[i], d.State().Finished, len(d.State().Running()), d.ProposeWith(now, tournoi.External{}))
	}
	t.Fatal("the festival never ended")
	return 0, 0, 0
}

// personClashes counts the persons — by name, a pair's members each — sat at more than one
// running match of the room at once.
func (f *festival) personClashes() int {
	seen := map[string]int{}
	for i, d := range f.dirs {
		for _, m := range d.State().Running() {
			for _, id := range []tournoi.PlayerID{m.A, m.B} {
				for _, name := range persons(d.State(), id, f.members[i]) {
					seen[name]++
				}
			}
		}
	}
	n := 0
	for _, c := range seen {
		if c > 1 {
			n += c - 1
		}
	}
	return n
}

// tableClashes counts the tables that carry more than one running match across the room.
func tableClashes(dirs []*Direction) int {
	seen := map[int]int{}
	for _, d := range dirs {
		for _, m := range d.State().Running() {
			if m.Table > 0 {
				seen[m.Table]++
			}
		}
	}
	n := 0
	for _, c := range seen {
		if c > 1 {
			n += c - 1
		}
	}
	return n
}

// TestRencontreS3SansCollision is ADR-0056's guard: S3 replayed with each Direction told the
// tables and players its sisters occupy puts no two matches on one table and no person — a
// pair's members included — at two matches, with no one held back by hand, and every event
// still ends.
func TestRencontreS3SansCollision(t *testing.T) {
	f := newFestival(t)
	collisions, doubled, launched := f.play(t, true)
	if collisions != 0 {
		t.Errorf("S3 with the room shared: %d collision(s), want 0", collisions)
	}
	if doubled != 0 {
		t.Errorf("S3 with the room shared: %d person(s) at two matches at once, want 0", doubled)
	}
	for i, d := range f.dirs {
		if !d.State().Finished {
			t.Errorf("%s did not finish", f.names[i])
		}
	}
	if launched < 63+31+15 {
		t.Errorf("only %d matches launched, the three brackets need %d", launched, 63+31+15)
	}

	// The same weekend with each Direction believing it has the room to itself: the measure S3
	// made. If this stops colliding, the guard above proves nothing.
	blind := newFestival(t)
	c, twice, _ := blind.play(t, false)
	t.Logf("S3: %d matches, 0 collision with the room shared; blind, %d table and %d person clash(es)", launched, c, twice)
	if c == 0 {
		t.Error("the blind replay no longer collides: the guard measures nothing")
	}
	if twice == 0 {
		t.Error("the blind replay never sits a person at two matches: the guard measures nothing")
	}
}

func TestRoomAlignsTablesAndKeepsReservations(t *testing.T) {
	cfg := tournoi.Config{Tables: tournoi.Tables{Count: 8, Unavailable: []int{2},
		Reserved: []tournoi.TableRule{{Table: 1}}}}
	got := WithTables(cfg, Room{Tables: 14, Unavailable: []int{12, 3, 3}})
	if got.Tables.Count != 14 || fmt.Sprint(got.Tables.Unavailable) != "[3 12]" || len(got.Tables.Reserved) != 1 {
		t.Fatalf("WithTables = %+v", got.Tables)
	}
	if !SameRoom(got, Room{Tables: 14, Unavailable: []int{12, 3}}) {
		t.Error("SameRoom must ignore order")
	}
}
