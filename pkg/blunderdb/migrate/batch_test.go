package migrate

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func openMemory(t *testing.T) storage.Storage {
	t.Helper()
	s, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}

// TestRunCopiesPositionsAcrossBatches copies more positions than one batch
// holds into a destination that already stores one of them, and checks each
// position arrives with its own analysis and its comments in the source's
// order: a batch boundary or a position merged by hash must not shift what
// belongs to whom.
func TestRunCopiesPositionsAcrossBatches(t *testing.T) {
	CheckCopiesAcrossBatches(t, openMemory(t))
}

// CheckCopiesAcrossBatches is TestRunCopiesPositionsAcrossBatches into dst, an
// empty destination of any backend.
func CheckCopiesAcrossBatches(t *testing.T, dst storage.Storage) {
	t.Helper()
	ctx := context.Background()
	src := openMemory(t)

	n := copyBatch + 7
	const signedAt, signedText, signedAuthor = 4, "watch the gap", "  Coach Martin "
	var zobrists []uint64
	for i := range n {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Score = [2]int{i % 40, i / 40}
		id, err := src.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
		zobrists = append(zobrists, engine.PopulatePositionColumns(&p).ZobristHash)
		if i%3 == 0 {
			a := &domain.PositionAnalysis{XGID: fmt.Sprintf("xgid-%d", i), AnalysisType: "XG Roller++"}
			if err := src.Analyses().Save(ctx, "", id, a); err != nil {
				t.Fatalf("Save analysis %d: %v", i, err)
			}
		}
		if i%250 == 1 {
			for _, text := range []string{"first", "second"} {
				if _, err := src.Comments().Add(ctx, "", id, fmt.Sprintf("%s %d", text, i)); err != nil {
					t.Fatalf("Add comment %d: %v", i, err)
				}
			}
		}
		if i == signedAt {
			// Signed with an author a normalisation would trim: the copy must
			// keep it byte for byte, or the note arrives signed but unverified.
			coach, err := issuance.NewIdentity("Coach")
			if err != nil {
				t.Fatal(err)
			}
			sig, err := issuance.SealComment(coach, signedText, signedAuthor, zobrists[i])
			if err != nil {
				t.Fatal(err)
			}
			if _, err := src.Comments().AddFrom(storage.WithCarriedComment(ctx, signedAuthor, sig), "", id, signedText, domain.CommentOriginUser); err != nil {
				t.Fatalf("Add signed comment: %v", err)
			}
		}
		if i == copyBatch+2 {
			// Already in the destination, with an analysis the copy replaces.
			pc := p
			pc.ID = 0
			held, err := dst.Positions().Save(ctx, "1", &pc)
			if err != nil {
				t.Fatalf("Save held: %v", err)
			}
			if err := dst.Analyses().Save(ctx, "1", held, &domain.PositionAnalysis{XGID: "stale"}); err != nil {
				t.Fatalf("Save held analysis: %v", err)
			}
		}
	}

	rep, err := Run(ctx, src, dst, "1", Options{OnConflict: "skip"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Positions != n || rep.Analyses != (n+2)/3 || rep.Comments != 2*((n+248)/250)+1 {
		t.Fatalf("report = %+v for %d positions", rep, n)
	}

	srcIDs, err := src.Positions().ExistsMany(ctx, "", zobrists)
	if err != nil {
		t.Fatal(err)
	}
	dstIDs, err := dst.Positions().ExistsMany(ctx, "1", zobrists)
	if err != nil {
		t.Fatal(err)
	}
	if len(dstIDs) != n {
		t.Fatalf("destination holds %d of the %d positions", len(dstIDs), n)
	}
	for i, z := range zobrists {
		want, err := src.Analyses().LoadMany(ctx, "", []int64{srcIDs[z]})
		if err != nil {
			t.Fatal(err)
		}
		got, err := dst.Analyses().LoadMany(ctx, "1", []int64{dstIDs[z]})
		if err != nil {
			t.Fatal(err)
		}
		if (want[srcIDs[z]] == nil) != (got[dstIDs[z]] == nil) ||
			(want[srcIDs[z]] != nil && want[srcIDs[z]].XGID != got[dstIDs[z]].XGID) {
			t.Fatalf("position %d: analysis %+v, want %+v", i, got[dstIDs[z]], want[srcIDs[z]])
		}
		wantC, err := src.Comments().ByPositions(ctx, "", []int64{srcIDs[z]})
		if err != nil {
			t.Fatal(err)
		}
		gotC, err := dst.Comments().ByPositions(ctx, "1", []int64{dstIDs[z]})
		if err != nil {
			t.Fatal(err)
		}
		texts := func(es []*domain.CommentEntry) []string {
			var out []string
			for _, e := range es {
				out = append(out, e.Text)
			}
			return out
		}
		if w, g := texts(wantC[srcIDs[z]]), texts(gotC[dstIDs[z]]); !slices.Equal(w, g) {
			t.Fatalf("position %d: comments %q, want %q", i, g, w)
		}
		if i == signedAt {
			if len(gotC[dstIDs[z]]) != 1 {
				t.Fatalf("signed position: %d comments, want 1", len(gotC[dstIDs[z]]))
			}
			c := gotC[dstIDs[z]][0]
			if c.Author != signedAuthor || c.Signer == nil || !c.Signer.Verified {
				t.Errorf("signed comment arrives with author %q, signer %+v; want %q, verified", c.Author, c.Signer, signedAuthor)
			}
		}
	}
}
