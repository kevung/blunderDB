package database

import (
	"encoding/json"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// Changing the configuration of a tournament under way.
//
// The panel must SAY, before the click, exactly what the engine will accept and refuse.

// configOf reads the configuration in force as the frontend does — as JSON, so a test edits
// what a form would send.
func configOf(t *testing.T, d *Database, tID int64) tournoi.Config {
	t.Helper()
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	return v.Config
}

func mustJSON(t *testing.T, cfg tournoi.Config) string {
	t.Helper()
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// changeCodes lists the codes of a preview, so a test names what it expects.
func changeCodes(changes []ConfigChange) string {
	var s []string
	for _, c := range changes {
		s = append(s, c.Code)
	}
	return strings.Join(s, ",")
}

func TestDirectionConfig_PreviewNamesEveryChange(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)

	cfg := configOf(t, d, tID)
	cfg.Phases[1].Length = 11
	cfg.Tables.Count = 12
	p, err := d.PreviewDirectionConfig(tID, mustJSON(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes) != 2 {
		t.Fatalf("expected two changes, got %s", changeCodes(p.Changes))
	}
	if p.Changes[0].Code != "tableCount" || p.Changes[0].From != "8" || p.Changes[0].To != "12" {
		t.Errorf("table change reads %+v", p.Changes[0])
	}
	if p.Changes[1].Code != "length" || p.Changes[1].Phase != 2 || p.Changes[1].To != "11" {
		t.Errorf("length change reads %+v", p.Changes[1])
	}
	if len(p.Refusals) != 0 {
		t.Errorf("nothing is refused in preparation, got %+v", p.Refusals)
	}
}

// The unmodified configuration is the case the settings view opens on: no change, but the locks
// must already be there — that is how the view knows what to grey out.
func TestDirectionConfig_PreviewOfNoChangeStillLocks(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 16)
	runningMatch(t, d, tID)

	p, err := d.PreviewDirectionConfig(tID, mustJSON(t, configOf(t, d, tID)))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes) != 0 {
		t.Fatalf("an unmodified configuration changes nothing, got %s", changeCodes(p.Changes))
	}
	if len(p.Locks) != 2 {
		t.Fatalf("two phases, two locks, got %d", len(p.Locks))
	}
	if !p.Locks[0].Locked || p.Locks[0].Reason == "" {
		t.Errorf("the phase under way is frozen with a reason, got %+v", p.Locks[0])
	}
	if p.Locks[1].Locked {
		t.Errorf("a phase not yet opened is not frozen, got %+v", p.Locks[1])
	}
	if p.Opened != 1 || !p.Started {
		t.Errorf("one phase opened on a started tournament, got opened=%d started=%v", p.Opened, p.Started)
	}
}

func TestDirectionConfig_KindOfAStartedPhaseIsRefused(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 16)
	runningMatch(t, d, tID)

	cfg := configOf(t, d, tID)
	cfg.Phases[0].Kind = tournoi.KindRoundRobin
	body := mustJSON(t, cfg)

	p, err := d.PreviewDirectionConfig(tID, body)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Refusals) != 1 || p.Refusals[0].Code != "kind" || p.Refusals[0].Phase != 1 {
		t.Fatalf("the format of the phase under way is refused, got %+v", p.Refusals)
	}
	if p.Refusals[0].Reason == "" {
		t.Error("a refusal without a reason leaves the director with nothing to do")
	}
	// And the preview told the truth: the engine refuses it too.
	if err := d.SetDirectionConfig(tID, body); err == nil {
		t.Error("the engine accepted a format change on a phase under way")
	}
}

// The switch lowered at 22 h: the change that made this issue exist.
func TestDirectionConfig_LowerTheSwitchMidTournament(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 16)
	runningMatch(t, d, tID)

	cfg := configOf(t, d, tID)
	cfg.Phases[0].Target = 8
	cfg.Phases[1].Length = 11
	if err := d.SetDirectionConfig(tID, mustJSON(t, cfg)); err != nil {
		t.Fatalf("lowering the switch: %v", err)
	}
	after := configOf(t, d, tID)
	if after.Phases[0].Target != 8 || after.Phases[1].Length != 11 {
		t.Fatalf("the configuration did not take: %+v", after.Phases)
	}
	// A change of mind is a decision, and a decision is readable in the history.
	entries, err := d.History(tID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.Kind == string(tournoi.EvConfigChanged) {
			found = true
		}
	}
	if !found {
		t.Error("the configuration change left no trace in the history")
	}
}

