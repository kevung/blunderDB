package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ArkID (Animal Register Key) is blunderDB's own readable, pronounceable
// spelling of a Position: [score]dice:cube(gates)rest.
//
//	[34]65:x2(ySQmLhfA)S3Qm3L3hf3
//
// # The viewpoint
//
// A Position is stored from the side of the player on roll (Black, index 0,
// after NormalizeForStorage), so the ArkID is written from that side too:
// lowercase letters are the player on roll, uppercase letters the opponent,
// and both are numbered the way the player on roll numbers the board. The
// letters A to W are points 1 to 23, Y is point 24, Z is the bar. X is the
// cube. A letter is the initial of an animal (the bestiary of the glossary)
// that helps remember which point it is.
//
// # The fields
//
//   - Score    "[34]": the Away scores of the player on roll then of the
//     opponent, glued when both are 9 or less, "a-b" otherwise
//     ("[11-9]"). A "c" suffix marks the Crawford game ("[14c]"); an
//     Away score of 1 without it is post-Crawford (CONTEXT.md, « Away
//     score »). "[$]" is money play.
//   - Dice     two digits, larger first; absent for a cube decision.
//   - Cube     "x" held by the player on roll, "X" held by the opponent,
//     followed by the exponent when it is not 1 ("x" is a cube at 2,
//     "x2" a cube at 4, "x0" a cube at 1 held); nothing when centred. A
//     cube offered, to take or pass, is "X" (the offer comes from the
//     opponent of the player answering), the exponent when it is not 1, and
//     "?": "X?" is a cube offered at 2, "X2?" at 4. It reads back as a
//     centred cube of that exponent, the stored shape of a response
//     position (IsResponsePosition); "x?" is refused.
//   - Gates    in parentheses, one letter per point of two checkers or more
//     (never the bar).
//   - Rest     each square once as a letter and a count k, omitted when 1: the
//     checkers beyond two on a gate, 1 for a lone checker, the number
//     of checkers for the bar. Checkers borne off are 15 minus the
//     total.
//
// # Reading is tolerant, writing is strict
//
// The reader takes the compact form and the developed form (every lowercase
// letter, then every uppercase letter, each with its own parentheses), any
// order and any number of parenthesised groups. The writer produces the
// canonical compact form (EncodeArkID), whose order is the bar of the player
// on roll, the points 24 to 1 with both sides mixed, then the opponent's bar;
// or the developed form on request (EncodeArkIDExpanded). Two spellings of one
// Position by blunderDB are therefore identical to the character.
//
// # What the notation does not carry
//
// The session rules (Jacoby, beaver), the cube ceiling and the provenance are
// not part of a Position's identity (ADR-0028): decoding sets none of them, as
// DecodeOGID does. A centred cube above 1 is spelled only as an offered cube
// (a response position); on a checker decision it is refused, as are dice on
// a cube decision, since the hash folds both in and the notation has no room
// for them. Decoding returns the Position normalised, Black on roll, which is
// the hash's own view.

// ErrInvalidArkID is returned for malformed ArkID strings. Callers map it to a
// 4xx response, as they do ErrInvalidXGID.
var ErrInvalidArkID = errors.New("invalid ArkID")

const arkMaxAway = 99

// arkLetter is a square's letter: index 1..24 are points A..Y without X,
// index 25 (the bar) is Z. The on-roll bar of a board is index 25, the
// opponent's index 0; both are the letter Z in their own case.
func arkLetter(point int) byte {
	switch {
	case point == 24:
		return 'Y'
	case point == 25 || point == 0:
		return 'Z'
	default:
		return byte('A' + point - 1)
	}
}

// arkPoint is arkLetter's inverse on an uppercase letter: 1..24, or 0 for Z.
func arkPoint(upper byte) (int, bool) {
	switch {
	case upper >= 'A' && upper <= 'W':
		return int(upper-'A') + 1, true
	case upper == 'Y':
		return 24, true
	case upper == 'Z':
		return 0, true
	}
	return 0, false
}

type arkSquare struct {
	point int // 1..24 for a point, 25 for a bar
	n     int
	upper bool
}

