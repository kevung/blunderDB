package engine

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// « Des positions comme celle-ci » : ce que « proche » veut dire, en une
// distance.
//
// Vecteur brut des points (P7 : l'espace latent du réseau n'est validé par
// aucune littérature pour le backgammon) et TRANSPORT OPTIMAL en une dimension
// (Wasserstein-1) : contrairement à L1/L2, W₁ tient compte de la proximité des
// points — un pion avancé d'un cran coûte 1, de six crans 6, là où L1 compte 2
// pour les deux. Sur un axe, W₁ est la somme des écarts absolus des sommes
// PRÉFIXES (forme close de Vallender), en O(n), et se lit en PIONS-PAS.
//
// Toujours du point de vue du joueur au trait ; aucune invariance par
// translation : les points ont une valeur absolue (P7 §2).

// SimilarityVector is a position reduced to what "close" is measured on: each
// side's fifteen checkers laid out on one axis, seen from the side on roll.
//
// Index 0 holds the checkers borne off and index 25 the bar, so BOTH
// distributions always carry exactly fifteen checkers — which is what lets the
// distance be a transport distance rather than an ad-hoc sum. Indices 1..24
// run in the mover's own direction, 1 being its own ace point.
type SimilarityVector struct {
	Mover    [26]int
	Opponent [26]int
}

// BuildSimilarityVector reduces a position to its similarity vector, seen from
// the side on roll.
func BuildSimilarityVector(p *domain.Position) SimilarityVector {
	var v SimilarityVector
	if p == nil {
		return v
	}
	mover := p.PlayerOnRoll
	opponent := domain.White
	if mover == domain.White {
		opponent = domain.Black
	}
	place := func(dst *[26]int, color int) {
		for i := 1; i <= domain.NumPoints; i++ {
			pt := p.Board.Points[i]
			if pt.Checkers == 0 || pt.Color != color {
				continue
			}
			// Black travels 24→1, so its own numbering is the board's. White
			// travels 1→24 and its ace point is 24: mirror it.
			idx := i
			if color == domain.White {
				idx = domain.NumPoints + 1 - i
			}
			dst[idx] += pt.Checkers
		}
		bar := domain.BlackBar
		if color == domain.White {
			bar = domain.WhiteBar
		}
		dst[25] += p.Board.Points[bar].Checkers
	}
	place(&v.Mover, mover)
	place(&v.Opponent, opponent)

	// Whatever is not on the board has been borne off, and a borne-off checker
	// sits at distance zero. Deriving it rather than reading Bearoff keeps the
	// two distributions at fifteen even for a board that lost a checker to a
	// malformed import — the distance stays defined instead of silently
	// meaning something else.
	v.Mover[0] = 15 - sumFrom(&v.Mover, 1)
	v.Opponent[0] = 15 - sumFrom(&v.Opponent, 1)
	if v.Mover[0] < 0 {
		v.Mover[0] = 0
	}
	if v.Opponent[0] < 0 {
		v.Opponent[0] = 0
	}
	return v
}

func sumFrom(v *[26]int, from int) int {
	n := 0
	for i := from; i < len(v); i++ {
		n += v[i]
	}
	return n
}

// SimilarityDistance is the 1-D transport distance between two positions, in
// checker-pips: how much checker movement separates them. Zero means the same
// arrangement seen from the side on roll.
func SimilarityDistance(a, b SimilarityVector) int {
	return transport1D(&a.Mover, &b.Mover) + transport1D(&a.Opponent, &b.Opponent)
}

// transport1D is Wasserstein-1 on a discrete axis: the sum of the absolute
// differences of the two prefix sums. Both distributions carry the same total,
// which is what makes the last prefix difference zero and the quantity a
// distance rather than a difference of masses.
func transport1D(a, b *[26]int) int {
	total, cumA, cumB := 0, 0, 0
	for i := 0; i < len(a); i++ {
		cumA += a[i]
		cumB += b[i]
		d := cumA - cumB
		if d < 0 {
			d = -d
		}
		total += d
	}
	return total
}
