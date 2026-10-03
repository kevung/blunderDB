//go:build postgres

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestShareCollectionBetweenTenants_Postgres: tenant 1 exports one collection
// with exports.sqlite, tenant 2 imports the file with imports.db. Tenant 2
// receives that collection and its positions only — never tenant 1's other
// positions — and no route ever reads across the tenants: the file travels
// through the client.
func TestShareCollectionBetweenTenants_Postgres(t *testing.T) {
	ts, srv := newPostgresTestServerAndHandler(t)
	ctx := context.Background()
	st := srv.opts.Storage

	shared := domain.InitializePosition()
	kept := domain.InitializePosition()
	kept.Dice = [2]int{6, 5}
	sharedID, err := st.Positions().Save(ctx, "1", &shared)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Positions().Save(ctx, "1", &kept); err != nil {
		t.Fatal(err)
	}
	cid, err := st.Collections().Create(ctx, "1", "Ouvertures du club", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Collections().AddPosition(ctx, "1", cid, sharedID); err != nil {
		t.Fatal(err)
	}

	resp := postAs(t, ts, "1", "/v1/exports.sqlite", exportSQLiteReq{CollectionIDs: []int64{cid}})
	file, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.HasPrefix(file, []byte("SQLite format 3")) {
		t.Fatalf("exports.sqlite: status %d, %.120q", resp.StatusCode, file)
	}

	uploadAsTenant(t, ts, "2", "/v1/imports.db", "club.db", file)

	var colls []storage.Collection
	for c, err := range st.Collections().List(ctx, "2") {
		if err != nil {
			t.Fatal(err)
		}
		colls = append(colls, *c)
	}
	if len(colls) != 1 || colls[0].Name != "Ouvertures du club" {
		t.Fatalf("tenant 2 receives the shared collection: %+v", colls)
	}
	n := 0
	for _, err := range st.Positions().List(ctx, "2", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Errorf("tenant 2 holds %d position(s); want the shared one only", n)
	}
	if m := countPositions(t, st, "1"); m != 2 {
		t.Errorf("tenant 1 keeps its %d positions; want 2", m)
	}
}

func countPositions(t *testing.T, st storage.Storage, scope string) int {
	t.Helper()
	n := 0
	for _, err := range st.Positions().List(context.Background(), scope, storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	return n
}

// uploadAsTenant posts a file to an import route as the given tenant and
// requires the NDJSON stream to end without an error event.
func uploadAsTenant(t *testing.T, ts *httptest.Server, tenant, path, filename string, file []byte) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(file); err != nil {
		t.Fatal(err)
	}
	mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, &body)
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s as %s: status %d, %.300s", path, tenant, resp.StatusCode, out)
	}
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev map[string]any
		if json.Unmarshal(line, &ev) == nil && ev["error"] != nil {
			t.Fatalf("%s as %s: %v", path, tenant, ev["error"])
		}
	}
}
