package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The cases in this file pin the *kind* of error a backend reports for a
// client mistake, because the HTTP daemon maps kinds to status codes: an
// unmapped driver error is ErrInternal and becomes a 500 for what the
// caller can fix by themselves. Found by the server's route smoke test.

// testDanglingReferenceIsNotFound: pointing at a row that does not exist (a
// position or collection id nobody created) is ErrNotFound, exactly as Get
// and Load already report — not the driver's FOREIGN KEY violation.
func testDanglingReferenceIsNotFound(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	const nobody int64 = 987654321

	coll, err := s.Collections().Create(ctx, "", "real", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	cp := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &cp)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	_, commentErr := s.Comments().Add(ctx, "", nobody, "orphan")
	checks := []struct {
		name string
		err  error
	}{
		{"AddPosition unknown position", s.Collections().AddPosition(ctx, "", coll, nobody)},
		{"AddPosition unknown collection", s.Collections().AddPosition(ctx, "", nobody, posID)},
		{"CopyPosition unknown collection", s.Collections().CopyPosition(ctx, "", nobody, posID)},
		{"MovePosition unknown target", s.Collections().MovePosition(ctx, "", coll, nobody, posID)},
		{"Comments.Add unknown position", commentErr},
	}
	for _, c := range checks {
		if !errors.Is(c.err, storage.ErrNotFound) {
			t.Errorf("%s: got %v, want ErrNotFound", c.name, c.err)
		}
	}
}

// testMergePlayersRejectsEmptyAsInvalid: an empty canonical name or an empty
// name list is the caller's mistake, so ErrInvalid — not ErrInternal, which
// the daemon would hide behind "internal error".
func testMergePlayersRejectsEmptyAsInvalid(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if err := s.Matches().MergePlayers(ctx, "", []string{"a"}, ""); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("empty canonical: got %v, want ErrInvalid", err)
	}
	if err := s.Matches().MergePlayers(ctx, "", nil, "a"); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("no names: got %v, want ErrInvalid", err)
	}
}
