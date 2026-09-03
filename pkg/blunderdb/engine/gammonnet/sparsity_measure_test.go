// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// La sparsité, SÉPARÉE PAR RÉSEAU — poste 4 de la verticale ADR-0003.
//
// Ce portage a mesuré la compaction des colonnes nulles à ~6 % globalement, et
// −9 % sur des plateaux sans rapport (kernel.go, 2026-09-02). Il n'a jamais
// séparé le grand réseau du petit. gammonNet l'a fait (T89) et a trouvé que le
// petit représente 77 % des voies calculées mais 5 % du temps : l'optimiser
// vaut le cinquième d'optimiser le grand, alors que son registre le classait en
// priorité sur la foi du 78 %. Le chiffre a été retiré là-bas.
//
// Savoir ce qu'il vaut ICI demande le même banc séparé, et c'est ce fichier.
// Il n'assère rien.

// sparsityCase est la décision canonique, au score, avec le videau — la
// configuration que l'application fait tourner.
func sparsityConfig() SearchConfig {
	cfg := DefaultConfig(2)
	cfg.UseMatch = true
	cfg.Match = MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	cfg.UseCube = true
	cfg.CubeOwner = CubeCentred
	cfg.CubeX = DefaultEfficiency(CubeCentred)
	return cfg
}

// TestMeasureSparsityByNetwork chronomètre la MÊME décision sous quatre
// configurations de compaction, entrelacées, et rapporte ce que chaque réseau
// rend séparément.
//
// Les quatre configurations calculent exactement les mêmes bits — la
// compaction est exacte en IEEE 754 (acc + w×0,0 == acc), ce que
// kernel_identity_test.go prouve — donc l'arbre exploré est identique et l'A/B
// est propre.
func TestMeasureSparsityByNetwork(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE") == "" {
		t.Skip("poser BLUNDERDB_MEASURE pour mesurer ; ce test n'assère rien")
	}
	cases := measureCorpus(t, measureCorpusSize())
	repeats := measureRepeats()

	type variant struct {
		name             string
		bigOff, smallOff bool
	}
	variants := []variant{
		{"aucune", true, true},
		{"grand seul", false, true},
		{"petit seul", true, false},
		{"les deux", false, false},
	}

	sums := make([][]float64, len(variants))
	var evals, pruneEvals uint64
	for i := range sums {
		sums[i] = make([]float64, 0, len(cases))
	}

	for _, c := range cases {
		searchers := make([]*Searcher, len(variants))
		for v := range variants {
			s, err := NewSearcher(sparsityConfig())
			if err != nil {
				t.Fatal(err)
			}
			s.ev.noSkipZeros = variants[v].bigOff
			if s.pruneEv != nil {
				s.pruneEv.noSkipZeros = variants[v].smallOff
			}
			searchers[v] = s
		}
		acc := make([]time.Duration, len(variants))
		for r := 0; r < repeats; r++ {
			// L'ordre tourne : aucune configuration n'est systématiquement la
			// première (caches froids) ni la dernière.
			for k := 0; k < len(variants); k++ {
				v := (k + r) % len(variants)
				d, _ := timeCubeDecision(t, searchers[v], c)
				acc[v] += d
			}
		}
		base := float64(acc[0])
		for v := range variants {
			sums[v] = append(sums[v], float64(acc[v])/base)
		}
		e, pe, _ := searchers[len(variants)-1].Counters()
		evals, pruneEvals = e, pe
	}

	fmt.Printf("noyau %s, largeur de lot %d\n\n", KernelName(), EvalBatchWidth)
	fmt.Printf("%-14s %10s %10s\n", "compaction", "rapport", "gain")
	for v, va := range variants {
		m := medianOf(sums[v])
		fmt.Printf("%-14s %10.3f %9.1f%%\n", va.name, m, 100*(1-m))
	}
	total := evals + pruneEvals
	if total > 0 {
		fmt.Printf("\ndernière décision : %d évaluations grand réseau (%.1f %%), %d petit (%.1f %%)\n",
			evals, 100*float64(evals)/float64(total), pruneEvals, 100*float64(pruneEvals)/float64(total))
	}
	fmt.Printf("%d cas, %d répétitions ; la ligne « aucune » est la référence de chaque rapport\n",
		len(cases), repeats)
}

