// Package matrender writes a stored match as a Jellyfish/gnubg .mat text. It
// depends on domain alone, so the transcript package's tests can render a
// transcription without pulling in ingest (which imports transcript).
package matrender

import (
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Render writes match m with its games (in order) and each game's moves
// (keyed by game id, in order) as a .mat transcript.
func Render(m *domain.Match, games []*domain.Game, movesByGame map[int64][]*domain.Move) string {
	var b strings.Builder
	p1, p2 := orDefault(m.Player1Name, "Player 1"), orDefault(m.Player2Name, "Player 2")

	// PGN-style headers help XG show names/event; all optional to the parser.
	// Each is written only when it carries something: gnubg's reader stores
	// whatever is between the quotes, so an empty [Site ""] would replace "we
	// do not know where" by "nowhere" on every round-trip.
	if m.Event != "" {
		fmt.Fprintf(&b, "; [Event \"%s\"]\n", m.Event)
	}
	fmt.Fprintf(&b, "; [Player 1 \"%s\"]\n", p1)
	fmt.Fprintf(&b, "; [Player 2 \"%s\"]\n", p2)
	if m.Location != "" {
		fmt.Fprintf(&b, "; [Site \"%s\"]\n", m.Location)
	}
	if m.Round != "" {
		fmt.Fprintf(&b, "; [Round \"%s\"]\n", m.Round)
	}
	// The dotted YYYY.MM.DD is the only shape the reader's [EventDate] regex
	// accepts — an ISO date with dashes goes back in as no date at all.
	if !m.MatchDate.IsZero() {
		fmt.Fprintf(&b, "; [EventDate \"%s\"]\n", m.MatchDate.Format("2006.01.02"))
	}
	// Who typed the match in. It is never a column of `match` (ADR-0045): the
	// header exists so a transcribed .mat says whose reading it is, and it is
	// the transcription graph, not the database, that supplies it.
	if m.Transcriber != "" {
		fmt.Fprintf(&b, "; [Transcriber \"%s\"]\n", m.Transcriber)
	}
	if m.PostCrawford && m.MatchLength > 0 {
		b.WriteString(domain.MATPostCrawfordHeader + "\n")
	}
	b.WriteString("\n")

	// A money session has no match length. gnubg/Jellyfish (and gnubgparser,
	// our round-trip gate) encode it as "0 point match" — 0 means money game.
	// Normalise the Unlimited (-1) sentinel to 0: the parser's `\d+` header
	// regex rejects "-1 point match".
	matchLen := m.MatchLength
	if matchLen < 0 {
		matchLen = 0
	}
	fmt.Fprintf(&b, "%d point match\n\n", matchLen)

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

	rows := layoutRows(movesToPlays(moves))

	// The result is a cell like any other, and it belongs to the WINNER: gnubg
	// writes " Wins N points" in the winner's column, appended to the running
	// line when that column is still free there and otherwise on a line of its
	// own — a line with no number, because no turn was played on it. At the
	// margin it would read as player 1's win on a round-trip.
	unnumbered := -1
	if wins := winsLine(m, g); wins != "" {
		last := len(rows) - 1
		// Player 2's column can take the cell on the spot; player 1's opens a
		// line, exactly as an ordinary play of theirs would.
		if g.Winner == domain.WinnerPlayer2 && last >= 0 && rows[last].left != "" && rows[last].right == "" {
			rows[last].right = wins
		} else if g.Winner == domain.WinnerPlayer1 {
			rows = append(rows, matRow{left: wins})
			unnumbered = len(rows) - 1
		} else {
			rows = append(rows, matRow{right: wins})
			unnumbered = len(rows) - 1
		}
	}

	for i, row := range rows {
		// The unnumbered line still starts at the column the numbers vacate, so
		// the cell it carries stays under the ones above it.
		head := fmt.Sprintf("%3d) ", i+1)
		if i == unnumbered {
			head = strings.Repeat(" ", len(head))
		}
		text := head + row.left + gap(len(row.left), 30) + row.right
		b.WriteString(strings.TrimRight(text, " ") + "\n")
	}
}

// matRow is one numbered line of a .mat game: player 1's cell and player 2's.
type matRow struct{ left, right string }

// layoutRows deals the cells into lines. A play by player 1 opens a line; a play
// by player 2 joins the line player 1 just opened, or takes one of its own with
// the left column blank.
func layoutRows(plays []play) []matRow {
	var rows []matRow
	for i := 0; i < len(plays); {
		var row matRow
		if plays[i].player == 1 {
			row.left = plays[i].text
			i++
			if i < len(plays) && plays[i].player == -1 {
				row.right = plays[i].text
				i++
			}
		} else {
			row.right = plays[i].text
			i++
		}
		rows = append(rows, row)
	}
	return rows
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
		// Checker move: "DD: notation". A forced no-move (dance) is stored as the
		// display marker "Cannot Move"; a .mat leaves the cell blank after the
		// dice, so emit dice only for it (and for an empty notation).
		//
		// A play the record does not carry is NOT that: it is stored as "???", the
		// mark gnubg itself writes, and it goes out as it came in. Blanking it here
		// would turn a play nobody wrote down into a dance the player never had —
		// see transcript.KindUnrecorded.
		cell := fmt.Sprintf("%d%d:", mv.Dice[0], mv.Dice[1])
		if mv.CheckerMove != "" && mv.CheckerMove != "Cannot Move" {
			cell += " " + mv.CheckerMove
		}
		plays = append(plays, play{mv.Player, cell})
	}
	return plays
}

// winsLine renders the game result, empty for an unfinished game.
func winsLine(m *domain.Match, g *domain.Game) string {
	side := domain.WinnerSide(g.Winner)
	if g.PointsWon <= 0 || side == domain.None {
		return ""
	}
	pts := int(g.PointsWon)
	unit := "point"
	if pts != 1 {
		unit = "points"
	}
	// "and the match" when the win reaches the match length.
	winnerScoreBefore := g.InitialScore[side]
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
