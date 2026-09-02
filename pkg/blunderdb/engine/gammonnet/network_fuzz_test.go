// SPDX-License-Identifier: MIT

package gammonnet

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// encodeBGNN writes a BGNN file (network.go's format) with the given header
// and per-layer sizes, every weight and bias set by fill(i). A tiny network —
// one hidden layer of three neurons — is under two kilobytes, which is what a
// fuzz seed has to be; the embedded weights are two megabytes and would make
// every mutation cost a network's worth of parsing.
func encodeBGNN(numHidden, inputSize, activation, outputMode int32, hidden []int32, fill func(i int) float32) []byte {
	var buf bytes.Buffer
	buf.WriteString("BGNN")
	for _, v := range []int32{numHidden, inputSize, activation, outputMode} {
		_ = binary.Write(&buf, binary.LittleEndian, v)
	}
	for _, h := range hidden {
		_ = binary.Write(&buf, binary.LittleEndian, h)
	}
	prev := inputSize
	k := 0
	for _, out := range append(append([]int32{}, hidden...), NumOutputs) {
		for i := 0; i < int(prev)*int(out)+int(out); i++ {
			_ = binary.Write(&buf, binary.LittleEndian, math.Float32bits(fill(k)))
			k++
		}
		prev = out
	}
	return buf.Bytes()
}

func tinyBGNN() []byte {
	return encodeBGNN(1, NumFeatures, activationReLU, outputModeProb5, []int32{3}, func(i int) float32 { return float32(i%7) * 0.01 })
}

// FuzzLoadBGNN exercises the weight-file loader (#188): a binary format whose
// layer sizes are lengths read from the file itself, so a hostile or merely
// corrupt header can ask for any allocation. Load must refuse — never panic,
// never allocate what the file cannot contain — and a network it accepts
// must be one the evaluator can run: consistent layer chain, the port's input
// width, five outputs.
//
// The seeds cover each refusal in Load, one truncation per section of the
// file, and two networks it accepts.
func FuzzLoadBGNN(f *testing.F) {
	valid := tinyBGNN()
	f.Add(valid)
	f.Add(encodeBGNN(2, NumFeatures, activationReLU, outputModeProb5, []int32{4, 2}, func(int) float32 { return 0.5 }))
	f.Add([]byte{})
	f.Add([]byte("BGN"))
	f.Add([]byte("BGNN"))
	f.Add(valid[:19])                                         // header too short for Load's first check
	f.Add(valid[:20])                                         // header, no hidden sizes
	f.Add(valid[:24])                                         // header and one hidden size, no weights
	f.Add(valid[:len(valid)-1])                               // one byte short of the last bias
	f.Add(append(append([]byte{}, valid...), 0))              // one trailing byte
	f.Add(append([]byte("XGNN"), valid[4:]...))               // wrong magic
	f.Add(encodeBGNN(0, NumFeatures, 0, 2, nil, zero))        // no hidden layer
	f.Add(encodeBGNN(9, NumFeatures, 0, 2, nine(3), zero))    // too many hidden layers
	f.Add(encodeBGNN(1, 100, 0, 2, []int32{3}, zero))         // wrong input width
	f.Add(encodeBGNN(1, NumFeatures, 1, 2, []int32{3}, zero)) // sigmoid: not ported
	f.Add(encodeBGNN(1, NumFeatures, 0, 0, []int32{3}, zero)) // probability output: not ported
	f.Add(encodeBGNN(1, NumFeatures, 0, 2, []int32{0}, zero)) // empty hidden layer
	// A hidden size the header promises but the file cannot hold: the
	// allocation must be refused by the length check, not attempted.
	f.Add(append(valid[:20], []byte{0xff, 0xff, 0xff, 0x7f}...))
	f.Add(append(valid[:20], []byte{0x00, 0x00, 0x00, 0x80}...)) // negative hidden size
	// NaN and infinite weights are bit patterns the loader does not judge.
	f.Add(encodeBGNN(1, NumFeatures, 0, 2, []int32{2}, func(int) float32 { return float32(math.NaN()) }))
	f.Add(encodeBGNN(1, NumFeatures, 0, 2, []int32{2}, func(int) float32 { return float32(math.Inf(1)) }))

	pos := Position{}
	pos.Points[0], pos.Points[23] = 15, -15
	var feat [NumFeatures]float32
	Encode(&pos, &feat)

	f.Fuzz(func(t *testing.T, data []byte) {
		// A real weight file is two megabytes; what the fuzzer is for is the
		// header and the length checks, which a few kilobytes exercise
		// fully. The cap keeps the mutator from growing inputs whose only
		// cost is the minimisation of a megabyte.
		if len(data) > 64<<10 {
			t.Skip("a weight file this size is not what the fuzzer is for")
		}
		n, err := Load(data)
		if err != nil {
			if n != nil {
				t.Fatalf("Load returned both a network and an error: %v", err)
			}
			return
		}
		if n == nil {
			t.Fatal("Load returned neither a network nor an error")
		}

		// The accepted network is internally consistent.
		if len(n.layers) < 2 || len(n.layers) > 9 {
			t.Fatalf("%d layers accepted", len(n.layers))
		}
		if n.inputSize != NumFeatures || n.layers[0].in != NumFeatures {
			t.Fatalf("input width %d/%d accepted", n.inputSize, n.layers[0].in)
		}
		if last := n.layers[len(n.layers)-1]; last.out != NumOutputs {
			t.Fatalf("output width %d accepted", last.out)
		}
		prev := NumFeatures
		for i, l := range n.layers {
			if l.in != prev || l.out < 1 {
				t.Fatalf("layer %d: %d×%d does not chain from %d", i, l.in, l.out, prev)
			}
			if len(l.weight) != l.in*l.out || len(l.bias) != l.out {
				t.Fatalf("layer %d: %d weights, %d biases for %d×%d", i, len(l.weight), len(l.bias), l.in, l.out)
			}
			if l.out > n.widest {
				t.Fatalf("layer %d is wider (%d) than widest (%d)", i, l.out, n.widest)
			}
			prev = l.out
		}

		// And it runs: the scalar path and the batched kernel both take it
		// without panicking. The numbers are not judged — a NaN weight is a
		// NaN output, and the loader does not claim otherwise.
		ev := NewEvaluator(n)
		var probs [NumOutputs]float32
		if err := ev.Evaluate(feat[:], &probs); err != nil {
			t.Fatalf("Evaluate on an accepted network: %v", err)
		}
		var batchFeat [EvalBatchWidth][NumFeatures]float32
		var batchProbs [EvalBatchWidth][NumOutputs]float32
		batchFeat[0] = feat
		if err := ev.EvaluateBatch(&batchFeat, 1, &batchProbs); err != nil {
			t.Fatalf("EvaluateBatch on an accepted network: %v", err)
		}
	})
}

