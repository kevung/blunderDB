package storage

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// boardVec lays fifteen checkers per side out from point → count maps, the
// rest borne off.
func boardVec(mover, opponent map[int]int) engine.SimilarityVector {
	var v engine.SimilarityVector
	fill := func(dst *[26]int, m map[int]int) {
		n := 0
		for pt, c := range m {
			dst[pt] += c
			n += c
		}
		dst[0] += 15 - n
	}
	fill(&v.Mover, mover)
	fill(&v.Opponent, opponent)
	return v
}

var refOpponent = map[int]int{24: 2, 13: 5, 8: 3, 6: 5}

// shifted is the opening seen from the mover with one checker moved from the
// 13-point to `to`: a family of boards one or a few checker-pips apart.
func shifted(to int) engine.SimilarityVector {
	m := map[int]int{24: 2, 13: 4, 8: 3, 6: 5}
	m[to]++
	return boardVec(m, refOpponent)
}

func refRow(id int64, match int64, theme string, excess float64, v engine.SimilarityVector) ReferenceRow {
	loss, diff := excess, 0.0
	return ReferenceRow{StudyPlanRow: StudyPlanRow{
		RecurringErrorRow: RecurringErrorRow{PositionID: id, GameType: "contact", Kind: "checker", Theme: theme, ErrorMP: 100},
		MatchID:           match, Loss: &loss, Difficulty: &diff}, Vector: v, GapMP: 100}
}

// TestSuggestReferencesCoversClusters pins ADR-0079 §3 and §5: the centre of a
// cluster of errors beats a single larger error, and a cluster yields one
// reference, never two near-identical ones.
func TestSuggestReferencesCoversClusters(t *testing.T) {
	var rows []ReferenceRow
	// Five errors 1 to 3 checker-pips apart: 0.05 together.
	for i, to := range []int{12, 11, 10, 11, 12} {
		rows = append(rows, refRow(int64(10+i), int64(i+1), "blots", 0.01, shifted(to)))
	}
	// One error far away: 0.03 alone.
	rows = append(rows, refRow(99, 9, "blots", 0.03, boardVec(map[int]int{1: 5, 2: 5, 3: 5}, refOpponent)))
	got := SuggestReferences(rows, nil, 1000, 50, 10)
	if len(got.References) != 2 {
		t.Fatalf("got %d references, want 2: %+v", len(got.References), got.References)
	}
	first, second := got.References[0], got.References[1]
	if first.PositionID == 99 || first.Errors != 5 || first.Matches != 5 {
		t.Errorf("first = %+v, want a cluster member standing for 5 errors in 5 matches", first)
	}
	if second.PositionID != 99 || second.Errors != 1 {
		t.Errorf("second = %+v, want the lone position 99", second)
	}
	if first.FamilyErrors != 6 || got.Candidates != 6 {
		t.Errorf("FamilyErrors %d, Candidates %d; want 6, 6", first.FamilyErrors, got.Candidates)
	}
	if !(first.Covered > 0.049 && first.Covered < 0.051) {
		t.Errorf("Covered %v, want 0.05", first.Covered)
	}
}

// TestSuggestReferencesHandled: a handled position is never proposed and
// counts as already picked, so its cluster yields nothing (ADR-0079 §6).
func TestSuggestReferencesHandled(t *testing.T) {
	var rows []ReferenceRow
	for i, to := range []int{12, 11, 10} {
		rows = append(rows, refRow(int64(10+i), 1, "blots", 0.01, shifted(to)))
	}
	rows = append(rows, refRow(99, 2, "blots", 0.03, boardVec(map[int]int{1: 5, 2: 5, 3: 5}, refOpponent)))
	got := SuggestReferences(rows, map[int64]bool{11: true}, 100, 50, 10)
	if len(got.References) != 1 || got.References[0].PositionID != 99 {
		t.Fatalf("references = %+v, want 99 alone", got.References)
	}
	if got.Handled != 1 || got.Candidates != 3 {
		t.Errorf("Handled %d, Candidates %d; want 1, 3", got.Handled, got.Candidates)
	}
}

