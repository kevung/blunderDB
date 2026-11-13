//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// corpusFilters parses the commands of the search corpus.
func corpusFilters(t *testing.T) (filters []domain.SearchFilters, commands []string) {
	t.Helper()
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
	for _, c := range corpus.Cases {
		f, _ := searchquery.Parse(c.Command)
		filters = append(filters, f)
		commands = append(commands, c.Command)
	}
	return filters, commands
}

// tenantAnswers is what a search answers in one tenant through every reader
// of a window: a first window, one read at an offset, three pages chained by
// cursor, the positions themselves, and with count the count.
func tenantAnswers(ctx context.Context, s *Storage, scope string, f domain.SearchFilters, count bool) string {
	var b strings.Builder
	ids, err := s.Search().FindIDs(ctx, scope, f, storage.ListOpts{Limit: 7})
	fmt.Fprintf(&b, "ids %v %v;", ids, err)
	ids, err = s.Search().FindIDs(ctx, scope, f, storage.ListOpts{Limit: 4, Offset: 3})
	fmt.Fprintf(&b, " offset %v %v;", ids, err)
	cursor := ""
	for page := 0; page < 3; page++ {
		ids, next, err := s.Search().FindIDsPage(ctx, scope, f, storage.PageOpts{Limit: 5, Cursor: cursor})
		fmt.Fprintf(&b, " page %v %v;", ids, err)
		if err != nil || next == "" {
			break
		}
		cursor = next
	}
	var found []int64
	for p, err := range s.Search().Find(ctx, scope, f, storage.ListOpts{Limit: 9}) {
		if err != nil {
			fmt.Fprintf(&b, " find %v", err)
			break
		}
		found = append(found, p.ID)
	}
	fmt.Fprintf(&b, " find %v;", found)
	if count {
		n, err := s.Search().Count(ctx, scope, f)
		fmt.Fprintf(&b, " count %d %v", n, err)
	}
	return b.String()
}

// replay answers every filter in the two tenants of the fixture.
func replay(ctx context.Context, s *Storage, filters []domain.SearchFilters, count bool) []string {
	out := make([]string, 0, 2*len(filters))
	for _, f := range filters {
		for _, scope := range []string{"1", "2"} {
			out = append(out, tenantAnswers(ctx, s, scope, f, count))
		}
	}
	return out
}

// isBlunderWindow spots a window cut from the two lists of candidates.
func isBlunderWindow(sql string) bool { return strings.Contains(sql, "AS single UNION ALL") }

// A move-error search from a blunder up reads its windows from the blunder
// index and the positions played several ways, instead of naming every
// candidate in the clause. It is a way of reading, not a filter: alone or
// with any other filter, a structure, a sort key, in chunks of three rows, it
// answers what the named candidates answer — same ids, same order, same
// pages, same count, in each tenant. The chunks of three rows are not counted:
// a count reads the whole tenant, three rows a statement.
func TestMoveErrorSearchAnswersTheSameThroughBlunderWindows(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	structure := madePoints(ctx, t, s)
	// One position in three is marked as played several ways, blunder or not:
	// both lists of candidates hold rows, some of them in both, and Go, which
	// settles the marked ones, rejects some within a window.
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	if _, err := conn.Exec(ctx, `UPDATE position SET multi_played = TRUE WHERE id % 3 = 0`); err != nil {
		t.Fatal(err)
	}
	var blunders, marked int
	if err := conn.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE p.multi_played) FROM analysis a JOIN position p ON p.id = a.position_id
		WHERE a.tenant_id = 1 AND (a.best_move_equity_error >= 100 OR a.cube_error >= 100)`).Scan(&blunders, &marked); err != nil {
		t.Fatal(err)
	}
	if marked < 5 || blunders-marked < 5 {
		t.Fatalf("%d blunders, %d of them marked: the fixture cannot tell the two lists apart", blunders, marked)
	}

	corpus, commands := corpusFilters(t)
	var filters []domain.SearchFilters
	var names []string
	for i, f := range corpus {
		for _, v := range []struct {
			errors    string
			structure bool
			sort      string
		}{{"E>100", false, ""}, {"E120,400", true, ""}, {"E>150", false, "pip_asc"}} {
			g := f
			g.MoveErrorFilter, g.Sort = v.errors, v.sort
			if v.structure {
				g.Filter.Board = structure
			}
			filters = append(filters, g)
			names = append(names, fmt.Sprintf("%s + %s (structure %v, sort %q)", commands[i], v.errors, v.structure, v.sort))
		}
	}

	restore := sqlshared.SetBlunderBranches(false)
	rec.record("named")
	want := replay(ctx, s, filters, true)
	rec.record("")
	restore()

	answered := 0
	for _, w := range want {
		if !strings.Contains(w, "find [];") {
			answered++
		}
	}
	if answered < len(want)/8 {
		t.Fatalf("%d answers of %d find a position: the fixture holds too few blunders to tell two readings apart", answered, len(want))
	}

	for _, chunk := range []int{0, 3} {
		restoreChunk := func() {}
		if chunk > 0 {
			restoreChunk = sqlshared.SetSearchChunk(chunk)
		}
		rec.record("windows")
		got := replay(ctx, s, filters, chunk == 0)
		rec.record("")
		restoreChunk()
		for i := range got {
			if !strings.HasPrefix(want[i], got[i]) || (chunk == 0 && got[i] != want[i]) {
				t.Errorf("chunk %d, tenant %d, %s:\n  got  %s\n  want %s", chunk, 1+i%2, names[i/2], got[i], want[i])
			}
		}
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	windows, listed := 0, 0
	probes := map[string]int{}
	for _, st := range rec.stmts {
		if isProbe(st.sql) || strings.HasPrefix(st.sql, "EXPLAIN") {
			probes[st.phase]++
		}
		switch {
		case !isBlunderWindow(st.sql):
		case st.phase == "named":
			t.Fatalf("a blunder window was sent with the windows off: %s", oneLine(st.sql))
		default:
			windows++
			if strings.Contains(st.sql, "ANY(") && !strings.Contains(st.sql, "game_id") {
				listed++
			}
		}
	}
	if windows < len(filters) {
		t.Fatalf("%d blunder windows for %d filters: the move-error searches no longer read them", windows, len(filters))
	}
	t.Logf("%d blunders in tenant 1, %d of them marked; %d blunder windows, %d of them with a list of ids from another filter; probes and estimates: %v",
		blunders, marked, windows, listed, probes)
}
