package storage

import (
	"math"
	"testing"
)

func TestPressureOf(t *testing.T) {
	for _, c := range []struct {
		away [2]int
		want string
	}{
		{[2]int{0, 0}, PressureDMP},
		{[2]int{1, 1}, PressureDMP}, // a one-point match: both need one point
		{[2]int{1, 4}, PressureCrawford},
		{[2]int{5, 1}, PressureCrawford},
		{[2]int{0, 3}, PressurePostCrawford},
		{[2]int{2, 0}, PressurePostCrawford},
		{[2]int{2, 2}, PressureOther},
		{[2]int{7, 3}, PressureOther},
	} {
		if got := PressureOf(c.away); got != c.want {
			t.Errorf("PressureOf(%v) = %s, want %s", c.away, got, c.want)
		}
	}
}

func TestCompare(t *testing.T) {
	// 10 ± 2 against 5 ± 1 (half-widths at 95 %): Δ = 5, half √5 ≈ 2.24.
	c := Compare(10, 12, true, 40, 5, 6, true, 100)
	if c.Verdict != VerdictWorse || math.Abs(c.Delta-5) > 1e-12 || math.Abs(c.Low-(5-math.Sqrt(5))) > 1e-9 {
		t.Errorf("a clear gap is worse: %+v", c)
	}
	if c := Compare(5, 7, true, 40, 10, 12, true, 100); c.Verdict != VerdictBetter {
		t.Errorf("a clear gap down is better: %+v", c)
	}
	if c := Compare(6, 9, true, 40, 5, 6, true, 100); c.Verdict != VerdictUsual {
		t.Errorf("an interval across zero is the usual level: %+v", c)
	}
	for _, c := range []Comparison{
		Compare(10, 12, false, 40, 5, 6, true, 100),
		Compare(10, 12, true, 40, 5, 6, false, 100),
		Compare(10, 12, true, TournamentReviewMinDecisions-1, 5, 6, true, 100),
		Compare(10, 12, true, 40, 5, 6, true, TournamentReviewMinDecisions-1),
	} {
		if c.Verdict != VerdictInsufficient || c.Delta != 0 {
			t.Errorf("no verdict without an interval and enough decisions: %+v", c)
		}
	}
}

func TestRankKeys(t *testing.T) {
	want := []string{"1-30", "31-60", "61-90", "91+"}
	got := RankKeys()
	if len(got) != len(want) {
		t.Fatalf("RankKeys() = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("RankKeys()[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

// reviewDecisions builds n counted decisions of seat in one game each of
// errMP, alternating with a decision of the other seat, durations from dur.
func reviewDecisions(seat, n int, errMP func(i int) int64, dur func(i int) *int64, away [2]int) []DecisionLoss {
	var out []DecisionLoss
	for i := range n {
		e := errMP(i)
		loss := float64(e) / 10000
		out = append(out, DecisionLoss{GameNumber: i/10 + 1, MoveNumber: 2 * i, Player: seat, DecisionType: "checker",
			ErrorMP: &e, MWCLoss: &loss, DurationMS: dur(i), Away: away})
		zero := int64(0)
		zl := 0.0
		out = append(out, DecisionLoss{GameNumber: i/10 + 1, MoveNumber: 2*i + 1, Player: 1 - seat, DecisionType: "checker",
			ErrorMP: &zero, MWCLoss: &zl, Away: away})
	}
	return out
}

func TestBuildTournamentReview(t *testing.T) {
	noTime := func(int) *int64 { return nil }
	// The tournament: three matches of 40 decisions at 20 mP, every decision
	// past the 30th at 80 mP (fatigue), timed: odd decisions quick.
	timed := func(i int) *int64 { d := int64(1000 + 1000*(i%2)); return &d }
	var matches []ReviewMatch
	for m := range 3 {
		errMP := func(i int) int64 {
			if i >= 30 {
				return 80 + int64(m)
			}
			return 20 + int64(m)
		}
		matches = append(matches, ReviewMatch{MatchID: int64(m + 1), Opponent: "Opp", Length: 7, Seat: m % 2,
			Decisions: reviewDecisions(m%2, 40, errMP, timed, [2]int{4, 4})})
	}
	// The usual level: six matches of 40 decisions at 18–23 mP.
	var usual []ReviewMatch
	for m := range 6 {
		errMP := func(int) int64 { return 18 + int64(m) }
		usual = append(usual, ReviewMatch{MatchID: int64(100 + m), Length: 7,
			Decisions: reviewDecisions(0, 40, errMP, noTime, [2]int{4, 4})})
	}
	plan := &StudyPlan{Families: []StudyPlanFamily{{Theme: "a"}, {Theme: "b"}, {Theme: "c"}, {Theme: "d"}},
		Tentative: []StudyPlanFamily{{Theme: "e"}}}
	r := BuildTournamentReview(TournamentReview{Usual: UsualLevel{From: "2024-01-01", To: "2025-01-01"}}, matches, usual, plan)

	if r.Matches != 3 || r.Decisions != 120 || len(r.Rounds) != 3 {
		t.Fatalf("three rounds of 40 decisions: %+v", r)
	}
	if !r.Usual.Available || r.Usual.Matches != 6 || r.Usual.Decisions != 240 {
		t.Errorf("usual level of six matches: %+v", r.Usual)
	}
	if !r.MWC7.Available || !r.Usual.MWC7.Available {
		t.Errorf("both sides have an L7: %+v / %+v", r.MWC7, r.Usual.MWC7)
	}
	if r.ByRank[1].Decisions != 30 || r.ByRank[0].Decisions != 90 {
		t.Errorf("ranks 1–30 and 31–60: %+v", r.ByRank)
	}
	if r.ByRank[1].Versus.Verdict != VerdictWorse {
		t.Errorf("the late decisions cost far more than usual: %+v", r.ByRank[1])
	}
	if r.ByRank[1].UsualDecisions != 60 {
		t.Errorf("the usual level is cut the same way: %+v", r.ByRank[1])
	}
	if r.ByClock[0].Decisions != 60 || r.ByClock[1].Decisions != 60 {
		t.Errorf("half quick, half considered: %+v", r.ByClock)
	}
	if r.ByClock[0].Versus.Verdict != VerdictInsufficient {
		t.Errorf("the usual level has no time, so no verdict: %+v", r.ByClock[0])
	}
	if r.Rounds[1].Round != 2 || r.Rounds[1].Decisions != 40 {
		t.Errorf("round 2 is the second match, its player seated second: %+v", r.Rounds[1])
	}
	if len(r.Families) != TournamentFamilies || r.Families[0].Theme != "a" || r.Tentative != 1 {
		t.Errorf("at most three families, in the plan's order: %+v, tentative %d", r.Families, r.Tentative)
	}

	short := BuildTournamentReview(TournamentReview{Usual: UsualLevel{From: "2024-01-01"}}, matches, usual[:TournamentUsualMinMatches-1], nil)
	if short.Usual.Available || short.PRVersus.Verdict != VerdictInsufficient || short.ByRank[1].Versus.Verdict != VerdictInsufficient {
		t.Errorf("under five usual matches there is no usual level and no verdict: %+v", short.Usual)
	}
	if short.Families == nil || short.Rounds == nil {
		t.Errorf("lists stay lists")
	}
	undated := BuildTournamentReview(TournamentReview{}, matches, usual, nil)
	if undated.Usual.Available {
		t.Errorf("an undated tournament has no usual window")
	}
}
