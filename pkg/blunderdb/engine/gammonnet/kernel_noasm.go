// SPDX-License-Identifier: MIT

//go:build !(amd64 || arm64) || purego

package gammonnet

// acceleratedKernels is empty on every architecture without an assembly
// kernel, so those builds run the pure-Go path — correct, simply not
// accelerated.
func acceleratedKernels() []denseKernel { return nil }
