//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// probeFixture imports the same match into two tenants and records what the
// searches of tenant "1" send.
func probeFixture(ctx context.Context, t *testing.T) (*Storage, *stmtRecorder, string) {
	t.Helper()
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
	return s, rec, dsn
}

// madePoints is a structure many positions of the tenant share: two points
// the first position holds, asked as made points.
func madePoints(ctx context.Context, t *testing.T, s *Storage) domain.Board {
	t.Helper()
	var board domain.Board
	for p, err := range s.Search().Find(ctx, "1", domain.SearchFilters{}, storage.ListOpts{Limit: 1}) {
		if err != nil {
			t.Fatal(err)
		}
		made := 0
		for i, pt := range p.Board.Points {
			if pt.Checkers >= 2 && made < 2 {
				board.Points[i] = domain.Point{Checkers: 2, Color: pt.Color}
				made++
			}
		}
		if made < 2 {
			t.Fatalf("the first position has %d made points", made)
		}
	}
	return board
}

// searchAnswers is what a search answers through the windows a probe may read:
// a first window, and three pages chained by cursor.
func searchAnswers(ctx context.Context, s *Storage, f domain.SearchFilters) string {
	var b strings.Builder
	ids, err := s.Search().FindIDs(ctx, "1", f, storage.ListOpts{Limit: 7})
	fmt.Fprintf(&b, "ids %v %v;", ids, err)
	cursor := ""
	for page := 0; page < 3; page++ {
		ids, next, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 5, Cursor: cursor})
		fmt.Fprintf(&b, " page %v %v;", ids, err)
		if err != nil || next == "" {
			break
		}
		cursor = next
	}
	var found []int64
	for p, err := range s.Search().Find(ctx, "1", f, storage.ListOpts{Limit: 9}) {
		if err != nil {
			fmt.Fprintf(&b, " find %v", err)
			break
		}
		found = append(found, p.ID)
	}
	fmt.Fprintf(&b, " find %v", found)
	return b.String()
}

// isProbe spots the statement of a probe: a window read from a bounded walk.
func isProbe(sql string) bool { return strings.Contains(sql, "FROM (SELECT * FROM position p WHERE") }

