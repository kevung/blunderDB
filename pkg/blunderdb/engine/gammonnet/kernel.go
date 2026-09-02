// SPDX-License-Identifier: MIT

package gammonnet

// The kernel seam: which arithmetic path evaluates the network, and how wide a
// batch it consumes. Today there is one path — the scalar Go loop in
// Evaluate — and the constants below exist so the measurements taken before
// the batched kernel lands are comparable with the ones taken after (#145,
// ADR-0024). A figure that does not name its kernel cannot be compared to
// another, and comparing them is the whole point of the probe.
//
// Whatever replaces the scalar path is bound by ADR-0024: the batch vectorises
// over POSITIONS, one per lane, each lane accumulating over j in ascending
// order in float32, multiply and add kept separate. Bit-identity with this
// scalar path is the acceptance test, not the gold suites — those tolerate
// 1e-6 and would let an FMA through.

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

// KernelName is the arithmetic path in use. The probe prints it beside every
// timing so a measurement carries the code it measured.
func KernelName() string { return "go" }

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
