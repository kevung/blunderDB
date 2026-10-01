package service

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// pageWatch records, for every Rencontre read made while a page is written, whether the
// room's lock was held at that moment; and how many wall pages were started.
type pageWatch struct {
	storage.Storage
	room *sync.RWMutex

	mu         sync.Mutex
	wallWrites int
	underLock  int
	watchPages bool
}

func (w *pageWatch) Rencontres() storage.RencontreStore {
	return &watchedRencontres{RencontreStore: w.Storage.Rencontres(), w: w}
}

type watchedRencontres struct {
	storage.RencontreStore
	w *pageWatch
}

func (r *watchedRencontres) Get(ctx context.Context, scope string, id int64) (*domain.Rencontre, error) {
	r.w.observe()
	return r.RencontreStore.Get(ctx, scope, id)
}

func (w *pageWatch) observe() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.watchPages {
		return
	}
	pcs := make([]uintptr, 64)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(3, pcs)])
	var inWall, inHTML, inPage bool
	for {
		f, more := frames.Next()
		switch {
		case strings.HasSuffix(f.Function, ".(*Service).WriteRencontrePage"):
			inWall, inPage = true, true
		case strings.HasSuffix(f.Function, ".(*Service).RencontrePageHTML"):
			inHTML = true
		case strings.HasSuffix(f.Function, ".(*Service).writeOwnPage"),
			strings.HasSuffix(f.Function, ".(*Service).WriteDirectionPage"):
			inPage = true
		}
		if !more {
			break
		}
	}
	if inWall && !inHTML {
		w.wallWrites++
	}
	if inPage {
		if w.room.TryLock() {
			w.room.Unlock()
		} else {
			w.underLock++
		}
	}
}

// A swap between two events of a room rewrites both display pages and the wall page once, and
// none of them under the room's lock: writing files is not part of the gesture.
func TestSwapWritesItsPagesOnceAndOutsideTheLock(t *testing.T) {
	ctx := context.Background()
	raw, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "d.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	mem := &Memory{}
	room, _ := mem.gestureLocks("", 0)
	w := &pageWatch{Storage: raw, room: room}
	svc := New(w, "", mem)

	r, err := svc.CreateRencontre(ctx, "Festival", "", "", 8)
	if err != nil {
		t.Fatal(err)
	}
	cfg := tournoi.Config{Name: "Open", Tables: tournoi.Tables{Count: 8},
		Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissLives, Length: 7, Target: 16}}}
	var tids []int64
	for _, name := range []string{"Principal", "Speed"} {
		tid, err := raw.Tournaments().Create(ctx, "", name, "2026-09-12", "")
		if err != nil {
			t.Fatal(err)
		}
		dir, err := direction.Create(ctx, svc.dirStore(), tid, cfg, 7, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{"a", "b", "c", "d"} {
			if err := dir.Enter(ctx, tournoi.Player{ID: tournoi.PlayerID(id), Name: id}, time.Now()); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := svc.AttachToRencontre(ctx, tid, r.ID); err != nil {
			t.Fatal(err)
		}
		tids = append(tids, tid)
	}
	if _, err := svc.SetRencontreOutputDir(ctx, r.ID, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	va, err := svc.StartMatchManually(ctx, tids[0], "a", "b", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartMatchManually(ctx, tids[1], "c", "d", 0, 2); err != nil {
		t.Fatal(err)
	}

	w.mu.Lock()
	w.watchPages = true
	w.mu.Unlock()
	if _, err := svc.MoveMatchToTable(ctx, tids[0], string(va.Running[0].ID), 2); err != nil {
		t.Fatal(err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.wallWrites != 1 {
		t.Errorf("the swap wrote the wall page %d times, want once", w.wallWrites)
	}
	if w.underLock != 0 {
		t.Errorf("%d page reads ran under the room's lock, want none", w.underLock)
	}
}
