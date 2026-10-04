package sqlshared

import (
	"context"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestBuildWhereCubeResponseAloneImpliesCubeDecision(t *testing.T) {
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{CubeResponseFilter: "double"})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if !strings.Contains(wc.where, "p.decision_type = ? AND p.is_cube_response = 0") {
		t.Errorf("where = %q, want a cube decision that is no response", wc.where)
	}
	if len(wc.args) != 1 || wc.args[0] != domain.CubeAction {
		t.Errorf("args = %v, want [CubeAction]", wc.args)
	}
}

func TestBuildWhereMatchDateIsHalfOpenOnTheLastDay(t *testing.T) {
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{MatchDateFilter: "md:2024-01..2024-12"})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if !strings.Contains(wc.where, "p.match_date >= ? AND p.match_date < ?") {
		t.Errorf("where = %q, want a half-open interval on position.match_date", wc.where)
	}
	// Day bounds are midnight UTC, bound as Unix seconds (ADR-0071).
	if len(wc.args) != 2 || wc.args[0] != int64(1704067200) || wc.args[1] != int64(1735689600) {
		t.Errorf("args = %v, want [1704067200 1735689600] (2024-01-01 and 2025-01-01 UTC)", wc.args)
	}
}

func TestBuildWhereMatchLevelTokensShareOneSubquery(t *testing.T) {
	wc, err := buildWhereStore().buildWhere(context.Background(), "", domain.SearchFilters{
		PlayerFilter: `pl!"Alice"`, OpponentFilter: `op"Bob"`, RoundFilter: "3", PlayerPRFilter: "pr>8",
	})
	if err != nil {
		t.Fatalf("buildWhere: %v", err)
	}
	if n := strings.Count(wc.where, "FROM move mv"); n != 1 {
		t.Errorf("where = %q, want the match-level tokens in one subquery, got %d", wc.where, n)
	}
	if !strings.Contains(wc.where, "ms.seat = (CASE WHEN mv.player = 1 THEN 1 ELSE 2 END)") {
		t.Errorf("where = %q, want the PR read at the seat of the player who decided", wc.where)
	}
}
