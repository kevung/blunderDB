// Tenant write isolation cases: tenant a owns a row, tenant b writes to it by
// a's id, and a's data must come out unchanged. b's write may be refused (an
// error) or be a no-op; it must never land. Writes that attach a new row to a
// parent (a comment, a membership, a pair) must be refused outright: a row of
// b's hanging on a's parent would be invisible to a yet hold a's data, or take
// a unique slot a's own write needs.
//
// Like RunTenantIsolationTests this is for a multi-tenant backend only:
// SQLite has a single tenant, so b's write is a's write by construction and
// the contract holds there trivially, with nothing to run.
package storagetest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type tenantWriteCase struct {
	name string
	fn   func(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string)
}

var tenantWriteCases = []tenantWriteCase{
	{"Analysis/SaveOverForeign", checkWriteAnalysisSave},
	{"Analysis/FirstSaveOnForeignPosition", checkWriteAnalysisFirstSave},
	{"Analysis/Delete", checkWriteAnalysisDelete},
	{"Position/UpdateDelete", checkWritePosition},
	{"Comment/UpdateDelete", checkWriteComment},
	{"Comment/AddOnForeignPosition", checkWriteCommentAdd},
	{"Collection/UpdateDelete", checkWriteCollection},
	{"Collection/Membership", checkWriteCollectionMembership},
	{"Collection/AttachForeign", checkWriteCollectionAttach},
	{"Tournament/UpdateDelete", checkWriteTournament},
	{"Tournament/AddMatch", checkWriteTournamentAddMatch},
	{"Match/UpdateDelete", checkWriteMatch},
	{"Direction/SetPair", checkWriteDirectionPair},
}

// RunTenantWriteIsolationTests runs every by-id write against another
// tenant's row, on a fresh Storage from factory per case. a and b are two
// distinct tenant scopes; cx gives the context each scope's calls run under
// (a backend enforcing row-level security reads the tenant from it), nil for
// context.Background.
func RunTenantWriteIsolationTests(t *testing.T, factory func() storage.Storage, cx func(scope string) context.Context, a, b string) {
	t.Helper()
	if cx == nil {
		cx = func(string) context.Context { return context.Background() }
	}
	for _, tc := range tenantWriteCases {
		t.Run(tc.name, func(t *testing.T) {
			s := factory()
			defer s.Close()
			tc.fn(t, cx, s, a, b)
		})
	}
}

func savePos(t *testing.T, cx func(scope string) context.Context, s storage.Storage, scope string, p domain.Position) int64 {
	t.Helper()
	id, err := s.Positions().Save(cx(scope), scope, &p)
	if err != nil {
		t.Fatalf("Save position(%s): %v", scope, err)
	}
	return id
}

func checkWriteAnalysisSave(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	idA := savePos(t, cx, s, a, checkerPos())
	if err := s.Analyses().Save(cx(a), a, idA, &domain.PositionAnalysis{XGID: "OWNER"}); err != nil {
		t.Fatalf("Save analysis(%s): %v", a, err)
	}
	_ = s.Analyses().Save(cx(b), b, idA, &domain.PositionAnalysis{XGID: "INTRUDER"})
	got, err := s.Analyses().Load(cx(a), a, idA)
	if err != nil {
		t.Fatalf("Load analysis(%s): %v", a, err)
	}
	if got.XGID != "OWNER" {
		t.Errorf("tenant %s's analysis reads XGID %q after tenant %s wrote to it, want %q", a, got.XGID, b, "OWNER")
	}
}

func checkWriteAnalysisFirstSave(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	idA := savePos(t, cx, s, a, checkerPos())
	if err := s.Analyses().Save(cx(b), b, idA, &domain.PositionAnalysis{XGID: "INTRUDER"}); err == nil {
		t.Errorf("tenant %s saved an analysis on tenant %s's position, want a refusal", b, a)
	}
	// The intruder's attempt must not take the one-analysis slot the owner needs.
	if err := s.Analyses().Save(cx(a), a, idA, &domain.PositionAnalysis{XGID: "OWNER"}); err != nil {
		t.Fatalf("owner's Save after the intrusion: %v", err)
	}
	if got, err := s.Analyses().Load(cx(a), a, idA); err != nil || got.XGID != "OWNER" {
		t.Errorf("owner's analysis after the intrusion: %+v, %v", got, err)
	}
}

func checkWriteAnalysisDelete(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	idA := savePos(t, cx, s, a, checkerPos())
	if err := s.Analyses().Save(cx(a), a, idA, &domain.PositionAnalysis{XGID: "OWNER"}); err != nil {
		t.Fatalf("Save analysis(%s): %v", a, err)
	}
	_ = s.Analyses().Delete(cx(b), b, idA)
	if _, err := s.Analyses().Load(cx(a), a, idA); err != nil {
		t.Errorf("tenant %s's analysis after tenant %s deleted it by id: %v", a, b, err)
	}
}

