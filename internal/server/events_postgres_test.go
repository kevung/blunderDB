//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run TestEvents -v
package server

import "testing"

// TestEvents_Postgres: the event checks over PostgreSQL, where ids are global and the tenant
// is a column — a subscriber still hears its own tenant only.
func TestEvents_Postgres(t *testing.T) {
	_, pgSrv := newPostgresTestServerAndHandler(t)
	checkEvents(t, pgSrv.opts.Storage)
}
