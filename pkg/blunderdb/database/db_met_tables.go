package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
)

// The library's match equity tables (ADR-0068), shared by GUI and CLI; the
// daemon calls package mets over its own storage.

// ImportMET imports a gnubg .xml table into the library. It does not make it
// current.
func (d *Database) ImportMET(path string) (*domain.MatchEquityTable, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return mets.ImportFile(context.Background(), d.store, "", path)
}

// ListMETs returns the built-in table then the imported ones, the current
// one marked.
func (d *Database) ListMETs() ([]*domain.MatchEquityTable, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return mets.Overview(context.Background(), d.store, "")
}

// SetCurrentMET makes the table id current, or the built-in one for 0. No
// analysis is rewritten: only what the comparisons retain changes.
func (d *Database) SetCurrentMET(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.store.MatchEquityTables().SetCurrent(context.Background(), "", id)
}

// AnalysisMETStatus says which table the position's analysis was valued with
// and whether it is "MET différente".
func (d *Database) AnalysisMETStatus(positionID int64) (mets.Status, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	pos, err := d.store.Positions().Load(context.Background(), "", positionID)
	if err != nil {
		return mets.Status{}, err
	}
	return mets.AnalysisStatus(context.Background(), d.store, "", positionID, pos.IsMoney())
}
