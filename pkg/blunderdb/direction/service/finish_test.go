package service_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
)

// playOut directs a tournament until only the close is left to confirm, the lower identifier
// winning every match so the run is deterministic.
func playOut(t *testing.T, ctx context.Context, svc *service.Service, tid int64) *service.DirectionView {
	t.Helper()
	for step := 0; step < 400; step++ {
		v, err := svc.GetDirection(ctx, tid)
		if err != nil {
			t.Fatal(err)
		}
		actionable := 0
		for _, a := range v.Proposals {
			if a.Kind == tournoi.ActWait || a.Kind == tournoi.ActFinish || a.Reason == tournoi.ReasonWaitingTable {
				continue
			}
			actionable++
		}
		if actionable == 0 && len(v.Running) == 0 {
			return v
		}
		if actionable > 0 {
			if v, err = svc.ConfirmAllProposals(ctx, tid); err != nil {
				t.Fatal(err)
			}
		}
		for _, m := range v.Running {
			w := m.A
			if string(m.B) < string(m.A) {
				w = m.B
			}
			if _, err := svc.EnterResult(ctx, tid, string(m.ID), string(w), m.Length, 2, ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Fatal("the tournament never ran out of matches")
	return nil
}

// confirmFinish confirms the close the queue proposes, as the director does from the queue.
func confirmFinish(t *testing.T, ctx context.Context, svc *service.Service, tid int64) {
	t.Helper()
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range v.Proposals {
		if a.Kind != tournoi.ActFinish {
			continue
		}
		blob, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.ConfirmProposal(ctx, tid, string(blob)); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatalf("no close proposed in %+v", v.Proposals)
}

// assertState checks the stored state through both reads the panel and the CLI use.
func assertState(t *testing.T, ctx context.Context, svc *service.Service, tid int64, want string) {
	t.Helper()
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if v.State != want {
		t.Errorf("GetDirection state %q, want %q", v.State, want)
	}
	all, err := svc.ListDirections(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range all {
		if s.TournamentID == tid && s.State != want {
			t.Errorf("ListDirections state %q, want %q", s.State, want)
		}
	}
}

// Closing from the queue is the same close as the standings' button: the stored state follows
// the log, or the tournament list calls a closed tournament "running".
func TestClosingFromTheQueueStoresFinished(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid := swissOf(t, ctx, svc, raw, "Club", 4, names("P", 6)...)
	playOut(t, ctx, svc, tid)

	confirmFinish(t, ctx, svc, tid)
	assertState(t, ctx, svc, tid, "finished")

	if _, err := svc.ReopenDirection(ctx, tid); err != nil {
		t.Fatal(err)
	}
	assertState(t, ctx, svc, tid, "running")

	confirmFinish(t, ctx, svc, tid)
	assertState(t, ctx, svc, tid, "finished")
}

// The CSV names the phase each player reached: the qualified reach the bracket, the others stop
// in the Swiss, and a spreadsheet filters on that column.
func TestStandingsCSVNamesThePhaseReached(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid, err := raw.Tournaments().Create(ctx, "", "Club", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Club","tables":{"count":4},"phases":[` +
		`{"kind":"swiss_lives","length":5,"lives":2,"mode":"continuous","name":"Suisse"},` +
		`{"kind":"bracket","length":5,"entry":"top:4","name":"Tableau"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 5); err != nil {
		t.Fatal(err)
	}
	for i, p := range names("P", 8) {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	playOut(t, ctx, svc, tid)

	out, err := svc.StandingsCSV(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	head := strings.Split(lines[0], ";")
	col := -1
	for i, h := range head {
		if h == "Phase" {
			col = i
		}
	}
	if col < 0 {
		t.Fatalf("no Phase column in %q", lines[0])
	}
	reached := map[string]int{}
	for _, l := range lines[1:9] {
		reached[strings.Split(l, ";")[col]]++
	}
	if reached["Tableau"] != 4 || reached["Suisse"] != 4 {
		t.Errorf("phase reached by the 8 players: %v, want 4 in the bracket and 4 in the Swiss\n%s", reached, out)
	}
}
