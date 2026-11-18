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

// readsMatchRest spots the window of a match-level search read from the
// positions its other filters keep, each checked against its own moves.
func readsMatchRest(sql string) bool {
	return strings.Contains(sql, "p.id = ANY(") && strings.Contains(sql, "JOIN game g ON mv.game_id = g.id") &&
		strings.Contains(sql, "mv.position_id = p.id LIMIT 1 OFFSET 0")
}

// A player, an opponent or a match length joined to the filters of the corpus
// — a cube, dice, a score, filters that keep nothing — in each writing of the
// match-level filter (sorted moves, a check per position, a subquery): read
// from the positions the other filters keep, the search answers what it
// answered without that read, with the read thrown away when they keep more
// than its limit. Same ids, same order, same pages, same count, in each tenant.
func TestMatchLevelSearchWithARareFilterAnswersTheSame(t *testing.T) {
	ctx := context.Background()
	s, rec, dsn := probeFixture(ctx, t)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	var player, opponent string
	var length int
	if err := conn.QueryRow(ctx, `SELECT player1_name, player2_name, match_length FROM match WHERE tenant_id = 1 ORDER BY id LIMIT 1`).Scan(&player, &opponent, &length); err != nil {
		t.Fatal(err)
	}
	// The walk of eight rows comes short: the read of the other filters is reached.
	defer sqlshared.SetIDWalkProbe(8)()

	corpus, commands := corpusFilters(t)
	var filters []domain.SearchFilters
	var names []string
	for i := range corpus {
		for _, suffix := range []string{fmt.Sprintf(`pl"%s"`, player), fmt.Sprintf(`ml:%d`, length), fmt.Sprintf(`op"%s"`, opponent)} {
			g, _ := searchquery.Parse(commands[i] + " " + suffix)
			filters = append(filters, g)
			names = append(names, commands[i]+" "+suffix)
		}
	}

	for _, mode := range []struct {
		name    string
		setting func() func()
	}{
		{"sorted moves", func() func() { return sqlshared.SetMatchListLimit(10) }},
		{"check per position", func() func() { return sqlshared.SetNarrowLimit(0) }},
		{"subquery", func() func() { return sqlshared.SetNarrowLimit(-1) }},
	} {
		restoreMode := mode.setting()
		restoreRest := sqlshared.SetMatchRestLimit(0)
		rec.record(mode.name + " off")
		want := replay(ctx, s, filters, true)
		rec.record("")
		restoreRest()
		answered := 0
		for _, w := range want {
			if !strings.Contains(w, "find [];") {
				answered++
			}
		}
		if answered < len(want)/8 || answered == len(want) {
			t.Fatalf("%s: %d answers of %d find a position: the fixture cannot tell two readings apart", mode.name, answered, len(want))
		}
		for _, limit := range []int{3, 10000} {
			restoreRest := sqlshared.SetMatchRestLimit(limit)
			rec.record(fmt.Sprintf("%s %d", mode.name, limit))
			got := replay(ctx, s, filters, true)
			rec.record("")
			restoreRest()
			for i := range got {
				if got[i] != want[i] {
					t.Errorf("%s, limit %d, tenant %d, %s:\n  got  %s\n  want %s", mode.name, limit, 1+i%2, names[i/2], got[i], want[i])
				}
			}
		}
		restoreMode()
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	reads := map[string]int{}
	for _, st := range rec.stmts {
		if readsMatchRest(st.sql) {
			reads[st.phase]++
		}
	}
	t.Logf("windows read from the other filters: %v", reads)
	for _, mode := range []string{"sorted moves", "check per position", "subquery"} {
		switch {
		case reads[mode+" off"] > 0:
			t.Errorf("%s: a window was read from the other filters with the read off", mode)
		case reads[mode+" 10000"] < len(filters)/4:
			t.Errorf("%s: %d windows read from the other filters for %d searches: the read is no longer reached", mode, reads[mode+" 10000"], len(filters))
		case reads[mode+" 3"] >= reads[mode+" 10000"]:
			t.Errorf("%s: %d windows under a limit of 3, %d under 10000: a read beyond its limit must be thrown away", mode, reads[mode+" 3"], reads[mode+" 10000"])
		}
	}
}
