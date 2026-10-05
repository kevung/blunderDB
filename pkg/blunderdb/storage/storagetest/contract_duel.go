// Contract cases for Duels: the draft's life cycle under its revision, the
// seed written once, and the origin of the Match a Duel became.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testDuelDraft pins the draft: an insert carries its seed and revision 1, a
// rewrite under the right revision advances it and leaves the seed alone, a
// stale revision is a conflict, and Delete removes the row once.
func testDuelDraft(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Duels()

	if _, err := ds.Save(ctx, "", &storage.Duel{FormatVersion: "1", Document: "{}"}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Save without a seed: got %v, want ErrInvalid", err)
	}
	d := &storage.Duel{FormatVersion: "1", Label: "Alice — Bob", Document: `{"actions":[]}`, DiceSeed: "s1"}
	id, err := ds.Save(ctx, "", d)
	if err != nil || id == 0 || d.Revision != 1 {
		t.Fatalf("Save: id=%d rev=%d err=%v", id, d.Revision, err)
	}
	got, err := ds.Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "Alice — Bob" || got.Document != `{"actions":[]}` || got.DiceSeed != "s1" || got.Revision != 1 ||
		got.CreatedAt == "" || got.UpdatedAt == "" {
		t.Errorf("Get: got %+v", got)
	}

	got.Document, got.DiceSeed = `{"actions":[1]}`, "another"
	if _, err := ds.Save(ctx, "", got); err != nil || got.Revision != 2 {
		t.Fatalf("Save under revision 1: rev=%d err=%v", got.Revision, err)
	}
	again, err := ds.Get(ctx, "", id)
	if err != nil || again.Document != `{"actions":[1]}` || again.DiceSeed != "s1" {
		t.Errorf("after a rewrite: got %+v, %v; the seed is written once", again, err)
	}

	stale := *again
	stale.Revision = 1
	if _, err := ds.Save(ctx, "", &stale); !errors.Is(err, storage.ErrConflict) {
		t.Errorf("Save under a stale revision: got %v, want ErrConflict", err)
	}

	second := &storage.Duel{FormatVersion: "1", Document: "{}", DiceSeed: "s2"}
	id2, err := ds.Save(ctx, "", second)
	if err != nil {
		t.Fatalf("Save second: %v", err)
	}
	var ids []int64
	for row, err := range ds.List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		ids = append(ids, row.ID)
	}
	if len(ids) != 2 || ids[0] != id2 || ids[1] != id {
		t.Errorf("List = %v, want [%d %d] (newest first)", ids, id2, id)
	}

	if err := ds.Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := ds.Get(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get after Delete: got %v, want ErrNotFound", err)
	}
	if err := ds.Delete(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Delete twice: got %v, want ErrNotFound", err)
	}
	if _, err := ds.Save(ctx, "", again); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Save onto a deleted Duel: got %v, want ErrNotFound", err)
	}
}

// testMatchOrigin pins the origin: written and read back whole, replaced by a
// second write, refused for a Match that does not exist, absent for a Match
// not played here, and gone with its Match.
func testMatchOrigin(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Duels()

	matchID, err := s.Matches().Save(ctx, "", &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, MatchHash: "duel-h1"})
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if _, err := ds.Origin(ctx, "", matchID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Origin of a Match not played here: got %v, want ErrNotFound", err)
	}
	if err := ds.SetOrigin(ctx, "", &storage.MatchOrigin{MatchID: matchID + 10_000, DiceSeed: "s"}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetOrigin of an unknown Match: got %v, want ErrNotFound", err)
	}

	want := storage.MatchOrigin{MatchID: matchID, DiceSeed: "seed", StoppedEarly: true}
	if err := ds.SetOrigin(ctx, "", &want); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	want.Start, want.StoppedEarly, want.OverTime, want.BotLevel, want.Cadence, want.BotEngine =
		"-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:5:10", false, 2, "normal", `{"reserve":600}`, "gammonNet v1.5.0"
	if err := ds.SetOrigin(ctx, "", &want); err != nil {
		t.Fatalf("SetOrigin again: %v", err)
	}
	got, err := ds.Origin(ctx, "", matchID)
	if err != nil || *got != want {
		t.Errorf("Origin = %+v, %v; want %+v", got, err, want)
	}

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}
	if _, err := ds.Origin(ctx, "", matchID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Origin after its Match is deleted: got %v, want ErrNotFound", err)
	}
}
