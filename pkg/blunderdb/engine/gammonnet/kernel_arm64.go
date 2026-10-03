// SPDX-License-Identifier: MIT

//go:build arm64 && !purego

package gammonnet

import "golang.org/x/sys/cpu"

// acceleratedKernels lists the vectorised paths this CPU provides. ASIMD is
// part of the arm64 baseline, but the gate is still read rather than assumed.
//
// The NEON kernel is opt-in: it runs only when BLUNDERDB_GAMMONNET_KERNEL=neon
// names it, and the default stays the pure-Go path until the identity test has
// passed on real Apple Silicon (ADR-0024: an unproven fast path is how a
// silent wrong answer ships). kernel_identity_test.go checks it on every arm64
// run whatever the selector says.
func acceleratedKernels() []denseKernel {
	if !cpu.ARM64.HasASIMD {
		return nil
	}
	return []denseKernel{{name: "neon", dense: denseNEON, optIn: true}}
}
