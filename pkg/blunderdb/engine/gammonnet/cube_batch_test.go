// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Les preuves d'exactitude du videau par lot (ADR-0003, spec §7.1 point 6) —
// écrites avant le code, et exigeant l'égalité AU BIT PRÈS, jamais une
// tolérance. Une valuation par lot n'est pas une révision du modèle : c'est
// la même arithmétique, dans le même ordre, sur les mêmes valeurs, seul
// l'entrelacement change.

// realCubeDistributions rend n distributions RÉELLES — la réponse brute du
// grand réseau sur des plateaux aléatoires — plutôt que des cinq-uplets
// écrits à la main. Le modèle de videau bissecte des courbes dont les
// ruptures dépendent du mélange gammon de la position ; des distributions
// inventées ratent les régimes que le vrai réseau produit.
func realCubeDistributions(t *testing.T, n int) [][NumOutputs]float32 {
	t.Helper()
	net, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ev := NewEvaluator(net)
	rng := rand.New(rand.NewSource(20260903))

	var out [][NumOutputs]float32
	var features [NumFeatures]float32
	for len(out) < n {
		onRoll := domain.White
		if len(out)%2 == 1 {
			onRoll = domain.Black
		}
		b := randomBoard(rng, onRoll)
		p, err := FromDomain(&b)
		if err != nil || p.isOver() || !Encode(&p, &features) {
			continue
		}
		var probs [NumOutputs]float32
		if err := ev.Evaluate(features[:], &probs); err != nil {
			t.Fatal(err)
		}
		out = append(out, probs)
	}
	return out
}

// cubeBatchStates est le croisement que la preuve amont exige : les
// possessions et les états de match, argent compris.
func cubeBatchStates() []struct {
	name  string
	state *MatchState
} {
	money := struct {
		name  string
		state *MatchState
	}{"argent", nil}
	mk := func(name string, s MatchState) struct {
		name  string
		state *MatchState
	} {
		st := s
		return struct {
			name  string
			state *MatchState
		}{name, &st}
	}
	return []struct {
		name  string
		state *MatchState
	}{
		money,
		mk("5-away/5-away", MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}),
		mk("2-away/4-away", MatchState{AwayOnRoll: 2, AwayOpponent: 4, Cube: 1}),
		mk("videau à 2", MatchState{AwayOnRoll: 7, AwayOpponent: 3, Cube: 2}),
		mk("1-away/1-away", MatchState{AwayOnRoll: 1, AwayOpponent: 1, Cube: 1}),
		mk("Crawford", MatchState{AwayOnRoll: 4, AwayOpponent: 1, Cube: 1, Crawford: true}),
		mk("25-away/25-away", MatchState{AwayOnRoll: 25, AwayOpponent: 25, Cube: 1}),
	}
}

// TestCubeBatchMatchesScalar : out[j] est, au bit près, ce que Value aurait
// rendu seul. Pas de tolérance — l'égalité est `==`.
func TestCubeBatchMatchesScalar(t *testing.T) {
	dists := realCubeDistributions(t, 141)
	probs := make([]*[NumOutputs]float32, len(dists))
	for i := range dists {
		probs[i] = &dists[i]
	}
	out := make([]float64, len(dists))
	scratch := new(cubeScratch)

	for _, owner := range []CubeOwner{CubeCentred, CubeOwned, CubeOpponent} {
		for _, st := range cubeBatchStates() {
			x := DefaultEfficiency(owner)
			if !cubeValueBatch(scratch, probs, owner, st.state, x, out) {
				t.Fatalf("%s / %v : le lot a refusé", st.name, owner)
			}
			for j := range probs {
				want, ok := Value(probs[j], owner, st.state, x)
				if !ok {
					t.Fatalf("%s / %v / %d : le scalaire a refusé là où le lot a accepté", st.name, owner, j)
				}
				if out[j] != want {
					t.Fatalf("%s / %v / %d : lot %.17g, scalaire %.17g — le lot n'est pas une révision du modèle",
						st.name, owner, j, out[j], want)
				}
			}
		}
	}
}

