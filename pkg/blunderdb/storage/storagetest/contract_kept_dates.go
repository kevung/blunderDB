// Contract cases for the dates an import keeps: KeepDates and KeepCreatedAt
// write back the text TimestampText reads, the same on both backends.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

const (
	keptCreated = "2019-03-04 05:06:07"
	keptUpdated = "2020-11-12 13:14:15"
)

// testKeptDates pins that a tournament, a draft and an alias read back the
// dates written through the contract exactly (UTC text in, the same text
// out), that an empty date leaves its column alone, and that keeping a
// draft's dates does not advance its revision.
func testKeptDates(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	tid, err := s.Tournaments().Create(ctx, "", "Kept Open", "2019-03-04", "Paris")
	if err != nil {
		t.Fatalf("Tournaments.Create: %v", err)
	}
	if err := s.Tournaments().KeepDates(ctx, "", tid, keptCreated, keptUpdated); err != nil {
		t.Fatalf("Tournaments.KeepDates: %v", err)
	}
	tour, err := s.Tournaments().Get(ctx, "", tid)
	if err != nil {
		t.Fatalf("Tournaments.Get: %v", err)
	}
	if tour.CreatedAt != keptCreated || tour.UpdatedAt != keptUpdated {
		t.Errorf("tournament dates = %q / %q, want %q / %q", tour.CreatedAt, tour.UpdatedAt, keptCreated, keptUpdated)
	}
	if err := s.Tournaments().KeepDates(ctx, "", tid, "", ""); err != nil {
		t.Fatalf("Tournaments.KeepDates empty: %v", err)
	}
	if tour, _ = s.Tournaments().Get(ctx, "", tid); tour.CreatedAt != keptCreated {
		t.Errorf("empty KeepDates moved created_at to %q", tour.CreatedAt)
	}

	ts := s.Transcriptions()
	did, err := ts.Save(ctx, "", &storage.Transcription{FormatVersion: "1", Label: "Kept", Document: `{"actions":[]}`})
	if err != nil {
		t.Fatalf("Transcriptions.Save: %v", err)
	}
	before, err := ts.Get(ctx, "", did)
	if err != nil {
		t.Fatalf("Transcriptions.Get: %v", err)
	}
	if err := ts.KeepDates(ctx, "", did, keptCreated, keptUpdated); err != nil {
		t.Fatalf("Transcriptions.KeepDates: %v", err)
	}
	draft, err := ts.Get(ctx, "", did)
	if err != nil {
		t.Fatalf("Transcriptions.Get: %v", err)
	}
	if draft.CreatedAt != keptCreated || draft.UpdatedAt != keptUpdated {
		t.Errorf("draft dates = %q / %q, want %q / %q", draft.CreatedAt, draft.UpdatedAt, keptCreated, keptUpdated)
	}
	if draft.Revision != before.Revision {
		t.Errorf("KeepDates moved the revision from %d to %d", before.Revision, draft.Revision)
	}

	as := s.Aliases()
	if err := as.Set(ctx, "", storage.AliasPlayer, "Bobby", "Bob"); err != nil {
		t.Fatalf("Aliases.Set: %v", err)
	}
	if err := as.KeepCreatedAt(ctx, "", storage.AliasPlayer, "Bobby", keptCreated); err != nil {
		t.Fatalf("Aliases.KeepCreatedAt: %v", err)
	}
	if err := as.KeepCreatedAt(ctx, "", storage.AliasPlayer, "Nobody", keptCreated); err != nil {
		t.Errorf("KeepCreatedAt on an unknown alias: %v, want nil", err)
	}
	list, err := as.List(ctx, "", storage.AliasPlayer)
	if err != nil {
		t.Fatalf("Aliases.List: %v", err)
	}
	if len(list) != 1 || list[0].Alias != "Bobby" || list[0].CreatedAt != keptCreated {
		t.Errorf("aliases = %+v, want Bobby created %q", list, keptCreated)
	}
}
