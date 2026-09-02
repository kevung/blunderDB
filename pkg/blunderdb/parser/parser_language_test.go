package parser

import (
	"errors"
	"strings"
	"testing"
)

const xgidLine = "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10"

// enCubeText is an English XG cube export, the shape every XG locale shares.
const enCubeText = xgidLine + `

X:Player   O:Opponent
Score is X:0 O:0 7 point match
Cube: 1
X on roll, cube decision?

Analyzed in Rollout
Player Winning Chances:   54.00% (G:15.00% B:1.00%)
Opponent Winning Chances: 46.00% (G:12.00% B:0.50%)

Cubeless Equities: No Double=+0.123, Double=+0.246

Cubeful Equities:
       No double:     +0.150 (-0.050)
       Double/Take:   +0.200
       Double/Pass:   +1.000 (+0.800)

Best Cube action: Double / Take

eXtreme Gammon Version: 2.19`

// TestUnsupportedXGLanguageIsReportedNotSwallowed pins the diagnostic of
// issue #175: an analysis block whose labels the parser does not know used to
// come back as an empty analysis with no error, and the position was saved
// as if it had none. XG ships in English, German, French, Spanish, Japanese,
// Greek and Russian (docs/recherche/P9-formats-de-fichiers.md); the first
// four are read, the other three are not — their exact labels have not been
// verified on a real installation, and guessed markers would parse some lines
// and miss others in silence, which is worse than refusing. The Spanish text
// below is therefore a stand-in, not a sample: what the test asserts is that
// unknown labels around an analysis block produce ErrUnrecognisedAnalysis.
// When a verified ES/EL/RU sample lands, add its markers and turn this case
// into a parse assertion.
func TestUnsupportedXGLanguageIsReportedNotSwallowed(t *testing.T) {
	es := strings.NewReplacer(
		"Player Winning Chances:", "Probabilidad de victoria del jugador:",
		"Opponent Winning Chances:", "Probabilidad de victoria del oponente:",
		"Cubeless Equities:", "Equidades sin cubo:",
		"Cubeful Equities:", "Equidades con cubo:",
		"No double:", "No doblar:",
		"Double/Take:", "Doblar/Aceptar:",
		"Double/Pass:", "Doblar/Rechazar:",
		"Best Cube action:", "Mejor acción del cubo:",
	).Replace(enCubeText)

	_, err := ParsePosition(es)
	if !errors.Is(err, ErrUnrecognisedAnalysis) {
		t.Fatalf("stand-in Spanish cube block: got %v, want ErrUnrecognisedAnalysis", err)
	}

	// Same for a checker list: XG writes "eq:" in most locales, so the move
	// lines read, but the win chances under them do not — and 0 % for every
	// candidate on both sides is not a result, it is a hole.
	checker := xgidLine + `

X:Player   O:Opponent
Score is X:0 O:0 7 point match
X to play 31

    1. 4-ply       8/5 6/5                      eq:+0.170
      Jugador:  56.00% (G:16.00% B:0.80%)
      Oponente: 44.00% (G:11.00% B:0.40%)

    2. 4-ply       24/21 13/12                  eq:+0.050 (-0.120)
      Jugador:  52.00% (G:13.00% B:0.60%)
      Oponente: 48.00% (G:12.00% B:0.50%)

eXtreme Gammon Version: 2.19`
	if _, err := ParsePosition(checker); !errors.Is(err, ErrUnrecognisedAnalysis) {
		t.Fatalf("stand-in Spanish checker list: got %v, want ErrUnrecognisedAnalysis", err)
	}

	// The English original still parses, so the diagnostic is about the
	// labels and not about the shape of the block.
	res, err := ParsePosition(enCubeText)
	if err != nil {
		t.Fatalf("English cube block: %v", err)
	}
	if res.Analysis.AnalysisType != "DoublingCube" || res.Analysis.DoublingCubeAnalysis.PlayerWinChances != 54 {
		t.Errorf("English cube block misread: %+v", res.Analysis.DoublingCubeAnalysis)
	}
}

