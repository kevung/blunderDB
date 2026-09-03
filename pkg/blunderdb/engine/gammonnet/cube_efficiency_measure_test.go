// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"os"
	"sort"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// L'instrument de mesure de #192/C.5 — « à quel point l'efficacité du videau
// est-elle lue au mauvais endroit ? »
//
// LA QUESTION. `DefaultEfficiency` rend TROIS valeurs, une par état du videau
// (possédé 0,566 / centré 0,688 / adverse 0,687), ajustées séparément par
// gammonNet contre les TROIS colonnes de sa table bilatérale exacte
// (docs/mesures/2026-08-07-T34-ajustement.md). Ce sont donc trois
// coefficients de BRANCHE, pas une propriété de la position. Or :
//
//   - `search.go` fixe `cfg.CubeX` à la racine (`ConfigForPosition`) et
//     valorise chaque feuille avec ce x-là, alors que le PROPRIÉTAIRE est
//     miroité à chaque ply : une feuille sur deux est donc tarifée avec le
//     coefficient ajusté pour l'AUTRE branche.
//   - `Decide` tarife la branche `eDT` — celle où l'adversaire détient le
//     videau doublé — avec l'efficacité passée par l'appelant, qui est celle
//     du propriétaire COURANT.
//
// Le C fait exactement pareil (`gn_search.c:299,740`, `gn_cube.c:754,790`) :
// ce n'est pas un trou de portage, c'est une question de modèle. Ces mesures
// la chiffrent avant de la trancher (ADR-0028).
//
// CE QU'ELLES ONT DONNÉ le 2026-09-03, 16 cœurs, sur les 669 décisions
// analysées réelles des fixtures de la porte d'intégration :
//
//   - 369 / 669 (55,2 %) se jouent videau déjà tourné — les seules
//     concernées ; sur les 300 autres la racine est centrée et l'écart est
//     rigoureusement nul.
//   - Écart à UNE feuille : money 0,0227 moyen / 0,0480 max (points par unité
//     de videau) ; match 0,0050 moyen, 0,0040 médian, 0,0137 p95, 0,0404 max
//     (équité normalisée), 10,2 % au-dessus de 0,01.
//   - eDT retarifé sur sa propre branche, 604 décisions de videau au score :
//     |Δ| moyen 0,00126 MWC, max 0,0159 ; take point déplacé de 0,0011 en
//     moyenne, 0,0069 au pire ; **0 verdict basculé sur 604**.
//   - Recherche 2-ply canonique, 60 positions à videau tourné : la variante
//     exacte (nodeValue rapiécé en DefaultEfficiency(owner), mesuré puis
//     révoqué) change **0 meilleur coup sur 60**, |Δ équité| 0,0040 moyen,
//     0,0084 max ; l'encadrement ci-dessous (x uniforme 0,566 vs 0,687) rend
//     les mêmes chiffres, 0/60 et 0,0040 / 0,0089.
//
// Autrement dit : lu au mauvais endroit souvent, visible à la troisième
// décimale d'une équité affichée, et jamais vu changer ce que l'outil dit.
//
// Rien n'est asserté ici : le fichier est derrière BLUNDERDB_MEASURE_CUBEX,
// comme cube_measure_test.go l'est derrière BLUNDERDB_MEASURE.

// cubeXCorpus is a corpus of REAL analysed positions — the same two gnubg
// fixtures and the same XG fixture the integration gate replays — because the
// question is "how often, and by how much, on a real board", and a corpus of
// randomised boards would answer a different one.
func cubeXCorpus(t *testing.T) []gateDecision {
	t.Helper()
	var out []gateDecision
	for _, path := range []string{"../../../../testdata/test.sgf", "../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf"} {
		mg, err := ingest.MapGnuBG(path)
		if err != nil {
			t.Fatalf("MapGnuBG(%s): %v", path, err)
		}
		out = append(out, extractDecisions(mg, "gnubg")...)
	}
	xg, err := ingest.MapXG("../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg")
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	return append(out, extractDecisions(xg, "xg")...)
}

type deltaStats struct {
	n            int
	sum, max     float64
	worst        string
	values       []float64
	overOneMilli int
	overTenMilli int
}

func (d *deltaStats) add(v float64, label string) {
	a := math.Abs(v)
	d.n++
	d.sum += a
	d.values = append(d.values, a)
	if a > d.max {
		d.max, d.worst = a, label
	}
	if a > 0.001 {
		d.overOneMilli++
	}
	if a > 0.01 {
		d.overTenMilli++
	}
}