// TestSuggestReferencesDiscount: a close lesson and an unstable verdict each
// halve the gain, so a clean single error can outrank a murky pair.
func TestSuggestReferencesDiscount(t *testing.T) {
	a := refRow(1, 1, "blots", 0.02, shifted(12))
	a.GapMP = 10 // second best within half the 50 mp threshold: close
	a.Unstable = true
	b := refRow(2, 2, "blots", 0.02, shifted(11))
	c := refRow(3, 3, "gammon", 0.03, boardVec(map[int]int{1: 5, 2: 5, 3: 5}, refOpponent))
	got := SuggestReferences([]ReferenceRow{a, b, c}, nil, 100, 50, 1)
	if len(got.References) != 1 {
		t.Fatalf("got %d references, want 1", len(got.References))
	}
	// The pair covers 0.04: from b, undiscounted; from a, a quarter of it.
	if r := got.References[0]; r.PositionID != 2 || r.Gain < 0.039 {
		t.Errorf("first = %+v, want position 2 with gain 0.04", r)
	}
	got = SuggestReferences([]ReferenceRow{a, c}, nil, 100, 50, 2)
	if r := got.References[0]; r.PositionID != 3 {
		t.Errorf("first = %+v, want position 3: 0.03 beats a quarter of 0.02", r)
	}
	if r := got.References[1]; !r.Close || !r.Unstable || r.Gain > 0.0051 {
		t.Errorf("second = %+v, want a close, unstable reference worth 0.005", r)
	}
}

// TestSuggestReferencesCubeByScore: one cube board at two scores is two
// lessons; at one score, one (ADR-0079 §1).
func TestSuggestReferencesCubeByScore(t *testing.T) {
	cube := func(id int64, a0, a1 int) ReferenceRow {
		r := refRow(id, id, "missed-double", 0.02, shifted(12))
		r.Kind, r.AwayOnRoll, r.AwayOpponent = "cube", a0, a1
		return r
	}
	got := SuggestReferences([]ReferenceRow{cube(1, 3, 5), cube(2, 2, 5), cube(3, 3, 5)}, nil, 100, 50, 10)
	if len(got.References) != 2 {
		t.Fatalf("got %+v, want two references", got.References)
	}
	for _, r := range got.References {
		if r.AwayOnRoll == 3 && r.Errors != 2 {
			t.Errorf("3-away/5-away reference stands for %d errors, want 2", r.Errors)
		}
	}
}

// TestSuggestReferencesSkipsUnpricedAndUnthemed: what the study plan leaves
// out is not a candidate either.
func TestSuggestReferencesSkipsUnpricedAndUnthemed(t *testing.T) {
	unpriced := refRow(1, 1, "blots", 0.02, shifted(12))
	unpriced.Loss = nil
	got := SuggestReferences([]ReferenceRow{unpriced, refRow(2, 1, RecurringThemeNone, 0.02, shifted(11)),
		refRow(3, 1, "blots", -0.01, shifted(5))}, nil, 100, 50, 10)
	if got.Candidates != 1 || len(got.References) != 0 {
		t.Errorf("Candidates %d, references %+v; want 1 and none (nothing to recover)", got.Candidates, got.References)
	}
}

// TestSimilarityKeyMatchesDistance: the prefix-sum form and its pip bound
// agree with engine.SimilarityDistance on every pair.
func TestSimilarityKeyMatchesDistance(t *testing.T) {
	vs := randomGameVectors(rand.New(rand.NewSource(1)), 400)
	for i := range vs {
		ki := newSimilarityKey(&vs[i], "checker", 0, 0)
		for j := range vs {
			kj := newSimilarityKey(&vs[j], "checker", 0, 0)
			for _, radius := range []int{0, 4, 12, 30} {
				want := engine.SimilarityDistance(vs[i], vs[j]) <= radius
				if got := ki.within(&kj, radius); got != want {
					t.Fatalf("pair %d,%d radius %d: within %v, distance %d", i, j, radius, got, engine.SimilarityDistance(vs[i], vs[j]))
				}
			}
		}
	}
}

