package sqlshared

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Compute runs some fifteen passes over the same decisions. Each pass that
// joins position, analysis, move and game pays that join again — four index
// lookups per decision, cheap on a demo database and minutes on a corpus of
// millions of positions. Compute therefore performs the join once, into a
// single session-local table holding, per decision, the columns the passes
// read of those four tables (selectionCols), and every pass scans that table,
// joined only to match and tournament: small tables, and the only ones holding
// text, which the copy would repeat on every decision.
//
// The passes keep their SQL as written against statsBaseJoin's aliases:
// selectionExecer rewrites p., a., mv. and g. columns into the flat table's
// alias_column on the queries that read it, so a pass has one text, whichever
// source it runs on. A column missing from selectionCols fails with
// an SQL error rather than a wrong figure.
//
// The copy holds a superset: the filter without its decision type and with the
// seat-agnostic player clause (both seats of every selected match, which the
// Snowie denominator needs). Every pass still applies its own WHERE clause, so
// the copy changes what is read, never what is counted.

// selectionCols are the columns of statsBaseJoin the passes read, by alias.
var selectionCols = []struct{ alias, cols string }{
	{"p", "id, decision_type, cube_value, game_phase, game_type, match_length, score_1, score_2"},
	{"a", "position_id, analysis_depth, analysis_engine, best_cube_action, best_move_equity_error, cube_error, is_close_cube, is_forced, met_id"},
	{"mv", "id, position_id, game_id, cube_action, luck_mp, move_number, player"},
	{"g", "id, match_id"},
}

// selectionTable is the flat copy; selectionJoin the FROM clause that reads it.
const (
	selectionTable = "stats_sel_d"
	selectionJoin  = "FROM " + selectionTable + " d" + `
JOIN match m ON m.id = d.g_match_id
LEFT JOIN tournament t ON t.id = m.tournament_id`
)

var selectionTables = []string{selectionTable}

// selectionColumn matches a column read through one of the copied tables'
// aliases; the word boundary keeps al.code, ms.seat or m.id out.
var selectionColumn = regexp.MustCompile(`\b(mv|p|a|g)\.([A-Za-z_][A-Za-z0-9_]*)`)

// onSelection renders query against the flat table when it reads it, and
// returns it unchanged otherwise (the match_stats passes share the handle).
func onSelection(query string) string {
	if !strings.Contains(query, selectionJoin) {
		return query
	}
	return selectionColumn.ReplaceAllString(query, "d.${1}_${2}")
}

// selectionExecer is the transaction the passes run on, with every query that
// reads the selection rewritten by onSelection.
type selectionExecer struct{ Execer }

func (e selectionExecer) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	return e.Execer.Exec(ctx, onSelection(query), args...)
}

func (e selectionExecer) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	return e.Execer.Query(ctx, onSelection(query), args...)
}

func (e selectionExecer) QueryRow(ctx context.Context, query string, args ...any) Row {
	return e.Execer.QueryRow(ctx, onSelection(query), args...)
}

// materializeSelection fills the selection table for filter and returns the
// FROM fragment that reads it. s.DB must be a single connection (an open
// transaction): the table is temporary, visible to that session only. Queries
// reading the fragment must go through selectionExecer.
func (s *StatsStore) materializeSelection(ctx context.Context, scope string, filter storage.StatsFilter) (string, error) {
	cols := selectionCols
	// The tenant predicate every pass opens with names p's tenant column when
	// the backend has one, and the action-label reads name mv's and a's
	// (ActionLabelFor); the copy must carry them.
	if _, tenantArgs := s.DB.TenantFilter("p", scope); len(tenantArgs) > 0 {
		cols = append(cols[:len(cols):len(cols)],
			struct{ alias, cols string }{"p", "tenant_id"},
			struct{ alias, cols string }{"a", "tenant_id"},
			struct{ alias, cols string }{"mv", "tenant_id"})
	}
	var sel []string
	for _, c := range cols {
		for _, col := range strings.Split(c.cols, ", ") {
			sel = append(sel, c.alias+"."+col+" AS "+c.alias+"_"+col)
		}
	}
	selectList := strings.Join(sel, ", ")

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
		{`CREATE TEMP TABLE ` + selectionTable + ` AS SELECT ` + selectList + ` ` + statsBaseJoin + ` WHERE 1 = 0`, nil},
		{`INSERT INTO ` + selectionTable + ` SELECT ` + selectList + ` ` + statsBaseJoin + where, args},
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
