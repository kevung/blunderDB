// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// The kernel seam: which arithmetic path evaluates the network, and how wide a
// batch it consumes.
//
// Two paths exist. Evaluate (network.go) is the scalar one, one position at a
// time, and remains a valid entry point. EvaluateBatch below is the batched
// one: EvalBatchWidth positions, one per SIMD lane, run through denseBatch —
// either the generated AVX2 assembly or the pure-Go twin of the same layout.
//
// All of them are bound by ADR-0024: the batch vectorises over POSITIONS, not
// over the reduction. Each lane accumulates over j in ascending order, in
// float32, starting from the bias, with multiply and add kept separate. No
// FMA, no reassociation, no float64 accumulation. Bit-identity with the scalar
// path is the acceptance test, not the gold suites — those tolerate 1e-6 and
// would let an FMA through. kernel_identity_test.go is where that is proved.

// EvalBatchWidth is how many positions a batched evaluation consumes at once.
// It is a property of the kernel, not a tuning knob a caller may vary: a lane
// is a position, and a partial batch is filled by duplicating one, never with
// zeros, so no caller ever reasons about the tail.
//
// Eight is the AVX2 float32 lane count and the starting point measured by
// #145; NEON's four lanes make a group of eight two registers. The number is
// deliberately here rather than in the search: the search asks how many
// candidates it has, the kernel says how many it takes.
const EvalBatchWidth = 8

// KernelEnv names the environment variable that pins the arithmetic path.
// Accepted values are "go" and, on amd64, "avx2". It is a diagnosis and test
// knob, deliberately undocumented for users (decision D7 of the plan).
//
// A value that names a path this build or this CPU cannot provide is an error
// at load time, never a silent fallback: an unverified fast path is exactly
// how a silent wrong answer ships (acceptance criterion 4 of #133).
const KernelEnv = "BLUNDERDB_GAMMONNET_KERNEL"

// goKernelName is the pure-Go fallback, always available, and the reference
// the assembly is checked against bit for bit.
const goKernelName = "go"

// denseFunc evaluates one dense layer over a whole batch.
//
//	acc[n]      = bias[i]                        n ∈ [0, EvalBatchWidth)
//	acc[n]     += w[i*in+j] * act[j*W+n]         j ascending
//	out[i*W+n]  = relu ? max(acc[n], +0) : acc[n]
//
// w is row-major [outDim][in]; act and out are feature-major, one lane per
// position. Every implementation must produce the same bits.
type denseFunc func(w, bias, act, out []float32, in, outDim int, relu bool)

type denseKernel struct {
	name  string
	dense denseFunc
}

var goKernel = denseKernel{name: goKernelName, dense: denseGo}

var resolveKernelOnce = sync.OnceValues(func() (denseKernel, error) {
	return resolveKernel(os.Getenv(KernelEnv), acceleratedKernels())
})

// resolveKernel picks the arithmetic path. Empty request means "the fastest
// one this machine actually provides"; a named request is honoured or refused,
// never approximated.
func resolveKernel(requested string, accelerated []denseKernel) (denseKernel, error) {
	available := append(append([]denseKernel{}, accelerated...), goKernel)

	requested = strings.TrimSpace(strings.ToLower(requested))
	if requested == "" {
		return available[0], nil
	}
	for _, k := range available {
		if k.name == requested {
			return k, nil
		}
	}

	names := make([]string, 0, len(available))
	for _, k := range available {
		names = append(names, k.name)
	}
	return denseKernel{}, fmt.Errorf(
		"gammonnet: %s=%q is not available in this build on this CPU (available: %s); "+
			"an unavailable kernel is refused rather than silently replaced",
		KernelEnv, requested, strings.Join(names, ", "))
}

// KernelName is the arithmetic path in use. The probe prints it beside every
// timing so a measurement carries the code it measured. A misconfigured
// selector reports "invalid" here and a real error where it can be returned —
// Load and EvaluateBatch.
func KernelName() string {
	k, err := resolveKernelOnce()
	if err != nil {
		return "invalid"
	}
	return k.name
}

// KernelError reports a selector that names an unavailable path. It is nil in
// every ordinary run.
func KernelError() error {
	_, err := resolveKernelOnce()
	return err
}

// batchSlots is how many lanes a batch of n positions occupies — n rounded up
// to a whole number of batches. The gap between n and batchSlots(n) is the
// work a batched kernel would do and throw away, which is the number that
// decides whether the twenty-one rolls need grouping (decision D4 of the plan,
// and the open half of #146).
func batchSlots(n int) int {
	if n <= 0 {
		return 0
	}
	return ((n + EvalBatchWidth - 1) / EvalBatchWidth) * EvalBatchWidth
}

