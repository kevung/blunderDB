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
	"path/filepath"
	"sync/atomic"
	"testing"

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
		for _, err := range s.Search().Find(ctx, "", f) {
			if err != nil {
				b.Fatalf("Find: %v", err)
			}
			n++
		}
	}
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
