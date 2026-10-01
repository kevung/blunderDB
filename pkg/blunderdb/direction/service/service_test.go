package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestServiceRunsOnTheStorageContract: a Direction is created, played and filled through the
// storage contract alone, with no desktop wrapper — what the serve daemon will call.
func TestServiceRunsOnTheStorageContract(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := service.New(st, "", nil)

	tid, err := st.Tournaments().Create(ctx, "", "Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":4},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatalf("CreateDirection: %v", err)
	}
	var players []string
	for _, id := range []string{"aa", "bb", "cc", "dd"} {
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := svc.EnterParticipants(ctx, tid, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatalf("EnterParticipants: %v", err)
	}
	v, err := svc.ConfirmAllProposals(ctx, tid)
	if err != nil || len(v.Running) == 0 {
		t.Fatalf("ConfirmAllProposals = %v; want matches running", err)
	}
	m := v.Running[0]
	if _, err := svc.EnterResult(ctx, tid, string(m.ID), string(m.A), 0, 0, ""); err != nil {
		t.Fatalf("EnterResult: %v", err)
	}

	match := domain.Match{Player1Name: "Joueur " + string(m.A), Player2Name: "Joueur " + string(m.B), MatchLength: 7}
	mid, err := st.Matches().Save(ctx, "", &match)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AttachMatchToSlot(ctx, tid, string(m.ID), mid); err != nil {
		t.Fatalf("AttachMatchToSlot: %v", err)
	}
	slot, err := svc.SlotOfMatch(ctx, mid)
	if err != nil || slot == nil || slot.SlotID != string(m.ID) || slot.TournamentName != "Open de Lyon" {
		t.Fatalf("SlotOfMatch = %+v, %v", slot, err)
	}
	h, err := svc.SlotHeader(ctx, tid, string(m.ID))
	if err != nil || h.Event != "Open de Lyon" || h.MatchLength != 7 || h.TournamentID == nil || *h.TournamentID != tid {
		t.Fatalf("SlotHeader = %+v, %v", h, err)
	}
	if err := svc.DeleteDirection(ctx, tid); err != nil {
		t.Fatalf("DeleteDirection: %v", err)
	}
	if _, slotID, err := st.Directions().SlotOf(ctx, "", mid); err != nil || slotID != "" {
		t.Errorf("the Slot survives its Direction: %q, %v", slotID, err)
	}
}
