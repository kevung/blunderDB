package ingest

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/gnubgparser"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestWinnerEncodingRoundTrip: every format stores Game.Winner in the one
// encoding (domain.WinnerPlayer1 / WinnerPlayer2), the scores agree with it —
// the winner's score plus the points is where the next game starts, and the
// last game takes its winner to the length — and the .mat rendered from the
// stored match names the same winner of every game as the source does.
func TestWinnerEncodingRoundTrip(t *testing.T) {
	const root = "../../../testdata/"
	// Each fixture with the winner of its games, by side, as its scores give
	// them. The three charlot files are one match, as are the three test ones.
	const p1, p2 = domain.WinnerPlayer1, domain.WinnerPlayer2
	charlot := []int32{p2, p1, p1, p1}
	testMatch := []int32{p1, p1, p2, p2, p2, p1, p1}
	fixtures := map[string][]int32{
		"charlot1-charlot2_7p_2025-11-08-2305.xg":  charlot,
		"charlot1-charlot2_7p_2025-11-08-2305.mat": charlot,
		"charlot1-charlot2_7p_2025-11-08-2305.sgf": charlot,
		"test.xg":  testMatch,
		"test.mat": testMatch,
		"test.sgf": testMatch,
		"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg": {p1, p2, p1, p1, p1},
		"gnubg_selfplay_drops_7p.mat":                                     {p1, p2, p2, p2, p1, p2},
		"gnubg_roll_then_resign_1p.mat":                                   {p1},
		"TachiAI_V_player_Nov_2__2025__16_55.bgf":                         {p1},
	}
	ctx := context.Background()
	for name, want := range fixtures {
		t.Run(name, func(t *testing.T) {
			var graph *MatchGraph
			var err error
			switch filepath.Ext(name) {
			case ".xg":
				graph, err = MapXG(root + name)
			case ".bgf":
				graph, err = MapBGF(root + name)
			default:
				graph, err = MapGnuBG(root + name)
			}
			if err != nil {
				t.Fatalf("map: %v", err)
			}
			s, err := sqlite.Open(ctx, ":memory:", nil)
			if err != nil {
				t.Fatalf("sqlite.Open: %v", err)
			}
			defer s.Close()
			res := writeGraph(t, s, graph)
			m, games, moves, err := ReadMatchForMAT(ctx, s, "", res.MatchID)
			if err != nil {
				t.Fatalf("ReadMatchForMAT: %v", err)
			}
			if len(games) != len(want) {
				t.Fatalf("%d games stored, want %d", len(games), len(want))
			}
			for i, g := range games {
				if g.Winner != want[i] {
					t.Errorf("game %d: stored winner %d, want %d", i+1, g.Winner, want[i])
				}
				end := g.InitialScore
				end[domain.WinnerSide(g.Winner)] += g.PointsWon
				if i+1 < len(games) && end != games[i+1].InitialScore {
					t.Errorf("game %d: winner %d ends at %v, next game starts at %v", i+1, g.Winner, end, games[i+1].InitialScore)
				}
				if i+1 == len(games) && end[domain.WinnerSide(g.Winner)] < m.MatchLength {
					t.Errorf("last game: winner %d ends at %v, short of %d", g.Winner, end, m.MatchLength)
				}
			}

			// The stats credit the match to the winner of its last game.
			rows, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
			if err != nil {
				t.Fatalf("PlayerTable: %v", err)
			}
			winner, loser := m.Player1Name, m.Player2Name
			if want[len(want)-1] == p2 {
				winner, loser = loser, winner
			}
			for _, r := range rows {
				if (r.Name == winner && r.Wins != 1) || (r.Name == loser && r.Losses != 1) {
					t.Errorf("player table: %s has %d wins, %d losses; the match is %s's", r.Name, r.Wins, r.Losses, winner)
				}
			}

			out := RenderMAT(m, games, moves)
			rt, err := gnubgparser.ParseMAT(strings.NewReader(out))
			if err != nil {
				t.Fatalf("re-parse: %v\n%s", err, out)
			}
			if len(rt.Games) != len(want) {
				t.Fatalf("rendered %d games, want %d", len(rt.Games), len(want))
			}
			for i, g := range rt.Games {
				if got := domain.WinnerFromSide(g.Winner); got != want[i] || int32(g.Points) != games[i].PointsWon {
					t.Errorf("rendered game %d: winner %d for %d points, want %d for %d", i+1, got, g.Points, want[i], games[i].PointsWon)
				}
			}
		})
	}
}

// TestBGFWinnerFromScores: BGBlitz records no winner on a game. The next game's
// scores name it, and for the last game the match's final score does; a game
// whose scores name nobody stays unfinished.
func TestBGFWinnerFromScores(t *testing.T) {
	game := func(green, red, won int) interface{} {
		return map[string]interface{}{"scoreGreen": green, "scoreRed": red, "wonPoints": won}
	}
	games := []interface{}{game(0, 0, 2), game(2, 0, 1), game(2, 1, 4)}
	cases := []struct {
		name string
		data map[string]interface{}
		want []int32
	}{
		{"final score", map[string]interface{}{"finalGreen": 2, "finalRed": 5},
			[]int32{domain.WinnerPlayer1, domain.WinnerPlayer2, domain.WinnerPlayer2}},
		{"no final score", map[string]interface{}{},
			[]int32{domain.WinnerPlayer1, domain.WinnerPlayer2, domain.WinnerUnfinished}},
	}
	for _, c := range cases {
		for i, w := range c.want {
			if got := bgfWinner(c.data, games, i); got != w {
				t.Errorf("%s, game %d: winner %d, want %d", c.name, i+1, got, w)
			}
		}
	}
	unplayed := []interface{}{game(0, 0, 0)}
	if got := bgfWinner(map[string]interface{}{"finalGreen": 0, "finalRed": 0}, unplayed, 0); got != domain.WinnerUnfinished {
		t.Errorf("a game worth no points: winner %d, want unfinished", got)
	}
}

// TestGnuBGWinnerNeedsPoints: an SGF game without a result parses as player
// 1's (the parser's zero value) for no points; it is unfinished.
func TestGnuBGWinnerNeedsPoints(t *testing.T) {
	if got := gnuBGWinner(gnubgparser.Game{Winner: 0, Points: 0}); got != domain.WinnerUnfinished {
		t.Errorf("no points: winner %d, want unfinished", got)
	}
	if got := gnuBGWinner(gnubgparser.Game{Winner: 1, Points: 2}); got != domain.WinnerPlayer2 {
		t.Errorf("gnubg 1: winner %d, want player 2", got)
	}
}
