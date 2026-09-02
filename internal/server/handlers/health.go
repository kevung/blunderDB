// Package handlers holds the HTTP handlers for the blunderdb serve daemon.
// Ops handlers (health, readiness, metrics) live here; the domain handlers
// (positions, matches, …) land in PR2.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Health serves the liveness, readiness, and metrics endpoints. These are the
// only routes reachable without an X-Tenant-ID header.
//
// Liveness and readiness answer two different questions and must not be
// conflated: /healthz says "this process is up and serving HTTP", /readyz
// says "this process can currently do useful work". An orchestrator restarts
// a container whose liveness fails and merely stops routing traffic to one
// whose readiness fails — so a probe that reaches the database belongs in
// readiness only. Live once queried the storage; a database that was briefly
// unreachable then restarted a perfectly healthy daemon in a loop (#166).
type Health struct {
	Storage         storage.Storage
	Metrics         *metrics.Registry
	ExpectedVersion string
}

// Live answers GET /healthz: always 200. Reaching this handler is the proof
// that the process is alive and its HTTP server is accepting requests; it
// deliberately touches neither the storage nor anything else that can fail
// for reasons a restart would not cure.
func (h *Health) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready answers GET /readyz: 200 only when the backend is reachable AND its
// schema version matches the version this binary expects.
func (h *Health) Ready(w http.ResponseWriter, r *http.Request) {
	version, err := h.Storage.Version(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "down"})
		return
	}
	if h.ExpectedVersion != "" && version != h.ExpectedVersion {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "version_mismatch",
			"version":  version,
			"expected": h.ExpectedVersion,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready", "version": version})
}

// Expose answers GET /metrics with the Prometheus text exposition.
func (h *Health) Expose(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	h.Metrics.WritePrometheus(w)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
