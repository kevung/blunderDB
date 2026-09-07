package database

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The standalone display page (issue #386).
//
// What these tests hold is the page's promise: it opens OFFLINE — no network resource of any
// kind — it shows names and words rather than identifiers and codes, it credits the engine's
// author, and a folder that has gone away is reported without interrupting the tournament.

// frenchStrings hands the backend the frontend's own `direction` catalogue, which is what the
// panel does when a Direction is opened.
func frenchStrings(t *testing.T, d *Database, lang string) {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("frontend", "src", "i18n", "locales", lang+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(blob, &whole); err != nil {
		t.Fatal(err)
	}
	if err := d.SetDirectionStrings(lang, string(whole["direction"])); err != nil {
		t.Fatal(err)
	}
}

func TestDirectionPage_OpensOffline(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	tID := startedDirection(t, d, 16)
	runningMatch(t, d, tID)

	page, err := d.DirectionPageHTML(tID)
	if err != nil {
		t.Fatal(err)
	}
	// One file, no network. A page that fetched anything would be blank in a hall with no
	// wifi, which is every hall on a Sunday morning.
	for _, forbidden := range []string{"http://", "https://", "<script", "srcset"} {
		if strings.Contains(page, forbidden) {
			t.Errorf("the page reaches outside itself: %q found", forbidden)
		}
	}
	if !strings.Contains(page, "Open de Lyon") {
		t.Error("the page does not name the tournament")
	}
	// The other half of "updates without a gesture": blunderDB rewrites the file at every
	// event, and the page reloads itself. Without this meta the director would have to press
	// F5 on the hall's computer all day.
	if !strings.Contains(page, `http-equiv="refresh"`) {
		t.Error("the page does not reload itself")
	}
	// The engine's author is credited on the page it produced.
	if !strings.Contains(page, "Nicolas Harmand") {
		t.Error("the page does not credit the engine's author")
	}
}

// The page shows NAMES and WORDS. An identifier or a raw code on a wall display is a bug the
// players see before the director does.
func TestDirectionPage_ShowsNamesNotIdentifiers(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	tID := startedDirection(t, d, 16)
	runningMatch(t, d, tID)

	page, err := d.DirectionPageHTML(tID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page, "Joueur a") {
		t.Error("the page shows no player name")
	}
	for _, code := range []string{"swiss_lives", "lives_bracket", "waiting_table", "start_match"} {
		if strings.Contains(page, code) {
			t.Errorf("the engine's code %q reached the page", code)
		}
	}
	if !strings.Contains(page, `lang="fr"`) {
		t.Error("the page does not declare its language")
	}
	// A proposal still waiting for a free table carries no table number, and "Table 0" on a
	// wall is a number the players go looking for between table 1 and table 2 (fixed upstream
	// in Nicomaque v0.2.1, found here).
	if strings.Contains(page, "Table 0") {
		t.Error("the page names table 0")
	}
}

func TestDirectionPage_SpeaksTheInterfacesLanguage(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 16)
	runningMatch(t, d, tID)

	frenchStrings(t, d, "de")
	page, err := d.DirectionPageHTML(tID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page, `lang="de"`) {
		t.Error("the page does not declare German")
	}
	if !strings.Contains(page, "Tische") {
		t.Error("the page's headings are not in German")
	}
	// The credit is owed in every language, not only in the engine's own.
	if !strings.Contains(page, "Nicolas Harmand") || strings.Contains(page, "moteur de tournoi") {
		t.Error("the credit did not follow the language of the page")
	}
}

// After the folder is chosen once, the display updates with no gesture: writing is a
// consequence, so it must be free to call at every event and silent when there is nothing to do.
func TestDirectionPage_WrittenAtomicallyOrNotAtAll(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	tID := startedDirection(t, d, 16)

	// No folder: nothing written, nothing said.
	path, err := d.WriteDirectionPage(tID)
	if err != nil || path != "" {
		t.Fatalf("with no folder chosen: path=%q err=%v", path, err)
	}

	out := t.TempDir()
	if err := d.SetDirectionOutputDir(tID, out); err != nil {
		t.Fatal(err)
	}
	path, err = d.WriteDirectionPage(tID)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != direction.PageName {
		t.Errorf("the page was written to %q", path)
	}
	// Nothing but the page is left behind: an atomic write that forgot its temporary file
	// would litter the director's folder a little more at every result.
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != direction.PageName {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("the folder holds %v", names)
	}

	// And it is rewritten in place, not appended to: same path after a second event.
	runningMatch(t, d, tID)
	again, err := d.WriteDirectionPage(tID)
	if err != nil || again != path {
		t.Errorf("second write: path=%q err=%v", again, err)
	}
}

// A folder that has gone away — an unplugged USB key — is reported and stops nothing. The
// director keeps directing.
func TestDirectionPage_MissingFolderDoesNotStopTheTournament(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	tID := startedDirection(t, d, 16)
	if err := d.SetDirectionOutputDir(tID, filepath.Join(t.TempDir(), "disparu")); err != nil {
		t.Fatal(err)
	}
	if _, err := d.WriteDirectionPage(tID); err == nil {
		t.Error("a folder that does not exist should be reported")
	}
	// The tournament runs on regardless.
	m := runningMatch(t, d, tID)
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 0, 0, ""); err != nil {
		t.Fatalf("the tournament stopped because a folder was missing: %v", err)
	}
}

// The page is rewritten at EVERY event, so its cost is paid at every result a director types.
// A 64-player tournament is the largest this is meant for; the bound is generous on purpose —
// what it catches is an accidental blow-up, not a few milliseconds.
func TestDirectionPage_CostOfARewrite(t *testing.T) {
	d := newTestDB(t)
	frenchStrings(t, d, "fr")
	tID := startedDirection(t, d, 64)
	// The whole tournament played out: the longest journal this page ever replays.
	playToTheEnd(t, d, tID)

	start := time.Now()
	const rewrites = 20
	for i := 0; i < rewrites; i++ {
		if _, err := d.DirectionPageHTML(tID); err != nil {
			t.Fatal(err)
		}
	}
	each := time.Since(start) / rewrites
	t.Logf("one rewrite of a 64-player tournament: %v (replay included)", each)
	if each > 2*time.Second {
		t.Errorf("a rewrite costs %v, which a director would feel at every result", each)
	}
}
