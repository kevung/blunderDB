//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// errPlanNode is planNode plus how a node hangs from its parent: a read of
// analysis under a SubPlan is one probe per position kept so far.
type errPlanNode struct {
	NodeType string        `json:"Node Type"`
	Relation string        `json:"Relation Name"`
	Parent   string        `json:"Parent Relationship"`
	Plans    []errPlanNode `json:"Plans"`
}

// A move-error search below the blunder floor, with a filter on position,
// reads the error of each position it walks by one index probe of analysis.
// Joined instead, the planner hashes the analysis of the whole tenant to keep
// one page, under the row-level policy as owner-less as in production.
func TestMoveErrorSearchProbesAnalysisPerPosition(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	if _, err := conn.Exec(ctx, `ANALYZE position; ANALYZE analysis`); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `DO $$ BEGIN CREATE ROLE move_error_plan_rls NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
		GRANT USAGE ON SCHEMA public TO move_error_plan_rls; GRANT SELECT ON ALL TABLES IN SCHEMA public TO move_error_plan_rls`); err != nil {
		t.Fatal(err)
	}
	explain := func(sql string, args []any, hash bool) ([]errPlanNode, error) {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE move_error_plan_rls; SELECT set_config('app.tenant_id', '1', true)`); err != nil {
			return nil, err
		}
		if hash {
			// A fixture this small is joined cheapest by a nested loop: the
			// planner is pushed toward what it chooses on a full tenant.
			if _, err := tx.Exec(ctx, `SET LOCAL enable_nestloop = off; SET LOCAL enable_mergejoin = off`); err != nil {
				return nil, err
			}
		}
		var out []byte
		if err := tx.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+sql, args...).Scan(&out); err != nil {
			return nil, err
		}
		var plans []struct {
			Plan errPlanNode `json:"Plan"`
		}
		if err := json.Unmarshal(out, &plans); err != nil {
			return nil, err
		}
		return []errPlanNode{plans[0].Plan}, nil
	}
	// joinedReads lists the reads of analysis that no SubPlan holds.
	var joinedReads func(nodes []errPlanNode, inSubPlan bool) []string
	joinedReads = func(nodes []errPlanNode, inSubPlan bool) []string {
		var got []string
		for _, n := range nodes {
			sub := inSubPlan || n.Parent == "SubPlan"
			if n.Relation == "analysis" && !sub {
				got = append(got, n.NodeType)
			}
			got = append(got, joinedReads(n.Plans, sub)...)
		}
		return got
	}

	for _, c := range []struct {
		name   string
		filter domain.SearchFilters
	}{
		{"common and phase", domain.SearchFilters{MoveErrorFilter: "E>20", GamePhaseFilter: "middlegame"}},
		{"band and phase", domain.SearchFilters{MoveErrorFilter: "E20,60", GamePhaseFilter: "race"}},
		{"common alone", domain.SearchFilters{MoveErrorFilter: "E>5"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			// Candidates too many to name, as on a full library: the clause
			// is then what keeps them.
			defer sqlshared.SetNarrowLimit(2)()
			rec.record(c.name)
			_, _, err := s.Search().FindIDsPage(ctx, "1", c.filter, storage.PageOpts{Limit: 26})
			if err == nil {
				_, err = s.Search().Count(ctx, "1", c.filter)
			}
			rec.record("")
			if err != nil {
				t.Fatal(err)
			}
			rec.mu.Lock()
			stmts := rec.stmts
			rec.stmts = nil
			rec.mu.Unlock()
			checked := 0
			for _, st := range stmts {
				// The probe naming the candidates reads the error indexes of
				// analysis, a bounded list; the clause lands elsewhere.
				if !explainable.MatchString(st.sql) || !strings.Contains(st.sql, "multi_played") ||
					strings.Contains(st.sql, "FROM analysis an ") {
					continue
				}
				checked++
				for _, hash := range []bool{false, true} {
					plan, err := explain(st.sql, st.args, hash)
					if err != nil {
						t.Fatalf("explain: %v\n  in: %s", err, oneLine(st.sql))
					}
					if got := joinedReads(plan, false); len(got) > 0 {
						t.Errorf("analysis joined (%v), not probed per position\n  in: %s", got, oneLine(st.sql))
					}
				}
			}
			if checked == 0 {
				t.Fatalf("no statement carries the clause; sent: %d", len(stmts))
			}
		})
	}
}
