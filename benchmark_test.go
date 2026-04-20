package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// silenceLogs redirects os.Stderr and the log package to /dev/null so that
// verbose import/search messages don't bloat benchmark output. Call the
// returned function to restore normal output.
func silenceLogs() func() {
	origStderr := os.Stderr
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return func() {}
	}
	os.Stderr = devNull
	log.SetOutput(devNull)
	return func() {
		os.Stderr = origStderr
		log.SetOutput(origStderr)
		devNull.Close()
	}
}

// sharedBenchDB holds the pre-imported database reused across all benchmarks.
var sharedBenchDB *Database
var benchOnce sync.Once
var benchSetupErr error
var benchFileCount int
var benchTotalBytes int64

// setupBenchDB walks testdata/tournois/ and imports every .xg file into a
// shared in-memory Database. Call it from every Benchmark* via benchOnce so
// the import happens exactly once per test binary invocation.
func setupBenchDB(tb testing.TB) *Database {
	tb.Helper()
	benchOnce.Do(func() {
		tournoisDir := filepath.Join("testdata", "tournois")
		if _, err := os.Stat(tournoisDir); os.IsNotExist(err) {
			benchSetupErr = fmt.Errorf("fixture directory %s does not exist", tournoisDir)
			return
		}

		db := NewDatabase()
		dbPath := ":memory:"
		if os.Getenv("BENCH_DISK") == "1" {
			dbPath = filepath.Join("testdata", "tournois.db")
			os.Remove(dbPath)
		}
		if err := db.SetupDatabase(dbPath); err != nil {
			benchSetupErr = fmt.Errorf("SetupDatabase: %v", err)
			return
		}

		err := filepath.Walk(tournoisDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".xg":
				benchFileCount++
				benchTotalBytes += info.Size()
				if _, importErr := db.ImportXGMatch(path); importErr != nil {
					// Skip duplicates silently; log real errors
					if !strings.Contains(importErr.Error(), "duplicate") {
						fmt.Fprintf(os.Stderr, "WARN: import %s: %v\n", path, importErr)
					}
				}
			case ".sgf", ".mat":
				benchFileCount++
				benchTotalBytes += info.Size()
				if _, importErr := db.ImportGnuBGMatch(path); importErr != nil {
					if !strings.Contains(importErr.Error(), "duplicate") {
						fmt.Fprintf(os.Stderr, "WARN: import %s: %v\n", path, importErr)
					}
				}
			case ".bgf":
				benchFileCount++
				benchTotalBytes += info.Size()
				if _, importErr := db.ImportBGFMatch(path); importErr != nil {
					if !strings.Contains(importErr.Error(), "duplicate") {
						fmt.Fprintf(os.Stderr, "WARN: import %s: %v\n", path, importErr)
					}
				}
			}
			return nil
		})
		if err != nil {
			benchSetupErr = fmt.Errorf("walking tournois: %v", err)
			return
		}

		fmt.Fprintf(os.Stderr, "bench fixture: %d files, %d bytes\n", benchFileCount, benchTotalBytes)
		sharedBenchDB = db
	})
	if benchSetupErr != nil {
		tb.Fatalf("bench setup failed: %v", benchSetupErr)
	}
	if sharedBenchDB == nil {
		tb.Fatal("bench setup: database is nil")
	}
	return sharedBenchDB
}

// ---------- Import benchmarks ----------

func BenchmarkImport_TournoisCold(b *testing.B) {
	// Ensure fixtures exist by triggering the shared setup (but this benchmark
	// creates its own fresh DB each iteration).
	setupBenchDB(b)

	// Collect the file list once.
	var files []string
	tournoisDir := filepath.Join("testdata", "tournois")
	filepath.Walk(tournoisDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".xg" || ext == ".sgf" || ext == ".mat" || ext == ".bgf" {
			files = append(files, path)
		}
		return nil
	})

	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db := NewDatabase()
		if err := db.SetupDatabase(":memory:"); err != nil {
			b.Fatal(err)
		}
		for _, f := range files {
			ext := strings.ToLower(filepath.Ext(f))
			switch ext {
			case ".xg":
				db.ImportXGMatch(f)
			case ".sgf", ".mat":
				db.ImportGnuBGMatch(f)
			case ".bgf":
				db.ImportBGFMatch(f)
			}
		}
	}
}

