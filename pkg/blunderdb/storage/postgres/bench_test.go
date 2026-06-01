//go:build postgres

// Microbenchmarks for the PostgreSQL backend (P9). They need Docker and are
// gated behind the `postgres` build tag:
//
//	go test -tags postgres -bench . -benchmem ./pkg/blunderdb/storage/postgres/
//
// A single throwaway PostgreSQL container is shared across all benchmarks
// (started lazily, reaped by ryuk on process exit). Each benchmark uses its own
// tenant scope so their data does not interfere.
package postgres_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/migrate"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

var (
	benchOnce   sync.Once
	benchStore  *pg.Storage
	benchSkip   string
	benchTenant int64 // atomic; hands each benchmark a private scope
)

// benchPG returns the shared Postgres storage, starting the container once.
func benchPG(b *testing.B) (*pg.Storage, string) {
	b.Helper()
	benchOnce.Do(func() {
		ctx := context.Background()
		container, err := tcpg.Run(ctx, "postgres:16-alpine",
			tcpg.WithDatabase("blunderdb"),
			tcpg.WithUsername("test"),
			tcpg.WithPassword("test"),
			tcpg.BasicWaitStrategies(),
		)
		if err != nil {
			benchSkip = fmt.Sprintf("postgres container unavailable (Docker required): %v", err)
			return
		}
		dsn, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			benchSkip = fmt.Sprintf("connection string: %v", err)
			return
		}
		st, err := pg.Open(ctx, dsn, nil)
		if err != nil {
			benchSkip = fmt.Sprintf("open: %v", err)
			return
		}
		if err := st.Migrate(ctx); err != nil {
			benchSkip = fmt.Sprintf("migrate: %v", err)
			return
		}
		benchStore = st
		// container is intentionally left to the ryuk reaper.
	})
	if benchSkip != "" {
		b.Skip(benchSkip)
	}
	scope := fmt.Sprintf("%d", atomic.AddInt64(&benchTenant, 1))
	return benchStore, scope
}

func benchPosPG(i int) domain.Position {
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
	s, scope := benchPG(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := benchPosPG(i)
		if _, err := s.Positions().Save(ctx, scope, &p); err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

func BenchmarkLoadPosition(b *testing.B) {
	s, scope := benchPG(b)
	ctx := context.Background()
	p := benchPosPG(1)
	id, err := s.Positions().Save(ctx, scope, &p)
	if err != nil {
		b.Fatalf("seed Save: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Positions().Load(ctx, scope, id); err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

func BenchmarkSearchByZobrist(b *testing.B) {
	s, scope := benchPG(b)
	ctx := context.Background()
	p := benchPosPG(7)
	if _, err := s.Positions().Save(ctx, scope, &p); err != nil {
		b.Fatalf("seed Save: %v", err)
	}
	hash := engine.ZobristHash(&p)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, found, err := s.Positions().Exists(ctx, scope, hash); err != nil || !found {
			b.Fatalf("Exists: found=%v err=%v", found, err)
		}
	}
}

func BenchmarkSearchByFilter(b *testing.B) {
	s, scope := benchPG(b)
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		p := benchPosPG(i)
		if i%2 == 0 {
			p.DecisionType = domain.CubeAction
		}
		if _, err := s.Positions().Save(ctx, scope, &p); err != nil {
			b.Fatalf("seed Save: %v", err)
		}
	}
	f := domain.SearchFilters{DecisionTypeFilter: true}
	f.Filter.DecisionType = domain.CheckerAction
	f.Filter.PlayerOnRoll = domain.Black

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, err := range s.Search().Find(ctx, scope, f) {
			if err != nil {
				b.Fatalf("Find: %v", err)
			}
		}
	}
}

func BenchmarkStatsCompute(b *testing.B) {
	s, scope := benchPG(b)
	ctx := context.Background()

	// Seed: import an XG match into SQLite, migrate it into this tenant.
	dbPath := filepath.Join(b.TempDir(), "src.db")
	d := database.NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		b.Fatalf("SetupDatabase: %v", err)
	}
	xg := filepath.Join("..", "..", "..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg")
	if _, err := d.ImportXGMatch(xg); err != nil {
		b.Fatalf("seed ImportXGMatch: %v", err)
	}
	d.Close()
	src, err := sqlite.Open(ctx, dbPath, nil)
	if err != nil {
		b.Fatalf("open src: %v", err)
	}
	defer src.Close()
	if _, err := migrate.Run(ctx, src, s, scope, migrate.Options{}); err != nil {
		b.Fatalf("migrate: %v", err)
	}

	filter := storage.StatsFilter{DecisionType: -1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Stats().Compute(ctx, scope, filter); err != nil {
			b.Fatalf("Compute: %v", err)
		}
	}
}

func BenchmarkConcurrentInsert(b *testing.B) {
	s, scope := benchPG(b)
	ctx := context.Background()
	var counter int64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(atomic.AddInt64(&counter, 1))
			p := benchPosPG(i)
			if _, err := s.Positions().Save(ctx, scope, &p); err != nil {
				b.Fatalf("Save: %v", err)
			}
		}
	})
}
