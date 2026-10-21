package database

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// studyImpactFixture plays three plans on either side of the 30-day cutoff,
// both seats, with a blunder or two and one plan revised since the cutoff.
func studyImpactFixture(t *testing.T) *Database {
	t.Helper()
	db := newTestDB(t)
	day := func(ago int) string { return time.Now().AddDate(0, 0, -ago).Format("2006-01-02") }
	tid := createTournament(t, db, "T", day(90))
	type row struct {
		errMP, dt, player int
		gt                domain.GameType
	}
	var revised []int64
	play := func(date string, rows []row) {
		m := createMatch(t, db, "Alice", "Bob", date, 7, tid)
		g := createGame(t, db, m)
		for i, r := range rows {
			pid := insertStatsFixtureRow(t, db, m, g, r.errMP, r.dt, r.player, i+1)
			if _, err := db.db.Exec(`UPDATE position SET game_type = ? WHERE id = ?`, int(r.gt), pid); err != nil {
				t.Fatalf("game type: %v", err)
			}
			if r.gt == domain.TypeBackgame {
				revised = append(revised, pid)
			}
		}
	}
	play(day(60), []row{
		{10, 0, 0, domain.TypeRace}, {30, 0, 1, domain.TypeRace}, {150, 0, 0, domain.TypeRace},
		{5, 1, 0, domain.TypeBackgame}, {80, 0, 1, domain.TypeBackgame},
		{0, 0, 0, domain.TypeCrunch},
	})
	play(day(45), []row{
		{20, 0, 0, domain.TypeRace}, {12, 1, 1, domain.TypeCrunch}, {40, 0, 0, domain.TypeCrunch},
	})
	play(day(5), []row{
		{3, 0, 0, domain.TypeRace}, {7, 0, 1, domain.TypeRace},
		{200, 0, 0, domain.TypeBackgame}, {15, 1, 1, domain.TypeBackgame},
	})
	play(day(2), []row{
		{25, 0, 0, domain.TypeRace}, {9, 1, 0, domain.TypeAcePoint},
	})
	deck, err := db.db.Exec(`INSERT INTO anki_deck (name) VALUES ('deck')`)
	if err != nil {
		t.Fatalf("insert deck: %v", err)
	}
	deckID, _ := deck.LastInsertId()
	for _, pid := range revised[:2] {
		card, err := db.db.Exec(`INSERT INTO anki_card (deck_id, kind, key, position_id) VALUES (?, ?, CAST(? AS TEXT), ?)`,
			deckID, domain.AnkiKindPosition, pid, pid)
		if err != nil {
			t.Fatalf("insert card: %v", err)
		}
		cardID, _ := card.LastInsertId()
		if _, err := db.db.Exec(
			`INSERT INTO anki_review_log (card_id, deck_id, kind, key, position_id, rating)
			 VALUES (?, ?, ?, CAST(? AS TEXT), ?, 3)`, cardID, deckID, domain.AnkiKindPosition, pid, pid); err != nil {
			t.Fatalf("insert review: %v", err)
		}
	}
	return db
}

// TestStudyImpact_Frozen pins StudyImpact's figures on a fixture: the PR
// columns are the Stats tab's per-plan breakdown over each window, so any
// faster route to them must give these same numbers.
func TestStudyImpact_Frozen(t *testing.T) {
	t.Parallel()
	db := studyImpactFixture(t)
	rows, err := db.StudyImpact(30)
	if err != nil {
		t.Fatalf("StudyImpact: %v", err)
	}
	got, _ := json.Marshal(rows)
	const want = `[{"gameType":"backgame","reviewed":2,"prBefore":21.25,"decisionsBefore":2,"prAfter":53.75,"decisionsAfter":2},{"gameType":"acepoint","reviewed":0,"prBefore":0,"decisionsBefore":0,"prAfter":4.5,"decisionsAfter":1},{"gameType":"crunch","reviewed":0,"prBefore":8.666666666666666,"decisionsBefore":3,"prAfter":0,"decisionsAfter":0},{"gameType":"race","reviewed":0,"prBefore":26.25,"decisionsBefore":4,"prAfter":5.833333333333333,"decisionsAfter":3}]`
	if string(got) != want {
		t.Errorf("StudyImpact:\n got %s\nwant %s", got, want)
	}
}

// TestStudyImpact_IsTheStatsBreakdown checks each PR column against the
// per-plan row ComputeStats returns over the same window.
func TestStudyImpact_IsTheStatsBreakdown(t *testing.T) {
	t.Parallel()
	db := studyImpactFixture(t)
	rows, err := db.StudyImpact(30)
	if err != nil {
		t.Fatalf("StudyImpact: %v", err)
	}
	cutoff := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	for _, w := range []struct {
		name     string
		from, to string
		pick     func(StudyImpactRow) (float64, int)
	}{
		{"before", "", cutoff, func(r StudyImpactRow) (float64, int) { return r.PRBefore, r.DecisionsBefore }},
		{"after", cutoff, "", func(r StudyImpactRow) (float64, int) { return r.PRAfter, r.DecisionsAfter }},
	} {
		stats, err := db.ComputeStats(StatsFilter{DateFrom: w.from, DateTo: w.to, DecisionType: -1})
		if err != nil {
			t.Fatalf("ComputeStats: %v", err)
		}
		want := map[string]GameTypeStats{}
		for _, g := range stats.PerGameType {
			want[g.GameType] = g
		}
		for _, r := range rows {
			pr, n := w.pick(r)
			if pr != want[r.GameType].PR || n != want[r.GameType].NumDecisions {
				t.Errorf("%s %s: PR %v/%d, Stats tab %v/%d", w.name, r.GameType, pr, n, want[r.GameType].PR, want[r.GameType].NumDecisions)
			}
		}
	}
}
