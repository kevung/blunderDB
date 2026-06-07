package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// XGID (eXtreme Gammon Position ID) decoding. An XGID is a compact, portable
// string describing a full position + match state:
//
//	XGID=<26-char board>:<cube>:<cubePos>:<turn>:<dice>:<scoreP1>:<scoreP2>:<jacoby/crawford>:<matchLen>:<maxCube>
//
// Board string (26 chars, one per location):
//   - index 0      → the lowercase player's (O) bar
//   - indices 1-24 → points 1..24
//   - index 25     → the uppercase player's (X) bar
//   - '-'          → empty; 'A'..'O' → 1..15 of player X; 'a'..'o' → 1..15 of player O
//
// Mapping to blunderDB's Position: X = Black (bar at index 25 = BlackBar), O =
// White (bar at index 0 = WhiteBar); point numbering is identical. Score is the
// AWAY score (matchLen − absolute), matching the engine's import convention
// (money games → −1). The cube field is log2 of the cube value.
//
// This is the inverse of the encoder used by XG/GNU exports; it is generic and
// useful to the Desktop app too (paste an XGID → position).

// ErrInvalidXGID is returned for malformed XGID strings. Callers (the server)
// map it to a 4xx response.
var ErrInvalidXGID = errors.New("invalid XGID")

// DecodeXGID parses an XGID string into a Position. The "XGID=" prefix is
// optional and surrounding whitespace is ignored.
func DecodeXGID(xgid string) (Position, error) {
	var pos Position
	s := strings.TrimSpace(xgid)
	s = strings.TrimPrefix(s, "XGID=")
	if s == "" {
		return pos, fmt.Errorf("%w: empty", ErrInvalidXGID)
	}

	fields := strings.Split(s, ":")
	board := fields[0]
	if len(board) != 26 {
		return pos, fmt.Errorf("%w: board must be 26 characters, got %d", ErrInvalidXGID, len(board))
	}

	// Empty board: every location starts at {0, None}.
	for i := range pos.Board.Points {
		pos.Board.Points[i] = Point{Checkers: 0, Color: None}
	}

	var onBoard [2]int // checkers on board (incl. bars) per colour
	for i := 0; i < 26; i++ {
		c := board[i]
		if c == '-' {
			continue
		}
		var count, color int
		switch {
		case c >= 'A' && c <= 'Z':
			count, color = int(c-'A')+1, Black // X
		case c >= 'a' && c <= 'z':
			count, color = int(c-'a')+1, White // O
		default:
			return pos, fmt.Errorf("%w: bad board character %q at %d", ErrInvalidXGID, string(c), i)
		}
		if count > 15 {
			return pos, fmt.Errorf("%w: %d checkers at %d exceeds 15", ErrInvalidXGID, count, i)
		}
		pos.Board.Points[i] = Point{Checkers: count, Color: color}
		onBoard[color] += count
	}
	if onBoard[Black] > 15 || onBoard[White] > 15 {
		return pos, fmt.Errorf("%w: a player has more than 15 checkers", ErrInvalidXGID)
	}
	// Off the board = 15 − on the board. Bearoff is indexed by colour.
	pos.Board.Bearoff[Black] = 15 - onBoard[Black]
	pos.Board.Bearoff[White] = 15 - onBoard[White]

	// --- Cube (field 1: log2 value; field 2: owner) ---
	if v, ok := xgidInt(fields, 1); ok && v >= 0 && v <= 30 {
		pos.Cube.Value = 1 << uint(v)
	} else {
		pos.Cube.Value = 1 // centred / unset → shows 1
	}
	switch owner, _ := xgidInt(fields, 2); owner {
	case 1:
		pos.Cube.Owner = Black // X owns
	case -1:
		pos.Cube.Owner = White // O owns
	default:
		pos.Cube.Owner = None // centred
	}

	// --- Turn (field 3): 1 → X (Black), -1 → O (White), 0 → default Black ---
	if turn, _ := xgidInt(fields, 3); turn == -1 {
		pos.PlayerOnRoll = White
	} else {
		pos.PlayerOnRoll = Black
	}

	// --- Dice (field 4): two digits, e.g. "64"; "00"/absent → not rolled ---
	if len(fields) > 4 && len(fields[4]) == 2 &&
		fields[4][0] >= '0' && fields[4][0] <= '9' &&
		fields[4][1] >= '0' && fields[4][1] <= '9' {
		pos.Dice = [2]int{int(fields[4][0] - '0'), int(fields[4][1] - '0')}
	}
	if pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6 {
		pos.DecisionType = CheckerAction
	} else {
		pos.DecisionType = CubeAction
	}

	// --- Score (fields 5,6) + match length (field 8) → away score ---
	score1, ok1 := xgidInt(fields, 5)
	score2, ok2 := xgidInt(fields, 6)
	matchLen, okM := xgidInt(fields, 8)
	if ok1 && ok2 && okM && matchLen > 0 {
		pos.Score = [2]int{matchLen - score1, matchLen - score2} // [X=Black, O=White], away
	} else {
		pos.Score = [2]int{-1, -1} // money game (engine convention)
	}

	return pos, nil
}

// EncodeXGIDBoard renders the 26-character board portion of an XGID from a
// Position. It is the inverse of the board decode and is used to round-trip
// (validate) the codec; it does not emit the full XGID (match metadata that the
// Position does not retain, e.g. absolute score / match length, cannot be
// reconstructed).
func EncodeXGIDBoard(pos *Position) string {
	b := make([]byte, 26)
	for i := 0; i < 26; i++ {
		p := pos.Board.Points[i]
		switch {
		case p.Checkers <= 0 || p.Color == None:
			b[i] = '-'
		case p.Color == Black:
			b[i] = byte('A' + p.Checkers - 1)
		default: // White
			b[i] = byte('a' + p.Checkers - 1)
		}
	}
	return string(b)
}

// xgidInt parses fields[idx] as an int; ok is false when absent or non-numeric.
func xgidInt(fields []string, idx int) (int, bool) {
	if idx >= len(fields) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(fields[idx]))
	if err != nil {
		return 0, false
	}
	return n, true
}
