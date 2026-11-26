package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document"
)

// A Relevé is imported into its Collection, a second import adds nothing, and
// an unverified entry is a 400 that writes nothing.
func TestDocumentsImportReleve(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	post := func(body string) (int, string) {
		rec := serveAcross(t, srv, "1", "", "/v1/documents.importReleve", body)
		return rec.Code, rec.Body.String()
	}
	releve, err := os.ReadFile("../../pkg/blunderdb/document/testdata/releve-exemple.json")
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		CollectionID int64 `json:"collectionId"`
		Positions    int   `json:"positions"`
		Added        int   `json:"added"`
		Repeated     int   `json:"repeated"`
	}
	code, out := post(`{"releve":` + string(releve) + `}`)
	if code != http.StatusOK || json.Unmarshal([]byte(out), &res) != nil || res.Positions != 3 || res.Added != 3 || res.Repeated != 1 {
		t.Fatalf("import: %d %s", code, out)
	}
	code, out = post(`{"releve":` + string(releve) + `}`)
	if code != http.StatusOK || json.Unmarshal([]byte(out), &res) != nil || res.Added != 0 {
		t.Fatalf("second import: %d %s", code, out)
	}

	unverified := strings.Replace(string(releve), `"verified": true`, `"verified": false`, 1)
	unverified = strings.Replace(unverified, "Exemples inventés", "Autre", 1)
	if code, out = post(`{"releve":` + unverified + `}`); code != http.StatusBadRequest {
		t.Errorf("unverified entry: %d %s; want 400", code, out)
	}
	if code, out = post(`{"releve":{"format":"autre"}}`); code != http.StatusBadRequest {
		t.Errorf("malformed relevé: %d %s; want 400", code, out)
	}
	code, out = post(`{"releve":` + unverified + `,"force":true}`)
	if code != http.StatusOK || json.Unmarshal([]byte(out), &res) != nil {
		t.Fatalf("forced import: %d %s", code, out)
	}
	out = serveAcross(t, srv, "1", "", "/v1/collections.list", `{}`).Body.String()
	if strings.Count(out, `"name":`) != 2 {
		t.Errorf("collections after the refusals: %s; want 2", out)
	}
}

// The daemon imports, under the context its position carries, an entry
// whose checkers are certified and whose context is to verify when the
// caller asks for the fallback, and refuses it otherwise.
func TestDocumentsImportReleveFallback(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	d := document.StructureDefaults
	d.Confirmed = true
	r := &document.Releve{Format: document.Format, Version: document.Version, Document: document.Source{Title: "Repli", Defaults: &d},
		Entries: []document.Entry{{Rank: 1, Page: 1, Position: "XGID=-b----E-C---eE---c-e----B-:" + document.StructureDefaults.Context, Checkers: document.Sure,
			Context: document.ContextToVerify, Doubts: []capture.Doubt{document.ContextDoubt, {Field: "dice", Reason: "dice drawn, not counted"}}}}}
	data, err := r.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/documents.importReleve", `{"releve":`+string(data)+`}`); rec.Code != http.StatusBadRequest {
		t.Errorf("without fallback: %d %s; want 400", rec.Code, rec.Body.String())
	}
	rec := serveAcross(t, srv, "1", "", "/v1/documents.importReleve", `{"releve":`+string(data)+`,"fallback":true}`)
	var res struct {
		Fallback int `json:"fallback"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &res) != nil || res.Fallback != 1 {
		t.Errorf("with fallback: %d %s", rec.Code, rec.Body.String())
	}
}