func (d *deltaStats) report(t *testing.T, title string) {
	t.Helper()
	if d.n == 0 {
		t.Logf("%s: aucun cas", title)
		return
	}
	sort.Float64s(d.values)
	p95 := d.values[(len(d.values)*95)/100]
	t.Logf("%s: n=%d  moyen=%.6f  médian=%.6f  p95=%.6f  max=%.6f (%s)  >1e-3: %d (%.1f%%)  >1e-2: %d (%.1f%%)",
		title, d.n, d.sum/float64(d.n), medianOf(d.values), p95, d.max, d.worst,
		d.overOneMilli, 100*float64(d.overOneMilli)/float64(d.n),
		d.overTenMilli, 100*float64(d.overTenMilli)/float64(d.n))
}

// TestMeasureCubeXLeafGap — MESURE 1 : l'erreur commise à UNE feuille.
//
// Une feuille dont le videau local est `o` devrait, si x est un coefficient
// de branche, être tarifée à `DefaultEfficiency(o)`. La recherche la tarife à
// `DefaultEfficiency(root)` où root est le videau de la RACINE. Comme le
// propriétaire est miroité à chaque ply, l'écart mesuré ici est exactement
// celui d'une feuille sur deux — et il est NUL quand le videau de la racine
// est centré (Mirror(Centred) == Centred).
func TestMeasureCubeXLeafGap(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE_CUBEX") == "" {
		t.Skip("set BLUNDERDB_MEASURE_CUBEX to measure the cube-efficiency model gap")
	}
	corpus := cubeXCorpus(t)
	net, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := EmbeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	var turned, centred, money int
	moneyGap := &deltaStats{}
	matchGap := &deltaStats{}

	for i, d := range corpus {
		state, owner, ok := matchStateFor(d.pos, d.crawford)
		if !ok {
			money++
			continue
		}
		if owner == CubeCentred {
			centred++
		} else {
			turned++
		}

		// The leaf distribution: a 0-ply pre-roll walk, which is exactly what
		// leafValue prices.
		cfg := DefaultConfig(0)
		cfg.UseMatch, cfg.Match = true, state
		s := NewSearcherWith(cfg, net, prune)
		pos, err := FromDomain(d.pos)
		if err != nil {
			continue
		}
		probs, ok := s.Probs(&pos)
		if !ok {
			continue
		}

		// Money scale, per unit of cube: what a leaf under an OWNED cube is
		// worth priced at 0.566 (the fit's own branch) against 0.687 (what
		// the search hands it when the root cube is the opponent's), and its
		// mirror.
		for _, o := range []CubeOwner{CubeOwned, CubeOpponent} {
			local, ok1 := Value(&probs, o, nil, DefaultEfficiency(o))
			foreign, ok2 := Value(&probs, o, nil, DefaultEfficiency(o.Mirror()))
			if ok1 && ok2 {
				moneyGap.add(local-foreign, fmt.Sprintf("#%d %v money", i, o))
			}
			ls, ok1 := Value(&probs, o, &state, DefaultEfficiency(o))
			fs, ok2 := Value(&probs, o, &state, DefaultEfficiency(o.Mirror()))
			if ok1 && ok2 {
				matchGap.add(ls-fs, fmt.Sprintf("#%d %v %v", i, o, state))
			}
		}
	}

	t.Logf("corpus: %d décisions — %d au score videau tourné, %d au score videau centré, %d hors MET/money",
		len(corpus), turned, centred, money)
	t.Logf("part des décisions concernées (videau tourné): %.1f%%",
		100*float64(turned)/float64(max(1, turned+centred+money)))
	moneyGap.report(t, "écart de feuille, money (points par unité de videau)")
	matchGap.report(t, "écart de feuille, match (équité normalisée)")
}

