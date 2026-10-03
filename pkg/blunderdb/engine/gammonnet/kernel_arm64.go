// SPDX-License-Identifier: MIT

//go:build arm64 && !purego

package gammonnet

import "golang.org/x/sys/cpu"

// acceleratedKernels lists the vectorised paths this CPU provides. ASIMD is
// part of the arm64 baseline, but the gate is still read rather than assumed.
//
// NEON is the default wherever ASIMD is present: kernel_identity_test.go has
// proved it bit-identical to the pure-Go fallback on Apple Silicon, and keeps
// checking it on every arm64 run (ADR-0024).
func acceleratedKernels() []denseKernel {
	if !cpu.ARM64.HasASIMD {
		return nil
	}
	return []denseKernel{{name: "neon", dense: denseNEON}}
}