func checkWritePosition(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	idA := savePos(t, cx, s, a, checkerPos())
	before, err := s.Positions().Load(cx(a), a, idA)
	if err != nil {
		t.Fatalf("Load(%s): %v", a, err)
	}
	edited := *before
	edited.Score = [2]int{before.Score[0] + 1, before.Score[1] + 2}
	_ = s.Positions().Update(cx(b), b, &edited)
	_ = s.Positions().Delete(cx(b), b, idA)
	got, err := s.Positions().Load(cx(a), a, idA)
	if err != nil {
		t.Fatalf("tenant %s's position after tenant %s's update and delete: %v", a, b, err)
	}
	if got.Score != before.Score {
		t.Errorf("tenant %s's position has score %v after tenant %s's update, want %v", a, got.Score, b, before.Score)
	}
}

func checkWriteComment(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	idA := savePos(t, cx, s, a, checkerPos())
	cid, err := s.Comments().Add(cx(a), a, idA, "owner's note")
	if err != nil {
		t.Fatalf("Add comment(%s): %v", a, err)
	}
	_ = s.Comments().Update(cx(b), b, cid, "intruder's note")
	_ = s.Comments().Delete(cx(b), b, cid)
	_ = s.Comments().DeleteForPosition(cx(b), b, idA)
	if got, err := s.Comments().Text(cx(a), a, idA); err != nil || got != "owner's note" {
		t.Errorf("tenant %s's comment after tenant %s's writes: %q, %v", a, b, got, err)
	}
}

func checkWriteCommentAdd(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	idA := savePos(t, cx, s, a, checkerPos())
	if _, err := s.Comments().Add(cx(b), b, idA, "intruder's note"); err == nil {
		t.Errorf("tenant %s added a comment to tenant %s's position, want a refusal", b, a)
	}
	if _, err := s.Comments().Upsert(cx(b), b, idA, "intruder's note"); err == nil {
		t.Errorf("tenant %s upserted a comment on tenant %s's position, want a refusal", b, a)
	}
}

func tenantCollectionPositionIDs(t *testing.T, cx func(scope string) context.Context, s storage.Storage, scope string, cid int64) []int64 {
	t.Helper()
	var ids []int64
	for p, err := range s.Collections().Positions(cx(scope), scope, cid, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("Positions(%s, %d): %v", scope, cid, err)
		}
		ids = append(ids, p.ID)
	}
	return ids
}

