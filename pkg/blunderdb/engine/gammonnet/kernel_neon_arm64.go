// SPDX-License-Identifier: MIT

//go:build arm64 && !purego

package gammonnet

// denseNEONReLU evaluates one dense layer over a batch of 8 positions and
// applies ReLU. Same arithmetic contract as denseGo: one lane per position,
// ascending-j float32 sums from the bias, FMUL then FADD, never FMLA
// (ADR-0024).
//
//go:noescape
func denseNEONReLU(w *float32, bias *float32, act *float32, out *float32, in int, outDim int)

// denseNEONLinear is denseNEONReLU without the activation.
//
//go:noescape
func denseNEONLinear(w *float32, bias *float32, act *float32, out *float32, in int, outDim int)

// denseNEON adapts the assembly to the denseFunc contract. As with AVX2, the
// two variants are separate functions so the inner loop carries no branch.
func denseNEON(w, bias, act, out []float32, in, outDim int, relu bool) {
	if in == 0 || outDim == 0 {
		// The inner loop is a do-while and would read one weight past the end
		// with in == 0 (an all-zero sparse first layer).
		denseGo(w, bias, act, out, in, outDim, relu)
		return
	}
	if relu {
		denseNEONReLU(&w[0], &bias[0], &act[0], &out[0], in, outDim)
		return
	}
	denseNEONLinear(&w[0], &bias[0], &act[0], &out[0], in, outDim)
}
