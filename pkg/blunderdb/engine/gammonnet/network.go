// SPDX-License-Identifier: MIT

package gammonnet

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

// Weight file format `BGNN`, as written by gammonNet's export_weights.py and
// read by the reference C loader (nn_eval.c). Little-endian throughout.
//
//	magic          4 bytes "BGNN"
//	numHidden      int32
//	inputSize      int32
//	activation     int32   0 relu · 1 sigmoid · 2 tanh · 3 leaky relu · 4 hard sigmoid
//	outputMode     int32   0 probability · 1 equity · 2 prob5
//	hiddenSizes    int32 × numHidden
//	per layer, numHidden+1 of them:
//	   weight      float32 × (out × in), row-major
//	   bias        float32 × out
//
// The output layer has five neurons in prob5 mode, one otherwise.
const (
	activationReLU  = 0
	outputModeProb5 = 2

	// NumOutputs is the width of the probability vector.
	NumOutputs = 5
)

// The five outputs, in the order the reference engine establishes. They are
// NESTED, not exclusive: PWinGammon counts backgammons too.
//
// The order is forced by the equity identity E = 2·p0 + p1 + p2 − p3 − p4 − 1,
// which holds for this reading and no other.
const (
	PWin = iota
	PWinGammon
	PWinBackgammon
	PLoseGammon
	PLoseBackgammon
)

//go:embed strehl-prob5-512-512-256-256_v1.8.1_2026-10-22.bin16
var embeddedWeights []byte

type layer struct {
	weight   []float32 // row-major, out × in; column-major, in × out, when colMajor
	bias     []float32 // out
	in       int
	out      int
	colMajor bool
}

// Network is a loaded set of weights. It is read-only once built and therefore
// safe to share between goroutines; the per-evaluation scratch lives in an
// Evaluator, not here.
type Network struct {
	layers     []layer
	inputSize  int
	activation int
	outputMode int
	widest     int
}

var (
	embeddedOnce sync.Once
	embeddedNet  *Network
	embeddedErr  error
)

// embeddedNetwork returns the network compiled into the binary: the float16
// transport artifact (BGN6), widened to float32 at load. It halves the weight
// bytes the binary carries; the evaluation itself stays float32.
func embeddedNetwork() (*Network, error) {
	embeddedOnce.Do(func() {
		embeddedNet, embeddedErr = Load(embeddedWeights)
	})
	return embeddedNet, embeddedErr
}

