package database

import (
	"os"
	"testing"
)

// The library-opening benchmarks run on the base named by BLUNDERDB_SEARCH_DB
// (a copy: opening it may migrate it). They measure what the GUI waits for
// before the first board shows: the database opened, the library counted,
// the id page holding the last position (loadAllPositions lands there) and
// the positions around it. Figures: tasks/bench/search-windows.md.

// lastPageOffset is where the GUI's id page holding rank total-1 starts
// (positionList.js, DEFAULT_ID_PAGE_SIZE).
const libraryIDPage = 1000

func lastPageOffset(total int) int {
	if total == 0 {
		return 0
	}
	return (total - 1) / libraryIDPage * libraryIDPage
}

func BenchmarkLibraryOpen(b *testing.B) {
	path := os.Getenv("BLUNDERDB_SEARCH_DB")
	db := openSearchDB(b)
	total, err := db.CountPositions()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(total), "positions")

	b.Run("open", func(b *testing.B) {
		for b.Loop() {
			fresh := NewDatabase()
			if err := fresh.OpenDatabase(path); err != nil {
				b.Fatal(err)
			}
			_ = fresh.Close()
		}
	})
	b.Run("count", func(b *testing.B) {
		for b.Loop() {
			if _, err := db.CountPositions(); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("last-id-page", func(b *testing.B) {
		for b.Loop() {
			if _, err := db.ListPositionIDs(lastPageOffset(total), libraryIDPage); err != nil {
				b.Fatal(err)
			}
		}
	})
	ids, err := db.ListPositionIDs(lastPageOffset(total), libraryIDPage)
	if err != nil {
		b.Fatal(err)
	}
	if len(ids) > 50 {
		ids = ids[len(ids)-50:]
	}
	b.Run("last-positions", func(b *testing.B) {
		for b.Loop() {
			if _, err := db.LoadPositionsByIDs(ids); err != nil {
				b.Fatal(err)
			}
		}
	})
	// The Matches tab the library opens on loads its list alongside.
	b.Run("all-matches", func(b *testing.B) {
		for b.Loop() {
			if _, err := db.GetAllMatches(); err != nil {
				b.Fatal(err)
			}
		}
	})
	// What the GUI chains, end to end, from a database already open.
	b.Run("count+page+positions", func(b *testing.B) {
		for b.Loop() {
			n, err := db.CountPositions()
			if err != nil {
				b.Fatal(err)
			}
			page, err := db.ListPositionIDs(lastPageOffset(n), libraryIDPage)
			if err != nil {
				b.Fatal(err)
			}
			if len(page) > 50 {
				page = page[len(page)-50:]
			}
			if _, err := db.LoadPositionsByIDs(page); err != nil {
				b.Fatal(err)
			}
		}
	})
}
