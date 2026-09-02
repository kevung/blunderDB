// SPDX-License-Identifier: MIT

//go:build amd64 && !purego

package gammonnet

import "golang.org/x/sys/cpu"

//go:generate sh -c "cd _asm && go run . -out ../kernel_avx2_amd64.s -stubs ../kernel_avx2_amd64.go -pkg gammonnet"

// acceleratedKernels lists the vectorised paths this CPU actually provides,
// fastest first. It never guesses: a path that is not detected is not offered,
// and a caller that names it gets an error rather than a substitute.
//
// The generated assembly only uses AVX1 encodings (VBROADCASTSS, VMULPS,
// VADDPS, VMAXPS on YMM), but the gate is HasAVX2 because "avx2" is the name
// ADR-0024 and the selector give this path, and a name that promises more than
// it checks is the kind of thing that goes wrong quietly. Every CPU shipped
// since 2013 satisfies it.
func acceleratedKernels() []denseKernel {
	if !cpu.X86.HasAVX2 {
		return nil
	}
	return []denseKernel{{name: "avx2", dense: denseAVX2}}
}

// denseAVX2 adapts the generated stubs to the denseFunc contract. The ReLU and
// linear variants are two functions rather than one with a flag so the inner
// loop carries no branch.
func denseAVX2(w, bias, act, out []float32, in, outDim int, relu bool) {
	if in == 0 {
		// The assembly's inner loop is a do-while: with in == 0 it would read
		// one weight that does not exist. A layer with no inputs is just its
		// bias, and only the sparse first layer can produce one (a feature
		// vector that is zero everywhere).
		denseGo(w, bias, act, out, in, outDim, relu)
		return
	}
	if relu {
		denseAVX2ReLU(&w[0], &bias[0], &act[0], &out[0], in, outDim)
		return
	}
	denseAVX2Linear(&w[0], &bias[0], &act[0], &out[0], in, outDim)
}
