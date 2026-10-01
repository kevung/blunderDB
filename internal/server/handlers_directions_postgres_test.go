//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run TestDirectionReads -v
package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestDirectionReads_Postgres(t *testing.T) {
	ts, srv := newPostgresTestServerAndHandler(t)
	f := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	checkDirectionReads(t, ts, f)
}

// TestDirectionReads_TenantIsolationPostgres: ids are global on PostgreSQL, so another
// tenant naming this tenant's Tournament or Rencontre reads nothing — a 404, even with the tag
// tenant 1 holds — and its lists hold only its own.
func TestDirectionReads_TenantIsolationPostgres(t *testing.T) {
	ts, srv := newPostgresTestServerAndHandler(t)
	mine := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	theirs := seedDirection(t, srv.opts.Storage, "2", "Open de Nice")

	for path, body := range directionReadCalls(mine) {
		if body == `{}` {
			continue
		}
		resp, b := readAs(t, ts, theirs.tenant, path, body, "")
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s as tenant 2 on tenant 1's id: status %d, body %.200s; want 404", path, resp.StatusCode, b)
		}
		mineResp, _ := readAs(t, ts, mine.tenant, path, body, "")
		if resp, _ := readAs(t, ts, theirs.tenant, path, body, mineResp.ETag); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s as tenant 2 with tenant 1's tag: status %d; want 404", path, resp.StatusCode)
		}
	}
	for _, path := range []string{"/v1/directions.list", "/v1/rencontres.list", "/v1/directions.directory"} {
		_, b := readAs(t, ts, theirs.tenant, path, `{}`, "")
		if s := string(b); containsID(s, mine.tournamentID) && path == "/v1/directions.list" || containsText(s, "Open de Lyon") {
			t.Errorf("%s as tenant 2 shows tenant 1's data: %.300s", path, s)
		}
	}
	_, b := readAs(t, ts, theirs.tenant, "/v1/directions.get", `{"tournamentId":`+strconv.FormatInt(theirs.tournamentID, 10)+`}`, "")
	if !containsText(string(b), "Joueur") {
		t.Errorf("tenant 2 cannot read its own Direction: %.300s", b)
	}
}

// containsID reports whether a JSON answer names a tournamentId.
func containsID(body string, id int64) bool {
	return strings.Contains(body, `"tournamentId":`+strconv.FormatInt(id, 10)+`,`) ||
		strings.Contains(body, `"tournamentId":`+strconv.FormatInt(id, 10)+`}`)
}

// containsText reports whether an answer contains s.
func containsText(body, s string) bool { return strings.Contains(body, s) }
