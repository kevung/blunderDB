package ingest

import (
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// mat_export.go renders a stored match as a Jellyfish/gnubg .mat text — the
// inverse of the .mat parser, and the format XG re-imports. It is the writer
// blunderDB lacked (import was read-only). The output round-trips through
// gnubgparser.ParseMAT; analysis is NOT part of .mat (a pure move transcript).

// Move.Player encoding is XG {1 = player 1, -1 = player 2}; Game.Winner is the
// gnubg encoding {0 = player 1, 1 = player 2, -1 = unfinished}.

// RenderMAT writes match m with its games (in order) and each game's moves
// (keyed by game id, in order) as a .mat transcript.
func RenderMAT(m *domain.Match, games []*domain.Game, movesByGame map[int64][]*domain.Move) string {
	var b strings.Builder
	p1, p2 := orDefault(m.Player1Name, "Player 1"), orDefault(m.Player2Name, "Player 2")

	// PGN-style headers help XG show names/event; all optional to the parser.
	if m.Event != "" {
		fmt.Fprintf(&b, "; [Event \"%s\"]\n", m.Event)
	}
	fmt.Fprintf(&b, "; [Player 1 \"%s\"]\n", p1)
	fmt.Fprintf(&b, "; [Player 2 \"%s\"]\n\n", p2)

	fmt.Fprintf(&b, "%d point match\n\n", m.MatchLength)

	for _, g := range games {
		renderMATGame(&b, m, p1, p2, g, movesByGame[g.ID])
		b.WriteString("\n")
	}
	return b.String()
}

// play is one half-line cell: which player made it and the rendered text.
type play struct {
	player int32 // 1 or -1
	text   string
}

func renderMATGame(b *strings.Builder, m *domain.Match, p1, p2 string, g *domain.Game, moves []*domain.Move) {
	fmt.Fprintf(b, " Game %d\n", g.GameNumber)
	left := fmt.Sprintf(" %s : %d", p1, g.InitialScore[0])
	fmt.Fprintf(b, "%s%s%s : %d\n", left, gap(len(left), 30), p2, g.InitialScore[1])

	plays := movesToPlays(moves)
	line := 1
	for i := 0; i < len(plays); {
		var l, r string
		if plays[i].player == 1 {
			l = plays[i].text
			i++
			if i < len(plays) && plays[i].player == -1 {
				r = plays[i].text
				i++
			}
		} else {
			r = plays[i].text
			i++
		}
		row := fmt.Sprintf("%3d) %s%s%s", line, l, gap(len(l), 30), r)
		b.WriteString(strings.TrimRight(row, " ") + "\n")
		line++
	}

	if wins := winsLine(m, g); wins != "" {
		b.WriteString(wins + "\n")
	}
}

// movesToPlays turns stored moves into ordered cells, expanding a combined cube
// action ("Double/Take", "Double/Pass") into the doubler's offer + the
// opponent's response, and tracking the cube value for "Doubles => N".
func movesToPlays(moves []*domain.Move) []play {
	var plays []play
	cube := 1
	for _, mv := range moves {
		if mv.MoveType == "cube" {
			switch mv.CubeAction {
			case "Double":
				cube *= 2
				plays = append(plays, play{mv.Player, fmt.Sprintf(" Doubles => %d", cube)})
			case "Take":
				plays = append(plays, play{mv.Player, " Takes"})
			case "Pass":
				plays = append(plays, play{mv.Player, " Drops"})
			case "Double/Take":
				cube *= 2
				plays = append(plays,
					play{mv.Player, fmt.Sprintf(" Doubles => %d", cube)},
					play{-mv.Player, " Takes"})
			case "Double/Pass":
				cube *= 2
				plays = append(plays,
					play{mv.Player, fmt.Sprintf(" Doubles => %d", cube)},
					play{-mv.Player, " Drops"})
			case "No Double", "":
				// Not represented in a .mat transcript — skip.
			}
			continue
		}
		// Checker move: "DD: notation" (notation may be empty for a forced no-move).
		cell := fmt.Sprintf("%d%d:", mv.Dice[0], mv.Dice[1])
		if mv.CheckerMove != "" {
			cell += " " + mv.CheckerMove
		}
		plays = append(plays, play{mv.Player, cell})
	}
	return plays
}

// winsLine renders the game result. Game.Winner is 0=p1, 1=p2, -1=unfinished.
func winsLine(m *domain.Match, g *domain.Game) string {
	if g.PointsWon <= 0 || (g.Winner != 0 && g.Winner != 1) {
		return ""
	}
	pts := int(g.PointsWon)
	unit := "point"
	if pts != 1 {
		unit = "points"
	}
	// "and the match" when the win reaches the match length.
	winnerScoreBefore := g.InitialScore[int(g.Winner)]
	andMatch := ""
	if m.MatchLength > 0 && int(winnerScoreBefore)+pts >= int(m.MatchLength) {
		andMatch = " and the match"
	}
	return fmt.Sprintf(" Wins %d %s%s", pts, unit, andMatch)
}

// gap returns spacing that keeps at least 3 spaces between columns (the parser
// splits cells on runs of ≥3 spaces) while aligning to a minimum width.
func gap(used, width int) string {
	n := width - used
	if n < 3 {
		n = 3
	}
	return strings.Repeat(" ", n)
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
