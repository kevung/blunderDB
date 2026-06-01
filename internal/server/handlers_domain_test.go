package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

const testTenant = "tenant-a"

// post issues a POST to the daemon with the tenant header and a JSON body.
func post(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestPositionsRoundtrip(t *testing.T) {
	ts := newTestServer(t)

	// Save a position.
	p := domain.InitializePosition()
	resp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save status = %d, want 200", resp.StatusCode)
	}
	var saved idResp
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID == 0 {
		t.Fatal("save returned id 0")
	}

	// Load it back.
	resp2 := post(t, ts, "/v1/positions.load", idReq(saved))
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("load status = %d, want 200", resp2.StatusCode)
	}
	var got domain.Position
	if err := json.NewDecoder(resp2.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != saved.ID {
		t.Fatalf("loaded id = %d, want %d", got.ID, saved.ID)
	}
}

func TestPositionsListNDJSON(t *testing.T) {
	ts := newTestServer(t)
	p := domain.InitializePosition()
	post(t, ts, "/v1/positions.save", positionReq{Position: &p}).Body.Close()

	resp := post(t, ts, "/v1/positions.list", listReq{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != ndjsonContentType {
		t.Fatalf("content-type = %q, want %q", ct, ndjsonContentType)
	}
	n := 0
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("ndjson line not JSON: %q (%v)", line, err)
		}
		n++
	}
	if n != 1 {
		t.Fatalf("ndjson rows = %d, want 1", n)
	}
}

func TestLoadMissingReturnsNotFound(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/positions.load", idReq{ID: 999999})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeNotFound {
		t.Fatalf("code = %q, want %q", env.Error.Code, CodeNotFound)
	}
}

func TestVoidHandlerReturnsOK(t *testing.T) {
	ts := newTestServer(t)
	// Deleting a non-existent position is a no-op that succeeds.
	resp := post(t, ts, "/v1/positions.delete", idReq{ID: 12345})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var ok okResp
	if err := json.NewDecoder(resp.Body).Decode(&ok); err != nil {
		t.Fatal(err)
	}
	if !ok.OK {
		t.Fatal("expected ok=true")
	}
}

func TestMetadataCounts(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/metadata.counts", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["positions"]; !ok {
		t.Fatalf("counts missing 'positions' field: %v", body)
	}
}

func TestInvalidJSONBody(t *testing.T) {
	ts := newTestServer(t)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/positions.load", strings.NewReader("{not json"))
	req.Header.Set(middleware.TenantHeader, testTenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeInvalid {
		t.Fatalf("code = %q, want %q", env.Error.Code, CodeInvalid)
	}
}
