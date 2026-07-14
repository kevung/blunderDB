package parser

import (
	"testing"

	"golang.org/x/text/unicode/norm"
)

// macOS's pasteboard hands accented Latin text back in NFD (decomposed) form
// for some sources — observed with XG run under Sikarugir/Wine — while every
// accent-bearing literal in this package is written in NFC (precomposed).
// Without normalizing the input first, a pasted French analysis silently
// parses to an empty AnalysisType (github.com/kevung/blunderdb#105).
func TestParsePositionNFDClipboard(t *testing.T) {
	const checkerFR = "XGID=---BBCBAA---bB---bBb-bbcb-:1:1:1:43:0:0:0:9:10\n\n" +
		"X:Nicolas   O:Kévin\n" +
		"Le score est X:0 O:0 match en 9 pt(s)\n" +
		"Videau: 2, X a le videau\n" +
		"X à jouer 43\n\n" +
		"    1. XG Roller++ 8/4 5/2                      éq:-0.253\n" +
		"      Joueur:     33.09% (G:3.12% B:0.06%)\n" +
		"      Adversaire: 66.91% (G:3.32% B:0.05%)\n\n" +
		"eXtreme Gammon Version: 2.19.211.pre-release"

	const cubeFR = "XGID=bA----D-C---dE---d-e----B-:0:0:1:00:0:0:3:0:10\n\n" +
		"X:Joueur 1   O:Joueur 2\n" +
		"Le score est X:0 O:0. Partie illimitée, Jacoby Beaver\n" +
		"Videau: 1\n" +
		"X: lance ou double\n\n" +
		"Analysé avec XG Roller++\n" +
		"Chance de gain du joueur: 54.09% (G:20.06% B:0.47%)\n" +
		"Chance de gain de l'adversaire: 45.91% (G:12.14% B:0.58%)\n\n" +
		"Equités sans videau: Pas de double=+0.160, Double=+0.324\n\n" +
		"Equités avec videau\n" +
		"       Pas de double: +0.224\n" +
		"       Double/Beaver: -0.021 (-0.245)\n" +
		"       Double/Passe:  +1.000 (+0.776)\n\n" +
		"Meilleur action du videau: Pas de double / Beaver\n" +
		"Pourcentage de passes incorrectes pour rendre la décision de double correcte: 24.0%\n\n" +
		"eXtreme Gammon Version: 2.19.211.pre-release"

	cases := []struct {
		name         string
		input        string
		wantType     string
		wantMoveText string
	}{
		{"checker_fr", checkerFR, "CheckerMove", "8/4 5/2"},
		{"cube_fr", cubeFR, "DoublingCube", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			decomposed := norm.NFD.String(c.input)
			if decomposed == c.input {
				t.Fatalf("fixture has no decomposable accents; test wouldn't exercise NFD")
			}

			res, err := ParsePosition(decomposed)
			if err != nil {
				t.Fatalf("ParsePosition(NFD input): %v", err)
			}
			if res.Analysis.AnalysisType != c.wantType {
				t.Fatalf("AnalysisType = %q, want %q (analysis silently dropped on decomposed input)", res.Analysis.AnalysisType, c.wantType)
			}
			if c.wantMoveText != "" {
				if res.Analysis.CheckerAnalysis == nil || len(res.Analysis.CheckerAnalysis.Moves) == 0 {
					t.Fatalf("no checker moves parsed from NFD input")
				}
				if got := res.Analysis.CheckerAnalysis.Moves[0].Move; got != c.wantMoveText {
					t.Errorf("first move = %q, want %q", got, c.wantMoveText)
				}
			}

			// The NFC form must parse identically, confirming NFD was the only gap.
			nfcRes, err := ParsePosition(c.input)
			if err != nil {
				t.Fatalf("ParsePosition(NFC input): %v", err)
			}
			if nfcRes.Analysis.AnalysisType != c.wantType {
				t.Fatalf("NFC AnalysisType = %q, want %q", nfcRes.Analysis.AnalysisType, c.wantType)
			}
		})
	}
}
