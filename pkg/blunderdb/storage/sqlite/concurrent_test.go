package sqlite_test

// Concurrency tests for the SQLite backend (P5). With the global Database
// mutex gone on the storage path, these exercise the bare *sql.DB pool +
// busy_timeout under parallel readers and writers. They run on a file-backed
// database (t.TempDir) because ":memory:" pins the pool to a single
// connection (each connection is a separate in-memory DB) and so would not
// exercise pool concurrency. Run with -race to catch data races.

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// openTempDB opens a fresh file-backed SQLite Storage that is closed when the
// test ends. File-backed (not ":memory:") so the pool may hold several
// connections concurrently.
func openTempDB(t *testing.T) *sqlite.Storage {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "concurrent.db")
	s, err := sqlite.Open(context.Background(), dsn, nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// posVariant returns a position whose Zobrist hash is unique for each i in
// [0, 64*64*11). InitializePosition leaves PlayerOnRoll = Black (0), so
// NormalizeForStorage does not mirror the board — Score and Cube.Value feed
// the hash verbatim, giving a collision-free encoding of i.
func posVariant(i int) domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score[0] = i & 63
	p.Score[1] = (i >> 6) & 63
	p.Cube.Value = (i >> 12) % 11
	return p
}

// countPositions drains the List iterator and returns the row count. The
// iterator must be fully consumed (a follow-up query nested inside it would
// grab a second pooled connection).
func countPositions(t *testing.T, s storage.Storage) int {
	t.Helper()
	n := 0
	for _, err := range s.Positions().List(context.Background(), "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		n++
	}
	return n
}

// TestConcurrentWrites runs 100 goroutines × 100 inserts of distinct positions
// and asserts the final row count is exactly 10 000 (no lost writes, no
// SQLITE_BUSY).
func TestConcurrentWrites(t *testing.T) {
	s := openTempDB(t)
	ctx := context.Background()

	const goroutines, perG = 100, 100
	var wg sync.WaitGroup
	errs := make(chan error, goroutines*perG)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				p := posVariant(g*perG + j)
				if _, err := s.Positions().Save(ctx, "", &p); err != nil {
					errs <- err
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent Save: %v", err)
	}

	if got := countPositions(t, s); got != goroutines*perG {
		t.Errorf("position count: got %d, want %d", got, goroutines*perG)
	}
}

// TestConcurrentReads runs 100 goroutines listing positions while a writer
// keeps inserting. No deadlock, no error, no panic.
func TestConcurrentReads(t *testing.T) {
	s := openTempDB(t)
	ctx := context.Background()

	// Seed a handful so List has something to stream.
	for i := 0; i < 50; i++ {
		p := posVariant(i)
		if _, err := s.Positions().Save(ctx, "", &p); err != nil {
			t.Fatalf("seed Save: %v", err)
		}
	}

	stop := make(chan struct{})
	var writer sync.WaitGroup
	writer.Add(1)
	go func() {
		defer writer.Done()
		i := 50
		for {
			select {
			case <-stop:
				return
			default:
				p := posVariant(i)
				if _, err := s.Positions().Save(ctx, "", &p); err != nil {
					t.Errorf("background Save: %v", err)
					return
				}
				i++
			}
		}
	}()

	const readers = 100
	var wg sync.WaitGroup
	errs := make(chan error, readers)
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
				if err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(stop)
	writer.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent List: %v", err)
	}
}

// TestMixedWorkload runs readers, writers and a repeated search side by side
// for a bounded duration and asserts no error surfaces.
func TestMixedWorkload(t *testing.T) {
	s := openTempDB(t)
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		p := posVariant(i)
		if _, err := s.Positions().Save(ctx, "", &p); err != nil {
			t.Fatalf("seed Save: %v", err)
		}
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	fail := func(err error) {
		select {
		case errs <- err:
		default:
		}
	}

	// 10 readers.
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				for _, err := range s.Positions().List(ctx, "", storage.ListOpts{Limit: 20}) {
					if err != nil {
						fail(err)
						return
					}
				}
			}
		}()
	}

	// 5 writers, each in its own disjoint id range so hashes stay unique.
	for w := 0; w < 5; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			i := 1000 + w*100000
			for time.Now().Before(deadline) {
				p := posVariant(i)
				if _, err := s.Positions().Save(ctx, "", &p); err != nil {
					fail(err)
					return
				}
				i++
			}
		}(w)
	}

	// 1 long search loop (full scan via Find with empty filters).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for time.Now().Before(deadline) {
			for _, err := range s.Search().Find(ctx, "", domain.SearchFilters{}) {
				if err != nil {
					fail(err)
					return
				}
			}
		}
	}()

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("mixed workload: %v", err)
	}
}