// TestPositionWithoutAnalysisIsNotAnError: the diagnostic must not fire on
// what has no analysis to read — a bare XGID, or a board diagram with the
// score and the roll but no evaluation under it.
func TestPositionWithoutAnalysisIsNotAnError(t *testing.T) {
	for name, text := range map[string]string{
		"bare XGID": xgidLine,
		"board only": xgidLine + `

X:Player   O:Opponent
Score is X:0 O:0 7 point match
 +13-14-15-16-17-18------19-20-21-22-23-24-+
 |                  |   |                  |
 +12-11-10--9--8--7-------6--5--4--3--2--1-+
Pip count  X: 167  O: 167 X-O: 0-0/7
Cube: 1
X to play 31

eXtreme Gammon Version: 2.19`,
	} {
		res, err := ParsePosition(text)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if res.Analysis.AnalysisType != "" {
			t.Errorf("%s: analysis type %q, want none", name, res.Analysis.AnalysisType)
		}
	}
}

// TestCommaIsNormalisedOnlyInNumbers: the JS parser rewrote every comma of
// the text into a dot so that "0,491" would read as a number; the comment
// under the analysis went through the same rewrite. Numbers in either
// notation must read the same, and prose must keep its punctuation.
func TestCommaIsNormalisedOnlyInNumbers(t *testing.T) {
	withComment := strings.Replace(enCubeText,
		"Best Cube action: Double / Take\n",
		"Best Cube action: Double / Take\n\nA note, with commas, and 1,5 as a number.\n", 1)
	res, err := ParsePosition(withComment)
	if err != nil {
		t.Fatalf("ParsePosition: %v", err)
	}
	if want := "A note, with commas, and 1,5 as a number."; res.Comment != want {
		t.Errorf("comment = %q, want %q", res.Comment, want)
	}

	// German XG prints decimals with a comma; the numbers must come out equal
	// to the English ones, and the cube-line separator must survive it.
	de := strings.NewReplacer(
		"Player Winning Chances:", "Spieler Gewinnchancen:",
		"Opponent Winning Chances:", "Gewinnchancen des Gegners:",
		"Cubeless Equities:", "Equities ohne Dopplerwürfel:",
		"Cubeful Equities:", "Equities mit Dopplerwürfel:",
		"No Double=+0.123, Double=+0.246", "Nicht Doppeln=+0,123, Doppeln=+0,246",
		"No double:", "Nicht Doppeln:",
		"Double/Take:", "Doppeln/Annehmen:",
		"Double/Pass:", "Doppeln/Ablehnen:",
		"Best Cube action:", "Beste Dopplerwürfel Aktion",
		"Analyzed in", "Analysiert in",
		"X:Player   O:Opponent", "X:Spieler   O:Gegner",
		"54.00% (G:15.00% B:1.00%)", "54,00% (G:15,00% B:1,00%)",
		"46.00% (G:12.00% B:0.50%)", "46,00% (G:12,00% B:0,50%)",
		"+0.150 (-0.050)", "+0,150 (-0,050)",
		"+0.200", "+0,200",
		"+1.000 (+0.800)", "+1,000 (+0,800)",
	).Replace(enCubeText)
	en, err := ParsePosition(enCubeText)
	if err != nil {
		t.Fatalf("English: %v", err)
	}
	got, err := ParsePosition(de)
	if err != nil {
		t.Fatalf("German with comma decimals: %v", err)
	}
	e, g := en.Analysis.DoublingCubeAnalysis, got.Analysis.DoublingCubeAnalysis
	if e.PlayerWinChances != g.PlayerWinChances || e.PlayerGammonChances != g.PlayerGammonChances ||
		e.CubelessNoDoubleEquity != g.CubelessNoDoubleEquity || e.CubelessDoubleEquity != g.CubelessDoubleEquity ||
		e.CubefulNoDoubleEquity != g.CubefulNoDoubleEquity || e.CubefulNoDoubleError != g.CubefulNoDoubleError ||
		e.CubefulDoublePassEquity != g.CubefulDoublePassEquity || e.CubefulDoublePassError != g.CubefulDoublePassError {
		t.Errorf("comma decimals read differently:\n en %+v\n de %+v", *e, *g)
	}
	if g.CubelessDoubleEquity != 0.246 {
		t.Errorf("German cubeless double equity = %v, want 0.246 (the comma before \"Doppeln=\" must not be eaten)", g.CubelessDoubleEquity)
	}
}
