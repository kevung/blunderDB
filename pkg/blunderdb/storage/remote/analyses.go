package remote

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type positionIDReq struct {
	PositionID int64 `json:"positionId"`
}

func (s analysisStore) Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error {
	return s.c.call(ctx, scope, "analyses.save", analysisSaveReq{positionID, a}, nil)
}

// SaveMerged relays analyses.merge: the daemon folds the analysis into the
// stored one in its own transaction (storage.MergeSaver).
func (s analysisStore) SaveMerged(ctx context.Context, scope string, positionID int64, a domain.PositionAnalysis) error {
	return s.c.call(ctx, scope, "analyses.merge", analysisSaveReq{positionID, &a}, nil)
}

// LoadWithPlayed relays analyses.load with withPlayed: the daemon joins the
// moves the matches played, a table this backend does not serve
// (storage.PlayedLoader). A daemon that does not announce the join
// (storage.WithPlayedHeader) answered the bare analysis: that is refused as
// unsupported rather than shown as a position nobody played.
func (s analysisStore) LoadWithPlayed(ctx context.Context, scope string, positionID int64) (*domain.PositionAnalysis, error) {
	var a *domain.PositionAnalysis
	req := struct {
		PositionID int64 `json:"positionId"`
		WithPlayed bool  `json:"withPlayed"`
	}{positionID, true}
	h, err := s.c.callHeader(ctx, scope, "analyses.load", req, &a)
	if err != nil {
		return nil, err
	}
	if h.Get(storage.WithPlayedHeader) == "" {
		return nil, fmt.Errorf("%w: the daemon ignores withPlayed; upgrade it", unsupported("analyses.load withPlayed"))
	}
	return a, nil
}

type analysisSaveReq struct {
	PositionID int64                    `json:"positionId"`
	Analysis   *domain.PositionAnalysis `json:"analysis"`
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

// SaveMany has no batch route: it asks analyses.save once per analysis.
func (s analysisStore) SaveMany(ctx context.Context, scope string, analyses map[int64]*domain.PositionAnalysis) error {
	return storage.SaveAnalysesEach(ctx, s, scope, analyses)
}
