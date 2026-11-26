package gui

import (
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

const mapsOpeningXGID = "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10"

// mapEvents captures what a map emits, without a window.
func mapEvents(a *App) <-chan event {
	ch := make(chan event, 256)
	hook := func(name string, payload ...any) {
		var p any
		if len(payload) > 0 {
			p = payload[0]
		}
		ch <- event{name, p}
	}
	a.emitHook.Store(&hook)
	return ch
}

// waitMapEnd drains ch until an event whose name is one of ends, counting the
// others by name.
func waitMapEnd(t *testing.T, ch <-chan event, ends ...string) (event, map[string]int) {
	t.Helper()
	seen := map[string]int{}
	deadline := time.After(60 * time.Second)
	for {
		select {
		case e := <-ch:
			for _, end := range ends {
				if e.name == end {
					return e, seen
				}
			}
			seen[e.name]++
		case <-deadline:
			t.Fatalf("no end event; saw %v", seen)
		}
	}
}

func TestStartRollMap_EmitsRowsThenDone(t *testing.T) {
	a := &App{}
	ch := mapEvents(a)
	pos, err := domain.DecodeXGID(mapsOpeningXGID)
	if err != nil {
		t.Fatal(err)
	}
	a.StartRollMap(pos, 1, 0, 7)
	last, seen := waitMapEnd(t, ch, "gammonnet-rolls:done", "gammonnet-rolls:error", "gammonnet-rolls:cancelled")
	if last.name != "gammonnet-rolls:done" {
		t.Fatalf("ended with %s %v", last.name, last.payload)
	}
	if done := last.payload.(RollMapDoneEvent); done.Tag != 7 || len(done.Map.Rows) != 21 || done.Map.Ply != 1 {
		t.Fatalf("done = %+v", done)
	}
	if seen["gammonnet-rolls:row"] != 42 {
		t.Fatalf("rows emitted = %d, want 21 at 0-ply and 21 at 1-ply", seen["gammonnet-rolls:row"])
	}
	a.gnEvalMu.Lock()
	busy := a.gnEvalCancel != nil
	a.gnEvalMu.Unlock()
	if busy {
		t.Fatal("the interactive slot is still held after done")
	}
}

func TestStartRollMap_SupersededIsCancelled(t *testing.T) {
	a := &App{}
	ch := mapEvents(a)
	pos, _ := domain.DecodeXGID(mapsOpeningXGID)
	a.StartRollMap(pos, 2, 0, 1)
	a.CancelRollMap()
	last, _ := waitMapEnd(t, ch, "gammonnet-rolls:done", "gammonnet-rolls:error", "gammonnet-rolls:cancelled")
	if last.name != "gammonnet-rolls:cancelled" {
		t.Fatalf("ended with %s", last.name)
	}
}

func TestStartRollMapAfterPlay_RefusesAnIllegalPlay(t *testing.T) {
	a := &App{}
	ch := mapEvents(a)
	pos, _ := domain.DecodeXGID(mapsOpeningXGID)
	a.StartRollMapAfterPlay(pos, 3, 1, "24/18", 0, 0, 3)
	last, _ := waitMapEnd(t, ch, "gammonnet-rolls:done", "gammonnet-rolls:error", "gammonnet-rolls:cancelled")
	if last.name != "gammonnet-rolls:error" {
		t.Fatalf("ended with %s", last.name)
	}
}

func TestStartScoreMap_EmitsCellsThenDone(t *testing.T) {
	a := &App{}
	ch := mapEvents(a)
	pos, err := domain.DecodeXGID("XGID=-b----E-C---eE---c-e----B-:0:0:1:31:6:4:0:9:10")
	if err != nil {
		t.Fatal(err)
	}
	a.StartScoreMap(pos, 3, 1, 0, 5)
	last, seen := waitMapEnd(t, ch, "gammonnet-scoremap:done", "gammonnet-scoremap:error", "gammonnet-scoremap:cancelled")
	if last.name != "gammonnet-scoremap:done" {
		t.Fatalf("ended with %s %v", last.name, last.payload)
	}
	if done := last.payload.(ScoreMapDoneEvent); done.Tag != 5 || len(done.Map.Cells) != 9 || done.Map.Decision != "checker" {
		t.Fatalf("done = %+v", done)
	}
	if seen["gammonnet-scoremap:cell"] != 18 {
		t.Fatalf("cells emitted = %d, want 9 at 0-ply and 9 at 1-ply", seen["gammonnet-scoremap:cell"])
	}
}

func TestStartScoreMap_CancelLeavesThePanelAlone(t *testing.T) {
	a := &App{}
	ch := mapEvents(a)
	pos, _ := domain.DecodeXGID(mapsOpeningXGID)
	a.StartScoreMap(pos, 15, 2, 0, 2)
	a.CancelScoreMap()
	last, _ := waitMapEnd(t, ch, "gammonnet-scoremap:done", "gammonnet-scoremap:error", "gammonnet-scoremap:cancelled")
	if last.name != "gammonnet-scoremap:cancelled" {
		t.Fatalf("ended with %s", last.name)
	}
	// A later search is not the map's: its cancel must not reach it.
	ctx, end := a.beginInteractiveSearch(nil)
	a.CancelScoreMap()
	if ctx.Err() != nil {
		t.Fatal("CancelScoreMap cancelled a search it did not start")
	}
	end()
}

// Two Starts racing a Cancel: the sequence is recorded under the slot's
// lock, so -race sees no unguarded access and both maps end cancelled.
func TestStartRollMap_ConcurrentStartsThenCancel(t *testing.T) {
	a := &App{}
	ch := mapEvents(a)
	pos, _ := domain.DecodeXGID(mapsOpeningXGID)
	var wg sync.WaitGroup
	for tag := 1; tag <= 2; tag++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.StartRollMap(pos, 2, 0, tag)
		}()
	}
	go a.CancelRollMap()
	wg.Wait()
	a.CancelRollMap()
	ended := map[int]string{}
	for len(ended) < 2 {
		e, _ := waitMapEnd(t, ch, "gammonnet-rolls:done", "gammonnet-rolls:error", "gammonnet-rolls:cancelled")
		ended[e.payload.(map[string]int)["tag"]] = e.name
	}
	for tag, name := range ended {
		if name != "gammonnet-rolls:cancelled" {
			t.Errorf("tag %d ended with %s", tag, name)
		}
	}
}

func TestStartMaps_RefuseTheBoundsTheCLIRefuses(t *testing.T) {
	a := &App{}
	ch := mapEvents(a)
	pos, _ := domain.DecodeXGID(mapsOpeningXGID)
	a.StartRollMap(pos, 3, 0, 1)
	if e, _ := waitMapEnd(t, ch, "gammonnet-rolls:done", "gammonnet-rolls:error", "gammonnet-rolls:cancelled"); e.name != "gammonnet-rolls:error" {
		t.Fatalf("roll map at 3 ply: %s", e.name)
	}
	for _, c := range []struct{ length, ply int }{{8, 0}, {17, 0}, {9, 3}} {
		a.StartScoreMap(pos, c.length, c.ply, 0, 1)
		if e, _ := waitMapEnd(t, ch, "gammonnet-scoremap:done", "gammonnet-scoremap:error", "gammonnet-scoremap:cancelled"); e.name != "gammonnet-scoremap:error" {
			t.Fatalf("score map %+v: %s", c, e.name)
		}
	}
}
