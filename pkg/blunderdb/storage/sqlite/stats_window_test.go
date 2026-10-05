package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func gameIDs(db *sql.DB, matchID int64) ([]int64, error) {
	rows, err := db.Query(`SELECT id FROM game WHERE match_id = ? ORDER BY id`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// duplicateMatch copies a match, its games and its moves under a new date.
// The copies point at the same positions, hence the same analyses. Each copied
// move takes its row id as move number: the direct passes order the recent
// decisions by date then move number, and without a total order the 1000th
// decision would depend on how SQLite breaks ties, not on the code tested.
func duplicateMatch(t *testing.T, db *sql.DB, matchID int64, date string) {
	t.Helper()
	res, err := db.Exec(`INSERT INTO match (player1_name, player2_name, event, location, round,
		match_length, match_date, game_count, tournament_id)
		SELECT player1_name, player2_name, event, location, round, match_length, ?, game_count, tournament_id
		FROM match WHERE id = ?`, date, matchID)
	if err != nil {
		t.Fatal(err)
	}
	newMatch, _ := res.LastInsertId()
	games, err := gameIDs(db, matchID)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range games {
		res, err := db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2,
			winner, points_won, move_count)
			SELECT ?, game_number, initial_score_1, initial_score_2, winner, points_won, move_count
			FROM game WHERE id = ?`, newMatch, g)
		if err != nil {
			t.Fatal(err)
		}
		newGame, _ := res.LastInsertId()
		if _, err := db.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player,
			dice_1, dice_2, checker_move, cube_action, luck_mp, error_mp)
			SELECT ?, move_number, move_type, position_id, player, dice_1, dice_2, checker_move,
			cube_action, luck_mp, error_mp FROM move WHERE game_id = ? ORDER BY id`, newGame, g); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE move SET move_number = id WHERE game_id = ?`, newGame); err != nil {
			t.Fatal(err)
		}
	}
}

// The rolling figures cover the 1000 most recent decisions. Read from the
// per-match table they come from the matches holding those decisions, the
// last date reached taken whole; they must equal the figures read decision by
// decision, including when the 1000th decision falls inside a group of
// matches sharing one date.
func TestRollingWindowFromCellsMatchesDirectRead(t *testing.T) {
	ctx := context.Background()
	path := demoCopy(t)

	// Demo matches 1, 2 and 3 count 169, 284 and 61 decisions. Newest first:
	// match 1 and three copies of match 3, then all three on one date (866 so
	// far), then matches 1 and 2 on one date: whichever comes first closes the
	// window, and the other still holds some of the 1000 most recent decisions.
	raw, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []int64{1, 3, 3, 3} {
		duplicateMatch(t, raw, m, "2026-03-01 00:00:00 +0000 UTC")
	}
	for _, m := range []int64{1, 2, 3} {
		duplicateMatch(t, raw, m, "2026-02-01 00:00:00 +0000 UTC")
	}
	for _, m := range []int64{1, 2} {
		duplicateMatch(t, raw, m, "2026-01-01 00:00:00 +0000 UTC")
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	w, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	ro := openQueryOnly(t, path)

	filters := []storage.StatsFilter{
		{DecisionType: -1},
		{DecisionType: 0},
		{DecisionType: 1},
		{DecisionType: -1, PlayerName: "Iris Okonkwo"},
	}
	for _, f := range filters {
		got, err := w.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("writable Compute %+v: %v", f, err)
		}
		if f.DecisionType == -1 && f.PlayerName == "" {
			if _, ok := got.PRRolling[1000]; !ok {
				t.Fatalf("fixture does not fill the window: %d decisions, rolling %v",
					got.Totals.NumDecisions, got.PRRolling)
			}
		}
		if n := matchStatsRows(t, w); n == 0 {
			t.Fatal("writable Compute did not take the match_stats path")
		}
		want, err := ro.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("read-only Compute %+v: %v", f, err)
		}
		if g, d := asJSON(t, got), asJSON(t, want); !sameStatsJSON(t, g, d) {
			t.Errorf("filter %+v:\n cells  %s\n direct %s", f, g, d)
		}
	}
}

// Reclassifying the phase and game type rewrites two dimensions of the
// match_stats cells: the cells filled from the old values must go with them.
func TestReclassifyDerivedInvalidatesMatchStats(t *testing.T) {
	ctx := context.Background()
	path := demoCopy(t)
	w, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := w.sqlDB.Exec(`UPDATE position SET
		game_phase = CASE WHEN game_phase = 1 THEN 2 ELSE 1 END,
		game_type = CASE WHEN game_type = 1 THEN 2 ELSE 1 END`); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Stats().RebuildMatchStats(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	n, err := w.Positions().ReclassifyDerived(ctx, "")
	if err != nil || n == 0 {
		t.Fatalf("ReclassifyDerived = %d, %v; want a repair", n, err)
	}
	f := storage.StatsFilter{DecisionType: -1}
	got, err := w.Stats().Compute(ctx, "", f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Stats().RebuildMatchStats(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	want, err := w.Stats().Compute(ctx, "", f)
	if err != nil {
		t.Fatal(err)
	}
	if g, d := asJSON(t, got), asJSON(t, want); !sameStatsJSON(t, g, d) {
		t.Errorf("stale cells after the reclassification:\n kept    %s\n rebuilt %s", g, d)
	}
}
