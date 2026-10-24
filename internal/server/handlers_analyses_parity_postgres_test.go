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
	analysesMergeParity(t, ts, srv.opts.Storage, "1")
}
