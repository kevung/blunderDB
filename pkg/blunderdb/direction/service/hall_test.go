package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// eventIn directs an event of the given phase on the room's tables and attaches it.
func eventIn(t *testing.T, ctx context.Context, svc *service.Service, raw storage.Storage, rid int64, name, phase string, tables int, players ...string) int64 {
	t.Helper()
	tid, err := raw.Tournaments().Create(ctx, "", name, "2026-10-04", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf(`{"name":%q,"tables":{"count":%d},"phases":[%s]}`, name, tables, phase)
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	for i, p := range players {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.AttachToRencontre(ctx, tid, rid); err != nil {
		t.Fatal(err)
	}
	return tid
}

const bracket = `{"kind":"bracket","length":7}`

// drawAndLaunch confirms a knockout's draw, then every quarter-final it can launch.
func drawAndLaunch(t *testing.T, ctx context.Context, svc *service.Service, tid int64) *service.DirectionView {
	t.Helper()
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := json.Marshal(v.Proposals[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmProposal(ctx, tid, string(blob)); err != nil {
		t.Fatal(err)
	}
	if v, err = svc.ConfirmAllProposals(ctx, tid); err != nil {
		t.Fatal(err)
	}
	return v
}

// The person in a quarter-final of one event is in no launchable proposal of its sister, and
// launching that proposal anyway is refused: one person never sits at two matches.
func TestHallNeverLaunchesAPlayerAtASisterMatch(t *testing.T) {
	ctx, svc, raw := openService(t)
	r, err := svc.CreateRencontre(ctx, "Open de Lyon", "", "", 8)
	if err != nil {
		t.Fatal(err)
	}
	a := eventIn(t, ctx, svc, raw, r.ID, "A", bracket, 8, append([]string{"Hugo Bastide"}, names("A", 7)...)...)
	b := eventIn(t, ctx, svc, raw, r.ID, "B", bracket, 8, append([]string{"Hugo Bastide"}, names("B", 7)...)...)
	if v := drawAndLaunch(t, ctx, svc, a); len(v.Running) != 4 {
		t.Fatalf("A runs %d quarter-finals, want 4", len(v.Running))
	}
	v := drawAndLaunch(t, ctx, svc, b)
	if len(v.Running) != 3 {
		t.Fatalf("B runs %d quarter-finals, want the 3 without Hugo Bastide", len(v.Running))
	}
	hugo := tournoi.PlayerID("")
	for _, p := range v.Players {
		if p.Name == "Hugo Bastide" {
			hugo = p.ID
		}
	}
	hall, err := svc.RencontreTableGrid(ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range hall.Queue {
		if p.TournamentID == b && (p.Action.A == hugo || p.Action.B == hugo) {
			t.Fatalf("the Hall offers to launch Hugo Bastide's quarter-final of B while he plays in A: %+v", p.Action)
		}
	}
	var held *tournoi.Action
	for _, p := range hall.Held {
		if p.TournamentID == b && (p.Action.A == hugo || p.Action.B == hugo) {
			held = &p.Action
		}
	}
	if held == nil || held.Reason != tournoi.ReasonPlayerBusy {
		t.Fatalf("Hugo Bastide's quarter-final of B is not held as playing elsewhere: %+v", hall.Held)
	}
	blob, err := json.Marshal(held)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmProposal(ctx, b, string(blob)); !errors.Is(err, direction.ErrRefused) {
		t.Fatalf("launching Hugo Bastide in B while he plays in A: err %v, want a refusal", err)
	}
}

// With every table of the room taken, the Hall launches nothing of a sister event: its matches
// wait for a table, and the wall page shows no match running without one.
func TestHallHoldsMatchesWithNoTable(t *testing.T) {
	ctx, svc, raw := openService(t)
	r, err := svc.CreateRencontre(ctx, "Open de Lyon", "", "", 8)
	if err != nil {
		t.Fatal(err)
	}
	swiss := `{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}`
	a := eventIn(t, ctx, svc, raw, r.ID, "A", swiss, 8, names("A", 16)...)
	b := eventIn(t, ctx, svc, raw, r.ID, "B", swiss, 8, names("B", 4)...)
	if v, err := svc.ConfirmAllProposals(ctx, a); err != nil || len(v.Running) != 8 {
		t.Fatalf("A fills the room: %v", err)
	}
	hall, err := svc.RencontreTableGrid(ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range hall.Queue {
		if p.TournamentID == b && p.Action.Kind == tournoi.ActStartMatch {
			t.Errorf("the Hall offers to launch a match of B with every table taken: %+v", p.Action)
		}
	}
	waiting := 0
	for _, p := range hall.Held {
		if p.TournamentID == b && p.Action.Reason == tournoi.ReasonWaitingTable {
			waiting++
		}
	}
	if waiting != 2 {
		t.Errorf("%d matches of B wait for a table, want 2", waiting)
	}
	for _, c := range hall.Cells {
		if c.NoTable {
			t.Errorf("the grid shows a match with no table: %+v", c)
		}
	}
}

// The Hall's queue mixes the events: the second event's first match is not behind every
// proposal of the first one.
func TestHallQueueInterleavesEvents(t *testing.T) {
	ctx, svc, raw := openService(t)
	r, err := svc.CreateRencontre(ctx, "Open de Lyon", "", "", 16)
	if err != nil {
		t.Fatal(err)
	}
	swiss := `{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}`
	a := eventIn(t, ctx, svc, raw, r.ID, "A", swiss, 16, names("A", 8)...)
	b := eventIn(t, ctx, svc, raw, r.ID, "B", swiss, 16, names("B", 8)...)
	hall, err := svc.RencontreTableGrid(ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	var order []int64
	for _, p := range hall.Queue {
		order = append(order, p.TournamentID)
	}
	want := []int64{a, b, a, b, a, b, a, b}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Fatalf("Hall queue by event %v, want %v", order, want)
	}

	// A's first match ends: its players have just played, and wait behind everyone else.
	v, err := svc.ConfirmAllProposals(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	m := v.Running[0]
	if _, err := svc.EnterResult(ctx, a, string(m.ID), string(m.A), 7, 2, ""); err != nil {
		t.Fatal(err)
	}
	if hall, err = svc.RencontreTableGrid(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if len(hall.Queue) == 0 || hall.Queue[0].TournamentID != b {
		t.Fatalf("after A's first result the Hall's queue starts with %+v, want B, which has waited since the start", hall.Queue)
	}
}
