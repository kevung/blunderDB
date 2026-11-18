package database

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Scale benchmarks run against a large pre-built database rather than the
// fixtures: the search, stats and listing paths only show their cost at the
// size an imported corpus reaches. Build the base with
//
//	go run ./cmd/blunderdb-synthdb -out /path/scale.db -positions 1000000
//
// then point BLUNDERDB_SCALE_DB at it (relative paths resolve from the repo
// root, where TestMain runs). Without it every BenchmarkScale_* skips.
//
// The base is opened once and shared. BenchmarkScale_ImportSingleXG deletes
// the match it imported, so a rerun measures the same import again instead
// of a duplicate refused in a few milliseconds.

var (
	scaleOnce sync.Once
	scaleDB   *Database
	scaleErr  error
)

func openScaleDB(b *testing.B) *Database {
	b.Helper()
	path := os.Getenv("BLUNDERDB_SCALE_DB")
	if path == "" {
		b.Skip("BLUNDERDB_SCALE_DB not set")
	}
	scaleOnce.Do(func() {
		if _, err := os.Stat(path); err != nil {
			scaleErr = err
			return
		}
		db := NewDatabase()
		if err := db.OpenDatabase(path); err != nil {
			scaleErr = err
			return
		}
		scaleDB = db
	})
	if scaleErr != nil {
		b.Fatalf("scale database %s: %v", path, scaleErr)
	}
	return scaleDB
}

func BenchmarkScale_ImportSingleXG(b *testing.B) {
	db := openScaleDB(b)
	fixture := filepath.Join("testdata", "test.xg")
	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := db.ImportXGMatch(fixture)
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		if err := db.DeleteMatch(id); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func benchScaleSearch(b *testing.B, f SearchFilters) {
	db := openScaleDB(b)
	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		res, err := db.LoadPositionsByFilters(f)
		if err != nil {
			b.Fatal(err)
		}
		n = len(res)
	}
	b.ReportMetric(float64(n), "positions")
}

// Wide: a large share of the base comes back.
func BenchmarkScale_SearchWideCube(b *testing.B) {
	filter := emptyFilter()
	filter.DecisionType = CubeAction
	benchScaleSearch(b, SearchFilters{Filter: filter, DecisionTypeFilter: true})
}

func BenchmarkScale_SearchWideErrorAboveTenth(b *testing.B) {
	benchScaleSearch(b, SearchFilters{Filter: emptyFilter(), MoveErrorFilter: "E>100"})
}

// BenchmarkScale_SearchIDsErrorAboveTenth is the GUI's side of the same
// filter: ids only, the whole result at once.
func BenchmarkScale_SearchIDsErrorAboveTenth(b *testing.B) {
	db := openScaleDB(b)
	f := SearchFilters{Filter: emptyFilter(), MoveErrorFilter: "E>100"}
	b.ReportAllocs()
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		ids, err := db.LoadPositionIDsByFilters(f)
		if err != nil {
			b.Fatal(err)
		}
		n = len(ids)
	}
	b.ReportMetric(float64(n), "positions")
}

// Narrow: a handful of rows out of the whole base.
func BenchmarkScale_SearchNarrowDiceAndScore(b *testing.B) {
	filter := emptyFilter()
	filter.Dice = [2]int{6, 5}
	filter.Score = [2]int{6, 4}
	filter.DecisionType = CheckerAction
	benchScaleSearch(b, SearchFilters{Filter: filter, DiceRollFilter: true, IncludeScore: true})
}

func BenchmarkScale_StatsCompute(b *testing.B) {
	db := openScaleDB(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.ComputeStats(StatsFilter{DecisionType: -1}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScale_ListPositionIDs(b *testing.B) {
	db := openScaleDB(b)
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		ids, err := db.ListPositionIDs(0, 1000)
		if err != nil {
			b.Fatal(err)
		}
		n = len(ids)
	}
	b.ReportMetric(float64(n), "positions")
}

func BenchmarkScale_GetAllMatches(b *testing.B) {
	db := openScaleDB(b)
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		ms, err := db.GetAllMatches()
		if err != nil {
			b.Fatal(err)
		}
		n = len(ms)
	}
	b.ReportMetric(float64(n), "matches")
}

// BenchmarkScale_SearchPageListFields walks the first pages of 24 positions of
// the whole library by cursor, as a table view does, and reports the median
// page (search page and its analyses) without and with its list fields
// (played error, date, collections): the share the list fields add.
func BenchmarkScale_SearchPageListFields(b *testing.B) {
	db := openScaleDB(b)
	const pages = 200
	var bare, full []time.Duration
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cursor := ""
		for p := 0; p < pages; p++ {
			start := time.Now()
			positions, _, next, err := db.LoadPositionsPageByFiltersCtx(context.Background(), domain.SearchFilters{},
				storage.PageOpts{Limit: 24, Cursor: cursor})
			if err != nil {
				b.Fatal(err)
			}
			page := time.Since(start)
			ids := make([]int64, len(positions))
			for j := range positions {
				ids[j] = positions[j].ID
			}
			if _, err := db.PositionListFields(ids); err != nil {
				b.Fatal(err)
			}
			bare, full = append(bare, page), append(full, time.Since(start))
			if next == "" {
				break
			}
			cursor = next
		}
	}
	b.StopTimer()
	median := func(d []time.Duration) float64 {
		slices.Sort(d)
		return float64(d[len(d)/2].Microseconds()) / 1000
	}
	p50, p50Fields := median(bare), median(full)
	b.ReportMetric(p50, "p50-page-ms")
	b.ReportMetric(p50Fields, "p50-page+fields-ms")
	b.ReportMetric(100*(p50Fields-p50)/p50, "added-%")
}
