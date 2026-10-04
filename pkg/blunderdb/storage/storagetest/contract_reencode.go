package storagetest

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// reencodeFixture saves n positions with an analysis each and returns their
// ids and the analyses as Load returns them.
func reencodeFixture(t *testing.T, s storage.Storage, n int) ([]int64, []*domain.PositionAnalysis) {
	t.Helper()
	ctx := context.Background()
	var ids []int64
	var loaded []*domain.PositionAnalysis
	for i := range n {
		p := checkerPos()
		p.Board.Points[3+i] = domain.Point{Checkers: 1, Color: domain.White}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		a := domain.PositionAnalysis{
			AnalysisType: "CheckerMove",
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "13/11 24/23", Equity: 0.123 + float64(i), PlayerWinChance: 54.32, OpponentWinChance: 45.68},
			}},
		}
		if err := s.Analyses().Save(ctx, "", id, &a); err != nil {
			t.Fatalf("Save analysis: %v", err)
		}
		got, err := s.Analyses().Load(ctx, "", id)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		loaded = append(loaded, got)
	}
	return ids, loaded
}

// testReencodeSkipsBinary: a library written by this release holds only
// binary blobs, and the pass finds nothing to do without decoding any.
func testReencodeSkipsBinary(t *testing.T, s storage.Storage) {
	reencodeFixture(t, s, 3)
	next, k, err := s.Analyses().ReencodeAnalyses(context.Background(), "", 0, 100)
	if err != nil || next != 0 || k != 0 {
		t.Fatalf("ReencodeAnalyses on binary blobs = %d, %d, %v; want nothing", next, k, err)
	}
}

// CheckReencodeUpgradesLegacy is the ReencodeAnalyses check that needs
// legacy blobs, which only a backend's own test can plant (no contract
// method writes bytes behind the codec): plant overwrites the analysis blob
// of a position with raw bytes. The pass rewrites raw-JSON and zlib rows in
// batches, keeps their content exactly, leaves an undecodable row alone, and
// a restarted pass revisits only that row.
func CheckReencodeUpgradesLegacy(t *testing.T, s storage.Storage, plant func(positionID int64, blob []byte)) {
	t.Helper()
	ctx := context.Background()
	ids, want := reencodeFixture(t, s, 4)
	legacy := func(a *domain.PositionAnalysis) []byte {
		js, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		return js
	}
	var zbuf bytes.Buffer
	zw := zlib.NewWriter(&zbuf)
	_, _ = zw.Write(legacy(want[1]))
	_ = zw.Close()
	plant(ids[0], legacy(want[0]))
	plant(ids[1], zbuf.Bytes())
	plant(ids[2], []byte("{not json"))
	// ids[3] stays binary.

	var next int64
	rewritten, batches := 0, 0
	for {
		n, k, err := s.Analyses().ReencodeAnalyses(ctx, "", next, 1)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			break
		}
		if n <= next {
			t.Fatalf("ReencodeAnalyses went back from %d to %d", next, n)
		}
		next, rewritten, batches = n, rewritten+k, batches+1
	}
	if rewritten != 2 || batches != 3 {
		t.Fatalf("pass rewrote %d rows in %d batches; want 2 in 3 (the binary row is never visited)", rewritten, batches)
	}
	for i := range 2 {
		got, err := s.Analyses().Load(ctx, "", ids[i])
		if err != nil {
			t.Fatal(err)
		}
		g, _ := json.Marshal(got)
		if w := legacy(want[i]); !bytes.Equal(g, w) {
			t.Fatalf("position %d after re-encoding:\n got %s\nwant %s", ids[i], g, w)
		}
	}
	n, k, err := s.Analyses().ReencodeAnalyses(ctx, "", 0, 100)
	if err != nil || n != ids[2] || k != 0 {
		t.Fatalf("restarted pass = %d, %d, %v; want only the undecodable row %d visited", n, k, err, ids[2])
	}
	if _, err := s.Analyses().Load(ctx, "", ids[2]); err == nil {
		t.Fatal("the undecodable row was rewritten")
	}
}
