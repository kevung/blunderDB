package database

import (
	"strings"
	"testing"
)

// wallBracketDirection starts an event whose only phase is a drawn single-elimination bracket
// (with a consolation), 8 players named "<prefix>a".."<prefix>h".
func wallBracketDirection(t *testing.T, d *Database, name, prefix string) int64 {
	t.Helper()
	tID, err := d.CreateTournament(name, "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"` + name + `","tables":{"count":8},"phases":[{"kind":"bracket","length":5,"consolation":true}]}`
	if err := d.CreateDirection(tID, cfg, 5); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 8; i++ {
		id := string(rune('a' + i))
		players = append(players, `{"id":"`+id+`","name":"`+prefix+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatal(err)
	}
	return tID
}

func TestWallPage_RotatesTheBracketsOfEventsInBracketPhase(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	swiss := startedDirection(t, d, 16) // a Swiss phase: no tree
	br := wallBracketDirection(t, d, "Coupe du dimanche", "Tab ")

	r, err := d.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 6)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{swiss, br} {
		if _, err := d.AttachToRencontre(id, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	page, err := d.RencontrePageHTML(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(page, `<svg class="arbre-mur"`); n != 1 {
		t.Fatalf("%d trees on the wall, want 1 (the Swiss event has none)", n)
	}
	if !strings.Contains(page, "Tab a") {
		t.Error("the tree does not name its players")
	}
	if n := strings.Count(page, `<section class="vue"`); n != 2 {
		t.Errorf("%d views in rotation, want 2 (tables, then the tree)", n)
	}
	if !strings.Contains(page, "animation-delay:12s") || !strings.Contains(page, `content="30"`) && !strings.Contains(page, `content="24"`) {
		t.Error("the views are not scheduled one after the other")
	}
	if strings.Contains(page, "<script") {
		t.Error("the wall page must carry no script")
	}
	if n := strings.Count(page, `<li class="`); n != 6 {
		t.Errorf("%d table lines, want 6: the tables stay in the first view", n)
	}
}

func TestWallPage_NoTreeWithoutBracket(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	swiss := startedDirection(t, d, 16)
	r, err := d.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 6)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.AttachToRencontre(swiss, r.ID); err != nil {
		t.Fatal(err)
	}
	page, err := d.RencontrePageHTML(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"<svg", `class="rot"`, "animation"} {
		if strings.Contains(page, unwanted) {
			t.Errorf("an event with no bracket must leave the wall page plain, found %q", unwanted)
		}
	}
	if !strings.Contains(page, `content="30"`) {
		t.Error("the plain page reloads every 30 seconds")
	}
}

func TestDirectionPage_ShowsItsOwnBracketInRotation(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	br := wallBracketDirection(t, d, "Coupe du dimanche", "Tab ")
	page, err := d.DirectionPageHTML(br)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(page, `<svg`) != 1 || !strings.Contains(page, `<svg class="arbre-mur"`) {
		t.Errorf("want exactly the wall's own tree (the engine's small one is replaced), got %d svg", strings.Count(page, "<svg"))
	}
	if strings.Count(page, `<section class="vue"`) != 2 {
		t.Error("the event's page rotates between its usual content and the tree")
	}
	if !strings.Contains(page, "<h2>") || !strings.Contains(page, `class="credit"`) {
		t.Error("the usual page content or the credit went missing")
	}

	swiss := startedDirection(t, d, 16)
	plain, err := d.DirectionPageHTML(swiss)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "<svg") || strings.Contains(plain, "animation") {
		t.Error("an event with no bracket keeps its plain page")
	}
}
