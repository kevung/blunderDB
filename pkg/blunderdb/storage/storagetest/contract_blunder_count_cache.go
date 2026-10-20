package storagetest

import (
	"context"
	"fmt"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// RunBlunderCountCache holds the cached blunder count to the one promise that
// makes it worth having: after any write that could change the count — an
// import, an analysis rewritten or deleted, a play added, a position or a
// match deleted, the threshold moved — the counter answers what a full
// recount answers. The oracle deletes the cached row through exec, the
// backend's own connection, and counts again; the cache is primed before each
// gesture, so a gesture that failed to invalidate it shows as a disagreement.
// The row is reached through exec because no store exposes it: it is the
// library's settings table, metadata on SQLite, library_settings per tenant
// on PostgreSQL.
func RunBlunderCountCache(t *testing.T, s storage.Storage, exec sqlshared.Execer) {
	ctx := context.Background()
	md := s.Metadata()

	table, scoped := exec.LibrarySettingsTable()
	where, whereArgs := `key = ?`, []any{storage.LibrarySettingBlunderCountKey}
	if scoped {
		where = exec.ScopeColumn() + ` = ? AND ` + where
		whereArgs = append([]any{exec.ScopeArg("")}, whereArgs...)
	}
	cacheRow := func() (string, bool) {
		var v string
		if err := exec.QueryRow(ctx, `SELECT COALESCE(value,'') FROM `+table+` WHERE `+where, whereArgs...).Scan(&v); err != nil {
			return "", false
		}
		return v, true
	}
	blunders := func(when string) int {
		c, err := md.Counts(ctx, "")
		if err != nil {
			t.Fatalf("Counts %s: %v", when, err)
		}
		return c.Blunders
	}
	recount := func(when string) int {
		if _, err := exec.Exec(ctx, `DELETE FROM `+table+` WHERE `+where, whereArgs...); err != nil {
			t.Fatalf("drop the cache %s: %v", when, err)
		}
		return blunders(when)
	}
	check := func(gesture string) int {
		t.Helper()
		cached := blunders("after " + gesture)
		truth := recount("after " + gesture)
		if cached != truth {
			t.Fatalf("after %s the counter says %d blunders, a recount %d: the cache lied", gesture, cached, truth)
		}
		if _, ok := cacheRow(); !ok {
			t.Fatalf("after %s the count was not cached", gesture)
		}
		return truth
	}

	check("opening an empty library")

	// The cache is what the counter reads: a forged row is believed. Without
	// this the oracle below could pass on a counter that never caches.
	settings, err := s.LibrarySettings().Load(ctx, "")
	if err != nil {
		t.Fatalf("Load settings: %v", err)
	}
	if _, err := exec.Exec(ctx, `UPDATE `+table+` SET value = ? WHERE `+where,
		append([]any{fmt.Sprintf("%d:%d", settings.BlunderThresholdMP, 999)}, whereArgs...)...); err != nil {
		t.Fatalf("forge the cache: %v", err)
	}
	if got := blunders("with a forged cache"); got != 999 {
		t.Fatalf("Blunders = %d with the cache saying 999: the counter does not read its cache", got)
	}
	if est, err := md.EstimatedCounts(ctx, "", 0); err != nil || !est.BlundersKnown || est.Blunders != 999 {
		t.Fatalf("EstimatedCounts with every table estimated = %+v, %v; want the cached 999", est, err)
	}
	recount("after forging the cache")

	matchID, fixture := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	check("importing a match")

	game := domain.Game{MatchID: matchID, GameNumber: 2, Winner: 1, PointsWon: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &game)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	pos := statsDecisionPos(t, 4)
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	check("saving a position")

	mv := domain.Move{GameID: gameID, MoveNumber: 20, MoveType: "checker",
		PositionID: posID, Player: 1, CheckerMove: "13/11 24/23"}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	analysisAt := func(dearCost float64) *domain.PositionAnalysis {
		cheap := 0.020
		return &domain.PositionAnalysis{
			AnalysisType: "CheckerMove",
			PlayedMoves:  []string{"13/11 24/23", "8/6 6/4"},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "24/22 13/11", Equity: 0.500},
				{Move: "13/11 24/23", Equity: 0.480, EquityError: &cheap},
				{Move: "8/6 6/4", Equity: 0.500 - dearCost, EquityError: &dearCost},
			}},
		}
	}
	if err := s.Analyses().Save(ctx, "", posID, analysisAt(0.300)); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	check("analysing a position played once, cheaply")

	// The second play is the dear one: only the move table says the
	// position is a Blunder, the denormalised column still scores the first.
	mv2 := domain.Move{GameID: gameID, MoveNumber: 21, MoveType: "checker",
		PositionID: posID, Player: 1, CheckerMove: "8/6 6/4"}
	if _, err := s.Matches().CreateMove(ctx, "", &mv2); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	if n := check("a second, dear play of the same position"); n != 1 {
		t.Fatalf("Blunders = %d after the dear play, want 1", n)
	}

	if err := s.Analyses().Save(ctx, "", posID, analysisAt(0.030)); err != nil {
		t.Fatalf("rewrite analysis: %v", err)
	}
	if n := check("rewriting the analysis below the threshold"); n != 0 {
		t.Fatalf("Blunders = %d after the rewrite, want 0", n)
	}
	if err := s.Analyses().Save(ctx, "", posID, analysisAt(0.300)); err != nil {
		t.Fatalf("rewrite analysis: %v", err)
	}
	check("rewriting the analysis above the threshold")

	if err := s.LibrarySettings().Save(ctx, "", storage.LibrarySettings{
		ErrorThresholdMP: 20, BlunderThresholdMP: 40}); err != nil {
		t.Fatalf("Save settings: %v", err)
	}
	if n := check("lowering the blunder threshold"); n < 2 {
		t.Fatalf("Blunders = %d at a 40 mp threshold, want the fixture's 50 mp decisions too", n)
	}
	if err := s.LibrarySettings().Save(ctx, "", settings); err != nil {
		t.Fatalf("restore settings: %v", err)
	}
	check("restoring the threshold")

	if err := s.Analyses().Delete(ctx, "", posID); err != nil {
		t.Fatalf("Delete analysis: %v", err)
	}
	check("deleting an analysis")
	if err := s.Analyses().Save(ctx, "", posID, analysisAt(0.300)); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	check("analysing it again")

	loaded, err := s.Positions().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load position: %v", err)
	}
	if err := s.Positions().Update(ctx, "", loaded); err != nil {
		t.Fatalf("Update position: %v", err)
	}
	check("updating a position")

	if err := s.Positions().Delete(ctx, "", fixture[0]); err != nil {
		t.Fatalf("Delete position: %v", err)
	}
	check("deleting a position")

	if _, err := s.Matches().DeleteGames(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteGames: %v", err)
	}
	check("deleting a match's games")

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}
	check("deleting a match")

	if err := s.Positions().Delete(ctx, "", posID); err != nil {
		t.Fatalf("Delete position: %v", err)
	}
	if n := check("deleting the last blunder"); n != 0 {
		t.Fatalf("Blunders = %d with no analysed position left, want 0", n)
	}
}
