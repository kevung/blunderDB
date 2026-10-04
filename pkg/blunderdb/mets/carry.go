package mets

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A table travels with the analyses that cite it (ADR-0068, rule 5): an
// export, an import or a merge of databases copies each cited table into the
// receiver, where tables are merged by digest and arrive not current, and
// gives the analysis the receiver's id for it.

// ReadTables returns every table of a source with its text.
func ReadTables(ctx context.Context, src storage.MatchEquityTableStore, scope string) ([]*domain.MatchEquityTable, error) {
	list, err := src.List(ctx, scope)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.MatchEquityTable, 0, len(list))
	for _, t := range list {
		full, err := src.Load(ctx, scope, t.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, full)
	}
	return out, nil
}

// Carrier maps a source's table ids onto a receiver's, storing each table in
// the receiver the first time an analysis cites it.
type Carrier struct {
	dst    storage.MatchEquityTableStore
	scope  string
	tables map[int64]*domain.MatchEquityTable
	target map[int64]int64
}

// NewCarrier carries the source tables into dst's scope. Save never sets the
// current flag, so the receiver's choice of table is left as it was
// (ADR-0007).
func NewCarrier(dst storage.MatchEquityTableStore, scope string, tables []*domain.MatchEquityTable) *Carrier {
	c := &Carrier{dst: dst, scope: scope, tables: map[int64]*domain.MatchEquityTable{}, target: map[int64]int64{}}
	for _, t := range tables {
		c.tables[t.ID] = t
	}
	return c
}

// Add makes one more source table known, for a source that reveals its
// tables as it is read (an NDJSON stream).
func (c *Carrier) Add(t *domain.MatchEquityTable) {
	c.tables[t.ID] = t
}

// Target returns the receiver's id for the source table srcID: 0 for the
// built-in table, whether the source names it by 0 or by its digest.
func (c *Carrier) Target(ctx context.Context, srcID int64) (int64, error) {
	if srcID == 0 {
		return 0, nil
	}
	if id, ok := c.target[srcID]; ok {
		return id, nil
	}
	t, ok := c.tables[srcID]
	if !ok {
		return 0, fmt.Errorf("source match equity table %d: %w", srcID, storage.ErrNotFound)
	}
	var id int64
	if t.Digest != engine.KazarossXG2Digest() {
		var err error
		if id, err = c.dst.Save(ctx, c.scope, domain.MatchEquityTable{Name: t.Name, Digest: t.Digest, Source: t.Source}); err != nil {
			return 0, err
		}
	}
	c.target[srcID] = id
	return id, nil
}

// Side is one of the two analyses a merge combines, with the table its
// verdict was valued with, taken before the merge runs (a merge may mutate
// its inputs).
type Side struct {
	engine          string
	depth, creation int64
	met             int64
	present         bool
}

// SideOf describes a, valued with the table met (0: the built-in one); a nil
// a is no analysis.
func SideOf(a *domain.PositionAnalysis, met int64) Side {
	if a == nil {
		return Side{}
	}
	e, d, c := engine.AnalysisProvenance(a)
	return Side{engine: e, depth: d, creation: c, met: met, present: true}
}

// ValuedWithTable says whether a verdict from engineLabel depends on the
// library's table: only gammonNet's does. Imported verdicts and rollouts are
// read as Kazaross-XG2.
func ValuedWithTable(engineLabel string) bool {
	return strings.HasPrefix(engineLabel, "gammonNet")
}

// AfterMerge is the table the merged analysis's verdict was valued with: the
// table of the side whose verdict it kept, the existing one on a tie, and the
// built-in one when the verdict is not gammonNet's.
func AfterMerge(merged *domain.PositionAnalysis, existing, imported Side) int64 {
	m := SideOf(merged, 0)
	if !ValuedWithTable(m.engine) {
		return 0
	}
	for _, s := range []Side{existing, imported} {
		if s.present && s.engine == m.engine && s.depth == m.depth && s.creation == m.creation {
			return s.met
		}
	}
	return 0
}
