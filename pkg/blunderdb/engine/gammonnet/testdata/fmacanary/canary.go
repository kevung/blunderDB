// Package fmacanary holds one multiply-add the compiler fuses on every
// FMA-capable target: TestEngineEmitsNoFusedMultiplyAdd builds it beside the
// engine to prove its detector still recognises a fused instruction.
package fmacanary

// MulAdd is fused into one FMA instruction where the target has one.
func MulAdd(a, b, c float64) float64 {
	return a*b + c
}