// A consolation decided on the Saturday evening: a phase added AFTER the current one does not
// interrupt what is being played.
func TestDirectionConfig_PhaseAddedAfterTheCurrentOne(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 16)
	runningMatch(t, d, tID)
	before, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}

	cfg := configOf(t, d, tID)
	cfg.Phases = append(cfg.Phases, tournoi.PhaseConfig{Kind: tournoi.KindBracket, Length: 5, Consolation: true})
	p, err := d.PreviewDirectionConfig(tID, mustJSON(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Refusals) != 0 {
		t.Fatalf("adding a phase after the current one is allowed, got %+v", p.Refusals)
	}
	if len(p.Changes) != 1 || p.Changes[0].Code != "phaseAdded" || p.Changes[0].Phase != 3 {
		t.Fatalf("the added phase should be the only change, got %+v", p.Changes)
	}
	if err := d.SetDirectionConfig(tID, mustJSON(t, cfg)); err != nil {
		t.Fatal(err)
	}
	after, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Config.Phases) != 3 {
		t.Fatalf("three phases expected, got %d", len(after.Config.Phases))
	}
	if after.Phase != before.Phase {
		t.Errorf("the phase under way moved from %d to %d", before.Phase, after.Phase)
	}
	if len(after.Running) != len(before.Running) {
		t.Errorf("the running matches were disturbed: %d then %d", len(before.Running), len(after.Running))
	}
}

func TestDirectionConfig_AnOpenedPhaseIsNotRemovable(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 16)
	runningMatch(t, d, tID)

	cfg := configOf(t, d, tID)
	cfg.Phases = cfg.Phases[:1]
	body := mustJSON(t, cfg)
	p, err := d.PreviewDirectionConfig(tID, body)
	if err != nil {
		t.Fatal(err)
	}
	// The second phase is not open: removing it is legal, and the engine accepts it.
	if len(p.Refusals) != 0 {
		t.Fatalf("removing a phase not yet opened is allowed, got %+v", p.Refusals)
	}
	if err := d.SetDirectionConfig(tID, body); err != nil {
		t.Fatal(err)
	}

	// The FIRST phase, however, is open, and nothing removes it.
	cfg = configOf(t, d, tID)
	cfg.Phases = nil
	if _, err := d.PreviewDirectionConfig(tID, mustJSON(t, cfg)); err == nil {
		t.Fatal("a configuration without any phase is not a configuration")
	}
}

