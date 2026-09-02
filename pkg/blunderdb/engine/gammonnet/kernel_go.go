// SPDX-License-Identifier: MIT

package gammonnet

// denseGo is the portable kernel and, more importantly, the REFERENCE the
// generated assembly is checked against bit for bit on every runner
// (kernel_identity_test.go). It is not a degraded path: it is the statement of
// what the arithmetic must be, in a form a reader can check.
//
// Layout and order, both mandated by ADR-0024:
//
//   - act and out are feature-major, act[j*EvalBatchWidth+n]; lane n is
//     position n and nothing else. The vectorisation a SIMD twin performs is
//     over n, so the sum each lane takes is untouched by it.
//   - acc starts at bias[i] and takes the terms in ascending j, in float32.
//     No second accumulator, no tree reduction, no float64.
//   - float32(wj * col[n]) is written out rather than left implicit: the
//     conversion forbids the compiler from contracting the multiply-add into
//     an FMA, which Go does fuse on arm64. An FMA keeps more precision than
//     the reference C and would put cross-machine agreement out of reach —
//     see the same conversion in Evaluate (network.go).
func denseGo(w, bias, act, out []float32, in, outDim int, relu bool) {
	for i := 0; i < outDim; i++ {
		var acc [EvalBatchWidth]float32
		b := bias[i]
		for n := range acc {
			acc[n] = b
		}
		row := w[i*in : i*in+in]
		for j := 0; j < in; j++ {
			wj := row[j]
			col := act[j*EvalBatchWidth : j*EvalBatchWidth+EvalBatchWidth]
			for n := 0; n < EvalBatchWidth; n++ {
				acc[n] += float32(wj * col[n])
			}
		}
		dst := out[i*EvalBatchWidth : i*EvalBatchWidth+EvalBatchWidth]
		for n := 0; n < EvalBatchWidth; n++ {
			v := acc[n]
			// Written as `!(v > 0)` and not `v <= 0` so a NaN lands on +0,
			// which is what the scalar `if sum > 0` does and what VMAXPS with
			// zero as the second source does.
			if relu && !(v > 0) {
				v = 0
			}
			dst[n] = v
		}
	}
}
