// SPDX-License-Identifier: MIT

//go:build !amd64 || purego

package gammonnet

// acceleratedKernels is empty on every architecture without a generated
// kernel, so those builds run the pure-Go path — correct, simply not
// accelerated.
//
// arm64 included: avo has no arm64 back end, and hand-written NEON shipped
// unverified would risk a silent wrong answer (ADR-0024). A NEON kernel plugs
// in here by returning one more entry.
func acceleratedKernels() []denseKernel { return nil }
