// Club and coach reads (storage/club.go, ADR-0065), run across a read set
// like the rest of RunReadAcrossTests: a coach's comments joined to a board
// by its Zobrist hash, a library's collections, a club ranking. Each tenant
// below owns the same board (checkerPos), so the hash names a position in
// every tenant and only the tenant tag says whose comment or row it is.
package storagetest

import (
	"context"
	"iter"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func runClubReads(t *testing.T, ctx context.Context, s storage.Storage, set storage.ReadTenants, read, owners []string, outsider string, in func(string) context.Context) {
	t.Helper()
	board := checkerPos()
	norm := board.NormalizeForStorage()
	hash := engine.ZobristHash(&norm)

	for _, scope := range owners {
		cx := in(scope)
		p := checkerPos()
		id, err := s.Positions().Save(cx, scope, &p)
		if err != nil {
			t.Fatalf("Save position(%s): %v", scope, err)
		}
		if _, err := s.Comments().Add(cx, scope, id, "note-"+scope); err != nil {
			t.Fatalf("Add comment(%s): %v", scope, err)
		}
		cid, err := s.Collections().Create(cx, scope, "library-"+scope, "")
		if err != nil {
			t.Fatalf("Create collection(%s): %v", scope, err)
		}
		if err := s.Collections().AddPosition(cx, scope, cid, id); err != nil {
			t.Fatalf("AddPosition(%s): %v", scope, err)
		}
	}

	t.Run("CommentsByZobristJoinEachTenantsBoard", func(t *testing.T) {
		got, err := storage.ReadAcross(ctx, set, func(ctx context.Context, scope string) ([]storage.ZobristComment, error) {
			return storage.CommentsByZobrist(ctx, s, scope, []uint64{hash, hash, hash ^ 1})
		})
		if err != nil {
			t.Fatalf("ReadAcross comments: %v", err)
		}
		for _, tg := range got {
			notes := 0
			for _, c := range tg.Item {
				if c.Zobrist != hash {
					t.Errorf("tenant %s: a comment on hash %x, asked %x", tg.Tenant, c.Zobrist, hash)
				}
				if outsider != "" && c.Comment.Text == "note-"+outsider {
					t.Errorf("tenant %s's comments hold the outsider's note", tg.Tenant)
				}
				if c.Comment.Text == "note-"+tg.Tenant {
					notes++
				}
			}
			if notes != 1 {
				t.Errorf("tenant %s: its own note read %d time(s), want once (a repeated hash is read once)", tg.Tenant, notes)
			}
		}
		if len(got) != len(set) {
			t.Errorf("%d answers for %d read tenants", len(got), len(set))
		}
	})

	t.Run("CommentsByZobristBoundsTheLookup", func(t *testing.T) {
		_, err := storage.CommentsByZobrist(in(read[0]), s, read[0], make([]uint64, storage.MaxZobristLookups+1))
		if err == nil {
			t.Error("a lookup past MaxZobristLookups is accepted")
		}
	})

	t.Run("LibraryCollectionsTagEachTenant", func(t *testing.T) {
		names := map[string][]string{}
		for tg, err := range storage.StreamAcross(ctx, set, func(ctx context.Context, scope string) iter.Seq2[*storage.Collection, error] {
			return s.Collections().List(ctx, scope)
		}) {
			if err != nil {
				t.Fatalf("StreamAcross collections: %v", err)
			}
			names[tg.Tenant] = append(names[tg.Tenant], tg.Item.Name)
		}
		for _, scope := range read {
			if len(names[scope]) != 1 || names[scope][0] != "library-"+scope {
				t.Errorf("tenant %s: collections %v, want its own library alone", scope, names[scope])
			}
		}
		if outsider != "" && len(names[outsider]) != 0 {
			t.Errorf("outsider %s's collections are read", outsider)
		}
	})

	t.Run("ClubRankingTagsEachRow", func(t *testing.T) {
		tables, err := storage.ReadAcross(ctx, set, func(ctx context.Context, scope string) ([]storage.PlayerRow, error) {
			return s.Stats().PlayerTable(ctx, scope, storage.StatsFilter{DecisionType: -1})
		})
		if err != nil {
			t.Fatalf("ReadAcross player tables: %v", err)
		}
		seen := map[string]bool{}
		for _, row := range storage.ClubRanking(tables, nil, 0) {
			if !set.Contains(row.Tenant) {
				t.Errorf("a row tagged %s, outside the read set", row.Tenant)
			}
			if outsider != "" && (row.Player.Name == "player-"+outsider || row.Player.Name == "mover-"+outsider) {
				t.Errorf("tenant %s's ranking holds the outsider's player", row.Tenant)
			}
			if row.Player.Name == "player-"+row.Tenant {
				seen[row.Tenant] = true
			}
		}
		for _, scope := range read {
			if !seen[scope] {
				t.Errorf("tenant %s: its player is not in the club ranking", scope)
			}
		}
	})
}
