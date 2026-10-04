package server

import (
	"context"
	"net/http"
)

// tenantQuotaResp is what tenants.quota answers: the bounds and the calling
// tenant's use of them.
type tenantQuotaResp struct {
	Limits TenantQuotas     `json:"limits"`
	Usage  tenantQuotaUsage `json:"usage"`
}

type tenantQuotaUsage struct {
	Positions            int     `json:"positions"`
	StoredBytes          int64   `json:"storedBytes"`
	AnalysisSecondsToday float64 `json:"analysisSecondsToday"`
	ImportsInFlight      int     `json:"importsInFlight"`
}

func (s *Server) tenantQuotaRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/tenants.quota", rpc(func(ctx context.Context, scope string, _ struct{}) (tenantQuotaResp, error) {
			counts, err := s.opts.Storage.Metadata().Counts(ctx, scope)
			if err != nil {
				return tenantQuotaResp{}, err
			}
			stored, err := s.opts.Storage.Metadata().StoredBytes(ctx, scope)
			if err != nil {
				return tenantQuotaResp{}, err
			}
			spent, imports := s.quota.usage(scope)
			return tenantQuotaResp{Limits: s.quota.limits, Usage: tenantQuotaUsage{
				Positions: counts.Positions, StoredBytes: stored, AnalysisSecondsToday: spent.Seconds(), ImportsInFlight: imports,
			}}, nil
		})},
	}
}
