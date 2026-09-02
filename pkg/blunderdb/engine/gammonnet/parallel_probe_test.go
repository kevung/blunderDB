// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// coldCaches vide les tables d'évaluation du chercheur et de ses ouvriers.
// Une mesure enchaînant plusieurs décisions sur une table chaude ne mesure
// plus la décision mais le cache : 13 000 évaluations au premier tour, une
// poignée aux suivants.
func coldCaches(s *Searcher) {
	for i := range s.cache.entries {
		s.cache.entries[i].occupied = false
	}
	for _, w := range s.workers {
		coldCaches(w)
	}
}

// quantile est le q-ième centile d'un échantillon déjà trié (interpolation
// par index, suffisante ici : on rapporte p75 et p95, jamais des moyennes).
func quantile(sorted []time.Duration, q float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	i := int(q * float64(len(sorted)-1))
	return sorted[i]
}

// timeDecision chronomètre n décisions canoniques et rend p50/p75/p95.
func timeDecision(t *testing.T, s *Searcher, p Position, n int) (p50, p75, p95 time.Duration) {
	t.Helper()
	samples := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		coldCaches(s)
		pos := p
		start := time.Now()
		_, ok, err := s.BestPlay(&pos, 3, 1)
		d := time.Since(start)
		if err != nil || !ok {
			t.Fatalf("BestPlay: %v", err)
		}
		samples = append(samples, d)
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return quantile(samples, 0.50), quantile(samples, 0.75), quantile(samples, 0.95)
}

// TestProbeParallelSpeedup mesure ce que le parallélisme rend sur la décision
// 2-ply canonique (position d'ouverture, 3-1, DefaultConfig(2)). Fixer
// GOMAXPROCS au nombre de CŒURS PHYSIQUES : sur du calcul vectoriel qui sature
// les unités FP, le SMT n'apporte quasiment rien et « 16 threads » n'est pas
// « 16 cœurs » (P3).
//
//	GOMAXPROCS=8 BLUNDERDB_PROBE=1 go test -run TestProbeParallelSpeedup -v ./pkg/blunderdb/engine/gammonnet/
func TestProbeParallelSpeedup(t *testing.T) {
	if os.Getenv("BLUNDERDB_PROBE") == "" {
		t.Skip("set BLUNDERDB_PROBE to measure; this test asserts nothing")
	}
	p := openingPosition(t)
	reps := 15
	if v := os.Getenv("BLUNDERDB_PROBE_REPS"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &reps)
	}

	fmt.Printf("noyau %s, GOMAXPROCS=%d, %d cœurs logiques, %d répétitions\n\n",
		KernelName(), runtime.GOMAXPROCS(0), runtime.NumCPU(), reps)

	counts := []int{1, 2, 4, 6, 8, 12, 16}
	fmt.Printf("%10s %12s %12s %12s %10s %10s\n", "ouvriers", "p50", "p75", "p95", "×p75", "×p95")

	var base75, base95 time.Duration
	for _, n := range counts {
		s, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatal(err)
		}
		if n > 1 {
			s = s.WithWorkers(n)
		}
		p50, p75, p95 := timeDecision(t, s, p, reps)
		if n == 1 {
			base75, base95 = p75, p95
		}
		fmt.Printf("%10d %12s %12s %12s %9.2f× %9.2f×\n", n,
			p50.Round(time.Microsecond*100), p75.Round(time.Microsecond*100), p95.Round(time.Microsecond*100),
			float64(base75)/float64(p75), float64(base95)/float64(p95))
	}
}

// TestProbeRollCost mesure le coût de chacun des 21 lancers, pour fonder
// l'ordre LPT : le nombre de coups légaux générés à la racine du lancer, le
// nombre total d'évaluations que son sous-arbre déclenche, et son temps.
func TestProbeRollCost(t *testing.T) {
	if os.Getenv("BLUNDERDB_PROBE") == "" {
		t.Skip("set BLUNDERDB_PROBE to measure; this test asserts nothing")
	}
	var positions []Position
	for _, c := range buildCorpus() {
		if c.pos.isOver() {
			continue
		}
		positions = append(positions, c.pos)
		if len(positions) == 24 {
			break
		}
	}

	rolls := buildRolls()
	var plays, evals [NumRolls]uint64
	var elapsed [NumRolls]time.Duration

	s, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	gen := s.genAt(0)
	buf := make([]Play, MaxPlays)
	cands := make([]Candidate, MaxPlays)

	for _, p := range positions {
		for r := 0; r < NumRolls; r++ {
			pos := p
			n := gen.LegalPlays(&pos, int(rolls[r].d1), int(rolls[r].d2), buf)
			if n > 0 {
				plays[r] += uint64(n)
			}
			s.ResetCounters()
			coldCaches(s)
			start := time.Now()
			_, _ = s.oneRoll(&pos, 2, 0, r, nil, CubeCentred, cands)
			elapsed[r] += time.Since(start)
			e, pe, _ := s.Counters()
			evals[r] += e + pe
		}
	}

	type row struct {
		r     int
		plays uint64
		evals uint64
		d     time.Duration
	}
	rows := make([]row, 0, NumRolls)
	for r := 0; r < NumRolls; r++ {
		rows = append(rows, row{r, plays[r], evals[r], elapsed[r]})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].d > rows[j].d })

	fmt.Printf("%d positions, coût par lancer, trié par temps décroissant\n\n", len(positions))
	fmt.Printf("%3s %6s %10s %12s %12s\n", "idx", "dés", "coups", "évals", "temps")
	order := make([]int, 0, NumRolls)
	for _, x := range rows {
		fmt.Printf("%3d %6s %10d %12d %12s\n", x.r,
			fmt.Sprintf("%d-%d", rolls[x.r].d1, rolls[x.r].d2), x.plays, x.evals, x.d.Round(time.Millisecond))
		order = append(order, x.r)
	}
	fmt.Printf("\nordre par temps décroissant : %v\n", order)

	sort.SliceStable(rows, func(i, j int) bool { return rows[i].plays > rows[j].plays })
	order = order[:0]
	for _, x := range rows {
		order = append(order, x.r)
	}
	fmt.Printf("ordre par coups décroissants : %v\n", order)

	sort.SliceStable(rows, func(i, j int) bool { return rows[i].evals > rows[j].evals })
	order = order[:0]
	for _, x := range rows {
		order = append(order, x.r)
	}
	fmt.Printf("ordre par évals décroissantes : %v\n", order)
}

