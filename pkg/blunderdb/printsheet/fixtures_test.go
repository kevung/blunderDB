package printsheet

import (
	"bytes"
	"flag"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The browser checks of the printed layout (frontend/tests/e2e/print-sheet.spec.js)
// load these documents: they are what RenderPositions and RenderMatch write
// for lists dense enough to reach the smallest body size, so that a style
// rule letting an ArkID overlap its neighbour shows there before it shows on
// paper.
var updateFixtures = flag.Bool("update-fixtures", false, "rewrite the print sheet fixtures of the e2e suite")

const fixtureDir = "../../../frontend/tests/e2e/fixtures/printsheet"

// randomArkID draws a legal-looking position: fifteen checkers a side on
// distinct points, the rest borne off, so that the ArkIDs have the lengths
// real positions give.
func randomArkID(t *testing.T, r *rand.Rand, cube bool) string {
	var p domain.Position
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Color: domain.None}
	}
	for c := domain.Black; c <= domain.White; c++ {
		n := 15 - r.IntN(4)
		for k := 0; k < n; k++ {
			for {
				pt := 1 + r.IntN(domain.NumPoints)
				q := &p.Board.Points[pt]
				if q.Checkers == 0 || q.Color == c {
					q.Color, q.Checkers = c, q.Checkers+1
					break
				}
			}
		}
		p.Board.Bearoff[c] = 15 - n
	}
	p.Cube = domain.Cube{Owner: domain.None, Value: 0}
	p.Score = [2]int{1 + r.IntN(7), 1 + r.IntN(7)}
	p.PlayerOnRoll = domain.Black
	if cube {
		p.DecisionType = domain.CubeAction
	} else {
		p.DecisionType = domain.CheckerAction
		p.Dice = [2]int{1 + r.IntN(6), 1 + r.IntN(6)}
	}
	id, err := domain.EncodeArkID(&p)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func fixtureList(t *testing.T, n int, long map[int]int) *PositionList {
	r := rand.New(rand.NewPCG(7, 11))
	l := &PositionList{Title: "Fixture"}
	words := strings.Fields("le pion arrière tient la barre pendant que le videau attend une meilleure occasion")
	for i := 1; i <= n; i++ {
		cube := i%5 == 0
		w, g := 40+r.Float64()*30, 5+r.Float64()*20
		e := &ListEntry{N: i, ArkID: randomArkID(t, r, cube), Pips: [2]int{80 + r.IntN(90), 80 + r.IntN(90)}, Win: &w, Gammon: &g}
		if cube {
			e.Cube, e.Best = true, "NoDoubleTake"
			e.Others = []Choice{{Text: "DoubleTake", CostMP: r.IntN(200)}, {Text: "DoublePass", CostMP: r.IntN(900)}}
		} else {
			e.Best = "24/18 13/11"
			e.Others = []Choice{{Text: "13/7 6/4", CostMP: r.IntN(300)}}
		}
		if size, ok := long[i]; ok {
			var b strings.Builder
			for b.Len() < size {
				b.WriteString(words[b.Len()%len(words)] + " ")
			}
			e.Comments = []string{b.String()}
		} else if i%7 == 0 {
			e.Comments = []string{"Garder l'atout du videau."}
		}
		l.Entries = append(l.Entries, e)
	}
	return l
}

func fixtureMatch(t *testing.T) *MatchSheet {
	r := rand.New(rand.NewPCG(3, 5))
	ms := &MatchSheet{Players: [2]string{"Alpha", "Bravo"}, Length: 7, Plies: 3}
	g := SheetGame{Number: 1, Away: [2]int{7, 7}, Winner: 1, Points: 2}
	for i := 1; i <= 60; i++ {
		cube := i%4 == 0
		cost := 30 + r.IntN(400)
		e := &Entry{Ref: i, Kind: KindBlunder, ArkID: randomArkID(t, r, cube), Game: 1, Row: i, Player: i % 2, CostMP: &cost}
		if cube {
			e.Cube, e.Played, e.Best = true, "DoublePass", "DoubleTake"
		} else {
			e.Played, e.Best, e.Alt, e.AltRank, e.AltCostMP = "8/5 6/5", "13/10 6/5", "24/21 6/5", 3, 40
		}
		if i%6 == 0 {
			e.Comments = []string{"Le blot sur la barre se paie cher quand l'adversaire tient quatre points."}
		}
		g.Rows = append(g.Rows, [2]Cell{{Parts: []Part{{Dice: "31", Text: "8/5 6/5", Grade: GradeBlunder, Ref: i, Kind: KindBlunder}}},
			{Parts: []Part{{Dice: "42", Text: "24/20 13/11"}}}})
		ms.Entries = append(ms.Entries, e)
	}
	ms.Games = []SheetGame{g}
	return ms
}

// fixtureCodes codes the fixture's positions as a printed list does.
func fixtureCodes(t *testing.T, l *PositionList) []TrancheCode {
	t.Helper()
	var ps []*domain.Position
	for _, e := range l.Entries {
		p, err := domain.DecodeArkID(e.ArkID)
		if err != nil {
			t.Fatal(err)
		}
		ps = append(ps, &p)
	}
	codes, err := trancheCodes(ps)
	if err != nil {
		t.Fatal(err)
	}
	return codes
}

// TestPrintFixtures: the fixtures of the e2e suite are what the renderers
// write today. Run with -update-fixtures after changing a renderer or the
// style sheet.
func TestPrintFixtures(t *testing.T) {
	_, c := catalogue("fr")
	list := func(n int, long map[int]int, o PositionsOptions) []byte {
		l := fixtureList(t, n, long)
		if o.ReimportCode {
			l.Codes = fixtureCodes(t, l)
		}
		doc, err := RenderPositions(l, o, c)
		if err != nil {
			t.Fatal(err)
		}
		return Document("fr", "fixture", []string{doc}, "")
	}
	match, err := RenderMatch(fixtureMatch(t), DefaultMatchOptions(), c)
	if err != nil {
		t.Fatal(err)
	}
	fixtures := map[string][]byte{
		"list-analysis.html":   list(400, nil, PositionsOptions{Analysis: true, Comments: true}),
		"list-plain.html":      list(400, nil, PositionsOptions{}),
		"list-long.html":       list(60, map[int]int{12: 9000}, PositionsOptions{Analysis: true, Comments: true}),
		"list-verso-long.html": list(40, map[int]int{9: 16000}, PositionsOptions{Analysis: true, Verso: true, Comments: true}),
		"list-codes.html":      list(330, nil, PositionsOptions{Analysis: true, ReimportCode: true}),
		"match.html":           Document("fr", "fixture", []string{match}, ""),
	}
	for name, want := range fixtures {
		path := filepath.Join(fixtureDir, name)
		if *updateFixtures {
			if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, want, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s is stale: go test ./pkg/blunderdb/printsheet -run TestPrintFixtures -update-fixtures", name)
		}
	}
}
