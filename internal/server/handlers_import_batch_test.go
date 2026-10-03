package server

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

var batchFixtures = []string{"match_with_comment.xg", "charlot1-charlot2_7p_2025-11-08-2305.xg"}

// must panics on the error of a Write into a buffer, which cannot happen.
func must(_ int, err error) {
	if err != nil {
		panic(err)
	}
}

func batchServer(t *testing.T, importDir string) (*httptest.Server, *Server, storage.Storage) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, Metrics: metrics.New(), ImportDir: importDir})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv, st
}

// batchDir copies the fixtures into a fresh directory, one of them in a
// subfolder, and returns it.
func batchDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i, name := range batchFixtures {
		b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(dir, name)
		if i == 1 {
			dst = filepath.Join(dir, "sub", name)
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// batchReply is the part of a response the tests read, taken before its body
// is closed.
type batchReply struct {
	StatusCode int
	Header     http.Header
}

func postBatch(t *testing.T, ts *httptest.Server, key string, body any) (batchReply, importBatchResp) {
	t.Helper()
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/imports.batch", &buf)
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set(IdempotencyKeyHeader, key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out importBatchResp
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return batchReply{resp.StatusCode, resp.Header}, out
}

func waitBatch(t *testing.T, srv *Server, id string) *batchStatus {
	t.Helper()
	job, ok := srv.batches.get(testTenant, id)
	if !ok {
		t.Fatalf("no job %s", id)
	}
	<-job.done
	return job.status()
}

func canonicalHashes(t *testing.T, st storage.Storage) []string {
	t.Helper()
	var out []string
	for m, err := range st.Matches().List(context.Background(), testTenant, storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, m.CanonicalHash)
	}
	sort.Strings(out)
	return out
}

func TestImportBatch_PathOff(t *testing.T) {
	ts, _, _ := batchServer(t, "")
	resp, _ := postBatch(t, ts, "", importBatchPathReq{Path: "."})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 while no --import-dir is set", resp.StatusCode)
	}
}

func TestImportBatch_PathOutsideRefused(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	ts, _, _ := batchServer(t, root)
	for _, p := range []string{outside, "../" + filepath.Base(outside), "link"} {
		resp, _ := postBatch(t, ts, "", importBatchPathReq{Path: p})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("path %q: status = %d, want 400", p, resp.StatusCode)
		}
	}
}