// TestMeasureCubeXDecisionEDT — MESURE 2 : la branche eDT de `Decide`.
//
// `Decide` calcule eDT sur la branche ADVERSE (le videau doublé appartient à
// l'adversaire) mais à l'efficacité du propriétaire COURANT. Cette mesure
// recalcule eDT à l'efficacité de la branche adverse et compte : l'écart
// d'équité, le déplacement du take point, et les verdicts qui basculent.
func TestMeasureCubeXDecisionEDT(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE_CUBEX") == "" {
		t.Skip("set BLUNDERDB_MEASURE_CUBEX to measure the cube-efficiency model gap")
	}
	corpus := cubeXCorpus(t)
	net, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := EmbeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	edtGap := &deltaStats{}
	tpGap := &deltaStats{}
	var flips, decided int
	byOwner := map[CubeOwner]int{}

	for i, d := range corpus {
		state, owner, ok := matchStateFor(d.pos, d.crawford)
		if !ok || state.Crawford {
			continue
		}
		cfg := DefaultConfig(0)
		cfg.UseMatch, cfg.Match = true, state
		s := NewSearcherWith(cfg, net, prune)
		pos, err := FromDomain(d.pos)
		if err != nil {
			continue
		}
		probs, ok := s.Probs(&pos)
		if !ok {
			continue
		}

		x := DefaultEfficiency(owner)
		xOpp := DefaultEfficiency(CubeOpponent)
		dec, ok := Decide(&probs, owner, &state, x, false)
		if !ok {
			continue
		}
		outcomes := probsExclusive(&probs)
		in := CubeInputsFromProbs(&probs)
		levels, count := buildLevels(state, outcomes)
		if count < 2 {
			continue
		}
		// The branch-local variant: eND keeps the current owner's fitted x,
		// eDT takes the OPPONENT branch's — the only two coefficients the
		// two curves were ever fitted for.
		eND := levelBlend(&levels[0], in.Win, owner, x)
		eDTLocal := levelBlend(&levels[1], in.Win, CubeOpponent, xOpp)
		eDP := levels[0].cash
		tpLocal := levelSolve(&levels[1], CubeOpponent, xOpp, eDP)

		decided++
		byOwner[owner]++
		edtGap.add(dec.EquityDoubleTake-eDTLocal, fmt.Sprintf("#%d %v %v", i, owner, state))
		tpGap.add(dec.TakePoint-tpLocal, fmt.Sprintf("#%d %v", i, owner))
		if owner != CubeOpponent && Verdict(eND, dec.EquityDoubleTake, eDP) != Verdict(eND, eDTLocal, eDP) {
			flips++
			t.Logf("  BASCULE #%d %v %v: %v -> %v (eDT %.6f -> %.6f, eND %.6f, eDP %.6f)",
				i, owner, state, Verdict(eND, dec.EquityDoubleTake, eDP),
				Verdict(eND, eDTLocal, eDP), dec.EquityDoubleTake, eDTLocal, eND, eDP)
		}
	}

	t.Logf("décisions de videau au score évaluées: %d (centré %d, possédé %d, adverse %d)",
		decided, byOwner[CubeCentred], byOwner[CubeOwned], byOwner[CubeOpponent])
	edtGap.report(t, "écart eDT (MWC, avant normalisation)")
	tpGap.report(t, "déplacement du take point")
	t.Logf("verdicts qui basculent: %d / %d", flips, decided)
}

// TestMeasureCubeXSearchSensitivity — MESURE 3 : le choix du coup est-il
// sensible à x du tout ?
//
// La variante « x local » assigne à chaque feuille l'un des deux coefficients
// {0,566 ; 0,687}. Faute d'un drapeau en production (et il n'y en aura pas
// avant que gammonNet ait tranché), cette mesure encadre : elle fait tourner
// la MÊME recherche 2-ply aux deux extrêmes, x = 0,566 partout puis x = 0,687
// partout. Si le meilleur coup ne bouge jamais sur toute l'amplitude, aucune
// répartition intermédiaire ne le fera bouger non plus.
//
// C'est un encadrement empirique, pas une preuve : janowskiEquity est affine
// en x, de pente (live − dead), dont le SIGNE dépend de p et de la branche,
// donc la valeur d'une recherche n'est pas monotone en un mélange par feuille.
func TestMeasureCubeXSearchSensitivity(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE_CUBEX") == "" {
		t.Skip("set BLUNDERDB_MEASURE_CUBEX to measure the cube-efficiency model gap")
	}
	corpus := cubeXCorpus(t)
	net, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := EmbeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	limit := 60
	eqGap := &deltaStats{}
	var checked, disagree int

	for i, d := range corpus {
		if checked >= limit {
			break
		}
		if d.kind != "checker" {
			continue
		}
		state, owner, ok := matchStateFor(d.pos, d.crawford)
		if !ok || owner == CubeCentred {
			continue // only a turned cube can mis-price a mirrored leaf
		}
		pos := *d.pos
		pos.Dice = [2]int{d.dice[0], d.dice[1]}
		if len(domain.LegalMoves(&pos)) < 2 {
			continue
		}
		gp, err := FromDomain(&pos)
		if err != nil {
			continue
		}

		var best [2]Candidate
		okBoth := true
		for k, x := range [2]float64{DefaultEfficiency(CubeOwned), DefaultEfficiency(CubeOpponent)} {
			cfg := DefaultConfig(2)
			cfg.UseMatch, cfg.Match = true, state
			cfg.UseCube, cfg.CubeOwner, cfg.CubeX = true, owner, x
			s := NewSearcherWith(cfg, net, prune).WithWorkers(16)
			c, ok, err := s.BestPlay(&gp, d.dice[0], d.dice[1])
			if err != nil || !ok {
				okBoth = false
				break
			}
			best[k] = c
		}
		if !okBoth {
			continue
		}
		checked++
		eqGap.add(best[0].Equity-best[1].Equity, fmt.Sprintf("#%d %v", i, state))
		if best[0].Play.Result != best[1].Play.Result {
			disagree++
			t.Logf("  COUP DIFFÉRENT #%d %v videau=%v: équités %.6f / %.6f",
				i, state, owner, best[0].Equity, best[1].Equity)
		}
	}

	t.Logf("positions à videau tourné recherchées 2-ply aux deux extrêmes de x: %d", checked)
	t.Logf("meilleurs coups différents: %d / %d", disagree, checked)
	eqGap.report(t, "amplitude de l'équité 2-ply sur x ∈ {0,566 ; 0,687} (équité normalisée)")
}

