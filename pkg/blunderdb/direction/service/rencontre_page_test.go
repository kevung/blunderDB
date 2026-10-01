package service_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// replayCounter counts the logs read per Tournament: a replay is a log read.
type replayCounter struct {
	storage.Storage
	mu    sync.Mutex
	loads map[int64]int
}

func (c *replayCounter) Directions() storage.DirectionStore {
	return &countingDirections{DirectionStore: c.Storage.Directions(), c: c}
}

type countingDirections struct {
	storage.DirectionStore
	c *replayCounter
}

func (d *countingDirections) LoadEvents(ctx context.Context, scope string, tid int64) ([]direction.StoredEvent, error) {
	d.c.mu.Lock()
	d.c.loads[tid]++
	d.c.mu.Unlock()
	return d.DirectionStore.LoadEvents(ctx, scope, tid)
}

// updateWall rewrites the reference wall page from the code under test. The reference was
// produced by the code of main, before this branch made the grid and the brackets share one
// replay: rerun it there, never here, when the scenario changes.
var updateWall = flag.Bool("update-wall", false, "rewrite testdata/rencontre_wall_page.html")

// rotationDelay is the only part of the wall page that depends on the clock: where each view
// of the rotation starts.
var rotationDelay = regexp.MustCompile(`animation-delay:-?\d+s`)

// busyRoom fills a room of eight tables with two events: a swiss event whose four matches run on
// tables 1 to 4, and a knockout event whose quarter-finals run on tables 5 to 8, one of them
// already won.
func busyRoom(t *testing.T, ctx context.Context, svc *service.Service, raw storage.Storage) (int64, []int64) {
	t.Helper()
	room, err := svc.CreateRencontre(ctx, "Festival", "2026-09-12", "2026-09-13", 8)
	if err != nil {
		t.Fatal(err)
	}
	events := []struct{ name, phase string }{
		{"Principal", `{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}`},
		{"Speed", `{"kind":"bracket","length":5}`},
	}
	var tids []int64
	for _, e := range events {
		tid, err := raw.Tournaments().Create(ctx, "", e.name, "2026-09-12", "")
		if err != nil {
			t.Fatal(err)
		}
		cfg := fmt.Sprintf(`{"name":%q,"tables":{"count":8},"phases":[%s]}`, e.name, e.phase)
		if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
			t.Fatal(err)
		}
		for i := range 8 {
			if _, err := svc.AddParticipant(ctx, tid, fmt.Sprintf("%s %d", e.name, i+1), "", float64(1600-10*i)); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := svc.AttachToRencontre(ctx, tid, room.ID); err != nil {
			t.Fatal(err)
		}
		tids = append(tids, tid)
	}
	if _, err := svc.ConfirmAllProposals(ctx, tids[0]); err != nil {
		t.Fatal(err)
	}
	// The knockout's draw, then its quarter-finals, are confirmed one by one, as the director
	// does from the queue.
	var v *service.DirectionView
	for range 3 {
		var err error
		if v, err = svc.GetDirection(ctx, tids[1]); err != nil {
			t.Fatal(err)
		}
		if len(v.Running) == 4 || len(v.Proposals) == 0 {
			break
		}
		blob, err := json.Marshal(v.Proposals[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.ConfirmProposal(ctx, tids[1], string(blob)); err != nil {
			t.Fatal(err)
		}
		if v, err = svc.ConfirmAllProposals(ctx, tids[1]); err != nil {
			t.Fatal(err)
		}
	}
	if len(v.Running) != 4 {
		t.Fatalf("the knockout event runs %d quarter-finals, want 4", len(v.Running))
	}
	m := v.Running[0]
	if _, err := svc.EnterResult(ctx, tids[1], string(m.ID), string(m.A), 5, 2, ""); err != nil {
		t.Fatal(err)
	}
	return room.ID, tids
}

// TestWallPageReplaysEachEventOnce: regenerating a room's wall page replays every member's log
// once — the Hall grid and the brackets are read from the same replay, not one each — and the
// page is the one main renders for the same room, byte for byte but the rotation's clock.
func TestWallPageReplaysEachEventOnce(t *testing.T) {
	ctx := context.Background()
	raw, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "d.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	counter := &replayCounter{Storage: raw, loads: map[int64]int{}}
	svc := service.New(counter, "", &service.Memory{})
	rid, tids := busyRoom(t, ctx, svc, counter)

	counter.mu.Lock()
	counter.loads = map[int64]int{}
	counter.mu.Unlock()
	page, err := svc.RencontrePageHTML(ctx, rid)
	if err != nil {
		t.Fatal(err)
	}
	page = rotationDelay.ReplaceAllString(page, "animation-delay:0s")
	const golden = "testdata/rencontre_wall_page.html"
	if *updateWall {
		if err := os.WriteFile(golden, []byte(page), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if page != string(want) {
		t.Errorf("wall page differs from %s:\n%s", golden, page)
	}
	counter.mu.Lock()
	defer counter.mu.Unlock()
	for _, tid := range tids {
		if got := counter.loads[tid]; got != 1 {
			t.Errorf("tournament %d replayed %d times for one wall page, want 1", tid, got)
		}
	}
}
