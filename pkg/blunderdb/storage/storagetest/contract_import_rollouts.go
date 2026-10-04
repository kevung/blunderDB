package storagetest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// importRolloutWriters is how many rollouts are stored on the target, and how
// many imports run, at the same moment.
const importRolloutWriters = 6

// rolloutSource writes a native database holding checkerPos with one rollout
// of the given seed, and returns its path: imported, it adds that rollout to
// the target's analysis of the same position.
func rolloutSource(t *testing.T, seed uint64) string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), fmt.Sprintf("source-%d.db", seed))
	src, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer src.Close()
	p := checkerPos()
	id, err := src.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("save source position: %v", err)
	}
	set := rollout.Fast()
	set.Seed = seed
	if err := rollouts.Store(ctx, src, "", id, movesRollout(set, "13/10 6/5", 0.7)); err != nil {
		t.Fatalf("store source rollout: %v", err)
	}
	return path
}

// importsAndRolloutsAllKept runs importRolloutWriters imports (each bringing a
// rollout of its own) beside as many rollouts stored on the target position,
// and checks every one of them is kept.
func importsAndRolloutsAllKept(t *testing.T, s storage.Storage, runImport func(path string) error) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	sources := make([]string, importRolloutWriters)
	for i := range sources {
		sources[i] = rolloutSource(t, uint64(100+i))
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2*importRolloutWriters)
	for i := range importRolloutWriters {
		wg.Add(2)
		go func() {
			defer wg.Done()
			set := rollout.Fast()
			set.Seed = uint64(i + 1)
			errs <- rollouts.Store(ctx, s, "", id, movesRollout(set, "13/10 6/5", 0.8))
		}()
		go func() {
			defer wg.Done()
			errs <- runImport(sources[i])
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent write: %v", err)
		}
	}
	got, err := s.Analyses().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := 2 * importRolloutWriters; len(got.Rollouts) != want {
		t.Errorf("%d rollouts kept, want %d (stored and imported at once)", len(got.Rollouts), want)
	}
}

// testDBImportAndRolloutsAllKept: a native database import merging into an
// analysis while rollouts are stored on it loses neither side.
func testDBImportAndRolloutsAllKept(t *testing.T, s storage.Storage) {
	importsAndRolloutsAllKept(t, s, func(path string) error {
		_, err := ingest.DBImporter{S: s}.Import(context.Background(), "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil)
		return err
	})
}

// testJSONImportAndRolloutsAllKept: the NDJSON import, same rule.
func testJSONImportAndRolloutsAllKept(t *testing.T, s storage.Storage) {
	importsAndRolloutsAllKept(t, s, func(path string) error {
		ctx := context.Background()
		src, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		err = ingest.JSONExporter{S: src}.Export(ctx, "", &buf, ingest.ExportOptions{})
		_ = src.Close()
		if err != nil {
			return err
		}
		_, err = ingest.JSONImporter{S: s}.Import(ctx, "", ingest.Source{Format: ingest.FormatJSON, Reader: &buf}, nil)
		return err
	})
}

// windowWait is how long the rollout written inside the Crawford merge's
// window is given to finish before the merge goes on. A writer the merge
// locks out waits for the merge's commit, so the wait runs out; one it does
// not lock out finishes at once, in the window.
const windowWait = 300 * time.Millisecond

// testRepairCrawfordAndRolloutsAllKept: a rollout written on the stale row
// between the Crawford repair's read of its analysis and the fold either
// waits for the fold — and then finds its position gone — or is carried to
// the twin; never lost. Rollouts stored on the twin meanwhile are kept too.
// The stale-row writer is put in the window by sqlshared's hook, so a repair
// that reads without locking loses it every time.
func testRepairCrawfordAndRolloutsAllKept(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	twinID, staleID := crawfordTwins(t, s)
	first := rollout.Fast()
	first.Seed = 1000
	if err := rollouts.Store(ctx, s, "", staleID, movesRollout(first, "8/5 6/5", 0.5)); err != nil {
		t.Fatalf("Store rollout: %v", err)
	}

	inWindow := rollout.Fast()
	inWindow.Seed = 2000
	windowDone := make(chan error, 1)
	sqlshared.AfterDuplicateAnalysisRead = func(dupID int64) {
		if dupID != staleID {
			return
		}
		go func() {
			windowDone <- rollouts.Store(ctx, s, "", staleID, movesRollout(inWindow, "8/5 6/5", 0.6))
		}()
		select {
		case err := <-windowDone:
			windowDone <- err
		case <-time.After(windowWait):
		}
	}
	t.Cleanup(func() { sqlshared.AfterDuplicateAnalysisRead = nil })

	var wg sync.WaitGroup
	errs := make(chan error, importRolloutWriters+1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := s.Positions().RepairCrawfordSentinel(ctx, ""); err != nil {
			errs <- fmt.Errorf("repair: %w", err)
		}
	}()
	for i := range importRolloutWriters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			set := rollout.Fast()
			set.Seed = uint64(1 + i)
			if err := rollouts.Store(ctx, s, "", twinID, movesRollout(set, "8/5 6/5", 0.6)); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	if err := errors.Join(slices.Collect(chanValues(errs))...); err != nil {
		t.Fatalf("concurrent write: %v", err)
	}
	windowErr := <-windowDone
	want := 1 + importRolloutWriters
	switch {
	case windowErr == nil:
		want++
	case errors.Is(windowErr, storage.ErrNotFound):
		// It waited for the fold and found its position gone.
	default:
		t.Fatalf("rollout written in the repair's window: %v, want success or ErrNotFound", windowErr)
	}

	a, err := s.Analyses().Load(ctx, "", twinID)
	if err != nil {
		t.Fatalf("Load twin analysis: %v", err)
	}
	if len(a.Rollouts) != want {
		t.Errorf("%d rollouts on the twin, want %d (window write: %v)", len(a.Rollouts), want, windowErr)
	}
	if _, err := s.Analyses().Load(ctx, "", staleID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("stale row's analysis after the repair: err = %v, want ErrNotFound", err)
	}
}

// chanValues yields what c holds until it is closed.
func chanValues[T any](c <-chan T) func(func(T) bool) {
	return func(yield func(T) bool) {
		for v := range c {
			if !yield(v) {
				return
			}
		}
	}
}
