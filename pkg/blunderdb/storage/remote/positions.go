package remote

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type positionReq struct {
	Position *domain.Position `json:"position"`
}

type idReq struct {
	ID int64 `json:"id"`
}

type idsReq struct {
	IDs []int64 `json:"ids"`
}

type listReq struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type savedResp struct {
	ID      int64 `json:"id"`
	Created bool  `json:"created"`
}

func (s positionStore) Save(ctx context.Context, scope string, p *domain.Position) (int64, error) {
	id, _, err := s.SaveCreated(ctx, scope, p)
	return id, err
}

func (s positionStore) SaveCreated(ctx context.Context, scope string, p *domain.Position) (int64, bool, error) {
	var r savedResp
	if err := s.c.call(ctx, scope, "positions.save", positionReq{p}, &r); err != nil {
		return 0, false, err
	}
	p.ID = r.ID
	return r.ID, r.Created, nil
}

func (s positionStore) SetIndividuallyImported(ctx context.Context, scope string, id int64, value bool) error {
	return s.c.call(ctx, scope, "positions.setIndividuallyImported", struct {
		ID    int64 `json:"id"`
		Value bool  `json:"value"`
	}{id, value}, nil)
}

func (s positionStore) Update(ctx context.Context, scope string, p *domain.Position) error {
	return s.c.call(ctx, scope, "positions.update", positionReq{p}, nil)
}

func (s positionStore) Load(ctx context.Context, scope string, id int64) (*domain.Position, error) {
	var p *domain.Position
	if err := s.c.call(ctx, scope, "positions.load", idReq{id}, &p); err != nil {
		return nil, err
	}
	return p, nil
}

type existsResp struct {
	ID    int64 `json:"id"`
	Found bool  `json:"found"`
}

func (s positionStore) Exists(ctx context.Context, scope string, zobrist uint64) (int64, bool, error) {
	var r existsResp
	err := s.c.call(ctx, scope, "positions.exists", struct {
		Zobrist uint64 `json:"zobrist"`
	}{zobrist}, &r)
	return r.ID, r.Found, err
}

// ExistsMany has no batch route: it asks positions.exists once per hash.
func (s positionStore) ExistsMany(ctx context.Context, scope string, zobrists []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64, len(zobrists))
	for _, z := range zobrists {
		id, found, err := s.Exists(ctx, scope, z)
		if err != nil {
			return nil, err
		}
		if found {
			out[z] = id
		}
	}
	return out, nil
}

func (s positionStore) Delete(ctx context.Context, scope string, id int64) error {
	return s.c.call(ctx, scope, "positions.delete", idReq{id}, nil)
}

func (s positionStore) List(ctx context.Context, scope string, opts storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return stream[*domain.Position](ctx, s.c, scope, "positions.list", listReq{opts.Limit, opts.Offset})
}

func (s positionStore) ListIDs(ctx context.Context, scope string, opts storage.ListOpts) ([]int64, error) {
	var ids []int64
	err := s.c.call(ctx, scope, "positions.listIds", listReq{opts.Limit, opts.Offset}, &ids)
	return ids, err
}

func (s positionStore) Count(ctx context.Context, scope string) (int, error) {
	var n int
	err := s.c.call(ctx, scope, "positions.count", nil, &n)
	return n, err
}

// IndexOf reads positions.indexOf, which answers -1 for an id it does not hold.
func (s positionStore) IndexOf(ctx context.Context, scope string, id int64) (int, bool, error) {
	var i int
	if err := s.c.call(ctx, scope, "positions.indexOf", idReq{id}, &i); err != nil {
		return 0, false, err
	}
	if i < 0 {
		return 0, false, nil
	}
	return i, true, nil
}

func (s positionStore) LoadByIDs(ctx context.Context, scope string, ids []int64) ([]domain.Position, error) {
	var ps []domain.Position
	err := s.c.call(ctx, scope, "positions.loadByIds", idsReq{ids}, &ps)
	return ps, err
}

func (s positionStore) ReclassifyDerived(ctx context.Context, scope string) (int, error) {
	var r struct {
		Reclassified int `json:"reclassified"`
	}
	err := s.c.call(ctx, scope, "positions.reclassifyPhases", nil, &r)
	return r.Reclassified, err
}

func (s positionStore) RepairCrawfordSentinel(ctx context.Context, scope string) (int, error) {
	var r struct {
		Repaired int `json:"repaired"`
	}
	err := s.c.call(ctx, scope, "positions.repairCrawford", nil, &r)
	return r.Repaired, err
}
