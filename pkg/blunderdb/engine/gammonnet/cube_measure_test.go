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
// Pourquoi pas deux exécutions de `go test -bench` : gammonNet a lu LE MÊME
// poste à 10,6 %, 26 % ou 20,5 % le même après-midi sur la même machine, selon
// qu'il soustrayait deux exécutions consécutives, alternait trois passes, ou
// entrelaçait décision par décision (docs/mesures/2026-09-02-T85-videau-par-lot
// §1.3). Un facteur 2,5, sur un seuil d'abandon de 5 %. Le plancher de bruit
// d'une machine partagée est plus large que ce qu'on cherche à mesurer, donc
// la seule mesure exploitable est un rapport pris décision par décision, les
// deux configurations chronométrées à quelques millisecondes l'une de l'autre.
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

// measureCorpus est un corpus de décisions de contact — l'ouverture et des
// plateaux aléatoires — toutes au même score, videau centré. Le score compte :
// en argent le modèle §3 a une forme fermée et ne construit aucune chaîne
// d'enjeux, donc le poste y est nul (mesuré, pas supposé : T85 §1.2).
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
// Réserve, la même qu'en amont : allumer le videau change le meilleur coup,
// donc les deux configurations n'explorent pas exactement le même arbre. La
// part lue est celle du poste dans une décision, pas une soustraction de deux
// exécutions du même arbre — et c'est bien ce qu'on veut savoir avant de
// décider si le poste vaut d'être attaqué. La mesure de GAIN, elle
// (TestMeasureCubeBatch, ajouté avec le lot), compare deux codes qui rendent les mêmes bits et
// explorent donc l'arbre identique.
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

// TestMeasureCubeLift est la mesure de GAIN de la levée, sur la décision
// ENTIÈRE : le même arbre, les mêmes bits (TestLiftedSolveMatchesUnlifted),
// bissection non levée puis levée, entrelacées décision par décision dans le
// même processus.
func TestMeasureCubeLift(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE") == "" {
		t.Skip("poser BLUNDERDB_MEASURE pour mesurer ; ce test n'assère rien")
	}
	cases := measureCorpus(t, measureCorpusSize())
	repeats := measureRepeats()
	defer func() { cubeSolveLifted = true }()

	run := func(s *Searcher, c measureCase, lifted bool) time.Duration {
		cubeSolveLifted = lifted
		d, _ := timeCubeDecision(t, s, c)
		return d
	}

	var ratios []float64
	fmt.Printf("noyau %s\n", KernelName())
	fmt.Printf("%-4s %10s %10s %8s %12s\n", "cas", "non levé", "levé", "rapport", "valuations")
	for i, c := range cases {
		s, err := NewSearcher(measureConfig(c, true))
		if err != nil {
			t.Fatal(err)
		}
		var baseSum, liftSum time.Duration
		var vals uint64
		for r := 0; r < repeats; r++ {
			if r%2 == 0 {
				baseSum += run(s, c, false)
				liftSum += run(s, c, true)
			} else {
				liftSum += run(s, c, true)
				baseSum += run(s, c, false)
			}
			vals = s.CubeValuations()
		}
		a := baseSum / time.Duration(repeats)
		b := liftSum / time.Duration(repeats)
		ratio := float64(b) / float64(a)
		ratios = append(ratios, ratio)
		fmt.Printf("%-4d %10s %10s %8.3f %12d\n", i, a.Round(time.Millisecond), b.Round(time.Millisecond), ratio, vals)
	}
	m := medianOf(ratios)
	fmt.Printf("\nmédiane : levé / non levé = %.3f, soit %.1f %% de moins sur la décision entière (%d cas, %d répétitions)\n",
		m, 100*(1-m), len(cases), repeats)
}

// measureCorpusSize et measureRepeats laissent régler la mesure sans
// recompiler : une décision 2-ply au score coûte des centaines de
// millisecondes, et le corpus par défaut tient en quelques minutes.
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

