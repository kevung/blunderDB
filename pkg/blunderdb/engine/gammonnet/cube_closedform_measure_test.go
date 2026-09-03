// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// L'instrument de mesure de #194/C.7 — « la bissection de levelSolve
// peut-elle devenir une forme close, et à quel prix ? »
//
// LA QUESTION. `levelSolve` (cube.go) inverse une fonction AFFINE PAR
// MORCEAUX ET MONOTONE dont les deux ou trois segments sont connus d'avance
// (`levelLive`) — et il l'inverse par soixante pas de bissection. Le rapport
// docs/recherche/P6-videau-janowski.md le dit : l'inversion d'une telle
// fonction est en forme close, O(nombre de segments), exacte. La bissection
// n'est pas une approximation nécessaire, c'est une recherche numérique là où
// une division suffit.
//
// CE QUE ÇA PÈSE, remesuré le 2026-09-03 APRÈS C.8/C.9/C.10 (le profil de la
// fiche datait d'avant) — Ryzen 7 PRO 6850U, 16 cœurs, Go 1.25.13, décision
// 2-ply k=12 canonique au score 5-away/5-away :
//
//	BenchmarkDecision2Ply       (money) 193 ms
//	BenchmarkDecision2PlyMatch  (score) 306 ms
//
// et le profil de la seconde : levelSolve 35,4 % cumulé (dont laneCurve.at
// 9,3 %), buildLevels 38,7 %, Value 39,8 %, EvaluateBatch 52,6 %. Le videau
// est donc bien le second poste, et la bissection est PRESQUE TOUT le videau
// — l'écart money↔score, 113 ms, est du même ordre que les 35 % que la
// bissection prend.
//
// POURQUOI RIEN N'EST RAPIÉCÉ ICI. Le gain survit au changement de langage
// (soixante itérations d'une chaîne sérielle contre une division : c'est la
// forme de l'algorithme, pas son écriture), donc l'ADR-0003 de gammonNet et
// l'invariant de CLAUDE.md le rangent en amont. Et il n'est PAS bit-identique
// — c'est ce que ce fichier mesure. Ce qui est livré ici est l'instrument,
// le chiffre et le correctif proposé ; l'ADR « la forme close de l'inversion
// du videau se décide en amont » porte la décision.
//
// TestClosedFormAgreesWithBisection tourne TOUJOURS : il est la garantie que
// la forme close proposée en amont est la bonne fonction. Les mesures d'écart
// et de gain sont derrière BLUNDERDB_MEASURE_CLOSEDFORM.

// levelSegments écrit les segments de la courbe vive d'un niveau, dans
// l'ordre des p croissants, EXACTEMENT comme levelLive les choisit : mêmes
// bornes, mêmes ordonnées, même convention de segment dégénéré (`segment`
// rend y1 quand x1 <= x0).
//
// Rend le nombre de segments écrits. C'est la seule chose que la forme close
// a besoin de savoir de la courbe, et c'est aussi la seule chose qu'il faut
// relire pour vérifier qu'elle décrit la même courbe.
func levelSegments(lv *matchLevel, owner CubeOwner, segs *[3][4]float64) int {
	if lv.dead {
		segs[0] = [4]float64{0.0, lv.loseAvg, 1.0, lv.winAvg}
		return 1
	}
	switch owner {
	case CubeOwned:
		segs[0] = [4]float64{0.0, lv.loseAvg, lv.cp, lv.cash}
		segs[1] = [4]float64{lv.cp, lv.cash, 1.0, lv.winAvg}
		return 2
	case CubeOpponent:
		segs[0] = [4]float64{0.0, lv.loseAvg, lv.tp, lv.pass}
		segs[1] = [4]float64{lv.tp, lv.pass, 1.0, lv.winAvg}
		return 2
	default: // CubeCentred
		segs[0] = [4]float64{0.0, lv.loseAvg, lv.tp, lv.pass}
		segs[1] = [4]float64{lv.tp, lv.pass, lv.cp, lv.cash}
		segs[2] = [4]float64{lv.cp, lv.cash, 1.0, lv.winAvg}
		return 3
	}
}

