package server

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// One gammonNet sweep per tenant (G.11, #239).
//
// Two of them do not go twice as fast: each asks for NumCPU goroutines, so
// they halve each other, and both write analyses into the rows the other is
// reading as missing. The second caller is told so — with an ordinary 409,
// BEFORE the NDJSON stream opens, because an error event inside a 200 is a
// failure a client has to parse a stream to discover.
func TestGammonNetSweep_OnePerTenant(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)

	// A sweep of an empty database finishes before a second request could
	// possibly race it, so the conflict is staged rather than raced: a job is
	// registered for the tenant, exactly as an in-flight sweep would, and the
	// route is then asked for another. A timing-dependent version of this test
	// would pass on a fast machine and say nothing.
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

	// That the exclusion is per TENANT and not global is asserted on the
	// registry below, not here: this server is backed by SQLite, which has
	// exactly one tenant (ADR-0005's SingleTenant guard refuses any other), so
	// "another tenant" is not something this handler can be shown.

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
