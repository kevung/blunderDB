package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The three listing families that streamed everything and nothing else
// (issue #237): comments.listAll, tournaments.list, collections.positions.
//
// What is asserted is that the page is a WINDOW on the unbounded list — same
// order, same rows, offset and limit composing — because that is the property
// a client pages on. A limit that reordered would be worse than no limit: a
// reader would see one row twice and never see another.
func testListPaging(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// Five comments, on five positions of their own.
	for i := 0; i < 5; i++ {
		p := checkerPos()
		p.Board.Points[6].Checkers = i + 1
		p.Board.Points[6].Color = domain.Black
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("save position %d: %v", i, err)
		}
		if _, err := s.Comments().Upsert(ctx, "", id, "comment "+string(rune('a'+i))); err != nil {
			t.Fatalf("comment %d: %v", i, err)
		}
	}

	all := drainComments(t, "ListAll", s.Comments().ListAll(ctx, "", storage.ListOpts{}))
	if len(all) < 5 {
		t.Fatalf("ListAll returned %d comments, want at least 5", len(all))
	}
	window := func(opts storage.ListOpts) []domain.CommentEntry {
		return drainComments(t, "ListAll page", s.Comments().ListAll(ctx, "", opts))
	}
	if got := window(storage.ListOpts{Limit: 2}); len(got) != 2 ||
		got[0].ID != all[0].ID || got[1].ID != all[1].ID {
		t.Errorf("limit 2 is not the first two of the unbounded list: %v", commentIDs(got))
	}
	if got := window(storage.ListOpts{Limit: 2, Offset: 2}); len(got) != 2 ||
		got[0].ID != all[2].ID || got[1].ID != all[3].ID {
		t.Errorf("limit 2 offset 2 is not the third and fourth: %v", commentIDs(got))
	}
	if got := window(storage.ListOpts{Offset: 2}); len(got) != len(all)-2 || got[0].ID != all[2].ID {
		t.Errorf("offset alone is not the tail: %d rows, first %v", len(got), commentIDs(got[:1]))
	}
	// A limit past the end is not an error and not a wrap: it is the list.
	if got := window(storage.ListOpts{Limit: len(all) + 10}); len(got) != len(all) {
		t.Errorf("a limit past the end returned %d rows, want %d", len(got), len(all))
	}

	// Tournaments: three of them, paged the same way.
	for _, name := range []string{"Paris", "Lyon", "Nice"} {
		if _, err := s.Tournaments().Create(ctx, "", name, "", ""); err != nil {
			t.Fatalf("create tournament %s: %v", name, err)
		}
	}
	tours := drainTournaments(t, s.Tournaments().List(ctx, "", storage.ListOpts{}))
	if len(tours) < 3 {
		t.Fatalf("List returned %d tournaments, want at least 3", len(tours))
	}
	page := drainTournaments(t, s.Tournaments().List(ctx, "", storage.ListOpts{Limit: 2}))
	if len(page) != 2 || page[0].ID != tours[0].ID || page[1].ID != tours[1].ID {
		t.Errorf("tournament limit 2 is not the first two of the unbounded list")
	}
	page = drainTournaments(t, s.Tournaments().List(ctx, "", storage.ListOpts{Limit: 1, Offset: 1}))
	if len(page) != 1 || page[0].ID != tours[1].ID {
		t.Errorf("tournament limit 1 offset 1 is not the second")
	}

	// Collection positions: four of them, in their sort order.
	cid, err := s.Collections().Create(ctx, "", "paged", "")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	var ids []int64
	for i := 0; i < 4; i++ {
		p := checkerPos()
		p.Board.Points[8].Checkers = i + 1
		p.Board.Points[8].Color = domain.White
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("save collection position %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	if err := s.Collections().AddPositions(ctx, "", cid, ids); err != nil {
		t.Fatalf("add positions: %v", err)
	}
	drainPos := func(opts storage.ListOpts) []int64 {
		var out []int64
		for p, err := range s.Collections().Positions(ctx, "", cid, opts) {
			if err != nil {
				t.Fatalf("collection positions: %v", err)
			}
			out = append(out, p.ID)
		}
		return out
	}
	full := drainPos(storage.ListOpts{})
	if len(full) != 4 {
		t.Fatalf("collection holds %d positions, want 4", len(full))
	}
	if got := drainPos(storage.ListOpts{Limit: 2}); len(got) != 2 || got[0] != full[0] || got[1] != full[1] {
		t.Errorf("collection limit 2 = %v, want the first two of %v", got, full)
	}
	if got := drainPos(storage.ListOpts{Limit: 2, Offset: 2}); len(got) != 2 || got[0] != full[2] || got[1] != full[3] {
		t.Errorf("collection limit 2 offset 2 = %v, want the last two of %v", got, full)
	}
}

func drainTournaments(t *testing.T, seq func(func(*domain.Tournament, error) bool)) []*domain.Tournament {
	t.Helper()
	var out []*domain.Tournament
	for tr, err := range seq {
		if err != nil {
			t.Fatalf("list tournaments: %v", err)
		}
		out = append(out, tr)
	}
	return out
}
