package openingbook

import (
	"compress/gzip"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The exercise « which play at this score? »: a reply position is shown at a
// score, the user plays, and every play tied with the best is a right answer.
// Everything here is a pure function of the book, so the desktop, the CLI and
// the daemon draw and judge alike, with no database behind them.

// Status says what the book holds, so an exercise over an empty book says so
// instead of failing.
type Status struct {
	Lines int `json:"lines"`
	// Scores are the score keys the book holds, in grid order.
	Scores []string `json:"scores"`
}

// Status reads the book's size and scores.
func (b *Book) Status() Status {
	seen := map[Score]bool{}
	var scores []Score
	for _, l := range b.Lines() {
		s, err := ParseScore(l.Score)
		if err != nil || seen[s] {
			continue
		}
		seen[s] = true
		scores = append(scores, s)
	}
	sort.Slice(scores, func(i, j int) bool { return scoreLess(scores[i], scores[j]) })
	out := Status{Lines: b.Len(), Scores: make([]string, len(scores))}
	for i, s := range scores {
		out.Scores[i] = s.String()
	}
	return out
}

// Money sorts last; the others by the replier's away score, then the opener's.
func scoreLess(a, b Score) bool {
	if (a == Money) != (b == Money) {
		return b == Money
	}
	if a.Replier != b.Replier {
		return a.Replier < b.Replier
	}
	return a.Opener < b.Opener
}

// QuestionRequest narrows the draw. Every list is optional: empty means any.
type QuestionRequest struct {
	Scores   []string `json:"scores,omitempty"`   // score keys, "4-2", "money"
	Openings []string `json:"openings,omitempty"` // opening rolls, "31"
	Replies  []string `json:"replies,omitempty"`  // reply rolls, "52"
	// Seed makes the draw reproducible; 0 draws from the clock.
	Seed uint64 `json:"seed,omitempty"`
}

// Question is the first two rolls of a game at one score: the opener's roll
// to play first, then the reply roll, asked once the opening is judged (the
// Verdict's Next). Empty says the book (or the part the request narrowed it
// to) holds nothing to ask, which is not an error.
type Question struct {
	Empty   bool   `json:"empty"`
	Opening string `json:"opening,omitempty"`
	Reply   string `json:"reply,omitempty"`
	Score   string `json:"score,omitempty"`
	// Position is the opening position, the opener on roll.
	Position domain.Position `json:"position"`
	XGID     string          `json:"xgid,omitempty"`
	// Candidates is how many opening plays the book ranks.
	Candidates int `json:"candidates,omitempty"`
}

// Draw picks one question among those the request allows, uniformly over
// (opening, reply, score). A question needs both lines: the opening at the
// score, and the reply after the book's best opening play at that score.
func (b *Book) Draw(req QuestionRequest) (Question, error) {
	scores, err := keySet(req.Scores, func(s string) (string, error) {
		sc, err := ParseScore(s)
		return sc.String(), err
	})
	if err != nil {
		return Question{}, err
	}
	rollKey := func(s string) (string, error) {
		r, err := ParseRoll(s)
		return r.String(), err
	}
	openings, err := keySet(req.Openings, rollKey)
	if err != nil {
		return Question{}, err
	}
	replies, err := keySet(req.Replies, rollKey)
	if err != nil {
		return Question{}, err
	}
	type pick struct {
		k     Key
		reply Roll
		n     int
	}
	var pool []pick
	for _, l := range b.Lines() {
		k, err := l.Key()
		if err != nil || !k.IsOpening() || len(l.Candidates) == 0 ||
			!allows(scores, l.Score) || !allows(openings, l.Opening) {
			continue
		}
		best := l.Rank()[0].Move
		for _, r := range Replies() {
			if !allows(replies, r.String()) {
				continue
			}
			if rl, ok := b.Lookup(ReplyKey(k.Opening, best, r, k.Score)); ok && len(rl.Candidates) > 0 {
				pool = append(pool, pick{k, r, len(l.Candidates)})
			}
		}
	}
	if len(pool) == 0 {
		return Question{Empty: true}, nil
	}
	seed := req.Seed
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	p := pool[rand.New(rand.NewPCG(seed, seed>>32|1)).IntN(len(pool))]
	pos, err := p.k.Position()
	if err != nil {
		return Question{}, err
	}
	return Question{
		Opening: p.k.Opening.String(), Reply: p.reply.String(), Score: p.k.Score.String(),
		Position: pos, XGID: "XGID=" + domain.EncodeXGID(&pos), Candidates: p.n,
	}, nil
}

func keySet(in []string, norm func(string) (string, error)) (map[string]bool, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := map[string]bool{}
	for _, s := range in {
		k, err := norm(s)
		if err != nil {
			return nil, err
		}
		out[k] = true
	}
	return out, nil
}

func allows(set map[string]bool, k string) bool { return set == nil || set[k] }

// Answer is one answer of a question: the opening play when OpeningPlay is
// empty, else the reply played after OpeningPlay (the Next the opening's
// verdict named). The play is the board the user left, or its notation when
// typed; Board wins when both are given. Neither is no answer (out of time):
// the verdict is then the correction alone, and the opening's still names the
// reply step, after the book's best play.
type Answer struct {
	Opening     string        `json:"opening"`
	OpeningPlay string        `json:"openingPlay,omitempty"`
	Reply       string        `json:"reply"`
	Score       string        `json:"score"`
	Board       *domain.Board `json:"board,omitempty"`
	Notation    string        `json:"notation,omitempty"`
}

// Verdict judges an answer against its line.
type Verdict struct {
	// Legal is false when the answer is no legal play of the roll.
	Legal bool `json:"legal"`
	// Notation is the play as the move generator writes it.
	Notation string `json:"notation"`
	// Played is the index of the answer in Ranked, −1 when the book does not
	// rank it: a play outside the candidates is taken as behind all of them.
	Played int `json:"played"`
	// Right says the answer is tied with the best (Ranked.Tied).
	Right bool `json:"right"`
	// Ranked is the line's correction: every candidate against the best.
	Ranked []Ranked `json:"ranked"`
	// Signature and Games are the line's rollout: its settings and how many
	// games the best play rolled. A coarse rollout widens σ(diff) and with it
	// the tied group; the screen shows them so the reader knows how fine the
	// verdict is.
	Signature string `json:"signature"`
	Games     int    `json:"games"`
	// Next is the reply step, on an opening verdict whose answer named a
	// reply roll the book holds.
	Next *ReplyStep `json:"next,omitempty"`
}

// ReplyStep is the second half of a question: the reply roll in the
// position the opening play left.
type ReplyStep struct {
	// OpeningPlay is the opening play the reply follows: the user's when it
	// is tied with the best and the book holds the reply after it, else the
	// book's best at the score.
	OpeningPlay string `json:"openingPlay"`
	// FollowsBest says OpeningPlay is the book's best rather than the user's
	// play, which the screen states.
	FollowsBest bool            `json:"followsBest"`
	Reply       string          `json:"reply"`
	Position    domain.Position `json:"position"`
	XGID        string          `json:"xgid"`
	Candidates  int             `json:"candidates"`
}

// Grade judges an answer. A key the book does not hold is an error: the
// question came from the book, so the answer must name one of its lines.
func (b *Book) Grade(a Answer) (Verdict, error) {
	o, err := ParseRoll(a.Opening)
	if err != nil {
		return Verdict{}, err
	}
	s, err := ParseScore(a.Score)
	if err != nil {
		return Verdict{}, err
	}
	var reply Roll
	if a.Reply != "" {
		if reply, err = ParseRoll(a.Reply); err != nil {
			return Verdict{}, err
		}
	}
	k := OpeningKey(o, s)
	if a.OpeningPlay != "" {
		if a.Reply == "" {
			return Verdict{}, fmt.Errorf("openingbook: a reply answer names its reply roll")
		}
		k = ReplyKey(o, a.OpeningPlay, reply, s)
	}
	line, ok := b.Lookup(k)
	if !ok {
		return Verdict{}, fmt.Errorf("openingbook: the book holds no line %s", k)
	}
	v, err := grade(k, line, a)
	if err != nil || !k.IsOpening() || a.Reply == "" {
		return v, err
	}
	if v.Next, err = b.next(k, v, reply); err != nil {
		return Verdict{}, err
	}
	return v, nil
}

func grade(k Key, line Line, a Answer) (Verdict, error) {
	pos, err := k.Position()
	if err != nil {
		return Verdict{}, err
	}
	legal := domain.LegalMoves(&pos)
	v := Verdict{Played: -1, Ranked: line.Rank(), Signature: line.Signature, Games: line.Games}
	var played domain.LegalPlay
	switch {
	case a.Board != nil:
		for _, p := range legal {
			if domain.SameBoard(p.Result.Board, *a.Board) {
				played, v.Legal = p, true
				break
			}
		}
	case strings.TrimSpace(a.Notation) != "":
		v.Notation = a.Notation
		if p, err := engine.ResolvePlay(&pos, legal, a.Notation, false); err == nil {
			played, v.Legal = p, true
		}
	}
	if !v.Legal {
		return v, nil
	}
	v.Notation = played.Notation
	for i, c := range v.Ranked {
		cp, err := engine.ResolvePlay(&pos, legal, c.Move, true)
		if err != nil || !domain.SameBoard(cp.Result.Board, played.Result.Board) {
			continue
		}
		v.Played, v.Right = i, c.Tied
		break
	}
	return v, nil
}

// next is the reply step after an opening verdict: after the user's play
// when it is right and the book follows it, after the book's best otherwise.
func (b *Book) next(k Key, v Verdict, reply Roll) (*ReplyStep, error) {
	if len(v.Ranked) == 0 {
		return nil, nil
	}
	play, best := v.Ranked[0].Move, true
	if v.Right && v.Played > 0 {
		mine := v.Ranked[v.Played].Move
		if _, ok := b.Lookup(ReplyKey(k.Opening, mine, reply, k.Score)); ok {
			play, best = mine, false
		}
	} else if v.Played == 0 {
		best = false
	}
	rk := ReplyKey(k.Opening, play, reply, k.Score)
	rl, ok := b.Lookup(rk)
	if !ok {
		return nil, fmt.Errorf("openingbook: the book holds no line %s", rk)
	}
	pos, err := rk.Position()
	if err != nil {
		return nil, err
	}
	return &ReplyStep{
		OpeningPlay: play, FollowsBest: best, Reply: reply.String(), Position: pos,
		XGID: "XGID=" + domain.EncodeXGID(&pos), Candidates: len(rl.Candidates),
	}, nil
}

// Open reads a book file, gzipped or not, strictly (Load): a run's own book
// can be tried before it is embedded.
func Open(path string) (*Book, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("openingbook: %w", err)
	}
	defer func() { _ = f.Close() }()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		zr, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("openingbook: %s: %w", path, err)
		}
		defer func() { _ = zr.Close() }()
		r = zr
	}
	return Load(r)
}
