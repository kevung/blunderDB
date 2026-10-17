package openingbook

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// testdata/book.jsonl holds three questions — 31 then 21 at 4-2 and at 2-4,
// 65 then 64 at 4-2 — with made-up numbers: at 4-2 the opening 31 has two
// tied plays and the book follows both with the reply 21.
func testBook(t *testing.T) *Book {
	t.Helper()
	b, err := Open("testdata/book.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestStatus_EmptyAndTestBook(t *testing.T) {
	if s := (&Book{}).Status(); s.Lines != 0 || len(s.Scores) != 0 {
		t.Fatalf("empty book status = %+v", s)
	}
	s := testBook(t).Status()
	if s.Lines != 7 || len(s.Scores) != 2 || s.Scores[0] != "2-4" || s.Scores[1] != "4-2" {
		t.Fatalf("status = %+v", s)
	}
}

func TestDraw_EmptyIsNotAnError(t *testing.T) {
	q, err := (&Book{}).Draw(QuestionRequest{})
	if err != nil || !q.Empty {
		t.Fatalf("empty book: %+v, %v", q, err)
	}
	q, err = testBook(t).Draw(QuestionRequest{Scores: []string{"5-5"}})
	if err != nil || !q.Empty {
		t.Fatalf("score the book lacks: %+v, %v", q, err)
	}
	if _, err := testBook(t).Draw(QuestionRequest{Scores: []string{"0-0"}}); err == nil {
		t.Fatal("an invalid score must be refused")
	}
}

func TestDraw_FirstRollAsksTheOpening(t *testing.T) {
	b := testBook(t)
	seen := map[string]bool{}
	for seed := uint64(1); seed <= 40; seed++ {
		q, err := b.Draw(QuestionRequest{Seed: seed})
		if err != nil || q.Empty || q.Roll != RollFirst || q.Reply != "" || q.OpeningPlay != "" {
			t.Fatalf("draw: %+v, %v", q, err)
		}
		seen[q.Opening+"@"+q.Score] = true
		if q.Position.PlayerOnRoll != domain.Black || q.Position.Dice[0] != int(q.Opening[0]-'0') {
			t.Fatalf("question position: %+v", q.Position)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("drew %v, want the three openings", seen)
	}
	q, _ := b.Draw(QuestionRequest{Roll: RollFirst, Openings: []string{"56"}})
	if q.Opening != "65" || q.Candidates == 0 {
		t.Fatalf("filtered draw: %+v", q)
	}
	a, _ := b.Draw(QuestionRequest{Seed: 7})
	c, _ := b.Draw(QuestionRequest{Seed: 7})
	if a != c {
		t.Fatal("a seed must make the draw reproducible")
	}
	if _, err := b.Draw(QuestionRequest{Roll: "third"}); err == nil {
		t.Fatal("an unknown roll must be refused")
	}
}

func TestDraw_SecondRollAsksTheReplyAfterTheBest(t *testing.T) {
	b := testBook(t)
	seen := map[string]bool{}
	for seed := uint64(1); seed <= 40; seed++ {
		q, err := b.Draw(QuestionRequest{Roll: RollSecond, Seed: seed})
		if err != nil || q.Empty || q.Roll != RollSecond || q.Reply == "" {
			t.Fatalf("draw: %+v, %v", q, err)
		}
		seen[q.Opening+"/"+q.OpeningPlay+"/"+q.Reply+"@"+q.Score] = true
		if q.Position.PlayerOnRoll != domain.White || q.Position.Dice[0] != int(q.Reply[0]-'0') {
			t.Fatalf("question position: %+v", q.Position)
		}
		// The question is answered as a reply after its opening play.
		if _, err := b.Grade(Answer{Opening: q.Opening, OpeningPlay: q.OpeningPlay, Reply: q.Reply, Score: q.Score}); err != nil {
			t.Fatalf("%+v cannot be graded: %v", q, err)
		}
	}
	// The book's best opening play only: never the tied 24/23 13/10 it also follows.
	want := map[string]bool{"31/8/5 6/5/21@4-2": true, "31/8/5 6/5/21@2-4": true, "65/24/18 18/13/64@4-2": true}
	if len(seen) != len(want) {
		t.Fatalf("drew %v, want %v", seen, want)
	}
	for k := range seen {
		if !want[k] {
			t.Fatalf("drew %v, want %v", seen, want)
		}
	}
	if q, _ := b.Draw(QuestionRequest{Roll: RollSecond, Replies: []string{"66"}}); !q.Empty {
		t.Fatalf("a reply the book lacks: %+v", q)
	}
}

func TestGrade_OpeningAndReply(t *testing.T) {
	b := testBook(t)
	cases := []struct {
		score, play string
		right       bool
		played      int
	}{
		{"4-2", "8/5 6/5", true, 0},
		{"4-2", "24/23 13/10", true, 1}, // tied with the best
		{"4-2", "13/10 6/5", false, 2},
		{"4-2", "24/21 24/23", false, -1}, // legal, outside the candidates
		{"2-4", "24/23 13/10", false, 1},  // the same play at another score
	}
	for _, c := range cases {
		v, err := b.Grade(Answer{Opening: "31", Score: c.score, Notation: c.play})
		if err != nil {
			t.Fatal(err)
		}
		if !v.Legal || v.Right != c.right || v.Played != c.played {
			t.Fatalf("%s at %s: %+v", c.play, c.score, v)
		}
	}
	v, err := b.Grade(Answer{Opening: "31", OpeningPlay: "24/23 13/10", Reply: "21", Score: "4-2", Notation: "13/11 6/5"})
	if err != nil || !v.Right || v.Played != 1 {
		t.Fatalf("reply after the tied opening play: %+v, %v", v, err)
	}
	v, err = b.Grade(Answer{Opening: "31", OpeningPlay: "8/5 6/5", Reply: "21", Score: "4-2", Notation: "13/11 8/7"})
	if err != nil || v.Right || v.Played != 2 {
		t.Fatalf("wrong reply: %+v, %v", v, err)
	}
}

func TestGrade_ByBoardAndIllegal(t *testing.T) {
	b := testBook(t)
	pos, _ := ReplyPosition(Roll{6, 5}, "24/18 18/13", Roll{6, 4}, Score{4, 2})
	p, err := engine.ResolvePlay(&pos, domain.LegalMoves(&pos), "24/18 18/14", false)
	if err != nil {
		t.Fatal(err)
	}
	a := Answer{Opening: "65", OpeningPlay: "24/18 18/13", Reply: "64", Score: "4-2", Board: &p.Result.Board}
	v, err := b.Grade(a)
	if err != nil || !v.Legal || !v.Right || v.Played != 1 || v.Notation == "" {
		t.Fatalf("board answer: %+v, %v", v, err)
	}
	a.Board, a.Notation = nil, "24/12"
	v, err = b.Grade(a)
	if err != nil || v.Legal || v.Right || len(v.Ranked) != 3 {
		t.Fatalf("illegal answer: %+v, %v", v, err)
	}
	if _, err := b.Grade(Answer{Opening: "65", Reply: "11", Score: "5-5", Notation: "8/7"}); err == nil {
		t.Fatal("a line the book lacks must be refused")
	}
}
