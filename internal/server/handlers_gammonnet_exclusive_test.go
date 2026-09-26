package server

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// One gammonNet sweep per tenant: two would halve each other (NumCPU each)
// and write into rows the other reads as missing. The second gets a plain
// 409 before the NDJSON stream opens, not an error event inside a 200.
func TestGammonNetSweep_OnePerTenant(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)

	// Staged, not raced: an empty sweep ends too fast, and a timing-dependent
	// test would prove nothing.
	held, err := srv.gammonnetJobs.startExclusive(testTenant, func() {})
	if err != nil {
		t.Fatal(err)
	}

	resp := post(t, ts, "/v1/gammonnet.analyzeMissing", map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, want 409 — a second sweep for the same tenant must be refused", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "ndjson") {
		t.Errorf("the refusal opened an NDJSON stream (%s): a client should not have to parse one to learn it was refused", ct)
	}
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "already running") {
		t.Errorf("the refusal does not say why: %s", raw)
	}

	// Per-tenant exclusion is asserted on the registry: SQLite has only one
	// tenant (ADR-0005).

	// Finishing releases it.
	srv.gammonnetJobs.finish(held)
	again := post(t, ts, "/v1/gammonnet.analyzeMissing", map[string]any{})
	defer again.Body.Close()
	if again.StatusCode != http.StatusOK {
		t.Fatalf("the tenant was still blocked after its job finished: %d", again.StatusCode)
	}
	_, _ = io.Copy(io.Discard, again.Body)
}

// The registry itself, without the HTTP dance: a tenant with a job in flight is
// refused a second, another tenant is not, and finishing releases it.
func TestImportRegistry_StartExclusive(t *testing.T) {
	reg := newImportRegistry()
	noop := func() {}

	id, err := reg.startExclusive("t1", noop)
	if err != nil {
		t.Fatalf("the first job was refused: %v", err)
	}
	if _, err := reg.startExclusive("t1", noop); err == nil {
		t.Error("a second job for the same tenant was allowed")
	}
	other, err := reg.startExclusive("t2", noop)
	if err != nil {
		t.Fatalf("another tenant was refused: %v", err)
	}
	reg.finish(id)
	again, err := reg.startExclusive("t1", noop)
	if err != nil {
		t.Fatalf("the tenant was still blocked after its job finished: %v", err)
	}
	reg.finish(again)
	reg.finish(other)

	// start (non-exclusive) is unchanged: imports may run several at a time.
	a := reg.start("t1", noop)
	b := reg.start("t1", noop)
	if a == b {
		t.Error("two concurrent imports got the same id")
	}
}