func BenchmarkImport_TournoisIncremental(b *testing.B) {
	db := setupBenchDB(b)

	// Pick the smallest .xg file to re-import (will be a duplicate, but
	// exercises the full "load all positions" warmup path).
	tournoisDir := filepath.Join("testdata", "tournois")
	var smallest string
	var smallestSize int64 = 1<<63 - 1
	filepath.Walk(tournoisDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".xg" && info.Size() < smallestSize {
			smallest = path
			smallestSize = info.Size()
		}
		return nil
	})
	if smallest == "" {
		b.Skip("no .xg files found")
	}

	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.ImportXGMatch(smallest) // likely duplicate — that's fine
	}
}

// ---------- Search benchmarks ----------

// emptyFilter returns a zero-value Position (matches any checker layout).
func emptyFilter() Position {
	return Position{}
}

func BenchmarkSearch_DecisionCube(b *testing.B) {
	db := setupBenchDB(b)
	filter := emptyFilter()
	filter.DecisionType = CubeAction
	filter.PlayerOnRoll = 0

	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LoadPositionsByFilters(SearchFilters{Filter: filter, DecisionTypeFilter: true})
	}
}

func BenchmarkSearch_ErrorAboveTenth(b *testing.B) {
	db := setupBenchDB(b)
	// moveErrorFilter "E>100" means ≥ 100 millipoints = 0.1 equity
	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LoadPositionsByFilters(SearchFilters{Filter: emptyFilter(), MoveErrorFilter: "E>100"})
	}
}

func BenchmarkSearch_PipWindow(b *testing.B) {
	db := setupBenchDB(b)
	// pipCountFilter "p-2,2" means pip difference in [-2, 2]
	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LoadPositionsByFilters(SearchFilters{Filter: emptyFilter(), PipCountFilter: "p-2,2"})
	}
}

func BenchmarkSearch_WinGammonCombo(b *testing.B) {
	db := setupBenchDB(b)
	// Player1 win > 55% AND gammon > 20%
	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LoadPositionsByFilters(SearchFilters{Filter: emptyFilter(), WinRateFilter: "w>0.55", GammonRateFilter: "g>0.2"})
	}
}

func BenchmarkSearch_ScoreSpecific(b *testing.B) {
	db := setupBenchDB(b)
	filter := emptyFilter()
	filter.Score = [2]int{6, 4}
	// Note: MatchLength is not on Position — the score filter just checks Score[0] and Score[1].

	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LoadPositionsByFilters(SearchFilters{Filter: filter, IncludeScore: true})
	}
}

func BenchmarkSearch_DiceAndPlayer(b *testing.B) {
	db := setupBenchDB(b)
	filter := emptyFilter()
	filter.Dice = [2]int{6, 5}
	filter.PlayerOnRoll = 0
	filter.DecisionType = CheckerAction

	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LoadPositionsByFilters(SearchFilters{Filter: filter, DiceRollFilter: true})
	}
}

func BenchmarkSearch_CheckerStructure(b *testing.B) {
	db := setupBenchDB(b)
	// Anchor on 20-point: Black has ≥ 2 checkers on point 20.
	// Points are 1-indexed in the array (Points[0]=WhiteBar, Points[1..24]=points).
	filter := emptyFilter()
	filter.Board.Points[20] = Point{Checkers: 2, Color: Black}

	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LoadPositionsByFilters(SearchFilters{Filter: filter})
	}
}

func BenchmarkSearch_PrimePattern(b *testing.B) {
	db := setupBenchDB(b)
	// 5-prime: Black has ≥ 2 checkers on points 4, 5, 6, 7, 8.
	filter := emptyFilter()
	for _, pt := range []int{4, 5, 6, 7, 8} {
		filter.Board.Points[pt] = Point{Checkers: 2, Color: Black}
	}

	restore := silenceLogs()
	defer restore()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LoadPositionsByFilters(SearchFilters{Filter: filter})
	}
}
