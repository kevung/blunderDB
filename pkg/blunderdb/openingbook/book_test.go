package openingbook

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestReplyPositions_AllOpeningsResolveAndDiffer(t *testing.T) {
	if len(Openings()) != 15 || len(Replies()) != 21 || len(OpeningPlays) != 15 {
		t.Fatalf("openings %d, replies %d, plays %d; want 15, 21, 15", len(Openings()), len(Replies()), len(OpeningPlays))
	}
	seen := map[string]Key{}
	for _, k := range Jobs([]Score{{4, 2}}) {
		pos, err := ReplyPosition(k.Opening, k.Reply, k.Score)
		if err != nil {
			t.Fatal(err)
		}
		if pos.PlayerOnRoll != domain.White || pos.Score[domain.White] != 4 || pos.Score[domain.Black] != 2 {
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
	if len(seen) != 315 {
		t.Fatalf("%d reply positions, want 315", len(seen))
	}
}

func TestReplyPosition_CrawfordReadsBack(t *testing.T) {
	pos, err := ReplyPosition(Roll{3, 1}, Roll{5, 2}, Score{domain.Crawford, 4})
	if err != nil {
		t.Fatal(err)
	}
	back, err := domain.DecodeXGID(domain.EncodeXGID(&pos))
	if err != nil || back.Score != pos.Score {
		t.Fatalf("Crawford score %v reads back as %v (%v)", pos.Score, back.Score, err)
	}
}

func TestScores(t *testing.T) {
	if n := len(GridScores()); n != 20 {
		t.Fatalf("grid has %d scores, want 20", n)
	}
	if n := len(FullScores()); n != 48 {
		t.Fatalf("full grid has %d scores, want 48", n)
	}
	for _, s := range append(FullScores(), Money) {
		back, err := ParseScore(s.String())
		if err != nil || back != s {
			t.Fatalf("%s reads back as %v (%v)", s, back, err)
		}
	}
	for _, bad := range []string{"1-1", "0-3", "3", "a-b", "-1-3"} {
		if _, err := ParseScore(bad); err == nil {
			t.Fatalf("score %q accepted", bad)
		}
	}
	got, err := ParseScores("4-2, 2-4,4-2,money")
	if err != nil || len(got) != 3 {
		t.Fatalf("list gives %v (%v), want 3 distinct scores", got, err)
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
	if d := r[1].CI95 - 1.96*r[1].SigmaDiff; d != 0 {
		t.Fatalf("CI95 is not 1.96 σ(diff): %g", d)
	}
}

func TestLoad_RefusesDuplicateCompileKeepsLater(t *testing.T) {
	a := `{"opening":"31","reply":"52","score":"4-2","candidates":[{"move":"x","equity":0.1}]}`
	b := `{"opening":"31","reply":"52","score":"4-2","candidates":[{"move":"y","equity":0.2}]}`
	if _, err := Load(strings.NewReader(a + "\n" + b + "\n")); err == nil {
		t.Fatal("Load accepted a key twice")
	}
	book, err := Compile(strings.NewReader(a + "\n" + b + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	l, ok := book.Lookup(Roll{3, 1}, Roll{5, 2}, Score{4, 2})
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

func TestDefault_EmbeddedBookLoads(t *testing.T) {
	if _, err := Default(); err != nil {
		t.Fatal(err)
	}
}
