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
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
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
	{"Trash/MatchDeleteRestore", checkTrashMatch},
	{"Direction/SetPair", checkWriteDirectionPair},
	{"Oracle/Analysis", checkOracleAnalysis},
	{"Oracle/Collection", checkOracleCollection},
	{"Oracle/Anki", checkOracleAnki},
	{"Oracle/TournamentAddMatch", checkOracleTournament},
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
	if err := s.Analyses().Save(cx(b), b, idA, &domain.PositionAnalysis{XGID: "INTRUDER"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s's Save over tenant %s's analysis: %v, want ErrNotFound", b, a, err)
	}
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
	if err := s.Analyses().Save(cx(b), b, idA, &domain.PositionAnalysis{XGID: "INTRUDER"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s saved an analysis on tenant %s's position, want ErrNotFound", b, a)
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
	if _, err := s.Comments().Add(cx(b), b, idA, "intruder's note"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s added a comment to tenant %s's position, want ErrNotFound", b, a)
	}
	if _, err := s.Comments().Upsert(cx(b), b, idA, "intruder's note"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s upserted a comment on tenant %s's position, want ErrNotFound", b, a)
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

	if err := s.Collections().AddPosition(cx(b), b, cidA, posB); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s added a position to tenant %s's collection, want ErrNotFound", b, a)
	}
	if err := s.Collections().AddPositions(cx(b), b, cidA, []int64{posB}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s added positions to tenant %s's collection, want ErrNotFound", b, a)
	}
	if err := s.Collections().AddPosition(cx(b), b, cidB, posA); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s put tenant %s's position in its own collection, want ErrNotFound", b, a)
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
	if err := s.Tournaments().AddMatch(cx(b), b, tid, mid); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("tenant %s put a match in tenant %s's tournament, want ErrNotFound", b, a)
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

// sameRefusal asserts that every probe was refused with ErrNotFound and with
// one message once the ids the caller passed are masked: a refusal that
// differs between "a's member", "a's non-member" and "absent" tells tenant b
// what tenant a holds.
func sameRefusal(t *testing.T, probes map[string]error, ids ...int64) {
	t.Helper()
	strs := make([]string, 0, len(ids))
	for _, id := range ids {
		strs = append(strs, strconv.FormatInt(id, 10))
	}
	sort.Slice(strs, func(i, j int) bool { return len(strs[i]) > len(strs[j]) })
	mask := func(msg string) string {
		for _, s := range strs {
			msg = strings.ReplaceAll(msg, s, "#")
		}
		return msg
	}
	first, firstLabel := "", ""
	for label, err := range probes {
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", label, err)
			continue
		}
		m := mask(err.Error())
		if firstLabel == "" {
			first, firstLabel = m, label
		} else if m != first {
			t.Errorf("refusals differ: %s says %q, %s says %q", firstLabel, first, label, m)
		}
	}
}

func checkOracleAnalysis(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	withAnalysis := savePos(t, cx, s, a, provenancePos(1))
	without := savePos(t, cx, s, a, provenancePos(2))
	if err := s.Analyses().Save(cx(a), a, withAnalysis, &domain.PositionAnalysis{XGID: "OWNER"}); err != nil {
		t.Fatalf("Save analysis(%s): %v", a, err)
	}
	absent := without + 1_000_000
	sameRefusal(t, map[string]error{
		"position with an analysis":    s.Analyses().Save(cx(b), b, withAnalysis, &domain.PositionAnalysis{XGID: "X"}),
		"position without an analysis": s.Analyses().Save(cx(b), b, without, &domain.PositionAnalysis{XGID: "X"}),
		"absent position":              s.Analyses().Save(cx(b), b, absent, &domain.PositionAnalysis{XGID: "X"}),
	}, withAnalysis, without, absent)
}

func checkOracleCollection(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	member := savePos(t, cx, s, a, provenancePos(1))
	nonMember := savePos(t, cx, s, a, provenancePos(2))
	cidA, err := s.Collections().Create(cx(a), a, "owner", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	if err := s.Collections().AddPosition(cx(a), a, cidA, member); err != nil {
		t.Fatalf("AddPosition(%s): %v", a, err)
	}
	cidB, err := s.Collections().Create(cx(b), b, "intruder", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", b, err)
	}
	absent := cidA + 1_000_000
	ids := []int64{member, nonMember, cidA, cidB, absent}
	sameRefusal(t, map[string]error{
		"AddPosition member":     s.Collections().AddPosition(cx(b), b, cidA, member),
		"AddPosition non-member": s.Collections().AddPosition(cx(b), b, cidA, nonMember),
		"AddPosition absent":     s.Collections().AddPosition(cx(b), b, absent, member),
	}, ids...)
	sameRefusal(t, map[string]error{
		"CopyPosition member":     s.Collections().CopyPosition(cx(b), b, cidA, member),
		"CopyPosition non-member": s.Collections().CopyPosition(cx(b), b, cidA, nonMember),
		"CopyPosition absent":     s.Collections().CopyPosition(cx(b), b, absent, member),
	}, ids...)
	sameRefusal(t, map[string]error{
		"MovePosition member":     s.Collections().MovePosition(cx(b), b, cidB, cidA, member),
		"MovePosition non-member": s.Collections().MovePosition(cx(b), b, cidB, cidA, nonMember),
		"MovePosition absent":     s.Collections().MovePosition(cx(b), b, cidB, absent, member),
	}, ids...)
	sameRefusal(t, map[string]error{
		"AddPositions member":     s.Collections().AddPositions(cx(b), b, cidA, []int64{member}),
		"AddPositions non-member": s.Collections().AddPositions(cx(b), b, cidA, []int64{nonMember}),
		"AddPositions absent":     s.Collections().AddPositions(cx(b), b, absent, []int64{member}),
	}, ids...)
	if got := tenantCollectionPositionIDs(t, cx, s, a, cidA); !slices.Equal(got, []int64{member}) {
		t.Errorf("tenant %s's collection holds %v after the probes, want [%d]", a, got, member)
	}
}

func checkOracleAnki(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	card := savePos(t, cx, s, a, provenancePos(1))
	nonCard := savePos(t, cx, s, a, provenancePos(2))
	deck, err := s.Anki().CreateDeck(cx(a), a, "owner", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck(%s): %v", a, err)
	}
	if err := s.Anki().SyncWithPositions(cx(a), a, deck, []int64{card}); err != nil {
		t.Fatalf("SyncWithPositions(%s): %v", a, err)
	}
	absent := deck + 1_000_000
	sameRefusal(t, map[string]error{
		"existing card": s.Anki().SyncWithPositions(cx(b), b, deck, []int64{card}),
		"no card":       s.Anki().SyncWithPositions(cx(b), b, deck, []int64{nonCard}),
		"absent deck":   s.Anki().SyncWithPositions(cx(b), b, absent, []int64{card}),
	}, card, nonCard, deck, absent)
}

func checkOracleTournament(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	tid, err := s.Tournaments().Create(cx(a), a, "owner", "2026-01-01", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	m := domain.Match{Player1Name: "Intruder", Player2Name: "Bob", MatchLength: 5}
	mid, err := s.Matches().Save(cx(b), b, &m)
	if err != nil {
		t.Fatalf("Save match(%s): %v", b, err)
	}
	absent := tid + 1_000_000
	sameRefusal(t, map[string]error{
		"a's tournament":    s.Tournaments().AddMatch(cx(b), b, tid, mid),
		"absent tournament": s.Tournaments().AddMatch(cx(b), b, absent, mid),
	}, tid, absent, mid)
}

// checkTrashMatch: a match deleted through the trash and restored by one
// tenant, beside the same match in the other's library. Neither tenant's
// delete or restore sees or touches the other's rows, and the restore puts
// the match and its purged position back under their own ids.
func checkTrashMatch(t *testing.T, cx func(scope string) context.Context, s storage.Storage, a, b string) {
	newMatch := func(scope string) (int64, int64) {
		pid := savePos(t, cx, s, scope, provenancePos(61))
		m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7}
		mid, err := s.Matches().Save(cx(scope), scope, &m)
		if err != nil {
			t.Fatalf("Save match(%s): %v", scope, err)
		}
		gid, err := s.Matches().CreateGame(cx(scope), scope, &domain.Game{MatchID: mid, GameNumber: 1})
		if err != nil {
			t.Fatalf("CreateGame(%s): %v", scope, err)
		}
		if _, err := s.Matches().CreateMove(cx(scope), scope, &domain.Move{GameID: gid, MoveNumber: 1,
			MoveType: "checker", PositionID: pid, Player: 1, Dice: [2]int32{3, 1}}); err != nil {
			t.Fatalf("CreateMove(%s): %v", scope, err)
		}
		return mid, pid
	}
	ma, pa := newMatch(a)
	mb, pb := newMatch(b)
	intact := func(when string) {
		t.Helper()
		if _, err := s.Matches().Get(cx(b), b, mb); err != nil {
			t.Errorf("tenant %s's match %s: %v", b, when, err)
		}
		if _, err := s.Positions().Load(cx(b), b, pb); err != nil {
			t.Errorf("tenant %s's position %s: %v", b, when, err)
		}
		if n, err := s.Trash().Count(cx(b), b); err != nil || n != 0 {
			t.Errorf("tenant %s's trash %s holds %d (%v), want 0", b, when, n, err)
		}
	}

	if _, err := trash.Match(cx(b), s, b, ma); err == nil {
		t.Errorf("tenant %s deleted tenant %s's match", b, a)
	}
	if _, err := s.Matches().Get(cx(a), a, ma); err != nil {
		t.Fatalf("tenant %s's match after tenant %s's delete: %v", a, b, err)
	}

	entry, err := trash.Match(cx(a), s, a, ma)
	if err != nil {
		t.Fatalf("trash.Match(%s): %v", a, err)
	}
	intact("after the other tenant's delete")
	if _, err := trash.Restore(cx(b), s, b, entry); err == nil {
		t.Errorf("tenant %s restored tenant %s's trash entry", b, a)
	}
	intact("after a refused restore")

	res, err := trash.Restore(cx(a), s, a, entry)
	if err != nil {
		t.Fatalf("Restore(%s): %v", a, err)
	}
	if res.ID != ma {
		t.Errorf("restored match has id %d, want its own %d", res.ID, ma)
	}
	if _, err := s.Positions().Load(cx(a), a, pa); err != nil {
		t.Errorf("tenant %s's purged position after restore, under its own id %d: %v", a, pa, err)
	}
	intact("after the other tenant's restore")
}
