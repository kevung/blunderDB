package database

import (
	"strings"
	"testing"
)

// The directory (issue #391).
//
// It is a DERIVED VIEW and never a table: two spellings stay two lines, and deleting a
// Direction takes its Participants out of it, because there was never anywhere else they were
// written. That is the property these tests hold — not the CSV format, which is only how it
// travels.

func TestDirectory_IsDerivedFromTheDirections(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)
	if err := d.EnterParticipants(tID, `[
		{"id":"ha","name":"Hugo Andrieu","club":"Lyon","rating":5},
		{"id":"lb","name":"Léa Bonnet","club":"Lyon","rating":4.5}]`); err != nil {
		t.Fatal(err)
	}

	entries, err := d.Directory()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("%d entries, expected 2: %+v", len(entries), entries)
	}
	if entries[0].Name != "Hugo Andrieu" || entries[0].Club != "Lyon" || entries[0].Rating != 5 {
		t.Errorf("first entry reads %+v", entries[0])
	}
	if entries[0].LastTournamentID != tID || entries[0].LastTournament != "Open de Lyon" {
		t.Errorf("the entry does not say where it came from: %+v", entries[0])
	}

	// Nothing is stored: removing the Direction removes its Participants.
	if err := d.DeleteDirection(tID); err != nil {
		t.Fatal(err)
	}
	entries, err = d.Directory()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the directory survived the Direction it was derived from: %+v", entries)
	}
}

// The same person entered twice is one line, and the LAST entry says the club: someone who
// changed club is listed under the new one.
func TestDirectory_TheLastEntryWins(t *testing.T) {
	d := newTestDB(t)
	first := preparedDirection(t, d)
	if err := d.EnterParticipants(first, `[{"id":"ha","name":"Hugo Andrieu","club":"Lyon","rating":5}]`); err != nil {
		t.Fatal(err)
	}
	second, err := d.CreateTournament("Open de Paris", "2026-10-10", "Paris")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Paris","tables":{"count":8},"phases":[{"kind":"swiss_lives","length":7,"lives":2}]}`
	if err := d.CreateDirection(second, cfg, 7); err != nil {
		t.Fatal(err)
	}
	if err := d.EnterParticipants(second, `[{"id":"ha","name":"Hugo Andrieu","club":"Paris","rating":6}]`); err != nil {
		t.Fatal(err)
	}

	entries, err := d.Directory()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("%d entries for one person entered twice: %+v", len(entries), entries)
	}
	e := entries[0]
	if e.Entries != 2 {
		t.Errorf("the entry counts %d entries, expected 2", e.Entries)
	}
	if e.Club != "Paris" || e.Rating != 6 {
		t.Errorf("the last entry did not win: %+v", e)
	}
	if e.LastTournament != "Open de Paris" {
		t.Errorf("the entry names %q as its last tournament", e.LastTournament)
	}
}

// Two SPELLINGS stay two lines. Deciding that "J. Dupont" and "Jean Dupont" are one person is
// exactly the inference blunderDB does not make.
func TestDirectory_TwoSpellingsStayTwoLines(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)
	if err := d.EnterParticipants(tID, `[
		{"id":"a","name":"Jean Dupont"},
		{"id":"b","name":"J. Dupont"},
		{"id":"c","name":"  jean dupont  "}]`); err != nil {
		t.Fatal(err)
	}
	entries, err := d.Directory()
	if err != nil {
		t.Fatal(err)
	}
	// Three entries, two names: only case and surrounding space fold together.
	if len(entries) != 2 {
		t.Fatalf("%d lines, expected 2: %+v", len(entries), entries)
	}
}

