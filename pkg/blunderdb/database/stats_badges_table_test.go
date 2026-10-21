package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/statsequal"
)

// badgeReadings are the reads match_stats now serves: the match and
// tournament badges and the players table, under the filters it admits.
func badgeReadings(t *testing.T, st storage.Storage, tournamentID int64) map[string]string {
	t.Helper()
	ctx := context.Background()
	out := map[string]string{}
	put := func(name string, v any, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = string(b)
	}
	mb, err := st.Stats().MatchBadges(ctx, "", nil)
	put("MatchBadges(all)", mb, err)
	mb, err = st.Stats().MatchBadges(ctx, "", []int64{1, 2, 3})
	put("MatchBadges(1,2,3)", mb, err)
	tb, err := st.Stats().TournamentBadges(ctx, "")
	put("TournamentBadges", tb, err)
	for name, f := range map[string]storage.StatsFilter{
		"PlayerTable":             {DecisionType: -1},
		"PlayerTable(tournament)": {DecisionType: -1, TournamentIDs: []int64{tournamentID}},
		"PlayerTable(7 points)":   {DecisionType: -1, MatchLength: []int{7}},
		"PlayerTable(XG engine)":  {DecisionType: -1, AnalysisEngine: "XG"},
	} {
		rows, err := st.Stats().PlayerTable(ctx, "", f)
		put(name, rows, err)
	}
	return out
}

