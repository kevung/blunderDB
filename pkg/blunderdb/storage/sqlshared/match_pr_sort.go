package sqlshared

// match_pr_sort.go — the match list ordered by a PR. A PR is no column of
// match: it is read from the seat rows of match_stats, the source of the
// list's badges, so the order is the one the badges show, over the whole
// library rather than over the page on screen.

import (
	"context"
	"strconv"
	"strings"
)

// MatchPROrder is what a match list query adds for a PR sort: a LEFT JOIN
// placed after `FROM match m …`, whose args precede the WHERE's, and an
// ORDER BY body whose args follow the WHERE's. Both are written with `?`
// placeholders; a backend numbering its own ($n) renumbers them with
// RebindFrom.
type MatchPROrder struct {
	Join      string
	JoinArgs  []any
	OrderBy   string
	OrderArgs []any
}

// prSortKeys maps each PR sort key to the PR it reads and whether it
// descends. "main" is the library's main player, wherever he sat, aliases
// included; "opponent" is the other seat of his matches.
var prSortKeys = map[string]struct {
	side string
	desc bool
}{
	"pr1": {"1", false}, "pr1_asc": {"1", false}, "pr1_desc": {"1", true},
	"pr2": {"2", false}, "pr2_asc": {"2", false}, "pr2_desc": {"2", true},
	"pr_main": {"main", false}, "pr_main_asc": {"main", false}, "pr_main_desc": {"main", true},
	"pr_opponent": {"opponent", false}, "pr_opponent_asc": {"opponent", false}, "pr_opponent_desc": {"opponent", true},
}

// IsMatchPRSort reports whether sort is a PR key MatchPRSort serves.
func IsMatchPRSort(sort string) bool {
	_, ok := prSortKeys[sort]
	return ok
}

// MatchPRSort builds the join and order of a match list sorted by a PR; ok
// is false when sort is no PR key.
//
// The PR is the badge's: the seat's error over its counted decisions. A
// match without one — no counted decision on that seat, or, for main and
// opponent, a match the main player did not play or a library without a
// main player — sorts last in both directions. m.id breaks ties in the
// direction of the sort, so the order is total and a page boundary never
// moves between two reads.
//
// match_stats is filled first when the connection can write, as the badges
// fill it; a reader that cannot, or loses the file to another writer, reads
// the same seat rows computed from the decisions.
func MatchPRSort(ctx context.Context, db Execer, scope, sort string) (MatchPROrder, bool, error) {
	key, ok := prSortKeys[sort]
	if !ok {
		return MatchPROrder{}, false, nil
	}
	var spellings []string
	if key.side == "main" || key.side == "opponent" {
		main, _, err := mainPlayer(ctx, db, scope)
		if err != nil {
			return MatchPROrder{}, true, errf(db, "match PR sort: main player", err)
		}
		if spellings, err = PlayerSpellings(ctx, db, scope, main); err != nil {
			return MatchPROrder{}, true, errf(db, "match PR sort: aliases", err)
		}
	}

	stats := &StatsStore{DB: db}
	readable, err := stats.matchStatsReadable(ctx, scope)
	if err != nil && !LockContention(db, err) {
		return MatchPROrder{}, true, err
	}
	var src string
	var out MatchPROrder
	if readable {
		tenant, targs := db.TenantFilter("ms", scope)
		src = `match_stats ms WHERE ` + tenant
		out.JoinArgs = targs
	} else {
		sub, args := matchStatsDerived(db, scope)
		src = sub + ` ms`
		out.JoinArgs = args
	}
	prExpr := `CASE WHEN ms.decisions > 0 THEN CAST(ms.error_mp AS DOUBLE PRECISION) / ms.decisions END`
	out.Join = ` LEFT JOIN (SELECT ms.match_id AS match_id,
		MAX(CASE WHEN ms.seat = 1 THEN ` + prExpr + ` END) AS pr1,
		MAX(CASE WHEN ms.seat = 2 THEN ` + prExpr + ` END) AS pr2
		FROM ` + src + ` GROUP BY ms.match_id) prs ON prs.match_id = m.id`

	var expr string
	switch key.side {
	case "1":
		expr = `prs.pr1`
	case "2":
		expr = `prs.pr2`
	default:
		if len(spellings) == 0 {
			expr = `CAST(NULL AS DOUBLE PRECISION)`
			break
		}
		mine, theirs := `prs.pr1`, `prs.pr2`
		if key.side == "opponent" {
			mine, theirs = theirs, mine
		}
		in := func() string {
			p := make([]string, len(spellings))
			for i, n := range spellings {
				p[i] = "?"
				out.OrderArgs = append(out.OrderArgs, n)
			}
			return strings.Join(p, ",")
		}
		expr = `CASE WHEN m.player1_name IN (` + in() + `) THEN ` + mine +
			` WHEN m.player2_name IN (` + in() + `) THEN ` + theirs + ` END`
	}
	if key.desc {
		out.OrderBy = expr + ` DESC NULLS LAST, m.id DESC`
	} else {
		out.OrderBy = expr + ` ASC NULLS LAST, m.id ASC`
	}
	return out, true, nil
}

// RebindFrom numbers the `?` placeholders of query $first, $first+1, … in
// order: a fragment placed after the first first-1 args of a query.
func RebindFrom(query string, first int) string {
	if !strings.Contains(query, "?") {
		return query
	}
	var b strings.Builder
	n := first
	for _, part := range strings.SplitAfter(query, "?") {
		if strings.HasSuffix(part, "?") {
			b.WriteString(part[:len(part)-1])
			b.WriteString("$" + strconv.Itoa(n))
			n++
			continue
		}
		b.WriteString(part)
	}
	return b.String()
}
