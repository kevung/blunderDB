// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// levelSolve (cube.go) inverts a level curve in closed form, as gn_cube.c's
// level_solve does. TestClosedFormAgreesWithBisection holds it to the sixty
// bisection steps it replaced, on real levels: the same function, up to a
// converging bisection's tolerance. The benchmarks below time the decision
// paths that pay for it.

// levelSolveBisection is the inversion levelSolve replaced, kept as the
// reference the closed form is held to: sixty bisection steps converging on
// inf{p : f(p) >= target}.
func levelSolveBisection(lv *matchLevel, owner CubeOwner, blend, target float64) float64 {
	low, high := 0.0, 1.0
	for i := 0; i < 60; i++ {
		mid := 0.5 * (low + high)
		value := levelLive(lv, mid, owner)
		if blend >= 0.0 {
			value = (1.0-blend)*levelDead(lv, mid) + blend*value
		}
		if value < target {
			low = mid
		} else {
			high = mid
		}
	}
	return 0.5 * (low + high)
}

// closedFormLevels rend un corpus de niveaux réels : la chaîne complète de
// chaque état de match plausible, sur des mélanges de résultats de la course
// sèche au jeu à fort gammon, résolue par buildLevelAnchors et resolveLevels.
func closedFormLevels(t testing.TB) []struct {
	lv    matchLevel
	label string
} {
	t.Helper()
	mixes := [][numOutcomes]float64{
		{0.50, 0.00, 0.00, 0.50, 0.00, 0.00},
		{0.40, 0.10, 0.01, 0.35, 0.13, 0.01},
		{0.20, 0.25, 0.05, 0.30, 0.18, 0.02},
		{0.62, 0.05, 0.00, 0.30, 0.03, 0.00},
		{0.05, 0.02, 0.00, 0.70, 0.20, 0.03},
		{0.30, 0.30, 0.10, 0.20, 0.08, 0.02},
	}
	var out []struct {
		lv    matchLevel
		label string
	}
	for _, away := range [][2]int{{1, 1}, {1, 2}, {2, 1}, {2, 2}, {3, 5}, {5, 3}, {5, 5}, {7, 4}, {11, 2}, {15, 15}, {2, 25}} {
		for _, cube := range []int{1, 2, 4, 8} {
			for _, crawford := range []bool{false, true} {
				st := MatchState{AwayOnRoll: away[0], AwayOpponent: away[1], Cube: cube, Crawford: crawford}
				if !st.IsValid() {
					continue
				}
				for mi, mix := range mixes {
					var levels [maxCubeLevels]matchLevel
					count := buildLevelAnchors(st, mix, &levels)
					if count == 0 {
						continue
					}
					resolveLevels(&levels, count)
					for i := 0; i < count; i++ {
						out = append(out, struct {
							lv    matchLevel
							label string
						}{levels[i], fmt.Sprintf("%d-away/%d-away cube %d crawford=%v mix#%d niveau %d",
							away[0], away[1], cube, crawford, mi, i)})
					}
				}
			}
		}
	}
	return out
}

// closedFormTargets balaie les cibles d'inversion : les cibles RÉELLES (pass
// et cash du niveau, ce que resolveLevels résout vraiment, et eDP, ce que
// Decide résout) plus un balayage régulier de la plage utile, bornes et
// points de rupture inclus.
func closedFormTargets(lv *matchLevel) []float64 {
	out := []float64{lv.pass, lv.cash, lv.loseAvg, lv.winAvg}
	for k := 0; k <= 20; k++ {
		p := float64(k) / 20.0
		out = append(out, levelDead(lv, p), levelLive(lv, p, CubeCentred))
	}
	return out
}

// TestClosedFormAgreesWithBisection : sur des niveaux réels, la forme close
// rend le même p que soixante pas de bissection, à 1e-9 près (la tolérance
// d'une bissection qui converge).
func TestClosedFormAgreesWithBisection(t *testing.T) {
	corpus := closedFormLevels(t)
	if len(corpus) == 0 {
		t.Fatal("corpus vide")
	}
	owners := []CubeOwner{CubeCentred, CubeOwned, CubeOpponent}
	blends := []float64{-1.0, 0.0, 0.566, 0.687, 0.688, 1.0}

	const tol = 1e-9
	var worst float64
	var worstLabel string
	checked := 0
	for _, c := range corpus {
		lv := c.lv
		for _, owner := range owners {
			for _, blend := range blends {
				for _, target := range closedFormTargets(&lv) {
					got := levelSolve(&lv, owner, blend, target)
					want := levelSolveBisection(&lv, owner, blend, target)
					checked++
					if d := math.Abs(got - want); d > worst {
						worst, worstLabel = d, fmt.Sprintf("%s owner=%v blend=%.3f target=%.9f (close %.12f, bissection %.12f)",
							c.label, owner, blend, target, got, want)
					}
				}
			}
		}
	}
	t.Logf("forme close contre bissection: %d inversions, écart max %.3e (%s)", checked, worst, worstLabel)
	if worst > tol {
		t.Fatalf("la forme close diverge de la bissection: %.3e > %.3e — %s", worst, tol, worstLabel)
	}
}