// EvaluateBatch runs the forward pass over EvalBatchWidth positions at once and
// writes each one's five post-processed probabilities into probs.
//
// features holds n encoded positions in its first n rows; n must be in
// [1, EvalBatchWidth]. The lanes beyond n are filled by DUPLICATING row n-1 —
// never with zeros. Two things follow, and both are deliberate: the caller
// hands over what it has and reads back the first n results without reasoning
// about the tail, and a duplicated lane is checkable against its twin, which
// is a test the zero-filled alternative cannot offer.
//
// The result of every lane is bit-identical to what Evaluate returns for the
// same features. That is the contract of ADR-0024 and the reason the kernel
// vectorises over positions rather than over the sum.
func (e *Evaluator) EvaluateBatch(features *[EvalBatchWidth][NumFeatures]float32, n int, probs *[EvalBatchWidth][NumOutputs]float32) error {
	if n < 1 || n > EvalBatchWidth {
		return fmt.Errorf("gammonnet: batch of %d positions, expected 1..%d", n, EvalBatchWidth)
	}
	if e.net.inputSize != NumFeatures {
		return fmt.Errorf("gammonnet: network takes %d features, this batch carries %d", e.net.inputSize, NumFeatures)
	}
	if err := e.ensureBatchScratch(); err != nil {
		return err
	}

	last := len(e.net.layers) - 1
	lay0 := &e.net.layers[0]

	// Dropping the features that are zero in every lane is exact in IEEE 754 —
	// acc + w×0.0 == acc — with ONE exception: acc == -0.0, where the skipped
	// +0.0 would have turned it into +0.0. The exception is neutralised by the
	// ReLU that follows, which maps both zeros to +0.0, so the shortcut is
	// taken only on a layer that has one. A single-layer network (its first
	// layer is its output layer) keeps every column.
	// kernel_identity_test.go proves both halves rather than assuming them.
	skipZeros := last > 0

	// Transpose to feature-major, and strip the dead columns in the same pass.
	// The 196-feature thermometer is ~80 % zeros — ~38 survive in union over a
	// batch — so this is where a fifth of the first layer goes.
	k := 0
	nz := e.nz[:0]
	for j := 0; j < NumFeatures; j++ {
		col := e.batchAct[k*EvalBatchWidth : k*EvalBatchWidth+EvalBatchWidth]
		nonzero := false
		for lane := 0; lane < EvalBatchWidth; lane++ {
			src := lane
			if src >= n {
				src = n - 1 // the tail duplicates the last real position
			}
			v := features[src][j]
			col[lane] = v
			if v != 0 {
				nonzero = true
			}
		}
		if nonzero || !skipZeros {
			nz = append(nz, int32(j))
			k++
		}
	}
	e.nz = nz

	// Compact the first layer's weights to the surviving columns. Indexing
	// w[nz[t]] indirectly inside the inner loop measured SLOWER than the dense
	// loop upstream (gn_infer_reference.c), and a vgatherdps is microcoded to
	// ~40-52 µops on Zen 3, so the columns are gathered once per batch into a
	// contiguous buffer and the same dense kernel runs over it with in = k.
	//
	// The gather is not free and the margin is thin: it moves out×k floats to
	// save (in−k)/in of the first layer, which is 19 % of the big network. On
	// the batch the search actually assembles — eight plays of one roll, union
	// ~32 of 196 — this is worth ~6 % of a batch. On eight UNRELATED boards
	// (union ~64) it is worth nothing and costs ~9 %. Measured 2026-09-02; if
	// the shortcut is ever revisited, measure it on siblings.
	cw := lay0.weight
	if k != lay0.in {
		cw = e.batchWeight[:lay0.out*k]
		for i := 0; i < lay0.out; i++ {
			row := lay0.weight[i*lay0.in : (i+1)*lay0.in]
			dst := cw[i*k : i*k+k]
			for t, j := range nz {
				dst[t] = row[j]
			}
		}
	}

	dense := e.kernel.dense
	dense(cw, lay0.bias, e.batchAct[:k*EvalBatchWidth], e.batchA, k, lay0.out, last > 0)

	in, out := e.batchA, e.batchB
	for l := 1; l <= last; l++ {
		lay := &e.net.layers[l]
		dense(lay.weight, lay.bias, in[:lay.in*EvalBatchWidth], out, lay.in, lay.out, l < last)
		in, out = out, in
	}

	// The output layer's sigmoid stays in Go: five neurons per position, and
	// math.Exp in float64 is what the scalar path does (network.go).
	for lane := 0; lane < EvalBatchWidth; lane++ {
		for i := 0; i < NumOutputs; i++ {
			probs[lane][i] = sigmoid(in[i*EvalBatchWidth+lane])
		}
		postprocess(&probs[lane])
	}
	return nil
}

// ensureBatchScratch allocates the batched scratch on first use. It is lazy on
// purpose: the compacted first-layer weights are as large as the layer itself
// (400 KB for the big network), and a caller that only ever evaluates one
// position at a time should not pay for them.
func (e *Evaluator) ensureBatchScratch() error {
	if e.batchA != nil {
		return e.kernelErr
	}
	k, err := resolveKernelOnce()
	if err != nil {
		e.kernelErr = err
		return err
	}
	e.kernel = k
	lay0 := &e.net.layers[0]
	e.batchA = make([]float32, e.net.widest*EvalBatchWidth)
	e.batchB = make([]float32, e.net.widest*EvalBatchWidth)
	e.batchAct = make([]float32, NumFeatures*EvalBatchWidth)
	e.batchWeight = make([]float32, lay0.out*lay0.in)
	e.nz = make([]int32, 0, NumFeatures)
	return nil
}