// TestMeasureNetworkCost chronomètre un lot plein à travers chaque réseau,
// entrelacé, pour dire ce qu'une voie coûte de part et d'autre. Croisé avec les
// évaluations COMPTÉES d'une décision, il donne la part de temps de chaque
// réseau — le chiffre que gammonNet a mesuré à 77 % des voies pour 5 % du
// temps, et que ce portage n'avait jamais séparé.
func TestMeasureNetworkCost(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE") == "" {
		t.Skip("poser BLUNDERDB_MEASURE pour mesurer ; ce test n'assère rien")
	}
	big, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	small, err := embeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	// Une fratrie réelle : les huit premiers coups légaux d'une ouverture,
	// c'est-à-dire ce que la recherche assemble effectivement dans un lot. Un
	// banc sur huit plateaux QUELCONQUES mesure autre chose — l'union de leurs
	// entrées actives est deux fois plus large (leçon consignée dans kernel.go
	// et dans l'ADR-0003 d'amont).
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatal(err)
	}
	dp.PlayerOnRoll = domain.White
	pos, err := FromDomain(&dp)
	if err != nil {
		t.Fatal(err)
	}
	gen := new(Generator)
	plays := make([]Play, MaxPlays)
	n := gen.LegalPlays(&pos, 3, 1, plays)
	if n < EvalBatchWidth {
		t.Fatalf("%d coups légaux, il en faut %d", n, EvalBatchWidth)
	}
	var feats [EvalBatchWidth][NumFeatures]float32
	for i := 0; i < EvalBatchWidth; i++ {
		p := plays[i].Result
		if !Encode(&p, &feats[i]) {
			t.Fatalf("encodage refusé pour le coup %d", i)
		}
	}

	evBig, evSmall := NewEvaluator(big), NewEvaluator(small)
	var probs [EvalBatchWidth][NumOutputs]float32
	run := func(e *Evaluator) time.Duration {
		start := time.Now()
		if err := e.EvaluateBatch(&feats, EvalBatchWidth, &probs); err != nil {
			t.Fatal(err)
		}
		return time.Since(start)
	}

	const reps = 2000
	var bigNs, smallNs []float64
	for r := 0; r < reps; r++ {
		var a, b time.Duration
		if r%2 == 0 {
			a, b = run(evBig), run(evSmall)
		} else {
			b, a = run(evSmall), run(evBig)
		}
		bigNs = append(bigNs, float64(a.Nanoseconds())/EvalBatchWidth)
		smallNs = append(smallNs, float64(b.Nanoseconds())/EvalBatchWidth)
	}
	mb, ms := medianOf(bigNs), medianOf(smallNs)
	fmt.Printf("noyau %s — coût d'une position dans un lot plein, entrelacé, %d paires\n", KernelName(), reps)
	fmt.Printf("grand réseau : %.0f ns (min %.0f)\npetit réseau : %.0f ns (min %.0f)\nrapport      : ×%.1f\n",
		mb, minOf(bigNs), ms, minOf(smallNs), mb/ms)

	// Le dénominateur COMPTÉ, sur une vraie décision.
	s, err := NewSearcher(sparsityConfig())
	if err != nil {
		t.Fatal(err)
	}
	p := pos
	if _, _, err := s.BestPlay(&p, 3, 1); err != nil {
		t.Fatal(err)
	}
	e, pe, _ := s.Counters()
	tBig, tSmall := float64(e)*mb, float64(pe)*ms
	fmt.Printf("\ndécision 2-ply k=12 (0,1,3) : %d évaluations grand, %d petit — %.1f %% des voies au petit\n",
		e, pe, 100*float64(pe)/float64(e+pe))
	fmt.Printf("part de temps estimée : grand %.1f %%, petit %.1f %%\n",
		100*tBig/(tBig+tSmall), 100*tSmall/(tBig+tSmall))
}
