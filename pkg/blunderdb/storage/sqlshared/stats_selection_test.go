package sqlshared

import "testing"

// onSelection rewrites only the columns read through statsBaseJoin's aliases,
// and only on a query that reads the selection.
func TestOnSelectionRewritesBaseAliasesOnly(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{
			"SELECT mv.player, m.id, t.name, p.decision_type " + selectionJoin + " WHERE a.met_id IS NOT DISTINCT FROM (SELECT mt.id FROM match_equity_table mt)",
			"SELECT d.mv_player, m.id, t.name, d.p_decision_type " + selectionJoin + " WHERE d.a_met_id IS NOT DISTINCT FROM (SELECT mt.id FROM match_equity_table mt)",
		},
		{
			"SELECT al.label FROM action_label al WHERE al.code = a.best_cube_action AND p.id IN (SELECT c.position_id FROM comment c) " + selectionJoin,
			"SELECT al.label FROM action_label al WHERE al.code = d.a_best_cube_action AND d.p_id IN (SELECT c.position_id FROM comment c) " + selectionJoin,
		},
		// A match_stats pass shares the handle and is left alone.
		{"SELECT m.id, ms.seat FROM match_stats ms JOIN match m ON m.id = ms.match_id", "SELECT m.id, ms.seat FROM match_stats ms JOIN match m ON m.id = ms.match_id"},
	} {
		if got := onSelection(c.in); got != c.want {
			t.Errorf("onSelection(%q)\n got  %q\n want %q", c.in, got, c.want)
		}
	}
}
