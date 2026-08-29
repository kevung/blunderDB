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
// The order is not taken on trust. It is forced by the equity identity
// E = 2·p0 + p1 + p2 − p3 − p4 − 1, which is the algebraic consequence of this
// reading and of no other. Permute the five and the identity breaks.
const (
	PWin = iota
	PWinGammon
	PWinBackgammon
	PLoseGammon
	PLoseBackgammon
)

//go:embed strehl-prob5-512-512-256-128_v1.0.1_2026-08-27.bin
var embeddedWeights []byte

// Version identifies the gammonNet release this port reproduces. It is the
// string written into a stored analysis's engine field, and it is the same
// string gammonGo writes for the same configuration.
const Version = "gammonNet v1.0.1"

type layer struct {
	weight []float32 // row-major, out × in
	bias   []float32 // out
	in     int
	out    int
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

// Embedded returns the network compiled into the binary. The weights are the
// float32 reference artifact, not the float16 transport variant: a desktop
// application transports nothing, so the format that halves a download answers
// a constraint that does not exist here.
func Embedded() (*Network, error) {
	embeddedOnce.Do(func() {
		embeddedNet, embeddedErr = Load(embeddedWeights)
	})
	return embeddedNet, embeddedErr
}

// Load parses a BGNN weight file. It refuses anything it does not fully
// understand rather than filling in a default: a network handed an input it has
// never seen returns five perfectly plausible numbers and says nothing about it.
func Load(raw []byte) (*Network, error) {
	if len(raw) < 20 || string(raw[:4]) != "BGNN" {
		return nil, fmt.Errorf("gammonnet: not a BGNN weight file")
	}
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

	for _, d := range dims {
		in, out := d[0], d[1]
		w, err := readFloats(in * out)
		if err != nil {
			return nil, err
		}
		b, err := readFloats(out)
		if err != nil {
			return nil, err
		}
		n.layers = append(n.layers, layer{weight: w, bias: b, in: in, out: out})
	}
	if pos != len(raw) {
		return nil, fmt.Errorf("gammonnet: %d trailing bytes after the weights", len(raw)-pos)
	}
	return n, nil
}

// Evaluator holds the scratch buffers one evaluation needs. A Network is
// read-only and shared; an Evaluator is not, and each goroutine takes its own.
//
// gammonNet declares itself not thread-safe on purpose ("parallelism is by
// PROCESS ... never by thread"). Separating the immutable weights from the
// mutable scratch is what removes that constraint here rather than inheriting
// it.
type Evaluator struct {
	net *Network
	a   []float32
	b   []float32
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
// The arithmetic is deliberately float32 throughout, accumulating from the bias
// in ascending index order, exactly as the reference C does. The explicit
// float32 conversion around each product is not redundant: it forbids the
// compiler from contracting the multiply-add into an FMA, which would keep more
// precision than the reference and put cross-platform agreement out of reach on
// the architectures where Go fuses.
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
	return float32(1.0 / (1.0 + math.Exp(float64(-x))))
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

// MoneyEquity reduces a distribution to cubeless money equity.
//
//	  1·(p0−p1) + 2·(p1−p2) + 3·p2            winning single, gammon, backgammon
//	− 1·((1−p0)−p3) − 2·(p3−p4) − 3·p4        losing the same three ways
//	= 2·p0 + p1 + p2 − p3 − p4 − 1
func MoneyEquity(p *[NumOutputs]float32) float32 {
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
