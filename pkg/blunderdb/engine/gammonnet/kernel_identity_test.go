// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The acceptance test of ADR-0024, and the reason the batched kernel is
// allowed to exist at all: every arithmetic path must return the SAME BITS,
// position by position, as the scalar loop in Evaluate.
//
// It is `==` and not a tolerance on purpose. parity_test.go and gold_test.go
// tolerate 1e-6, which is above the ~3e-9 an FMA contraction moves an equity
// by and above the 4.8e-7 a reassociation moves it by: they would let both
// through. A gammonNet analysis stored in a .db has to be reproducible on
// another machine, otherwise "stale" (AnalyzeStaleGammonNet) has no
// definition — so the threshold here is zero.

// batchKernels returns every arithmetic path this binary can run on this CPU,
// so the test compares the ACTIVE kernel against the pure-Go twin on the same
// machine rather than against a figure recorded elsewhere.
func batchKernels(t *testing.T) []denseKernel {
	t.Helper()
	active, err := resolveKernelOnce()
	if err != nil {
		t.Fatalf("kernel selection: %v", err)
	}
	kernels := append([]denseKernel{goKernel}, acceleratedKernels()...)
	t.Logf("active kernel %q; comparing %d path(s) against the scalar loop", active.name, len(kernels))
	return kernels
}

// batchEvaluator returns an evaluator pinned to one arithmetic path.
func batchEvaluator(t *testing.T, net *Network, k denseKernel) *Evaluator {
	t.Helper()
	ev := NewEvaluator(net)
	if err := ev.ensureBatchScratch(); err != nil {
		t.Fatalf("batch scratch: %v", err)
	}
	ev.kernel = k
	return ev
}

// bits renders a float32 as the thing the test actually compares, so a failure
// message shows the difference instead of two identical-looking decimals.
func bits(v float32) string {
	return fmt.Sprintf("%v (0x%08x)", v, math.Float32bits(v))
}

// compareBatch runs one batch of n positions through every kernel and against
// the scalar path, and reports the first bit that differs.
func compareBatch(t *testing.T, kernels []denseKernel, evs []*Evaluator, scalar *Evaluator, feats *[EvalBatchWidth][NumFeatures]float32, n int, label string) {
	t.Helper()
	var want [EvalBatchWidth][NumOutputs]float32
	for lane := 0; lane < n; lane++ {
		if err := scalar.Evaluate(feats[lane][:], &want[lane]); err != nil {
			t.Fatalf("%s: scalar lane %d: %v", label, lane, err)
		}
	}
	// The lanes past n duplicate lane n-1, so their expected answer is its answer.
	for lane := n; lane < EvalBatchWidth; lane++ {
		want[lane] = want[n-1]
	}

	var got [EvalBatchWidth][NumOutputs]float32
	for ki, k := range kernels {
		if err := evs[ki].EvaluateBatch(feats, n, &got); err != nil {
			t.Fatalf("%s: kernel %s: %v", label, k.name, err)
		}
		for lane := 0; lane < EvalBatchWidth; lane++ {
			for o := 0; o < NumOutputs; o++ {
				if got[lane][o] != want[lane][o] {
					t.Fatalf("%s: kernel %s, lane %d, output %d: batched %s, scalar %s",
						label, k.name, lane, o, bits(got[lane][o]), bits(want[lane][o]))
				}
			}
		}
	}
}

// TestKernelIdentityOnReferenceVectors runs the 2 000 vectors gammonNet's own C
// build produced through every kernel, in full batches, and demands the exact
// bits the scalar path returns.
func TestKernelIdentityOnReferenceVectors(t *testing.T) {
	ref := loadReference(t)
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	kernels := batchKernels(t)
	evs := make([]*Evaluator, len(kernels))
	for i, k := range kernels {
		evs[i] = batchEvaluator(t, net, k)
	}
	scalar := NewEvaluator(net)

	var feats [EvalBatchWidth][NumFeatures]float32
	full := 0
	for base := 0; base+EvalBatchWidth <= ref.count; base += EvalBatchWidth {
		for lane := 0; lane < EvalBatchWidth; lane++ {
			copy(feats[lane][:], ref.features[(base+lane)*NumFeatures:(base+lane+1)*NumFeatures])
		}
		compareBatch(t, kernels, evs, scalar, &feats, EvalBatchWidth,
			fmt.Sprintf("reference.bin[%d:%d]", base, base+EvalBatchWidth))
		full++
	}
	t.Logf("%d reference vectors, %d full batches, bit-identical", ref.count, full)
}

