//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The `tm` filter asks each position for a timed move of player 1. Few
// libraries time their moves: the search must read the timed moves alone
// (idx_move_timed), not probe the moves of every position of the tenant —
// 2 s on 15.6 M positions for a tenant that timed none.
func TestDecisionTimeSearchReadsTheTimedMovesAlone(t *testing.T) {
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
	if _, err := conn.Exec(ctx, `ANALYZE move; ANALYZE position`); err != nil {
		t.Fatal(err)
	}

	rec.record("tm")
	ids, _, err := s.Search().FindIDsPage(ctx, "1", domain.SearchFilters{DecisionTimeFilter: "tm>30"}, storage.PageOpts{Limit: 25})
	rec.record("")
	if err != nil || len(ids) != 0 {
		t.Fatalf("a library without a timed move: %v, %v", ids, err)
	}
	rec.mu.Lock()
	stmts := rec.stmts
	rec.stmts = nil
	rec.mu.Unlock()
	if len(stmts) == 0 {
		t.Fatal("the search sent no statement")
	}
	st := stmts[0]
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
	timed, others := false, []string{}
	var walk func(planNode)
	walk = func(n planNode) {
		if n.Relation == "move" {
			if n.IndexName == "idx_move_timed" {
				timed = true
			} else {
				others = append(others, n.NodeType+" "+n.IndexName)
			}
		}
		for _, c := range n.Plans {
			walk(c)
		}
	}
	walk(plans[0].Plan)
	if !timed || len(others) > 0 {
		t.Errorf("move read through idx_move_timed: %v; otherwise: %v\n  in: %s\n  plan: %s", timed, others, oneLine(st.sql), out)
	}
}
