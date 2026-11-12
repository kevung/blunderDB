package service_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	tournoi "github.com/kevung/nicomaque"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// directed opens a fresh database, creates a Direction of cfg and enters players a, b, c, d.
func directed(t *testing.T, cfg string) (context.Context, *service.Service, func() *tournoi.State, int64) {
	t.Helper()
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := service.New(st, "", nil)
	tid, err := st.Tournaments().Create(ctx, "", "Club", "2026-11-12", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b", "c", "d"} {
		if _, err := svc.AddParticipantAs(ctx, tid, id, "Joueur "+id, "", 0); err != nil {
			t.Fatal(err)
		}
	}
	state := func() *tournoi.State {
		t.Helper()
		dir, err := direction.Open(ctx, storage.BindDirection(st.Directions(), ""), tid)
		if err != nil {
			t.Fatal(err)
		}
		return dir.State()
	}
	return ctx, svc, state, tid
}

func refused(t *testing.T, err error, what string) {
	t.Helper()
	if !errors.Is(err, direction.ErrRefused) {
		t.Errorf("%s: %v, want a refusal", what, err)
	}
}

// TestManualPhaseTakesMatchesAndByesByRound: in a manual phase the director places each match
// and each bye in a round; the engine refuses a player placed twice in one round.
func TestManualPhaseTakesMatchesAndByesByRound(t *testing.T) {
	ctx, svc, state, tid := directed(t, `{"name":"Club","tables":{"count":4},"phases":[{"kind":"manual","length":5}]}`)
	if _, err := svc.StartMatchInRound(ctx, tid, 1, "a", "b", 0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PlaceBye(ctx, tid, "c", 0); err != nil {
		t.Fatal(err)
	}
	_, err := svc.StartMatchInRound(ctx, tid, 1, "c", "d", 0, 0)
	refused(t, err, "c already has a bye in round 1")
	if _, err := svc.CancelBye(ctx, tid, "c", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartMatchManually(ctx, tid, "c", "d", 0, 0); err != nil {
		t.Fatalf("round 0 is the latest round: %v", err)
	}
	s := state()
	if got := len(s.Running()); got != 2 {
		t.Fatalf("%d matches running, want 2", got)
	}
	for _, m := range s.Running() {
		if m.Round != 1 || m.Length != 5 || m.Table == 0 {
			t.Errorf("match %s: round %d, length %d, table %d; want round 1, length 5, a table", m.ID, m.Round, m.Length, m.Table)
		}
	}
	if byes := s.Phases[0].ByeRounds["c"]; len(byes) != 0 {
		t.Errorf("c's bye was cancelled, still %v", byes)
	}
}

// TestOnlyAManualPhaseTakesARound: elsewhere the engine numbers its own rounds and a bye is its
// decision.
func TestOnlyAManualPhaseTakesARound(t *testing.T) {
	ctx, svc, _, tid := directed(t, `{"name":"Club","tables":{"count":4},"phases":[{"kind":"swiss_rounds","length":5,"rounds":3}]}`)
	_, err := svc.StartMatchInRound(ctx, tid, 2, "a", "b", 0, 0)
	refused(t, err, "a round outside a manual phase")
	_, err = svc.PlaceBye(ctx, tid, "a", 1)
	refused(t, err, "a director's bye outside a manual phase")
}

// TestHalfByeIsAnnouncedAndWithdrawn: a Swiss by rounds records the rounds a player sits out.
func TestHalfByeIsAnnouncedAndWithdrawn(t *testing.T) {
	ctx, svc, state, tid := directed(t, `{"name":"Club","tables":{"count":4},"phases":[{"kind":"swiss_rounds","length":5,"rounds":3}]}`)
	if _, err := svc.GiveHalfBye(ctx, tid, "a", 2); err != nil {
		t.Fatal(err)
	}
	if got := state().Phases[0].HalfByes["a"]; !slices.Equal(got, []int{2}) {
		t.Fatalf("a's half-byes %v, want [2]", got)
	}
	_, err := svc.GiveHalfBye(ctx, tid, "a", 0)
	refused(t, err, "a half-bye without a round")
	if _, err := svc.CancelHalfBye(ctx, tid, "a", 2); err != nil {
		t.Fatal(err)
	}
	if got := state().Phases[0].HalfByes["a"]; len(got) != 0 {
		t.Errorf("a's half-bye was withdrawn, still %v", got)
	}
}

// TestReplaceAndSwapBeforeAnyMatch: an entry nobody has played against is replaced or moved.
func TestReplaceAndSwapBeforeAnyMatch(t *testing.T) {
	ctx, svc, state, tid := directed(t, `{"name":"Club","tables":{"count":4},"phases":[{"kind":"swiss_rounds","length":5,"rounds":3}]}`)
	if _, err := svc.ReplaceParticipant(ctx, tid, "d", "e", "Joueur e", "Lyon", 0); err != nil {
		t.Fatal(err)
	}
	s := state()
	if _, still := s.Players["d"]; still {
		t.Error("d was replaced, still entered")
	}
	if p := s.Players["e"]; p == nil || p.Club != "Lyon" {
		t.Errorf("e took d's place: %+v", p)
	}
	_, err := svc.ReplaceParticipant(ctx, tid, "zz", "", "Personne", "", 0)
	refused(t, err, "replacing an unknown entry")
	_, err = svc.ReplaceParticipant(ctx, tid, "a", "b", "Joueur b", "", 0)
	refused(t, err, "a newcomer under a taken id")

	order := slices.Clone(state().Order)
	if _, err := svc.SwapParticipants(ctx, tid, "a", "b"); err != nil {
		t.Fatal(err)
	}
	after := state().Order
	ia, ib := slices.Index(order, "a"), slices.Index(order, "b")
	if after[ia] != "b" || after[ib] != "a" {
		t.Errorf("order %v after swapping a and b, was %v", after, order)
	}
}

// TestFinalBarrageIsAccepted: the pools' tie-break barrage of 0.9 is a configuration, served as
// the engine reads it.
func TestFinalBarrageIsAccepted(t *testing.T) {
	directed(t, `{"name":"Club","tables":{"count":4},"phases":[{"kind":"round_robin","length":5,"group_size":4,"qualifiers":1,"final_barrage":true,"tiebreak":["point_diff"]}]}`)
}
