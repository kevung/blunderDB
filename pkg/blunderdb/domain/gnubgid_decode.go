package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidGNUbgID is wrapped by every refusal of DecodeGNUbgID.
var ErrInvalidGNUbgID = errors.New("invalid GNUbg ID")

// DecodeGNUbgID reads "<Position ID>:<Match ID>", or a Position ID alone, the
// inverse of EncodeGNUbgID (layout in gnubgid.go).
//
// A Position ID alone says nothing of who is on roll, the cube, the score or
// the dice: it decodes as DecodeXGID reads a bare board, Black on roll with no
// dice (a cube decision), the cube centred at money play.
//
// A Match ID states points scored; the position keeps the distances, so
// Encode(Decode(id)) gives back the same id except when both players have
// points, where the length is derived as the larger distance (pointsScored).
func DecodeGNUbgID(id string) (Position, error) {
	s := strings.TrimSpace(id)
	posID, matchID, hasMatch := strings.Cut(s, ":")
	var pos Position
	pos.PlayerOnRoll = Black
	pos.Cube.Owner = None
	pos.Score = [2]int{Unlimited, Unlimited}
	pos.DecisionType = CubeAction
	if hasMatch {
		if err := decodeGNUbgMatchID(&pos, matchID); err != nil {
			return Position{}, err
		}
	}
	if err := decodeGNUbgPositionID(&pos, posID); err != nil {
		return Position{}, err
	}
	return pos, nil
}

// decodeGNUbgPositionID places the checkers of a Position ID, read relative
// to pos.PlayerOnRoll.
func decodeGNUbgPositionID(pos *Position, posID string) error {
	if len(posID) != 14 {
		return fmt.Errorf("%w: a Position ID has 14 characters, got %d", ErrInvalidGNUbgID, len(posID))
	}
	key, err := gnubgBase64Decode(posID, 10)
	if err != nil {
		return err
	}
	onRoll, other := pos.PlayerOnRoll, White
	if onRoll == White {
		other = Black
	}
	for i := range pos.Board.Points {
		pos.Board.Points[i] = Point{Checkers: 0, Color: None}
	}
	bit := 0
	for _, color := range [2]int{other, onRoll} {
		total := 0
		for slot := 0; slot < 25; slot++ {
			n := 0
			for {
				if bit >= 80 {
					return fmt.Errorf("%w: the Position ID ends inside a board", ErrInvalidGNUbgID)
				}
				set := key[bit/8]&(1<<(bit%8)) != 0
				bit++
				if !set {
					break
				}
				n++
			}
			if n == 0 {
				continue
			}
			total += n
			if total > 15 {
				return fmt.Errorf("%w: more than 15 checkers for one player", ErrInvalidGNUbgID)
			}
			var i int
			switch {
			case slot == 24 && color == Black:
				i = 25
			case slot == 24:
				i = 0
			case color == Black:
				i = slot + 1
			default:
				i = 24 - slot
			}
			if pos.Board.Points[i].Color != None {
				return fmt.Errorf("%w: point %d holds both colours", ErrInvalidGNUbgID, i)
			}
			pos.Board.Points[i] = Point{Checkers: n, Color: color}
		}
		pos.Board.Bearoff[color] = 15 - total
	}
	return nil
}

