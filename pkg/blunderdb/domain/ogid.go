package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// OGID (OpenGammon Position ID) codec.
//
// # Source, and attribution
//
// OpenGammon publishes no specification. The reference is AnkiGammon's codec,
// ankigammon/utils/ogid.py (github.com/Deinonychus999/AnkiGammon, MIT
// License, Copyright (c) 2025 AnkiGammon; read at c5e71c3). Every field below
// is read as parse_ogid reads it and written as encode_ogid writes it. Its
// encoder produced the corpus's "cases" (testdata/ogid_corpus.json) and its
// decoder the "reference" section, which ogid_contract_test.go holds this
// codec to.
//
// # The format
//
//	OGID=<P1>:<P2>:<CUBE>[:<DICE>[:<TURN>[:<STATE>[:<S1>[:<S2>[:<ML>[:<MID>[:<N>]]]]]]]]
//
//   - P1, P2   one character per CHECKER, not per point: a point holding five
//     checkers appears five times. '0'–'9' are points 0–9, 'a'–'p'
//     points 10–25. P1 is White's, P2 is Black's. The board is
//     absolute: it does not turn with the player on roll.
//   - CUBE     three characters: owner (W/B/N), log2 of the value, action
//     (N normal, O offered, T taken, P passed).
//   - DICE     two digits, absent or empty for a cube decision.
//   - TURN     W or B; absent, the reference reads White.
//   - STATE    two characters (game state), carried by the format and read by
//     nothing here — see the note on what is dropped, below.
//   - S1, S2   points SCORED (not away), White then Black.
//   - ML       match length, optionally followed by a modifier letter (L, C
//     or G) and a number: "7", "7C", "9G15". "C" marks the Crawford
//     game; the reference gives the other two no meaning a position
//     could use. Empty is money play.
//   - MID, N   move id, checkers per side. Neither is part of a position.
//
// # Colour convention
//
// The mapping is the identity: OGID's White is blunderDB's White (O), its
// Black our Black (X), point 0 White's bar and 25 Black's. Not obvious, since
// the reference's docstring says "White = Player.X" (its X is the TOP player);
// the corpus settles it. The reference's encode_xgid mirrors the board when
// the top player is on roll, which DecodeXGID does not, so the corpus's
// OGID↔XGID pairs use blunderDB's absolute XGID convention.
//
// # The Crawford rule
//
// blunderDB keeps the Crawford rule inside the away score (CONTEXT.md,
// « Away score »). The game is Crawford exactly when the match length carries
// "C"; one point away without it is post-Crawford (away 0), as XGID field 7
// at 0. A 1-point match is always the Crawford game, as for XGID, so the same
// position pasted or imported hashes to one row.
//
// # What is dropped, deliberately
//
// The game state, move id, checker count and cube ACTION describe a game or a
// move, not a position (CONTEXT.md). Jacoby and beaver are not carried and not
// defaulted (ADR-0028); MaxCube stays 0.

// ErrInvalidOGID is returned for malformed OGID strings. Callers map it to a
// 4xx response, exactly as they do ErrInvalidXGID.
var ErrInvalidOGID = errors.New("invalid OGID")