// arkSquares lists the occupied squares of a normalised position, in the
// canonical order: the on-roll bar, points 24 to 1, the opponent's bar.
func arkSquares(norm *Position) ([]arkSquare, error) {
	var out []arkSquare
	var total [2]int
	at := func(i int) (Point, error) {
		p := norm.Board.Points[i]
		if p.Checkers <= 0 {
			return p, nil
		}
		if p.Color != Black && p.Color != White {
			return p, fmt.Errorf("%w: point %d holds checkers of no colour", ErrInvalidArkID, i)
		}
		total[p.Color] += p.Checkers
		return p, nil
	}
	order := make([]int, 0, 26)
	order = append(order, 25)
	for i := 24; i >= 1; i-- {
		order = append(order, i)
	}
	order = append(order, 0)
	for _, i := range order {
		p, err := at(i)
		if err != nil {
			return nil, err
		}
		if p.Checkers <= 0 {
			continue
		}
		switch {
		case i == 25 && p.Color != Black, i == 0 && p.Color != White:
			return nil, fmt.Errorf("%w: checkers of the wrong colour on the bar", ErrInvalidArkID)
		}
		out = append(out, arkSquare{point: i, n: p.Checkers, upper: p.Color == White})
	}
	if total[Black] > 15 || total[White] > 15 {
		return nil, fmt.Errorf("%w: more than 15 checkers for one player", ErrInvalidArkID)
	}
	return out, nil
}

func arkScore(norm *Position) (string, error) {
	a, b := norm.Score[0], norm.Score[1]
	if a == Unlimited && b == Unlimited {
		return "[$]", nil
	}
	if a == Unlimited || b == Unlimited {
		return "", fmt.Errorf("%w: money play for one player only", ErrInvalidArkID)
	}
	crawford := a == Crawford || b == Crawford
	a, b = PointsAway(a), PointsAway(b)
	if a < 1 || b < 1 || a > arkMaxAway || b > arkMaxAway {
		return "", fmt.Errorf("%w: away scores %d-%d", ErrInvalidArkID, norm.Score[0], norm.Score[1])
	}
	s := "[" + strconv.Itoa(a) + strconv.Itoa(b)
	if a > 9 || b > 9 {
		s = "[" + strconv.Itoa(a) + "-" + strconv.Itoa(b)
	}
	if crawford {
		s += "c"
	}
	return s + "]", nil
}

func arkEncode(pos *Position, expanded bool) (string, error) {
	norm := pos.NormalizeForStorage()
	score, err := arkScore(&norm)
	if err != nil {
		return "", err
	}
	squares, err := arkSquares(&norm)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(score)
	if norm.DecisionType != CubeAction {
		d1, d2 := max(norm.Dice[0], norm.Dice[1]), min(norm.Dice[0], norm.Dice[1])
		if d2 < 1 || d1 > 6 {
			return "", fmt.Errorf("%w: dice %d-%d", ErrInvalidArkID, norm.Dice[0], norm.Dice[1])
		}
		sb.WriteString(strconv.Itoa(d1) + strconv.Itoa(d2))
	}
	sb.WriteByte(':')

	switch {
	case norm.DecisionType == CubeAction && (norm.Dice[0] >= 1 && norm.Dice[0] <= 6 && norm.Dice[1] >= 1 && norm.Dice[1] <= 6):
		// The hash folds in valid dice, and a cube decision spells none.
		return "", fmt.Errorf("%w: a cube decision with dice %d-%d", ErrInvalidArkID, norm.Dice[0], norm.Dice[1])
	case IsResponsePosition(&norm):
		if norm.Cube.Value > 10 {
			return "", fmt.Errorf("%w: cube exponent %d", ErrInvalidArkID, norm.Cube.Value)
		}
		sb.WriteByte('X')
		if norm.Cube.Value != 1 {
			sb.WriteString(strconv.Itoa(norm.Cube.Value))
		}
		sb.WriteByte('?')
	case norm.Cube.Owner == None && norm.Cube.Value > 0:
		return "", fmt.Errorf("%w: a centred cube of exponent %d on a checker decision", ErrInvalidArkID, norm.Cube.Value)
	case norm.Cube.Owner == Black || norm.Cube.Owner == White:
		if norm.Cube.Value < 0 || norm.Cube.Value > 10 {
			return "", fmt.Errorf("%w: cube exponent %d", ErrInvalidArkID, norm.Cube.Value)
		}
		c := byte('x')
		if norm.Cube.Owner == White {
			c = 'X'
		}
		sb.WriteByte(c)
		if norm.Cube.Value != 1 {
			sb.WriteString(strconv.Itoa(norm.Cube.Value))
		}
	}

	// part writes the gates of the squares kept, then their rest.
	part := func(keep func(arkSquare) bool) {
		var gates, rest strings.Builder
		for _, s := range squares {
			if !keep(s) {
				continue
			}
			letter := arkLetter(s.point)
			if !s.upper {
				letter += 'a' - 'A'
			}
			k := s.n
			if s.point != 25 && s.point != 0 {
				if s.n >= 2 {
					gates.WriteByte(letter)
				}
				if s.n == 2 {
					continue
				}
				k = max(s.n-2, 1)
			}
			rest.WriteByte(letter)
			if k != 1 {
				rest.WriteString(strconv.Itoa(k))
			}
		}
		if gates.Len() > 0 {
			sb.WriteString("(" + gates.String() + ")")
		}
		sb.WriteString(rest.String())
	}
	if expanded {
		part(func(s arkSquare) bool { return !s.upper })
		part(func(s arkSquare) bool { return s.upper })
	} else {
		part(func(arkSquare) bool { return true })
	}
	return sb.String(), nil
}