// levelSolveClosed est levelSolve SANS bissection : le segment qui contient
// la cible est identifié, puis résolu linéairement. C'est le correctif
// proposé à l'amont (gn_cube.c `level_solve`), écrit ici pour être mesuré.
//
// Même convention que la bissection, qui converge vers
// inf{ p : f(p) >= target } écrêté à [0, 1] : une cible sous f(0) rend 0, une
// cible au-dessus de f(1) rend 1, un segment plat rend sa borne gauche.
//
// blend < 0 inverse la courbe vive ; sinon la courbe mélangée
// (1−blend)·M_dead + blend·M_live, qui est affine sur les MÊMES segments
// parce que M_dead est affine sur [0, 1] tout entier.
func levelSolveClosed(lv *matchLevel, owner CubeOwner, blend, target float64) float64 {
	var segs [3][4]float64
	n := levelSegments(lv, owner, &segs)

	blended := func(x, y float64) float64 {
		if blend < 0.0 {
			return y
		}
		return (1.0-blend)*levelDead(lv, x) + blend*y
	}

	for i := 0; i < n; i++ {
		x0, y0, x1, y1 := segs[i][0], segs[i][1], segs[i][2], segs[i][3]
		if x1-x0 <= 0.0 {
			continue // segment dégénéré : `segment` n'y rend que y1
		}
		v0, v1 := blended(x0, y0), blended(x1, y1)
		if target <= v0 {
			return x0
		}
		if target <= v1 {
			if v1-v0 <= 0.0 {
				return x0
			}
			return x0 + (x1-x0)*((target-v0)/(v1-v0))
		}
	}
	return 1.0
}

// closedFormLevels rend un corpus de niveaux réels : la chaîne d'enjeux
// complète de chaque état de match plausible, alimentée par des mélanges de
// résultats couvrant la course sèche comme le jeu à fort gammon.
//
// Les ANCRES viennent de buildLevelAnchors (donc de la vraie MET) et les
// points de rupture des niveaux profonds de resolveLevels, si bien que chaque
// niveau comparé est un niveau que le moteur résout vraiment.
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

// TestClosedFormAgreesWithBisection est le dispositif d'exactitude de la
// forme close : sur des niveaux réels, elle rend le même p que soixante pas
// de bissection, à 1e-9 près en p — la tolérance est celle d'une bissection
// qui converge, pas celle d'un modèle qui approxime.
//
// Il tourne toujours, parce que c'est lui qui dit à l'amont que le correctif
// proposé décrit bien la même fonction ; il coûte quelques millisecondes.
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
					got := levelSolveClosed(&lv, owner, blend, target)
					want := levelSolve(&lv, owner, blend, target)
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

// TestMeasureClosedFormGap — MESURE 1 : de combien de bits la forme close
// s'écarte-t-elle de la bissection, et qu'est-ce que ça fait à une équité ?
//
// C'est LA question qui décide si le portage peut suivre l'amont sans
// périmer les bases : l'écart en p, en ULP, la part des inversions
// rigoureusement bit-identiques, et l'écart propagé sur Value.
func TestMeasureClosedFormGap(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE_CLOSEDFORM") == "" {
		t.Skip("set BLUNDERDB_MEASURE_CLOSEDFORM to measure the closed-form gap")
	}
	corpus := closedFormLevels(t)
	owners := []CubeOwner{CubeCentred, CubeOwned, CubeOpponent}

	var maxP float64
	var maxPLabel string
	var identical, total int
	var maxULP uint64
	var maxULPLabel string

	for _, c := range corpus {
		lv := c.lv
		for _, owner := range owners {
			for _, target := range []float64{lv.pass, lv.cash} {
				got := levelSolveClosed(&lv, owner, -1.0, target)
				want := levelSolve(&lv, owner, -1.0, target)
				total++
				if got == want {
					identical++
				}
				if d := math.Abs(got - want); d > maxP {
					maxP, maxPLabel = d, fmt.Sprintf("%s owner=%v", c.label, owner)
				}
				// La distance ULP n'a de sens qu'entre deux nombres du même
				// ordre : quand la bissection converge vers 0 elle rend
				// 2^-61 et non 0, ce qui est un écart de 4e18 ULP pour 4e-19
				// de p. Restreinte aux inversions dont le résultat n'est pas
				// collé à une borne, elle dit ce qu'elle prétend dire.
				if math.Min(got, want) > 1e-6 && math.Max(got, want) < 1-1e-6 {
					if u := ulpDistance(got, want); u > maxULP {
						maxULP, maxULPLabel = u, fmt.Sprintf("%s owner=%v (%.17g vs %.17g)", c.label, owner, got, want)
					}
				}
			}
		}
	}

	t.Logf("inversions comparées: %d — bit-identiques: %d (%.2f%%)", total, identical, 100*float64(identical)/float64(total))
	t.Logf("écart max en p: %.3e (%s)", maxP, maxPLabel)
	t.Logf("distance ULP max hors bornes: %d (%s)", maxULP, maxULPLabel)

	// Propagation : la même chaîne d'enjeux résolue par bissection puis par
	// forme close, et l'équité normalisée que Value en tire.
	var maxV float64
	var maxVLabel string
	var vIdentical, vTotal int
	mixes := [][numOutcomes]float64{
		{0.40, 0.10, 0.01, 0.35, 0.13, 0.01},
		{0.20, 0.25, 0.05, 0.30, 0.18, 0.02},
		{0.62, 0.05, 0.00, 0.30, 0.03, 0.00},
	}
	for _, away := range [][2]int{{1, 1}, {2, 2}, {3, 5}, {5, 5}, {7, 4}, {15, 15}} {
		for _, cube := range []int{1, 2, 4} {
			st := MatchState{AwayOnRoll: away[0], AwayOpponent: away[1], Cube: cube}
			if !st.IsValid() {
				continue
			}
			for _, mix := range mixes {
				var bis, clo [maxCubeLevels]matchLevel
				count := buildLevelAnchors(st, mix, &bis)
				if count == 0 {
					continue
				}
				clo = bis
				resolveLevels(&bis, count)
				for i := count - 2; i >= 0; i-- {
					clo[i].tp = levelSolveClosed(&clo[i+1], CubeOwned, -1.0, clo[i].pass)
					clo[i].cp = levelSolveClosed(&clo[i+1], CubeOpponent, -1.0, clo[i].cash)
				}
				for _, owner := range owners {
					x := DefaultEfficiency(owner)
					for k := 0; k <= 20; k++ {
						p := float64(k) / 20.0
						a := 2.0*levelBlend(&bis[0], p, owner, x) - 1.0
						b := 2.0*levelBlend(&clo[0], p, owner, x) - 1.0
						vTotal++
						if a == b {
							vIdentical++
						}
						if d := math.Abs(a - b); d > maxV {
							maxV, maxVLabel = d, fmt.Sprintf("%d/%d cube %d p=%.2f %v", away[0], away[1], cube, p, owner)
						}
					}
				}
			}
		}
	}
	t.Logf("valuations comparées: %d — bit-identiques: %d (%.2f%%)", vTotal, vIdentical, 100*float64(vIdentical)/float64(vTotal))
	t.Logf("écart max propagé sur Value (équité normalisée): %.3e (%s)", maxV, maxVLabel)
}