func zero(int) float32 { return 0 }

func nine(size int32) []int32 {
	out := make([]int32, 9)
	for i := range out {
		out[i] = size
	}
	return out
}

// The seeds Load accepts are accepted, and the ones it must refuse are —
// a plain test so a broken seed fails on every push, not only when the
// fuzzer runs.
func TestLoadBGNNSeeds(t *testing.T) {
	if _, err := Load(tinyBGNN()); err != nil {
		t.Fatalf("the tiny valid seed is refused: %v", err)
	}
	for name, raw := range map[string][]byte{
		"empty":          {},
		"short header":   tinyBGNN()[:19],
		"no weights":     tinyBGNN()[:24],
		"one byte short": tinyBGNN()[:len(tinyBGNN())-1],
		"trailing byte":  append(tinyBGNN(), 0),
		"wrong magic":    append([]byte("XGNN"), tinyBGNN()[4:]...),
		"no hidden":      encodeBGNN(0, NumFeatures, 0, 2, nil, zero),
		"sigmoid":        encodeBGNN(1, NumFeatures, 1, 2, []int32{3}, zero),
		"not prob5":      encodeBGNN(1, NumFeatures, 0, 0, []int32{3}, zero),
		"wrong width":    encodeBGNN(1, 100, 0, 2, []int32{3}, zero),
		"empty layer":    encodeBGNN(1, NumFeatures, 0, 2, []int32{0}, zero),
		"huge layer":     append(tinyBGNN()[:20], []byte{0xff, 0xff, 0xff, 0x7f}...),
	} {
		if n, err := Load(raw); err == nil || n != nil {
			t.Errorf("%s: accepted (%v)", name, err)
		}
	}
}
