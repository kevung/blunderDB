package gammonnet

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"testing"
)

// The parity criterion of ADR-0011: this port must reproduce gammonNet's
// reference outputs. The reference is produced by the native C build and read
// here verbatim — recomputing both sides from a seed would establish nothing if
// the two implementations had drifted apart.
//
// The tolerance is gammonNet's own published criterion, 1e-6. The often-quoted
// 4.77e-07 is the worst deviation it MEASURED across seven platforms, not the
// threshold it set; using a measurement as a threshold would make the gate fail
// on a machine that is merely different rather than wrong.
const parityTolerance = 1e-6

// referenceMagic is 'GNRF' little-endian.
const referenceMagic = 0x46524E47

type reference struct {
	count       int
	numFeatures int
	numOutputs  int
	features    []float32
	outputs     []float32
}

func loadReference(t *testing.T) reference {
	t.Helper()
	raw, err := os.ReadFile("testdata/reference.bin")
	if err != nil {
		t.Fatalf("reference vectors: %v", err)
	}
	if len(raw) < 16 || binary.LittleEndian.Uint32(raw) != referenceMagic {
		t.Fatalf("reference.bin: unexpected magic")
	}
	r := reference{
		count:       int(int32(binary.LittleEndian.Uint32(raw[4:]))),
		numFeatures: int(int32(binary.LittleEndian.Uint32(raw[8:]))),
		numOutputs:  int(int32(binary.LittleEndian.Uint32(raw[12:]))),
	}
	readFloats := func(off, n int) []float32 {
		out := make([]float32, n)
		for i := range out {
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[off+4*i:]))
		}
		return out
	}
	featBytes := r.count * r.numFeatures * 4
	r.features = readFloats(16, r.count*r.numFeatures)
	r.outputs = readFloats(16+featBytes, r.count*r.numOutputs)
	return r
}

func TestNetworkParityAgainstReference(t *testing.T) {
	ref := loadReference(t)
	if ref.numFeatures != NumFeatures || ref.numOutputs != NumOutputs {
		t.Fatalf("reference shape %d×%d, port is %d×%d",
			ref.numFeatures, ref.numOutputs, NumFeatures, NumOutputs)
	}

	net, err := Embedded()
	if err != nil {
		t.Fatalf("embedded weights: %v", err)
	}
	ev := NewEvaluator(net)

	var worst float64
	worstAt := -1
	var probs [NumOutputs]float32
	for i := 0; i < ref.count; i++ {
		in := ref.features[i*NumFeatures : (i+1)*NumFeatures]
		if err := ev.Evaluate(in, &probs); err != nil {
			t.Fatalf("position %d: %v", i, err)
		}
		for k := 0; k < NumOutputs; k++ {
			d := math.Abs(float64(probs[k]) - float64(ref.outputs[i*NumOutputs+k]))
			if d > worst {
				worst, worstAt = d, i*NumOutputs+k
			}
		}
	}

	t.Logf("%d positions × %d outputs — max|Δ| = %s",
		ref.count, ref.numOutputs, fmt.Sprintf("%.3e", worst))
	if worst >= parityTolerance {
		t.Errorf("parity lost: max|Δ| = %.3e at position %d, output %d (tolerance %.0e)",
			worst, worstAt/NumOutputs, worstAt%NumOutputs, parityTolerance)
	}
}

// A weight file the port does not fully understand is refused rather than
// approximated: a network handed something it has never seen returns five
// perfectly plausible numbers and says nothing about it.
func TestLoadRefusesWhatItDoesNotUnderstand(t *testing.T) {
	good := embeddedWeights
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"empty", nil},
		{"bad magic", append([]byte("BGN6"), good[4:64]...)},
		{"truncated weights", good[:len(good)-8]},
		{"trailing bytes", append(append([]byte{}, good...), 0, 0, 0, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Load(tc.raw); err == nil {
				t.Fatal("accepted a weight file it should have refused")
			}
		})
	}
}
