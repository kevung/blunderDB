package arkqr

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The reimport code of a position list: the positions alone, a tranche of
// Tranche of them to an archive of their own (KindPositions), so a page that
// carries one tranche's codes reads back without the others. A position is
// what its ArkID says — score, dice, cube, checkers — in about twelve bytes;
// analyses, comments and provenance are left out, as the ArkID leaves them.

// Tranche is the number of positions one tranche codes, about a printed face
// of a list with its analyses. A position takes 109 bits at most, so a full
// tranche codes in 1 976 bytes, plus a header of 5 (the version, a first
// number below 16 384 and the count, each uvarint on two bytes): 1 981 bytes,
// never more than two data codes of maxPart bytes, and their parity.
const Tranche = 145

// PositionsVersion is the first byte of a coded tranche.
const PositionsVersion = 1

// ErrPositions is data that is not a readable coded tranche.
var ErrPositions = errors.New("arkqr: not a readable tranche of positions")

// PositionTranche is a decoded tranche: the number the list gave its first
// position, then the positions, normalised as DecodeArkID returns them.
type PositionTranche struct {
	First     int
	Positions []domain.Position
}

// EncodePositions codes a tranche of a list whose first position is numbered
// first. A position its ArkID cannot spell is refused: the list prints none.
func EncodePositions(first int, ps []*domain.Position) ([]byte, error) {
	out := []byte{PositionsVersion}
	out = binary.AppendUvarint(out, uint64(first))
	out = binary.AppendUvarint(out, uint64(len(ps)))
	var w bitWriter
	for i, p := range ps {
		if err := encodePosition(&w, p); err != nil {
			return nil, fmt.Errorf("position %d: %w", first+i, err)
		}
	}
	return append(out, w.bytes()...), nil
}

// DecodePositions reads a coded tranche back.
func DecodePositions(data []byte) (PositionTranche, error) {
	var t PositionTranche
	if len(data) < 1 || data[0] != PositionsVersion {
		return t, fmt.Errorf("%w: version", ErrPositions)
	}
	rest := data[1:]
	first, n := binary.Uvarint(rest)
	if n <= 0 {
		return t, fmt.Errorf("%w: header", ErrPositions)
	}
	rest = rest[n:]
	count, n := binary.Uvarint(rest)
	// A position takes at least 54 bits: count bounds what the data holds.
	if n <= 0 || count > uint64(len(rest))*8/54+1 {
		return t, fmt.Errorf("%w: header", ErrPositions)
	}
	r := bitReader{b: rest[n:]}
	t.First = int(first)
	for i := range int(count) {
		s, err := decodeArk(&r)
		if err != nil {
			return t, fmt.Errorf("%w: position %d: %w", ErrPositions, t.First+i, err)
		}
		p, err := domain.DecodeArkID(s)
		if err != nil {
			return t, fmt.Errorf("%w: position %d: %w", ErrPositions, t.First+i, err)
		}
		t.Positions = append(t.Positions, p)
	}
	return t, nil
}

// Cube tokens of an ArkID, on two bits.
const (
	cubeCentred = iota
	cubeOnRoll  // "x"
	cubeHeld    // "X"
	cubeOffered // "X?"
)

// encodePosition writes a position as its ArkID reads: the score (money, or
// both Away scores on 7 bits and the Crawford mark), the dice, the cube token
// and its exponent, then for each side its squares in unary.
func encodePosition(w *bitWriter, p *domain.Position) error {
	s, err := domain.EncodeArkID(p)
	if err != nil {
		return err
	}
	norm, err := domain.DecodeArkID(s)
	if err != nil {
		return err
	}
	end := strings.IndexByte(s, ']')
	colon := strings.IndexByte(s, ':')
	score, dice, rest := s[1:end], s[end+1:colon], s[colon+1:]
	if score == "$" {
		w.write(1, 1)
	} else {
		w.write(0, 1)
		crawford := strings.HasSuffix(score, "c")
		score = strings.TrimSuffix(score, "c")
		a, b := score[:1], score[1:]
		if i := strings.IndexByte(score, '-'); i >= 0 {
			a, b = score[:i], score[i+1:]
		}
		for _, v := range []string{a, b} {
			n, _ := strconv.Atoi(v)
			w.write(uint64(n), 7)
		}
		w.write(b2u(crawford), 1)
	}
	if dice == "" {
		w.write(0, 1)
	} else {
		w.write(1, 1)
		w.write(uint64(dice[0]-'1'), 3)
		w.write(uint64(dice[1]-'1'), 3)
	}
	if rest != "" && (rest[0] == 'x' || rest[0] == 'X') {
		kind := cubeOnRoll
		if rest[0] == 'X' {
			kind = cubeHeld
		}
		j := 1
		for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
			j++
		}
		exp := 1
		if j > 1 {
			exp, _ = strconv.Atoi(rest[1:j])
		}
		if j < len(rest) && rest[j] == '?' {
			kind = cubeOffered
		}
		w.write(uint64(kind), 2)
		w.write(uint64(exp), 4)
	} else {
		w.write(cubeCentred, 2)
	}
	for _, side := range squaresOf {
		for _, i := range side.points {
			n := 0
			if pt := norm.Board.Points[i]; pt.Checkers > 0 && pt.Color == side.color {
				n = pt.Checkers
			}
			for range n {
				w.write(1, 1)
			}
			w.write(0, 1)
		}
	}
	return nil
}

