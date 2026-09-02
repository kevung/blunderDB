// SPDX-License-Identifier: MIT

package gammonnet

// ── T85 : la même valuation, n candidats à la fois — L'EXPÉRIENCE, RÉFUTÉE ──
//
// Ce qui suit est le portage fidèle de gn_cube_value_batch (gammonNet, ADR-0003,
// spec §7.1 §7.1). Il est exact au bit près, il tient les deux dispositifs
// d'exactitude, et IL EST PLUS LENT QUE LE SCALAIRE ICI — d'où sa place dans un
// fichier de test plutôt que dans le moteur.
//
// Il n'est pas supprimé pour autant, et c'est délibéré. L'ADR-0003 dit qu'une
// optimisation conceptuelle se REMESURE chez chaque consommateur ; le verdict
// « ne paye pas » est donc un verdict de MACHINE autant que de langage, et
// quelqu'un sur un autre processeur doit pouvoir le rejouer sans réécrire
// l'expérience. TestMeasureCubePost est ce rejeu.
//
// Ce que la mesure a dit ici (Ryzen 7 PRO 6850U, Go 1.25.13, épinglé sur deux
// cœurs, machine partagée) :
//
//	          avant     levé      lot    levée    lot  les 2
//	 voies   ns/val   ns/val   ns/val        ×      ×      ×
//	     4     2272     1515     2422     1.50   0.63   0.94
//	    12     2649     1899     2599     1.40   0.73   1.02
//	    32     4408     3395     4438     1.30   0.77   0.99
//
// et sur une décision 2-ply entière au score, lot allumé contre lot éteint,
// entrelacé décision par décision : rapport médian 1,123 sur douze cas — le
// lot rend la décision 12 % PLUS LENTE, et les douze cas vont dans le même
// sens (1,07 à 1,23).
//
// POURQUOI. En C, level_live est inlinée dans la bissection, et le pas est une
// chaîne courte bornée par la latence d'une division : mener douze voies de
// front la recouvre, d'où le ×2,43 d'amont. En Go, levelLive n'est pas
// inlinable, et la levée des constantes de segment hors des soixante pas
// (laneCurve, cube.go) suffit à rendre le pas assez court pour que le
// prédicteur et la fenêtre d'exécution dans le désordre le couvrent DÉJÀ. Ce
// qui reste au lot, ce sont ses coûts : l'état des voies vit en mémoire au
// lieu de vivre en registres, et sa forme sans branche paye une division de
// plus par pas. C'est le cas d'école de l'ADR-0003 lu à l'envers — le gain
// conceptuel d'amont ne survit pas au changement de langage ici, exactement
// comme la sparsité de la couche 1 rend 16 % là-bas et 6 % ici.

// CubeBatchWidth est le nombre de voies d'une valuation de videau par lot.
//
// C'est un PARAMÈTRE DE COÛT, jamais un paramètre du moteur. Valuer N
// candidats en un lot, en deux moitiés ou un par un rend exactement les mêmes
// bits (TestCubeBatchSplitInvariance) : l'arithmétique d'une voie ne dépend
// jamais du nombre de ses voisines. La largeur ne se reprend donc PAS
// d'amont — gammonNet retient 32 en disant lui-même que sa mesure ne tranche
// pas entre 16, 32 et 64 — elle se mesure ici (docs des mesures de la
// verticale ADR-0003).
const CubeBatchWidth = 32

// cubeScratch est la mémoire de travail d'une valuation par lot. Elle vit
// avec le chercheur, comme le reste du brouillon : une allocation par nœud
// coûterait 14 Ko à chacun des milliers de nœuds d'une décision.
type cubeScratch struct {
	levels [CubeBatchWidth][maxCubeLevels]matchLevel
	lane   [CubeBatchWidth]cubeLane
	win    [CubeBatchWidth]float64
}

// cubeLane est l'état d'UNE voie pendant une bissection. Les quatre champs
// vivent ensemble plutôt que dans quatre tableaux parallèles : le pas cadencé
// les touche tous à chaque itération, et `for i := range lanes` sur une
// tranche de structures coûte un pointeur par voie là où quatre tableaux
// coûtent quatre indexations et leurs vérifications de borne.
type cubeLane struct {
	c      laneCurve
	low    float64
	high   float64
	target float64
}