// Reopen, correct, reclose: the point of reopening is that the final standings take the
// correction into account.
func TestDirectionConfig_ReopenCorrectReclose(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 8)
	playToTheEnd(t, d, tID)
	if _, err := d.CloseDirection(tID); err != nil {
		t.Fatal(err)
	}
	closed, err := d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	if !closed.Finished || len(closed.Sections) == 0 || len(closed.Sections[0].Rows) == 0 {
		t.Fatal("a closed tournament has final standings")
	}
	winner := closed.Sections[0].Rows[0].ID

	if _, err := d.ReopenDirection(tID); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Finished || v.State != "running" {
		t.Fatalf("a reopened tournament is running again, got finished=%v state=%s", v.Finished, v.State)
	}

	// The last match was wrong: the other player had won.
	last, err := d.FinishedMatches(tID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(last) == 0 {
		t.Fatal("no finished match to correct")
	}
	m := last[0]
	// Whose result it was is read from the log, which is where a director reads it too.
	entries, err := d.History(tID, "", m.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	won := ""
	for _, e := range entries {
		if e.Winner != "" {
			won = e.Winner
		}
	}
	other := m.A
	if won == m.A {
		other = m.B
	}
	if _, err := d.CorrectResult(tID, m.MatchID, other, 0, 0, ""); err != nil {
		t.Fatalf("correcting after reopening: %v", err)
	}
	playToTheEnd(t, d, tID)
	if _, err := d.CloseDirection(tID); err != nil {
		t.Fatal(err)
	}
	reclosed, err := d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	if !reclosed.Finished {
		t.Fatal("the tournament was reclosed")
	}
	if reclosed.Sections[0].Rows[0].ID == winner {
		t.Errorf("the final standings ignored the correction: %s still first", winner)
	}
}

// A table out of service (`tables.unavailable`): adding it and taking it back are both changes,
// and both are named, or "save" would look like a no-op.
func TestDirectionConfig_PreviewNamesAnUnavailableTable(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)

	cfg := configOf(t, d, tID)
	cfg.Tables.Unavailable = []int{7, 5}
	p, err := d.PreviewDirectionConfig(tID, mustJSON(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if changeCodes(p.Changes) != "tablesUnavailable" {
		t.Fatalf("expected the unavailable tables, got %s", changeCodes(p.Changes))
	}
	if c := p.Changes[0]; c.From != "" || c.To != "5, 7" {
		t.Errorf("the change reads the tables in order, got %+v", c)
	}
	if err := d.SetDirectionConfig(tID, mustJSON(t, cfg)); err != nil {
		t.Fatal(err)
	}

	back := configOf(t, d, tID)
	back.Tables.Unavailable = []int{5}
	p, err = d.PreviewDirectionConfig(tID, mustJSON(t, back))
	if err != nil {
		t.Fatal(err)
	}
	if changeCodes(p.Changes) != "tablesUnavailable" || p.Changes[0].From != "5, 7" || p.Changes[0].To != "5" {
		t.Errorf("taking a table back into service reads %+v", p.Changes)
	}
}

// A reserved table is a change too, including when only what it is reserved FOR changes.
func TestDirectionConfig_PreviewNamesAReservedTable(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)

	cfg := configOf(t, d, tID)
	cfg.Tables.Reserved = []tournoi.TableRule{{Table: 8, Section: "main", Phase: 2}}
	p, err := d.PreviewDirectionConfig(tID, mustJSON(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if changeCodes(p.Changes) != "tablesReserved" || p.Changes[0].To != "8" {
		t.Fatalf("expected the reserved table, got %+v", p.Changes)
	}
	if err := d.SetDirectionConfig(tID, mustJSON(t, cfg)); err != nil {
		t.Fatal(err)
	}

	other := configOf(t, d, tID)
	other.Tables.Reserved = []tournoi.TableRule{{Table: 8, Section: "conso"}}
	p, err = d.PreviewDirectionConfig(tID, mustJSON(t, other))
	if err != nil {
		t.Fatal(err)
	}
	if changeCodes(p.Changes) != "tablesReserved" {
		t.Errorf("a reservation that changes its purpose is a change, got %s", changeCodes(p.Changes))
	}
}

// The board breaks under a match being played: the list says so, and names a free table to
// move it to — the director should not have to go and count them.
func TestDirectionConfig_UnavailableTableUnderARunningMatch(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 12)
	runningMatch(t, d, tID)

	cfg := configOf(t, d, tID)
	busy := map[int]bool{}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range v.Running {
		busy[m.Table] = true
	}
	if !busy[3] || busy[7] {
		t.Fatalf("the fixture expects tables 1-6 busy, got %v", busy)
	}
	cfg.Tables.Unavailable = []int{3}
	p, err := d.PreviewDirectionConfig(tID, mustJSON(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if changeCodes(p.Changes) != "tablesUnavailable,unavailableBusy" {
		t.Fatalf("expected the table and its match, got %s", changeCodes(p.Changes))
	}
	if c := p.Changes[1]; c.From != "3" || c.To != "7" {
		t.Errorf("the match on table 3 should be offered table 7, got %+v", c)
	}

	// The whole hall is busy: the list still says it, without inventing a table.
	cfg.Tables.Count = 6
	p, err = d.PreviewDirectionConfig(tID, mustJSON(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(changeCodes(p.Changes), "unavailableBusyFull") {
		t.Errorf("no free table left must be said, got %s", changeCodes(p.Changes))
	}

	// And the engine takes the table out of service mid-tournament.
	cfg.Tables.Count = 8
	if err := d.SetDirectionConfig(tID, mustJSON(t, cfg)); err != nil {
		t.Fatalf("declaring a table out of service mid-tournament: %v", err)
	}
}