// Taking last time's entrants is one call: the point of the whole feature.
func TestDirectory_TakeLastTimesEntrants(t *testing.T) {
	d := newTestDB(t)
	previous := preparedDirection(t, d)
	if err := d.EnterParticipants(previous, `[
		{"id":"ha","name":"Hugo Andrieu","club":"Lyon","rating":5},
		{"id":"lb","name":"Léa Bonnet","club":"Lyon","rating":4.5}]`); err != nil {
		t.Fatal(err)
	}
	sources, err := d.DirectorySources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Entrants != 2 || sources[0].Name != "Open de Lyon" {
		t.Fatalf("the sources read %+v", sources)
	}

	next, err := d.CreateTournament("Open de Lyon 2", "2026-10-10", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon 2","tables":{"count":8},"phases":[{"kind":"swiss_lives","length":7,"lives":2}]}`
	if err := d.CreateDirection(next, cfg, 7); err != nil {
		t.Fatal(err)
	}
	entrants, err := d.DirectoryEntrants(previous)
	if err != nil {
		t.Fatal(err)
	}
	var blob strings.Builder
	blob.WriteString("[")
	for i, e := range entrants {
		if i > 0 {
			blob.WriteString(",")
		}
		blob.WriteString(`{"name":"` + e.Name + `","club":"` + e.Club + `"}`)
	}
	blob.WriteString("]")
	if err := d.EnterParticipants(next, blob.String()); err != nil {
		t.Fatal(err)
	}
	rows, err := d.Participants(next)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Name != "Hugo Andrieu" {
		t.Fatalf("the entrants did not come across: %+v", rows)
	}
}

// Export then import gives the same list back, accents included.
func TestDirectory_CSVRoundTrip(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)
	if err := d.EnterParticipants(tID, `[
		{"id":"a","name":"Éloïse Ångström","club":"Lyon","rating":4.5},
		{"id":"b","name":"Jean-Baptiste de la Villemarqué","club":"","rating":0}]`); err != nil {
		t.Fatal(err)
	}
	body, err := d.DirectoryCSV()
	if err != nil {
		t.Fatal(err)
	}
	back, err := d.ParseDirectoryCSV(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Errors) != 0 {
		t.Fatalf("our own CSV did not read back: %+v", back.Errors)
	}
	before, err := d.Directory()
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Rows) != len(before) {
		t.Fatalf("%d lines out, %d back", len(before), len(back.Rows))
	}
	for i := range before {
		if back.Rows[i].Name != before[i].Name || back.Rows[i].Club != before[i].Club || back.Rows[i].Rating != before[i].Rating {
			t.Errorf("line %d came back as %+v, was %+v", i, back.Rows[i], before[i])
		}
	}
}

// A malformed CSV shows the offending lines and imports nothing: parsing writes nothing at all,
// which is what makes "nothing until confirmed" true rather than promised.
func TestDirectory_AMalformedCSVWritesNothing(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)

	body := "name,club,rating\nHugo Andrieu,Lyon,5\n,Lyon,4\nLéa Bonnet,Lyon,beaucoup\nMarc Colin,Paris,6\n"
	got, err := d.ParseDirectoryCSV(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != 2 {
		t.Errorf("%d good lines, expected 2: %+v", len(got.Rows), got.Rows)
	}
	if len(got.Skipped) != 1 || got.Skipped[0].Code != "header" {
		t.Errorf("the header was not reported as skipped: %+v", got.Skipped)
	}
	if len(got.Errors) != 2 {
		t.Fatalf("%d faulty lines, expected 2: %+v", len(got.Errors), got.Errors)
	}
	if got.Errors[0].Line != 3 || got.Errors[0].Code != "noName" {
		t.Errorf("the nameless line reads %+v", got.Errors[0])
	}
	if got.Errors[1].Line != 4 || got.Errors[1].Code != "badRating" || got.Errors[1].Text != "beaucoup" {
		t.Errorf("the bad rating reads %+v", got.Errors[1])
	}
	// And nothing was entered anywhere.
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("parsing entered %d participants", len(rows))
	}
}

// A French spreadsheet writes semicolons and commas for decimals. A file that will not open is
// a file the director stops using.
func TestDirectory_ASpreadsheetsCSVAlsoReads(t *testing.T) {
	d := newTestDB(t)
	got, err := d.ParseDirectoryCSV("nom;club;cote\nHugo Andrieu;Lyon;4,5\n")
	if err != nil {
		t.Fatal(err)
	}
	// L'en-tête est reconnu par une RÈGLE et non par une liste de mots : « cote » n'est pas un
	// nombre, donc cette ligne n'est pas une donnée — et blunderDB parle neuf langues.
	if len(got.Skipped) != 1 {
		t.Errorf("the French header was not skipped: %+v", got.Skipped)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("errors on a semicolon file: %+v", got.Errors)
	}
	if len(got.Rows) != 1 || got.Rows[0].Name != "Hugo Andrieu" || got.Rows[0].Rating != 4.5 {
		t.Fatalf("the line read as %+v", got.Rows)
	}
}
