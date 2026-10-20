//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run TestTransferPull -v
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// importFixture fills scope of s from an XG fixture of the repository.
func importFixture(t *testing.T, s storage.Storage, scope, fixture string) {
	t.Helper()
	ctx := context.Background()
	if n, err := storage.ParseTenant(scope); err == nil && n > 0 {
		ctx = storage.WithTenant(ctx, n)
	}
	path := filepath.Join("..", "..", "testdata", fixture)
	if _, err := (ingest.XGImporter{S: s}).Import(ctx, scope, ingest.Source{Format: ingest.FormatXG, Path: path}, nil); err != nil {
		t.Fatalf("seed %s with %s: %v", scope, fixture, err)
	}
}

func pull(t *testing.T, srv *Server, writer, from string) ingest.TransferReport {
	t.Helper()
	rec := serveAcross(t, srv, writer, from, "/v1/across.transferPull", `{"tenant":"`+from+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("pull %s → %s: %d %s", from, writer, rec.Code, rec.Body)
	}
	var rep ingest.TransferReport
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	return rep
}

// TestTransferPull_Postgres: tenant 1 pulls tenant 2, then 2 pulls 1, and
// neither holds a duplicate; each pull counts what the CLI's transfer between
// two files of the same content counts.
func TestTransferPull_Postgres(t *testing.T) {
	_, srv := newPostgresTestServerAndHandlerWith(t, func(o *Options) { o.TrustReadTenants = true })
	st := srv.opts.Storage
	importFixture(t, st, "1", "test.xg")
	importFixture(t, st, "2", "match_with_comment.xg")

	// The CLI's transfer is ingest.Transfer between two SQLite files.
	ctx := context.Background()
	a, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "a.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "b.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	importFixture(t, a, "", "test.xg")
	importFixture(t, b, "", "match_with_comment.xg")
	want, err := ingest.Transfer(ctx, ingest.TransferEnd{S: b}, ingest.TransferEnd{S: a}, ingest.WholeLibraryOptions(), nil)
	if err != nil {
		t.Fatal(err)
	}

	got := pull(t, srv, "1", "2")
	if got.Exported.Positions != want.Exported.Positions || got.Exported.Matches != want.Exported.Matches ||
		got.Imported.Matches != want.Imported.Matches || got.Imported.SavedPositions != want.Imported.SavedPositions {
		t.Errorf("server pull %+v / %+v, CLI transfer %+v / %+v", got.Exported, got.Imported, want.Exported, want.Imported)
	}
	pull(t, srv, "2", "1")

	count := func(scope string) [2]int {
		n, _ := storage.ParseTenant(scope)
		tctx := storage.WithTenant(ctx, n)
		var c [2]int
		for _, err := range st.Positions().List(tctx, scope, storage.ListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			c[0]++
		}
		for _, err := range st.Matches().List(tctx, scope, storage.MatchListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			c[1]++
		}
		return c
	}
	one, two := count("1"), count("2")
	if one != two || one[1] != 2 {
		t.Fatalf("after 2 → 1 → 2: tenant 1 %v, tenant 2 %v, want the same union with 2 matches", one, two)
	}
	if again := pull(t, srv, "1", "2"); again.Imported.Matches != 0 {
		t.Errorf("a second pull added %d matches", again.Imported.Matches)
	}
	if got := count("1"); got != one {
		t.Errorf("a second pull changed tenant 1: %v, was %v", got, one)
	}

	// The read set is the proxy's: tenant 3, unlisted, cannot be pulled, and a
	// pull never writes anywhere but the writer's tenant.
	if rec := serveAcross(t, srv, "1", "2", "/v1/across.transferPull", `{"tenant":"3"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unlisted tenant: status %d, want 400", rec.Code)
	}
	if got := count("3"); got != [2]int{} {
		t.Errorf("tenant 3 holds %v after pulls it took no part in", got)
	}

	// The documented pull of one collection brings its members; an id the
	// source does not hold is refused, and nothing lands.
	ctx2 := storage.WithTenant(ctx, 2)
	var ids []int64
	for p, err := range st.Positions().List(ctx2, "2", storage.ListOpts{Limit: 3}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	col, err := st.Collections().Create(ctx2, "2", "picked", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Collections().AddPositions(ctx2, "2", col, ids); err != nil {
		t.Fatal(err)
	}
	if rec := serveAcross(t, srv, "4", "2", "/v1/across.transferPull", `{"tenant":"2","export":{"collectionIDs":[999]}}`); rec.Code != http.StatusBadRequest {
		t.Errorf("absent collection: status %d, want 400", rec.Code)
	}
	if got := count("4"); got != [2]int{} {
		t.Errorf("a refused pull wrote %v", got)
	}
	body := fmt.Sprintf(`{"tenant":"2","export":{"wholeLibrary":false,"includeCollections":true,"collectionIDs":[%d]}}`, col)
	if rec := serveAcross(t, srv, "4", "2", "/v1/across.transferPull", body); rec.Code != http.StatusOK {
		t.Fatalf("collection pull: %d %s", rec.Code, rec.Body)
	}
	if got := count("4"); got[0] != len(ids) {
		t.Errorf("collection of %d pulled: tenant 4 holds %v", len(ids), got)
	}
}
