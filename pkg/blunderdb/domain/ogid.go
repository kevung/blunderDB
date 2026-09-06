package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// OGID (OpenGammon Position ID) decoding — issue #260, fiche I.4.
//
// # Why this reader exists and the other one does not
//
// The fiche originally asked for an "OGXM" parser. The research report P9
// found that format has no publicly verifiable existence: no specification, no
// repository, no trace. It was not written. OGID is the opposite case: it is
// attested, implemented, and its implementation is readable — AnkiGammon
// (ankigammon/utils/ogid.py, MIT), whose own docstring is the specification
// this reader was written against, and whose encoder produced every string in
// testdata/ogid_corpus.json.
//
// That is what "collect real samples before writing anything" meant, and it is
// why this reader can exist while the bgammon.org importer still cannot.
//
// # The format
//
//	OGID=<P1>:<P2>:<CUBE>[:<DICE>[:<TURN>[:<STATE>[:<S1>[:<S2>[:<ML>[:<MID>[:<N>]]]]]]]]
//
//   - P1, P2   one character per CHECKER, not per point: a point holding five
//     checkers appears five times. '0'–'9' are points 0–9, 'a'–'p'
//     points 10–25. P1 is White's, P2 is Black's.
//   - CUBE     three characters: owner (W/B/N), log2 of the value, action.
//   - DICE     two digits, absent or empty for a cube decision.
//   - TURN     W or B.
//   - STATE    two characters (game state), carried by the format and read by
//     nothing here — see the note on what is dropped, below.
//   - S1, S2   points SCORED (not away), White then Black.
//   - ML       match length, optionally followed by a modifier and a number.
//   - MID, N   move id, checkers per side. Neither is part of a position.
//
// # The two conventions line up, and that was checked rather than assumed
//
// OGID's White is blunderDB's White (O) and its Black is our Black (X); its
// point 0 is White's bar and its 25 is Black's, exactly where blunderDB puts
// them. The mapping is therefore the identity — but it is stated here because
// it is not obvious, and because the first reading of the reference
// implementation's docstring ("White = Player.X") suggested the opposite. What
// settled it was testdata/ogid_corpus.json: twenty positions where DecodeOGID
// and DecodeXGID must agree, which is a fact rather than an interpretation.
//
// # What is dropped, deliberately
//
// The game state, the move id and the checker count describe a GAME, not a
// position, and blunderDB's Position is a position (CONTEXT.md). The cube
// ACTION is dropped for the same reason: whether the cube has been offered is
// a fact of a move, and the position's own decision type already comes from
// whether dice are set. Nothing here invents Jacoby or beaver either — OGID
// does not carry them, and defaulting them would put two different money
// positions under one Zobrist hash (ADR-0028 arrived at that from the other
// direction).

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

	// --- Score (fields 6, 7) and match length (field 8) ---
	// blunderDB stores the AWAY score; OGID carries points scored, as XGID
	// does. Money play is [-1, -1] (the engine's own sentinel).
	pos.Score = [2]int{-1, -1}
	// Field 6 is White's points, field 7 Black's; blunderDB indexes the score
	// by colour, and Black is index 0.
	s1, ok1 := ogidInt(fields, 7)
	s2, ok2 := ogidInt(fields, 6)
	if length, ok := ogidMatchLength(fields); ok && length > 0 && ok1 && ok2 {
		pos.Score = [2]int{length - s1, length - s2}
	}

	return pos, nil
}

// placeOGIDCheckers reads one checker-per-character run into the board. OGID
// writes one character per CHECKER, so a point holding five appears five
// times — a shape worth naming, because it is the one difference from XGID's
// board string that a reader is likely to mis-transcribe.
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

// ogidMatchLength reads field 8, which may carry a modifier and a game count
// after the number ("7", "5C", "9G15"). Only the number is a position's
// business.
func ogidMatchLength(fields []string) (int, bool) {
	if len(fields) <= 8 || fields[8] == "" {
		return 0, false
	}
	digits := 0
	for digits < len(fields[8]) && fields[8][digits] >= '0' && fields[8][digits] <= '9' {
		digits++
	}
	if digits == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(fields[8][:digits])
	if err != nil {
		return 0, false
	}
	return n, true
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
