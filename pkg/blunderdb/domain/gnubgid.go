package domain

import "fmt"

// GNUbg identifiers: the Position ID and the Match ID that GNU Backgammon
// shows, copies and loads with `set gnubgid` (`set board` keeps only the position).
//
// # Source
//
// The layout is positionid.c (oldPositionKey, PositionID) and matchid.c
// (MatchID) of gnubg-1.08.003. gnubgid_reference.json holds identifiers
// produced by gnubg itself (gnubg.gnubgid through its Python module) for
// boards, cubes and scores drawn at random; gnubgid_test.go holds the encoder
// to them.
//
// # The Position ID
//
// Ten bytes, written in base64 without padding (14 characters). Two boards of
// 25 slots each, the opponent's first, then the player on roll's: slots 0-23
// are the player's own points 1-24 and slot 24 his bar. Each slot writes as
// many 1-bits as it holds checkers, then one 0-bit; bits fill every byte from
// its least significant end. The board is therefore relative to the player on
// roll, whichever colour that is.
//
// # The Match ID
//
// Nine bytes, base64 (12 characters), 67 bits: log2 of the cube (4), cube
// owner (2; 3 centred), player on roll (1), Crawford (1), game state (3; 1
// playing), turn (1), doubled (1), resigned (2), the two dice (3 + 3, larger
// first), match length (15), the two scores (15 + 15) and the inverse of the
// Jacoby rule (1). Players are numbered, and the cube owner and the scores
// follow that numbering: here White is 0 and Black is 1. A position states
// only the distances to the match, so match length and scores are derived the
// way EncodeXGID derives them.
const gnubgBase64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// gnubgBoard returns the 25-slot board of one player, from his own side.
func gnubgBoard(pos *Position, color int) ([25]int, error) {
	var board [25]int
	total := 0
	for i, p := range pos.Board.Points {
		if p.Checkers <= 0 || p.Color != color {
			continue
		}
		var slot int
		switch {
		case color == Black && i == 25, color == White && i == 0:
			slot = 24
		case i == 0 || i == 25:
			return board, fmt.Errorf("checkers of the wrong colour on the bar")
		case color == Black:
			slot = i - 1
		default:
			slot = 24 - i
		}
		board[slot] += p.Checkers
		total += p.Checkers
	}
	if total > 15 {
		return board, fmt.Errorf("%d checkers on the board for one player, at most 15", total)
	}
	return board, nil
}

// EncodeGNUbgPositionID renders the board of pos as a GNUbg Position ID.
func EncodeGNUbgPositionID(pos *Position) (string, error) {
	onRoll, other := pos.PlayerOnRoll, White
	if onRoll == White {
		other = Black
	}
	if onRoll != Black && onRoll != White {
		return "", fmt.Errorf("no player on roll")
	}
	var key [10]byte
	bit := 0
	for _, color := range [2]int{other, onRoll} {
		board, err := gnubgBoard(pos, color)
		if err != nil {
			return "", err
		}
		for _, n := range board {
			for i := 0; i < n; i++ {
				key[bit/8] |= 1 << (bit % 8)
				bit++
			}
			bit++
		}
	}
	return gnubgBase64Encode(key[:]), nil
}

// EncodeGNUbgMatchID renders the cube, score and dice of pos as a GNUbg Match ID.
func EncodeGNUbgMatchID(pos *Position) string {
	player := func(color int) int {
		if color == Black {
			return 1
		}
		return 0
	}
	var key [9]byte
	put := func(at, n, v int) {
		for i := 0; i < n; i++ {
			if v&(1<<i) != 0 {
				key[(at+i)/8] |= 1 << ((at + i) % 8)
			}
		}
	}
	owner := 3
	if pos.Cube.Owner == Black || pos.Cube.Owner == White {
		owner = player(pos.Cube.Owner)
	}
	dice1, dice2 := 0, 0
	if pos.DecisionType != CubeAction {
		dice1, dice2 = max(pos.Dice[0], pos.Dice[1]), min(pos.Dice[0], pos.Dice[1])
	}
	matchLength, scoreBlack, scoreWhite, crawford := 0, 0, 0, false
	money := pos.Score[0] == Unlimited || pos.Score[1] == Unlimited
	if !money {
		matchLength, scoreBlack, scoreWhite, crawford = pointsScored(pos) // Score is [Black, White]
	}
	put(0, 4, pos.Cube.Value)
	put(4, 2, owner)
	put(6, 1, player(pos.PlayerOnRoll))
	if crawford {
		put(7, 1, 1)
	}
	put(8, 3, 1) // playing
	put(11, 1, player(pos.PlayerOnRoll))
	put(15, 3, dice1)
	put(18, 3, dice2)
	put(21, 15, matchLength)
	put(36, 15, scoreWhite)
	put(51, 15, scoreBlack)
	if !(money && pos.HasJacoby != 0) {
		put(66, 1, 1)
	}
	return gnubgBase64Encode(key[:])
}

// EncodeGNUbgID is "<Position ID>:<Match ID>", the form GNUbg copies.
func EncodeGNUbgID(pos *Position) (string, error) {
	p, err := EncodeGNUbgPositionID(pos)
	if err != nil {
		return "", err
	}
	return p + ":" + EncodeGNUbgMatchID(pos), nil
}

// gnubgBase64Encode is base64 without padding; the last 6-bit group is
// completed with zeros.
func gnubgBase64Encode(b []byte) string {
	out := make([]byte, 0, (len(b)*8+5)/6)
	acc, nbits := 0, 0
	for _, c := range b {
		acc = acc<<8 | int(c)
		nbits += 8
		for nbits >= 6 {
			nbits -= 6
			out = append(out, gnubgBase64[(acc>>nbits)&0x3f])
		}
	}
	if nbits > 0 {
		out = append(out, gnubgBase64[(acc<<(6-nbits))&0x3f])
	}
	return string(out)
}