// TestCubeBatchSplitInvariance : la largeur de voie est un PARAMÈTRE DE COÛT.
// Le lot entier, deux moitiés coupées à un endroit qui n'est pas un multiple
// de la largeur, et une voie à la fois rendent les mêmes bits.
func TestCubeBatchSplitInvariance(t *testing.T) {
	dists := realCubeDistributions(t, 77)
	probs := make([]*[NumOutputs]float32, len(dists))
	for i := range dists {
		probs[i] = &dists[i]
	}
	whole := make([]float64, len(dists))
	split := make([]float64, len(dists))
	one := make([]float64, len(dists))
	scratch := new(cubeScratch)

	for _, owner := range []CubeOwner{CubeCentred, CubeOwned, CubeOpponent} {
		for _, st := range cubeBatchStates() {
			x := DefaultEfficiency(owner)
			if !cubeValueBatch(scratch, probs, owner, st.state, x, whole) {
				t.Fatalf("%s : le lot entier a refusé", st.name)
			}
			// 37 : ni un multiple de CubeBatchWidth, ni la moitié de 77.
			const cut = 37
			if !cubeValueBatch(scratch, probs[:cut], owner, st.state, x, split[:cut]) ||
				!cubeValueBatch(scratch, probs[cut:], owner, st.state, x, split[cut:]) {
				t.Fatalf("%s : une moitié a refusé", st.name)
			}
			for j := range probs {
				if !cubeValueBatch(scratch, probs[j:j+1], owner, st.state, x, one[j:j+1]) {
					t.Fatalf("%s : la voie unique %d a refusé", st.name, j)
				}
			}
			for j := range probs {
				if whole[j] != split[j] || whole[j] != one[j] {
					t.Fatalf("%s / %v / %d : entier %.17g, moitiés %.17g, une par une %.17g — "+
						"l'arithmétique d'une voie dépend du nombre de ses voisines",
						st.name, owner, j, whole[j], split[j], one[j])
				}
			}
		}
	}
}

// ── La levée, mesurée et prouvée séparément du lot ──────────────────────────

// levelSolveUnlifted et valueUnlifted sont la valuation TELLE QU'ELLE ÉTAIT
// avant cette verticale : la bissection appelle levelLive à chaque pas, avec
// son switch sur la possession, son test « mort » et ses deux soustractions
// de segment refaits soixante fois.
//
// Elles vivent dans le fichier de test parce qu'elles ne sont plus du code de
// production : elles servent (a) à prouver que la levée est exacte au bit
// près, et (b) à chronométrer séparément les deux moitiés du gain — la levée,
// qui est une optimisation d'implémentation propre à Go, et l'entrelacement
// des voies, qui est l'optimisation conceptuelle reprise d'amont. Sans ce
// « avant » gardé sous la main, les deux gains se seraient additionnés dans
// un seul chiffre et le portage aurait crédité l'amont de ce que le
// compilateur Go lui devait.
func levelSolveUnlifted(lv *matchLevel, owner CubeOwner, target float64) float64 {
	low, high := 0.0, 1.0
	for i := 0; i < 60; i++ {
		mid := 0.5 * (low + high)
		if levelLive(lv, mid, owner) < target {
			low = mid
		} else {
			high = mid
		}
	}
	return 0.5 * (low + high)
}

func valueUnlifted(probs *[NumOutputs]float32, owner CubeOwner, state *MatchState, efficiency float64) (float64, bool) {
	in := CubeInputsFromProbs(probs)
	if state == nil {
		return janowskiEquity(in.Win, in.WinPoints, in.LosePoints, owner, efficiency), true
	}
	if !state.IsValid() {
		return 0, false
	}
	var levels [maxCubeLevels]matchLevel
	count := buildLevelAnchors(*state, probsExclusive(probs), &levels)
	if count < 2 {
		return 0, false
	}
	for i := count - 2; i >= 0; i-- {
		levels[i].tp = levelSolveUnlifted(&levels[i+1], CubeOwned, levels[i].pass)
		levels[i].cp = levelSolveUnlifted(&levels[i+1], CubeOpponent, levels[i].cash)
	}
	if state.Crawford {
		return 2.0*levelDead(&levels[0], in.Win) - 1.0, true
	}
	return 2.0*levelBlend(&levels[0], in.Win, owner, efficiency) - 1.0, true
}

// TestLiftedSolveMatchesUnlifted : sortir les constantes de segment des
// soixante pas, et calculer les deux segments pour n'en garder qu'un, ne
// déplace pas un bit. C'est la preuve que la levée n'est pas une révision du
// modèle — la même que le lot doit fournir, et pour la même raison.
func TestLiftedSolveMatchesUnlifted(t *testing.T) {
	dists := realCubeDistributions(t, 141)
	for _, owner := range []CubeOwner{CubeCentred, CubeOwned, CubeOpponent} {
		for _, st := range cubeBatchStates() {
			x := DefaultEfficiency(owner)
			for j := range dists {
				want, okw := valueUnlifted(&dists[j], owner, st.state, x)
				got, okg := Value(&dists[j], owner, st.state, x)
				if okw != okg {
					t.Fatalf("%s / %v / %d : refus %v contre %v", st.name, owner, j, okg, okw)
				}
				if okw && got != want {
					t.Fatalf("%s / %v / %d : levé %.17g, non levé %.17g", st.name, owner, j, got, want)
				}
			}
		}
	}
}
