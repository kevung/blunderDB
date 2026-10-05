// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// L'instrument de mesure du poste videau — entrelacé, dans un seul processus.
//
// Pas deux exécutions de `go test -bench` : sur une machine partagée, le même
// poste varie d'un facteur 2,5 selon la méthode. La seule mesure exploitable
// est un rapport pris décision par décision, les deux configurations
// chronométrées à quelques millisecondes l'une de l'autre.
//
// Ces tests n'assertent rien : ils sont derrière BLUNDERDB_MEASURE, comme
// TestProbeDecisionCost l'est derrière BLUNDERDB_PROBE.

// measureCase est une décision du corpus de mesure : une position, un lancer,
// et l'état de match sous lequel la valuer.
type measureCase struct {
	pos    Position
	d1, d2 int
	state  MatchState
	owner  CubeOwner
}

// measureCorpus est un corpus de décisions de contact (l'ouverture et des
// plateaux aléatoires), au même score, videau centré : en argent le modèle
// §3 est en forme fermée et le poste est nul.
func measureCorpus(t *testing.T, n int) []measureCase {
	t.Helper()
	rng := rand.New(rand.NewSource(20260903))
	state := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}

	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatal(err)
	}
	dp.PlayerOnRoll = domain.White
	opening, err := FromDomain(&dp)
	if err != nil {
		t.Fatal(err)
	}

	cases := []measureCase{
		{pos: opening, d1: 3, d2: 1, state: state, owner: CubeCentred},
		{pos: opening, d1: 6, d2: 5, state: state, owner: CubeCentred},
		{pos: opening, d1: 4, d2: 2, state: state, owner: CubeCentred},
	}
	for len(cases) < n {
		onRoll := domain.White
		if len(cases)%2 == 1 {
			onRoll = domain.Black
		}
		b := randomBoard(rng, onRoll)
		p, err := FromDomain(&b)
		if err != nil || p.isOver() {
			continue
		}
		d1, d2 := 1+rng.Intn(6), 1+rng.Intn(6)
		if d2 < d1 {
			d1, d2 = d2, d1
		}
		owner := []CubeOwner{CubeCentred, CubeOwned, CubeOpponent}[len(cases)%3]
		cases = append(cases, measureCase{pos: p, d1: d1, d2: d2, state: state, owner: owner})
	}
	return cases
}

// measureConfig est la configuration canonique de la mesure : 2-ply, k=12,
// filtre (0,1,3) — celle que BenchmarkDecision2PlyMatch chronomètre, et celle
// que l'application fait tourner.
func measureConfig(c measureCase, useCube bool) SearchConfig {
	cfg := DefaultConfig(2)
	cfg.UseMatch = true
	cfg.Match = c.state
	if useCube {
		cfg.UseCube = true
		cfg.CubeOwner = c.owner
		cfg.CubeX = DefaultEfficiency(c.owner)
	}
	return cfg
}

func medianOf(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return 0.5 * (s[len(s)/2-1] + s[len(s)/2])
}

// timeCubeDecision chronomètre une décision et rend sa durée et le nombre de
// valuations de videau qu'elle a portées.
func timeCubeDecision(t *testing.T, s *Searcher, c measureCase) (time.Duration, uint64) {
	t.Helper()
	s.ResetCounters()
	pos := c.pos
	start := time.Now()
	_, ok, err := s.BestPlay(&pos, c.d1, c.d2)
	d := time.Since(start)
	if err != nil {
		t.Fatalf("recherche refusée : %v", err)
	}
	_ = ok // une position sans coup légal (une danse) est une mesure valable
	return d, s.CubeValuations()
}

// TestMeasureCubeShare est la mesure d'ENTRÉE du poste : ce que le videau
// coûte sur une décision au score, ici, sur cette machine, avec ce build.
//
// Réserve : allumer le videau change le meilleur coup, donc les deux
// configurations n'explorent pas le même arbre ; c'est la part du poste dans
// une décision. Les mesures de GAIN comparent deux codes aux mêmes bits.
func TestMeasureCubeShare(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE") == "" {
		t.Skip("poser BLUNDERDB_MEASURE pour mesurer ; ce test n'assère rien")
	}
	cases := measureCorpus(t, measureCorpusSize())
	repeats := measureRepeats()

	var shares, perVal []float64
	fmt.Printf("noyau %s\n", KernelName())
	fmt.Printf("%-4s %10s %10s %10s %10s %10s\n", "cas", "sans", "avec", "coût", "part", "ns/val")
	for i, c := range cases {
		cubeless, err := NewSearcher(measureConfig(c, false))
		if err != nil {
			t.Fatal(err)
		}
		cubeful, err := NewSearcher(measureConfig(c, true))
		if err != nil {
			t.Fatal(err)
		}
		var offSum, onSum time.Duration
		var vals uint64
		for r := 0; r < repeats; r++ {
			// L'ordre alterne : sur une machine qui dérive, mesurer
			// toujours A avant B donne à B le bénéfice des caches chauds.
			if r%2 == 0 {
				d, _ := timeCubeDecision(t, cubeless, c)
				offSum += d
				d, v := timeCubeDecision(t, cubeful, c)
				onSum, vals = onSum+d, v
			} else {
				d, v := timeCubeDecision(t, cubeful, c)
				onSum, vals = onSum+d, v
				d, _ = timeCubeDecision(t, cubeless, c)
				offSum += d
			}
		}
		off := offSum / time.Duration(repeats)
		on := onSum / time.Duration(repeats)
		cost := on - off
		share := 100 * float64(cost) / float64(on)
		ns := 0.0
		if vals > 0 {
			ns = float64(cost.Nanoseconds()) / float64(vals)
		}
		shares = append(shares, share)
		if vals > 0 {
			perVal = append(perVal, ns)
		}
		fmt.Printf("%-4d %10s %10s %10s %9.1f%% %10.0f\n", i, off.Round(time.Millisecond), on.Round(time.Millisecond), cost.Round(time.Millisecond), share, ns)
	}
	fmt.Printf("\nmédiane : part %.1f %%, %.0f ns par valuation (%d cas, %d répétitions)\n",
		medianOf(shares), medianOf(perVal), len(cases), repeats)
}

func measureCorpusSize() int { return envInt("BLUNDERDB_MEASURE_CASES", 12) }
func measureRepeats() int    { return envInt("BLUNDERDB_MEASURE_REPEATS", 3) }

func envInt(name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	n := 0
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return def
	}
	return n
}

func minOf(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	m := v[0]
	for _, x := range v {
		if x < m {
			m = x
		}
	}
	return m
}
