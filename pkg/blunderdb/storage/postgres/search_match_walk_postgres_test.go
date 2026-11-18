//go:build postgres

package postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// isMatchWalk spots a walk that checks each position against its own moves;
// sortsMoves, a statement that reads and sorts the moves of the games.
func isMatchWalk(sql string) bool { return strings.Contains(sql, "mv.game_id = ANY('{") }
func sortsMoves(sql string) bool  { return strings.Contains(sql, "LIMIT 2147483647 OFFSET 0") }

// A player, an opponent or a match length whose moves are too many for a
// list: the window is first asked of a bounded walk that checks each position
// against its own moves, and the moves are read and sorted only when that
// walk comes short. Whatever the walk reads — eight rows, so that it nearly
// always comes short, or the whole tenant — alone, with a structure, a move
// error from a blunder up or a sort key, the search answers what the sorted
// moves answer: same ids, same order, same pages, same count, in each tenant.
func TestMatchLevelSearchAnswersTheSameThroughAWalk(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	structure := madePoints(ctx, t, s)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	// One position in two is a blunder, one in seven is played several ways:
	// a walk of eight blunders ends before the positions played several ways
	// have filled a window, and blunders it has not read stand between them.
	for _, change := range []string{
		`UPDATE position SET multi_played = TRUE WHERE id % 7 = 0`,
		`UPDATE analysis SET best_move_equity_error = 500, cube_error = 500 WHERE position_id % 2 = 0`,
	} {
		if _, err := conn.Exec(ctx, change); err != nil {
			t.Fatal(err)
		}
	}
	var player, opponent string
	var length int
	if err := conn.QueryRow(ctx, `SELECT player1_name, player2_name, match_length FROM match WHERE tenant_id = 1 ORDER BY id LIMIT 1`).Scan(&player, &opponent, &length); err != nil {
		t.Fatal(err)
	}
	// A single match has too many moves for a list of ten.
	defer sqlshared.SetMatchListLimit(10)()

	corpus, commands := corpusFilters(t)
	var filters []domain.SearchFilters
	var names []string
	for i := range corpus {
		// One search in two of the third kind is sorted by a key: no walk is asked of it.
		sort := []string{"", "pip_asc"}[i%2]
		for _, v := range []struct {
			suffix    string
			structure bool
			sort      string
		}{
			{fmt.Sprintf(`pl"%s"`, player), false, ""},
			{fmt.Sprintf(`ml:%d E>100`, length), false, ""},
			{fmt.Sprintf(`op"%s"`, opponent), true, sort},
		} {
			g, _ := searchquery.Parse(commands[i] + " " + v.suffix)
			g.Sort = v.sort
			if v.structure {
				g.Filter.Board = structure
			}
			filters = append(filters, g)
			names = append(names, fmt.Sprintf("%s %s (structure %v, sort %q)", commands[i], v.suffix, v.structure, v.sort))
		}
	}

	restore := sqlshared.SetMatchProbes(false)
	rec.record("sorted")
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
		t.Fatalf("%d answers of %d find a position: the fixture cannot tell two readings apart", answered, len(want))
	}

	for _, run := range []struct{ rows, chunk int }{{8, 0}, {20000, 0}, {20000, 3}} {
		restoreProbe := sqlshared.SetIDWalkProbe(run.rows)
		restoreChunk := func() {}
		if run.chunk > 0 {
			restoreChunk = sqlshared.SetSearchChunk(run.chunk)
		}
		rec.record(fmt.Sprintf("walk %d/%d", run.rows, run.chunk))
		got := replay(ctx, s, filters, run.chunk == 0)
		rec.record("")
		restoreChunk()
		restoreProbe()
		for i := range got {
			if !strings.HasPrefix(want[i], got[i]) || (run.chunk == 0 && got[i] != want[i]) {
				t.Errorf("walk of %d rows, chunk %d, tenant %d, %s:\n  got  %s\n  want %s", run.rows, run.chunk, 1+i%2, names[i/2], got[i], want[i])
			}
		}
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	walks, sorts, bounded := map[string]int{}, map[string]int{}, 0
	for _, st := range rec.stmts {
		if isMatchWalk(st.sql) {
			walks[st.phase]++
			if isBlunderWindow(st.sql) {
				bounded++
			}
		}
		if sortsMoves(st.sql) && strings.Contains(st.sql, "ORDER BY p.id") {
			sorts[st.phase]++
		}
	}
	t.Logf("walks %v (%d over the blunder index), windows read from sorted moves %v", walks, bounded, sorts)
	switch {
	case walks["sorted"] > 0:
		t.Fatal("a walk was sent with the walks off")
	case sorts["sorted"] < len(filters):
		t.Fatalf("%d windows read from sorted moves for %d filters: the fixture no longer reaches the sorted moves", sorts["sorted"], len(filters))
	case walks["walk 8/0"] < len(filters) || walks["walk 20000/0"] < len(filters) || bounded == 0:
		t.Fatalf("walks %v, %d over the blunder index, for %d filters: the match-level searches no longer walk first", walks, bounded, len(filters))
	case sorts["walk 8/0"] == 0 || sorts["walk 20000/0"] >= sorts["walk 8/0"]:
		t.Fatalf("windows read from sorted moves %v: a short walk must fall back on them, a full one must not", sorts)
	}
}
