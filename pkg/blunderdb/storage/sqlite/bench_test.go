package sqlite_test

// Microbenchmarks for the SQLite backend (P9). They run on file-backed temp
// databases (busy_timeout + pool sizing applied by Open) so the numbers reflect
// the real serve path, not a single in-memory connection.
//
//	go test -tags '' -bench . -benchmem ./pkg/blunderdb/storage/sqlite/
//
// BenchmarkStatsCompute seeds via the legacy Database importer (the external
// test package may import database), then measures the heaviest read path.

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// benchPos returns a position whose Zobrist hash is unique for each i across a
// very large range (≈2^28). PlayerOnRoll stays Black(0) so NormalizeForStorage
// does not mirror, keeping the encoding collision-free: four board points carry
// 4 bits each and the two scores carry 6 bits each.
func benchPos(i int) domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	for k := 0; k < 4; k++ {
		n := (i >> (4 * k)) & 15
		p.Board.Points[1+k] = domain.Point{Checkers: n, Color: domain.White}
	}
	p.Score[0] = (i >> 16) & 63
	p.Score[1] = (i >> 22) & 63
	return p
}

func BenchmarkSavePosition(b *testing.B) {
	s := openTempDBB(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := benchPos(i)
		if _, err := s.Positions().Save(ctx, "", &p); err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

func BenchmarkLoadPosition(b *testing.B) {
	s := openTempDBB(b)
	ctx := context.Background()
	p := benchPos(1)
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		b.Fatalf("seed Save: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Positions().Load(ctx, "", id); err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkSearchByZobrist(b *testing.B) {
	s := openTempDBB(b)
	ctx := context.Background()
	p := benchPos(7)
	if _, err := s.Positions().Save(ctx, "", &p); err != nil {
		b.Fatalf("seed Save: %v", err)
	}
	hash := engine.ZobristHash(&p)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, found, err := s.Positions().Exists(ctx, "", hash); err != nil || !found {
			b.Fatalf("Exists: found=%v err=%v", found, err)
		}
	}
}

func BenchmarkSearchByFilter(b *testing.B) {
	s := openTempDBB(b)
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		p := benchPos(i)
		if i%2 == 0 {
			p.DecisionType = domain.CubeAction
		}
		if _, err := s.Positions().Save(ctx, "", &p); err != nil {
			b.Fatalf("seed Save: %v", err)
		}
	}
	f := domain.SearchFilters{DecisionTypeFilter: true}
	f.Filter.DecisionType = domain.CheckerAction
	f.Filter.PlayerOnRoll = domain.Black

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := 0
		for _, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
			if err != nil {
				b.Fatalf("Find: %v", err)
			}
			n++
		}
	}
}

// seedSearchTextAndMoveErrorFixture saves n positions, each with a comment,
// one recorded player-1 move, and one analysis — the fixture
// BenchmarkSearchText and BenchmarkSearchMoveError both search over.
func seedSearchTextAndMoveErrorFixture(b *testing.B, s *sqlite.Storage, n int) {
	b.Helper()
	ctx := context.Background()

	m := domain.Match{Player1Name: "me", Player2Name: "them", MatchLength: 7}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		b.Fatalf("seed match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		b.Fatalf("seed game: %v", err)
	}
	small := 0.05
	for i := 0; i < n; i++ {
		p := benchPos(i)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			b.Fatalf("seed Save: %v", err)
		}
		if _, err := s.Comments().Add(ctx, "", id, "blunder review pending"); err != nil {
			b.Fatalf("seed comment: %v", err)
		}
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "checker",
			PositionID: id, Player: 1, CheckerMove: "13/11 24/23"}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			b.Fatalf("seed move: %v", err)
		}
		if err := s.Analyses().Save(ctx, "", id, &domain.PositionAnalysis{
			AnalysisType: "CheckerMove",
			PlayedMoves:  []string{"13/11 24/23"},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "13/11 24/23", Equity: 0.45, EquityError: &small},
			}},
		}); err != nil {
			b.Fatalf("seed analysis: %v", err)
		}
	}
}

func runSearchFind(b *testing.B, s *sqlite.Storage, f domain.SearchFilters, want int) {
	b.Helper()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cnt := 0
		for _, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
			if err != nil {
				b.Fatalf("Find: %v", err)
			}
			cnt++
		}
		if cnt != want {
			b.Fatalf("Find: got %d results, want %d", cnt, want)
		}
	}
}

// BenchmarkSearchText exercises the SearchText ("t\"…\"") N+1 B.10 (#178)
// folds into a single batched preload (loadCommentTexts): every SQL-matched
// candidate used to re-query its comment text on its own
// (loadCommentText), one query per row — n positions all SQL-matched used to
// mean n extra round trips on top of the search query itself. SearchText
// alone needs no analysis decode (needAnalysis stays false), so this isolates
// the comment-preload win from the cost of decoding the analysis blob.
func BenchmarkSearchText(b *testing.B) {
	s := openTempDBB(b)
	const n = 5000
	seedSearchTextAndMoveErrorFixture(b, s, n)
	runSearchFind(b, s, domain.SearchFilters{SearchText: `t"blunder"`}, n)
}

// BenchmarkSearchMoveErrorMirror is BenchmarkSearchText's counterpart for the
// move-error N+1 (B.10, #178): a MoveErrorFilter used to re-query every
// row's recorded plays on its own (getPlayer1MovesForPosition), folded into
// loadPlayer1Moves. A plain (non-mirror) MoveErrorFilter search is mostly
// settled in SQL already (the equity-error column is pushed down for a
// position player 1 played once — see the comment above the WHERE clause
// that builds it) and only re-checks the rare multi-played position in Go,
// so a mirror search (MirrorFilter, which routes every row through the
// Go-side predicate regardless) is what actually exercises the row-by-row
// query this fiche removes.
func BenchmarkSearchMoveErrorMirror(b *testing.B) {
	s := openTempDBB(b)
	const n = 5000
	seedSearchTextAndMoveErrorFixture(b, s, n)
	runSearchFind(b, s, domain.SearchFilters{MoveErrorFilter: "E>0", MirrorFilter: true}, n)
}

