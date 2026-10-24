//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run DatabaseParity -v
package server

import "testing"

// TestAnalysesSaveLoad_DatabaseParity_Postgres is the SQLite parity case
// against a PostgreSQL daemon.
func TestAnalysesSaveLoad_DatabaseParity_Postgres(t *testing.T) {
	ts, srv := newPostgresTestServerAndHandlerWith(t, func(*Options) {})
	want := databaseAnalysisParity(t)
	if got := daemonAnalysisParity(t, ts, srv.opts.Storage, "1"); got != want {
		t.Errorf("daemon analysis differs from the Database wrapper's:\n daemon: %s\nwrapper: %s", got, want)
	}
}
