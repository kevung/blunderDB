//go:build postgres

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestChangeSet_Postgres exchanges changes between two tenants of one
// database, under FORCE RLS.
func TestChangeSet_Postgres(t *testing.T) {
	s, _ := openRLS(t)
	ctx := context.Background()
	storagetest.CheckChangeSet(t, s, storage.WithTenant(ctx, 1), "1", s, storage.WithTenant(ctx, 2), "2")
}

// TestChangeSetPreJournal_Postgres: a tenant's rows with no journal entry, as
// an upgrade leaves them, under FORCE RLS.
func TestChangeSetPreJournal_Postgres(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatal(err)
	}
	rls, err := pg.Open(ctx, ownerDSN, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rls.Close() })
	storagetest.CheckChangeSetPreJournal(t, rls, storage.WithTenant(ctx, 1), "1", rls, storage.WithTenant(ctx, 2), "2", func() {
		asTenant(t, conn, "1", func() {
			if _, err := conn.Exec(ctx, `DELETE FROM change_log`); err != nil {
				t.Fatal(err)
			}
		})
	})
}

// The cursor stays at the last entry of a carried family, so a family a later
// version carries is not skipped by a peer that exported before it did.
func TestChangeSetCursorStopsAtUncarriedFamilies_Postgres(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatal(err)
	}
	rls, err := pg.Open(ctx, ownerDSN, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rls.Close() })
	tenant := storage.WithTenant(ctx, 1)
	id, err := rls.Collections().Create(tenant, "1", "Before the unknown", "")
	if err != nil {
		t.Fatal(err)
	}
	u, err := rls.ObjectUUIDs().ObjectUUID(tenant, "1", "collection", id)
	if err != nil {
		t.Fatal(err)
	}
	asTenant(t, conn, "1", func() {
		if _, err := conn.Exec(ctx, `INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (1, '0190a000-0000-7000-8000-00000000f00d', 'not-carried-yet', 'update')`); err != nil {
			t.Fatal(err)
		}
	})
	set, err := changeset.Export(tenant, rls, "1", storage.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	last, err := rls.ChangeJournal().Entry(tenant, "1", u)
	if err != nil {
		t.Fatal(err)
	}
	if set.Cursor.Seq != last.Seq {
		t.Errorf("cursor %d passed the carried entries (%d) over an uncarried family", set.Cursor.Seq, last.Seq)
	}
	for _, r := range set.Records {
		if r.Family == "not-carried-yet" {
			t.Errorf("a record of family %s", r.Family)
		}
	}
}

func TestChangeSetFirstSync_Postgres(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatal(err)
	}
	rls, err := pg.Open(ctx, ownerDSN, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rls.Close() })
	storagetest.CheckChangeSetFirstSync(t, rls, storage.WithTenant(ctx, 1), "1", rls, storage.WithTenant(ctx, 2), "2", func() {
		asTenant(t, conn, "1", func() {
			if _, err := conn.Exec(ctx, `DELETE FROM change_log`); err != nil {
				t.Fatal(err)
			}
		})
	})
}

// A pg_dump restored into another cluster brings rows whose xid the new
// cluster's counter has not reached: the journal stays readable, and a cursor
// from before the restore is refused, never answered with nothing.
func TestChangeSetCursorAfterRestore_Postgres(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatal(err)
	}
	rls, err := pg.Open(ctx, ownerDSN, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rls.Close() })
	ca := storage.WithTenant(ctx, 1)
	col, err := rls.Collections().Create(ca, "1", "Before the restore", "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := changeset.Export(ca, rls, "1", storage.Cursor{}, 100)
	if err != nil || len(first.Records) != 1 {
		t.Fatalf("export: %d records, %v", len(first.Records), err)
	}

	const ahead = int64(99_999_999_999)
	asTenant(t, conn, "1", func() {
		if _, err := conn.Exec(ctx, `UPDATE change_log SET xid = CAST('99999999999' AS xid8)`); err != nil {
			t.Fatal(err)
		}
	})

	full, err := changeset.Export(ca, rls, "1", storage.Cursor{}, 100)
	if err != nil {
		t.Fatalf("a full export after the restore: %v", err)
	}
	if len(full.Records) != 1 {
		t.Fatalf("a full export after the restore holds %d records, want 1", len(full.Records))
	}

	for name, cur := range map[string]storage.Cursor{
		"ahead of the counter": {Xid: ahead, Seq: first.Cursor.Seq, Epoch: first.Cursor.Epoch, Carried: first.Cursor.Carried},
		"of another cluster":   {Xid: first.Cursor.Xid, Seq: first.Cursor.Seq, Epoch: "another cluster", Carried: first.Cursor.Carried},
	} {
		if set, err := changeset.Export(ca, rls, "1", cur, 100); !errors.Is(err, storage.ErrCursorStale) {
			t.Errorf("cursor %s: %d records, %v, want ErrCursorStale", name, len(set.Records), err)
		}
	}

	// What is written after the restore comes after a cursor taken after it.
	if err := rls.Collections().Update(ca, "1", col, "After the restore", ""); err != nil {
		t.Fatal(err)
	}
	next, err := changeset.Export(ca, rls, "1", full.Cursor, 100)
	if err != nil || len(next.Records) != 1 {
		t.Errorf("incremental after the restore: %d records, %v, want 1", len(next.Records), err)
	}
}
