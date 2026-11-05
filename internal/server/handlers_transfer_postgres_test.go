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
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
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

// A drawing laid in tenant 2 arrives in tenant 1 with its position, and stays
// out of any third tenant; the pull is a PostgreSQL transaction under RLS.
func TestTransferPull_CarriesDrawing_Postgres(t *testing.T) {
	_, srv := newPostgresTestServerAndHandlerWith(t, func(o *Options) { o.TrustReadTenants = true })
	st := srv.opts.Storage
	importFixture(t, st, "2", "match_with_comment.xg")
	ctx2 := storage.WithTenant(context.Background(), 2)
	ctx1 := storage.WithTenant(context.Background(), 1)

	var src *domain.Position
	for p, err := range st.Positions().List(ctx2, "2", storage.ListOpts{Limit: 1}) {
		if err != nil {
			t.Fatal(err)
		}
		pc := *p
		src = &pc
	}
	marks := []domain.DrawingMark{{Kind: domain.DrawingArrow, From: 13, To: 7, Color: "red"}, {Kind: domain.DrawingSquare, From: 6, To: 6, Color: "green"}}
	d := st.(storage.DrawingProvider).PositionDrawings()
	if err := d.SetDrawing(ctx2, "2", src.ID, marks); err != nil {
		t.Fatal(err)
	}

	pull(t, srv, "1", "2")
	id, found, err := st.Positions().Exists(ctx1, "1", engine.ZobristHash(src))
	if err != nil || !found {
		t.Fatalf("the pulled position is not in tenant 1: %v", err)
	}
	got, err := d.Drawing(ctx1, "1", id)
	if err != nil || !reflect.DeepEqual(got, marks) {
		t.Errorf("pulled drawing = %+v, %v; want %+v", got, err, marks)
	}
	if _, err := d.Drawing(ctx1, "1", src.ID); err == nil {
		if all, _ := d.Drawings(ctx1, "1", []int64{src.ID}); len(all) != 0 {
			t.Errorf("tenant 1 reads a drawing of tenant 2: %+v", all)
		}
	}
}

// A variant pulled from tenant 2 keeps its link to the origin under tenant 1's
// ids, and the link stays out of a third tenant; the pull is a PostgreSQL
// transaction under RLS.
func TestTransferPull_CarriesDerivation_Postgres(t *testing.T) {
	_, srv := newPostgresTestServerAndHandlerWith(t, func(o *Options) { o.TrustReadTenants = true })
	st := srv.opts.Storage
	importFixture(t, st, "2", "match_with_comment.xg")
	ctx2 := storage.WithTenant(context.Background(), 2)
	ctx1 := storage.WithTenant(context.Background(), 1)

	var ps []*domain.Position
	for p, err := range st.Positions().List(ctx2, "2", storage.ListOpts{Limit: 2}) {
		if err != nil {
			t.Fatal(err)
		}
		pc := *p
		ps = append(ps, &pc)
	}
	if len(ps) < 2 {
		t.Fatal("fixture has fewer than two positions")
	}
	d := st.(storage.DerivationProvider).PositionDerivations()
	if err := d.SetDerivedFrom(ctx2, "2", ps[1].ID, &domain.Derivation{FromID: ps[0].ID, Move: "13/7"}); err != nil {
		t.Fatal(err)
	}

	pull(t, srv, "1", "2")
	vid, found, err := st.Positions().Exists(ctx1, "1", engine.ZobristHash(ps[1]))
	if err != nil || !found {
		t.Fatalf("the pulled variant is not in tenant 1: %v", err)
	}
	oid, found, err := st.Positions().Exists(ctx1, "1", engine.ZobristHash(ps[0]))
	if err != nil || !found {
		t.Fatalf("the pulled origin is not in tenant 1: %v", err)
	}
	got, err := d.DerivedFrom(ctx1, "1", vid)
	if err != nil || got == nil || got.FromID != oid || got.Move != "13/7" {
		t.Errorf("pulled derivation = %+v, %v; want origin %d move 13/7", got, err, oid)
	}
	// Tenant 1 cannot read tenant 2's rows through the ids it knows.
	if all, _ := d.Derivations(ctx1, "1", []int64{ps[1].ID}); len(all) != 0 {
		t.Errorf("tenant 1 reads a derivation of tenant 2: %+v", all)
	}
}

// A pull that leaves comments out leaves a match's own note and its file name
// at the source, keeps the hash that deduplicates, and files five positions
// pulled without their match as the recipient's individual positions; the
// pull is a PostgreSQL transaction under RLS.
func TestTransferPull_NoMatchNoteAndIndividualPositions_Postgres(t *testing.T) {
	_, srv := newPostgresTestServerAndHandlerWith(t, func(o *Options) { o.TrustReadTenants = true })
	st := srv.opts.Storage
	importFixture(t, st, "2", "match_with_comment.xg")
	ctx1 := storage.WithTenant(context.Background(), 1)
	ctx2 := storage.WithTenant(context.Background(), 2)

	var src *domain.Match
	for m, err := range st.Matches().List(ctx2, "2", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		mc := *m
		src = &mc
	}
	if err := st.Matches().UpdateComment(ctx2, "2", src.ID, "private note"); err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for p, err := range st.Positions().List(ctx2, "2", storage.ListOpts{Limit: 5}) {
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

	body := `{"tenant":"2","export":{"wholeLibrary":true,"includeMatches":true,"includeCollections":true,"includeAnalysis":true,"includeComments":false}}`
	if rec := serveAcross(t, srv, "1", "2", "/v1/across.transferPull", body); rec.Code != http.StatusOK {
		t.Fatalf("pull: %d %s", rec.Code, rec.Body)
	}
	var got *domain.Match
	for m, err := range st.Matches().List(ctx1, "1", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		mc := *m
		got = &mc
	}
	if got == nil {
		t.Fatal("no match arrived in tenant 1")
	}
	if got.Comment != "" || got.CommentAuthor != "" || got.FilePath != "" {
		t.Errorf("tenant 1 holds note %q author %q file %q", got.Comment, got.CommentAuthor, got.FilePath)
	}
	if got.MatchHash != src.MatchHash {
		t.Errorf("match hash %q, want %q", got.MatchHash, src.MatchHash)
	}

	body = fmt.Sprintf(`{"tenant":"2","export":{"wholeLibrary":false,"includeCollections":true,"collectionIDs":[%d]}}`, col)
	if rec := serveAcross(t, srv, "4", "2", "/v1/across.transferPull", body); rec.Code != http.StatusOK {
		t.Fatalf("collection pull: %d %s", rec.Code, rec.Body)
	}
	ctx4 := storage.WithTenant(context.Background(), 4)
	n, individual := 0, 0
	for p, err := range st.Positions().List(ctx4, "4", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
		if p.IndividuallyImported {
			individual++
		}
	}
	if n != len(ids) || individual != n {
		t.Errorf("tenant 4 holds %d positions, %d individual; want %d of %d", n, individual, len(ids), len(ids))
	}
}