// TestMeasureGateRedCasesAtDepth — MESURE 4, le second point de #192/C.5 :
// « rejouer les 2 cas rouges de integration_gate_test.go à 3-ply et avec la
// MET de XG ».
//
// Les deux cas sont les seules décisions au-dessus du bloc de 0,05 de la
// porte : score [1,5], dés [4,3] (coût 0,0552) et [1,1] (coût 0,0738). Ils
// sont dans la partie de Crawford, où il n'y a pas de videau du tout — ni
// `use_cube` ni l'efficacité ne peuvent les déplacer, c'est ce que
// l'en-tête de la porte a déjà mesuré au 2026-09-02. Reste la PROFONDEUR et
// la TABLE.
//
// La table est déjà réglée : engine/met.go EST Kazaross-XG2, la MET par
// défaut d'eXtreme Gammon (et de gnubg). Il n'y a pas de « MET de XG » à
// essayer en plus — c'est celle qui tourne. Ce test fait donc varier ce qui
// reste : la profondeur (2-ply canonique, 2-ply sans élagage, 3-ply), et
// reporte le coût contre les équités stockées de XG à chaque réglage.
func TestMeasureGateRedCasesAtDepth(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE_GATE_DEPTH") == "" {
		t.Skip("set BLUNDERDB_MEASURE_GATE_DEPTH to replay the gate's two red decisions at depth")
	}
	xg, err := ingest.MapXG("../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg")
	if err != nil {
		t.Fatal(err)
	}
	net, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := EmbeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	settings := []struct {
		name string
		cfg  SearchConfig
	}{
		{"2-ply k=12 (canonique)", DefaultConfig(2)},
		{"2-ply sans élagage", func() SearchConfig { c := DefaultConfig(2); c.PruneK = 0; return c }()},
		{"3-ply k=12", DefaultConfig(3)},
	}

	for _, d := range extractDecisions(xg, "xg") {
		if d.kind != "checker" || d.pos.Score != [2]int{1, 5} {
			continue
		}
		moves := d.analysis.CheckerAnalysis.Moves
		if len(moves) < 2 {
			continue
		}
		best := bestEquity(moves)
		for _, st := range settings {
			ours, ok := ourChoice(t, net, prune, st.cfg, d)
			if !ok {
				t.Logf("score %v dés %v — %s: non résolu", d.pos.Score, d.dice, st.name)
				continue
			}
			eq, found := equityFor(ours.Notation, moves)
			if !found {
				t.Logf("score %v dés %v — %s: %q hors du filet XG", d.pos.Score, d.dice, st.name, ours.Notation)
				continue
			}
			t.Logf("score %v dés %v — %-22s: %-22q coût %.4f (XG meilleur %.4f, le nôtre %.4f)",
				d.pos.Score, d.dice, st.name, ours.Notation, best-eq, best, eq)
		}
	}
}
