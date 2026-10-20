package remote

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

type positionIDReq struct {
	PositionID int64 `json:"positionId"`
}

func (s analysisStore) Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error {
	return s.c.call(ctx, scope, "analyses.save", struct {
		PositionID int64                    `json:"positionId"`
		Analysis   *domain.PositionAnalysis `json:"analysis"`
	}{positionID, a}, nil)
}

func (s analysisStore) Load(ctx context.Context, scope string, positionID int64) (*domain.PositionAnalysis, error) {
	var a *domain.PositionAnalysis
	if err := s.c.call(ctx, scope, "analyses.load", positionIDReq{positionID}, &a); err != nil {
		return nil, err
	}
	return a, nil
}

// LoadMany reads analyses.loadByIds, which answers a list carrying each
// analysis's position id and skips the positions without one.
func (s analysisStore) LoadMany(ctx context.Context, scope string, ids []int64) (map[int64]*domain.PositionAnalysis, error) {
	var list []domain.PositionAnalysis
	if err := s.c.call(ctx, scope, "analyses.loadByIds", idsReq{ids}, &list); err != nil {
		return nil, err
	}
	out := make(map[int64]*domain.PositionAnalysis, len(list))
	for i := range list {
		out[int64(list[i].PositionID)] = &list[i]
	}
	return out, nil
}

func (s analysisStore) Delete(ctx context.Context, scope string, positionID int64) error {
	return s.c.call(ctx, scope, "analyses.delete", positionIDReq{positionID}, nil)
}

func (s analysisStore) RepairDenormalisedColumns(ctx context.Context, scope string) (int, error) {
	var r struct {
		Repaired int `json:"repaired"`
	}
	err := s.c.call(ctx, scope, "analyses.repair", nil, &r)
	return r.Repaired, err
}
