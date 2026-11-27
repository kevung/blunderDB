package sqlshared

import (
	"context"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/searchfilter"
)

// movePatternCond is the condition on the best_move column col that every
// analysis matching patterns meets: a superset, settled in Go by
// searchfilter.AnalysisMatchesMovePattern. A NULL is a row not derived yet
// (written before the column, or by an older binary) and goes through; a cube
// pattern lets through the analyses with no checker play, an empty string
// (engine.BestMove).
// ok is false when a pattern matches every checker play: nothing to narrow.
func movePatternCond(col string, patterns []string) (cond string, args []any, ok bool) {
	var b strings.Builder
	b.WriteString("(" + col + " IS NULL")
	cube := false
	for _, p := range patterns {
		if p == "" {
			return "", nil, false
		}
		cube = cube || searchfilter.IsCubeMovePattern(p)
		b.WriteString(" OR " + col + " LIKE ?" + likeEscape)
		args = append(args, ContainsPattern(p))
	}
	if cube {
		b.WriteString(" OR " + col + " = ''")
	}
	b.WriteString(")")
	return b.String(), args, true
}

// appendMovePatternClause narrows a move-pattern search in SQL, so that the
// blob of an analysis that cannot match is neither read nor decoded.
//
// Under row-level security LIKE is not leakproof, so no index can answer it
// (a trigram index included): PostgreSQL reads the tenant's best moves from
// idx_analysis_best_move alone, short strings, never the heap with its blobs.
// A rare pattern is read that way first, as a list of ids; a common one is
// met soon by the walk, as a condition on the joined row. SQLite has no index
// on the column: the condition goes on the joined row, the row the walk reads.
func (s *SearchStore) appendMovePatternClause(ctx context.Context, scope string, f domain.SearchFilters, where *strings.Builder, args *[]any) error {
	patterns := searchfilter.MovePatterns(f.MovePatternFilter)
	joined, joinedArgs, ok := movePatternCond("a.best_move", patterns)
	if !ok {
		return nil
	}
	onRow := func() {
		where.WriteString(" AND a.position_id IS NOT NULL AND " + joined)
		*args = append(*args, joinedArgs...)
	}
	if s.DB.Name() == "sqlite" {
		onRow()
		return nil
	}
	cond, condArgs, _ := movePatternCond("best_move", patterns)
	own, ownArgs := s.DB.TenantFilter("", scope)
	sub := "SELECT position_id FROM analysis WHERE " + own + " AND " + cond
	return s.narrowToIDs(ctx, sub, append(append([]any{}, ownArgs...), condArgs...), where, args, onRow)
}
