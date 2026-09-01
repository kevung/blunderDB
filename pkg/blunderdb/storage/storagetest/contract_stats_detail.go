// Contract cases for statistics: per-match detail, the player table and the
// position-id lookups by match and tournament.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsMatchDetail exercises MatchDetail on both backends: only the
// PostgreSQL side had a dedicated test (via the fixture-driven parity gate in
// stats_parity_postgres_test.go, which compares real XG imports against the
// legacy Database — a different and heavier kind of coverage), so SQLite had
// none in the storage layer itself.
func testStatsMatchDetail(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchID, _ := statsFixtureMatch(t, s, 0, "Alice", "Bob")

	detail, err := s.Stats().MatchDetail(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchDetail: %v", err)
	}
	if detail.MatchID != matchID {
		t.Errorf("MatchID: got %d, want %d", detail.MatchID, matchID)
	}
	if detail.Player1.TotalDecisions != 1 || detail.Player2.TotalDecisions != 1 {
		t.Errorf("TotalDecisions: player1=%d player2=%d, want 1 and 1",
			detail.Player1.TotalDecisions, detail.Player2.TotalDecisions)
	}
	if detail.Player1.PR <= 0 || detail.Player2.PR <= 0 {
		t.Errorf("PR: player1=%v player2=%v, want both > 0 (a real equity error was recorded)",
			detail.Player1.PR, detail.Player2.PR)
	}

	// A match with no counted decisions returns zero-valued stats, not an
	// error: MatchDetail must be safe to call on every match in a list.
	empty := domain.Match{Player1Name: "Nobody", Player2Name: "Nowhere"}
	emptyID, err := s.Matches().Save(ctx, "", &empty)
	if err != nil {
		t.Fatalf("Save empty match: %v", err)
	}
	emptyDetail, err := s.Stats().MatchDetail(ctx, "", emptyID)
	if err != nil {
		t.Fatalf("MatchDetail(empty): %v", err)
	}
	if emptyDetail.Player1.TotalDecisions != 0 || emptyDetail.Player2.TotalDecisions != 0 {
		t.Errorf("MatchDetail(empty): got %+v, want zero decisions", emptyDetail)
	}
}

// testStatsPlayerTable covers the table behind the Stats panel's Players tab on
// both backends: one row per player NAME, the figures that row carries, and the
// order they arrive in.
func testStatsPlayerTable(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}

	// Two matches sharing one player, so Alice's row has to aggregate across
	// matches while Bob's and Carol's stay separate.
	statsFixtureMatch(t, s, 0, "Alice", "Bob")
	statsFixtureMatch(t, s, 2, "Alice", "Carol")

	rows, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerTable: %v", err)
	}
	byName := map[string]storage.PlayerRow{}
	for _, r := range rows {
		if _, dup := byName[r.Name]; dup {
			t.Errorf("player %q appears on two rows; one row per name", r.Name)
		}
		byName[r.Name] = r
	}
	if len(byName) != 3 {
		t.Fatalf("players in the table: got %d (%v), want 3", len(byName), byName)
	}

	if got := byName["Alice"].Matches; got != 2 {
		t.Errorf("Alice's matches: got %d, want 2", got)
	}
	if got := byName["Bob"].Matches; got != 1 {
		t.Errorf("Bob's matches: got %d, want 1", got)
	}
	// Each fixture match gives each player one counted checker decision, so
	// Alice — playing twice — has twice as many as her opponents.
	if got := byName["Alice"].Decisions; got != 2 {
		t.Errorf("Alice's counted decisions: got %d, want 2", got)
	}
	if got := byName["Bob"].Decisions; got != 1 {
		t.Errorf("Bob's counted decisions: got %d, want 1", got)
	}
	if a := byName["Alice"]; a.PR <= 0 || a.PRChecker <= 0 {
		t.Errorf("Alice's PR: got %v (checker %v), want both > 0 — the fixture records real errors",
			a.PR, a.PRChecker)
	}
	if a := byName["Alice"]; a.CheckerDecisions != a.Decisions {
		t.Errorf("the fixture holds checker decisions only: got %d of %d",
			a.CheckerDecisions, a.Decisions)
	}

	// The filter's player selection is ignored by design: this table is about
	// everyone. Asking for one player must not shrink it to one row.
	filtered, err := s.Stats().PlayerTable(ctx, "",
		storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("PlayerTable(PlayerName): %v", err)
	}
	if len(filtered) != len(rows) {
		t.Errorf("PlayerTable with a player filter: got %d rows, want the same %d — "+
			"the players table ignores the player selection", len(filtered), len(rows))
	}

	// Ordering: best PR first, so the table reads as a ranking.
	for i := 1; i < len(rows); i++ {
		prev, cur := rows[i-1], rows[i]
		if prev.Decisions == 0 && cur.Decisions > 0 {
			t.Errorf("row %d (%s) has no decisions but sits above %s, which has %d",
				i-1, prev.Name, cur.Name, cur.Decisions)
		}
		if prev.Decisions > 0 && cur.Decisions > 0 && prev.PR > cur.PR {
			t.Errorf("rows out of order: %s (PR %v) before %s (PR %v)",
				prev.Name, prev.PR, cur.Name, cur.PR)
		}
	}

	// A player with no analysed decision is still listed — the count beside the
	// figure is what tells the reader it means nothing, and silently dropping
	// people would make the table lie about who played.
	if _, err := s.Matches().Save(ctx, "",
		&domain.Match{Player1Name: "Dave", Player2Name: "Erin", MatchLength: 7}); err != nil {
		t.Fatalf("Save empty match: %v", err)
	}
	withEmpty, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerTable(after empty match): %v", err)
	}
	var dave *storage.PlayerRow
	for i := range withEmpty {
		if withEmpty[i].Name == "Dave" {
			dave = &withEmpty[i]
		}
	}
	if dave == nil {
		t.Fatal("a player whose match has no analysed decision must still be listed")
	}
	if dave.Decisions != 0 || dave.Matches != 1 {
		t.Errorf("Dave: got %d decisions over %d matches, want 0 over 1", dave.Decisions, dave.Matches)
	}
	if dave.LuckRolls != 0 {
		t.Errorf("Dave has no luck data, got %d measured rolls", dave.LuckRolls)
	}
	if _, ok := dave.LuckRateMP(); ok {
		t.Error("a player with no measured roll must report no luck rate, not an average of zero")
	}
	if last := withEmpty[len(withEmpty)-1]; last.Decisions != 0 {
		t.Errorf("players with nothing measured belong at the end, found %q (%d decisions) last",
			last.Name, last.Decisions)
	}
}