// ulpDistance est le nombre de float64 représentables entre a et b. Zéro
// signifie bit-identique.
func ulpDistance(a, b float64) uint64 {
	if a == b {
		return 0
	}
	ia, ib := math.Float64bits(a), math.Float64bits(b)
	order := func(u uint64) uint64 {
		if u&(1<<63) != 0 {
			return ^u + 1 + (1 << 63)
		}
		return u + (1 << 63)
	}
	oa, ob := order(ia), order(ib)
	if oa > ob {
		return oa - ob
	}
	return ob - oa
}

// ── Les repères que la fiche C.7 réclamait AVANT toute modification ────────

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

// BenchmarkLevelSolveBisection est le poste que C.7 vise : une inversion,
// soixante pas.
func BenchmarkLevelSolveBisection(b *testing.B) {
	cur, next := benchLevel(b)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		sink += levelSolve(&next, CubeOwned, -1.0, cur.pass)
	}
	runtimeSink = sink
}

// BenchmarkLevelSolveClosed est la même inversion en forme close — le
// correctif proposé à l'amont, mesuré ici pour que la proposition arrive avec
// son chiffre.
func BenchmarkLevelSolveClosed(b *testing.B) {
	cur, next := benchLevel(b)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		sink += levelSolveClosed(&next, CubeOwned, -1.0, cur.pass)
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

// BenchmarkCubeDecisionAtScore est la décision de videau que le panneau
// affiche : Decide au score, sur une distribution déjà calculée. C'est le
// chemin où l'inversion apparaît DEUX fois de plus que dans la valuation
// d'une feuille — la chaîne, puis le take point rapporté.
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

// BenchmarkAnalysisBatchThroughput est le DÉBIT DU LOT que la fiche C.7
// réclamait : des positions réelles au score, une goroutine par cœur, chacune
// avec son propre chercheur réutilisé (le motif que le lot d'analyse fait
// tourner, `db_gammonnet_batch.go`). La métrique utile est pos/s, pas ns/op :
// c'est celle dans laquelle une base de 88 000 positions se compte.
//
// Le lot est LE consommateur de la valuation de videau — chaque position y
// paye une chaîne d'enjeux par feuille — donc c'est ici qu'un gain sur
// levelSolve se lit en heures de calcul plutôt qu'en nanosecondes.
func BenchmarkAnalysisBatchThroughput(b *testing.B) {
	for _, ply := range []int{0, 2} {
		b.Run(fmt.Sprintf("%d-ply", ply), func(b *testing.B) {
			if ply > 0 && testing.Short() {
				b.Skip("une position 2-ply coûte des centaines de millisecondes")
			}
			positions := batchBenchPositions(b)
			workers := runtime.NumCPU()
			// Les chercheurs sont bâtis HORS du chrono : 5,5 Mo chacun, et
			// le lot réel n'en construit qu'un par goroutine pour toute la
			// passe (NewBatchSearcher, #147). Les chronométrer mesurerait
			// l'allocation, pas le débit.
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
