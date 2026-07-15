package ingest

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// mat_export.go renders a stored match as a Jellyfish/gnubg .mat text — the
// inverse of the .mat parser, and the format XG re-imports. It is the writer
// blunderDB lacked (import was read-only). The output round-trips through
// gnubgparser.ParseMAT; analysis is NOT part of .mat (a pure move transcript).

// Move.Player encoding is XG {1 = player 1, -1 = player 2}; Game.Winner is the
// gnubg encoding {0 = player 1, 1 = player 2, -1 = unfinished}.

// ReadMatchForMAT reconstructs a stored match into the shape RenderMAT wants:
// the match header, its games in order, and each game's moves keyed by game id.
// It is the read-side counterpart WriteMatch never had.
func ReadMatchForMAT(ctx context.Context, s storage.Storage, scope string, matchID int64) (*domain.Match, []*domain.Game, map[int64][]*domain.Move, error) {
	m, err := s.Matches().Get(ctx, scope, matchID)
	if err != nil {
		return nil, nil, nil, err
	}
	var games []*domain.Game
	for g, err := range s.Matches().Games(ctx, scope, matchID) {
		if err != nil {
			return nil, nil, nil, err
		}
		games = append(games, g)
	}
	movesByGame := map[int64][]*domain.Move{}
	for mv, err := range s.Matches().MovesByMatch(ctx, scope, matchID) {
		if err != nil {
			return nil, nil, nil, err
		}
		movesByGame[mv.GameID] = append(movesByGame[mv.GameID], mv)
	}
	return m, games, movesByGame, nil
}

// ExportMatchMAT reads a stored match and writes its .mat transcript to w.
func ExportMatchMAT(ctx context.Context, s storage.Storage, scope string, matchID int64, w io.Writer) error {
	m, games, moves, err := ReadMatchForMAT(ctx, s, scope, matchID)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, RenderMAT(m, games, moves))
	return err
}

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

	// A money session has no match length. gnubg/Jellyfish (and gnubgparser,
	// our round-trip gate) encode it as "0 point match" — 0 means money game.
	// Normalise the Unlimited (-1) sentinel and any negative value to 0 so a
	// money match renders a header that re-parses instead of "-1 point match",
	// which the parser's `\d+` header regex rejects.
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
		// Checker move: "DD: notation". A forced no-move (dance) is stored as the
		// display marker "Cannot Move"; a .mat leaves the cell blank after the
		// dice, so emit dice only for it (and for an empty notation).
		cell := fmt.Sprintf("%d%d:", mv.Dice[0], mv.Dice[1])
		if mv.CheckerMove != "" && mv.CheckerMove != "Cannot Move" {
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

// SuggestMATFilename builds a filesystem-friendly default name for a match's
// .mat export: "{Player1}_{Player2}_{YYYY-MM-DD}_{Np|unlimited}.mat". Each
// player's words are CamelCased and concatenated (no spaces), accents kept;
// the date is the match date (falling back to the import date, then omitted);
// MatchLength <= 0 renders as "unlimited" (money game). The same name feeds the
// GUI save dialog and the CLI batch export so both agree.
func SuggestMATFilename(m *domain.Match) string {
	p1 := camelConcat(m.Player1Name)
	if p1 == "" {
		p1 = "Player1"
	}
	p2 := camelConcat(m.Player2Name)
	if p2 == "" {
		p2 = "Player2"
	}
	parts := []string{p1, p2}
	if d := matFilenameDate(m); d != "" {
		parts = append(parts, d)
	}
	if m.MatchLength > 0 {
		parts = append(parts, fmt.Sprintf("%dp", m.MatchLength))
	} else {
		parts = append(parts, "unlimited")
	}
	return strings.Join(parts, "_") + ".mat"
}

// matFilenameDate returns YYYY-MM-DD from the match date, falling back to the
// import date, or "" when neither is set.
func matFilenameDate(m *domain.Match) string {
	d := m.MatchDate
	if d.IsZero() {
		d = m.ImportDate
	}
	if d.IsZero() {
		return ""
	}
	return d.Format("2006-01-02")
}

// camelConcat splits a player name on whitespace, capitalises each word's first
// rune, drops filesystem-forbidden characters, and concatenates the words.
func camelConcat(s string) string {
	var b strings.Builder
	for _, word := range strings.Fields(s) {
		word = stripForbidden(word)
		if word == "" {
			continue
		}
		r := []rune(word)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String()
}

// stripForbidden removes path separators, characters illegal in filenames on
// common filesystems, and control runes. Accents and other letters are kept.
func stripForbidden(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