// testStatsPositionIDsByMatch exercises PositionIDsByMatch on both backends
// (previously PostgreSQL-only, via the parity test's count comparison).
func testStatsPositionIDsByMatch(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchA, posA := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	matchB, _ := statsFixtureMatch(t, s, 2, "Carol", "Dave")

	ids, err := s.Stats().PositionIDsByMatch(ctx, "", matchA)
	if err != nil {
		t.Fatalf("PositionIDsByMatch: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("PositionIDsByMatch(matchA): got %v, want 2 ids", ids)
	}
	got := map[int64]bool{ids[0]: true, ids[1]: true}
	if !got[posA[0]] || !got[posA[1]] {
		t.Errorf("PositionIDsByMatch(matchA): got %v, want %v", ids, posA)
	}

	// Match B's positions must not leak into match A's result.
	for _, id := range ids {
		if id != posA[0] && id != posA[1] {
			t.Errorf("PositionIDsByMatch(matchA) returned a foreign position %d", id)
		}
	}
	_ = matchB
}

// testStatsPositionIDsByTournament exercises PositionIDsByTournament, which
// had no test at all on either backend before this (not even in the
// PostgreSQL parity test): it aggregates by tournament, one level above
// PositionIDsByMatch.
func testStatsPositionIDsByTournament(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchA, posA := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	matchB, posB := statsFixtureMatch(t, s, 2, "Carol", "Dave")
	_, posOutside := statsFixtureMatch(t, s, 4, "Eve", "Frank") // never joins the tournament

	tID, err := s.Tournaments().Create(ctx, "", "Cup", "2025-06-01", "Paris")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tID, matchA); err != nil {
		t.Fatalf("AddMatch A: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tID, matchB); err != nil {
		t.Fatalf("AddMatch B: %v", err)
	}

	ids, err := s.Stats().PositionIDsByTournament(ctx, "", tID)
	if err != nil {
		t.Fatalf("PositionIDsByTournament: %v", err)
	}
	if len(ids) != 4 {
		t.Fatalf("PositionIDsByTournament: got %v, want 4 ids", ids)
	}
	got := map[int64]bool{}
	for _, id := range ids {
		got[id] = true
	}
	for _, want := range append(posA[:], posB[:]...) {
		if !got[want] {
			t.Errorf("PositionIDsByTournament: missing %d, got %v", want, ids)
		}
	}
	for _, outside := range posOutside {
		if got[outside] {
			t.Errorf("PositionIDsByTournament: leaked position %d from a match outside the tournament", outside)
		}
	}
}
