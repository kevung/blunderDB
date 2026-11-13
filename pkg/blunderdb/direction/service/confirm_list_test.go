package service_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	tournoi "github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/render"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// phaseOf directs one phase of the given players on that many tables.
func phaseOf(t *testing.T, ctx context.Context, svc *service.Service, raw storage.Storage, phase string, tables int, players ...string) int64 {
	t.Helper()
	tid, err := raw.Tournaments().Create(ctx, "", "Open", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf(`{"name":"Open","tables":{"count":%d},"phases":[%s]}`, tables, phase)
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	for i, p := range players {
		if _, err := svc.AddParticipant(ctx, tid, p, "", float64(1600-10*i)); err != nil {
			t.Fatal(err)
		}
	}
	return tid
}

// shown is what the panel shows of a Direction: its proposals, split as a client splits them.
func shown(t *testing.T, ctx context.Context, svc *service.Service, tid int64) (launchable, held []tournoi.Action) {
	t.Helper()
	v, err := svc.GetDirection(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range v.Proposals {
		switch {
		case a.Kind == tournoi.ActWait:
		case service.Held(a):
			held = append(held, a)
		default:
			launchable = append(launchable, a)
		}
	}
	return launchable, held
}

func reasonIs(t *testing.T, err error, want direction.Reason, what string) {
	t.Helper()
	refused(t, err, what)
	if got := direction.ReasonOf(err); got != want {
		t.Errorf("%s: reason %q (%v), want %q", what, got, err, want)
	}
}

func batchesOf(t *testing.T, ctx context.Context, svc *service.Service, tid int64) [][]*tournoi.Match {
	t.Helper()
	return render.Batches(replayed(t, ctx, svc, tid))
}

// A list confirmed in one gesture is one batch: every match carries the same instant, so the
// engine reads one round — in a Swiss by rounds as in a continuous one.
func TestConfirmProposalsLaunchesOneBatch(t *testing.T) {
	for name, phase := range map[string]string{
		"rounds":     `{"kind":"swiss_lives","length":7,"lives":2,"mode":"rounds"}`,
		"continuous": `{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx, svc, raw := openService(t)
			tid := phaseOf(t, ctx, svc, raw, phase, 4, names("P", 8)...)
			list, _ := shown(t, ctx, svc, tid)
			if len(list) != 4 {
				t.Fatalf("%d launchable proposals, want 4: %+v", len(list), list)
			}
			v, err := svc.ConfirmProposals(ctx, tid, mustJSON(t, list))
			if err != nil {
				t.Fatal(err)
			}
			if len(v.Running) != 4 {
				t.Fatalf("%d matches running, want 4", len(v.Running))
			}
			if lots := batchesOf(t, ctx, svc, tid); len(lots) != 1 || len(lots[0]) != 4 {
				t.Errorf("%d batches, want one of four matches", len(lots))
			}
			for _, m := range v.Running {
				if !m.Start.Equal(v.Running[0].Start) {
					t.Errorf("match %s starts at %v, the first at %v", m.ID, m.Start, v.Running[0].Start)
				}
			}
		})
	}
}

// A list one proposal of which is no longer proposed as shown is refused whole.
func TestConfirmProposalsRefusesAListThatChanged(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid := swissOf(t, ctx, svc, raw, "Open", 4, names("P", 8)...)
	list, _ := shown(t, ctx, svc, tid)

	moved := slices.Clone(list)
	moved[1].Table = 9
	_, err := svc.ConfirmProposals(ctx, tid, mustJSON(t, moved))
	reasonIs(t, err, direction.ReasonProposalChanged, "a proposal on another table")

	twice := append(slices.Clone(list), list[0])
	_, err = svc.ConfirmProposals(ctx, tid, mustJSON(t, twice))
	reasonIs(t, err, direction.ReasonProposalChanged, "a proposal listed twice")

	if st := replayed(t, ctx, svc, tid); len(st.Running()) != 0 {
		t.Fatalf("%d matches launched by refused lists", len(st.Running()))
	}

	// Someone confirmed the last proposal meanwhile: the list read before is stale.
	if _, err := svc.ConfirmProposal(ctx, tid, mustJSON(t, list[3])); err != nil {
		t.Fatal(err)
	}
	_, err = svc.ConfirmProposals(ctx, tid, mustJSON(t, list))
	reasonIs(t, err, direction.ReasonProposalChanged, "a proposal confirmed meanwhile")
	if st := replayed(t, ctx, svc, tid); len(st.Running()) != 1 {
		t.Errorf("%d matches running, want the one confirmed alone", len(st.Running()))
	}

	_, err = svc.ConfirmProposals(ctx, tid, `[]`)
	refused(t, err, "an empty list")
	_, err = svc.ConfirmProposals(ctx, tid, `{"kind":"start_match"}`)
	refused(t, err, "a list that is not one")
}

// A proposal held for a table is not confirmable: a list holding one is refused whole, and the
// same list without it launches.
func TestConfirmProposalsRefusesAHeldProposal(t *testing.T) {
	ctx, svc, raw := openService(t)
	tid := swissOf(t, ctx, svc, raw, "Open", 2, names("P", 8)...)
	list, held := shown(t, ctx, svc, tid)
	if len(list) != 2 || len(held) != 2 {
		t.Fatalf("%d launchable and %d held, want 2 and 2", len(list), len(held))
	}
	_, err := svc.ConfirmProposals(ctx, tid, mustJSON(t, append(slices.Clone(list), held[0])))
	reasonIs(t, err, direction.ReasonProposalHeld, "a list with a held proposal")
	if st := replayed(t, ctx, svc, tid); len(st.Running()) != 0 {
		t.Fatalf("%d matches launched by a refused list", len(st.Running()))
	}
	v, err := svc.ConfirmProposals(ctx, tid, mustJSON(t, list))
	if err != nil || len(v.Running) != 2 {
		t.Fatalf("the launchable list: %v, %d running", err, len(v.Running))
	}
}

// What a repechage brings — the passage that enters the repechaged player — is proposed in the
// answer, never confirmed along.
func TestConfirmProposalsLeavesWhatFollowsARepechage(t *testing.T) {
	ctx, svc, raw := openService(t)
	phases := `{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2,"entry":"survivors"},` +
		`{"kind":"bracket","length":5,"entry":"survivors"}`
	tid := phaseOf(t, ctx, svc, raw, phases, 4, names("P", 4)...)
	playWith(t, ctx, svc, tid, tournoi.ActNextPhase, lowerWins)
	ids := slices.Clone(replayed(t, ctx, svc, tid).Order)
	slices.Sort(ids)
	if _, err := svc.WithdrawParticipant(ctx, tid, string(ids[0]), false); err != nil {
		t.Fatal(err)
	}
	list, _ := shown(t, ctx, svc, tid)
	reps := repechages(list)
	if len(reps) != 1 {
		t.Fatalf("the queue holds %+v, want one repechage", list)
	}
	v, err := svc.ConfirmProposals(ctx, tid, mustJSON(t, reps))
	if err != nil {
		t.Fatal(err)
	}
	if st := replayed(t, ctx, svc, tid); st.Current != 0 {
		t.Fatalf("the list of one repechage passed to phase %d", st.Current)
	}
	if !slices.ContainsFunc(v.Proposals, func(a tournoi.Action) bool { return a.Kind == tournoi.ActNextPhase }) {
		t.Errorf("the answer proposes %+v, want the passage to the next phase", v.Proposals)
	}
}

// In a shared room, a list read before a sister event took its tables is refused whole, and
// the refusal says why.
func TestConfirmProposalsRespectsTheSharedRoom(t *testing.T) {
	ctx, svc, raw := openService(t)
	r, err := svc.CreateRencontre(ctx, "Open de Lyon", "", "", 4)
	if err != nil {
		t.Fatal(err)
	}
	swiss := `{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}`
	a := eventIn(t, ctx, svc, raw, r.ID, "A", swiss, 4, names("A", 8)...)
	b := eventIn(t, ctx, svc, raw, r.ID, "B", swiss, 4, names("B", 4)...)
	list, _ := shown(t, ctx, svc, b)
	if len(list) != 2 {
		t.Fatalf("B proposes %+v, want two matches", list)
	}
	if v, err := svc.ConfirmAllProposals(ctx, a); err != nil || len(v.Running) != 4 {
		t.Fatalf("A fills the room: %v", err)
	}
	_, err = svc.ConfirmProposals(ctx, b, mustJSON(t, list))
	reasonIs(t, err, direction.ReasonTableTaken, "a list on the sister's tables")
	if st := replayed(t, ctx, svc, b); len(st.Running()) != 0 {
		t.Errorf("%d matches of B launched on taken tables", len(st.Running()))
	}
	// The single confirmation refuses for the same reason.
	_, err = svc.ConfirmProposal(ctx, b, mustJSON(t, list[0]))
	reasonIs(t, err, direction.ReasonTableTaken, "one proposal on a sister's table")
}

// A person seated next door is never launched here, and the refusal names that reason.
func TestConfirmProposalsRefusesAPlayerSeatedNextDoor(t *testing.T) {
	ctx, svc, raw := openService(t)
	r, err := svc.CreateRencontre(ctx, "Open de Lyon", "", "", 8)
	if err != nil {
		t.Fatal(err)
	}
	swiss := `{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}`
	a := eventIn(t, ctx, svc, raw, r.ID, "A", swiss, 8, names("P", 4)...)
	b := eventIn(t, ctx, svc, raw, r.ID, "B", swiss, 8, names("P", 4)...)
	list, _ := shown(t, ctx, svc, b)
	if len(list) != 2 {
		t.Fatalf("B proposes %+v, want two matches", list)
	}
	if v, err := svc.ConfirmAllProposals(ctx, a); err != nil || len(v.Running) != 2 {
		t.Fatalf("A launches: %v", err)
	}
	// A plays on tables 1 and 2; B's list, read before, is moved off them by hand.
	for i := range list {
		list[i].Table = 5 + i
	}
	_, err = svc.ConfirmProposals(ctx, b, mustJSON(t, list))
	reasonIs(t, err, direction.ReasonPlayerBusy, "a list seating persons who play next door")
	if st := replayed(t, ctx, svc, b); len(st.Running()) != 0 {
		t.Errorf("%d matches of B launched with their players next door", len(st.Running()))
	}
}
