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
// Evaluate (network.go) is the scalar path; EvaluateBatch runs EvalBatchWidth
// positions, one per SIMD lane, through the AVX2 assembly or its pure-Go twin.
//
// ADR-0024: the batch vectorises over POSITIONS, not over the reduction. Each
// lane accumulates over j in ascending order, in float32, from the bias, with
// multiply and add kept separate. No FMA, no reassociation, no float64
// accumulation. Bit-identity with the scalar path (kernel_identity_test.go)
// is the acceptance test; the gold suites tolerate 1e-6 and would let an FMA
// through.

// EvalBatchWidth is how many positions a batched evaluation consumes at once:
// a property of the kernel (the AVX2 float32 lane count), not a tuning knob.
const EvalBatchWidth = 8

// KernelEnv names the environment variable that pins the arithmetic path:
// "go" or, on amd64, "avx2". A diagnosis knob, undocumented for users. A path
// this build or CPU cannot provide is an error at load, never a silent
// fallback (ADR-0024).
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

// KernelName is the arithmetic path in use, printed beside every timing. A
// misconfigured selector reports "invalid" here and errors in Load and
// EvaluateBatch.
func KernelName() string {
	k, err := resolveKernelOnce()
	if err != nil {
		return "invalid"
	}
	return k.name
}

// kernelError reports a selector that names an unavailable path. It is nil in
// every ordinary run.
func kernelError() error {
	_, err := resolveKernelOnce()
	return err
}

// batchSlots is how many lanes a batch of n positions occupies — n rounded up
// to a whole number of batches.
func batchSlots(n int) int {
	if n <= 0 {
		return 0
	}
	return ((n + EvalBatchWidth - 1) / EvalBatchWidth) * EvalBatchWidth
}

// EvaluateBatch runs the forward pass over EvalBatchWidth positions at once and
// writes each one's five post-processed probabilities into probs.
//
// features holds n encoded positions in its first n rows, n in
// [1, EvalBatchWidth]. Lanes beyond n DUPLICATE row n-1, never zeros: the
// caller never reasons about the tail, and a duplicated lane is checkable
// against its twin. Every lane is bit-identical to Evaluate (ADR-0024).
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

	// Skipping features zero in every lane is exact in IEEE 754 except when
	// acc == -0.0 (the skipped +0.0 would make it +0.0); the following ReLU
	// maps both zeros to +0.0, so the shortcut is taken only on a layer that
	// has one. kernel_identity_test.go proves both halves.
	skipZeros := last > 0 && !e.noSkipZeros

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

	// Gather the first layer's surviving columns once per batch into a
	// contiguous buffer and run the same dense kernel with in = k: indirect
	// indexing in the inner loop is slower (gn_infer_reference.c), and
	// vgatherdps is microcoded. The margin is thin (~6 % on sibling plays,
	// a loss on unrelated boards): measure on siblings if revisited.
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

// ensureBatchScratch allocates the batched scratch (400 KB for the big
// network) on first use, so a scalar-only caller never pays for it.
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
