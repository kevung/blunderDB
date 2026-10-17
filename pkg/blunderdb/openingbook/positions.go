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
// first. 1 is the Crawford game; both −1 is money play; both 0 is DMP.
type Score struct{ Replier, Opener int }

// Money is the money-play score.
var Money = Score{domain.Unlimited, domain.Unlimited}

// DMP is double match point, both players one point away. Someone reached
// 1-away first and played the Crawford game, so the game at 1-1 is always
// post-Crawford: both sides carry the post-Crawford sentinel, the cube is
// alive but worthless and gammons count for nothing.
var DMP = Score{domain.PostCrawford, domain.PostCrawford}

// String is the book's key for the score: "4-2" (the replier 4-away, the
// opener 2-away), "1-4" for a Crawford game, "1-1" for DMP, "money".
func (s Score) String() string {
	switch s {
	case Money:
		return "money"
	case DMP:
		return "1-1"
	}
	return fmt.Sprintf("%d-%d", s.Replier, s.Opener)
}

// Valid refuses a score the book does not hold: post-Crawford against a
// player more than one point away, or a half-money score. Both sides one
// point away is DMP, never a Crawford game.
func (s Score) Valid() error {
	if s == Money || s == DMP {
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
	if key == "1-1" {
		return DMP, nil
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
	if s == DMP {
		// The sentinel is internal: DMP is written "1-1".
		return Score{}, fmt.Errorf("openingbook: score %q is not one the book holds (DMP is 1-1)", key)
	}
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

// XGScores are the four scores of an XG-style opening drill: money, DMP,
// gammon save (the replier 2-away against an opener 1-away Crawford) and
// gammon go (the reverse). Kept apart from GridScores: the grid is what the
// embedded book is held to cover, and money and DMP join it once their lines
// are rolled.
func XGScores() []Score {
	return []Score{Money, DMP, {2, domain.Crawford}, {domain.Crawford, 2}}
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

// ParseScores reads "grid", "full", "xg", or a comma-separated list of keys
// ("4-2,2-4,1-2,1-1,money").
func ParseScores(spec string) ([]Score, error) {
	switch strings.ToLower(strings.TrimSpace(spec)) {
	case "", "grid":
		return GridScores(), nil
	case "full":
		return FullScores(), nil
	case "xg":
		return XGScores(), nil
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
