package transcript

import (
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// lastGameEnd is how a game of these tests ends: by a resignation of the loser at
// a level, or by the winner doubling and the loser refusing.
type lastGameEnd struct {
	winner int
	level  int  // resignation level; 0 with refused
	refuse bool // the winner doubles, the loser passes
}

// playGames records one short game per entry: the loser opens with 31, then the
// game ends as the entry says.
func playGames(t *testing.T, length int, games []lastGameEnd) Document {
	t.Helper()
	doc := New(length)
	for _, g := range games {
		loser := opponent(g.winner)
		// The opening roll is player 1's die then player 2's: the opener holds the higher.
		d1, d2 := 3, 1
		if loser == domain.White {
			d1, d2 = 1, 3
		}
		doc.Actions = append(doc.Actions, firstCandidate(t, doc, loser, d1, d2))
		if g.refuse {
			doc.Actions = append(doc.Actions,
				Action{Side: g.winner, Kind: KindDouble},
				Action{Side: loser, Kind: KindPass})
		} else {
			doc.Actions = append(doc.Actions, Action{Side: loser, Kind: KindResign, Level: g.level})
		}
	}
	doc.Cursor = len(doc.Actions)
	return doc
}

// TestMATLastGameOfUnfinishedMatchKeepsItsWinner: the last game of a .mat has no
// next score line, so its winner is the column of its "Wins" line. The points
// alone would crown whoever they bring to the length — at 2-0 to 3, player 2
// winning one point would be read back as player 1 winning the match.
func TestMATLastGameOfUnfinishedMatchKeepsItsWinner(t *testing.T) {
	B, W := domain.Black, domain.White
	cases := []struct {
		name     string
		length   int
		games    []lastGameEnd
		score    [2]int
		finished bool
		crawford bool // the last game is the Crawford game
	}{
		{"player 2 wins the last game at 2-0 to 3", 3,
			[]lastGameEnd{{winner: B, level: 1}, {winner: B, level: 1}, {winner: W, level: 1}}, [2]int{2, 1}, false, true},
		{"player 1 wins the last game at 0-2 to 3", 3,
			[]lastGameEnd{{winner: W, level: 1}, {winner: W, level: 1}, {winner: B, level: 1}}, [2]int{1, 2}, false, true},
		{"player 2 wins a gammon in the Crawford game at 4-0 to 5", 5,
			[]lastGameEnd{{winner: B, level: 2}, {winner: B, level: 2}, {winner: W, level: 2}}, [2]int{4, 2}, false, true},
		{"player 1 wins a gammon at 0-4 to 5", 5,
			[]lastGameEnd{{winner: W, level: 2}, {winner: W, level: 2}, {winner: B, level: 2}}, [2]int{2, 4}, false, true},
		{"player 2 wins by a refused cube at 3-0 to 5", 5,
			[]lastGameEnd{{winner: B, level: 3}, {winner: W, refuse: true}}, [2]int{3, 1}, false, false},
		{"player 1 wins by a refused cube at 0-3 to 5", 5,
			[]lastGameEnd{{winner: W, level: 3}, {winner: B, refuse: true}}, [2]int{1, 3}, false, false},
		{"player 1 wins the match in the Crawford game", 3,
			[]lastGameEnd{{winner: B, level: 2}, {winner: B, level: 1}}, [2]int{3, 0}, true, true},
		{"player 2 wins the match in the Crawford game", 3,
			[]lastGameEnd{{winner: W, level: 2}, {winner: W, level: 1}}, [2]int{0, 3}, true, true},
		{"player 1 wins the match by a refused cube after Crawford", 3,
			[]lastGameEnd{{winner: B, level: 2}, {winner: W, level: 1}, {winner: B, refuse: true}}, [2]int{3, 1}, true, false},
		{"money: player 2 wins the last game", 0,
			[]lastGameEnd{{winner: B, level: 3}, {winner: W, level: 1}}, [2]int{}, false, false},
		{"money: player 1 wins the last game by a refused cube", 0,
			[]lastGameEnd{{winner: W, level: 2}, {winner: B, refuse: true}}, [2]int{}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := playGames(t, tc.length, tc.games)
			ann := Replay(doc, 0)
			last := ann.Games[len(ann.Games)-1]
			for _, a := range ann.Actions {
				if len(a.Inconsistencies) > 0 {
					t.Logf("action %d: %+v", a.Index, a.Inconsistencies)
				}
			}
			if ann.Inconsistent() || ann.Finished != tc.finished || last.Crawford != tc.crawford ||
				(tc.length > 0 && ann.Score != tc.score) {
				t.Fatalf("setup: inconsistent=%v finished=%v crawford=%v score=%v",
					ann.Inconsistent(), ann.Finished, last.Crawford, ann.Score)
			}
			if want := tc.games[len(tc.games)-1].winner; last.Winner != want {
				t.Fatalf("setup: last game won by %d, want %d", last.Winner, want)
			}

			text := ingest.RenderMAT(MatchParts(doc))
			back, err := FromMAT(text)
			if err != nil {
				t.Fatal(err)
			}
			got := Replay(back, 0)
			if got.Inconsistent() || got.Finished != tc.finished || (tc.length > 0 && got.Score != tc.score) {
				t.Errorf(".mat round trip: inconsistent=%v finished=%v score=%v, want finished=%v score=%v\n%s",
					got.Inconsistent(), got.Finished, got.Score, tc.finished, tc.score, text)
			}
			wl, wg := matShape(doc)
			gl, gg := matShape(back)
			if wl != gl || !reflect.DeepEqual(wg, gg) {
				t.Errorf(".mat round trip:\n got %+v\nwant %+v\n%s", gg, wg, text)
			}
			// The file read back renders as the same file.
			if again := ingest.RenderMAT(MatchParts(back)); again != text {
				t.Errorf(".mat → transcription → .mat differs:\n%s\n---\n%s", text, again)
			}
		})
	}
}