// EncodeArkID renders a Position as its canonical compact ArkID.
func EncodeArkID(pos *Position) (string, error) { return arkEncode(pos, false) }

// EncodeArkIDExpanded renders a Position as its developed ArkID: the player on
// roll's squares first, then the opponent's, each with its own gates.
func EncodeArkIDExpanded(pos *Position) (string, error) { return arkEncode(pos, true) }

// LooksLikeArkID reports whether text opens like an ArkID: an optional
// "ArkID=" prefix, then a bracketed score. It is a routing test only; the
// decoder is the judge.
func LooksLikeArkID(text string) bool {
	s := arkTrim(text)
	if !strings.HasPrefix(s, "[") {
		return false
	}
	end := strings.IndexByte(s, ']')
	return end > 0 && strings.Contains(s[end:], ":")
}

func arkTrim(text string) string {
	s := strings.TrimSpace(text)
	if len(s) >= 6 && strings.EqualFold(s[:6], "ArkID=") {
		s = strings.TrimSpace(s[6:])
	}
	return s
}

func arkErr(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidArkID}, a...)...)
}

// DecodeArkID parses an ArkID into a Position, normalised (Black on roll).
// The "ArkID=" prefix is optional.
func DecodeArkID(text string) (Position, error) {
	var pos Position
	s := arkTrim(text)
	if s == "" {
		return pos, arkErr("empty")
	}
	if s[0] != '[' {
		return pos, arkErr("the score between brackets comes first")
	}
	end := strings.IndexByte(s, ']')
	if end < 0 {
		return pos, arkErr("unclosed score bracket")
	}
	score, err := arkParseScore(s[1:end])
	if err != nil {
		return pos, err
	}
	rest := s[end+1:]
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return pos, arkErr("missing ':' after the dice")
	}
	dice := rest[:colon]
	rest = rest[colon+1:]

	pos.Score = score
	pos.PlayerOnRoll = Black
	switch {
	case dice == "":
		pos.DecisionType = CubeAction
	case len(dice) == 2 && dice[0] >= '1' && dice[0] <= '6' && dice[1] >= '1' && dice[1] <= '6':
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{max(int(dice[0]-'0'), int(dice[1]-'0')), min(int(dice[0]-'0'), int(dice[1]-'0'))}
	default:
		return pos, arkErr("dice %q: two digits from 1 to 6 or nothing", dice)
	}

	pos.Cube.Owner = None
	if rest != "" && (rest[0] == 'x' || rest[0] == 'X') {
		pos.Cube.Owner = Black
		if rest[0] == 'X' {
			pos.Cube.Owner = White
		}
		n, digits := 1, 0
		for digits < len(rest)-1 && rest[1+digits] >= '0' && rest[1+digits] <= '9' {
			digits++
		}
		if digits > 0 {
			n, _ = strconv.Atoi(rest[1 : 1+digits])
			if n > 10 {
				return pos, arkErr("cube exponent %d, at most 10", n)
			}
		}
		pos.Cube.Value = n
		rest = rest[1+digits:]
		if rest != "" && rest[0] == '?' {
			if pos.Cube.Owner != White || pos.DecisionType != CubeAction || n < 1 {
				return pos, arkErr("an offered cube is X, at least exponent 1, on a cube decision")
			}
			pos.Cube.Owner = None
			rest = rest[1:]
		}
	}

	for i := range pos.Board.Points {
		pos.Board.Points[i] = Point{Checkers: 0, Color: None}
	}
	if err := arkParseBoard(&pos, rest); err != nil {
		return Position{}, err
	}
	return pos, nil
}