// Load parses a BGNN (float32) or BGN6 (float16) weight file. The two share
// one layout; BGN6 stores the weights as IEEE binary16 and keeps the biases in
// float32 (upstream spec BGN6). Both load to the same float32 Network.
//
// It refuses anything it does not fully understand rather than filling in a
// default: a network handed an input it has never seen returns five perfectly
// plausible numbers and says nothing about it.
func Load(raw []byte) (*Network, error) {
	if len(raw) < 20 || (string(raw[:4]) != "BGNN" && string(raw[:4]) != "BGN6") {
		return nil, fmt.Errorf("gammonnet: not a BGNN or BGN6 weight file")
	}
	half := string(raw[:4]) == "BGN6"
	pos := 4
	readInt := func() (int, error) {
		if pos+4 > len(raw) {
			return 0, fmt.Errorf("gammonnet: truncated header")
		}
		v := int(int32(binary.LittleEndian.Uint32(raw[pos:])))
		pos += 4
		return v, nil
	}

	numHidden, err := readInt()
	if err != nil {
		return nil, err
	}
	inputSize, err := readInt()
	if err != nil {
		return nil, err
	}
	activation, err := readInt()
	if err != nil {
		return nil, err
	}
	outputMode, err := readInt()
	if err != nil {
		return nil, err
	}

	if numHidden < 1 || numHidden > 8 {
		return nil, fmt.Errorf("gammonnet: bad hidden-layer count %d", numHidden)
	}
	if outputMode != outputModeProb5 {
		return nil, fmt.Errorf("gammonnet: output mode %d is not prob5; this port evaluates distributions only", outputMode)
	}
	if activation != activationReLU {
		return nil, fmt.Errorf("gammonnet: activation %d is not supported; only ReLU is ported", activation)
	}
	if inputSize != NumFeatures {
		return nil, fmt.Errorf("gammonnet: network takes %d features, this port encodes %d", inputSize, NumFeatures)
	}

	hidden := make([]int, numHidden)
	for i := range hidden {
		if hidden[i], err = readInt(); err != nil {
			return nil, err
		}
		if hidden[i] < 1 {
			return nil, fmt.Errorf("gammonnet: bad hidden size %d", hidden[i])
		}
	}

	n := &Network{
		inputSize:  inputSize,
		activation: activation,
		outputMode: outputMode,
		widest:     inputSize,
	}
	prev := inputSize
	dims := make([][2]int, 0, numHidden+1)
	for _, h := range hidden {
		dims = append(dims, [2]int{prev, h})
		prev = h
		if h > n.widest {
			n.widest = h
		}
	}
	dims = append(dims, [2]int{prev, NumOutputs})
	if NumOutputs > n.widest {
		n.widest = NumOutputs
	}

	readFloats := func(count int) ([]float32, error) {
		if pos+4*count > len(raw) {
			return nil, fmt.Errorf("gammonnet: truncated weights")
		}
		out := make([]float32, count)
		for i := range out {
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[pos:]))
			pos += 4
		}
		return out, nil
	}

	readHalves := func(count int) ([]float32, error) {
		if pos+2*count > len(raw) {
			return nil, fmt.Errorf("gammonnet: truncated weights")
		}
		out := make([]float32, count)
		for i := range out {
			out[i] = halfToFloat32(binary.LittleEndian.Uint16(raw[pos:]))
			pos += 2
		}
		return out, nil
	}
	readWeights := readFloats
	if half {
		readWeights = readHalves
	}

	for _, d := range dims {
		in, out := d[0], d[1]
		w, err := readWeights(in * out)
		if err != nil {
			return nil, err
		}
		b, err := readFloats(out)
		if err != nil {
			return nil, err
		}
		n.layers = append(n.layers, layer{weight: w, bias: b, in: in, out: out})
	}
	// The first layer is stored column-major: a batch keeps only the input
	// columns some position sets (kernel.go), and a column is then one
	// contiguous copy. The file and its fingerprint keep gammonNet's layout.
	if len(n.layers) > 0 {
		lay := &n.layers[0]
		t := make([]float32, len(lay.weight))
		for i := 0; i < lay.out; i++ {
			for j := 0; j < lay.in; j++ {
				t[j*lay.out+i] = lay.weight[i*lay.in+j]
			}
		}
		lay.weight = t
		lay.colMajor = true
	}
	if pos != len(raw) {
		return nil, fmt.Errorf("gammonnet: %d trailing bytes after the weights", len(raw)-pos)
	}
	// A kernel selector naming an unavailable path fails HERE, at the first
	// door into the package, never as a silent downgrade (ADR-0024).
	if err := kernelError(); err != nil {
		return nil, err
	}
	return n, nil
}

// halfToFloat32 widens an IEEE 754 binary16 to binary32, exactly: every finite
// half is a float32. A binary16 subnormal is a float32 normal, so it is
// renormalised rather than copied; ±0, infinities and NaN payloads carry over.
// It is gammonNet's from_half, bit for bit.
func halfToFloat32(h uint16) float32 {
	sign := uint32(h>>15) << 31
	exponent := uint32(h>>10) & 0x1f
	mantissa := uint32(h) & 0x3ff
	var bits uint32
	switch {
	case exponent == 0 && mantissa == 0:
		bits = sign
	case exponent == 0:
		exponent = 127 - 15 + 1
		for mantissa&0x400 == 0 {
			mantissa <<= 1
			exponent--
		}
		mantissa &= 0x3ff
		bits = sign | exponent<<23 | mantissa<<13
	case exponent == 0x1f:
		bits = sign | 0x7f800000 | mantissa<<13
	default:
		bits = sign | (exponent+127-15)<<23 | mantissa<<13
	}
	return math.Float32frombits(bits)
}

// Evaluator holds the scratch buffers one evaluation needs. A Network is
// read-only and shared; an Evaluator is not, and each goroutine takes its own.
//
// Separating immutable weights from mutable scratch lifts gammonNet's own
// "parallelism by process, never by thread" constraint.
type Evaluator struct {
	net *Network
	a   []float32
	b   []float32

	// Batched scratch (kernel.go), allocated on the first EvaluateBatch.
	kernel      denseKernel
	kernelErr   error
	batchA      []float32 // feature-major activations, widest × EvalBatchWidth
	batchB      []float32 // the alternate buffer, same size
	batchAct    []float32 // the input layer, transposed and stripped of its zero columns
	batchWeight []float32 // the first layer's weights, compacted to those columns
	nz          []int32   // the surviving feature indices, ascending

	// noSkipZeros éteint la compaction des colonnes nulles pour cet
	// évaluateur, pour la mesure seulement. Par évaluateur pour pouvoir
	// mesurer le grand et le petit réseau séparément.
	noSkipZeros bool
}

