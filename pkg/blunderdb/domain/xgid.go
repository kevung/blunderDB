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
// (money games → −1), with the Crawford rule written into it from field 7 (see
// DecodeXGID). The cube field is log2 of the cube value.
//
// Field 7 means two things (eXtreme Gammon 2 Help, « XGID », part 8 "Crawford,
// Jacoby"): at a match score, "1 means the current game is Crawford, 0 means
// the current game is not played with the Crawford rule"; in money play, the
// Jacoby + 2×Beaver bitmask.
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
	// The XGID cube field is already the exponent (0→1, 1→2, 2→4, …), which is
	// exactly blunderDB's storage convention (Cube.Value is the exponent; see
	// engine/zobrist.go and ingest/xgmap.go). Store it verbatim — do NOT expand
	// to the actual value, or the Zobrist hash and cube rendering break.
	if v, ok := xgidInt(fields, 1); ok && v >= 0 && v <= 10 {
		pos.Cube.Value = v
	} else {
		pos.Cube.Value = 0 // centred / unset → exponent 0 (shows 1)
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

	// --- Score (fields 5,6) + match length (field 8) + Crawford (field 7) → away score ---
	// The away score carries the Crawford rule INSIDE the number (CONTEXT.md,
	// « Away score »): a player one point away is 1 in the Crawford game and 0
	// after it. matchLen − score says 1 for both, so field 7 decides, exactly as
	// the match importers decide from their own per-game flag (#338). This is
	// the one reading of it: the shared text parser, /v1/positions.fromXGID and
	// the BGBlitz position import all go through it (#360). An XGID whose field
	// 7 is empty states nothing, and the ambiguous 1 stays.
	score1, ok1 := xgidInt(fields, 5)
	score2, ok2 := xgidInt(fields, 6)
	matchLen, okM := xgidInt(fields, 8)
	if ok1 && ok2 && okM && matchLen > 0 {
		crawford, stated := xgidCrawfordGame(fields)
		pos.Score = AwayScoresWithCrawford(matchLen, score1, score2, crawford || !stated) // [X=Black, O=White], away
	} else {
		pos.Score = [2]int{-1, -1} // money game (engine convention)
	}

	// --- Jacoby / Beaver (field 7) ---
	// In a money game (matchLen 0/absent) field 7 is a bitmask: bit 0 = Jacoby,
	// bit 1 = Beaver (so 3 = both). In match play it is the Crawford flag, read
	// with the score above, and carries no jacoby/beaver information.
	if !okM || matchLen == 0 {
		if flag, ok := xgidInt(fields, 7); ok {
			if flag&1 != 0 {
				pos.HasJacoby = 1
			}
			if flag&2 != 0 {
				pos.HasBeaver = 1
			}
		}
	}

	// --- Cube ceiling (field 9) ---
	// XG writes the log2 exponent of the highest cube the session allows; 10
	// (1024) is its own "no limit". Anything outside 1..9 is stored as 0, the
	// value that means "no ceiling stated" — see Position.MaxCube.
	if maxCube, ok := xgidInt(fields, 9); ok && maxCube >= 1 && maxCube <= 9 {
		pos.MaxCube = maxCube
	}

	return pos, nil
}

// XGIDCrawfordGame reads what an XGID states about the Crawford rule. stated
// is false when it states nothing: money play, no match length, or a field 7
// that is neither 0 nor 1. Otherwise crawford is field 7's own word — the
// current game is the Crawford game — which is what DecodeXGID writes into the
// away score. The "XGID=" prefix is optional.
func XGIDCrawfordGame(xgid string) (crawford, stated bool) {
	s := strings.TrimPrefix(strings.TrimSpace(xgid), "XGID=")
	return xgidCrawfordGame(strings.Split(s, ":"))
}

func xgidCrawfordGame(fields []string) (crawford, stated bool) {
	if matchLen, ok := xgidInt(fields, 8); !ok || matchLen <= 0 {
		return false, false
	}
	switch flag, ok := xgidInt(fields, 7); {
	case !ok:
		return false, false
	case flag == 1:
		return true, true
	case flag == 0:
		return false, true
	default:
		return false, false
	}
}

// EncodeXGID renders a Position as a full XGID: DecodeXGID's inverse up to what
// a Position does not keep, and the Go twin of the GUI's generateXGID
// (frontend/src/services/xgid.js) — testdata/xgid_corpus.json holds both to the
// same `xgidCanonical`.
//
// A Position stores the away score only, never the match length nor the points
// scored, so a match score is re-encoded as the smallest match consistent with
// its distances (away [2, 4] becomes a 4-point match at 2-0), which decodes back
// to the same away scores. The distances are read through PointsAway, so the
// post-Crawford 0 is one point away and not a match already won; field 7 is
// read from the sentinel itself: 1 is the Crawford game, 0 is not (#338, #360).
// In money play field 7 is the Jacoby/Beaver bitmask instead. A cube decision
// writes dice 00, and the cube ceiling goes back out as the source stated it.
func EncodeXGID(pos *Position) string {
	cubeOwner := 0
	switch pos.Cube.Owner {
	case Black:
		cubeOwner = 1
	case White:
		cubeOwner = -1
	}
	turn := -1
	if pos.PlayerOnRoll == Black {
		turn = 1
	}
	dice := "00"
	if pos.DecisionType != CubeAction {
		dice = fmt.Sprintf("%d%d", pos.Dice[0], pos.Dice[1])
	}
	matchLength, score1, score2, field7 := 0, 0, 0, 0
	if pos.Score[0] == Unlimited || pos.Score[1] == Unlimited {
		if pos.HasJacoby != 0 {
			field7 |= 1
		}
		if pos.HasBeaver != 0 {
			field7 |= 2
		}
	} else {
		away1, away2 := PointsAway(pos.Score[0]), PointsAway(pos.Score[1])
		matchLength = max(away1, away2)
		score1, score2 = matchLength-away1, matchLength-away2
		if pos.Score[0] == Crawford || pos.Score[1] == Crawford {
			field7 = 1
		}
	}
	return fmt.Sprintf("%s:%d:%d:%d:%s:%d:%d:%d:%d:%d", EncodeXGIDBoard(pos),
		pos.Cube.Value, cubeOwner, turn, dice, score1, score2, field7, matchLength, pos.MaxCube)
}

// EncodeXGIDBoard renders the 26-character board portion of an XGID from a
// Position. It is the inverse of the board decode; EncodeXGID writes the rest.
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