// TestProbeMachineCeiling mesure le plafond de la MACHINE sur ce travail-ci,
// sans aucune synchronisation : N décisions 2-ply indépendantes tournent en
// boucle, une par goroutine, et l'on compare le DÉBIT à celui d'une seule.
// C'est le parallélisme embarrassant — pas de barrière, pas de déséquilibre,
// rien de partagé que les poids en lecture seule. Tout ce qui manque à ce
// chiffre pour atteindre N est de la bande passante et de la FRÉQUENCE :
// sur un portable à budget thermique, la fréquence tous-cœurs est bien plus
// basse que le boost mono-cœur, et aucun ordonnancement ne récupère cela.
//
// La mesure est un débit en régime établi (chauffe puis fenêtre fixe), pas
// une latence : c'est le seul protocole où la fréquence a fini de descendre
// avant qu'on ne commence à compter.
func TestProbeMachineCeiling(t *testing.T) {
	if os.Getenv("BLUNDERDB_PROBE") == "" {
		t.Skip("set BLUNDERDB_PROBE to measure; this test asserts nothing")
	}
	p := openingPosition(t)
	window := 4 * time.Second
	warm := 2 * time.Second

	run := func(n int) float64 {
		searchers := make([]*Searcher, n)
		for i := range searchers {
			s, err := NewSearcher(DefaultConfig(2))
			if err != nil {
				t.Fatal(err)
			}
			searchers[i] = s
		}
		var done int64
		var counting int64
		stop := make(chan struct{})
		var wg sync.WaitGroup
		for _, s := range searchers {
			wg.Add(1)
			go func(s *Searcher) {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
					}
					coldCaches(s)
					pos := p
					_, _, _ = s.BestPlay(&pos, 3, 1)
					if atomic.LoadInt64(&counting) == 1 {
						atomic.AddInt64(&done, 1)
					}
				}
			}(s)
		}
		time.Sleep(warm)
		atomic.StoreInt64(&counting, 1)
		start := time.Now()
		time.Sleep(window)
		atomic.StoreInt64(&counting, 0)
		elapsed := time.Since(start)
		close(stop)
		wg.Wait()
		return float64(atomic.LoadInt64(&done)) / elapsed.Seconds()
	}

	fmt.Printf("noyau %s, GOMAXPROCS=%d, %d cœurs logiques\n\n", KernelName(), runtime.GOMAXPROCS(0), runtime.NumCPU())
	fmt.Printf("%12s %16s %12s\n", "goroutines", "décisions/s", "débit×")
	var base float64
	for _, n := range []int{1, 2, 4, 6, 8, 12, 16} {
		tp := run(n)
		if n == 1 {
			base = tp
		}
		fmt.Printf("%12d %16.2f %11.2f×\n", n, tp, tp/base)
	}
}

// TestProbeCacheHitRate est la mesure sur laquelle se décide l'étape 4 de la
// fiche F4 : faut-il un cache partagé entre ouvriers, ou le cache par ouvrier
// suffit-il ?
//
// La règle de décision de P3 est « mesurer d'abord, et si le partage ne fait
// pas monter le taux de hit d'au moins 5 points, garder le cache par ouvrier,
// trivialement déterministe et plus simple ». La mesure ne demande aucun code
// de production : une recherche SÉRIELLE consulte une seule table, et voit
// donc exactement la suite de recherches qu'un cache partagé verrait ; une
// recherche à N ouvriers en consulte N. La différence de taux de hit entre
// les deux EST le gain que le partage rendrait, à ceci près qu'elle le
// SURESTIME — la table sérielle a 65 536 entrées pour tout le travail, les
// N tables en ont N fois plus au total.
func TestProbeCacheHitRate(t *testing.T) {
	if os.Getenv("BLUNDERDB_PROBE") == "" {
		t.Skip("set BLUNDERDB_PROBE to measure; this test asserts nothing")
	}
	p := openingPosition(t)

	fmt.Printf("%10s %12s %12s %12s %10s\n", "ouvriers", "évals", "hits", "recherches", "taux")
	var base float64
	for _, n := range []int{1, 2, 4, 8, 16} {
		s, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatal(err)
		}
		if n > 1 {
			s = s.WithWorkers(n)
		}
		coldCaches(s)
		s.ResetCounters()
		pos := p
		if _, _, err := s.BestPlay(&pos, 3, 1); err != nil {
			t.Fatal(err)
		}
		evals, _, hits := s.Counters()
		lookups := evals + hits
		rate := 100 * float64(hits) / float64(lookups)
		if n == 1 {
			base = rate
		}
		fmt.Printf("%10d %12d %12d %12d %9.2f%% (%+.2f pt)\n", n, evals, hits, lookups, rate, rate-base)
	}
}