// TestKernelIdentityOnPartialBatches covers the tail the search will hand the
// kernel: batches of 1 to EvalBatchWidth-1 real positions, with the rest of
// the lanes filled by duplicating the last one. Two properties are checked at
// once — the real lanes match the scalar path, and a duplicated lane returns
// exactly what its twin returns, which is the check the zero-filled
// alternative could not offer.
func TestKernelIdentityOnPartialBatches(t *testing.T) {
	ref := loadReference(t)
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	kernels := batchKernels(t)
	evs := make([]*Evaluator, len(kernels))
	for i, k := range kernels {
		evs[i] = batchEvaluator(t, net, k)
	}
	scalar := NewEvaluator(net)

	// Every (n, offset) pair costs n scalar evaluations plus two batched ones;
	// the full sweep is the tag build's job, and -short (what every push runs)
	// takes a quarter of it. The property is not statistical — one differing
	// bit anywhere fails it — so the shorter sweep still tests the thing.
	span := 200
	if testing.Short() {
		span = 48
	}

	var feats [EvalBatchWidth][NumFeatures]float32
	var got [EvalBatchWidth][NumOutputs]float32
	for n := 1; n <= EvalBatchWidth; n++ {
		for base := 0; base+n <= span; base += n {
			for lane := 0; lane < n; lane++ {
				copy(feats[lane][:], ref.features[(base+lane)*NumFeatures:(base+lane+1)*NumFeatures])
			}
			// Fill the unused rows with something that is NOT the duplicated
			// position, so a kernel that read them instead of lane n-1 would
			// be caught.
			for lane := n; lane < EvalBatchWidth; lane++ {
				copy(feats[lane][:], ref.features[((base+lane)%ref.count)*NumFeatures:])
			}
			compareBatch(t, kernels, evs, scalar, &feats, n,
				fmt.Sprintf("reference.bin[%d:+%d]", base, n))

			for ki := range kernels {
				if err := evs[ki].EvaluateBatch(&feats, n, &got); err != nil {
					t.Fatal(err)
				}
				for lane := n; lane < EvalBatchWidth; lane++ {
					if got[lane] != got[n-1] {
						t.Fatalf("kernel %s: filler lane %d is not its twin %d: %v vs %v",
							kernels[ki].name, lane, n-1, got[lane], got[n-1])
					}
				}
			}
		}
	}
}

// TestKernelIdentityOnSearchPositions is the second corpus #133 asks for: not
// the C's synthetic vectors but the positions a real 1-ply search meets — the
// root plays of the opening position and their children, which is exactly the
// set a depth-1 search evaluates.
func TestKernelIdentityOnSearchPositions(t *testing.T) {
	positions := oneplyCorpus(t, 2000)
	t.Logf("%d distinct positions from a 1-ply expansion of the opening", len(positions))

	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := embeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	for _, n := range []*Network{net, prune} {
		kernels := batchKernels(t)
		evs := make([]*Evaluator, len(kernels))
		for i, k := range kernels {
			evs[i] = batchEvaluator(t, n, k)
		}
		scalar := NewEvaluator(n)

		var feats [EvalBatchWidth][NumFeatures]float32
		for base := 0; base+EvalBatchWidth <= len(positions); base += EvalBatchWidth {
			for lane := 0; lane < EvalBatchWidth; lane++ {
				feats[lane] = positions[base+lane]
			}
			compareBatch(t, kernels, evs, scalar, &feats, EvalBatchWidth,
				fmt.Sprintf("search corpus[%d:%d]", base, base+EvalBatchWidth))
		}
	}
}