// TestMeasureCubePost isole le POSTE et l'entrelace vraiment : scalaire et
// lot alternés à quelques microsecondes l'un de l'autre, dans la même boucle,
// dans le même processus, et un rapport pris par paire.
//
// Les bancs `go test -bench` ci-dessous n'alternent qu'à la granularité du
// banc — une seconde entière chacun — et sur une machine dont la charge
// dérive en quelques secondes, cela suffit à donner à l'un ce que l'autre a
// perdu. C'est la leçon T85 §1.3 transposée d'un cran plus bas.
//
// n est la taille de fratrie ; le rapport est rendu par taille, parce que
// c'est exactement la question qu'un lot pose (combien de voies faut-il pour
// couvrir la latence d'une division).
func TestMeasureCubePost(t *testing.T) {
	if os.Getenv("BLUNDERDB_MEASURE") == "" {
		t.Skip("poser BLUNDERDB_MEASURE pour mesurer ; ce test n'assère rien")
	}
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	ev := NewEvaluator(net)
	rng := rand.New(rand.NewSource(20260903))
	var features [NumFeatures]float32
	var dists [][NumOutputs]float32
	for len(dists) < 64 {
		b := randomBoard(rng, domain.White)
		p, err := FromDomain(&b)
		if err != nil || p.isOver() || !Encode(&p, &features) {
			continue
		}
		var probs [NumOutputs]float32
		if err := ev.Evaluate(features[:], &probs); err != nil {
			t.Fatal(err)
		}
		dists = append(dists, probs)
	}
	all := make([]*[NumOutputs]float32, len(dists))
	for i := range dists {
		all[i] = &dists[i]
	}
	out := make([]float64, len(dists))
	scratch := new(cubeScratch)
	state := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	x := DefaultEfficiency(CubeCentred)

	const reps = 400
	fmt.Printf("noyau %s, lot videau %d — poste isolé, entrelacé\n", KernelName(), CubeBatchWidth)
	fmt.Printf("       %8s %8s %8s   %6s %6s %6s   %6s\n",
		"avant", "levé", "lot", "levée", "lot", "les 2", "lot méd")
	fmt.Printf("%6s %8s %8s %8s   %6s %6s %6s   %6s\n",
		"voies", "ns/val", "ns/val", "ns/val", "×", "×", "×", "×")
	for _, n := range []int{2, 4, 8, 12, 16, 24, 32, 48, 64} {
		probs := all[:n]
		var ratios, baseNs, scalarNs, batchNs []float64
		for r := 0; r < reps; r++ {
			var z, a, b time.Duration
			// Les trois configurations tournent dans un ordre qui change à
			// chaque tour, pour qu'aucune ne soit systématiquement la
			// première (caches chauds) ni la dernière.
			switch r % 3 {
			case 0:
				z = timeUnliftedCube(t, probs, &state, x, out)
				a = timeScalarCube(t, probs, &state, x, out)
				b = timeBatchCube(t, scratch, probs, &state, x, out)
			case 1:
				a = timeScalarCube(t, probs, &state, x, out)
				b = timeBatchCube(t, scratch, probs, &state, x, out)
				z = timeUnliftedCube(t, probs, &state, x, out)
			default:
				b = timeBatchCube(t, scratch, probs, &state, x, out)
				z = timeUnliftedCube(t, probs, &state, x, out)
				a = timeScalarCube(t, probs, &state, x, out)
			}
			ratios = append(ratios, float64(b)/float64(a))
			baseNs = append(baseNs, float64(z.Nanoseconds())/float64(n))
			scalarNs = append(scalarNs, float64(a.Nanoseconds())/float64(n))
			batchNs = append(batchNs, float64(b.Nanoseconds())/float64(n))
		}
		m := medianOf(ratios)
		// Le minimum est l'estimateur robuste sous interférence : sur une
		// machine partagée, une exécution ne peut être que ralentie par le
		// voisin, jamais accélérée, donc le plus petit relevé approche le
		// coût non contendu. La médiane et le minimum doivent dire la même
		// chose ; s'ils divergent, c'est la charge qu'on mesure.
		z, a, b := minOf(baseNs), minOf(scalarNs), minOf(batchNs)
		fmt.Printf("%6d %8.0f %8.0f %8.0f   %6.2f %6.2f %6.2f   %6.2f\n",
			n, z, a, b, z/a, a/b, z/b, 1/m)
	}
}

