package database

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// TestSeasonRanking: two finished tournaments of a period are summed per person by the scale;
// a tournament outside the period does not count; the club Elo is zero-sum.
func TestSeasonRanking(t *testing.T) {
	d := newTestDB(t)
	dates := []string{"2026-01-10", "2026-02-14", "2025-12-01"}
	for _, date := range dates {
		id := startedDirection(t, d, 8)
		if err := d.UpdateTournament(id, "Club "+date, date, "Lyon"); err != nil {
			t.Fatal(err)
		}
		playToTheEnd(t, d, id)
		if _, err := d.CloseDirection(id); err != nil {
			t.Fatal(err)
		}
	}

	q := SeasonQuery{From: "2026-01-01", To: "2026-12-31", Points: []float64{10, 6, 4}, Participation: 1, Elo: true}
	v, err := d.SeasonRanking(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Events) != 2 || v.Events[0].Date != "2026-01-10" || v.Events[1].Date != "2026-02-14" {
		t.Fatalf("the period selects its two events, in date order: %+v", v.Events)
	}
	if len(v.Rows) != 8 {
		t.Fatalf("the same eight people are one row each: %d rows", len(v.Rows))
	}

	// The total of the season is what the scale distributes, twice, plus participation.
	want := 2*(10+6+4) + 2*8*1.0
	got, elo := 0.0, 0.0
	for _, r := range v.Rows {
		got += r.Total
		elo += r.Elo
		if r.Played != 2 || len(r.Places) != 2 || r.Places[0] == 0 {
			t.Errorf("%s played both events: %+v", r.Name, r)
		}
	}
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("season points = %v; want %v", got, want)
	}
	if math.Abs(elo-8*1500) > 0.5 {
		t.Errorf("the club Elo is zero-sum: total %v", elo)
	}
	if v.Rows[0].Rank != 1 || v.Rows[0].Total < v.Rows[len(v.Rows)-1].Total {
		t.Errorf("rows are by points: %+v", v.Rows)
	}

	csvText, err := d.SeasonCSV(q)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(csvText), "\n")
	if len(lines) != 9 || !strings.Contains(lines[0], "elo") || !strings.Contains(lines[0], "2026-02-14") {
		t.Errorf("the CSV has a header and a line per person:\n%s", csvText)
	}

	if _, err := d.SeasonRanking(SeasonQuery{From: "2026-12-31", To: "2026-01-01"}); err == nil {
		t.Error("an inverted period is refused")
	}
	all, err := d.SeasonRanking(SeasonQuery{})
	if err != nil || len(all.Events) != 3 || all.Elo || all.Rows[0].Elo != 0 {
		t.Errorf("no bound is every directed tournament, without Elo: %+v, %v", all, err)
	}
}

// TestSeasonRankingRefusesHomonyms: two entrants of one finished event under one name would be
// one row, played twice and rated against themselves; the season refuses and says where.
func TestSeasonRankingRefusesHomonyms(t *testing.T) {
	d := newTestDB(t)
	id := startedDirection(t, d, 8)
	if err := d.UpdateTournament(id, "Open double", "2026-03-01", "Lyon"); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 8; i++ {
		name := fmt.Sprintf("Joueur %d", i)
		if i < 2 {
			name = "Dupont"
		}
		players = append(players, fmt.Sprintf(`{"id":"p%d","name":%q}`, i, name))
	}
	if err := d.EnterParticipants(id, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Skipf("the direction itself refuses homonyms: %v", err)
	}
	playToTheEnd(t, d, id)
	if _, err := d.CloseDirection(id); err != nil {
		t.Fatal(err)
	}
	_, err := d.SeasonRanking(SeasonQuery{})
	if err == nil || !strings.Contains(err.Error(), "Dupont") || !strings.Contains(err.Error(), "Open double") {
		t.Errorf("homonyms in one event are refused by name and event: %v", err)
	}
}
