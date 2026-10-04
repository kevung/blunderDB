package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
)

// playWith confirms proposals one at a time and enters every result, win deciding the winner,
// until the queue proposes an action of kind stop. A repechage is never confirmed here: it is
// the director's decision, and each test takes it itself.
func playWith(t *testing.T, ctx context.Context, svc *service.Service, tid int64, stop tournoi.ActionKind,
	win func(a, b tournoi.PlayerID) tournoi.PlayerID) {
	t.Helper()
	for step := 0; step < 400; step++ {
		v, err := svc.GetDirection(ctx, tid)
		if err != nil {
			t.Fatal(err)
		}
		var next *tournoi.Action
		for i, a := range v.Proposals {
			if a.Kind == stop {
				return
			}
			if next == nil && a.Kind != tournoi.ActWait && a.Kind != tournoi.ActFinish &&
				a.Kind != tournoi.ActRepechage && a.Reason != tournoi.ReasonWaitingTable {
				next = &v.Proposals[i]
			}
		}
		if next != nil {
			if v, err = svc.ConfirmProposal(ctx, tid, mustJSON(t, *next)); err != nil {
				t.Fatal(err)
			}
			if next.Kind != tournoi.ActStartMatch {
				continue
			}
		}
		if next == nil && len(v.Running) == 0 {
			t.Fatalf("stuck before %s: %+v", stop, v.Proposals)
		}
		for _, m := range v.Running {
			if _, err := svc.EnterResult(ctx, tid, string(m.ID), string(win(m.A, m.B)), m.Length, 2, ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Fatalf("never reached %s", stop)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func lowerWins(a, b tournoi.PlayerID) tournoi.PlayerID { return min(a, b) }

// replayed is the engine's own state of the tournament, from the journal the service holds.
func replayed(t *testing.T, ctx context.Context, svc *service.Service, tid int64) *tournoi.State {
	t.Helper()
	st, _ := replayedJournal(t, ctx, svc, tid)
	return st
}

func replayedJournal(t *testing.T, ctx context.Context, svc *service.Service, tid int64) (*tournoi.State, tournoi.Journal) {
	t.Helper()
	blob, err := svc.DirectionJournalJSON(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	j, err := tournoi.ParseJournal([]byte(blob))
	if err != nil {
		t.Fatal(err)
	}
	st, err := tournoi.Replay(j)
	if err != nil {
		t.Fatal(err)
	}
	return st, j
}

func repechages(acts []tournoi.Action) []tournoi.Action {
	var out []tournoi.Action
	for _, a := range acts {
		if a.Kind == tournoi.ActRepechage {
			out = append(out, a)
		}
	}
	return out
}

// The scenario of the club's Open A (simulation T3, N26): pools of four, two qualifiers, then a
// bracket. Pool A's winner withdraws before the draw; the director gives the place to the next
// of the pool. Played through the service end to end, then checked against the paper oracle of
// the simulation in its repechage variant: same ranks, player by player.
func TestRepechageReplaysT3AsTheOracle(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid, err := raw.Tournaments().Create(ctx, "", "Open A", "2026-10-04", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open A","tables":{"count":8},"phases":[` +
		`{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2,"entry":"survivors","name":"Poules"},` +
		`{"kind":"bracket","length":5,"entry":"survivors","name":"Tableau"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 5); err != nil {
		t.Fatal(err)
	}
	for i, p := range names("P", 16) {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}

	playWith(t, ctx, svc, tid, tournoi.ActNextPhase, lowerWins)
	st := replayed(t, ctx, svc, tid)
	var pool *tournoi.Section
	for _, sec := range st.Phases[0].Sections {
		if sec.Name == "poule:A" {
			pool = sec
		}
	}
	if pool == nil {
		t.Fatal("no pool A")
	}
	var members []tournoi.PlayerID
	for _, g := range pool.Matches {
		for _, p := range g.Players {
			if !slices.Contains(members, p) {
				members = append(members, p)
			}
		}
	}
	slices.Sort(members)
	// The lower identifier wins: the pool's order is its identifiers' order.
	gone, third := members[0], members[2]

	if _, err := svc.WithdrawParticipant(ctx, tid, string(gone), false); err != nil {
		t.Fatal(err)
	}
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	reps := repechages(v.Proposals)
	if len(reps) != 1 || reps[0].A != gone || reps[0].B != third || reps[0].Section != "poule:A" ||
		reps[0].Label.Kind != tournoi.LabelRepechage || reps[0].Label.Players != 1 {
		t.Fatalf("the queue proposes %+v, want the repechage of %s for %s in pool A", reps, third, gone)
	}
	// Before the director decides, the candidate's fate is not fixed, and the wall says so.
	page, err := svc.DirectionPageHTML(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if s := statusOf(page, playerName(v.Players, third)); !strings.HasPrefix(s, "pas encore fixé") {
		t.Errorf("the wall reads %q for the candidate, want it undecided", s)
	}

	if v, err = svc.ConfirmProposal(ctx, tid, mustJSON(t, reps[0])); err != nil {
		t.Fatal(err)
	}
	if again := repechages(v.Proposals); len(again) != 0 {
		t.Fatalf("still proposed after the confirmation: %+v", again)
	}
	if s := replayed(t, ctx, svc, tid).StatusOf(third); s.Kind != tournoi.StatusQualified || s.Phase != 1 {
		t.Errorf("the repechaged reads %+v, want qualified for the bracket", s)
	}

	playWith(t, ctx, svc, tid, tournoi.ActFinish, lowerWins)
	st, journal := replayedJournal(t, ctx, svc, tid)
	bracket := st.Phases[1]
	if !slices.Contains(bracket.Entrants, third) || slices.Contains(bracket.Entrants, gone) || len(bracket.Entrants) != 8 {
		t.Fatalf("the bracket enters %v, want eight players with %s in place of %s", bracket.Entrants, third, gone)
	}
	for _, sec := range bracket.Sections {
		for _, g := range sec.Matches {
			if g.Players[0] == tournoi.BYE || g.Players[1] == tournoi.BYE {
				t.Errorf("a bye in %s: the place went back to a bye", g.Key)
			}
		}
	}

	oracle := runOracle(t, oracleSheet(t, st, journal))
	if oracle == nil {
		return
	}
	stand, err := svc.Standings(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	app := map[string]int{}
	for _, sec := range stand.Sections {
		if sec.Name == "" {
			for _, r := range sec.Rows {
				app[r.ID] = r.Rank
			}
		}
	}
	if len(app) != len(oracle) {
		t.Errorf("the app ranks %d players, the oracle %d", len(app), len(oracle))
	}
	for p, rk := range oracle {
		if app[p] != rk {
			t.Errorf("%s: the oracle ranks %d, the app %d", p, rk, app[p])
		}
	}
}

func playerName(ps []tournoi.Player, id tournoi.PlayerID) string {
	for _, p := range ps {
		if p.ID == id {
			return p.Name
		}
	}
	return string(id)
}

// oracleSheet writes the paper sheet of the simulation's oracle from the engine's state: what a
// director notes — the pools, every result, the withdrawal, the passage, the bracket's places.
// No repechage hint: the oracle's repechage variant picks the next of the pool by itself.
func oracleSheet(t *testing.T, st *tournoi.State, journal tournoi.Journal) map[string]any {
	t.Helper()
	var players []map[string]any
	for _, p := range st.Order {
		players = append(players, map[string]any{"id": string(p), "name": st.Players[p].Name})
	}
	var phases []map[string]any
	for _, c := range st.Config.Phases {
		ph := map[string]any{"kind": c.Kind}
		if c.GroupSize > 0 {
			ph["group_size"] = c.GroupSize
		}
		if c.Qualifiers > 0 {
			ph["qualifiers"] = c.Qualifiers
		}
		if c.Entry != "" {
			ph["entry"] = c.Entry
		}
		phases = append(phases, ph)
	}
	var groups [][]string
	for _, sec := range st.Phases[0].Sections {
		var g []string
		for _, m := range sec.Matches {
			for _, p := range m.Players {
				if !slices.Contains(g, string(p)) {
					g = append(g, string(p))
				}
			}
		}
		groups = append(groups, g)
	}
	var slots []any
	for _, sec := range st.Phases[1].Sections {
		if sec.Kind != "main" || len(sec.Rounds) == 0 {
			continue
		}
		for _, i := range sec.Rounds[0] {
			for _, p := range sec.Matches[i].Players {
				if p == tournoi.BYE || p == "" {
					slots = append(slots, nil)
				} else {
					slots = append(slots, string(p))
				}
			}
		}
	}
	events := []map[string]any{{"type": "draw", "groups": groups}}
	for _, ev := range journal {
		switch ev.Kind {
		case tournoi.EvResult:
			m := st.Matches[ev.MatchID]
			events = append(events, map[string]any{"type": "match", "phase": m.Phase,
				"a": string(m.A), "b": string(m.B), "winner": string(ev.Winner)})
		case tournoi.EvPlayerWithdrawn:
			events = append(events, map[string]any{"type": "withdraw", "player": string(ev.ID)})
		case tournoi.EvNextPhase:
			events = append(events, map[string]any{"type": "next_phase"},
				map[string]any{"type": "draw", "slots": slots})
		}
	}
	return map[string]any{"name": st.Config.Name, "n26": "repechage", "players": players,
		"phases": phases, "events": events}
}

// runOracle runs the simulation's paper oracle on a sheet and returns its final ranks. Skipped
// without python3: the oracle is a script of the simulation, not a dependency of the build.
func runOracle(t *testing.T, sheet map[string]any) map[string]int {
	t.Helper()
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Log("python3 absent: the oracle is not run")
		return nil
	}
	_, here, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(here), "..", "..", "..", "..",
		"tasks", "nicomaque", "simulation-2026-10", "outils", "oracle", "oracle.py")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("oracle missing: %v", err)
	}
	path := filepath.Join(t.TempDir(), "feuille.json")
	if err := os.WriteFile(path, []byte(mustJSON(t, sheet)), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	cmd := exec.Command(py, script, path)
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		t.Fatalf("oracle: %v\n%s", err, errOut.String())
	}
	var res struct {
		Final []struct {
			Rank   int    `json:"rank"`
			Player string `json:"player"`
		} `json:"final"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) > 0 {
		t.Errorf("oracle warnings: %v", res.Warnings)
	}
	ranks := map[string]int{}
	for _, r := range res.Final {
		ranks[r.Player] = r.Rank
	}
	return ranks
}

// Between tied candidates the engine does not choose, and neither does "confirm all": the
// repechage and the passage it would change wait for the director, and once the director has
// chosen, the rest of the queue goes.
func TestConfirmAllLeavesATiedRepechageToTheDirector(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid, err := raw.Tournaments().Create(ctx, "", "Poule", "2026-10-04", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Poule","tables":{"count":4},"phases":[` +
		`{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2,"entry":"survivors"},` +
		`{"kind":"bracket","length":5,"entry":"survivors"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 5); err != nil {
		t.Fatal(err)
	}
	for i, p := range names("P", 4) {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	var ids []tournoi.PlayerID
	for _, p := range v.Players {
		ids = append(ids, p.ID)
	}
	slices.Sort(ids)
	// 2-2-1-1: the first two go through, the last two tie on one win each.
	beats := map[[2]tournoi.PlayerID]bool{
		{ids[0], ids[1]}: true, {ids[0], ids[2]}: true, {ids[1], ids[2]}: true,
		{ids[1], ids[3]}: true, {ids[2], ids[3]}: true, {ids[3], ids[0]}: true,
	}
	win := func(a, b tournoi.PlayerID) tournoi.PlayerID {
		if beats[[2]tournoi.PlayerID{a, b}] {
			return a
		}
		return b
	}
	playWith(t, ctx, svc, tid, tournoi.ActNextPhase, win)
	if _, err := svc.WithdrawParticipant(ctx, tid, string(ids[0]), false); err != nil {
		t.Fatal(err)
	}
	if v, err = svc.ConfirmAllProposals(ctx, tid); err != nil {
		t.Fatal(err)
	}
	reps := repechages(v.Proposals)
	if len(reps) != 2 || reps[0].Label.Players != 2 {
		t.Fatalf("after confirm all the queue holds %+v, want the two tied repechages still there", v.Proposals)
	}
	if st := replayed(t, ctx, svc, tid); st.Current != 0 {
		t.Fatalf("confirm all passed to phase %d over an undecided repechage", st.Current)
	}
	pick := reps[1]
	if _, err := svc.ConfirmProposal(ctx, tid, mustJSON(t, pick)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmAllProposals(ctx, tid); err != nil {
		t.Fatal(err)
	}
	st := replayed(t, ctx, svc, tid)
	if st.Current != 1 || !slices.Contains(st.Phases[1].Entrants, pick.B) {
		t.Fatalf("phase %d enters %v, want the bracket with %s", st.Current, st.Phases[st.Current].Entrants, pick.B)
	}
}

// A unique repechage rides along with "confirm all", and the queue is proposed again behind it:
// the passage that follows enters the repechaged player.
func TestConfirmAllTakesAUniqueRepechage(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid, err := raw.Tournaments().Create(ctx, "", "Poule", "2026-10-04", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Poule","tables":{"count":4},"phases":[` +
		`{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2,"entry":"survivors"},` +
		`{"kind":"bracket","length":5,"entry":"survivors"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 5); err != nil {
		t.Fatal(err)
	}
	for i, p := range names("P", 4) {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	playWith(t, ctx, svc, tid, tournoi.ActNextPhase, lowerWins)
	st := replayed(t, ctx, svc, tid)
	ids := slices.Clone(st.Order)
	slices.Sort(ids)
	if _, err := svc.WithdrawParticipant(ctx, tid, string(ids[0]), false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmAllProposals(ctx, tid); err != nil {
		t.Fatal(err)
	}
	st = replayed(t, ctx, svc, tid)
	if st.Current != 1 || !slices.Contains(st.Phases[1].Entrants, ids[2]) || slices.Contains(st.Phases[1].Entrants, ids[0]) {
		t.Fatalf("phase %d enters %v, want %s in place of %s", st.Current, st.Phases[st.Current].Entrants, ids[2], ids[0])
	}
}