// directReadings are badgeReadings over the decisions: a copy of the
// library with match_stats emptied, read through a connection that cannot
// fill it.
func directReadings(t *testing.T, d *Database, tournamentID int64) map[string]string {
	t.Helper()
	copyPath := filepath.Join(tempDir(t), "direct.db")
	if _, err := d.db.Exec(`VACUUM INTO ?`, copyPath); err != nil {
		t.Fatalf("VACUUM INTO: %v", err)
	}
	w, err := sql.Open("sqlite", sqlite.DSN(copyPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Exec(`DELETE FROM match_stats`); err != nil {
		t.Fatal(err)
	}
	w.Close()
	ro, err := sql.Open("sqlite", sqlite.DSN(copyPath))
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	ro.SetMaxOpenConns(1)
	if _, err := ro.Exec(`PRAGMA query_only = ON`); err != nil {
		t.Fatal(err)
	}
	return badgeReadings(t, sqlite.New(ro), tournamentID)
}

// The badges and the players table read from match_stats equal the direct
// calculation over the decisions, on a library of every import format, two
// tournaments, an alias and a Main player, and still do after each write
// that changes what they summarise: an import, a deletion, a seat swap, a
// re-analysis, a tournament assignment.
func TestBadgesFromMatchStatsEqualDirect(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(tempDir(t), "badges.db")); err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, d)

	aachen := "testdata/2024-08-10-Aachen-1x11pt-1x7pt-2x7ptDoubleConsultation/"
	var double, main []int64
	for _, f := range []struct {
		path  string
		load  func(string) (int64, error)
		group *[]int64
	}{
		{aachen + "double/ronde1-UngerHarmand-FriebeJacobi-7pt.xg", d.ImportXGMatch, &double},
		{aachen + "double/ronde3-UngerHarmand-HuyckLarsen.xg", d.ImportXGMatch, &double},
		{aachen + "double/Lux Heuler-Franzosen 7 point match 12.08.2024.xg", d.ImportXGMatch, &double},
		{aachen + "main-intermediate/1-Intermediate-Round1-Unger-Kozoman-11pt.xg", d.ImportXGMatch, &main},
		{aachen + "main-intermediate/2-Intermediate-Round2-Unger-Lahme-7pt.xg", d.ImportXGMatch, &main},
		{"testdata/HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg", d.ImportXGMatch, &main},
		{"testdata/charlot1-charlot2_7p_2025-11-08-2305.xg", d.ImportXGMatch, nil},
		{"testdata/Kev_GammonNetNormal_2026-10-06_7p.xg", d.ImportXGMatch, nil},
		{"testdata/test.xg", d.ImportXGMatch, nil},
		{"testdata/gnubg_selfplay_drops_7p.mat", d.ImportGnuBGMatch, nil},
		{"testdata/gnubg_roll_then_resign_1p.mat", d.ImportGnuBGMatch, nil},
		{"testdata/TachiAI_V_player_Nov_2__2025__16_55.bgf", d.ImportBGFMatch, nil},
		{"testdata/hedgehog-3pt-analysed.ogxm", d.ImportOGXMMatch, nil},
	} {
		id, err := f.load(f.path)
		if err != nil {
			t.Fatalf("import %s: %v", f.path, err)
		}
		if f.group != nil {
			*f.group = append(*f.group, id)
		}
	}
	tDouble, err := d.CreateTournament("Aachen double", "2024-08-10", "Aachen")
	if err != nil {
		t.Fatal(err)
	}
	tMain, err := d.CreateTournament("Main", "2024-08-10", "Aachen")
	if err != nil {
		t.Fatal(err)
	}
	for tid, ids := range map[int64][]int64{tDouble: double, tMain: main} {
		for _, id := range ids {
			if err := d.AddMatchToTournament(tid, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	var p1, p2 string
	if err := d.db.QueryRow(`SELECT player1_name, player2_name FROM match WHERE id = ?`, main[0]).Scan(&p1, &p2); err != nil {
		t.Fatal(err)
	}
	if err := d.SetAlias("player", p2+" (alias)", p2); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SetMainPlayer(p1); err != nil {
		t.Fatal(err)
	}

	check := func(step string) map[string]string {
		t.Helper()
		got := badgeReadings(t, d.store, tMain)
		var rows int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM match_stats`).Scan(&rows); err != nil || rows == 0 {
			t.Fatalf("%s: match_stats holds %d rows (%v): the readings did not come from it", step, rows, err)
		}
		want := directReadings(t, d, tMain)
		for name, w := range want {
			if ok, err := statsequal.JSON(w, got[name]); err != nil || !ok {
				t.Errorf("%s, %s (%v):\n table  %s\n direct %s", step, name, err, got[name], w)
			}
		}
		return got
	}
	before := check("imported")

	// A re-analysis: every move but the best one loses more.
	var reanalysed int
	positions := func() []int64 {
		rows, err := d.db.Query(`SELECT DISTINCT mv.position_id FROM move mv JOIN position p ON p.id = mv.position_id
			JOIN game g ON g.id = mv.game_id WHERE g.match_id = ? AND p.decision_type = 0 ORDER BY mv.position_id LIMIT 20`, main[0])
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return ids
	}()
	for _, id := range positions {
		a, err := d.LoadAnalysis(id)
		if err != nil || a == nil || a.CheckerAnalysis == nil || len(a.CheckerAnalysis.Moves) < 2 {
			continue
		}
		a.CheckerAnalysis.Moves[0].Equity += 0.2
		if err := d.SaveAnalysis(id, *a); err != nil {
			t.Fatal(err)
		}
		reanalysed++
	}
	if reanalysed == 0 {
		t.Fatal("no checker analysis to re-analyse")
	}
	after := check("re-analysed")
	if after["TournamentBadges"] == before["TournamentBadges"] {
		t.Error("the re-analysis left the tournament badges as they were")
	}

	if err := d.SwapMatchPlayers(main[1]); err != nil {
		t.Fatal(err)
	}
	check("seats swapped")
	if err := d.AddMatchToTournament(tMain, double[2]); err != nil {
		t.Fatal(err)
	}
	check("moved to another tournament")
	if err := d.DeleteMatch(double[0]); err != nil {
		t.Fatal(err)
	}
	check("deleted")
	if _, err := d.ImportXGMatch(aachen + "double/Squire-Jørgensen-Harmand-Unger 7 point match 18-08-2024.xg"); err != nil {
		t.Fatal(err)
	}
	check("imported again")
}
