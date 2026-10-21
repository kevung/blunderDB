package sqlshared

import "context"

// MultiPlayedSQL is the definition of position.multi_played for the
// position whose id is posExpr: player 1 recorded at least two distinct plays
// on it in the move table (a checker move or a cube action, compared as
// stored) — a position deduplicated across matches and played differently
// each time. On such a position the denormalised error column, which scores
// one play, cannot answer a move-error filter; the search lets it through to
// the Go re-check. A move written two ways counts twice here; the re-check
// normalises, so the cost is one needless decode, never a wrong answer.
//
// LIMIT 2 stops the scan at the second distinct play: a position reached
// thousands of times is not read whole. The result is 0 or 1. The SQLite
// triggers and migration step evaluate it; the PostgreSQL ones
// (migrations/052_multi_played.sql) write the same test with a tenant.
func MultiPlayedSQL(posExpr string) string {
	return `((SELECT COUNT(*) FROM (SELECT DISTINCT COALESCE(mp.checker_move, ''), ` + ActionCodeOrEmptySQL("mp.cube_action") +
		` FROM move mp WHERE mp.position_id = ` + posExpr + ` AND mp.player = 1 LIMIT 2) mp2) > 1)`
}

// MultiPlayedDrift lists the positions of scope whose multi_played indicator
// differs from a direct count of their moves: empty when the triggers kept
// it true. The count is written apart from MultiPlayedSQL (a grouped pass
// rather than a per-position probe), so a fault in one does not hide in the
// other. Only the contract suite calls it, through an optional interface.
func (s *SearchStore) MultiPlayedDrift(ctx context.Context, scope string) ([]int64, error) {
	pTenant, pArgs := s.DB.TenantFilter("p", scope)
	mTenant, mArgs := s.DB.TenantFilter("m", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT p.id FROM position p WHERE `+pTenant+` AND (`+s.DB.Bool("p.multi_played", true)+`) <> (p.id IN (
			SELECT m.position_id FROM move m
			 WHERE `+mTenant+` AND m.player = 1 AND m.position_id IS NOT NULL
			 GROUP BY m.position_id
			HAVING COUNT(DISTINCT COALESCE(m.checker_move, '') || '|' || `+ActionCodeOrEmptySQL("m.cube_action")+`) > 1))
		 ORDER BY p.id`,
		append(pArgs, mArgs...)...)
	if err != nil {
		return nil, errf(s.DB, "multi-played drift", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, errf(s.DB, "multi-played drift scan", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "multi-played drift rows", err)
	}
	return ids, nil
}
