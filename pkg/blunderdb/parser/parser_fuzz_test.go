package parser

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// FuzzParsePosition feeds arbitrary text into the clipboard/file position
// parser. The text comes straight from user paste and imported files, so the
// parser must never panic — it may only return an error. Its own doc comment
// promises "It never panics"; this guards that promise across the regex-heavy
// language-detection and decode paths.
func FuzzParsePosition(f *testing.F) {
	seeds := []string{
		"",
		"   ",
		"XGID=bA----D-C---dE---d-e----B-:0:0:1:00:0:0:3:0:10",
		"XGID=bA----D-C---dE---d-e----B-:0:0:1:00:0:0:3:0:10\n\n" +
			"X:Joueur 1   O:Joueur 2\nLe score est X:0 O:0. Partie illimitée",
		"Analysis:\nChecker Move Analysis:\nfoo",
		"Analysis:\nDoubling Cube Analysis:\nbar",
		"Spieler Gegner Dopplerwürfel",
		"プレーヤー 対戦相手 キューブ",
		"not an xgid at all, just prose, 3.14, 1,5",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, text string) {
		res, err := ParsePosition(text)
		if err != nil {
			return
		}
		// A successful parse must yield a board that round-trips into a
		// well-formed 26-char XGID board string; a corrupt decode surfaces here.
		if board := domain.EncodeXGIDBoard(&res.Position); len(board) != 26 {
			t.Fatalf("ParsePosition(%q) returned no error but a %d-char board: %q",
				trunc(text), len(board), board)
		}
	})
}

// trunc keeps fuzz failure messages readable for large inputs.
func trunc(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}