// A probe is a shortcut: whatever it walks, one row or the whole tenant, and
// whichever read follows a short one, the search answers what it answers
// without it — same ids, same order, same pages.
func TestSearchAnswersTheSameThroughAProbe(t *testing.T) {
	ctx := context.Background()
	s, rec, _ := probeFixture(ctx, t)
	structure := madePoints(ctx, t, s)

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "search_query_corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Cases []struct {
			Command string `json:"command"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	var filters []domain.SearchFilters
	for _, c := range corpus.Cases {
		f, _ := searchquery.Parse(c.Command)
		filters = append(filters, f)
		excluded := f
		excluded.ExcludeFilter.Board = structure
		filters = append(filters, excluded)
		f.Filter.Board = structure
		filters = append(filters, f)
	}

	restore := sqlshared.SetIDWalkProbe(0)
	want := make([]string, len(filters))
	for i, f := range filters {
		want[i] = searchAnswers(ctx, s, f)
	}
	restore()
	rec.mu.Lock()
	for _, st := range rec.stmts {
		if isProbe(st.sql) {
			t.Fatalf("a probe was sent with the probe off: %s", oneLine(st.sql))
		}
	}
	rec.mu.Unlock()

	// sorted false: a short probe is followed by the walk the planner chooses.
	for _, run := range []struct {
		rows   int
		sorted bool
	}{{4, true}, {32, true}, {240, true}, {80000, true}, {32, false}} {
		restoreProbe := sqlshared.SetIDWalkProbe(run.rows)
		restoreSort := func() {}
		if !run.sorted {
			restoreSort = sqlshared.SetSortedReadMax(-1)
		}
		rec.record(fmt.Sprintf("probe %d", run.rows))
		for i, f := range filters {
			if got := searchAnswers(ctx, s, f); got != want[i] {
				t.Errorf("probe of %d rows, sorted read %v, %q (case %d):\n  got  %s\n  want %s",
					run.rows, run.sorted, corpus.Cases[i/3].Command, i, got, want[i])
			}
		}
		restoreSort()
		restoreProbe()
	}
	rec.record("")
	rec.mu.Lock()
	defer rec.mu.Unlock()
	probes, sorted := 0, 0
	for _, st := range rec.stmts {
		if isProbe(st.sql) {
			probes++
			if strings.Contains(st.sql, "p.id IN (") {
				t.Errorf("a probe carries a list of ids: %s", oneLine(st.sql)[:200])
			}
		}
		if strings.Contains(st.sql, "ORDER BY p.id + 0") {
			sorted++
		}
	}
	if probes < len(filters) || sorted == 0 {
		t.Fatalf("%d probes and %d sorted reads for %d filters: the searches no longer probe", probes, sorted, len(filters))
	}
}

// growTenant copies the positions of tenant 1 until it holds enough of them
// for a walk of the whole tenant to show.
func growTenant(ctx context.Context, t *testing.T, conn *pgx.Conn, copies int) (rows int) {
	t.Helper()
	var cols []string
	found, err := conn.Query(ctx, `SELECT column_name::text FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'position' AND is_generated = 'NEVER' ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	for found.Next() {
		var c string
		if err := found.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	if found.Err() != nil {
		t.Fatal(found.Err())
	}
	cols = slices.DeleteFunc(cols, func(c string) bool { return c == "id" })
	values := make([]string, len(cols))
	for i, c := range cols {
		switch c {
		case "zobrist_hash":
			values[i] = "zobrist_hash # (g::bigint * 7919)"
		case "uuid":
			values[i] = "gen_random_uuid()"
		default:
			values[i] = c
		}
	}
	if _, err := conn.Exec(ctx, `INSERT INTO position (`+strings.Join(cols, ", ")+`) SELECT `+strings.Join(values, ", ")+
		` FROM position, generate_series(1, $1::int) g WHERE tenant_id = 1 ORDER BY g, id`, copies); err != nil {
		t.Fatalf("copy positions: %v", err)
	}
	if _, err := conn.Exec(ctx, `ANALYZE position`); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM position WHERE tenant_id = 1`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// A structure and another filter: the planner believes the masks keep one row
// and would read the tenant, then sort. The page comes from the walk in id
// order, which stops once the page is full: no sort, no pass over the tenant.
func TestStructureSearchPageStopsInTheIDWalk(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	tenantRows := growTenant(ctx, t, conn, 80)

	f := domain.SearchFilters{DecisionTypeFilter: true}
	f.Filter.Board = madePoints(ctx, t, s)
	f.Filter.DecisionType = domain.CheckerAction
	const page = 25
	rec.record("page")
	ids, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: page})
	rec.record("")
	if err != nil || len(ids) != page {
		t.Fatalf("page: %d ids, %v", len(ids), err)
	}
	rec.mu.Lock()
	stmts := rec.stmts
	rec.mu.Unlock()
	probes := 0
	for _, st := range stmts {
		if !isProbe(st.sql) {
			continue
		}
		probes++
		var out []byte
		if err := conn.QueryRow(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) "+st.sql, st.args...).Scan(&out); err != nil {
			t.Fatalf("explain: %v\n  in: %s", err, oneLine(st.sql))
		}
		var plans []struct {
			Plan walkNode `json:"Plan"`
		}
		if err := json.Unmarshal(out, &plans); err != nil {
			t.Fatal(err)
		}
		sorts, read := plans[0].Plan.sortsAndPositionsRead()
		if sorts > 0 || read >= float64(tenantRows)/4 {
			t.Errorf("%d sort(s), %.0f of the tenant's %d positions read for a page of %d\n  in: %s\n  plan: %s",
				sorts, read, tenantRows, page, oneLine(st.sql), out)
		}
	}
	if probes == 0 {
		t.Fatal("no probe sent: the structure search no longer walks the ids first")
	}
}

// walkNode is what an EXPLAIN (ANALYZE, FORMAT JSON) node says it did.
type walkNode struct {
	NodeType string     `json:"Node Type"`
	Relation string     `json:"Relation Name"`
	Rows     float64    `json:"Actual Rows"`
	Loops    float64    `json:"Actual Loops"`
	Removed  float64    `json:"Rows Removed by Filter"`
	Plans    []walkNode `json:"Plans"`
}

// sortsAndPositionsRead counts the sorts of a plan and the position rows its
// scans fetched, kept or not.
func (n walkNode) sortsAndPositionsRead() (sorts int, read float64) {
	if strings.Contains(n.NodeType, "Sort") {
		sorts++
	}
	if n.Relation == "position" {
		read = (n.Rows + n.Removed) * n.Loops
	}
	for _, c := range n.Plans {
		s, r := c.sortsAndPositionsRead()
		sorts, read = sorts+s, read+r
	}
	return sorts, read
}

// Two columns rarely true together — a cube decision has no dice: the planner
// believes them independent and bets on the walk in id order, which reads the
// whole tenant to find nothing. The probe stops that walk short, and the page
// is read by the columns' indexes; told the filter keeps too many rows to
// sort, the search leaves the choice to the planner, as before.
func TestSearchOfColumnsRarelyTrueTogetherLeavesTheIDWalk(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	tenantRows := growTenant(ctx, t, conn, 80)

	f := domain.SearchFilters{DecisionTypeFilter: true, DiceRollFilter: true}
	f.Filter.DecisionType = domain.CubeAction
	f.Filter.Dice = [2]int{3, 1}
	read := func(phase string) (probes, sorted, walked []recordedStmt) {
		rec.record(phase)
		ids, _, err := s.Search().FindIDsPage(ctx, "1", f, storage.PageOpts{Limit: 25})
		rec.record("")
		if err != nil || len(ids) != 0 {
			t.Fatalf("%s: %d ids, %v: a cube decision with dice", phase, len(ids), err)
		}
		rec.mu.Lock()
		defer rec.mu.Unlock()
		for _, st := range rec.stmts {
			switch {
			case st.phase != phase || !strings.Contains(st.sql, "ORDER BY p.id"):
			case isProbe(st.sql):
				probes = append(probes, st)
			case strings.Contains(st.sql, "ORDER BY p.id + 0"):
				sorted = append(sorted, st)
			default:
				walked = append(walked, st)
			}
		}
		return probes, sorted, walked
	}

	probes, sorted, walked := read("sorted")
	if len(probes) != 1 || len(sorted) != 1 || len(walked) != 0 {
		t.Fatalf("%d probe(s), %d sorted read(s), %d walk(s): want a short probe, then a sorted read", len(probes), len(sorted), len(walked))
	}
	var out []byte
	if err := conn.QueryRow(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) "+probes[0].sql, probes[0].args...).Scan(&out); err != nil {
		t.Fatal(err)
	}
	var plans []struct {
		Plan walkNode `json:"Plan"`
	}
	if err := json.Unmarshal(out, &plans); err != nil {
		t.Fatal(err)
	}
	if _, rows := plans[0].Plan.sortsAndPositionsRead(); rows >= float64(tenantRows)/4 {
		t.Errorf("the probe read %.0f of the tenant's %d positions: it is not bounded\n  plan: %s", rows, tenantRows, out)
	}

	defer sqlshared.SetSortedReadMax(-1)()
	probes, sorted, walked = read("walked")
	if len(probes) != 1 || len(sorted) != 0 || len(walked) != 1 {
		t.Fatalf("%d probe(s), %d sorted read(s), %d walk(s): want a short probe, then the planner's own read", len(probes), len(sorted), len(walked))
	}
}
