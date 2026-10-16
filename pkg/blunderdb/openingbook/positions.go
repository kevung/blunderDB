package openingbook

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Roll is a roll of two dice, the higher first.
type Roll struct{ Hi, Lo int }

func (r Roll) String() string { return fmt.Sprintf("%d%d", r.Hi, r.Lo) }

// ParseRoll reads "31" or "13".
func ParseRoll(s string) (Roll, error) {
	if len(s) != 2 || s[0] < '1' || s[0] > '6' || s[1] < '1' || s[1] > '6' {
		return Roll{}, fmt.Errorf("openingbook: roll %q is not two dice 1..6", s)
	}
	a, b := int(s[0]-'0'), int(s[1]-'0')
	return Roll{Hi: max(a, b), Lo: min(a, b)}, nil
}

// Openings lists the 15 non-double rolls in a fixed order: by the higher
// die, then the lower.
func Openings() []Roll {
	var out []Roll
	for hi := 2; hi <= 6; hi++ {
		for lo := 1; lo < hi; lo++ {
			out = append(out, Roll{hi, lo})
		}
	}
	return out
}

// Replies lists the 21 rolls the second player may answer with, doubles
// included, by the higher die, then the lower.
func Replies() []Roll {
	var out []Roll
	for hi := 1; hi <= 6; hi++ {
		for lo := 1; lo <= hi; lo++ {
			out = append(out, Roll{hi, lo})
		}
	}
	return out
}

// Score is the match score of a reply position, as away scores (CONTEXT.md,
// « Away score »): Replier is the player on roll, Opener the one who moved
// first. 1 is the Crawford game; both −1 is money play.
type Score struct{ Replier, Opener int }

// Money is the money-play score.
var Money = Score{domain.Unlimited, domain.Unlimited}

// String is the book's key for the score: "4-2" (the replier 4-away, the
// opener 2-away), "1-4" for a Crawford game, "money".
func (s Score) String() string {
	if s == Money {
		return "money"
	}
	return fmt.Sprintf("%d-%d", s.Replier, s.Opener)
}

// Valid refuses a score the book does not hold: post-Crawford (0), both
// players one point away, or a half-money score.
func (s Score) Valid() error {
	if s == Money {
		return nil
	}
	ok := func(a int) bool { return a >= domain.Crawford && a <= 25 }
	if !ok(s.Replier) || !ok(s.Opener) || (s.Replier == domain.Crawford && s.Opener == domain.Crawford) {
		return fmt.Errorf("openingbook: score %s is not one the book holds (each side 1..25 away, at most one Crawford side)", s)
	}
	return nil
}

// ParseScore reads the key String writes.
func ParseScore(key string) (Score, error) {
	key = strings.TrimSpace(key)
	if strings.EqualFold(key, "money") {
		return Money, nil
	}
	a, b, found := strings.Cut(key, "-")
	if !found {
		return Score{}, fmt.Errorf("openingbook: score %q is not replier-opener (e.g. 4-2) or money", key)
	}
	r, err1 := strconv.Atoi(a)
	o, err2 := strconv.Atoi(b)
	if err1 != nil || err2 != nil {
		return Score{}, fmt.Errorf("openingbook: score %q is not replier-opener (e.g. 4-2) or money", key)
	}
	s := Score{r, o}
	return s, s.Valid()
}

// GridScores is the default grid, twenty scores: every pair of 2- to 7-away
// at most one point apart (16), and the Crawford game at 1-away against 2-
// and 4-away, the replier leading or trailing (4).
func GridScores() []Score {
	var out []Score
	for r := 2; r <= 7; r++ {
		for o := 2; o <= 7; o++ {
			if r-o <= 1 && o-r <= 1 {
				out = append(out, Score{r, o})
			}
		}
	}
	for _, n := range []int{2, 4} {
		out = append(out, Score{domain.Crawford, n}, Score{n, domain.Crawford})
	}
	return out
}

// FullScores is the complete grid: every pair of 2- to 7-away (36) and every
// Crawford game at 1-away against 2- to 7-away, both ways (12).
func FullScores() []Score {
	var out []Score
	for r := 2; r <= 7; r++ {
		for o := 2; o <= 7; o++ {
			out = append(out, Score{r, o})
		}
	}
	for n := 2; n <= 7; n++ {
		out = append(out, Score{domain.Crawford, n}, Score{n, domain.Crawford})
	}
	return out
}

// ParseScores reads "grid", "full", or a comma-separated list of keys
// ("4-2,2-4,1-2,money").
func ParseScores(spec string) ([]Score, error) {
	switch strings.ToLower(strings.TrimSpace(spec)) {
	case "", "grid":
		return GridScores(), nil
	case "full":
		return FullScores(), nil
	}
	var out []Score
	seen := map[Score]bool{}
	for _, k := range strings.Split(spec, ",") {
		s, err := ParseScore(k)
		if err != nil {
			return nil, err
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out, nil
}

// OpeningPosition is the position of the first move: the opener (Black,
// moving from 24 to 1) on roll with opening, the cube centred at 1, at
// score s. Score is still read replier first: one question asks both moves
// of the same game.
func OpeningPosition(opening Roll, s Score) (domain.Position, error) {
	if opening.Hi == opening.Lo || opening.Lo < 1 || opening.Hi > 6 {
		return domain.Position{}, fmt.Errorf("openingbook: %s is not an opening roll", opening)
	}
	if err := s.Valid(); err != nil {
		return domain.Position{}, err
	}
	pos := domain.InitializePosition()
	pos.PlayerOnRoll = domain.Black
	pos.Dice = [2]int{opening.Hi, opening.Lo}
	pos.DecisionType = domain.CheckerAction
	pos.Cube = domain.Cube{Owner: domain.None, Value: 0}
	pos.Score = [2]int{}
	pos.Score[domain.White] = s.Replier
	pos.Score[domain.Black] = s.Opener
	return pos, nil
}

// ReplyPosition is the position the second player faces: the opener has
// played play (a candidate of the opening line, in the book's notation, hits
// marked), the replier (White) is on roll with reply, at score s. The play
// is the book's best at that score, not a fixed table: the reply follows
// what the opening line says.
func ReplyPosition(opening Roll, play string, reply Roll, s Score) (domain.Position, error) {
	pos, err := OpeningPosition(opening, s)
	if err != nil {
		return domain.Position{}, err
	}
	p, err := engine.ResolvePlay(&pos, domain.LegalMoves(&pos), play, true)
	if err != nil {
		return domain.Position{}, fmt.Errorf("openingbook: opening %s: %w", opening, err)
	}
	out := p.Result
	out.PlayerOnRoll = domain.White
	out.Dice = [2]int{reply.Hi, reply.Lo}
	out.DecisionType = domain.CheckerAction
	out.Cube = domain.Cube{Owner: domain.None, Value: 0}
	out.Score = pos.Score
	return out, nil
}
