// SPDX-License-Identifier: MIT

package gammonnet

// ── La même valuation, n candidats à la fois — L'EXPÉRIENCE, RÉFUTÉE ────────
//
// Portage fidèle de gn_cube_value_batch (spec §7.1), exact au bit près mais
// PLUS LENT QUE LE SCALAIRE ICI (~12 % sur une décision 2-ply au score) —
// d'où sa place dans un fichier de test. Gardé parce qu'une optimisation
// conceptuelle se remesure chez chaque consommateur (gammonNet ADR-0003) :
// TestMeasureCubePost rejoue la mesure sur une autre machine.
//
// En C, level_live est inlinée et mener douze voies de front recouvre la
// latence de la division. En Go, la levée laneCurve (cube.go) suffit déjà au
// prédicteur ; il ne reste au lot que ses coûts (état des voies en mémoire,
// une division de plus par pas en forme sans branche).

// CubeBatchWidth est le nombre de voies d'une valuation de videau par lot.
//
// Un PARAMÈTRE DE COÛT, jamais du moteur : N candidats en un lot, en deux
// moitiés ou un par un rendent les mêmes bits (TestCubeBatchSplitInvariance).
const CubeBatchWidth = 32

// cubeScratch est la mémoire de travail d'une valuation par lot. Elle vit
// avec le chercheur, comme le reste du brouillon : une allocation par nœud
// coûterait 14 Ko à chacun des milliers de nœuds d'une décision.
type cubeScratch struct {
	levels [CubeBatchWidth][maxCubeLevels]matchLevel
	lane   [CubeBatchWidth]cubeLane
	win    [CubeBatchWidth]float64
}

// cubeLane est l'état d'UNE voie pendant une bissection, en structure plutôt
// qu'en tableaux parallèles : le pas cadencé touche tous les champs.
type cubeLane struct {
	c      laneCurve
	low    float64
	high   float64
	target float64
}

// atSelect rend exactement la même valeur que at, SANS BRANCHE : les deux
// segments sont calculés puis l'un est choisi.
//
// Forme du pas cadencé, comme gammonNet (« two selects rather than an
// if/else ») : une erreur de prédiction jetterait le travail de toutes les
// voies en vol. La bissection sérielle garde at(), où la division de plus se
// paye.
//
// Le segment écarté peut rendre un infini ou un NaN, sans conséquence (Go ne
// piège pas) ; celui choisi est at() au bit près (TestCubeBatchMatchesScalar).
func (c *laneCurve) atSelect(p float64) float64 {
	lo := c.loseAvg + c.nLo*((p-0.0)/c.dLo)
	hi := c.mid + c.nHi*((p-c.brk)/c.dHi)
	if c.dLo <= 0.0 {
		lo = c.mid
	}
	if c.dHi <= 0.0 {
		hi = c.winAvg
	}
	v := hi
	if p <= c.brk {
		v = lo
	}
	if c.dead {
		v = (1.0-p)*c.loseAvg + p*c.winAvg
	}
	return v
}

// solveLanes résout UN point de rupture pour toutes les voies à la fois.
//
// La bissection est une chaîne sérielle qu'un processeur ne recouvre qu'avec
// celle d'un autre candidat : le lot mène les soixante pas de toutes les voies
// en pas cadencé. L'arithmétique d'une voie est inchangée, seul
// l'entrelacement change, d'où l'identité au bit avec le scalaire.
//
// owner == CubeOwned résout tp contre la courbe « possédée » du niveau
// au-dessus et le pass ; CubeOpponent résout cp contre l'« adverse » et le
// cash — resolveLevels, découpé par point de rupture.
func (b *cubeScratch) solveLanes(lanes, level int, owner CubeOwner) {
	ls := b.lane[:lanes]
	for j := range ls {
		// resolveLevels ne résout jamais un point de rupture contre la courbe
		// centrée : tp vient de la courbe possédée, cp de l'adverse.
		ls[j].c.set(&b.levels[j][level+1], owner)
		if owner == CubeOwned {
			ls[j].target = b.levels[j][level].pass
		} else {
			ls[j].target = b.levels[j][level].cash
		}
		ls[j].low, ls[j].high = 0.0, 1.0
	}

	// Soixante itérations FIXES, comme le scalaire : jamais « jusqu'à
	// convergence », qui ferait dépendre une voie de ses voisines.
	for it := 0; it < 60; it++ {
		for j := range ls {
			l := &ls[j]
			lo, hi := l.low, l.high
			mid := 0.5 * (lo + hi)
			below := l.c.atSelect(mid) < l.target
			if below {
				lo = mid
			}
			if !below {
				hi = mid
			}
			l.low, l.high = lo, hi
		}
	}

	for j := range ls {
		p := 0.5 * (ls[j].low + ls[j].high)
		if owner == CubeOwned {
			b.levels[j][level].tp = p
		} else {
			b.levels[j][level].cp = p
		}
	}
}

// cubeValueBatch value n distributions qui partagent UN SEUL état de videau.
// out[j] est, au bit près, ce que Value(probs[j], owner, state, efficiency)
// aurait rendu seul — ce n'est pas une révision du modèle.
//
// Rend false pour un lot que le modèle refuse : l'appelant retombe sur le
// scalaire, qui refuse candidat par candidat. Le dernier morceau tourne
// simplement moins de voies. Chaque voie paye ses propres consultations de
// MET, comme le scalaire : les dédupliquer ne rapportait que ~1 %.
func cubeValueBatch(b *cubeScratch, probs []*[NumOutputs]float32, owner CubeOwner, state *MatchState, efficiency float64, out []float64) bool {
	if len(out) < len(probs) {
		return false
	}
	// L'argent reste scalaire (spec §7.1) : sans récursion, rien à gagner.
	if state == nil {
		for j, p := range probs {
			v, ok := Value(p, owner, nil, efficiency)
			if !ok {
				return false
			}
			out[j] = v
		}
		return true
	}
	if !state.IsValid() {
		return false
	}

	for base := 0; base < len(probs); base += CubeBatchWidth {
		lanes := len(probs) - base
		if lanes > CubeBatchWidth {
			lanes = CubeBatchWidth
		}
		count := 0
		for j := 0; j < lanes; j++ {
			p := probs[base+j]
			b.win[j] = CubeInputsFromProbs(p).Win
			outcomes := probsExclusive(p)
			here := buildLevelAnchors(*state, outcomes, &b.levels[j])
			if here < 2 {
				return false
			}
			if j == 0 {
				count = here
			} else if here != count {
				// Inatteignable (la forme de la chaîne ne dépend que de
				// state), mais refusé : des voies en désaccord résoudraient
				// contre le mauvais niveau, en restant plausibles.
				return false
			}
		}

		// Les plus profonds d'abord, exactement comme resolveLevels — mais
		// chaque point de rupture résolu pour TOUTES les voies avant que le
		// suivant ne démarre.
		for i := count - 2; i >= 0; i-- {
			b.solveLanes(lanes, i, CubeOwned)
			b.solveLanes(lanes, i, CubeOpponent)
		}

		for j := 0; j < lanes; j++ {
			// La queue de Value, mot pour mot — partie de Crawford comprise,
			// où aucun videau n'est en jeu.
			if state.Crawford {
				out[base+j] = 2.0*levelDead(&b.levels[j][0], b.win[j]) - 1.0
			} else {
				out[base+j] = 2.0*levelBlend(&b.levels[j][0], b.win[j], owner, efficiency) - 1.0
			}
		}
	}
	return true
}
