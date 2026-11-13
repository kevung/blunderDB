//go:build postgres

package server

import "testing"

func TestSearchPageForgedCursorIsBadRequestPostgres(t *testing.T) {
	ts, srv := newPostgresTestServerAndHandler(t)
	checkForgedCursorIsBadRequest(t, ts, srv.opts.Storage, testTenant)
}
