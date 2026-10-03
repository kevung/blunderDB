package database

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The search-window benchmarks run on a real-sized base named by
// BLUNDERDB_SEARCH_DB (a copy: opening it may migrate it), and are skipped
// without one. Each filter is measured two ways, to show what a window saves:
// the whole result (every id, through Find) and what the GUI asks to show it —
// the first window, the count, the rank of the last id. Figures:
// tasks/bench/search-windows.md.
var (
	searchDBOnce sync.Once
	searchDB     *Database
	searchDBErr  error
)

func openSearchDB(b *testing.B) *Database {
	b.Helper()
	path := os.Getenv("BLUNDERDB_SEARCH_DB")
	if path == "" {
		b.Skip("BLUNDERDB_SEARCH_DB not set")
	}
	searchDBOnce.Do(func() {
		db := NewDatabase()
		if searchDBErr = db.OpenDatabase(path); searchDBErr == nil {
			searchDB = db
		}
	})
	if searchDBErr != nil {
		b.Fatalf("search database %s: %v", path, searchDBErr)
	}
	return searchDB
}

func benchSearchWindows(b *testing.B, f SearchFilters) {
	db := openSearchDB(b)
	ctx := context.Background()
	search := db.store.Search()
	var last int64
	b.Run("every-id-via-Find", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var ids []int64
			for pos, err := range search.Find(ctx, "", f, storage.ListOpts{}) {
				if err != nil {
					b.Fatal(err)
				}
				ids = append(ids, pos.ID)
			}
			b.ReportMetric(float64(len(ids)), "rows")
			if len(ids) > 0 {
				last = ids[len(ids)-1]
			}
		}
	})
	b.Run("first-window-100", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			ids, err := search.FindIDs(ctx, "", f, storage.ListOpts{Limit: 100})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(len(ids)), "rows")
		}
	})
	b.Run("count", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			n, err := search.Count(ctx, "", f)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(n), "rows")
		}
	})
	if last == 0 {
		return
	}
	b.Run("index-of-last", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, found, err := search.IndexOf(ctx, "", f, last); err != nil || !found {
				b.Fatalf("IndexOf(%d): %v, %v", last, found, err)
			}
		}
	})
}

// Wide, all in SQL: every position.
func BenchmarkSearchWindows_WideAll(b *testing.B) {
	benchSearchWindows(b, SearchFilters{Filter: emptyFilter()})
}

// Wide, all in SQL: the cube decisions.
func BenchmarkSearchWindows_WideCube(b *testing.B) {
	filter := emptyFilter()
	filter.DecisionType = CubeAction
	benchSearchWindows(b, SearchFilters{Filter: filter, DecisionTypeFilter: true})
}

// Narrow and indexed: one roll at one score.
func BenchmarkSearchWindows_NarrowDiceAndScore(b *testing.B) {
	filter := emptyFilter()
	filter.Dice = [2]int{6, 5}
	filter.Score = [2]int{6, 4}
	benchSearchWindows(b, SearchFilters{Filter: filter, DiceRollFilter: true, IncludeScore: true})
}

// Not indexed, decided in Go: the analysis date, read from every blob.
func BenchmarkSearchWindows_GoPhaseDate(b *testing.B) {
	benchSearchWindows(b, SearchFilters{Filter: emptyFilter(), DateFilter: "T>2000/01/01"})
}
