//go:build postgres

// Per-statement timing of an XG import against a real PostgreSQL, run on
// demand:
//
//	BLUNDERDB_BENCH_XG=testdata/test.xg go test -tags postgres -count=1 \
//	    ./pkg/blunderdb/storage/postgres/ -run TestBenchImportXG -v
//
// It prints, by statement shape, how many round-trips the import makes and
// where the wall time goes; without the variable it is skipped.
package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

type stmtStat struct {
	n int
	d time.Duration
}

type stmtTracer struct {
	mu    sync.Mutex
	stats map[string]*stmtStat
}

type startKey struct{}

var ws = regexp.MustCompile(`\s+`)

func (tr *stmtTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	sql := ws.ReplaceAllString(strings.TrimSpace(d.SQL), " ")
	if len(sql) > 90 {
		sql = sql[:90]
	}
	return context.WithValue(ctx, startKey{}, [2]any{time.Now(), sql})
}

func (tr *stmtTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	v, ok := ctx.Value(startKey{}).([2]any)
	if !ok {
		return
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	st := tr.stats[v[1].(string)]
	if st == nil {
		st = &stmtStat{}
		tr.stats[v[1].(string)] = st
	}
	st.n++
	st.d += time.Since(v[0].(time.Time))
}

func TestBenchImportXG(t *testing.T) {
	file := os.Getenv("BLUNDERDB_BENCH_XG")
	if file == "" {
		t.Skip("BLUNDERDB_BENCH_XG not set")
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join("..", "..", "..", "..", file)
	}
	ctx := context.Background()
	dsn := purgeTestDB(t)

	tr := &stmtTracer{stats: map[string]*stmtStat{}}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Tracer = tr
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := &Storage{binder: binder{db: pool}, pool: pool}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	tr.stats = map[string]*stmtStat{}

	t0 := time.Now()
	graph, err := ingest.MapXG(file)
	if err != nil {
		t.Fatal(err)
	}
	mapDur := time.Since(t0)

	t1 := time.Now()
	sum, err := ingest.XGImporter{S: s}.Import(ctx, "", ingest.Source{Path: file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	total := time.Since(t1)
	_ = graph

	type row struct {
		sql string
		st  *stmtStat
	}
	var rows []row
	var sqlTotal time.Duration
	var stmts int
	for k, v := range tr.stats {
		rows = append(rows, row{k, v})
		sqlTotal += v.d
		stmts += v.n
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].st.d > rows[j].st.d })
	pos := max(sum.SavedPositions, 1)
	fmt.Printf("BENCH xg=%s positions=%d map=%v import(total, incl. map)=%v per-position=%.2fms\n",
		filepath.Base(file), sum.SavedPositions, mapDur.Round(time.Millisecond), total.Round(time.Millisecond),
		float64(total.Microseconds())/1000/float64(pos))
	fmt.Printf("BENCH statements=%d (%.1f/position) sql-time=%v\n", stmts, float64(stmts)/float64(pos), sqlTotal.Round(time.Millisecond))
	for i, r := range rows {
		if i >= 40 {
			break
		}
		fmt.Printf("BENCH %6d x %9v  %5.1f%%  %s\n", r.st.n, r.st.d.Round(time.Millisecond), 100*float64(r.st.d)/float64(sqlTotal), r.sql)
	}
}