func isArkLetter(c byte) bool { return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' }

func arkParseScore(in string) ([2]int, error) {
	if in == "$" {
		return [2]int{Unlimited, Unlimited}, nil
	}
	crawford := strings.HasSuffix(in, "c")
	in = strings.TrimSuffix(in, "c")
	var a, b string
	if i := strings.IndexByte(in, '-'); i >= 0 {
		a, b = in[:i], in[i+1:]
	} else if len(in) == 2 {
		a, b = in[:1], in[1:]
	} else {
		return [2]int{}, arkErr("score %q: two digits, a-b, or $", in)
	}
	var away [2]int
	for i, t := range [2]string{a, b} {
		n, err := strconv.Atoi(t)
		if err != nil || t == "" || t[0] == '+' || n < 1 || n > arkMaxAway {
			return [2]int{}, arkErr("away score %q", t)
		}
		away[i] = n
	}
	if crawford && away[0] != 1 && away[1] != 1 {
		return [2]int{}, arkErr("Crawford game with no player one point away")
	}
	for i, n := range away {
		if n == 1 {
			away[i] = PostCrawford
			if crawford {
				away[i] = Crawford
			}
		}
	}
	return away, nil
}

func arkParseBoard(pos *Position, rest string) error {
	var gate [2][26]bool // [upper][point]
	type seen struct{ gate, item bool }
	var marks [26]seen
	var count [2][26]int
	var bar [2]int

	i := 0
	readNum := func() (int, bool, error) {
		j := i
		for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
			j++
		}
		if j == i {
			return 1, false, nil
		}
		n, err := strconv.Atoi(rest[i:j])
		if err != nil || n > 15 {
			return 0, true, arkErr("count %q", rest[i:j])
		}
		i = j
		return n, true, nil
	}
	for i < len(rest) {
		c := rest[i]
		switch {
		case c == '(':
			closeAt := strings.IndexByte(rest[i:], ')')
			if closeAt < 0 {
				return arkErr("unclosed parenthesis")
			}
			for _, g := range []byte(rest[i+1 : i+closeAt]) {
				upper := g >= 'A' && g <= 'Z'
				p, ok := arkPoint(g &^ 0x20)
				if !ok || p == 0 || !isArkLetter(g) {
					return arkErr("unknown letter %q in the gates (the bar has none)", string(g))
				}
				u := 0
				if upper {
					u = 1
				}
				if marks[p].gate {
					return arkErr("point %c is a gate twice", arkLetter(p))
				}
				marks[p].gate = true
				gate[u][p] = true
			}
			i += closeAt + 1
		case c == ')':
			return arkErr("unopened parenthesis")
		default:
			upper := c >= 'A' && c <= 'Z'
			if !isArkLetter(c) {
				return arkErr("unknown character %q", string(c))
			}
			p, ok := arkPoint(c &^ 0x20)
			if !ok {
				return arkErr("unknown letter %q", string(c))
			}
			i++
			k, given, err := readNum()
			if err != nil {
				return err
			}
			if given && k == 0 {
				return arkErr("count 0 after %q", string(c))
			}
			u := 0
			if upper {
				u = 1
			}
			if p == 0 {
				if bar[u] != 0 {
					return arkErr("the bar of one player is written twice")
				}
				bar[u] = k
				continue
			}
			if marks[p].item {
				return arkErr("point %c is written twice", arkLetter(p))
			}
			marks[p].item = true
			count[u][p] = k
		}
	}

	// Every gate and item resolves to a count; then the camps must not meet.
	for p := 1; p <= 24; p++ {
		var n [2]int
		for u := 0; u < 2; u++ {
			if gate[u][p] {
				n[u] = 2
				if count[u][p] > 0 {
					n[u] += count[u][p]
				}
			} else if count[u][p] > 0 {
				if count[u][p] != 1 {
					return arkErr("point %c holds %d checkers and needs a gate", arkLetter(p), count[u][p])
				}
				n[u] = 1
			}
		}
		if n[0] > 0 && n[1] > 0 {
			return arkErr("point %c is held by both players", arkLetter(p))
		}
		switch {
		case n[0] > 0:
			pos.Board.Points[p] = Point{Checkers: n[0], Color: Black}
		case n[1] > 0:
			pos.Board.Points[p] = Point{Checkers: n[1], Color: White}
		}
	}
	if bar[0] > 0 {
		pos.Board.Points[25] = Point{Checkers: bar[0], Color: Black}
	}
	if bar[1] > 0 {
		pos.Board.Points[0] = Point{Checkers: bar[1], Color: White}
	}
	var onBoard [2]int
	for _, p := range pos.Board.Points {
		if p.Checkers > 0 {
			onBoard[p.Color] += p.Checkers
		}
	}
	for color := 0; color < 2; color++ {
		if onBoard[color] > 15 {
			return arkErr("%d checkers for one player, at most 15", onBoard[color])
		}
		pos.Board.Bearoff[color] = 15 - onBoard[color]
	}
	return nil
}
