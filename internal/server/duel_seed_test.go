package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestDuelSeedLeavesByNoRoute: while a Duel is open, no route of the daemon —
// reads, exports and every other gesture, under {} and under the Duel's id —
// answers with its seed, which is every roll to come (ADR-0072 rule 11). The
// walk is driven by the route table, so a route added later is covered too.
func TestDuelSeedLeavesByNoRoute(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{
		Storage: st, Duel: true, Transcription: true, EnableDirection: true,
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	status, body := gesture(t, ts, testTenant, "/v1/duels.create", 0, map[string]any{"matchLength": 3})
	created := duelState(t, status, body)
	row, err := st.Duels().Get(ctx, testTenant, created.ID)
	if err != nil || row.DiceSeed == "" {
		t.Fatalf("the draft and its seed: %+v, %v", row, err)
	}
	seed := []byte(row.DiceSeed)

	idBody := fmt.Sprintf(`{"id":%d}`, created.ID)
	for _, path := range srv.Paths() {
		if path == "/ops/tenant.purge" {
			continue // it would take the Duel away, not reveal it
		}
		for _, payload := range []string{"{}", idBody} {
			resp := doPost(t, ts.URL+path, testTenant, payload)
			got, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if bytes.Contains(got, seed) || bytes.Contains(got, []byte(strings.ToUpper(row.DiceSeed))) {
				t.Errorf("%s %s answers with the seed of the open Duel", path, payload)
			}
			for k, vs := range resp.Header {
				if strings.Contains(strings.Join(vs, " "), row.DiceSeed) {
					t.Errorf("%s %s: header %s carries the seed", path, payload, k)
				}
			}
		}
	}

	// The walk saw the Duel throughout: it is still there, still unended.
	status, body = gesture(t, ts, testTenant, "/v1/duels.get", 0, map[string]any{"id": created.ID})
	if status != http.StatusOK {
		t.Fatalf("the Duel after the walk: status %d: %s", status, body)
	}
	var after struct {
		Ended json.RawMessage `json:"ended"`
	}
	if err := json.Unmarshal(body, &after); err != nil || len(after.Ended) != 0 && string(after.Ended) != "null" {
		t.Fatalf("the Duel ended during the walk: %s", body)
	}
}
