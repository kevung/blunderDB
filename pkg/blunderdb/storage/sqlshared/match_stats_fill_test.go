package sqlshared_test

import (
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// copyDemoDB writes the demo library to a file of the test's own.
func copyDemoDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "demo.db")
	src, err := os.Open(filepath.Join("..", "..", "..", "..", "internal", "gui", "demo.db.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	zr, err := gzip.NewReader(src)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, zr); err != nil {
		t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestColdStatsReadersFillOnce: readers arriving together on a library
// whose match_stats is empty all answer, and the table is filled by one of
// them while the others wait — none loses the write lock and fails.
func TestColdStatsReadersFillOnce(t *testing.T) {
	ctx := context.Background()
	path := copyDemoDB(t)
	raw, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`DELETE FROM match_stats`); err != nil {
		t.Fatal(err)
	}
	var matches int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&matches); err != nil {
		t.Fatal(err)
	}
	raw.Close()
	if matches == 0 {
		t.Fatal("the demo library holds no match")
	}

	st, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	var fills atomic.Int32
	defer sqlshared.SetBeforeMatchStatsFill(func(int) { fills.Add(1) })()

	const readers = 12
	f := storage.StatsFilter{DecisionType: -1}
	start := make(chan struct{})
	errs := make(chan error, readers)
	var wg sync.WaitGroup
	for i := range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			var err error
			switch i % 3 {
			case 0:
				_, err = st.Stats().Compute(ctx, "", f)
			case 1:
				_, err = st.Stats().MatchSeries(ctx, "", f)
			default:
				_, err = st.Stats().FillMatchStats(ctx, "", nil)
			}
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("reader: %v", err)
		}
	}
	if n := fills.Load(); n != 1 {
		t.Errorf("fills started = %d, want 1", n)
	}
	missing, err := st.Stats().MissingMatchStats(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Errorf("%d match(es) still lack their rows", len(missing))
	}
}

// TestColdStatsFillOutlivesItsLeader: the reader whose fill others wait for
// leaves (its request is cancelled); a waiter then runs the fill itself
// instead of returning the leader's cancellation.
func TestColdStatsFillOutlivesItsLeader(t *testing.T) {
	ctx := context.Background()
	path := copyDemoDB(t)
	raw, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`DELETE FROM match_stats`); err != nil {
		t.Fatal(err)
	}
	raw.Close()
	st, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	leaderIn, proceed := make(chan struct{}), make(chan struct{})
	var fills atomic.Int32
	defer sqlshared.SetBeforeMatchStatsFill(func(int) {
		if fills.Add(1) == 1 {
			close(leaderIn)
			<-proceed
		}
	})()

	leaderCtx, cancel := context.WithCancel(ctx)
	leaderErr := make(chan error, 1)
	go func() {
		_, err := st.Stats().FillMatchStats(leaderCtx, "", nil)
		leaderErr <- err
	}()
	<-leaderIn
	waiterErr := make(chan error, 1)
	go func() {
		_, err := st.Stats().FillMatchStats(ctx, "", nil)
		waiterErr <- err
	}()
	time.Sleep(50 * time.Millisecond) // the waiter reaches the gate
	cancel()
	close(proceed)
	if err := <-leaderErr; !errors.Is(err, context.Canceled) {
		t.Errorf("leader: %v, want context.Canceled", err)
	}
	if err := <-waiterErr; err != nil {
		t.Errorf("waiter: %v", err)
	}
	if n := fills.Load(); n != 2 {
		t.Errorf("fills started = %d, want 2 (the cut one, then the waiter's)", n)
	}
	missing, err := st.Stats().MissingMatchStats(ctx, "")
	if err != nil || len(missing) != 0 {
		t.Errorf("after the waiter's fill: %d missing, err %v", len(missing), err)
	}
}
