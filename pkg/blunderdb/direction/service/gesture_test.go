package service_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

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
