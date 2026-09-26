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

// Mesure : à quel point l'efficacité du videau est-elle lue au mauvais
// endroit ? (ADR-0029)
//
// DefaultEfficiency rend trois coefficients de BRANCHE. Or search.go fixe
// cfg.CubeX à la racine alors que le propriétaire est miroité à chaque ply,
// et Decide tarife eDT (branche adverse) à l'efficacité du propriétaire
// courant. Le C fait pareil (gn_search.c:299,740, gn_cube.c:754,790) : c'est
// une question de modèle, que ces mesures chiffrent.
//
// Sur les 669 décisions réelles de la porte d'intégration : écart moyen à
// une feuille de 0,005 en équité normalisée au score, aucun verdict basculé
// sur 604 décisions de videau, aucun meilleur coup changé sur 60. Lu au
// mauvais endroit souvent, jamais vu changer ce que l'outil dit.
//
// Rien n'est asserté : derrière BLUNDERDB_MEASURE_CUBEX.

// cubeXCorpus is the integration gate's corpus of REAL analysed positions:
// the question is how often, and by how much, on a real board.
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
// Une feuille de videau local `o` est tarifée à DefaultEfficiency(root) au
// lieu de DefaultEfficiency(o) : l'écart d'une feuille sur deux, nul quand la
// racine est centrée.
func TestMeasureCubeXLeafGap(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE_CUBEX") == "" {
		t.Skip("set BLUNDERDB_MEASURE_CUBEX to measure the cube-efficiency model gap")
	}
	corpus := cubeXCorpus(t)
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := embeddedPruneNetwork()
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
		s := newSearcherWith(cfg, net, prune)
		pos, err := FromDomain(d.pos)
		if err != nil {
			continue
		}
		probs, ok := s.Probs(&pos)
		if !ok {
			continue
		}

		// Money, per unit of cube: an OWNED leaf at 0.566 against 0.687, and
		// its mirror.
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
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := embeddedPruneNetwork()
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
		s := newSearcherWith(cfg, net, prune)
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
// Encadrement : la même recherche 2-ply à x = 0,566 partout puis x = 0,687
// partout. Empirique, pas une preuve : la pente en x (live − dead) change de
// signe selon p et la branche.
func TestMeasureCubeXSearchSensitivity(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE_CUBEX") == "" {
		t.Skip("set BLUNDERDB_MEASURE_CUBEX to measure the cube-efficiency model gap")
	}
	corpus := cubeXCorpus(t)
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := embeddedPruneNetwork()
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
			s := newSearcherWith(cfg, net, prune).WithWorkers(16)
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

// TestMeasureGateRedCasesAtDepth — MESURE 4 : rejoue les deux cas rouges de
// integration_gate_test.go (score [1,5], partie de Crawford) à 2-ply
// canonique, 2-ply sans élagage et 3-ply, et reporte le coût contre les
// équités de XG. La MET est déjà celle de XG (Kazaross-XG2).
func TestMeasureGateRedCasesAtDepth(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE_GATE_DEPTH") == "" {
		t.Skip("set BLUNDERDB_MEASURE_GATE_DEPTH to replay the gate's two red decisions at depth")
	}
	xg, err := ingest.MapXG("../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg")
	if err != nil {
		t.Fatal(err)
	}
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := embeddedPruneNetwork()
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
