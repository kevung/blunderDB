package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The room's wall page (ADR-0056 §6): one line per table whichever event plays on it, a link to
// each attached event's own page, and the room's own output folder redirects each member's pages
// into a subfolder of its own — the acceptance criteria of D6.5 (#448).

// TestWallPage_OneLinePerTableWhicheverEvent is the acceptance criterion itself: as many <li>
// lines as the room has tables, regardless of how many events share it.
func TestWallPage_OneLinePerTableWhicheverEvent(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	a := startedDirection(t, d, 16)
	b := startedDirectionNamed(t, d, 16, "Joueuse ")

	r, err := d.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 14)
	if err != nil {
		t.Fatal(err)
	}
	for _, tID := range []int64{a, b} {
		if _, err := d.AttachToRencontre(tID, r.ID); err != nil {
			t.Fatalf("attach %d: %v", tID, err)
		}
	}
	ma := runningMatch(t, d, a)
	mb := runningMatch(t, d, b)

	page, err := d.RencontrePageHTML(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(page, `<li class="`); n != 14 {
		t.Errorf("wall page: %d table lines, want 14 (one per table, whichever event plays on it)", n)
	}
	for _, m := range []struct{ id, a, b string }{
		{string(ma.ID), "Joueur " + string(ma.A), "Joueur " + string(ma.B)},
		{string(mb.ID), "Joueuse " + string(mb.A), "Joueuse " + string(mb.B)},
	} {
		if !strings.Contains(page, m.a) || !strings.Contains(page, m.b) {
			t.Errorf("wall page is missing the players of running match %s: %+v", m.id, m)
		}
	}
	if !strings.Contains(page, "Open de Lyon") {
		t.Error("wall page does not name the events playing in the room")
	}
	if !strings.Contains(page, "Nicomaque") {
		t.Error("wall page carries no credit to the engine")
	}
	if strings.Contains(page, "<script") {
		t.Error("the wall page is a wall display: it must carry no script (it reloads by a meta tag)")
	}
}

// TestWallPage_LinksToEachAttachedEvent checks the second half of the acceptance criteria: a
// link per event, and its announced round count.
func TestWallPage_LinksToEachAttachedEvent(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	a := startedDirection(t, d, 16)
	b := startedDirection(t, d, 16)
	r, err := d.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 14)
	if err != nil {
		t.Fatal(err)
	}
	for _, tID := range []int64{a, b} {
		if _, err := d.AttachToRencontre(tID, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	page, err := d.RencontrePageHTML(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(page, `href="open-de-lyon`); n != 1 && n != 2 {
		// Two events named alike get a stable, distinct slug each (the second one's id
		// appended) rather than colliding on one folder.
		t.Errorf("wall page: %d link(s) to a slug starting with open-de-lyon, want at least one distinct per event", n)
	}
	if !strings.Contains(page, "rondes annoncées") && !strings.Contains(page, "ronde") {
		t.Error("wall page does not say how many rounds each event has announced")
	}
}

// TestRencontreOutputDir_RedirectsMemberPages: setting the room's folder is the whole gesture
// (D6.5's own criterion) — the wall page is written right away, and an attached Tournament's own
// page lands in a subfolder of its own, its OutputDir column left untouched.
func TestRencontreOutputDir_RedirectsMemberPages(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	a := startedDirection(t, d, 16)
	r, err := d.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.AttachToRencontre(a, r.ID); err != nil {
		t.Fatal(err)
	}
	runningMatch(t, d, a)

	root := t.TempDir()
	if _, err := d.SetRencontreOutputDir(r.ID, root); err != nil {
		t.Fatal(err)
	}
	if _, err := d.WriteDirectionPage(a); err != nil {
		t.Fatal(err)
	}

	wall := filepath.Join(root, "index.html")
	if _, err := os.ReadFile(wall); err != nil {
		t.Errorf("the setting itself should have written the wall page at %s: %v", wall, err)
	}
	sub := filepath.Join(root, "open-de-lyon", "tournoi.html")
	if _, err := os.ReadFile(sub); err != nil {
		t.Errorf("the attached event's own page should land under the room's folder at %s: %v", sub, err)
	}

	if v, err := d.GetDirection(a); err != nil || v.OutputDir != "" {
		t.Errorf("the tournament's own OutputDir must stay untouched by attaching, got %q", v.OutputDir)
	}

	if err := d.DetachFromRencontre(a); err != nil {
		t.Fatal(err)
	}
	if _, err := d.WriteDirectionPage(a); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(sub); err == nil {
		t.Log("stale page under the room's folder after detaching: harmless, nothing reads it any more")
	}
}
