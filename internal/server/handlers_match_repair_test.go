package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
)

// postRepair uploads file as the source of match id to matches.repair.
func postRepair(t *testing.T, ts *httptest.Server, id int64, name string, file []byte, dryRun bool) (*http.Response, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(file); err != nil {
		t.Fatal(err)
	}
	_ = mw.WriteField("id", strconv.FormatInt(id, 10))
	_ = mw.WriteField("dryRun", strconv.FormatBool(dryRun))
	mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/matches.repair", &body)
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func TestMatchRepairUpload(t *testing.T) {
	ts := newTestServer(t)
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "test.xg"))
	if err != nil {
		t.Fatal(err)
	}
	events := uploadImportNamed(t, ts, "/v1/imports.xg", "test.xg", src)
	id := int64(events[len(events)-1]["match_id"].(float64))

	resp, rep := postRepair(t, ts, id, "test.xg", src, true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d (%v), want 200", resp.StatusCode, rep)
	}
	if rep["changed_moves"].(float64) != 0 || rep["dry_run"] != true {
		t.Errorf("a freshly imported match differs from its file: %v", rep)
	}

	other, err := os.ReadFile(filepath.Join("..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg"))
	if err != nil {
		t.Fatal(err)
	}
	if resp, body := postRepair(t, ts, id, "other.xg", other, false); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("another match's file: status = %d (%v), want 400", resp.StatusCode, body)
	}
}
