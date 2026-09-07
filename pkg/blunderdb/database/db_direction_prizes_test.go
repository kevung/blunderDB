package database

import (
	"strings"
	"testing"
)

// The prize fund (issue #393, fonctionnel.md §3.6).
//
// The engine computes it; what is held here is the arithmetic a director will be asked about at
// the prize-giving, with the numbers of the acceptance criteria. A prize list that does not add
// up is the one thing everyone in the room checks.

// fundedDirection prepares a tournament with an entry fee, a retention and per-section scales.
func fundedDirection(t *testing.T, d *Database, n int, prizes string) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":16},"prizes":` + prizes + `,
		"phases":[{"kind":"bracket","length":5,"consolation":true}]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < n; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	return tID
}

// The acceptance case: 24 players at 20, 10 % retained, 50/30/20.
func TestPrizes_TheNumbersAddUp(t *testing.T) {
	d := newTestDB(t)
	tID := fundedDirection(t, d, 24, `{"entry_fee":20,"retention":{"percent":10},
		"sections":{"all":{"percents":[50,30,20]}}}`)

	v, err := d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Entrants != 24 {
		t.Fatalf("%d entrants", v.Entrants)
	}
	if v.Pool != 480 {
		t.Errorf("pool = %v, expected 24 × 20 = 480", v.Pool)
	}
	if v.Retained != 48 {
		t.Errorf("retained = %v, expected 10 %% of 480 = 48", v.Retained)
	}
	if v.Payable != 432 {
		t.Errorf("payable = %v, expected 480 − 48 = 432", v.Payable)
	}

	playToTheEnd(t, d, tID)
	if _, err := d.CloseDirection(tID); err != nil {
		t.Fatal(err)
	}
	v, err = d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	// The sum of what is handed out equals what there was to hand out. This is the check
	// everyone in the room makes.
	total := 0.0
	for _, sec := range v.Sections {
		for _, r := range sec.Rows {
			total += r.Prize
		}
	}
	if total != v.Payable {
		t.Errorf("%v handed out for %v payable", total, v.Payable)
	}
}

// The pool follows entries and withdrawals with no intervention: it is derived, like everything
// else. Everyone who entered counts, withdrawn included — they paid.
func TestPrizes_ThePoolFollowsTheEntries(t *testing.T) {
	d := newTestDB(t)
	tID := fundedDirection(t, d, 8, `{"entry_fee":20,"sections":{"all":{"percents":[100]}}}`)

	v, err := d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Pool != 160 {
		t.Fatalf("pool = %v, expected 8 × 20", v.Pool)
	}
	if _, err := d.AddParticipant(tID, "Yanis Ferrand", "Lyon", 4); err != nil {
		t.Fatal(err)
	}
	v, err = d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Pool != 180 {
		t.Errorf("pool = %v after one more entry, expected 180", v.Pool)
	}
	// A withdrawal does not give the money back: they paid, and the engine has no say in it.
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.WithdrawParticipant(tID, rows[0].ID, false); err != nil {
		t.Fatal(err)
	}
	v, err = d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Pool != 180 {
		t.Errorf("pool = %v after a withdrawal, expected it unchanged at 180", v.Pool)
	}
}

// The consolation hands out its OWN scale on its OWN standings.
func TestPrizes_TheConsolationHasItsOwn(t *testing.T) {
	d := newTestDB(t)
	tID := fundedDirection(t, d, 8, `{"entry_fee":20,"sections":{
		"main":{"percents":[60,20]},"conso":{"percents":[15,5]}}}`)
	playToTheEnd(t, d, tID)
	if _, err := d.CloseDirection(tID); err != nil {
		t.Fatal(err)
	}
	v, err := d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}

	paid := map[string]float64{}
	winners := map[string]int{}
	for _, sec := range v.Sections {
		for _, r := range sec.Rows {
			if r.Prize > 0 {
				paid[sec.Name] += r.Prize
				winners[sec.Name]++
			}
		}
	}
	if paid["main"] == 0 || paid["conso"] == 0 {
		t.Fatalf("each section should hand out its own: %v", paid)
	}
	if paid["main"] <= paid["conso"] {
		t.Errorf("the main draw hands out %v, the consolation %v", paid["main"], paid["conso"])
	}
	if winners["main"] != 2 || winners["conso"] != 2 {
		t.Errorf("two places paid in each section, got %v", winners)
	}
	total := paid["main"] + paid["conso"]
	if total != v.Payable {
		t.Errorf("%v handed out for %v payable", total, v.Payable)
	}
}

// The prizes travel in the CSV, which is what a director pastes into their accounts.
func TestPrizes_TheCSVCarriesThem(t *testing.T) {
	d := newTestDB(t)
	tID := fundedDirection(t, d, 8, `{"entry_fee":20,"sections":{"all":{"percents":[70,30]}}}`)
	playToTheEnd(t, d, tID)
	if _, err := d.CloseDirection(tID); err != nil {
		t.Fatal(err)
	}
	frenchStrings(t, d, "fr")
	body, err := d.StandingsCSV(tID)
	if err != nil {
		t.Fatal(err)
	}
	head := strings.SplitN(body, "\n", 2)[0]
	if !strings.Contains(head, "Prix") {
		t.Errorf("the CSV has no prize column: %q", head)
	}
	if !strings.Contains(body, "112.00") {
		t.Errorf("the first prize (70 %% of 160 = 112) is not in the CSV:\n%s", body)
	}
	// The CSV is pasted into a director's accounts: it speaks their language, and carries no
	// code of the engine's.
	frenchStrings(t, d, "de")
	body, err = d.StandingsCSV(tID)
	if err != nil {
		t.Fatal(err)
	}
	head = strings.SplitN(body, "\n", 2)[0]
	if !strings.Contains(head, "Preis") {
		t.Errorf("the CSV header did not follow the language: %q", head)
	}
}
