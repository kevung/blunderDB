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

// The two halves of a game's first two rolls, asked apart.
const (
	// RollFirst asks the opening play, the opener on roll at the start.
	RollFirst = "first"
	// RollSecond asks the reply, in the position the book's best opening
	// play at the score leaves: the book ranks replies after that play only.
	RollSecond = "second"
)

// QuestionRequest narrows the draw. Every list is optional: empty means any.
type QuestionRequest struct {
	// Roll is RollFirst (the default when empty) or RollSecond.
	Roll     string   `json:"roll,omitempty"`
	Scores   []string `json:"scores,omitempty"`   // score keys, "4-2", "money"
	Openings []string `json:"openings,omitempty"` // opening rolls, "31"
	Replies  []string `json:"replies,omitempty"`  // reply rolls, "52"; a second-roll draw only
	// Seed makes the draw reproducible; 0 draws from the clock.
	Seed uint64 `json:"seed,omitempty"`
}

// Question is one play to find at one score: the opening play (RollFirst),
// or the reply after the book's best opening play (RollSecond). Empty says
// the book (or the part the request narrowed it to) holds nothing to ask,
// which is not an error.
type Question struct {
	Empty   bool   `json:"empty"`
	Roll    string `json:"roll,omitempty"`
	Opening string `json:"opening,omitempty"`
	// OpeningPlay is the opening play already on the board of a second-roll
	// question; Reply is the roll to play there.
	OpeningPlay string `json:"openingPlay,omitempty"`
	Reply       string `json:"reply,omitempty"`
	Score       string `json:"score,omitempty"`
	// Position is the position to play, its player on roll with the dice.
	Position domain.Position `json:"position"`
	XGID     string          `json:"xgid,omitempty"`
	// Candidates is how many plays the book ranks in Position.
	Candidates int `json:"candidates,omitempty"`
}

func rollOf(s string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "", RollFirst, "1":
		return RollFirst, nil
	case RollSecond, "2":
		return RollSecond, nil
	}
	return "", fmt.Errorf("openingbook: roll %q is neither %s nor %s", s, RollFirst, RollSecond)
}

// Draw picks one question among those the request allows, uniformly over
// the lines that can ask it: (opening, score) for the first roll, (opening,
// reply, score) after the book's best opening play for the second.
func (b *Book) Draw(req QuestionRequest) (Question, error) {
	roll, err := rollOf(req.Roll)
	if err != nil {
		return Question{}, err
	}
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
		k    Key
		play string
		n    int
	}
	var pool []pick
	for _, l := range b.Lines() {
		k, err := l.Key()
		if err != nil || !k.IsOpening() || len(l.Candidates) == 0 ||
			!allows(scores, l.Score) || !allows(openings, l.Opening) {
			continue
		}
		if roll == RollFirst {
			pool = append(pool, pick{k, "", len(l.Candidates)})
			continue
		}
		best := l.Rank()[0].Move
		for _, r := range Replies() {
			if !allows(replies, r.String()) {
				continue
			}
			rk := ReplyKey(k.Opening, best, r, k.Score)
			if rl, ok := b.Lookup(rk); ok && len(rl.Candidates) > 0 {
				pool = append(pool, pick{rk, best, len(rl.Candidates)})
			}
		}
	}
	if len(pool) == 0 {
		return Question{Empty: true, Roll: roll}, nil
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
	q := Question{
		Roll: roll, Opening: p.k.Opening.String(), OpeningPlay: p.play, Score: p.k.Score.String(),
		Position: pos, XGID: "XGID=" + domain.EncodeXGID(&pos), Candidates: p.n,
	}
	if roll == RollSecond {
		q.Reply = p.k.Reply.String()
	}
	return q, nil
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
// empty, else the reply (roll Reply) played after OpeningPlay. The play is
// the board the user left, or its notation when typed; Board wins when both
// are given. Neither is no answer (out of time): the verdict is then the
// correction alone.
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
	// ExactBearoff says whether the exact bearoff table served the line: the
	// signature only states the policy.
	ExactBearoff bool `json:"exact_bearoff"`
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
	k := OpeningKey(o, s)
	if a.OpeningPlay != "" {
		if a.Reply == "" {
			return Verdict{}, fmt.Errorf("openingbook: a reply answer names its reply roll")
		}
		reply, err := ParseRoll(a.Reply)
		if err != nil {
			return Verdict{}, err
		}
		k = ReplyKey(o, a.OpeningPlay, reply, s)
	}
	line, ok := b.Lookup(k)
	if !ok {
		return Verdict{}, fmt.Errorf("openingbook: the book holds no line %s", k)
	}
	return grade(k, line, a)
}

func grade(k Key, line Line, a Answer) (Verdict, error) {
	pos, err := k.Position()
	if err != nil {
		return Verdict{}, err
	}
	legal := domain.LegalMoves(&pos)
	v := Verdict{Played: -1, Ranked: line.Rank(), Signature: line.Signature, Games: line.Games, ExactBearoff: line.ExactBearoff}
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
