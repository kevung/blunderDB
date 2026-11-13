//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// firstRowsTables are the tables an import asks "are these the tenant's first
// rows here?", with the column the bound is drawn on.
var firstRowsTables = []struct{ table, idColumn string }{
	{"position", "id"}, {"analysis", "position_id"}, {"move", "id"},
}

// firstRowsFixture fills one large tenant and two small ones whose identifiers
// interleave with nothing, sit above everything, or are absent: the shapes a
// bound can meet (a tenant's first file, a later file, a re-import, an empty
// tenant, a bound below, at and above the tenant's lowest row).
func firstRowsFixture(ctx context.Context, t *testing.T) *pgx.Conn {
	t.Helper()
	dsn := purgeTestDB(t)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	s := &Storage{binder: binder{db: pool}, pool: pool}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	for _, q := range []string{
		`INSERT INTO position (id, tenant_id, state) SELECT g, 1000, '' FROM generate_series(1, 40000) g`,
		`INSERT INTO position (id, tenant_id, state) SELECT g, 7, '' FROM generate_series(50001, 50300) g`,
		`INSERT INTO position (id, tenant_id, state) SELECT g, 8, '' FROM generate_series(60001, 60002) g`,
		`INSERT INTO analysis (tenant_id, position_id) SELECT tenant_id, id FROM position`,
		`INSERT INTO move (id, tenant_id) SELECT id, tenant_id FROM position`,
		`ANALYZE position`, `ANALYZE analysis`, `ANALYZE move`,
	} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	return conn
}

func TestTenantHasRowsBelowAnswersAsThePlainBound(t *testing.T) {
	ctx := context.Background()
	conn := firstRowsFixture(ctx, t)
	tenants := []int64{1000, 7, 8, 9}
	bounds := []int64{-1, 0, 1, 2, 40000, 40001, 50000, 50001, 50002, 50300, 50301, 60001, 60002, 60003, 1 << 40}
	for _, tb := range firstRowsTables {
		plain := `SELECT EXISTS (SELECT 1 FROM ` + tb.table + ` WHERE tenant_id = $1 AND ` + tb.idColumn + ` < $2)`
		for _, tenant := range tenants {
			for _, bound := range bounds {
				var want, got bool
				if err := conn.QueryRow(ctx, plain, tenant, bound).Scan(&want); err != nil {
					t.Fatal(err)
				}
				if err := conn.QueryRow(ctx, tenantHasRowsBelowSQL(tb.table, tb.idColumn), tenant, bound).Scan(&got); err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Errorf("%s, tenant %d, bound %d: %v, the plain bound says %v", tb.table, tenant, bound, got, want)
				}
			}
		}
	}
}

// The import runs under a pinned generic plan: the statement must reach the
// tenant's key there too, with statistics that know one large tenant only.
func TestTenantHasRowsBelowReadsTheKeyUnderAGenericPlan(t *testing.T) {
	ctx := context.Background()
	conn := firstRowsFixture(ctx, t)
	if _, err := conn.Exec(ctx, `SET plan_cache_mode = force_generic_plan`); err != nil {
		t.Fatal(err)
	}
	for _, tb := range firstRowsTables {
		name := "first_rows_" + tb.table
		if _, err := conn.Exec(ctx, `PREPARE `+name+`(bigint, bigint) AS `+tenantHasRowsBelowSQL(tb.table, tb.idColumn)); err != nil {
			t.Fatal(err)
		}
		var out []byte
		if err := conn.QueryRow(ctx, `EXPLAIN (ANALYZE, FORMAT JSON) EXECUTE `+name+`(7, 1)`).Scan(&out); err != nil {
			t.Fatal(err)
		}
		var plans []struct {
			Plan readNode `json:"Plan"`
		}
		if err := json.Unmarshal(out, &plans); err != nil {
			t.Fatal(err)
		}
		for _, p := range plans {
			if scans, rows := scansAndRowsRead(p.Plan); scans > 0 || rows > 1 {
				t.Errorf("%s: %d sequential scan(s), %.0f rows read for a one-row answer\n%s", tb.table, scans, rows, out)
			}
		}
	}
}

// readNode is what an EXPLAIN (ANALYZE, FORMAT JSON) node says it read.
type readNode struct {
	NodeType string     `json:"Node Type"`
	Relation string     `json:"Relation Name"`
	Rows     float64    `json:"Actual Rows"`
	Loops    float64    `json:"Actual Loops"`
	Removed  float64    `json:"Rows Removed by Filter"`
	Plans    []readNode `json:"Plans"`
}

// scansAndRowsRead counts the sequential scans of a plan and the rows its
// scans of a table fetched, kept or not.
func scansAndRowsRead(n readNode) (scans int, rows float64) {
	if n.Relation != "" {
		rows = (n.Rows + n.Removed) * n.Loops
		if n.NodeType == "Seq Scan" {
			scans++
		}
	}
	for _, c := range n.Plans {
		s, r := scansAndRowsRead(c)
		scans, rows = scans+s, rows+r
	}
	return scans, rows
}
