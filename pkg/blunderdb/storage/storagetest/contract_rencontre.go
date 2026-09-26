package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// testRencontreLifecycle: a Rencontre is created, gathers Tournaments,
// loses one, and its deletion detaches the rest without deleting any
// (ADR-0056).
func testRencontreLifecycle(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	rs := s.Rencontres()

	id, err := rs.Create(ctx, "", domain.Rencontre{Name: "Festival", StartsOn: "2026-10-03", EndsOn: "2026-10-04", Tables: 14})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	main, _ := s.Tournaments().Create(ctx, "", "Principal", "2026-10-03", "")
	speed, _ := s.Tournaments().Create(ctx, "", "Speed", "2026-10-03", "")
	for _, tid := range []int64{main, speed} {
		if err := rs.Attach(ctx, "", tid, id); err != nil {
			t.Fatalf("Attach %d: %v", tid, err)
		}
	}
	r, err := rs.Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if r.Name != "Festival" || r.Tables != 14 || r.StartsOn != "2026-10-03" || len(r.TournamentIDs) != 2 {
		t.Fatalf("Get = %+v", r)
	}
	if of, err := rs.Of(ctx, "", speed); err != nil || of != id {
		t.Fatalf("Of(speed) = %d, %v; want %d", of, err, id)
	}

	r.Tables = 12
	r.OutputDir = "/tmp/salle"
	if err := rs.Update(ctx, "", *r); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := rs.Attach(ctx, "", speed, 0); err != nil {
		t.Fatalf("detach: %v", err)
	}
	list, err := rs.List(ctx, "")
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %d, %v", len(list), err)
	}
	if list[0].Tables != 12 || list[0].OutputDir != "/tmp/salle" || len(list[0].TournamentIDs) != 1 || list[0].TournamentIDs[0] != main {
		t.Fatalf("after update and detach: %+v", list[0])
	}
	if err := rs.Attach(ctx, "", main, 9999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Attach to a missing Rencontre = %v, want ErrNotFound", err)
	}

	if err := rs.Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := rs.Get(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get after Delete = %v, want ErrNotFound", err)
	}
	if _, err := s.Tournaments().Get(ctx, "", main); err != nil {
		t.Fatalf("deleting a Rencontre must not delete its Tournament: %v", err)
	}
	if of, err := rs.Of(ctx, "", main); err != nil || of != 0 {
		t.Errorf("Of(main) after Delete = %d, %v; want detached", of, err)
	}
}

// testRencontreTrashRestores: deleting a Rencontre goes through the trash
// (ADR-0036), and restoring it attaches again the Tournaments still free.
func testRencontreTrashRestores(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	rs := s.Rencontres()
	id, err := rs.Create(ctx, "", domain.Rencontre{Name: "Festival", Tables: 8})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.Tournaments().Create(ctx, "", "A", "", "")
	b, _ := s.Tournaments().Create(ctx, "", "B", "", "")
	_ = rs.Attach(ctx, "", a, id)
	_ = rs.Attach(ctx, "", b, id)

	entry, err := trash.Rencontre(ctx, s, "", id)
	if err != nil {
		t.Fatalf("trash.Rencontre: %v", err)
	}
	if of, _ := rs.Of(ctx, "", a); of != 0 {
		t.Fatalf("a trashed Rencontre still holds tournament A")
	}
	// B joins another room meanwhile: the restore must not take it back.
	other, _ := rs.Create(ctx, "", domain.Rencontre{Name: "Autre", Tables: 4})
	_ = rs.Attach(ctx, "", b, other)

	back, err := trash.Restore(ctx, s, "", entry)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	r, err := rs.Get(ctx, "", back)
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "Festival" || r.Tables != 8 || len(r.TournamentIDs) != 1 || r.TournamentIDs[0] != a {
		t.Fatalf("restored = %+v, want Festival with A only", r)
	}
}