func timeUnliftedCube(t *testing.T, probs []*[NumOutputs]float32, state *MatchState, x float64, out []float64) time.Duration {
	start := time.Now()
	for j := range probs {
		v, ok := valueUnlifted(probs[j], CubeCentred, state, x)
		if !ok {
			t.Fatal("le non levé a refusé")
		}
		out[j] = v
	}
	return time.Since(start)
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

func timeScalarCube(t *testing.T, probs []*[NumOutputs]float32, state *MatchState, x float64, out []float64) time.Duration {
	start := time.Now()
	for j := range probs {
		v, ok := Value(probs[j], CubeCentred, state, x)
		if !ok {
			t.Fatal("le scalaire a refusé")
		}
		out[j] = v
	}
	return time.Since(start)
}

func timeBatchCube(t *testing.T, scratch *cubeScratch, probs []*[NumOutputs]float32, state *MatchState, x float64, out []float64) time.Duration {
	start := time.Now()
	if !cubeValueBatch(scratch, probs, CubeCentred, state, x, out[:len(probs)]) {
		t.Fatal("le lot a refusé")
	}
	return time.Since(start)
}

// Le POSTE isolé — la valuation seule, sans la recherche autour.
//
// La mesure sur décision entière (TestMeasureCubeBatch) porte le bruit d'une
// machine partagée sur une décision qui coûte des centaines de
// millisecondes ; celle-ci porte le bruit sur une boucle de quelques
// microsecondes, et `-count` alterne les deux bancs dans le même processus.
// Les deux sont nécessaires : celle-ci dit si l'idée paye, celle-là dit ce
// qu'elle rend là où le poste vit.
//
// n est la taille de fratrie : 12 est ce que l'élagage k=12 laisse à la passe
// grand réseau, la plus fréquente ; 32 est une passe d'élagage typique.
func benchCubeValue(b *testing.B, n int, batched bool) {
	net, err := embeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	ev := NewEvaluator(net)
	rng := rand.New(rand.NewSource(20260903))
	var features [NumFeatures]float32
	dists := make([][NumOutputs]float32, 0, n)
	for len(dists) < n {
		p, err := FromDomain(&[]domain.Position{randomBoard(rng, domain.White)}[0])
		if err != nil || p.isOver() || !Encode(&p, &features) {
			continue
		}
		var probs [NumOutputs]float32
		if err := ev.Evaluate(features[:], &probs); err != nil {
			b.Fatal(err)
		}
		dists = append(dists, probs)
	}
	probs := make([]*[NumOutputs]float32, n)
	for i := range dists {
		probs[i] = &dists[i]
	}
	out := make([]float64, n)
	scratch := new(cubeScratch)
	state := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	x := DefaultEfficiency(CubeCentred)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if batched {
			if !cubeValueBatch(scratch, probs, CubeCentred, &state, x, out) {
				b.Fatal("le lot a refusé")
			}
			continue
		}
		for j := range probs {
			v, ok := Value(probs[j], CubeCentred, &state, x)
			if !ok {
				b.Fatal("le scalaire a refusé")
			}
			out[j] = v
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*n), "ns/valuation")
}

func BenchmarkCubeValueScalar12(b *testing.B) { benchCubeValue(b, 12, false) }
func BenchmarkCubeValueBatch12(b *testing.B)  { benchCubeValue(b, 12, true) }
func BenchmarkCubeValueScalar32(b *testing.B) { benchCubeValue(b, 32, false) }
func BenchmarkCubeValueBatch32(b *testing.B)  { benchCubeValue(b, 32, true) }
