package database

import (
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// playToTheEnd directs a whole tournament, entering a result for every match.
func playToTheEnd(t *testing.T, d *Database, tID int64) {
	t.Helper()
	for step := 0; step < 400; step++ {
		v, err := d.GetDirection(tID)
		if err != nil {
			t.Fatal(err)
		}
		if v.Finished {
			return
		}
		actionable := 0
		for _, a := range v.Proposals {
			// What "launch all" would actually confirm: not a wait, not a proposal with no
			// free table, and not the close — which is a click of its own.
			if a.Kind == tournoi.ActWait || a.Kind == tournoi.ActFinish {
				continue
			}
			if a.Reason == tournoi.ReasonWaitingTable {
				continue
			}
			actionable++
		}
		if actionable == 0 && len(v.Running) == 0 {
			return
		}
		if actionable > 0 {
			if _, err := d.ConfirmAllProposals(tID); err != nil {
				t.Fatal(err)
			}
		}
		v, err = d.GetDirection(tID)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range v.Running {
			// The lower identifier wins, so the run is deterministic.
			w := m.A
			if string(m.B) < string(m.A) {
				w = m.B
			}
			if _, err := d.EnterResult(tID, string(m.ID), string(w), m.Length, 2, ""); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestStandingsAndClose: a tournament played to its end ranks everyone, and closing freezes it.
func TestStandingsAndClose(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	playToTheEnd(t, d, tID)

	s, err := d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Sections) == 0 || len(s.Sections[0].Rows) != 24 {
		t.Fatalf("the overall standings rank all 24: %+v", s.Sections)
	}
	first := s.Sections[0].Rows[0]
	if first.Rank != 1 || first.Name == "" || first.Note.Kind == "" {
		t.Errorf("the winner is ranked first with a note: %+v", first)
	}
	if first.Name == first.ID {
		t.Error("the standings show names, not identifiers")
	}

	if _, err := d.CloseDirection(tID); err != nil {
		t.Fatalf("closing: %v", err)
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Finished || v.State != "finished" {
		t.Errorf("the tournament should be closed: finished=%v state=%q", v.Finished, v.State)
	}
	// A closed tournament refuses a decision.
	if _, err := d.AddParticipant(tID, "Tardif", "", 0); err == nil {
		t.Error("a closed tournament refuses an entry")
	}
	// But it can be reopened, because a result was wrong.
	if _, err := d.ReopenDirection(tID); err != nil {
		t.Fatalf("reopening: %v", err)
	}
	v, err = d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Finished {
		t.Error("a reopened tournament is no longer closed")
	}
}

// TestTiesShareTheirPlacesAndPrizes: there is no tie-break, by design. Ties stay ties and share
// the prizes of the places they occupy.
func TestTiesShareTheirPlacesAndPrizes(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("Dotation", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	// 24 entrants at 20 €, 10 % retained, 50/30/20 % to the first three of the main draw.
	cfg := `{"name":"Dotation","tables":{"count":8},
		"prizes":{"entry_fee":20,"retention":{"percent":10},
		          "sections":{"main":{"percents":[50,30,20]}}},
		"phases":[{"kind":"bracket","length":5}]}`
	if err := d.CreateDirection(tID, cfg, 5); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 24; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	playToTheEnd(t, d, tID)

	s, err := d.Standings(tID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Entrants != 24 {
		t.Errorf("%d entrants", s.Entrants)
	}
	if s.Pool != 480 {
		t.Errorf("pool %.2f, want 480 (24 × 20)", s.Pool)
	}
	if s.Payable != 432 {
		t.Errorf("payable %.2f, want 432 (480 less 10 %%)", s.Payable)
	}
	if s.Retained != 48 {
		t.Errorf("retained %.2f, want 48", s.Retained)
	}

	var main StandingsSection
	for _, sec := range s.Sections {
		if sec.Name == "main" {
			main = sec
		}
	}
	if main.Name == "" {
		t.Fatal("the main draw pays its own winners and must have its own standings")
	}
	// The distributed sum is exactly what is payable: rounding to the unit puts the remainder
	// on the first prize rather than losing it.
	total := 0.0
	for _, r := range main.Rows {
		total += r.Prize
	}
	if total != s.Payable {
		t.Errorf("distributed %.2f, payable %.2f — the remainder must land on the first prize", total, s.Payable)
	}
	// Ties are marked, so the view can say so rather than letting it look like a rounding
	// artefact.
	byRank := map[int]int{}
	for _, r := range main.Rows {
		byRank[r.Rank]++
	}
	for _, r := range main.Rows {
		if (byRank[r.Rank] > 1) != r.Shared {
			t.Errorf("%s at rank %d: shared=%v but %d people hold that rank", r.Name, r.Rank, r.Shared, byRank[r.Rank])
		}
	}
	// People sharing a rank share the same money.
	prizeAt := map[int]float64{}
	for _, r := range main.Rows {
		if p, seen := prizeAt[r.Rank]; seen && p != r.Prize {
			t.Errorf("rank %d pays %.2f to one and %.2f to another", r.Rank, p, r.Prize)
		}
		prizeAt[r.Rank] = r.Prize
	}
}

// TestStandingsCSV: the export is the engine's own, and it opens in a spreadsheet.
func TestStandingsCSV(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	playToTheEnd(t, d, tID)

	csv, err := d.StandingsCSV(tID)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) < 25 {
		t.Fatalf("%d lines, want a header plus 24 players", len(lines))
	}
	if !strings.Contains(lines[0], "rang") {
		t.Errorf("the first line is a header: %q", lines[0])
	}
}