// The same folder, imported through the CLI's pipeline entry and through the
// server, gives the same matches by canonical hash.
func TestImportBatch_ParityWithCLI(t *testing.T) {
	dir := batchDir(t)

	dbPath := filepath.Join(t.TempDir(), "cli.db")
	db := database.NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatal(err)
	}
	files, err := ingest.CollectFiles(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ImportFiles(files, database.ImportFilesOptions{}); err != nil {
		t.Fatal(err)
	}
	db.Close()
	cliStore, err := sqlite.Open(context.Background(), dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cliStore.Close()
	want := canonicalHashes(t, cliStore)

	ts, srv, st := batchServer(t, filepath.Dir(dir))
	resp, out := postBatch(t, ts, "", importBatchPathReq{Path: filepath.Base(dir)})
	if resp.StatusCode != http.StatusOK || out.ImportID == "" || out.Files != 2 {
		t.Fatalf("imports.batch = %d %+v, want 200 with an id and 2 files", resp.StatusCode, out)
	}
	status := waitBatch(t, srv, out.ImportID)
	if status.State != batchStateDone || status.Progress.Imported != 2 || !status.Progress.Done {
		t.Fatalf("status = %+v, want done with 2 imported", status)
	}
	got := canonicalHashes(t, st)
	if len(want) != 2 || len(got) != 2 || want[0] != got[0] || want[1] != got[1] {
		t.Fatalf("canonical hashes: server %v, CLI %v", got, want)
	}
}

func TestImportBatch_ArchivesAndDuplicates(t *testing.T) {
	ts, srv, st := batchServer(t, "")
	dir := batchDir(t)

	var zbuf, tbuf bytes.Buffer
	zw, tw := zip.NewWriter(&zbuf), tar.NewWriter(&tbuf)
	for _, name := range batchFixtures {
		b, _ := os.ReadFile(filepath.Join("..", "..", "testdata", name))
		w, _ := zw.Create("corpus/" + name)
		must(w.Write(b))
		w, _ = zw.Create("corpus/notes.pdf")
		must(w.Write([]byte("not a match")))
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(b)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		must(tw.Write(b))
	}
	zw.Close()
	tw.Close()
	_ = dir

	upload := func(name string, data []byte, key string) (batchReply, importBatchResp) {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		fw, _ := mw.CreateFormFile("file", name)
		must(fw.Write(data))
		mw.Close()
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/imports.batch", &body)
		req.Header.Set(middleware.TenantHeader, testTenant)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		if key != "" {
			req.Header.Set(IdempotencyKeyHeader, key)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out importBatchResp
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return batchReply{resp.StatusCode, resp.Header}, out
	}

	resp, out := upload("c.zip", zbuf.Bytes(), "k1")
	if resp.StatusCode != http.StatusOK || out.Files != 2 {
		t.Fatalf("zip batch = %d %+v", resp.StatusCode, out)
	}
	if s := waitBatch(t, srv, out.ImportID); s.State != batchStateDone || s.Progress.Imported != 2 {
		t.Fatalf("zip status = %+v", s)
	}

	// The same key names the same job: no second import, the replay says so.
	resp, again := upload("c.zip", zbuf.Bytes(), "k1")
	if again.ImportID != out.ImportID || resp.Header.Get(idempotencyReplayedHeader) != "true" {
		t.Fatalf("replay = %+v %v, want id %s and the replayed header", again, resp.Header, out.ImportID)
	}

	// The tar holds the same matches: all duplicates, nothing new.
	_, out = upload("c.tar", tbuf.Bytes(), "")
	s := waitBatch(t, srv, out.ImportID)
	if s.State != batchStateDone || s.Progress.Duplicates != 2 || s.Progress.Imported != 0 {
		t.Fatalf("tar status = %+v, want 2 duplicates", s)
	}
	if n := len(canonicalHashes(t, st)); n != 2 {
		t.Fatalf("%d matches stored, want 2", n)
	}

	if resp, _ := upload("c.rar", []byte("x"), ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("rar status = %d, want 400", resp.StatusCode)
	}
}

func TestImportBatch_StatusAndCancelAreTenantScoped(t *testing.T) {
	root := batchDir(t)
	ts, srv, _ := batchServer(t, filepath.Dir(root))
	_, out := postBatch(t, ts, "", importBatchPathReq{Path: filepath.Base(root)})
	waitBatch(t, srv, out.ImportID)

	for _, route := range []string{"/v1/imports.batch.status", "/v1/imports.batch.cancel"} {
		body, _ := json.Marshal(importBatchIDReq{ImportID: out.ImportID})
		rec := serve(t, srv, context.Background(), "2", route, bytes.NewReader(body))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s from another tenant: %d, want 404", route, rec.Code)
		}
		rec = serve(t, srv, context.Background(), testTenant, route, bytes.NewReader(body))
		if rec.Code != http.StatusOK {
			t.Errorf("%s from the owner: %d, want 200", route, rec.Code)
		}
	}
}

func TestImportBatch_CancelBeforeStart(t *testing.T) {
	root := batchDir(t)
	_, srv, _ := batchServer(t, filepath.Dir(root))
	job := &batchJob{id: "x", scope: testTenant, state: batchStateRunning, done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	job.cancel = cancel
	srv.batches.jobs["x"] = job
	srv.imports.register("x", testTenant, cancel)
	body, _ := json.Marshal(importBatchIDReq{ImportID: "x"})
	if rec := serve(t, srv, context.Background(), testTenant, "/v1/imports.batch.cancel", bytes.NewReader(body)); rec.Code != http.StatusOK {
		t.Fatalf("cancel = %d", rec.Code)
	}
	if ctx.Err() == nil {
		t.Fatal("the job's context is not cancelled")
	}
}