// peakHeapAlloc starts a background sampler of runtime.MemStats.HeapAlloc and
// returns a function that stops it and reports the maximum it observed. Used
// by BenchmarkRepairDenormalisedColumnsMemory to see the *peak* memory a call
// holds, not the total bytes it ever allocated (-benchmem's B/op sums
// allocations over the call's whole lifetime — paging does not shrink that
// sum, since every row's blob is still read and processed exactly once
// either way; what paging bounds is how many of those blobs are alive in
// memory *at once*, which only a live sample during the call can see).
func peakHeapAlloc() (stop func() uint64) {
	var peak atomic.Uint64
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var m runtime.MemStats
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		sample := func() {
			runtime.ReadMemStats(&m)
			for {
				cur := peak.Load()
				if m.HeapAlloc <= cur || peak.CompareAndSwap(cur, m.HeapAlloc) {
					return
				}
			}
		}
		for {
			select {
			case <-done:
				sample()
				return
			case <-ticker.C:
				sample()
			}
		}
	}()
	return func() uint64 {
		close(done)
		wg.Wait()
		return peak.Load()
	}
}

// BenchmarkRepairDenormalisedColumnsMemory is B.11's (#179) memory bench:
// RepairDenormalisedColumns used to read every analysis row's blob into one
// slice (`var all []row`) before decoding any of them — a real database
// holds tens of thousands, and the point of a repair is to run on the
// biggest ones. Paged (repairPageSize, 500 rows at a time), peak HeapAlloc
// should stay roughly flat as n grows instead of scaling with the table.
//
// Run alone, once: go test -run=^$ -bench BenchmarkRepairDenormalisedColumnsMemory
// -benchtime=1x ./pkg/blunderdb/storage/sqlite/
func BenchmarkRepairDenormalisedColumnsMemory(b *testing.B) {
	const n = 20000
	dsn := filepath.Join(b.TempDir(), "repair.db")
	ctx := context.Background()

	s, err := sqlite.Open(ctx, dsn, nil)
	if err != nil {
		b.Fatalf("sqlite.Open: %v", err)
	}
	for i := 0; i < n; i++ {
		p := benchPos(i)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			b.Fatalf("seed Save: %v", err)
		}
		if err := s.Analyses().Save(ctx, "", id, &domain.PositionAnalysis{
			AnalysisType: "CheckerMove",
			PlayedMoves:  []string{"13/11 24/23"},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "13/11 24/23", Equity: 0.45},
			}},
		}); err != nil {
			b.Fatalf("seed analysis: %v", err)
		}
	}
	s.Close()

	// A second, raw connection corrupts every row's denormalised column
	// directly — Analyses().Save would just recompute it correctly, defeating
	// the point: RepairDenormalisedColumns must have something to fix on
	// every single row (decode the blob, compare, write back), not skip it.
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		b.Fatalf("sql.Open: %v", err)
	}
	if _, err := raw.Exec(`UPDATE analysis SET best_move_equity_error = -999999`); err != nil {
		b.Fatalf("corrupt columns: %v", err)
	}
	raw.Close()

	s2, err := sqlite.Open(ctx, dsn, nil)
	if err != nil {
		b.Fatalf("reopen: %v", err)
	}
	defer s2.Close()

	runtime.GC()
	stop := peakHeapAlloc()
	b.ResetTimer()
	repaired, err := s2.Analyses().RepairDenormalisedColumns(ctx, "")
	b.StopTimer()
	peak := stop()

	if err != nil {
		b.Fatalf("RepairDenormalisedColumns: %v", err)
	}
	if repaired != n {
		b.Fatalf("repaired = %d, want %d (every row was corrupted)", repaired, n)
	}
	b.ReportMetric(float64(peak)/(1024*1024), "peak-heap-MB")
}

func BenchmarkStatsCompute(b *testing.B) {
	ctx := context.Background()
	dbPath := filepath.Join(b.TempDir(), "stats.db")
	d := database.NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		b.Fatalf("SetupDatabase: %v", err)
	}
	xg := filepath.Join("..", "..", "..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg")
	if _, err := d.ImportXGMatch(xg); err != nil {
		b.Fatalf("seed ImportXGMatch: %v", err)
	}
	d.Close()

	s, err := sqlite.Open(ctx, dbPath, nil)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer s.Close()

	filter := storage.StatsFilter{DecisionType: -1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Stats().Compute(ctx, "", filter); err != nil {
			b.Fatalf("Compute: %v", err)
		}
	}
}

func BenchmarkConcurrentInsert(b *testing.B) {
	s := openTempDBB(b)
	ctx := context.Background()
	var counter int64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(atomic.AddInt64(&counter, 1))
			p := benchPos(i)
			if _, err := s.Positions().Save(ctx, "", &p); err != nil {
				b.Fatalf("Save: %v", err)
			}
		}
	})
}

// openTempDBB is the *testing.B counterpart of openTempDB (concurrent_test.go).
func openTempDBB(b *testing.B) *sqlite.Storage {
	b.Helper()
	dsn := filepath.Join(b.TempDir(), "bench.db")
	s, err := sqlite.Open(context.Background(), dsn, nil)
	if err != nil {
		b.Fatalf("sqlite.Open: %v", err)
	}
	b.Cleanup(func() { _ = s.Close() })
	return s
}
