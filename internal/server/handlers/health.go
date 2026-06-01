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
type Health struct {
	Storage         storage.Storage
	Metrics         *metrics.Registry
	ExpectedVersion string
}

// Live answers GET /healthz: 200 when the storage backend answers a trivial
// query (Version performs a SELECT), 503 otherwise.
func (h *Health) Live(w http.ResponseWriter, r *http.Request) {
	if _, err := h.Storage.Version(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "down"})
		return
	}
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