// DecodeOGID parses an OGID string into a Position. The "OGID=" prefix is
// optional and surrounding whitespace is ignored.
func DecodeOGID(ogid string) (Position, error) {
	var pos Position
	s := strings.TrimSpace(ogid)
	if len(s) >= 5 && strings.EqualFold(s[:5], "OGID=") {
		s = s[5:]
	}
	if s == "" {
		return pos, fmt.Errorf("%w: empty", ErrInvalidOGID)
	}

	fields := strings.Split(s, ":")
	if len(fields) < 3 {
		return pos, fmt.Errorf("%w: expected at least 3 fields, got %d", ErrInvalidOGID, len(fields))
	}

	for i := range pos.Board.Points {
		pos.Board.Points[i] = Point{Checkers: 0, Color: None}
	}
	// P1 is OGID's White, which is blunderDB's White (O); P2 its Black, our
	// Black (X). See the header: checked against the corpus, not assumed.
	if err := placeOGIDCheckers(&pos, fields[0], White); err != nil {
		return Position{}, err
	}
	if err := placeOGIDCheckers(&pos, fields[1], Black); err != nil {
		return Position{}, err
	}
	var onBoard [2]int
	for _, p := range pos.Board.Points {
		if p.Color == Black || p.Color == White {
			onBoard[p.Color] += p.Checkers
		}
	}
	if onBoard[Black] > 15 || onBoard[White] > 15 {
		return Position{}, fmt.Errorf("%w: a player has more than 15 checkers", ErrInvalidOGID)
	}
	pos.Board.Bearoff[Black] = 15 - onBoard[Black]
	pos.Board.Bearoff[White] = 15 - onBoard[White]

	// --- Cube: owner, log2 value, action (the action is not a position) ---
	cube := fields[2]
	if len(cube) != 3 {
		return Position{}, fmt.Errorf("%w: cube field must be 3 characters, got %q", ErrInvalidOGID, cube)
	}
	switch cube[0] {
	case 'W', 'w':
		pos.Cube.Owner = White
	case 'B', 'b':
		pos.Cube.Owner = Black
	default:
		pos.Cube.Owner = None
	}
	// Stored as the EXPONENT, like XGID's own field and like blunderDB's
	// storage (engine/zobrist.go): expanding it here would break the hash.
	if exp, err := strconv.Atoi(string(cube[1])); err == nil && exp >= 0 && exp <= 10 {
		pos.Cube.Value = exp
	}

	// --- Dice (field 3) ---
	if len(fields) > 3 && len(fields[3]) == 2 &&
		fields[3][0] >= '0' && fields[3][0] <= '9' &&
		fields[3][1] >= '0' && fields[3][1] <= '9' {
		pos.Dice = [2]int{int(fields[3][0] - '0'), int(fields[3][1] - '0')}
	}
	if pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6 {
		pos.DecisionType = CheckerAction
	} else {
		pos.DecisionType = CubeAction
	}

	// --- Turn (field 4). Black is our Black; anything else is White. ---
	pos.PlayerOnRoll = White
	if len(fields) > 4 && (fields[4] == "B" || fields[4] == "b") {
		pos.PlayerOnRoll = Black
	}

	// --- Score (fields 6, 7), match length and Crawford (field 8) ---
	// blunderDB stores the AWAY score; OGID carries points scored, as XGID
	// does. Money play is [-1, -1] (the engine's own sentinel). Field 6 is
	// White's points, field 7 Black's; blunderDB indexes the score by colour,
	// and Black is index 0.
	pos.Score = [2]int{Unlimited, Unlimited}
	blackPts, okB := ogidInt(fields, 7)
	whitePts, okW := ogidInt(fields, 6)
	if length, modifier, ok := ogidMatchLength(fields); ok && length > 0 && okB && okW {
		crawford := modifier == 'C' || length == 1
		pos.Score = AwayScoresWithCrawford(length, blackPts, whitePts, crawford)
	}

	return pos, nil
}

// placeOGIDCheckers reads one checker-per-character run into the board: one
// character per CHECKER, unlike XGID's per-point board string.
func placeOGIDCheckers(pos *Position, run string, color int) error {
	for i := 0; i < len(run); i++ {
		point, err := ogidPointIndex(run[i])
		if err != nil {
			return err
		}
		p := &pos.Board.Points[point]
		if p.Color != None && p.Color != color {
			return fmt.Errorf("%w: point %d holds both colours", ErrInvalidOGID, point)
		}
		p.Color = color
		p.Checkers++
		if p.Checkers > 15 {
			return fmt.Errorf("%w: %d checkers on point %d exceeds 15", ErrInvalidOGID, p.Checkers, point)
		}
	}
	return nil
}

// ogidPointIndex maps an OGID character to blunderDB's board index: '0'–'9'
// are points 0–9 and 'a'–'p' points 10–25, bars included (0 is White's, 25 is
// Black's — the same ends blunderDB uses).
func ogidPointIndex(c byte) (int, error) {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), nil
	case c >= 'a' && c <= 'p':
		return int(c-'a') + 10, nil
	default:
		return 0, fmt.Errorf("%w: bad position character %q", ErrInvalidOGID, string(c))
	}
}

// ogidMatchLength reads field 8, which may carry a modifier letter and a game
// count after the number ("7", "7C", "9G15") — the reference's
// (\d+)([LCG]?)(\d*). The number and the modifier are a position's business
// (the "C" marks the Crawford game); the game count is not. modifier is 0
// when there is none.
func ogidMatchLength(fields []string) (length int, modifier byte, ok bool) {
	if len(fields) <= 8 || fields[8] == "" {
		return 0, 0, false
	}
	f := fields[8]
	digits := 0
	for digits < len(f) && f[digits] >= '0' && f[digits] <= '9' {
		digits++
	}
	if digits == 0 {
		return 0, 0, false
	}
	n, err := strconv.Atoi(f[:digits])
	if err != nil {
		return 0, 0, false
	}
	if digits < len(f) {
		modifier = f[digits]
	}
	return n, modifier, true
}