// oneplyCorpus expands the opening position over all 21 rolls, then expands
// each result again, and returns at least want DISTINCT encoded positions.
func oneplyCorpus(t *testing.T, want int) [][NumFeatures]float32 {
	t.Helper()
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatal(err)
	}
	root, err := FromDomain(&dp)
	if err != nil {
		t.Fatal(err)
	}

	var gen Generator
	plays := make([]Play, 4096)
	seen := map[Position]bool{}
	out := make([][NumFeatures]float32, 0, want+64)

	add := func(p *Position) {
		if seen[*p] {
			return
		}
		seen[*p] = true
		var f [NumFeatures]float32
		if Encode(p, &f) {
			out = append(out, f)
		}
	}

	frontier := []Position{root}
	for len(out) < want && len(frontier) > 0 {
		next := make([]Position, 0, 512)
		for _, p := range frontier {
			for d1 := 1; d1 <= 6; d1++ {
				for d2 := d1; d2 <= 6; d2++ {
					n := gen.LegalPlays(&p, d1, d2, plays)
					if n <= 0 {
						continue
					}
					for i := 0; i < n; i++ {
						r := plays[i].Result
						if !seen[r] {
							next = append(next, r)
						}
						add(&r)
					}
				}
			}
			if len(out) >= want {
				break
			}
		}
		frontier = next
	}
	if len(out) < want {
		t.Fatalf("only %d positions generated, wanted %d", len(out), want)
	}
	return out
}

// TestKernelPreservesSubnormals proves what the plan refuses to assume: Go sets
// neither FTZ nor DAZ in MXCSR (nor FZ in FPCR on AArch64), so every path
// follows IEEE 754 down into the subnormals. A kernel that flushed them would
// return +0 here and still pass every gold suite.
func TestKernelPreservesSubnormals(t *testing.T) {
	const tiny = 1e-30  // normal
	const scale = 1e-15 // product 1e-45: subnormal, two ulps above zero

	w := []float32{tiny}
	bias := []float32{0}
	act := make([]float32, EvalBatchWidth)
	for n := range act {
		act[n] = scale
	}

	var reference []float32
	for _, k := range batchKernels(t) {
		out := make([]float32, EvalBatchWidth)
		k.dense(w, bias, act, out, 1, 1, false)
		for n := range out {
			if out[n] == 0 {
				t.Fatalf("kernel %s flushed a subnormal to zero in lane %d", k.name, n)
			}
			if math.Abs(float64(out[n])) >= math.SmallestNonzeroFloat32*(1<<23) {
				t.Fatalf("kernel %s lane %d: %s is not subnormal — the test no longer tests anything",
					k.name, n, bits(out[n]))
			}
		}
		if reference == nil {
			reference = out
			t.Logf("subnormal product survives as %s", bits(out[0]))
			continue
		}
		for n := range out {
			if out[n] != reference[n] {
				t.Fatalf("kernel %s lane %d: %s, reference %s", k.name, n, bits(out[n]), bits(reference[n]))
			}
		}
	}
}

// TestKernelNegativeZeroIsNeutralisedByReLU is the exception that makes the
// sparse first layer legitimate, stated as a test.
//
// Skipping a zero column is exact — acc + w×0.0 == acc — for every finite acc
// EXCEPT acc == -0.0, where the addition of +0.0 yields +0.0 and the skip
// leaves -0.0. The two differ in their sign bit. What makes the shortcut safe
// is the ReLU that follows: it maps both zeros to +0.0. This test shows both
// halves, so the day someone applies the shortcut to a layer without a ReLU
// the reason it was safe is written down next to the reason it stops being.
func TestKernelNegativeZeroIsNeutralisedByReLU(t *testing.T) {
	const in = 4
	negZero := float32(math.Float32frombits(0x8000_0000))

	bias := []float32{negZero}
	dense := make([]float32, in)              // four zero weights
	act := make([]float32, in*EvalBatchWidth) // four zero columns
	empty := []float32{}

	for _, k := range batchKernels(t) {
		withZeros := make([]float32, EvalBatchWidth)
		skipped := make([]float32, EvalBatchWidth)

		k.dense(dense, bias, act, withZeros, in, 1, true)
		k.dense(empty, bias, empty, skipped, 0, 1, true)
		for n := 0; n < EvalBatchWidth; n++ {
			if withZeros[n] != skipped[n] || math.Signbit(float64(skipped[n])) {
				t.Fatalf("kernel %s lane %d: dense %s, sparse %s — ReLU must give +0 both ways",
					k.name, n, bits(withZeros[n]), bits(skipped[n]))
			}
		}

		// And without the ReLU the two genuinely differ, which is why the
		// shortcut is confined to a layer that has one.
		k.dense(dense, bias, act, withZeros, in, 1, false)
		k.dense(empty, bias, empty, skipped, 0, 1, false)
		if math.Signbit(float64(withZeros[0])) || !math.Signbit(float64(skipped[0])) {
			t.Fatalf("kernel %s: expected +0 dense and -0 sparse without ReLU, got %s and %s",
				k.name, bits(withZeros[0]), bits(skipped[0]))
		}
	}
}

