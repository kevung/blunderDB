package openingbook

import (
	"math"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

func TestPositions_OpeningsAndRepliesResolveAndDiffer(t *testing.T) {
	if len(Openings()) != 15 || len(Replies()) != 21 {
		t.Fatalf("openings %d, replies %d; want 15, 21", len(Openings()), len(Replies()))
	}
	seen := map[string]Key{}
	add := func(k Key, onRoll int) {
		t.Helper()
		pos, err := k.Position()
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if pos.PlayerOnRoll != onRoll || pos.Score[domain.White] != 4 || pos.Score[domain.Black] != 2 {
			t.Fatalf("%s: on roll %d, score %v", k, pos.PlayerOnRoll, pos.Score)
		}
		x := domain.EncodeXGID(&pos)
		if prev, dup := seen[x]; dup {
			t.Fatalf("%s and %s give the same position", prev, k)
		}
		seen[x] = k
		back, err := domain.DecodeXGID(x)
		if err != nil || back.Score != pos.Score || back.PlayerOnRoll != pos.PlayerOnRoll {
			t.Fatalf("%s: XGID %s reads back as %v (%v)", k, x, back.Score, err)
		}
	}
	s := []Score{{4, 2}}
	for _, k := range OpeningJobs(s, Filter{}) {
		add(k, domain.Black)
	}
	// The reply follows whatever play the opening line ranks first: here the
	// first legal play of each roll stands in for it. Two openings may
	// transpose, so the 21 replies differ within an opening only.
	n := 0
	for _, o := range Openings() {
		n += len(seen)
		seen = map[string]Key{}
		pos, _ := OpeningPosition(o, s[0])
		play := domain.LegalMoves(&pos)[0].Notation
		for _, r := range Replies() {
			add(ReplyKey(o, play, r, s[0]), domain.White)
		}
	}
	if n += len(seen); n != 15+315 {
		t.Fatalf("%d positions, want 330", n)
	}
	if _, err := ReplyPosition(Roll{3, 1}, "24/12", Roll{2, 1}, s[0]); err == nil {
		t.Fatal("an illegal opening play must be refused")
	}
}

func TestReplyPosition_CrawfordReadsBack(t *testing.T) {
	pos, err := ReplyPosition(Roll{3, 1}, "8/5 6/5", Roll{5, 2}, Score{domain.Crawford, 4})
	if err != nil {
		t.Fatal(err)
	}
	back, err := domain.DecodeXGID(domain.EncodeXGID(&pos))
	if err != nil || back.Score != pos.Score {
		t.Fatalf("Crawford score %v reads back as %v (%v)", pos.Score, back.Score, err)
	}
}

// TestReplyPosition_DMPReadsBack: a DMP position is post-Crawford on both
// sides; its XGID (a 2-point match at 1-1, Crawford flag 0) reads back as the
// same sentinel, never as a Crawford game.
func TestReplyPosition_DMPReadsBack(t *testing.T) {
	pos, err := ReplyPosition(Roll{3, 1}, "8/5 6/5", Roll{5, 2}, DMP)
	if err != nil {
		t.Fatal(err)
	}
	if pos.Score != [2]int{domain.PostCrawford, domain.PostCrawford} {
		t.Fatalf("DMP score %v, want both post-Crawford", pos.Score)
	}
	x := domain.EncodeXGID(&pos)
	if !strings.Contains(x, ":1:1:0:2:") {
		t.Fatalf("DMP XGID %s, want score 1-1 of a 2-point match, Crawford 0", x)
	}
	back, err := domain.DecodeXGID(x)
	if err != nil || back.Score != pos.Score {
		t.Fatalf("DMP score %v reads back as %v (%v)", pos.Score, back.Score, err)
	}
}

// TestDMPEquityIsGammonless: at DMP the cube and gammons are worth nothing,
// so the normalised equity of every play is 2·P(win) − 1. Money play, where
// gammons count, is the control that the identity is not trivial.
func TestDMPEquityIsGammonless(t *testing.T) {
	eval := func(s Score) []domain.CheckerMove {
		t.Helper()
		pos, err := ReplyPosition(Roll{3, 1}, "8/5 6/5", Roll{6, 4}, s)
		if err != nil {
			t.Fatal(err)
		}
		r, err := gammonnet.EvaluatePosition(pos, 0, 0, 0)
		if err != nil || len(r.Moves) == 0 {
			t.Fatalf("%s: %v (%d moves)", s, err, len(r.Moves))
		}
		return r.Moves
	}
	for _, m := range eval(DMP) {
		if m.PlayerGammonChance <= 0 {
			t.Fatalf("%s: no gammon chance, the check proves nothing", m.Move)
		}
		if want := 2*m.PlayerWinChance/100 - 1; math.Abs(m.Equity-want) > 1e-4 {
			t.Fatalf("DMP %s: equity %+.5f, want 2·P(win)−1 = %+.5f", m.Move, m.Equity, want)
		}
	}
	m := eval(Money)[0]
	if math.Abs(m.Equity-(2*m.PlayerWinChance/100-1)) < 0.01 {
		t.Fatalf("money %s: equity %+.5f is gammonless too, the control fails", m.Move, m.Equity)
	}
}

func TestScores(t *testing.T) {
	if n := len(GridScores()); n != 20 {
		t.Fatalf("grid has %d scores, want 20", n)
	}
	if n := len(FullScores()); n != 48 {
		t.Fatalf("full grid has %d scores, want 48", n)
	}
	for _, s := range append(FullScores(), Money, DMP) {
		back, err := ParseScore(s.String())
		if err != nil || back != s {
			t.Fatalf("%s reads back as %v (%v)", s, back, err)
		}
	}
	for _, bad := range []string{"0-3", "0-0", "1-0", "3", "a-b", "-1-3"} {
		if _, err := ParseScore(bad); err == nil {
			t.Fatalf("score %q accepted", bad)
		}
	}
	got, err := ParseScores("4-2, 2-4,4-2,money")
	if err != nil || len(got) != 3 {
		t.Fatalf("list gives %v (%v), want 3 distinct scores", got, err)
	}
	if err := (Score{domain.Crawford, domain.Crawford}).Valid(); err == nil {
		t.Fatal("two Crawford sides accepted: 1-1 is DMP, post-Crawford")
	}
	xg, err := ParseScores("xg")
	if err != nil || len(xg) != 4 || xg[0] != Money || xg[1] != DMP || xg[2].String() != "2-1" || xg[3].String() != "1-2" {
		t.Fatalf("xg scores %v (%v), want money, 1-1, 2-1, 1-2", xg, err)
	}
}

func TestRank_TieRule(t *testing.T) {
	l := Line{Candidates: []Candidate{
		{Move: "c", Equity: 0.080, StdErr: 0.002},
		{Move: "a", Equity: 0.100, StdErr: 0.003},
		{Move: "b", Equity: 0.092, StdErr: 0.002},
	}}
	r := l.Rank()
	if r[0].Move != "a" || !r[0].Tied || r[0].Gap != 0 {
		t.Fatalf("best %+v", r[0])
	}
	// b: gap 0.008, σ(diff) = √(9+4)·1e-3 ≈ 0.00361, threshold ≈ 0.01207.
	if r[1].Move != "b" || !r[1].Tied {
		t.Fatalf("b should be level with the best: %+v", r[1])
	}
	// c: gap 0.020 over the same threshold.
	if r[2].Move != "c" || r[2].Tied {
		t.Fatalf("c should trail: %+v", r[2])
	}
	// A tolerance, not equality: arm64 fuses the multiply-add and moves the last bit.
	if d := r[1].CI95 - 1.96*r[1].SigmaDiff; math.Abs(d) > 1e-12 {
		t.Fatalf("CI95 is not 1.96 σ(diff): %g", d)
	}
}

func TestLoad_RefusesDuplicateCompileKeepsLater(t *testing.T) {
	a := `{"opening":"31","opening_play":"8/5 6/5","reply":"52","score":"4-2","candidates":[{"move":"x","equity":0.1}]}`
	b := `{"opening":"31","opening_play":"8/5 6/5","reply":"52","score":"4-2","candidates":[{"move":"y","equity":0.2}]}`
	if _, err := Load(strings.NewReader(a + "\n" + b + "\n")); err == nil {
		t.Fatal("Load accepted a key twice")
	}
	book, err := Compile(strings.NewReader(a + "\n" + b + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	l, ok := book.Lookup(ReplyKey(Roll{3, 1}, "8/5 6/5", Roll{5, 2}, Score{4, 2}))
	if !ok || l.Candidates[0].Move != "y" {
		t.Fatalf("lookup %v %v, want the later line", l, ok)
	}
	var sb strings.Builder
	if err := book.Write(&sb); err != nil {
		t.Fatal(err)
	}
	again, err := Load(strings.NewReader(sb.String()))
	if err != nil || again.Len() != 1 {
		t.Fatalf("written book reads back as %d lines (%v)", again.Len(), err)
	}
}

func TestKey_OpeningAndReplyLinesReadBack(t *testing.T) {
	bad := []string{
		`{"opening":"31","opening_play":"8/5 6/5","score":"4-2"}`, // an opening line names no play
		`{"opening":"31","reply":"52","score":"4-2"}`,             // a reply follows a play
	}
	for _, l := range bad {
		if _, err := Load(strings.NewReader(l)); err == nil {
			t.Errorf("Load accepted %s", l)
		}
	}
	b, err := Load(strings.NewReader(`{"opening":"31","score":"4-2"}` + "\n" +
		`{"opening":"31","opening_play":"8/5 6/5","reply":"52","score":"4-2"}` + "\n" +
		`{"opening":"31","opening_play":"24/23 13/10","reply":"52","score":"4-2"}`))
	if err != nil || b.Len() != 3 {
		t.Fatalf("%v, %d lines", err, b.Len())
	}
	if _, ok := b.Lookup(OpeningKey(Roll{3, 1}, Score{4, 2})); !ok {
		t.Fatal("opening line not found")
	}
}

func TestDefault_EmbeddedBookLoads(t *testing.T) {
	if _, err := Default(); err != nil {
		t.Fatal(err)
	}
}

// The embedded book is what the exercise draws from: every opening at every
// grid score, and every reply after its best play, so that no question can
// reach a line the book lacks.
func TestDefault_EmbeddedBookCoversTheGrid(t *testing.T) {
	b, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range append(GridScores(), XGScores()...) {
		for _, o := range Openings() {
			line, ok := b.Lookup(OpeningKey(o, s))
			if !ok || len(line.Candidates) == 0 {
				t.Errorf("no opening line for %s at %s", o, s)
				continue
			}
			best := line.Candidates[0].Move
			for _, r := range Replies() {
				if _, ok := b.Lookup(ReplyKey(o, best, r, s)); !ok {
					t.Errorf("no reply %s after %s %s at %s", r, o, best, s)
				}
			}
		}
	}
}

// A drill at money or at DMP asks both rolls from the embedded book and
// grades the book's own best play as right.
func TestDefault_DrillsMoneyAndDMP(t *testing.T) {
	b, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"money", "1-1"} {
		s, err := ParseScore(key)
		if err != nil {
			t.Fatal(err)
		}
		for _, roll := range []string{RollFirst, RollSecond} {
			q, err := b.Draw(QuestionRequest{Roll: roll, Scores: []string{key}, Seed: 3})
			if err != nil || q.Empty || q.Roll != roll || q.Score != key {
				t.Fatalf("%s %s: %+v, %v", key, roll, q, err)
			}
			o, err := ParseRoll(q.Opening)
			if err != nil {
				t.Fatal(err)
			}
			line, ok := b.Lookup(OpeningKey(o, s))
			if roll == RollSecond {
				r, _ := ParseRoll(q.Reply)
				line, ok = b.Lookup(ReplyKey(o, q.OpeningPlay, r, s))
			}
			if !ok || len(line.Candidates) == 0 {
				t.Fatalf("%s %s: no line behind %+v", key, roll, q)
			}
			v, err := b.Grade(Answer{Opening: q.Opening, OpeningPlay: q.OpeningPlay, Reply: q.Reply, Score: key, Notation: line.Candidates[0].Move})
			if err != nil || !v.Legal || !v.Right || v.Played != 0 {
				t.Fatalf("%s %s: %+v, %v", key, roll, v, err)
			}
		}
	}
}
