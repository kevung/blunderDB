package sqlshared

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Compute runs some fifteen passes over the same decisions. Each pass that
// joins position and analysis directly pays the I/O of those tables again —
// wide rows spread over a large file, cheap on a demo database and minutes on
// a corpus of millions of positions. Compute therefore copies, once, the
// columns the passes read for the rows its filter can reach into three
// session-local tables, and every pass joins those instead (selectionJoin).
//
// The copy holds a superset: the filter without its decision type and with the
// seat-agnostic player clause (both seats of every selected match, which the
// Snowie denominator needs). Every pass still applies its own WHERE clause, so
// the copy changes what is read, never what is counted. A pass that reads a
// column missing here fails with an SQL error rather than a wrong figure.

const (
	selPositionCols = "id, decision_type, cube_value, game_phase, game_type, match_length, score_1, score_2"
	selAnalysisCols = "position_id, analysis_depth, analysis_engine, best_cube_action, best_move_equity_error, cube_error, is_close_cube, is_forced, met_id"
	selMoveCols     = "id, position_id, game_id, cube_action, luck_mp, move_number, player"
)

// selectionJoin is statsBaseJoin with position, analysis and move replaced
// by their selected copies; game, match and tournament are small and stay.
const selectionJoin = `FROM stats_sel_p p
JOIN stats_sel_a a ON a.position_id = p.id
JOIN stats_sel_mv mv ON mv.position_id = p.id
JOIN game g ON g.id = mv.game_id
JOIN match m ON m.id = g.match_id
LEFT JOIN tournament t ON t.id = m.tournament_id`

var selectionTables = []string{"stats_sel_mv", "stats_sel_a", "stats_sel_p"}

// prefixed renders a column list qualified by alias.
func prefixed(alias, cols string) string {
	parts := strings.Split(cols, ", ")
	for i, c := range parts {
		parts[i] = alias + "." + c
	}
	return strings.Join(parts, ", ")
}

// materializeSelection fills the selection tables for filter and returns the
// FROM fragment that reads them. s.DB must be a single connection (an open
// transaction): the tables are temporary, visible to that session only.
func (s *StatsStore) materializeSelection(ctx context.Context, scope string, filter storage.StatsFilter) (string, error) {
	posCols := selPositionCols
	moveCols, analysisCols := selMoveCols, selAnalysisCols
	// The tenant predicate every pass opens with names p's tenant column when
	// the backend has one, and the action-label reads name mv's and a's
	// (ActionLabelFor); the copies must carry them.
	if _, tenantArgs := s.DB.TenantFilter("p", scope); len(tenantArgs) > 0 {
		posCols += ", tenant_id"
		moveCols += ", tenant_id"
		analysisCols += ", tenant_id"
	}

	s.dropSelection(ctx)
	superset := filter
	superset.DecisionType = -1
	where, args := s.buildBaseWhereClauseSeat(scope, superset, false)

	steps := []struct {
		query string
		args  []any
	}{
		// Created empty from the source tables so each column keeps its type
		// on both backends; filled by INSERT, which takes parameters where
		// CREATE TABLE AS does not on PostgreSQL.
		{`CREATE TEMP TABLE stats_sel_mv AS SELECT ` + prefixed("mv", moveCols) + ` FROM move mv WHERE 1 = 0`, nil},
		{`CREATE TEMP TABLE stats_sel_p AS SELECT ` + prefixed("p", posCols) + ` FROM position p WHERE 1 = 0`, nil},
		{`CREATE TEMP TABLE stats_sel_a AS SELECT ` + prefixed("a", analysisCols) + ` FROM analysis a WHERE 1 = 0`, nil},
		{`INSERT INTO stats_sel_mv SELECT DISTINCT ` + prefixed("mv", moveCols) + ` ` + statsBaseJoin + where, args},
		{`INSERT INTO stats_sel_p SELECT ` + prefixed("p", posCols) + ` FROM position p WHERE p.id IN (SELECT position_id FROM stats_sel_mv)`, nil},
		{`INSERT INTO stats_sel_a SELECT ` + prefixed("a", analysisCols) + ` FROM analysis a WHERE a.position_id IN (SELECT id FROM stats_sel_p)`, nil},
		{`CREATE INDEX stats_sel_p_id ON stats_sel_p (id)`, nil},
		{`CREATE INDEX stats_sel_a_pos ON stats_sel_a (position_id)`, nil},
		{`CREATE INDEX stats_sel_mv_pos ON stats_sel_mv (position_id)`, nil},
	}
	for _, st := range steps {
		if _, err := s.DB.Exec(ctx, st.query, st.args...); err != nil {
			return "", fmt.Errorf("stats selection: %w", err)
		}
	}
	return selectionJoin, nil
}

// dropSelection removes the selection tables. Errors are ignored: on a
// failed transaction the rollback has already removed them.
func (s *StatsStore) dropSelection(ctx context.Context) {
	for _, t := range selectionTables {
		_, _ = s.DB.Exec(ctx, `DROP TABLE IF EXISTS `+t)
	}
}
