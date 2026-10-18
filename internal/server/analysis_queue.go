package server

import (
	"context"
	"sync"
)

// sweepProgress is the counters of one tenant's gammonNet sweep, the running
// one or, once it ended, the last one since the daemon started.
type sweepProgress struct {
	// State is "running", "done", "cancelled", "quota_exceeded" or "error".
	State     string `json:"state"`
	Total     int    `json:"total"`
	Done      int    `json:"done"`
	Evaluated int    `json:"evaluated"`
	Refused   int    `json:"refused"`
	Failed    int    `json:"failed"`
}

// sweepBoard keeps the latest sweepProgress per tenant. It is the only state
// the queue gauges need that the library cannot give: the library says what
// still lacks an analysis, the board says what the current run is doing.
type sweepBoard struct {
	mu   sync.Mutex
	last map[string]sweepProgress
}

func (b *sweepBoard) set(scope string, p sweepProgress) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.last == nil {
		b.last = make(map[string]sweepProgress)
	}
	b.last[scope] = p
}

func (b *sweepBoard) get(scope string) (sweepProgress, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p, ok := b.last[scope]
	return p, ok
}

// analysisQueueResp answers analyses.queueStatus.
type analysisQueueResp struct {
	// Positions is every position of the tenant; Analysed those with an
	// analysis from any engine; Pending those without one (the work an
	// analyzeMissing sweep would take).
	Positions int `json:"positions"`
	Analysed  int `json:"analysed"`
	Pending   int `json:"pending"`
	// Running is true while a sweep of this tenant is in flight.
	Running bool `json:"running"`
	// Sweep is the running sweep or, when none runs, the last one since the
	// daemon started; null when there has been none.
	Sweep *sweepProgress `json:"sweep"`
}

func (s *Server) analysisQueueStatus(ctx context.Context, scope string, _ struct{}) (analysisQueueResp, error) {
	st := s.opts.Storage
	total, err := st.Positions().Count(ctx, scope)
	if err != nil {
		return analysisQueueResp{}, err
	}
	missing, err := st.Analyses().CountWithoutAnalysis(ctx, scope)
	if err != nil {
		return analysisQueueResp{}, err
	}
	resp := analysisQueueResp{Positions: total, Pending: missing, Analysed: total - missing}
	if p, ok := s.sweeps.get(scope); ok {
		resp.Sweep = &p
		resp.Running = p.State == "running"
	}
	return resp, nil
}
