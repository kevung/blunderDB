//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run TestDirectionGestures -v
package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// TestDirectionGestures_RacePostgres: on a pooled backend too, two gestures on one reading
// apply once — the comparison runs under the gesture's lock, not on one connection.
func TestDirectionGestures_RacePostgres(t *testing.T) {
	_, pgSrv := newPostgresTestServerAndHandler(t)
	ts, srv := gestureServerOn(t, pgSrv.opts.Storage)
	checkRaceOneConflict(t, ts, seedDirection(t, srv.opts.Storage, "1", "Open de Lyon"))
}

// TestDirectionGestures_TenantIsolationPostgres: ids are global on PostgreSQL; another tenant
// naming this tenant's Direction or Rencontre writes nothing.
func TestDirectionGestures_TenantIsolationPostgres(t *testing.T) {
	_, pgSrv := newPostgresTestServerAndHandler(t)
	ts, srv := gestureServerOn(t, pgSrv.opts.Storage)
	mine := seedDirection(t, srv.opts.Storage, "1", "Open de Lyon")
	theirs := seedDirection(t, srv.opts.Storage, "2", "Open de Nice")
	checkGestureIsolation(t, ts, mine, theirs)
	// Tenant 2 still writes its own.
	if r := send(t, ts, theirs.tenant, "/v1/directions.addNote", theirs.body(`"text":"ok"`), theirs.versionOf(t, ts), ""); r.status != http.StatusOK {
		t.Errorf("tenant 2 on its own Direction: status %d, body %.200s", r.status, r.body)
	}
}

// checkGestureIsolation: a tenant naming another's Direction or Rencontre — even with the
// version that tenant read — writes nothing and gets 404.
func checkGestureIsolation(t *testing.T, ts *httptest.Server, mine, theirs directedFixture) {
	t.Helper()
	v := mine.versionOf(t, ts)
	m := mine.running(t)[0]
	calls := map[string]string{
		"/v1/directions.enterResult": mine.resultBody(m),
		"/v1/directions.addNote":     mine.body(`"text":"intrus"`),
		"/v1/rencontres.setBreaks":   `{"id":` + strconv.FormatInt(mine.rencontreID, 10) + `,"breaks":[]}`,
		"/v1/rencontres.trash":       `{"id":` + strconv.FormatInt(mine.rencontreID, 10) + `}`,
	}
	for path, body := range calls {
		if r := send(t, ts, theirs.tenant, path, body, v, ""); r.status != http.StatusNotFound {
			t.Errorf("%s as tenant %s on tenant %s's id: status %d, body %.200s; want 404", path, theirs.tenant, mine.tenant, r.status, r.body)
		}
	}
	if got := mine.versionOf(t, ts); got != v {
		t.Errorf("tenant %s's Direction moved under another tenant's gestures", mine.tenant)
	}
}
