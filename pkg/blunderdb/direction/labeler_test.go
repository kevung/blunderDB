package direction_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/render"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The Go side of the pivot (issue #386).
//
// The engine emits codes; the panel renders them in JavaScript, the display page in Go. The two
// read the SAME catalogue, so what is tested here is not the strings but the RULE: the same
// interpolation, the same fallbacks, and above all — no code, ever, reaches a reader raw when a
// translation exists for it.

// catalog loads one language's `direction` block, the way the frontend hands it over.
func catalog(t *testing.T, lang string) *direction.Catalog {
	t.Helper()
	root := repoRoot(t)
	blob, err := os.ReadFile(filepath.Join(root, "frontend", "src", "i18n", "locales", lang+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(blob, &whole); err != nil {
		t.Fatal(err)
	}
	cat, err := direction.NewCatalog(whole["direction"])
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// repoRoot walks up to the directory holding go.mod: this package's tests do not chdir.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the working directory")
		}
		dir = parent
	}
}

func TestLabeler_RendersEngineCodes(t *testing.T) {
	l := direction.NewLabeler(catalog(t, "fr"), nil)

	if got := l.Label(tournoi.Label{Kind: "round", N: 3}); got != "Ronde 3" {
		t.Errorf("round label = %q", got)
	}
	if got := l.Label(tournoi.Label{Kind: "swiss_group", Losses: 1, Match: 4}); got != "1 défaite(s), match 4" {
		t.Errorf("swiss group label = %q", got)
	}
	if got := l.Label(tournoi.Label{Kind: "main_draw", Sub: &tournoi.Label{Kind: "final"}}); got != "Principal, Finale" {
		t.Errorf("composed label = %q", got)
	}
	if got := l.Note(tournoi.Note{Kind: "record", Wins: 3, Losses: 1}); got != "3 victoires, 1 défaites" {
		t.Errorf("note = %q", got)
	}
	if got := l.Reason("waiting_table"); got != "aucune table libre" {
		t.Errorf("reason = %q", got)
	}
}

func TestLabeler_SectionNamesAreIdentifiers(t *testing.T) {
	l := direction.NewLabeler(catalog(t, "fr"), nil)
	cases := map[string]string{
		"main":       "principal",
		"conso":      "consolante",
		"poule:A":    "poule A",
		"barrage:B":  "barrage poule B",
		"":           "",
		"inconnue:X": "inconnue:X",
	}
	for in, want := range cases {
		if got := l.SectionName(in); got != want {
			t.Errorf("SectionName(%q) = %q, want %q", in, got, want)
		}
	}
}

// A warning is the one place a reader needs the FACTS as well as the sentence: who played whom,
// on which match. Rendering the code alone would say "the bracket expects other players" and
// leave the director hunting.
func TestLabeler_WarningCarriesItsFacts(t *testing.T) {
	l := direction.NewLabeler(catalog(t, "fr"), func(id tournoi.PlayerID) string {
		return "Joueur " + string(id)
	})
	got := l.Warning(tournoi.Warning{
		Code: "bracket_wrong_players", Match: "m12", Section: "main",
		Label: tournoi.Label{Kind: "final"},
		A:     "a", B: "b", ExpectedA: "c", ExpectedB: "d",
	})
	for _, want := range []string{"m12", "principal", "Finale", "Joueur a", "Joueur b", "Joueur c", "Joueur d"} {
		if !strings.Contains(got, want) {
			t.Errorf("warning %q lacks %q", got, want)
		}
	}
}

// An unknown code shows AS ITSELF rather than as a blank: a future version of the engine is
// then visible on screen, and the missing key names what to translate.
func TestLabeler_UnknownCodeShowsItself(t *testing.T) {
	l := direction.NewLabeler(catalog(t, "fr"), nil)
	if got := l.Label(tournoi.Label{Kind: "something_new"}); got != "something_new" {
		t.Errorf("unknown label = %q", got)
	}
	if got := l.Note(tournoi.Note{Kind: "something_new"}); got != "something_new" {
		t.Errorf("unknown note = %q", got)
	}
	if got := l.Term(render.Term("something_new"), 0); got != "something_new" {
		t.Errorf("unknown term = %q", got)
	}
	// And an empty code is empty, not the word "empty".
	if got := l.Label(tournoi.Label{}); got != "" {
		t.Errorf("empty label = %q", got)
	}
}

// The nine languages all answer, and none of them answers with the code.
func TestLabeler_EveryLanguageAnswers(t *testing.T) {
	for _, lang := range []string{"fr", "en", "de", "el", "es", "fi", "it", "ja", "ru"} {
		l := direction.NewLabeler(catalog(t, lang), nil)
		if got := l.Label(tournoi.Label{Kind: "final"}); got == "final" || got == "" {
			t.Errorf("%s: the final is rendered %q", lang, got)
		}
		if got := l.Term(render.TermTables, 0); got == "tables" || got == "" {
			t.Errorf("%s: the tables heading is rendered %q", lang, got)
		}
		if got := l.PhaseName(tournoi.PhaseConfig{Kind: tournoi.KindBracket}); got == "bracket" || got == "" {
			t.Errorf("%s: the bracket phase is named %q", lang, got)
		}
	}
}

// A phase the director named themselves keeps that name, in every language: it is their word,
// not a code.
func TestLabeler_DirectorsPhaseNameWins(t *testing.T) {
	l := direction.NewLabeler(catalog(t, "en"), nil)
	got := l.PhaseName(tournoi.PhaseConfig{Kind: tournoi.KindBracket, Name: "Consolante du dimanche"})
	if got != "Consolante du dimanche" {
		t.Errorf("phase name = %q", got)
	}
}

// The count reaches a term that wants it, and is ignored by one that does not.
func TestLabeler_TermTakesItsCount(t *testing.T) {
	l := direction.NewLabeler(catalog(t, "fr"), nil)
	if got := l.Term(render.TermPlayers, 24); !strings.Contains(got, "24") {
		t.Errorf("players term = %q", got)
	}
	if got := l.Term(render.TermTable, 24); strings.Contains(got, "24") {
		t.Errorf("the table heading should not carry a count: %q", got)
	}
}
