package searchquery

import (
	"reflect"
	"testing"
)

// TestTranslateIntent_TheFichesOwnExample is the sentence the fiche uses to
// state the problem: « s mes blunders de videau au score ». It must produce a
// query, and the two intentions that are NOT tokens must come back as board
// hints rather than as tokens that would mean something else.
func TestTranslateIntent_TheFichesOwnExample(t *testing.T) {
	got := TranslateIntent("mes blunders de videau au score")
	if !reflect.DeepEqual(got.Tokens, []string{"E>80"}) {
		t.Errorf("tokens: got %v, want [E>80]", got.Tokens)
	}
	if got.Board.Decision != "cube" {
		t.Errorf("decision: got %q, want cube", got.Board.Decision)
	}
	if got.Board.Score != "match" {
		t.Errorf("score: got %q, want match", got.Board.Score)
	}
	if len(got.Ignored) != 0 {
		t.Errorf("nothing should be left over: got %v", got.Ignored)
	}
}

// TestTranslateIntent_LongestPhraseWins: "grosse erreur" is a blunder, not an
// error qualified by an adjective nobody read. A vocabulary that matched the
// shortest phrase first would translate it as the weaker threshold and nobody
// would see it.
func TestTranslateIntent_LongestPhraseWins(t *testing.T) {
	if got := TranslateIntent("mes grosses erreurs"); !reflect.DeepEqual(got.Tokens, []string{"E>80"}) {
		t.Errorf("got %v, want [E>80]", got.Tokens)
	}
	if got := TranslateIntent("mes erreurs"); !reflect.DeepEqual(got.Tokens, []string{"E>20"}) {
		t.Errorf("got %v, want [E>20]", got.Tokens)
	}
	if got := TranslateIntent("en milieu de partie"); !reflect.DeepEqual(got.Tokens, []string{"ph:middlegame"}) {
		t.Errorf("got %v, want [ph:middlegame]", got.Tokens)
	}
}

// TestTranslateIntent_ReportsWhatItDidNotUnderstand is the property that keeps
// the layer honest. Translating half a sentence in silence is the only real
// danger here: the user believes they asked for something the query does not
// ask for.
func TestTranslateIntent_ReportsWhatItDidNotUnderstand(t *testing.T) {
	got := TranslateIntent("mes blunders contre Untel en holding")
	if !containsString(got.Tokens, "E>80") || !containsString(got.Tokens, "gt:holding") {
		t.Errorf("tokens: got %v", got.Tokens)
	}
	if !containsString(got.Ignored, "contre") || !containsString(got.Ignored, "untel") {
		t.Errorf("the unread words must be reported: got %v", got.Ignored)
	}
	// The words a sentence needs and a query does not are NOT reported:
	// listing "mes" and "en" as unrecognised would bury the word that was.
	if containsString(got.Ignored, "mes") || containsString(got.Ignored, "en") {
		t.Errorf("filler words must not be reported: got %v", got.Ignored)
	}
}

// TestTranslateIntent_IsDeterministicAndAccentBlind: the same phrase always
// gives the same line — that is what "déterministe, hors ligne" means — and a
// user typing without accents gets the same answer as one who types them.
func TestTranslateIntent_IsDeterministicAndAccentBlind(t *testing.T) {
	a := TranslateIntent("positions marquées en course")
	b := TranslateIntent("POSITIONS MARQUEES EN COURSE")
	if !reflect.DeepEqual(a.Tokens, b.Tokens) {
		t.Errorf("accents and case must not change the query: %v vs %v", a.Tokens, b.Tokens)
	}
	if !reflect.DeepEqual(a.Tokens, []string{"fl", "ph:race"}) {
		t.Errorf("tokens: got %v, want [fl ph:race]", a.Tokens)
	}
	for i := 0; i < 5; i++ {
		if again := TranslateIntent("positions marquées en course"); !reflect.DeepEqual(a.Tokens, again.Tokens) {
			t.Fatalf("not deterministic: %v vs %v", a.Tokens, again.Tokens)
		}
	}
}

// TestTranslateIntent_TokensParse closes the loop the file's doc opens: the
// output is not a third grammar, it is the SAME one. Whatever a translation
// produces must be readable by Parse.
func TestTranslateIntent_TokensParse(t *testing.T) {
	for _, phrase := range []string{
		"mes blunders en holding",
		"mes erreurs en course sans contact",
		"positions marquées commentées à l'ouverture",
		"mes bourdes en backgame",
	} {
		intent := TranslateIntent(phrase)
		if len(intent.Tokens) == 0 {
			t.Errorf("%q produced no token", phrase)
			continue
		}
		line := ""
		for _, tok := range intent.Tokens {
			line += tok + " "
		}
		if _, diags := Parse(line); len(diags) > 0 {
			t.Errorf("%q → %q was not read back cleanly: %v", phrase, line, diags)
		}
	}
}

// TestTranslateIntent_EmptyIsEmpty: a phrase nothing matches must produce no
// query at all, rather than a query that returns the whole library.
func TestTranslateIntent_EmptyIsEmpty(t *testing.T) {
	got := TranslateIntent("bonjour")
	if len(got.Tokens) != 0 || got.Board.Decision != "" || got.Board.Score != "" {
		t.Errorf("nothing understood must produce nothing: %+v", got)
	}
	if !containsString(got.Ignored, "bonjour") {
		t.Errorf("the word must be reported: %v", got.Ignored)
	}
}
