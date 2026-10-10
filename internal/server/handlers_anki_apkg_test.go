package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/apkg"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestExportApkgRoute: the daemon serves the package the CLI writes, from a
// living collection evaluated at export.
func TestExportApkgRoute(t *testing.T) {
	ts := newTestServer(t)
	for i := range 3 {
		p := domain.InitializePosition()
		p.Board.Points[6].Checkers = i + 2
		post(t, ts, "/v1/positions.save", positionReq{Position: &p}).Body.Close()
	}
	resp := post(t, ts, "/v1/collections.create", collectionCreateReq{Name: "Vivante", FilterQuery: "s n<1"})
	var col idResp
	if err := json.NewDecoder(resp.Body).Decode(&col); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp = post(t, ts, "/v1/anki.exportApkg", exportApkgReq{Source: apkg.Source{CollectionID: col.ID}, Language: "fr"})
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, body)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "Vivante.apkg") {
		t.Errorf("Content-Disposition = %q", cd)
	}
	if n := resp.Header.Get("X-Anki-Notes"); n != "3" {
		t.Errorf("X-Anki-Notes = %q", n)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	if len(zr.File) != 3+2 {
		t.Errorf("%d entries, want the collection, the media index and 3 boards", len(zr.File))
	}

	resp = post(t, ts, "/v1/anki.exportApkg", exportApkgReq{})
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("no source: want an error")
	}
}