func TestReferenceLesson(t *testing.T) {
	ana := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "13/7 8/7"}, {Move: "24/18 13/7"}}}}
	if gap, unstable, rolled := ReferenceLesson(ana, "checker", "", []float64{0, 0.042, 0.1}); gap != 42 || unstable || rolled {
		t.Errorf("plain checker: gap %d, unstable %v, rolled %v; want 42, false, false", gap, unstable, rolled)
	}
	ana.Rollouts = []domain.RolloutAnalysis{{Kind: domain.RolloutKindMoves, Candidates: []domain.RolloutCandidate{{Move: "13/7  8/7"}}}}
	if _, unstable, rolled := ReferenceLesson(ana, "checker", "", nil); unstable || !rolled {
		t.Errorf("confirming rollout: unstable %v, rolled %v; want false, true", unstable, rolled)
	}
	ana.Rollouts[0].Candidates[0].Move = "24/18 13/7"
	if _, unstable, _ := ReferenceLesson(ana, "checker", "", nil); !unstable {
		t.Error("a rollout ranking another play first must mark the verdict unstable")
	}

	cube := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "Double, Take"},
		AllCubeAnalyses:      []domain.DoublingCubeAnalysis{{BestCubeAction: "Double, Pass", AnalysisDepth: "4-ply"}},
	}
	// The deeper ply disagrees on the response, not on the double.
	if _, unstable, _ := ReferenceLesson(cube, "cube", "No Double", []float64{0.06, 0}); unstable {
		t.Error("a doubler's verdict is stable when every depth says double")
	}
	if _, unstable, _ := ReferenceLesson(cube, "cube", "Pass", nil); !unstable {
		t.Error("a responder's verdict is unstable when one depth says take and another pass")
	}
	if gap, _, _ := ReferenceLesson(cube, "cube", "No Double", []float64{0.06, 0}); gap != 60 {
		t.Errorf("cube gap %d, want 60", gap)
	}
}

// randomGameVectors plays random games on the similarity vectors alone: a
// checker moves by a die towards home, sides alternate. Positions of one game
// lie close together and the pip counts spread as in real play, which is what
// the band index of ADR-0079 §8 depends on.
func randomGameVectors(rng *rand.Rand, n int) []engine.SimilarityVector {
	opening := [26]int{}
	opening[24], opening[13], opening[8], opening[6] = 2, 5, 3, 5
	out := make([]engine.SimilarityVector, 0, n)
	// One board is one position, as the Zobrist hash makes it in a library.
	seen := map[engine.SimilarityVector]bool{}
	for len(out) < n {
		v := engine.SimilarityVector{Mover: opening, Opponent: opening}
		for ply := 0; ply < 60 && len(out) < n; ply++ {
			for range 2 {
				die := 1 + rng.Intn(6)
				var occupied []int
				for i := 1; i < 26; i++ {
					if v.Mover[i] > 0 {
						occupied = append(occupied, i)
					}
				}
				if len(occupied) == 0 {
					break
				}
				from := occupied[rng.Intn(len(occupied))]
				v.Mover[from]--
				v.Mover[max(from-die, 0)]++
			}
			v.Mover, v.Opponent = v.Opponent, v.Mover
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}

// BenchmarkSuggestReferences measures a proposal over n errors spread over
// some groups — the cost ADR-0079 §8 bounds. Run with -benchtime=1x.
func BenchmarkSuggestReferences(b *testing.B) {
	for _, n := range []int{2000, 10000, 50000} {
		for _, groups := range []int{1, 20} {
			rng := rand.New(rand.NewSource(int64(n)))
			vs := randomGameVectors(rng, n)
			rows := make([]ReferenceRow, n)
			for i := range rows {
				rows[i] = refRow(int64(i+1), int64(i/60), fmt.Sprintf("t%d", rng.Intn(groups)), rng.Float64()*0.02, vs[i])
			}
			b.Run(fmt.Sprintf("n=%d/groups=%d", n, groups), func(b *testing.B) {
				for b.Loop() {
					start := time.Now()
					SuggestReferences(rows, nil, n*10, 50, ReferenceMaxSize)
					b.ReportMetric(float64(time.Since(start).Milliseconds()), "ms")
				}
			})
		}
	}
}
