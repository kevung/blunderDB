package transcript

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// openedAt is a 5-point document whose first game is declared at score, its
// opening play by player 1 followed by player 2's double.
func openedAt(t *testing.T, score [2]int, postCrawford bool) Document {
	t.Helper()
	doc := docOf(5)
	doc.Header.PostCrawford = postCrawford
	first := firstCandidate(t, doc, domain.Black, 6, 3)
	first.Score = &score
	doc.Actions = append(doc.Actions, first, Action{Side: domain.White, Kind: KindDouble})
	doc.Cursor = len(doc.Actions)
	return doc
}

func hasRule(info ActionInfo, rule InconsistencyRule) bool {
	for _, inc := range info.Inconsistencies {
		if inc.Rule == rule {
			return true
		}
	}
	return false
}

func TestPostCrawfordDeclaration(t *testing.T) {
	t.Run("declared at one point away: the cube is alive", func(t *testing.T) {
		ann := Replay(openedAt(t, [2]int{4, 2}, true), 0)
		if ann.Games[0].Crawford {
			t.Fatal("a game declared post-Crawford is the Crawford game")
		}
		if ann.Actions[1].Inconsistencies != nil {
			t.Fatalf("the double is marked: %+v", ann.Actions[1].Inconsistencies)
		}
		if got := ann.Actions[0].Before.Score; got != [2]int{domain.PostCrawford, 3} {
			t.Errorf("away score = %v, want [0 3]", got)
		}
	})

	t.Run("not declared: the Crawford game, as before", func(t *testing.T) {
		ann := Replay(openedAt(t, [2]int{4, 2}, false), 0)
		if !ann.Games[0].Crawford || !hasRule(ann.Actions[1], RuleCrawfordCube) {
			t.Fatalf("Crawford = %v, double %+v", ann.Games[0].Crawford, ann.Actions[1].Inconsistencies)
		}
	})

	t.Run("no player one point away: the declaration is ignored", func(t *testing.T) {
		doc := openedAt(t, [2]int{3, 2}, true)
		doc.Actions = append(doc.Actions,
			Action{Side: domain.Black, Kind: KindTake},
			Action{Side: domain.Black, Kind: KindResign, Level: 1}, // 3-4 at a cube of 2: game 2 is Crawford
		)
		doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 5, 2),
			Action{Side: domain.White, Kind: KindDouble})
		ann := Replay(doc, 0)
		if len(ann.Games) != 2 || ann.Games[0].Crawford || !ann.Games[1].Crawford {
			t.Fatalf("games = %+v", ann.Games)
		}
		if !hasRule(ann.Actions[len(ann.Actions)-1], RuleCrawfordCube) {
			t.Fatal("the double of the Crawford game is not marked")
		}
	})

	t.Run("a gesture states it and the cached Replay follows", func(t *testing.T) {
		var r Replayer
		doc := openedAt(t, [2]int{4, 2}, false)
		if !r.Replay(doc, 0).Games[0].Crawford {
			t.Fatal("not Crawford before the declaration")
		}
		out, err := Apply(doc, Gesture{Kind: GestureSetPostCrawford, PostCrawford: true})
		if err != nil {
			t.Fatal(err)
		}
		if ann := r.Replay(out, 0); !out.Header.PostCrawford || ann.Games[0].Crawford || ann.Actions[1].Inconsistencies != nil {
			t.Fatalf("header %v, games %+v", out.Header.PostCrawford, ann.Games)
		}
	})
}

// TestPostCrawfordSurvivesTheMAT: the .mat states the declaration in a header
// line, and reading it back gives a draft that declares it again.
func TestPostCrawfordSurvivesTheMAT(t *testing.T) {
	for _, declared := range []bool{true, false} {
		text := ingest.RenderMAT(MatchParts(openedAt(t, [2]int{4, 2}, declared)))
		if got := strings.Contains(text, domain.MATPostCrawfordHeader); got != declared {
			t.Fatalf("declared %v: header written = %v\n%s", declared, got, text)
		}
		back, err := FromMAT(text)
		if err != nil {
			t.Fatal(err)
		}
		if back.Header.PostCrawford != declared {
			t.Fatalf("declared %v: read back %v", declared, back.Header.PostCrawford)
		}
		if Replay(back, 0).Games[0].Crawford == declared {
			t.Fatalf("declared %v: the Crawford mention did not survive", declared)
		}
	}
}
