//go:build postgres

package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestChangeSetLinks_Postgres(t *testing.T) {
	s, _ := openRLS(t)
	ctx := context.Background()
	var n int64
	storagetest.CheckChangeSetLinks(t, func(t *testing.T) *storagetest.ChangeSetPair {
		n += 2
		a, b := n, n+1
		return storagetest.NewChangeSetPair(t, s, storage.WithTenant(ctx, a), strconv.FormatInt(a, 10), s, storage.WithTenant(ctx, b), strconv.FormatInt(b, 10))
	})
}

// A point-in-time recovery keeps the cluster's identifier and rewinds its xid
// counter: a cursor from the abandoned timeline stays refused once the counter
// is back past it, where nothing else could tell what was lost.
func TestChangeSetCursorOfAnotherTimeline_Postgres(t *testing.T) {
	s, _ := openRLS(t)
	ctx := storage.WithTenant(context.Background(), 1)
	for n := 0; n < 3; n++ {
		if _, err := s.Collections().Create(ctx, "1", "c"+strconv.Itoa(n), ""); err != nil {
			t.Fatal(err)
		}
	}
	set, err := changeset.Export(ctx, s, "1", storage.Cursor{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	cluster, timeline, ok := strings.Cut(set.Cursor.Epoch, "/")
	if !ok || len(timeline) != 8 || cluster == "" {
		t.Fatalf("epoch %q does not name a timeline", set.Cursor.Epoch)
	}
	stale := set.Cursor
	stale.Epoch = cluster + "/" + fmt.Sprintf("%08X", otherTimeline(t, timeline))
	if _, err := changeset.Export(ctx, s, "1", stale, 100); !errors.Is(err, storage.ErrCursorStale) {
		t.Errorf("a cursor of the abandoned timeline: %v, want stale", err)
	}
	if _, err := changeset.Export(ctx, s, "1", set.Cursor, 100); err != nil {
		t.Errorf("the current cursor: %v", err)
	}
}

func otherTimeline(t *testing.T, hex string) uint64 {
	t.Helper()
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		t.Fatal(err)
	}
	return n + 1
}
