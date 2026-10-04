package sqlshared

// A match-play analysis valued with another table than the library's current
// one is "MET différente" (ADR-0068): read, never written, and left out of
// every comparison. Money is never different — the table plays no part in it.
// Both clauses are pure SQL, so a change of current table changes what the
// next read retains without rewriting a row.

// matchIsMoney is the two ways money is spelt on a match row (CONTEXT.md).
const matchIsMoney = `(m.match_length <= 0 OR m.match_length >= 99999)`

// currentMETExpr is the id of the scope's current table, NULL for the
// built-in Kazaross-XG2.
func currentMETExpr(d Dialect, scope string) (string, []any) {
	tenant, args := d.TenantFilter("mt", scope)
	return `(SELECT mt.id FROM match_equity_table mt WHERE ` + tenant + ` AND mt.is_current = 1 ORDER BY mt.id LIMIT 1)`, args
}

// metComparableDecision keeps a decision whose analysis (alias a, its match
// m) was valued with the current table.
func metComparableDecision(d Dialect, scope string) (string, []any) {
	cur, args := currentMETExpr(d, scope)
	return `(` + matchIsMoney + ` OR a.met_id IS NOT DISTINCT FROM ` + cur + `)`, args
}

// metComparableMatch keeps a match (alias m) every analysis of which was
// valued with the current table. With the built-in table current, the
// different matches are those touching a tagged analysis, which the partial
// index idx_analysis_met lists without reading the others; with an imported
// table current, only the matches touching an analysis it tagged can
// qualify, and only they are checked for an untagged one.
func metComparableMatch(d Dialect, scope string) (string, []any) {
	cur1, a1 := currentMETExpr(d, scope)
	cur2, a2 := currentMETExpr(d, scope)
	cur3, a3 := currentMETExpr(d, scope)
	cur4, a4 := currentMETExpr(d, scope)
	const movesOf = ` FROM analysis a JOIN move mv ON mv.position_id = a.position_id JOIN game g ON g.id = mv.game_id`
	sql := `(` + matchIsMoney +
		` OR (m.id NOT IN (SELECT g.match_id` + movesOf + ` WHERE a.met_id IS NOT NULL AND a.met_id IS DISTINCT FROM ` + cur1 + `)` +
		` AND (` + cur2 + ` IS NULL OR (m.id IN (SELECT g.match_id` + movesOf + ` WHERE a.met_id = ` + cur3 + `)` +
		` AND NOT EXISTS (SELECT 1 FROM game g JOIN move mv ON mv.game_id = g.id JOIN analysis a ON a.position_id = mv.position_id` +
		` WHERE g.match_id = m.id AND a.met_id IS DISTINCT FROM ` + cur4 + `)))))`
	args := append(append(append(a1, a2...), a3...), a4...)
	return sql, args
}