// ── Repères d'inversion ─────────────────────────────────────────────────────

// benchLevel est le niveau d'enjeu que les repères d'inversion résolvent : la
// chaîne 5-away/5-away, videau à 1, mélange de résultats ordinaire — le même
// score que BenchmarkDecision2PlyMatch.
func benchLevel(b *testing.B) (matchLevel, matchLevel) {
	b.Helper()
	st := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	mix := [numOutcomes]float64{0.40, 0.10, 0.01, 0.35, 0.13, 0.01}
	var levels [maxCubeLevels]matchLevel
	count := buildLevelAnchors(st, mix, &levels)
	if count < 2 {
		b.Fatal("chaîne refusée")
	}
	resolveLevels(&levels, count)
	return levels[0], levels[1]
}

// BenchmarkLevelSolveBisection est une inversion, soixante pas.
func BenchmarkLevelSolveBisection(b *testing.B) {
	cur, next := benchLevel(b)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		sink += levelSolveBisection(&next, CubeOwned, -1.0, cur.pass)
	}
	runtimeSink = sink
}

// BenchmarkLevelSolveClosed est la même inversion en forme close, celle que
// levelSolve fait.
func BenchmarkLevelSolveClosed(b *testing.B) {
	cur, next := benchLevel(b)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		sink += levelSolve(&next, CubeOwned, -1.0, cur.pass)
	}
	runtimeSink = sink
}

// BenchmarkBuildLevels est la chaîne d'enjeux entière — ancres ET points de
// rupture — telle que chaque feuille au score la paye.
func BenchmarkBuildLevels(b *testing.B) {
	st := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	mix := [numOutcomes]float64{0.40, 0.10, 0.01, 0.35, 0.13, 0.01}
	b.ReportAllocs()
	b.ResetTimer()
	var sink int
	for i := 0; i < b.N; i++ {
		_, count := buildLevels(st, mix)
		sink += count
	}
	if sink == 0 {
		b.Fatal("chaîne refusée")
	}
}

// BenchmarkBuildLevelAnchors isole l'autre moitié : les consultations de MET,
// sans une seule bissection. La différence avec BenchmarkBuildLevels est le
// coût des points de rupture, qui est ce que la forme close supprime.
func BenchmarkBuildLevelAnchors(b *testing.B) {
	st := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	mix := [numOutcomes]float64{0.40, 0.10, 0.01, 0.35, 0.13, 0.01}
	b.ReportAllocs()
	b.ResetTimer()
	var levels [maxCubeLevels]matchLevel
	var sink int
	for i := 0; i < b.N; i++ {
		sink += buildLevelAnchors(st, mix, &levels)
	}
	if sink == 0 {
		b.Fatal("chaîne refusée")
	}
}

// BenchmarkCubeDecisionAtScore est Decide au score sur une distribution déjà
// calculée : la chaîne, puis le take point rapporté.
func BenchmarkCubeDecisionAtScore(b *testing.B) {
	probs := benchCubeProbs(b)
	state := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	x := DefaultEfficiency(CubeCentred)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := Decide(&probs, CubeCentred, &state, x, false); !ok {
			b.Fatal("Decide a refusé")
		}
	}
}

// BenchmarkCubeDecisionMoney est son pendant money : la même décision sans
// chaîne d'enjeux ni bissection, forme close de bout en bout (§3). L'écart
// entre les deux repères EST le coût de la récursion au score.
func BenchmarkCubeDecisionMoney(b *testing.B) {
	probs := benchCubeProbs(b)
	x := DefaultEfficiency(CubeCentred)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := Decide(&probs, CubeCentred, nil, x, false); !ok {
			b.Fatal("Decide a refusé")
		}
	}
}

// benchCubeProbs est une distribution 0-ply réelle de la position d'ouverture,
// plutôt qu'un vecteur inventé : les six sorties d'un réseau ne sont pas six
// nombres arbitraires, et les points de rupture dépendent de leur mélange.
func benchCubeProbs(b *testing.B) [NumOutputs]float32 {
	b.Helper()
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		b.Fatal(err)
	}
	dp.PlayerOnRoll = domain.White
	p, err := FromDomain(&dp)
	if err != nil {
		b.Fatal(err)
	}
	s, err := NewSearcher(DefaultConfig(0))
	if err != nil {
		b.Fatal(err)
	}
	probs, ok := s.Probs(&p)
	if !ok {
		b.Fatal("Probs a refusé")
	}
	return probs
}