func checkWriteCollection(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	cid, err := s.Collections().Create(cx(a), a, "owner", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	_ = s.Collections().Update(cx(b), b, cid, "intruder", "")
	_ = s.Collections().Delete(cx(b), b, cid)
	got, err := s.Collections().Get(cx(a), a, cid)
	if err != nil {
		t.Fatalf("tenant %s's collection after tenant %s's update and delete: %v", a, b, err)
	}
	if got.Name != "owner" {
		t.Errorf("tenant %s's collection is named %q after tenant %s's update, want %q", a, got.Name, b, "owner")
	}
}

func checkWriteCollectionMembership(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	cid, err := s.Collections().Create(cx(a), a, "owner", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	p1 := savePos(t, cx, s, a, provenancePos(1))
	p2 := savePos(t, cx, s, a, provenancePos(2))
	if err := s.Collections().AddPositions(cx(a), a, cid, []int64{p1, p2}); err != nil {
		t.Fatalf("AddPositions(%s): %v", a, err)
	}
	want := []int64{p1, p2}
	cidB, err := s.Collections().Create(cx(b), b, "intruder", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", b, err)
	}

	_ = s.Collections().ReorderPositions(cx(b), b, cid, []int64{p2, p1})
	if got := tenantCollectionPositionIDs(t, cx, s, a, cid); !slices.Equal(got, want) {
		t.Errorf("after tenant %s's ReorderPositions, tenant %s's collection holds %v, want %v", b, a, got, want)
	}
	_ = s.Collections().RemovePosition(cx(b), b, cid, p1)
	_ = s.Collections().RemovePositions(cx(b), b, cid, []int64{p1, p2})
	if got := tenantCollectionPositionIDs(t, cx, s, a, cid); !slices.Equal(got, want) {
		t.Errorf("after tenant %s's removals, tenant %s's collection holds %v, want %v", b, a, got, want)
	}
	_ = s.Collections().MovePosition(cx(b), b, cid, cidB, p1)
	if got := tenantCollectionPositionIDs(t, cx, s, a, cid); !slices.Equal(got, want) {
		t.Errorf("after tenant %s's MovePosition, tenant %s's collection holds %v, want %v", b, a, got, want)
	}
}

func checkWriteCollectionAttach(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	cidA, err := s.Collections().Create(cx(a), a, "owner", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	posA := savePos(t, cx, s, a, checkerPos())
	cidB, err := s.Collections().Create(cx(b), b, "intruder", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", b, err)
	}
	posB := savePos(t, cx, s, b, checkerPos())

	if err := s.Collections().AddPosition(cx(b), b, cidA, posB); err == nil {
		t.Errorf("tenant %s added a position to tenant %s's collection, want a refusal", b, a)
	}
	if err := s.Collections().AddPositions(cx(b), b, cidA, []int64{posB}); err == nil {
		t.Errorf("tenant %s added positions to tenant %s's collection, want a refusal", b, a)
	}
	if err := s.Collections().AddPosition(cx(b), b, cidB, posA); err == nil {
		t.Errorf("tenant %s put tenant %s's position in its own collection, want a refusal", b, a)
	}
	if got := tenantCollectionPositionIDs(t, cx, s, a, cidA); len(got) != 0 {
		t.Errorf("tenant %s's collection holds %v after tenant %s's attempts, want none", a, got, b)
	}
}

func checkWriteTournament(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	tid, err := s.Tournaments().Create(cx(a), a, "owner", "2026-01-01", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	_ = s.Tournaments().Update(cx(b), b, tid, "intruder", "", "")
	_ = s.Tournaments().UpdateComment(cx(b), b, tid, "intruder")
	_ = s.Tournaments().Delete(cx(b), b, tid)
	got, err := s.Tournaments().Get(cx(a), a, tid)
	if err != nil {
		t.Fatalf("tenant %s's tournament after tenant %s's writes: %v", a, b, err)
	}
	if got.Name != "owner" || got.Comment != "" {
		t.Errorf("tenant %s's tournament is %q / %q after tenant %s's writes", a, got.Name, got.Comment, b)
	}
}

func checkWriteTournamentAddMatch(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	tid, err := s.Tournaments().Create(cx(a), a, "owner", "2026-01-01", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	m := domain.Match{Player1Name: "Intruder", Player2Name: "Bob", MatchLength: 5}
	mid, err := s.Matches().Save(cx(b), b, &m)
	if err != nil {
		t.Fatalf("Save match(%s): %v", b, err)
	}
	if err := s.Tournaments().AddMatch(cx(b), b, tid, mid); err == nil {
		t.Errorf("tenant %s put a match in tenant %s's tournament, want a refusal", b, a)
	}
}

func checkWriteMatch(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7}
	mid, err := s.Matches().Save(cx(a), a, &m)
	if err != nil {
		t.Fatalf("Save match(%s): %v", a, err)
	}
	_ = s.Matches().Update(cx(b), b, mid, "Intruder", "Intruder", "")
	_ = s.Matches().UpdateComment(cx(b), b, mid, "intruder")
	_ = s.Matches().SwapPlayers(cx(b), b, mid)
	_ = s.Matches().DeleteCascade(cx(b), b, mid)
	got, err := s.Matches().Get(cx(a), a, mid)
	if err != nil {
		t.Fatalf("tenant %s's match after tenant %s's writes: %v", a, b, err)
	}
	if got.Player1Name != "Alice" || got.Player2Name != "Bob" || got.Comment != "" {
		t.Errorf("tenant %s's match is %q vs %q / %q after tenant %s's writes", a, got.Player1Name, got.Player2Name, got.Comment, b)
	}
}

func checkWriteDirectionPair(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	tid, err := s.Tournaments().Create(cx(a), a, "owner", "2026-01-01", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	err = s.Directions().SetPair(cx(b), b, tid, "P1", []storage.PairMember{{Name: "Intruder"}})
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s's SetPair on tenant %s's tournament: %v, want ErrNotFound", b, a, err)
	}
	// The intruder's rows, had they landed, would hold the owner's seats.
	if err := s.Directions().SetPair(cx(a), a, tid, "P1", []storage.PairMember{{Name: "Anna"}}); err != nil {
		t.Fatalf("owner's SetPair after the intrusion: %v", err)
	}
	pairs, err := s.Directions().Pairs(cx(a), a, tid)
	if err != nil {
		t.Fatalf("Pairs(%s): %v", a, err)
	}
	if got := pairs["P1"]; len(got) != 1 || got[0].Name != "Anna" {
		t.Errorf("tenant %s's pair P1 is %+v, want [Anna]", a, got)
	}
}