// squaresOf lists each side's squares in a normalised board: the player on
// roll (Black) owns the bar at 25, the opponent the bar at 0.
var squaresOf = func() [2]struct {
	color  int
	points []int
} {
	var s [2]struct {
		color  int
		points []int
	}
	s[0].color, s[1].color = domain.Black, domain.White
	for i := 1; i <= 24; i++ {
		s[0].points = append(s[0].points, i)
		s[1].points = append(s[1].points, i)
	}
	s[0].points = append(s[0].points, 25)
	s[1].points = append(s[1].points, 0)
	return s
}()

// decodeArk reads one position and spells it as its canonical ArkID, which
// DecodeArkID then judges.
func decodeArk(r *bitReader) (string, error) {
	var sb strings.Builder
	money, err := r.read(1)
	if err != nil {
		return "", err
	}
	if money == 1 {
		sb.WriteString("[$]")
	} else {
		var v [3]uint64
		for i, n := range []int{7, 7, 1} {
			if v[i], err = r.read(n); err != nil {
				return "", err
			}
		}
		sep := ""
		if v[0] > 9 || v[1] > 9 {
			sep = "-"
		}
		sb.WriteString("[" + strconv.Itoa(int(v[0])) + sep + strconv.Itoa(int(v[1])))
		if v[2] == 1 {
			sb.WriteByte('c')
		}
		sb.WriteByte(']')
	}
	hasDice, err := r.read(1)
	if err != nil {
		return "", err
	}
	if hasDice == 1 {
		for range 2 {
			d, err := r.read(3)
			if err != nil {
				return "", err
			}
			sb.WriteByte(byte('1' + d))
		}
	}
	sb.WriteByte(':')
	kind, err := r.read(2)
	if err != nil {
		return "", err
	}
	if kind != cubeCentred {
		exp, err := r.read(4)
		if err != nil {
			return "", err
		}
		c := "X"
		if kind == cubeOnRoll {
			c = "x"
		}
		sb.WriteString(c)
		if exp != 1 {
			sb.WriteString(strconv.Itoa(int(exp)))
		}
		if kind == cubeOffered {
			sb.WriteByte('?')
		}
	}
	var count [2][26]int
	for s, side := range squaresOf {
		for _, i := range side.points {
			for {
				b, err := r.read(1)
				if err != nil {
					return "", err
				}
				if b == 0 {
					break
				}
				if count[s][i]++; count[s][i] > 15 {
					return "", errors.New("more than 15 checkers on a square")
				}
			}
		}
	}
	// The canonical order: the on-roll bar, the points 24 to 1, the
	// opponent's bar; gates in parentheses, then what lies beyond them.
	var gates, rest strings.Builder
	square := func(i, n int, lower bool) {
		if n == 0 {
			return
		}
		letter := byte('A' + i - 1)
		switch i {
		case 24:
			letter = 'Y'
		case 0, 25:
			letter = 'Z'
		}
		if lower {
			letter += 'a' - 'A'
		}
		k := n
		if i != 0 && i != 25 {
			if n >= 2 {
				gates.WriteByte(letter)
			}
			if n == 2 {
				return
			}
			k = max(n-2, 1)
		}
		rest.WriteByte(letter)
		if k != 1 {
			rest.WriteString(strconv.Itoa(k))
		}
	}
	square(25, count[0][25], true)
	for i := 24; i >= 1; i-- {
		square(i, count[0][i], true)
		square(i, count[1][i], false)
	}
	square(0, count[1][0], false)
	if gates.Len() > 0 {
		sb.WriteString("(" + gates.String() + ")")
	}
	sb.WriteString(rest.String())
	return sb.String(), nil
}

func b2u(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// bitWriter packs bits most significant first.
type bitWriter struct {
	buf []byte
	n   int // bits written
}

func (w *bitWriter) write(v uint64, bits int) {
	for i := bits - 1; i >= 0; i-- {
		if w.n%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		if v>>uint(i)&1 == 1 {
			w.buf[len(w.buf)-1] |= 0x80 >> uint(w.n%8)
		}
		w.n++
	}
}

func (w *bitWriter) bytes() []byte { return w.buf }

type bitReader struct {
	b []byte
	n int
}

var errShort = errors.New("data ends inside a position")

func (r *bitReader) read(bits int) (uint64, error) {
	var v uint64
	for range bits {
		if r.n >= len(r.b)*8 {
			return 0, errShort
		}
		v = v<<1 | uint64(r.b[r.n/8]>>(7-uint(r.n%8))&1)
		r.n++
	}
	return v, nil
}
