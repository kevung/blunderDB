// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"testing"
)

// TestBatchDropsDeadHiddenUnitsBitForBit holds the hidden-unit skip
// (compactLiveUnits) to the scalar path on both embedded networks, and checks
// it actually fires: a skip that never drops a unit would pass the identity
// suite while costing nothing and saving nothing.
func TestBatchDropsDeadHiddenUnitsBitForBit(t *testing.T) {
	ref := loadReference(t)
	alt, err := AlternateNetwork()
	if err != nil {
		t.Fatal(err)
	}
	def, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		net  *Network
	}{{NetworkName, def}, {AlternateNetworkName, alt}} {
		t.Run(c.name, func(t *testing.T) {
			kernels := batchKernels(t)
			evs := make([]*Evaluator, len(kernels))
			for i, k := range kernels {
				evs[i] = batchEvaluator(t, c.net, k)
			}
			scalar := NewEvaluator(c.net)
			lastHidden := c.net.layers[len(c.net.layers)-1].in
			dropped, batches := 0, 0
			var feats [EvalBatchWidth][NumFeatures]float32
			for base := 0; base+EvalBatchWidth <= ref.count && batches < 400; base += EvalBatchWidth {
				for lane := 0; lane < EvalBatchWidth; lane++ {
					copy(feats[lane][:], ref.features[(base+lane)*NumFeatures:(base+lane+1)*NumFeatures])
				}
				compareBatch(t, kernels, evs, scalar, &feats, EvalBatchWidth,
					fmt.Sprintf("%s reference16.bin[%d:%d]", c.name, base, base+EvalBatchWidth))
				dropped += lastHidden - len(evs[0].liveUnits)
				batches++
			}
			if dropped == 0 {
				t.Fatalf("no unit of the last hidden layer was ever dropped over %d batches", batches)
			}
			t.Logf("%d batches, %.1f %% of the last hidden layer dropped", batches,
				100*float64(dropped)/float64(batches*lastHidden))
		})
	}
}
