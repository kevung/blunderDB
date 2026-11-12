package service_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	tournoi "github.com/kevung/nicomaque"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestSimultaneousGesturesAllLand: gestures arriving at once on one Direction — two windows,
// two requests — are each recorded, none lost to a busy database or a taken sequence number.
func TestSimultaneousGesturesAllLand(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "d.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	mem := &service.Memory{}
	tid, err := st.Tournaments().Create(ctx, "", "Open", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := service.New(st, "", mem).CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	before, err := st.Directions().LoadEvents(ctx, "", tid)
	if err != nil {
		t.Fatal(err)
	}

	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// One service per call, over the one Memory: what the façade does.
			if _, err := service.New(st, "", mem).AddDirectionNote(ctx, tid, fmt.Sprintf("note %d", i)); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("a simultaneous gesture failed: %v", err)
	}
	after, err := st.Directions().LoadEvents(ctx, "", tid)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(after) - len(before); got != n {
		t.Errorf("%d gestures recorded, want %d", got, n)
	}
}

// TestSlotShowsItsMostRecentDraft: when two drafts were started from one Slot, the Slot leads
// to the one typed last.
func TestSlotShowsItsMostRecentDraft(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := service.New(st, "", nil)
	tid, err := st.Tournaments().Create(ctx, "", "Open", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnterParticipants(ctx, tid, `[{"id":"aa","name":"A"},{"id":"bb","name":"B"}]`); err != nil {
		t.Fatal(err)
	}
	v, err := svc.ConfirmAllProposals(ctx, tid)
	if err != nil || len(v.Running) == 0 {
		t.Fatalf("ConfirmAllProposals = %v", err)
	}
	slot := string(v.Running[0].ID)
	doc := fmt.Sprintf(`{"header":{"round":"Ronde #%s","tournament_id":%d}}`, slot, tid)
	var newest int64
	for range 2 {
		newest, err = st.Transcriptions().Save(ctx, "", &storage.Transcription{Document: doc})
		if err != nil {
			t.Fatal(err)
		}
	}
	rows, err := svc.Slots(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.SlotID == slot && r.DraftID != newest {
			t.Errorf("slot %s leads to draft %d, want the newest %d", slot, r.DraftID, newest)
		}
	}
}

// TestEntryIDIsNotTheEnginesOwn: an id the caller gives is refused when it is the engine's
// sentinel for an exemption, or too long to be an identifier — by both ways of entering, and
// a refused list enters nobody.
func TestEntryIDIsNotTheEnginesOwn(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := service.New(st, "", nil)
	tid, err := st.Tournaments().Create(ctx, "", "Open", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open","tables":{"count":4},"phases":[{"kind":"bracket","length":7}]}`
	if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("x", service.MaxEntryIDLength+1)
	for id, want := range map[string]string{string(tournoi.BYE): "stands for an exemption", " BYE ": "stands for an exemption", long: "at most 64 bytes, not 65"} {
		if _, err := svc.AddParticipantAs(ctx, tid, id, "Zoé", "", 0); !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), want) {
			t.Errorf("addParticipant with id %.20q: %v; want the refusal %q", id, err, want)
		}
		list := `[{"id":"aa","name":"A"},{"id":` + strconv.Quote(id) + `,"name":"Zoé"}]`
		if err := svc.EnterParticipants(ctx, tid, list); !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), want) {
			t.Errorf("enterParticipants with id %.20q: %v; want the refusal %q", id, err, want)
		}
	}
	v, err := svc.GetDirection(ctx, tid)
	if err != nil || len(v.Players) != 0 {
		t.Fatalf("after the refusals: %d entries (%v); want none, a refused list enters nobody", len(v.Players), err)
	}
	exact := strings.Repeat("x", service.MaxEntryIDLength)
	if _, err := svc.AddParticipantAs(ctx, tid, exact, "Zoé", "", 0); err != nil {
		t.Errorf("an id of exactly %d bytes: %v; want it entered", service.MaxEntryIDLength, err)
	}
}
