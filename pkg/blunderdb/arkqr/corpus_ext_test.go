package arkqr_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// TestRoundTripImportedFormats archives the matches of testdata that come
// as .xg and .sgf, read as a stored match is: every Action comes back with
// its kind, side, dice and boards, and coding again gives the same bytes.
func TestRoundTripImportedFormats(t *testing.T) {
	root := "../../../testdata"
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		var g *ingest.MatchGraph
		path := filepath.Join(root, e.Name())
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".xg":
			g, err = ingest.MapXG(path)
		case ".sgf":
			g, err = ingest.MapGnuBG(path)
		default:
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		doc := graphDocument(t, g)
		data, err := arkqr.Encode(arkqr.Match{Doc: doc})
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		back, err := arkqr.Decode(data)
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		a1, a2 := transcript.Replay(doc, 0), transcript.Replay(back.Doc, 0)
		if len(a1.Actions) != len(a2.Actions) || back.Doc.Header.Player1 != doc.Header.Player1 {
			t.Fatalf("%s: %d actions back for %d", e.Name(), len(a2.Actions), len(a1.Actions))
		}
		for i := range a1.Actions {
			x, y := doc.Actions[i], back.Doc.Actions[i]
			if x.Kind != y.Kind || x.Side != y.Side || x.Dice != y.Dice ||
				!domain.SameBoard(a1.Actions[i].Before.Board, a2.Actions[i].Before.Board) ||
				!domain.SameBoard(a1.Actions[i].After, a2.Actions[i].After) {
				t.Fatalf("%s: action %d differs", e.Name(), i)
			}
		}
		if again, err := arkqr.Encode(back); err != nil || !bytes.Equal(again, data) {
			t.Fatalf("%s: coding the decoded match again changes it", e.Name())
		}
		n++
	}
	if n < 6 {
		t.Fatalf("%d matches", n)
	}
}

func graphDocument(t *testing.T, g *ingest.MatchGraph) transcript.Document {
	t.Helper()
	m := g.Match
	var games []*domain.Game
	moves := map[int64][]*domain.Move{}
	for i := range g.Games {
		gm := g.Games[i].Game
		gm.ID = int64(i + 1)
		games = append(games, &gm)
		for j := range g.Games[i].Moves {
			mv := g.Games[i].Moves[j].Move
			mv.GameID = gm.ID
			moves[gm.ID] = append(moves[gm.ID], &mv)
		}
	}
	doc, err := transcript.FromMAT(ingest.RenderMAT(&m, games, moves))
	if err != nil {
		t.Fatalf("FromMAT: %v", err)
	}
	return doc
}