// atSelect rend exactement la même valeur que at, SANS BRANCHE : les deux
// segments sont calculés puis l'un est choisi.
//
// C'est la forme que veut le pas cadencé, et gammonNet écrit la même chose
// pour la même raison (« two selects rather than an if/else on purpose ») :
// une erreur de prédiction ne coûte pas seulement ses vingt cycles, elle vide
// le pipeline — donc elle jette le travail spéculatif de TOUTES les autres
// voies en vol, c'est-à-dire exactement le travail que le lot existe pour
// mettre en vol. Dans la bissection sérielle le calcul est inverse (rien
// d'autre n'est en vol, et la division de plus se paye), ce qui est pourquoi
// les deux formes coexistent au lieu qu'une seule serve les deux sites.
//
// Le segment non choisi peut diviser par zéro et rendre un infini ou un NaN :
// il est écarté par la sélection, et Go ne piège aucune exception flottante.
// L'arithmétique du segment CHOISI est celle de at(), au bit près
// (TestCubeBatchMatchesScalar le vérifie de bout en bout).
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
// POURQUOI CE N'EST PAS UNE BOUCLE AU SITE D'APPEL. levelSolve est une chaîne
// sérielle : soixante pas, chacun une division dont le résultat choisit
// l'entrée du pas suivant. Rien dans un processeur ne recouvre ça avec
// soi-même. Ça PEUT le recouvrir avec la bissection d'un autre candidat,
// puisque deux candidats ne partagent rien — et la recherche a toujours une
// fratrie entière en main quand elle value l'un de ses membres (valueSweep).
// Le lot mène donc les soixante pas de toutes les voies en pas cadencé :
// itération par itération à travers les voies, plutôt que voie par voie à
// travers les itérations. L'arithmétique d'une voie est la même, dans le même
// ordre, sur les mêmes valeurs ; seul l'entrelacement change, et c'est
// pourquoi le résultat est bit à bit celui du scalaire.
//
// owner == CubeOwned résout tp contre la courbe « possédée » du niveau
// au-dessus et le pass de ce niveau ; CubeOpponent résout cp contre sa courbe
// « adverse » et son cash. C'est resolveLevels, découpé par point de rupture
// au lieu de l'être par candidat.
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

	// Nombre d'itérations FIXE — soixante, toujours, exactement comme le
	// scalaire. Jamais « jusqu'à convergence des voies » : ce serait faire
	// dépendre la réponse d'une voie de celle de ses voisines, ce que la
	// largeur fixe interdit précisément.
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
// Rend false pour un lot que le modèle refuse (état invalide, chaîne qui ne
// meurt pas) : l'appelant retombe alors sur le scalaire, qui refuse candidat
// par candidat comme il l'a toujours fait.
//
// Largeur de voie FIXE : un morceau est rempli jusqu'à CubeBatchWidth et le
// dernier tourne simplement moins de voies.
//
// CE QUI EST DÉLIBÉRÉMENT EXCLU DU LOT : remonter ou dédupliquer les
// consultations de la table d'équité de match, bien que pass, cash et les
// trois metAfter de branchMwc ne dépendent que de l'état et soient donc
// identiques dans toutes les voies. Ce portage l'a écrit, mesuré à 1 % et
// annulé (#150) ; c'est 11 % d'un poste, sous le plancher de bruit. Chaque
// voie paye ses propres consultations, exactement comme le scalaire.
func cubeValueBatch(b *cubeScratch, probs []*[NumOutputs]float32, owner CubeOwner, state *MatchState, efficiency float64, out []float64) bool {
	if len(out) < len(probs) {
		return false
	}
	// L'argent reste scalaire (spec §7.1) : le §3 n'a pas de récursion, une
	// valuation y coûte quelques nanosecondes contre quelques microsecondes
	// au score, et le poste y est MESURÉ à zéro. Rassembler pour rien serait
	// un coût pur.
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
				// Inatteignable : la forme de la chaîne est une fonction de
				// state seul (buildLevelAnchors). Refusé plutôt que rattrapé
				// — un lot dont les voies seraient en désaccord sur le nombre
				// de niveaux en résoudrait certaines contre le mauvais
				// niveau, et chaque réponse resterait plausible.
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
