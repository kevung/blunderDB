package database

import (
	"context"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Le lot parallèle (#147) : les positions d'un lot sont indépendantes, donc
// les répartir sur plusieurs goroutines ne peut pas changer ce qui est écrit.
// C'est ce que ces tests vérifient — pas « à peu près », au bit près : chaque
// goroutine possède son propre Searcher réutilisé d'une position à l'autre,
// cache compris, et un cache chaud ne doit déplacer aucun bit (cache.go).

// seedBatchPositions writes n distinct, analysable positions and returns
// their ids in insertion order — the same ids in every database seeded this
// way, so two runs are directly comparable.
func seedBatchPositions(t *testing.T, d *Database, n int) []int64 {
	t.Helper()
	var ids []int64
	for i := 0; i < n; i++ {
		pos := racePosition(1+i, 13+i, domain.White)
		if i%2 == 1 {
			// Half the batch asks a checker question and half a cube one, so
			// both branches of EvaluatePositionWith are compared.
			pos.Dice = [2]int{3, 1}
		}
		id, err := d.SavePosition(&pos)
		if err != nil {
			t.Fatalf("SavePosition %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	return ids
}

// analysesOf loads every analysis, keyed by position id — the whole stored
// verdict, floats included, for an exact comparison. The two write timestamps
// are zeroed: they record WHEN a row was written, which two runs of the same
// batch never share and which no parallelism argument is about.
func analysesOf(t *testing.T, d *Database, ids []int64) map[int64]*PositionAnalysis {
	t.Helper()
	out := make(map[int64]*PositionAnalysis, len(ids))
	for _, id := range ids {
		a, err := d.LoadAnalysis(id)
		if err != nil {
			t.Fatalf("LoadAnalysis(%d): %v", id, err)
		}
		if a != nil {
			a.CreationDate = time.Time{}
			a.LastModifiedDate = time.Time{}
		}
		out[id] = a
	}
	return out
}

// TestAnalyzeGammonNetParallelMatchesSerial is the central test of #147: the
// same batch, run on 1, 2 and NumCPU goroutines, writes exactly the same
// analyses. Not "within a tolerance" — reflect.DeepEqual over the stored
// structs, so every float32 that reaches the database is compared bit for
// bit.
func TestAnalyzeGammonNetParallelMatchesSerial(t *testing.T) {
	t.Parallel()
	const n = 8

	run := func(jobs int) map[int64]*PositionAnalysis {
		d := newBatchTestDB(t)
		ids := seedBatchPositions(t, d, n)
		if _, err := d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, jobs, nil, nil); err != nil {
			t.Fatalf("AnalyzeMissingWithGammonNet(jobs=%d): %v", jobs, err)
		}
		if missing, err := d.CountPositionsWithoutAnalysis(); err != nil || missing != 0 {
			t.Fatalf("after jobs=%d: %d position(s) still without analysis (err=%v)", jobs, missing, err)
		}
		return analysesOf(t, d, ids)
	}

	want := run(1)
	for _, jobs := range []int{2, runtime.NumCPU()} {
		got := run(jobs)
		if !reflect.DeepEqual(want, got) {
			t.Errorf("jobs=%d wrote different analyses than the serial batch", jobs)
		}
	}
}

// TestAnalyzeGammonNetProgressIsMonotone: with several goroutines the loop
// index is meaningless, so progress comes from a counter the single writer
// owns. It must never go backwards, never exceed the total, and must end on
// the total when nothing failed.
func TestAnalyzeGammonNetProgressIsMonotone(t *testing.T) {
	t.Parallel()
	const n = 8
	d := newBatchTestDB(t)
	seedBatchPositions(t, d, n)

	var last int
	var calls int
	summary, err := d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, runtime.NumCPU(), nil, func(done, total int) {
		calls++
		if done <= last {
			t.Errorf("progress went from %d to %d: not monotone", last, done)
		}
		if done > total || total != n {
			t.Errorf("progress %d/%d, want a running count out of %d", done, total, n)
		}
		last = done
	})
	if err != nil {
		t.Fatalf("AnalyzeMissingWithGammonNet: %v", err)
	}
	if calls != n || last != n {
		t.Errorf("progress ended at %d after %d call(s), want %d/%d", last, calls, n, n)
	}
	if got := summary.Processed(); got != n {
		t.Errorf("summary.Processed() = %d, want %d", got, n)
	}
}

// TestAnalyzeGammonNetParallelCancellation: an already-cancelled context
// writes nothing at all, and a context cancelled part-way leaves the run
// strictly unfinished — nothing after the cancellation, nothing lost before
// it (whatever was computed is still written, and re-running picks up the
// rest).
func TestAnalyzeGammonNetParallelCancellation(t *testing.T) {
	t.Parallel()
	const n = 12

	t.Run("cancelled before it starts", func(t *testing.T) {
		d := newBatchTestDB(t)
		seedBatchPositions(t, d, n)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := d.AnalyzeMissingWithGammonNet(ctx, 0, 0, 0, 4, nil, nil); err == nil {
			t.Fatal("expected context.Canceled")
		}
		if missing, _ := d.CountPositionsWithoutAnalysis(); missing != n {
			t.Errorf("%d position(s) missing after a cancelled-before-start run, want %d", missing, n)
		}
	})

	t.Run("cancelled part-way", func(t *testing.T) {
		d := newBatchTestDB(t)
		seedBatchPositions(t, d, n)

		ctx, cancel := context.WithCancel(context.Background())
		var yields atomic.Int64
		_, err := d.AnalyzeMissingWithGammonNet(ctx, 0, 0, 0, 2, func() {
			if yields.Add(1) == 3 {
				cancel()
			}
		}, nil)
		if err == nil {
			t.Fatal("expected context.Canceled from the aborted run")
		}

		missing, err := d.CountPositionsWithoutAnalysis()
		if err != nil {
			t.Fatalf("CountPositionsWithoutAnalysis: %v", err)
		}
		if missing == 0 || missing == n {
			t.Fatalf("missing after cancel = %d, want strictly between 0 and %d", missing, n)
		}

		// Resume finishes the rest: nothing was lost, nothing was written twice.
		if _, err := d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, 4, nil, nil); err != nil {
			t.Fatalf("resume: %v", err)
		}
		if missing, _ := d.CountPositionsWithoutAnalysis(); missing != 0 {
			t.Errorf("missing after resume = %d, want 0", missing)
		}
	})
}

// TestAnalyzeGammonNetParallelYieldGates is the parallel half of the
// preemption criterion: every goroutine calls yield before taking a
// position, so a yield that never returns stalls the WHOLE batch, not just
// one goroutine. That is what internal/gui relies on to let an interactive
// evaluation through.
func TestAnalyzeGammonNetParallelYieldGates(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)
	seedBatchPositions(t, d, 8)

	release := make(chan struct{})
	// `blocked` dit que le premier yield est ENTRÉ, donc que le lot est
	// effectivement arrêté à sa grille — un fait, là où un time.Sleep n'était
	// qu'un pari sur la vitesse de la machine (E.3, #219).
	blocked := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		_, err := d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, 4, func() {
			select {
			case blocked <- struct{}{}:
			default:
			}
			<-release
		}, nil)
		done <- err
	}()

	<-blocked
	if missing, err := d.CountPositionsWithoutAnalysis(); err != nil || missing != 8 {
		t.Fatalf("%d position(s) missing while every yield was blocked (err=%v), want 8: the batch did not yield", missing, err)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("AnalyzeMissingWithGammonNet: %v", err)
	}
	if missing, _ := d.CountPositionsWithoutAnalysis(); missing != 0 {
		t.Errorf("missing after release = %d, want 0", missing)
	}
}