// ogidInt reads an optional numeric field.
func ogidInt(fields []string, i int) (int, bool) {
	if i >= len(fields) || fields[i] == "" {
		return 0, false
	}
	n, err := strconv.Atoi(fields[i])
	if err != nil {
		return 0, false
	}
	return n, true
}

// LooksLikeOGID reports whether text carries an OGID, so a paste or a typed
// identifier can be routed without asking the user which format they have.
//
// It is deliberately narrow: the explicit "OGID=" prefix, or three or more
// colon-separated fields whose third is a three-character cube. An XGID's
// third field is a lone number, so the two can never be confused.
func LooksLikeOGID(text string) bool {
	s := strings.TrimSpace(text)
	if len(s) >= 5 && strings.EqualFold(s[:5], "OGID=") {
		return true
	}
	if strings.Contains(s, "XGID=") {
		return false
	}
	fields := strings.Split(s, ":")
	if len(fields) < 3 || len(fields[2]) != 3 {
		return false
	}
	switch fields[2][0] {
	case 'W', 'B', 'N', 'w', 'b', 'n':
	default:
		return false
	}
	return fields[2][1] >= '0' && fields[2][1] <= '9'
}

// EncodeOGID renders a Position as an OGID: DecodeOGID's inverse up to what a
// Position does not keep, written the way the reference's encode_ogid writes
// it — each side's characters sorted, the cube action "N", no game state, no
// move id, and the checker count left out because it is 15.
//
// Like EncodeXGID, it has only the away score to go on, never the match
// length nor the points scored, so a match score goes out as the smallest
// match consistent with its distances (away [2, 4] is a 4-point match at
// 2-0), which decodes back to the same away scores. The Crawford game is the
// "C" modifier. Two post-Crawford sentinels go out as a 2-point match at 1-1,
// since a 1-point match is always Crawford (EncodeXGID's rule). Money play
// writes 0-0 and no length, as the reference does.
//
// The cube's exponent is one character, so a cube past 512 has no OGID
// spelling; no source blunderDB reads produces one.
func EncodeOGID(pos *Position) string {
	var white, black []byte
	for i := 0; i < len(pos.Board.Points); i++ {
		p := pos.Board.Points[i]
		if p.Checkers <= 0 {
			continue
		}
		c := ogidPointChar(i)
		for n := 0; n < p.Checkers; n++ {
			switch p.Color {
			case White:
				white = append(white, c)
			case Black:
				black = append(black, c)
			}
		}
	}

	owner := byte('N')
	switch pos.Cube.Owner {
	case White:
		owner = 'W'
	case Black:
		owner = 'B'
	}
	cube := fmt.Sprintf("%c%dN", owner, pos.Cube.Value)

	dice := ""
	if pos.DecisionType != CubeAction &&
		pos.Dice[0] >= 1 && pos.Dice[0] <= 6 && pos.Dice[1] >= 1 && pos.Dice[1] <= 6 {
		dice = fmt.Sprintf("%d%d", pos.Dice[0], pos.Dice[1])
	}

	turn := "W"
	if pos.PlayerOnRoll == Black {
		turn = "B"
	}

	whitePts, blackPts, length := 0, 0, ""
	if pos.Score[0] != Unlimited && pos.Score[1] != Unlimited {
		awayBlack, awayWhite := PointsAway(pos.Score[0]), PointsAway(pos.Score[1])
		n := max(awayBlack, awayWhite)
		crawford := pos.Score[0] == Crawford || pos.Score[1] == Crawford
		if n == 1 && !crawford {
			n = 2 // a 1-point match would say Crawford
		}
		whitePts, blackPts = n-awayWhite, n-awayBlack
		length = strconv.Itoa(n)
		if crawford {
			length += "C"
		}
	}

	return fmt.Sprintf("%s:%s:%s:%s:%s::%d:%d:%s:",
		white, black, cube, dice, turn, whitePts, blackPts, length)
}

// ogidPointChar is ogidPointIndex's inverse.
func ogidPointChar(point int) byte {
	if point < 10 {
		return byte('0' + point)
	}
	return byte('a' + point - 10)
}

// DecodePositionID reads a position identifier of either kind: an OGID when
// LooksLikeOGID says so, an XGID otherwise. It is for the places that take an
// identifier ALONE as an argument (the CLI's `epc` and `cubematrix`); pasted
// text, which may also be an analysis, goes through parser.ParsePosition.
func DecodePositionID(id string) (Position, error) {
	if LooksLikeOGID(id) {
		return DecodeOGID(id)
	}
	return DecodeXGID(id)
}