// runtimeSink empêche le compilateur d'éliminer le corps des repères
// d'inversion, dont le résultat n'est autrement lu par personne.
var runtimeSink float64

// BenchmarkAnalysisBatchThroughput est le débit du lot d'analyse : positions
// réelles au score, une goroutine par cœur, chacune avec son chercheur
// réutilisé (db_gammonnet_batch.go). La métrique est pos/s.
func BenchmarkAnalysisBatchThroughput(b *testing.B) {
	for _, ply := range []int{0, 2} {
		b.Run(fmt.Sprintf("%d-ply", ply), func(b *testing.B) {
			if ply > 0 && testing.Short() {
				b.Skip("une position 2-ply coûte des centaines de millisecondes")
			}
			positions := batchBenchPositions(b)
			workers := runtime.NumCPU()
			// Chercheurs bâtis hors chrono, un par goroutine comme le lot
			// réel (NewBatchSearcher).
			searchers := make([]*Searcher, workers)
			for w := range searchers {
				s, err := NewBatchSearcher(ply, 0)
				if err != nil {
					b.Fatalf("NewBatchSearcher: %v", err)
				}
				searchers[w] = s
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var next atomic.Int64
				var wg sync.WaitGroup
				for w := 0; w < workers; w++ {
					wg.Add(1)
					go func(s *Searcher) {
						defer wg.Done()
						for {
							j := int(next.Add(1)) - 1
							if j >= len(positions) {
								return
							}
							_, _ = EvaluatePositionWith(s, positions[j], ply, 0, 3)
						}
					}(searchers[w])
				}
				wg.Wait()
			}
			b.StopTimer()
			elapsed := b.Elapsed().Seconds()
			if elapsed > 0 {
				b.ReportMetric(float64(b.N*len(positions))/elapsed, "pos/s")
			}
		})
	}
}

// batchBenchPositions est un échantillon de positions RÉELLES au score, tiré
// des mêmes fixtures que la porte d'intégration : un débit mesuré sur des
// plateaux inventés mesurerait la génération de coups d'un autre jeu.
func batchBenchPositions(b *testing.B) []domain.Position {
	b.Helper()
	mg, err := ingest.MapGnuBG("../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf")
	if err != nil {
		b.Fatalf("MapGnuBG: %v", err)
	}
	var out []domain.Position
	for _, g := range mg.Games {
		for _, mv := range g.Moves {
			if mv.Position == nil || mv.Move.MoveType != "checker" {
				continue
			}
			p := *mv.Position
			if p.Dice[0] < 1 || p.Dice[0] > 6 || p.Dice[1] < 1 || p.Dice[1] > 6 {
				continue
			}
			out = append(out, p)
			if len(out) == 32 {
				return out
			}
		}
	}
	if len(out) == 0 {
		b.Fatal("aucune position exploitable dans la fixture")
	}
	return out
}

// The inversion's conventions, as gammonNet's level_solve_probe: a target at
// or under f(0) answers 0, one above f(1) answers 1, a flat piece answers its
// left bound, and a NaN target or curve answers NaN rather than a plausible 1.
func TestLevelSolveConventions(t *testing.T) {
	lv := matchLevel{loseAvg: 0.2, winAvg: 0.9, pass: 0.4, cash: 0.7, tp: 0.3, cp: 0.6}
	if got := levelSolve(&lv, CubeOwned, -1.0, 0.1); got != 0 {
		t.Errorf("under f(0): %v, want 0", got)
	}
	if got := levelSolve(&lv, CubeOwned, -1.0, 0.95); got != 1 {
		t.Errorf("above f(1): %v, want 1", got)
	}
	if got := levelSolve(&lv, CubeOwned, -1.0, 0.7); got != lv.cp {
		t.Errorf("at the breakpoint: %v, want %v", got, lv.cp)
	}
	flat := matchLevel{loseAvg: 0.5, winAvg: 0.9, cash: 0.5, cp: 0.4}
	if got := levelSolve(&flat, CubeOwned, -1.0, 0.5); got != 0 {
		t.Errorf("flat piece: %v, want its left bound 0", got)
	}
	if got := levelSolve(&lv, CubeOwned, -1.0, math.NaN()); !math.IsNaN(got) {
		t.Errorf("NaN target: %v, want NaN", got)
	}
	bad := lv
	bad.cash = math.NaN()
	if got := levelSolve(&bad, CubeOwned, -1.0, 0.5); !math.IsNaN(got) {
		t.Errorf("NaN curve: %v, want NaN", got)
	}
}
