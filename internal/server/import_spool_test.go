package server

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func spoolServer(t *testing.T, maxSpool int64) (*httptest.Server, *Server) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "spool.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	srv, err := New(Options{Storage: st, Metrics: metrics.New(), MaxSpoolBytes: maxSpool})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv
}

func spoolRepoXG(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func spoolPost(t *testing.T, ts *httptest.Server, path, filename string, file []byte) int {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write(file)
	_ = mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(middleware.TenantHeader, testTenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

// A small upload reserves what it can hold on disk, not the largest upload the
// server accepts: an instance is not limited to four imports by their maximum.
func TestImportSpoolReservesTheUploadNotTheMaximum(t *testing.T) {
	xg := spoolRepoXG(t)
	ts, srv := spoolServer(t, int64(4*len(xg)))
	if got := spoolPost(t, ts, "/v1/imports.xg", "a.xg", xg); got != http.StatusOK {
		t.Fatalf("upload within the spool: status %d, want 200", got)
	}
	if got := srv.spool.usage(); got != 0 {
		t.Errorf("spool still holds %d bytes after the import", got)
	}
}

func TestImportSpoolRefusesWhatDoesNotFit(t *testing.T) {
	xg := spoolRepoXG(t)
	ts, srv := spoolServer(t, int64(len(xg)/2))
	if got := spoolPost(t, ts, "/v1/imports.xg", "a.xg", xg); got != http.StatusTooManyRequests {
		t.Fatalf("upload beyond the spool: status %d, want 429", got)
	}
	if got := srv.spool.usage(); got != 0 {
		t.Errorf("a refused upload left %d bytes reserved", got)
	}
}

func TestServeMaxSpoolBytes(t *testing.T) {
	cfg, err := parseServeArgs([]string{"--max-spool-bytes", "12345"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.maxSpool != 12345 {
		t.Errorf("maxSpool = %d, want 12345", cfg.maxSpool)
	}
	if _, err := parseServeArgs([]string{"--max-spool-bytes", "-1"}); err == nil {
		t.Error("a negative spool bound must be refused")
	}
}

func spoolTar(t *testing.T, name string, file []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(file)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write(file)
	_ = tw.Close()
	return buf.Bytes()
}

// A batch holds the bytes it unpacked while it runs: two batches fit where the
// worst case of one (three times the largest upload) would not.
func TestImportBatchSpoolHoldsWhatItUnpacked(t *testing.T) {
	archive := spoolTar(t, "a.xg", spoolRepoXG(t))
	ts, srv := spoolServer(t, int64(5*len(archive)))
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "lot.tar")
	_, _ = fw.Write(archive)
	_ = mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/imports.batch", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(middleware.TenantHeader, testTenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out importBatchResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode/100 != 2 {
		t.Fatalf("batch within the spool: status %d", resp.StatusCode)
	}
	waitBatch(t, srv, out.ImportID)
	job, _ := srv.batches.get(testTenant, out.ImportID)
	<-job.done
	if got := srv.spool.usage(); got != 0 {
		t.Errorf("spool still holds %d bytes after the batch", got)
	}
}

func spoolEvents(t *testing.T, ts *httptest.Server, path, filename string, file []byte) []map[string]any {
	t.Helper()
	return uploadImportNamed(t, ts, path, filename, file)
}

// The file-by-file stream says why in a stable code, its text kept.
func TestImportStreamErrorCarriesItsCode(t *testing.T) {
	ts, _ := spoolServer(t, 0)
	xg := spoolRepoXG(t)
	events := spoolEvents(t, ts, "/v1/imports.xg", "a.xg", xg[:len(xg)/2])
	last := events[len(events)-1]
	if last["event"] != "error" || last["import_code"] != domain.ImportErrCorrupt {
		t.Errorf("last event %v, want an error of code %s", last, domain.ImportErrCorrupt)
	}
	if last["error"] == nil {
		t.Error("the error body must stay beside the code")
	}
}

// A batch names each refused file's code, in its status and in its report.
func TestImportBatchFileErrorsCarryTheirCode(t *testing.T) {
	ts, srv := spoolServer(t, 0)
	xg := spoolRepoXG(t)
	archive := spoolTar(t, "cut.xg", xg[:len(xg)/2])
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "lot.tar")
	_, _ = fw.Write(archive)
	_ = mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/imports.batch", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(middleware.TenantHeader, testTenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out importBatchResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	st := waitBatch(t, srv, out.ImportID)
	if len(st.Errors) != 1 || st.Errors[0].Code != domain.ImportErrCorrupt || st.Errors[0].Error == "" {
		t.Fatalf("errors %+v, want one of code %s", st.Errors, domain.ImportErrCorrupt)
	}
	batch, err := srv.opts.Storage.ImportBatches().Load(context.Background(), testTenant, st.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Report.Failures) != 1 || batch.Report.Failures[0].Code != domain.ImportErrCorrupt {
		t.Errorf("report failures %+v", batch.Report.Failures)
	}
}

// report=deferred leaves the report to imports.report; the default keeps it in
// the terminal event.
func TestImportReportDeferred(t *testing.T) {
	ts, _ := spoolServer(t, 0)
	xg := spoolRepoXG(t)
	events := spoolEvents(t, ts, "/v1/imports.xg", "a.xg", xg)
	if _, ok := events[len(events)-1]["report"]; !ok {
		t.Errorf("default terminal event has no report: %v", events[len(events)-1])
	}
	ts2, _ := spoolServer(t, 0)
	events = spoolEvents(t, ts2, "/v1/imports.xg?report=deferred", "a.xg", xg)
	last := events[len(events)-1]
	if _, ok := last["report"]; ok || last["batch_id"] == nil {
		t.Errorf("deferred terminal event: %v, want a batch id and no report", last)
	}
}
