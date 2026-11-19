// Contract cases for the reader floor a match's result raises: an older
// binary would add matches whose result nothing derives later, so the first
// known result a write stores raises the floor to
// domain.MatchResultReaderVersion, in that write, on every path (ADR-0114).
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// resultMatch is a finished match carrying its result.
func resultMatch() domain.Match {
	return domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		FinalScore: &[2]int32{7, 4}, Winner: domain.WinnerPlayer1}
}

// requireFloor fails unless s records want.
func requireFloor(t *testing.T, s storage.Storage, when, want string) {
	t.Helper()
	if got := readerFloor(t, s); got != want {
		t.Fatalf("floor %s = %q, want %q", when, got, want)
	}
}

// requireRefusedByOlder checks that a binary speaking the schema before
// domain.MatchResultReaderVersion refuses s, and that this one opens it.
func requireRefusedByOlder(t *testing.T, s storage.Storage) {
	t.Helper()
	floor := readerFloor(t, s)
	if err := storage.CheckReaderFloorAs(floor, domain.CheckerSourcesReaderVersion); !errors.Is(err, storage.ErrDatabaseTooNew) {
		t.Errorf("a %s reader on floor %q: err = %v, want ErrDatabaseTooNew", domain.CheckerSourcesReaderVersion, floor, err)
	}
	if err := storage.CheckReaderFloor(floor); err != nil {
		t.Errorf("this binary on floor %q: %v", floor, err)
	}
}

// saveBare stores a match whose result is unknown, which raises nothing.
func saveBare(t *testing.T, s storage.Storage) (int64, domain.Match) {
	t.Helper()
	bare := domain.Match{Player1Name: "Carol", Player2Name: "Dan", MatchLength: 5}
	id, err := s.Matches().Save(context.Background(), "", &bare)
	if err != nil {
		t.Fatal(err)
	}
	requireFloor(t, s, "after a match without result", domain.MinReaderVersion)
	return id, bare
}

// testMatchResultRaisesFloorOnSave: Save of a match with a result, the path
// an import and a migration to PostgreSQL take.
func testMatchResultRaisesFloorOnSave(t *testing.T, s storage.Storage) {
	requireFloor(t, s, "of a fresh database", domain.MinReaderVersion)
	saveBare(t, s)
	m := resultMatch()
	if _, err := s.Matches().Save(context.Background(), "", &m); err != nil {
		t.Fatal(err)
	}
	requireFloor(t, s, "after the first result", domain.MatchResultReaderVersion)
	requireRefusedByOlder(t, s)
}

// testMatchResultRaisesFloorOnReplaceHeader: a header restated without its
// result raises nothing; with one, it raises the floor.
func testMatchResultRaisesFloorOnReplaceHeader(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	id, bare := saveBare(t, s)
	bare.Event = "Restated"
	if err := s.Matches().ReplaceHeader(ctx, "", id, &bare); err != nil {
		t.Fatal(err)
	}
	requireFloor(t, s, "after a header without result", domain.MinReaderVersion)
	bare.FinalScore, bare.Winner = &[2]int32{5, 2}, domain.WinnerPlayer1
	if err := s.Matches().ReplaceHeader(ctx, "", id, &bare); err != nil {
		t.Fatal(err)
	}
	requireFloor(t, s, "after a header with result", domain.MatchResultReaderVersion)
	requireRefusedByOlder(t, s)
}

// testMatchResultRaisesFloorOnReinstate: a match put back under its own id
// with a result raises the floor.
func testMatchResultRaisesFloorOnReinstate(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	id, _ := saveBare(t, s)
	got, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Matches().DeleteCascade(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	back := *got
	back.FinalScore, back.Winner = &[2]int32{5, 1}, domain.WinnerPlayer1
	if err := s.Matches().Reinstate(ctx, "", &back); err != nil {
		t.Fatal(err)
	}
	requireFloor(t, s, "after a reinstated result", domain.MatchResultReaderVersion)
}

// testMatchResultFloorAfterSources: the floor never goes down, whichever of
// the two raises comes first.
func testMatchResultFloorAfterSources(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	mergeSaved(t, s, "", posID, sourceAnalysis("XG", "XG Roller++", 0.30, 0.20, 0.10))
	mergeSaved(t, s, "", posID, sourceAnalysis("GNUbg", "2-ply", 0.25, 0.28))
	requireFloor(t, s, "after two sources", domain.CheckerSourcesReaderVersion)
	m := resultMatch()
	if _, err := s.Matches().Save(ctx, "", &m); err != nil {
		t.Fatal(err)
	}
	requireFloor(t, s, "after sources then a result", domain.MatchResultReaderVersion)
}

// testMatchResultFloorBeforeSources is testMatchResultFloorAfterSources in
// the other order: the sources' lower floor leaves the result's in place.
func testMatchResultFloorBeforeSources(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	m := resultMatch()
	if _, err := s.Matches().Save(ctx, "", &m); err != nil {
		t.Fatal(err)
	}
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	mergeSaved(t, s, "", posID, sourceAnalysis("XG", "XG Roller++", 0.30, 0.20, 0.10))
	mergeSaved(t, s, "", posID, sourceAnalysis("GNUbg", "2-ply", 0.25, 0.28))
	requireFloor(t, s, "after a result then sources", domain.MatchResultReaderVersion)
}

// CheckMatchResultFloorUnderRLS stores a match with its result as tenant a
// in a store that holds row-level security as a restricted role, the
// daemon's deployment: the role raises the floor in the global metadata
// table (ADR-0114, as ADR-0113 §7 for the analyses).
func CheckMatchResultFloorUnderRLS(t *testing.T, s storage.Storage, a string) {
	t.Helper()
	ctx := tenantContext(t, context.Background(), a)
	m := resultMatch()
	if _, err := s.Matches().Save(ctx, a, &m); err != nil {
		t.Fatalf("Save match(%s): %v", a, err)
	}
	md, err := s.Metadata().Load(ctx, a)
	if err != nil {
		t.Fatalf("Metadata.Load(%s): %v", a, err)
	}
	if got := md[storage.MinReaderVersionKey]; got != domain.MatchResultReaderVersion {
		t.Errorf("floor = %q, want %q raised by the restricted role", got, domain.MatchResultReaderVersion)
	}
}
