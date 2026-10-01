package service_test

import (
	"context"
	"os"
	"path/filepath"
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

// TestWallPageReplaysEachEventOnce: regenerating a room's wall page replays every member's log
// once — the Hall grid and the brackets are read from the same replay, not one each.
func TestWallPageReplaysEachEventOnce(t *testing.T) {
	ctx := context.Background()
	raw, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "d.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	counter := &replayCounter{Storage: raw, loads: map[int64]int{}}
	svc := service.New(counter, "", &service.Memory{})

	room, err := svc.CreateRencontre(ctx, "Open", "2026-09-12", "2026-09-13", 4)
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open","tables":{"count":4},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	var tids []int64
	for _, name := range []string{"Principal", "Speed"} {
		tid, err := raw.Tournaments().Create(ctx, "", name, "2026-09-12", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := svc.CreateDirection(ctx, tid, cfg, 7); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AttachToRencontre(ctx, tid, room.ID); err != nil {
			t.Fatal(err)
		}
		tids = append(tids, tid)
	}

	counter.mu.Lock()
	counter.loads = map[int64]int{}
	counter.mu.Unlock()
	page, err := svc.RencontrePageHTML(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	// The golden file was produced before the grid and the brackets shared one replay: sharing
	// it must not change a byte of the page.
	want, err := os.ReadFile("testdata/rencontre_wall_page.html")
	if err != nil {
		t.Fatal(err)
	}
	if page != string(want) {
		t.Errorf("wall page differs from testdata/rencontre_wall_page.html:\n%s", page)
	}
	counter.mu.Lock()
	defer counter.mu.Unlock()
	for _, tid := range tids {
		if got := counter.loads[tid]; got != 1 {
			t.Errorf("tournament %d replayed %d times for one wall page, want 1", tid, got)
		}
	}
}