// NewEvaluator returns an evaluator over net. It allocates once; evaluating
// afterwards allocates nothing.
func NewEvaluator(net *Network) *Evaluator {
	return &Evaluator{
		net: net,
		a:   make([]float32, net.widest),
		b:   make([]float32, net.widest),
	}
}

// Evaluate runs the forward pass over pre-encoded features and writes the five
// post-processed probabilities into probs.
//
// The arithmetic is float32 throughout, accumulating from the bias in
// ascending index order, as the reference C does. The explicit float32
// conversion around each product is a fusion barrier: it forbids an FMA
// contraction on the architectures where Go fuses (ADR-0024).
func (e *Evaluator) Evaluate(features []float32, probs *[NumOutputs]float32) error {
	if len(features) != e.net.inputSize {
		return fmt.Errorf("gammonnet: %d features given, %d expected", len(features), e.net.inputSize)
	}
	in := features
	last := len(e.net.layers) - 1
	// The two scratch buffers alternate: a layer never reads and writes the
	// same one, and the input slice is never written to.
	useA := true

	for l := range e.net.layers {
		lay := &e.net.layers[l]
		out := e.a
		if !useA {
			out = e.b
		}
		if lay.colMajor {
			// Column by column, each out[i] still sums from its bias in
			// ascending j: the same float32 sequence as the row loop below.
			copy(out[:lay.out], lay.bias)
			for j := 0; j < lay.in; j++ {
				x := in[j]
				col := lay.weight[j*lay.out : (j+1)*lay.out]
				for i, w := range col {
					out[i] += float32(w * x)
				}
			}
			for i := 0; i < lay.out; i++ {
				if l < last {
					if !(out[i] > 0) { // ReLU
						out[i] = 0
					}
				} else {
					out[i] = sigmoid(out[i])
				}
			}
			in = out[:lay.out]
			useA = !useA
			continue
		}
		for i := 0; i < lay.out; i++ {
			sum := lay.bias[i]
			row := lay.weight[i*lay.in : (i+1)*lay.in]
			for j := 0; j < lay.in; j++ {
				sum += float32(row[j] * in[j])
			}
			if l < last {
				if sum > 0 { // ReLU
					out[i] = sum
				} else {
					out[i] = 0
				}
			} else {
				out[i] = sigmoid(sum)
			}
		}
		in = out[:lay.out]
		useA = !useA
	}

	copy(probs[:], in[:NumOutputs])
	postprocess(probs)
	return nil
}

func sigmoid(x float32) float32 {
	return float32(1.0 / (1.0 + portableExp(float64(-x))))
}

// postprocess clamps the nested-event inequalities, mirroring the reference's
// prob5_postprocess. The network can emit a distribution that violates them by
// a hair; the clamps make the vector coherent before anything reads it.
func postprocess(p *[NumOutputs]float32) {
	if p[PWinGammon] > p[PWin] {
		p[PWinGammon] = p[PWin] // P(wg) ≤ P(win)
	}
	lose := 1 - p[PWin]
	if p[PLoseGammon] > lose {
		p[PLoseGammon] = lose // P(lg) ≤ 1-P(win)
	}
	if p[PWinBackgammon] > p[PWinGammon] {
		p[PWinBackgammon] = p[PWinGammon] // P(wbg) ≤ P(wg)
	}
	if p[PLoseBackgammon] > p[PLoseGammon] {
		p[PLoseBackgammon] = p[PLoseGammon] // P(lbg) ≤ P(lg)
	}
}

// moneyEquity reduces a distribution to cubeless money equity.
//
//	  1·(p0−p1) + 2·(p1−p2) + 3·p2            winning single, gammon, backgammon
//	− 1·((1−p0)−p3) − 2·(p3−p4) − 3·p4        losing the same three ways
//	= 2·p0 + p1 + p2 − p3 − p4 − 1
func moneyEquity(p *[NumOutputs]float32) float32 {
	return 2*p[PWin] + p[PWinGammon] + p[PWinBackgammon] - p[PLoseGammon] - p[PLoseBackgammon] - 1
}

// EvaluatePosition encodes p and evaluates it in one step. It is the cold-path
// entry point; the search calls Encode and Evaluate separately so it can reuse
// its own feature buffer.
func (e *Evaluator) EvaluatePosition(p *Position, probs *[NumOutputs]float32) error {
	var features [NumFeatures]float32
	if !Encode(p, &features) {
		return fmt.Errorf("gammonnet: position is not structurally valid")
	}
	return e.Evaluate(features[:], probs)
}
