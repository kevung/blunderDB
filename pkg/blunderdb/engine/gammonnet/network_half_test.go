// SPDX-License-Identifier: MIT

package gammonnet

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

// TestHalfToFloat32AllValues checks the widening on every one of the 65 536
// binary16 patterns against the value the format defines, computed in float64
// by a route that shares nothing with the bit shuffling: a finite half is
// mantissa·2^(exponent−25) with the implicit bit, or mantissa·2^−24 when
// subnormal, and that value is exactly representable in float32.
func TestHalfToFloat32AllValues(t *testing.T) {
	for i := 0; i < 1<<16; i++ {
		h := uint16(i)
		sign := uint32(h>>15) << 31
		exponent := int(h>>10) & 0x1f
		mantissa := int(h) & 0x3ff
		got := math.Float32bits(halfToFloat32(h))

		var want uint32
		switch exponent {
		case 0x1f:
			// Infinity, or a NaN whose payload moves to the top of the float32 mantissa.
			want = sign | 0x7f800000 | uint32(mantissa)<<13
		default:
			var v float64
			if exponent == 0 {
				v = math.Ldexp(float64(mantissa), -24)
			} else {
				v = math.Ldexp(float64(1024+mantissa), exponent-25)
			}
			if sign != 0 {
				v = -v
			}
			if sign != 0 && v == 0 {
				v = math.Copysign(0, -1)
			}
			f := float32(v)
			if float64(f) != v {
				t.Fatalf("half %#04x: %g is not a float32", h, v)
			}
			want = math.Float32bits(f)
		}
		if got != want {
			t.Fatalf("half %#04x widened to %#08x, want %#08x", h, got, want)
		}
	}
}

// TestLoadBGN6RefusesTrailingBytes: the file ends exactly after the last bias
// (upstream spec BGN6 §2); an extra byte is a shape the reader does not
// understand, not padding to trim.
func TestLoadBGN6RefusesTrailingBytes(t *testing.T) {
	if _, err := Load(embeddedPruneWeights); err != nil {
		t.Fatalf("the embedded BGN6 prune network does not load: %v", err)
	}
	over := append(bytes.Clone(embeddedPruneWeights), 0)
	if _, err := Load(over); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("a BGN6 file with one byte too many loaded (err = %v)", err)
	}
	if _, err := Load(embeddedPruneWeights[:len(embeddedPruneWeights)-1]); err == nil {
		t.Fatal("a truncated BGN6 file loaded")
	}
}

// TestEmbeddedWeightsAreBGN6 guards the choice of artifact: the binary carries
// the float16 transport files, and none of their weights is an infinity or a
// NaN, which a float16 overflow at packing would have produced.
func TestEmbeddedWeightsAreBGN6(t *testing.T) {
	for name, raw := range map[string][]byte{"main": embeddedWeights, "prune": embeddedPruneWeights} {
		if string(raw[:4]) != "BGN6" {
			t.Errorf("%s network: magic %q, want BGN6", name, raw[:4])
		}
	}
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	for _, lay := range net.layers {
		for _, w := range lay.weight {
			if math.IsNaN(float64(w)) || math.IsInf(float64(w), 0) {
				t.Fatalf("weight %v is not a finite value", w)
			}
		}
	}
}
