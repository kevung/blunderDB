// SPDX-License-Identifier: MIT

package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// BenchmarkEvaluateBatchKernel is the figure #133 is judged on: the cost of one
// position when EvalBatchWidth of them go through the kernel together.
//
// It is deliberately the sibling of BenchmarkEvaluateBatch, which measures the
// same batch of DISTINCT positions through the scalar path — the two print
// ns/position and divide by each other. The reference to beat is the upstream
// C at 41 µs per position (gcc -O3, batch width 32); the plan's target is
// ≤ 60 µs.
//
// Run the pure-Go twin with BLUNDERDB_GAMMONNET_KERNEL=go to separate what the
// batched LAYOUT buys from what the vector instructions buy.
func BenchmarkEvaluateBatchKernel(b *testing.B) {
	benchmarkBatchKernel(b, EvalBatchWidth)
}

// BenchmarkEvaluateBatchKernelPartial measures the tail the search will hand
// over: one real position, the other seven lanes duplicating it. The per-lane
// figure is the same work, so this reports the cost of a batch that is 1/8
// useful — the price of a poorly filled batch, which decision D4 sized at
// 84.3 % filled in practice.
func BenchmarkEvaluateBatchKernelPartial(b *testing.B) {
	benchmarkBatchKernel(b, 1)
}

func benchmarkBatchKernel(b *testing.B, n int) {
	net, err := embeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	ev := NewEvaluator(net)

	// EvalBatchWidth positions that differ, so no two share an encoding, and
	// the sparse first layer sees a realistic union of non-zero features.
	var feats [EvalBatchWidth][NumFeatures]float32
	for i := range feats {
		p := bearoffBoard(24-i, 1+i)
		gn, err := FromDomain(&p)
		if err != nil {
			b.Fatal(err)
		}
		if !Encode(&gn, &feats[i]) {
			b.Fatal("encoding refused")
		}
	}
	var probs [EvalBatchWidth][NumOutputs]float32
	if err := ev.EvaluateBatch(&feats, n, &probs); err != nil {
		b.Fatal(err) // also warms the lazily allocated scratch
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ev.EvaluateBatch(&feats, n, &probs); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*n), "ns/position")
}

// BenchmarkEvaluateBatchKernelSiblings is the same measurement over the batch
// the SEARCH actually assembles: eight plays of one roll from one position.
//
// The distinction is not cosmetic. Siblings share most of their board, so the
// union of their non-zero features is ~32 of 196; eight unrelated bearoff
// boards union to 64. The first-layer shortcut is therefore worth about twice
// as much here as it is in the benchmark above — measuring it on unrelated
// boards made it look like a regression.
func BenchmarkEvaluateBatchKernelSiblings(b *testing.B) {
	net, err := embeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	ev := NewEvaluator(net)
	feats := siblingBatch(b)

	var probs [EvalBatchWidth][NumOutputs]float32
	if err := ev.EvaluateBatch(&feats, EvalBatchWidth, &probs); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ev.EvaluateBatch(&feats, EvalBatchWidth, &probs); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*EvalBatchWidth), "ns/position")
}

// siblingBatch returns the first EvalBatchWidth plays of the 3-1 opening —
// eight positions that differ by a few checkers, which is what one node of the
// search hands the kernel.
func siblingBatch(tb testing.TB) [EvalBatchWidth][NumFeatures]float32 {
	tb.Helper()
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		tb.Fatal(err)
	}
	root, err := FromDomain(&dp)
	if err != nil {
		tb.Fatal(err)
	}
	var gen Generator
	plays := make([]Play, 4096)
	n := gen.LegalPlays(&root, 3, 1, plays)
	if n < EvalBatchWidth {
		tb.Fatalf("the 3-1 opening yielded %d plays, need %d", n, EvalBatchWidth)
	}
	var out [EvalBatchWidth][NumFeatures]float32
	for l := 0; l < EvalBatchWidth; l++ {
		if !Encode(&plays[l].Result, &out[l]) {
			tb.Fatal("encoding refused a legal play")
		}
	}
	return out
}
