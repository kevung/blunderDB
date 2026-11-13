//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A blot filter asking for a rare count reads the partial index of its zone:
// the count the search writes must stay the predicate of idx_position_blot_*,
// bound in clear, or the search walks the tenant looking for the few rows
// that pass (20 s cut-off on 15.6 M positions).
func TestRareBlotSearchReadsItsPartialIndex(t *testing.T) {
	ctx := context.Background()
	dsn := purgeTestDB(t)

	rec := &stmtRecorder{}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Tracer = rec
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := &Storage{binder: binder{db: pool}, pool: pool}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("..", "..", "..", "..", "testdata", "test.xg")
	for _, scope := range []string{"2", "1"} {
		if _, err := (ingest.XGImporter{S: s}).Import(ctx, scope, ingest.Source{Path: file}, nil); err != nil {
			t.Fatal(err)
		}
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	if _, err := conn.Exec(ctx, `ANALYZE position; SET enable_seqscan = off; SET enable_bitmapscan = off`); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name  string
		f     domain.SearchFilters
		index string
	}{
		{"bo>4", domain.SearchFilters{Player1OutfieldBlotFilter: "bo>4"}, "idx_position_blot_o1"},
		{"BO6,7", domain.SearchFilters{Player2OutfieldBlotFilter: "BO6,7"}, "idx_position_blot_o2"},
		{"bj>3", domain.SearchFilters{Player1JanBlotFilter: "bj>3"}, "idx_position_blot_j1"},
		{"BJ>5", domain.SearchFilters{Player2JanBlotFilter: "BJ>5"}, "idx_position_blot_j2"},
	} {
		t.Run(c.name, func(t *testing.T) {
			rec.record(c.name)
			_, _, err := s.Search().FindIDsPage(ctx, "1", c.f, storage.PageOpts{Limit: 25})
			rec.record("")
			if err != nil {
				t.Fatal(err)
			}
			rec.mu.Lock()
			stmts := rec.stmts
			rec.stmts = nil
			rec.mu.Unlock()
			explained := 0
			for _, st := range stmts {
				// The bounded probe of the next rows in id order and the
				// planner's estimate come first; what reads the tenant is
				// what must go through the index.
				if !strings.Contains(st.sql, "bit_count") || strings.HasPrefix(st.sql, "EXPLAIN") || strings.Contains(st.sql, "FROM (SELECT") {
					continue
				}
				explained++
				var out []byte
				if err := conn.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+st.sql, st.args...).Scan(&out); err != nil {
					t.Fatalf("explain: %v\n  in: %s", err, oneLine(st.sql))
				}
				var plans []struct {
					Plan planNode `json:"Plan"`
				}
				if err := json.Unmarshal(out, &plans); err != nil {
					t.Fatal(err)
				}
				read := ""
				var walk func(planNode)
				walk = func(n planNode) {
					if n.Relation == "position" {
						read = n.NodeType + " " + n.IndexName
					}
					for _, child := range n.Plans {
						walk(child)
					}
				}
				walk(plans[0].Plan)
				if !strings.HasSuffix(read, " "+c.index) {
					t.Errorf("position read by %q, want %s\n  in: %s\n  plan: %s", read, c.index, oneLine(st.sql), out)
				}
			}
			if explained == 0 {
				t.Fatalf("no statement reads the tenant by the blot count; sent: %d", len(stmts))
			}
		})
	}
}
