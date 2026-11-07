// SPDX-License-Identifier: MIT

package gammonnet

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
)

// TestAlternateNetworkParityAgainstReference holds AlternateNetwork to the C
// reference on the feature vectors of reference16.bin. Upstream publishes no
// reference for the alternate network, so testdata/reference16_alternate.bin
// is the C's own answer, built by testdata/gold/reference_alternate.c — the
// same harness reproduces the release's reference16.bin byte for byte on the
// default network (README.md).
func TestAlternateNetworkParityAgainstReference(t *testing.T) {
	ref := loadReference(t)
	raw, err := os.ReadFile("testdata/reference16_alternate.bin")
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 12 || string(raw[:4]) != "GNRO" {
		t.Fatal("reference16_alternate.bin: unexpected magic")
	}
	count := int(binary.LittleEndian.Uint32(raw[4:]))
	if count != ref.count || int(binary.LittleEndian.Uint32(raw[8:])) != NumOutputs || len(raw) != 12+4*count*NumOutputs {
		t.Fatalf("reference16_alternate.bin: shape does not match reference16.bin")
	}
	net, err := AlternateNetwork()
	if err != nil {
		t.Fatal(err)
	}
	ev := NewEvaluator(net)
	var probs [NumOutputs]float32
	var worst float64
	for i := 0; i < count; i++ {
		if err := ev.Evaluate(ref.features[i*NumFeatures:(i+1)*NumFeatures], &probs); err != nil {
			t.Fatal(err)
		}
		for k := 0; k < NumOutputs; k++ {
			want := math.Float32frombits(binary.LittleEndian.Uint32(raw[12+4*(i*NumOutputs+k):]))
			worst = math.Max(worst, math.Abs(float64(probs[k])-float64(want)))
		}
	}
	t.Logf("%d positions — max|Δ| = %.3e", count, worst)
	if worst >= parityTolerance {
		t.Errorf("alternate network parity lost: max|Δ| = %.3e (tolerance %.0e)", worst, parityTolerance)
	}
}