// decodeGNUbgMatchID reads the cube, turn, dice and score of a Match ID.
func decodeGNUbgMatchID(pos *Position, matchID string) error {
	if len(matchID) != 12 {
		return fmt.Errorf("%w: a Match ID has 12 characters, got %d", ErrInvalidGNUbgID, len(matchID))
	}
	key, err := gnubgBase64Decode(matchID, 9)
	if err != nil {
		return err
	}
	get := func(at, n int) int {
		v := 0
		for i := 0; i < n; i++ {
			if key[(at+i)/8]&(1<<((at+i)%8)) != 0 {
				v |= 1 << i
			}
		}
		return v
	}
	color := func(player int) int {
		if player == 1 {
			return Black
		}
		return White
	}
	cube := get(0, 4)
	if cube > 12 {
		return fmt.Errorf("%w: cube 2^%d", ErrInvalidGNUbgID, cube)
	}
	pos.Cube.Value = cube
	switch owner := get(4, 2); owner {
	case 0, 1:
		pos.Cube.Owner = color(owner)
	case 3:
		pos.Cube.Owner = None
	default:
		return fmt.Errorf("%w: cube owner %d", ErrInvalidGNUbgID, owner)
	}
	// The board is written from the side of the player whose turn it is
	// (bit 6); bit 11 differs only while a double awaits its answer.
	pos.PlayerOnRoll = color(get(6, 1))
	crawford := get(7, 1) == 1

	d1, d2 := get(15, 3), get(18, 3)
	switch {
	case d1 == 0 && d2 == 0:
		pos.DecisionType = CubeAction
	case d1 >= 1 && d1 <= 6 && d2 >= 1 && d2 <= 6:
		pos.Dice = [2]int{max(d1, d2), min(d1, d2)}
		pos.DecisionType = CheckerAction
	default:
		return fmt.Errorf("%w: dice %d-%d", ErrInvalidGNUbgID, d1, d2)
	}

	length, whitePts, blackPts := get(21, 15), get(36, 15), get(51, 15)
	jacobyOff := get(66, 1) == 1
	if length == 0 {
		if !jacobyOff {
			pos.HasJacoby = 1
		}
		return nil
	}
	if whitePts >= length || blackPts >= length {
		return fmt.Errorf("%w: score %d-%d does not fit a %d-point match", ErrInvalidGNUbgID, whitePts, blackPts, length)
	}
	pos.Score = AwayScoresWithCrawford(length, blackPts, whitePts, crawford || length == 1)
	return nil
}

// gnubgBase64Decode is gnubgBase64Encode's inverse; it keeps the first n bytes.
func gnubgBase64Decode(s string, n int) ([]byte, error) {
	out := make([]byte, 0, n+1)
	acc, nbits := 0, 0
	for i := 0; i < len(s); i++ {
		v := strings.IndexByte(gnubgBase64, s[i])
		if v < 0 {
			return nil, fmt.Errorf("%w: %q is not a base64 character", ErrInvalidGNUbgID, s[i])
		}
		acc = acc<<6 | v
		nbits += 6
		if nbits >= 8 {
			nbits -= 8
			out = append(out, byte(acc>>nbits))
			acc &= 1<<nbits - 1
		}
	}
	if len(out) < n {
		return nil, fmt.Errorf("%w: too short", ErrInvalidGNUbgID)
	}
	return out[:n], nil
}

var (
	gnubgPair      = regexp.MustCompile(`(?:^|[^A-Za-z0-9+/=:])([A-Za-z0-9+/]{14}):([A-Za-z0-9+/]{12})(?:$|[^A-Za-z0-9+/:])`)
	gnubgPosLabel  = regexp.MustCompile(`(?i)position\s*id\s*:?\s*([A-Za-z0-9+/]{14})(?:$|[^A-Za-z0-9+/])`)
	gnubgMatchLbl  = regexp.MustCompile(`(?i)match\s*id\s*:?\s*([A-Za-z0-9+/]{12})(?:$|[^A-Za-z0-9+/])`)
	gnubgBarePosID = regexp.MustCompile(`^[A-Za-z0-9+/]{14}$`)
)

// FindGNUbgID picks a GNUbg identifier out of pasted text and returns it in
// DecodeGNUbgID's form: "PositionID:MatchID" anywhere in the text, the
// "Position ID: … Match ID: …" lines GNUbg copies, or a text that is a
// Position ID and nothing else (a lone 14-character word is too common
// inside prose to be taken for one).
func FindGNUbgID(text string) (string, bool) {
	if m := gnubgPair.FindStringSubmatch(text); m != nil {
		return m[1] + ":" + m[2], true
	}
	if m := gnubgPosLabel.FindStringSubmatch(text); m != nil {
		if mm := gnubgMatchLbl.FindStringSubmatch(text); mm != nil {
			return m[1] + ":" + mm[1], true
		}
		return m[1], true
	}
	if s := strings.TrimSpace(text); gnubgBarePosID.MatchString(s) {
		return s, true
	}
	return "", false
}
