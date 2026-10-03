// Tenant isolation cases: one family at a time, tenant A writes a row and
// tenant B must see none of it — no List entry, no Get/Load by A's id.
//
// This table lives here (backend-agnostic, like the rest of storagetest) but
// is deliberately NOT part of RunContractTests: SQLite has no tenants (see
// storage.go's package doc — the desktop/CLI pass the single implicit tenant
// "", and `scope` is otherwise unused by that backend), so asserting cross-
// tenant invisibility against it would either be vacuous or fail by
// construction. RunTenantIsolationTests is for a real multi-tenant backend
// (PostgreSQL), called from its own build-tag-gated test
// (tenant_isolation_postgres_test.go).
package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// tenantIsolationCase is one family's isolation check.
type tenantIsolationCase struct {
	name string
	fn   func(t *testing.T, ctx context.Context, s storage.Storage, a, b string)
}

var tenantIsolationCases = []tenantIsolationCase{
	{"Position", checkPositionIsolation},
	{"TranscriptionRevision", checkTranscriptionRevisionIsolation},
	{"Analysis", checkAnalysisIsolation},
	{"Collection", checkCollectionIsolation},
	{"Tournament", checkTournamentIsolation},
	{"Filter", checkFilterIsolation},
	{"Match", checkMatchIsolation},
	{"Comment", checkCommentIsolation},
	{"Anki/Deck", checkAnkiDeckIsolation},
	{"Direction", checkDirectionIsolation},
	{"TableSetting", checkTableSettingIsolation},
	{"Training", checkTrainingIsolation},
}

// RunTenantIsolationTests runs every family's isolation check against a
// fresh Storage from factory (called once per case, like RunContractTests).
// a and b are two distinct tenant scopes.
func RunTenantIsolationTests(t *testing.T, factory func() storage.Storage, a, b string) {
	t.Helper()
	ctx := context.Background()
	for _, tc := range tenantIsolationCases {
		t.Run(tc.name, func(t *testing.T) {
			s := factory()
			defer s.Close()
			tc.fn(t, ctx, s, a, b)
		})
	}
}

func checkPositionIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	p := checkerPos()
	id, err := s.Positions().Save(ctx, a, &p)
	if err != nil {
		t.Fatalf("Save(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Positions().List(ctx, b, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d position(s) belonging to tenant %s, want 0", b, n, a)
	}
	if _, err := s.Positions().Load(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Load(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkAnalysisIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	pa := checkerPos()
	idA, err := s.Positions().Save(ctx, a, &pa)
	if err != nil {
		t.Fatalf("Save position(%s): %v", a, err)
	}
	if err := s.Analyses().Save(ctx, a, idA, &domain.PositionAnalysis{}); err != nil {
		t.Fatalf("Save analysis(%s): %v", a, err)
	}

	// idA cannot collide with any id tenant b already holds (ids are
	// assigned from one global sequence, not per tenant), so this alone
	// proves b cannot read a's analysis by a's id.
	if _, err := s.Analyses().Load(ctx, b, idA); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Load(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkCollectionIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	cid, err := s.Collections().Create(ctx, a, "priv-coll", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Collections().List(ctx, b) {
		if err != nil {
			t.Fatalf("List(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d collection(s) belonging to tenant %s, want 0", b, n, a)
	}
	if _, err := s.Collections().Get(ctx, b, cid); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkTournamentIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	id, err := s.Tournaments().Create(ctx, a, "priv-tourney", "2026-01-01", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Tournaments().List(ctx, b, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d tournament(s) belonging to tenant %s, want 0", b, n, a)
	}
	if _, err := s.Tournaments().Get(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkFilterIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	if _, err := s.Filters().Save(ctx, a, "fav", "cmd"); err != nil {
		t.Fatalf("Save(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Filters().List(ctx, b) {
		if err != nil {
			t.Fatalf("List(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d filter(s) belonging to tenant %s, want 0", b, n, a)
	}
	// The same name is free to reuse under a different tenant: it collides
	// only within one tenant's own filter library.
	if _, err := s.Filters().Save(ctx, b, "fav", "cmd"); err != nil {
		t.Errorf("Save(%s) with a's filter name: %v, want no collision across tenants", b, err)
	}
}

func checkMatchIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7}
	id, err := s.Matches().Save(ctx, a, &m)
	if err != nil {
		t.Fatalf("Save(%s): %v", a, err)
	}
	if _, err := s.Matches().Get(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkCommentIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, a, &p)
	if err != nil {
		t.Fatalf("Save position(%s): %v", a, err)
	}
	if _, err := s.Comments().Add(ctx, a, posID, "private note"); err != nil {
		t.Fatalf("Add comment(%s): %v", a, err)
	}

	// posID belongs to tenant a alone (global id sequence — see
	// checkAnalysisIsolation), so tenant b addressing it directly must read
	// no comment at all.
	got, err := s.Comments().Text(ctx, b, posID)
	if err != nil {
		t.Fatalf("Text(%s, position from %s): %v", b, a, err)
	}
	if got != "" {
		t.Errorf("tenant %s reads tenant %s's comment via its position id: %q", b, a, got)
	}
}

func checkAnkiDeckIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	if _, err := s.Anki().CreateDeck(ctx, a, "priv-deck", "", "collection", 0, ""); err != nil {
		t.Fatalf("CreateDeck(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Anki().ListDecks(ctx, b) {
		if err != nil {
			t.Fatalf("ListDecks(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d deck(s) belonging to tenant %s, want 0", b, n, a)
	}
}

func checkDirectionIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	tid, err := s.Tournaments().Create(ctx, a, "priv-directed", "2026-10-03", "")
	if err != nil {
		t.Fatalf("Create tournament(%s): %v", a, err)
	}
	ds := s.Directions()
	if err := ds.Create(ctx, b, direction.Record{TournamentID: tid, State: direction.StateDraft}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Create(%s) on a Tournament of %s: got %v, want ErrNotFound", b, a, err)
	}
	if err := ds.Create(ctx, a, direction.Record{TournamentID: tid, State: direction.StateDraft}); err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}
	if err := ds.AppendEvent(ctx, a, tid, direction.StoredEvent{Seq: 0, Kind: "created", Time: time.Now(), Payload: []byte(`{}`)}); err != nil {
		t.Fatalf("AppendEvent(%s): %v", a, err)
	}

	if list, err := ds.List(ctx, b); err != nil || len(list) != 0 {
		t.Errorf("List(%s) = %d, %v; want none of %s's", b, len(list), err, a)
	}
	if _, err := ds.Get(ctx, b, tid); !errors.Is(err, direction.ErrNoDirection) {
		t.Errorf("Get(%s, id from %s): got %v, want ErrNoDirection", b, a, err)
	}
	if evs, err := ds.LoadEvents(ctx, b, tid); err != nil || len(evs) != 0 {
		t.Errorf("LoadEvents(%s) = %d, %v; want none of %s's", b, len(evs), err, a)
	}
	if n, last, err := ds.EventsHead(ctx, b, tid); err != nil || n != 0 || last != -1 {
		t.Errorf("EventsHead(%s) = %d, %d, %v; want none of %s's", b, n, last, err, a)
	}
	if err := ds.AppendEvent(ctx, b, tid, direction.StoredEvent{Seq: 1, Kind: "result", Time: time.Now(), Payload: []byte(`{}`)}); !errors.Is(err, direction.ErrNoDirection) {
		t.Errorf("AppendEvent(%s) into %s's log: got %v, want ErrNoDirection", b, a, err)
	}
	if err := ds.Update(ctx, b, direction.Record{TournamentID: tid, State: direction.StateFinished}); !errors.Is(err, direction.ErrNoDirection) {
		t.Errorf("Update(%s) of %s's record: got %v, want ErrNoDirection", b, a, err)
	}
	if err := ds.Delete(ctx, b, tid); err != nil {
		t.Fatalf("Delete(%s): %v", b, err)
	}
	rec, err := ds.Get(ctx, a, tid)
	if err != nil || rec.State != direction.StateDraft {
		t.Errorf("Get(%s) after %s's writes = %+v, %v; want the draft untouched", a, b, rec, err)
	}
	if evs, _ := ds.LoadEvents(ctx, a, tid); len(evs) != 1 {
		t.Errorf("LoadEvents(%s) after %s's Delete = %d events, want 1", a, b, len(evs))
	}

	// Slots and pairs: another tenant neither reads nor moves them.
	m := domain.Match{Player1Name: "Anna", Player2Name: "Bruno", MatchLength: 5}
	mid, err := s.Matches().Save(ctx, a, &m)
	if err != nil {
		t.Fatalf("Save match(%s): %v", a, err)
	}
	if err := ds.AttachSlot(ctx, b, tid, "M1", mid); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("AttachSlot(%s) of %s's Match: got %v, want ErrNotFound", b, a, err)
	}
	if err := ds.AttachSlot(ctx, a, tid, "M1", mid); err != nil {
		t.Fatalf("AttachSlot(%s): %v", a, err)
	}
	own := domain.Match{Player1Name: "Carl", Player2Name: "Dora", MatchLength: 5}
	ownID, err := s.Matches().Save(ctx, b, &own)
	if err != nil {
		t.Fatalf("Save match(%s): %v", b, err)
	}
	if err := ds.AttachSlot(ctx, b, tid, "M2", ownID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("AttachSlot(%s) of its own Match into %s's Tournament: got %v, want ErrNotFound", b, a, err)
	}
	if _, slot, err := ds.SlotOf(ctx, b, ownID); err != nil || slot != "" {
		t.Errorf("SlotOf(%s) after the refused attach = %q, %v; want none", b, slot, err)
	}
	if got, err := ds.FilledSlots(ctx, b, tid); err != nil || len(got) != 0 {
		t.Errorf("FilledSlots(%s) = %d, %v; want none of %s's", b, len(got), err, a)
	}
	if _, _, err := ds.SlotOf(ctx, b, mid); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SlotOf(%s) of %s's Match: got %v, want ErrNotFound", b, a, err)
	}
	if err := ds.DetachSlot(ctx, b, tid, "M1"); err != nil {
		t.Fatalf("DetachSlot(%s): %v", b, err)
	}
	if _, slot, err := ds.SlotOf(ctx, a, mid); err != nil || slot != "M1" {
		t.Errorf("SlotOf(%s) after %s's DetachSlot = %q, %v; want M1", a, b, slot, err)
	}
	if err := ds.SetPair(ctx, a, tid, "P1", []storage.PairMember{{Name: "Anna"}, {Name: "Bruno"}}); err != nil {
		t.Fatalf("SetPair(%s): %v", a, err)
	}
	if got, err := ds.Pairs(ctx, b, tid); err != nil || len(got) != 0 {
		t.Errorf("Pairs(%s) = %v, %v; want none of %s's", b, got, err, a)
	}
}

// checkTrainingIsolation: a Training journal is the person's own, read back by
// /v1/training.* — another tenant sees neither its sessions nor its aggregates.
func checkTrainingIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	session := storage.TrainingSession{
		Exercise: "pips", SeedSource: "pool", NumbersAsked: 2, Faults: 1,
		Items: []storage.TrainingItem{{NumberType: "pips.bottom", Wrong: true}, {NumberType: "pips.top"}},
	}
	if _, err := s.Training().Save(ctx, a, session); err != nil {
		t.Fatalf("Training.Save(%s): %v", a, err)
	}
	sessions, err := s.Training().Sessions(ctx, b, "", 0)
	if err != nil {
		t.Fatalf("Training.Sessions(%s): %v", b, err)
	}
	if len(sessions) != 0 {
		t.Errorf("tenant %s sees %d training session(s) of tenant %s, want 0", b, len(sessions), a)
	}
	stats, err := s.Training().NumberStats(ctx, b, "pips")
	if err != nil {
		t.Fatalf("Training.NumberStats(%s): %v", b, err)
	}
	if len(stats) != 0 {
		t.Errorf("tenant %s sees %d training aggregate(s) of tenant %s, want 0", b, len(stats), a)
	}
	if own, err := s.Training().Sessions(ctx, a, "", 0); err != nil || len(own) != 1 {
		t.Errorf("tenant %s reads back its own session: %d, %v", a, len(own), err)
	}
}