// TestDenseKernelsAgreeOnRandomLayers checks the kernels against each other one
// LAYER at a time — the intermediate activations #133 asks for, not only the
// five outputs — over shapes that exercise the tiling: output counts that are
// and are not multiples of the tile, and input widths of one and of many.
func TestDenseKernelsAgreeOnRandomLayers(t *testing.T) {
	kernels := batchKernels(t)
	if len(kernels) < 2 {
		t.Skip("only the pure-Go path is available on this machine; nothing to compare it against")
	}
	rng := rand.New(rand.NewSource(20260902))

	for _, shape := range [][2]int{
		{1, 1}, {1, 5}, {3, 4}, {5, 3}, {196, 5}, {196, 512}, {512, 512},
		{512, 256}, {256, 128}, {128, 5}, {38, 512}, {7, 6},
	} {
		in, out := shape[0], shape[1]
		w := randFloats(rng, in*out)
		bias := randFloats(rng, out)
		act := randFloats(rng, in*EvalBatchWidth)

		for _, relu := range []bool{false, true} {
			ref := make([]float32, out*EvalBatchWidth)
			kernels[0].dense(w, bias, act, ref, in, out, relu)
			for _, k := range kernels[1:] {
				got := make([]float32, out*EvalBatchWidth)
				k.dense(w, bias, act, got, in, out, relu)
				for idx := range ref {
					if got[idx] != ref[idx] {
						t.Fatalf("%dx%d relu=%v: kernel %s at row %d lane %d: %s, %s says %s",
							in, out, relu, k.name, idx/EvalBatchWidth, idx%EvalBatchWidth,
							bits(got[idx]), kernels[0].name, bits(ref[idx]))
					}
				}
			}
		}
	}
}

func randFloats(rng *rand.Rand, n int) []float32 {
	out := make([]float32, n)
	for i := range out {
		out[i] = float32(rng.NormFloat64())
	}
	return out
}

// TestResolveKernelRefusesWhatItCannotProvide: a selector that names a path
// this build or this CPU does not have is an error, never a quiet downgrade.
// An unverified fast path is how a silent wrong answer ships (#133, criterion 4).
func TestResolveKernelRefusesWhatItCannotProvide(t *testing.T) {
	fake := denseKernel{name: "avx2", dense: denseGo}

	if k, err := resolveKernel("", nil); err != nil || k.name != goKernelName {
		t.Fatalf("empty selector with no acceleration: got %q, %v", k.name, err)
	}
	if k, err := resolveKernel("", []denseKernel{fake}); err != nil || k.name != "avx2" {
		t.Fatalf("empty selector must take the fastest available: got %q, %v", k.name, err)
	}
	if k, err := resolveKernel("  GO ", []denseKernel{fake}); err != nil || k.name != goKernelName {
		t.Fatalf("explicit go: got %q, %v", k.name, err)
	}
	if _, err := resolveKernel("avx2", nil); err == nil {
		t.Fatal("a kernel this CPU does not provide was accepted")
	}
	if _, err := resolveKernel("neon", []denseKernel{fake}); err == nil {
		t.Fatal("a kernel this build does not contain was accepted")
	}
}
