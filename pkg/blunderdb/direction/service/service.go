// Package service runs the Direction of a Tournament and the Rencontre it plays in (ADR-0047,
// ADR-0056) on the storage contract, so the desktop, the CLI and the serve daemon share one
// implementation over SQLite and PostgreSQL alike (ADR-0057 rule 2).
//
// The direction package holds the rules and the engine; this one reads and writes what they
// need through storage.Stores and shapes the views the callers show. It takes no lock: every
// write is one transaction of the backend, and a gesture that writes several Directions runs in
// one storage.Tx.
package service

import (
	"context"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Service is the Direction service bound to one scope (a tenant, "" for a private database).
// It is cheap to build: a caller makes one per request, over the Memory it keeps.
type Service struct {
	st    storage.Storage
	scope string
	*Memory
}

// New binds the service to a backend and a scope. mem carries what outlives a call — the
// clock forecasts and the page catalogue; nil gives the call a memory of its own.
func New(st storage.Storage, scope string, mem *Memory) *Service {
	if mem == nil {
		mem = &Memory{}
	}
	return &Service{st: st, scope: scope, Memory: mem}
}

// Memory is the process-side state of the service: nothing in it is persisted, and losing it
// costs a recomputation, never a decision.
type Memory struct {
	// forecasts memoises the estimated end of each Direction's clock; forecastMu guards it.
	forecastMu sync.Mutex
	forecasts  map[forecastKey]forecastMemo
	// directionCatalog and directionLang translate the pages written for a tournament. They
	// belong to the interface, so nothing is persisted.
	directionMu      sync.RWMutex
	directionCatalog *direction.Catalog
	directionLang    string
}

type forecastKey struct {
	scope        string
	tournamentID int64
}

// dirStore is the direction.Store of the scope, outside any transaction.
func (d *Service) dirStore() direction.Store {
	return storage.BindDirection(d.st.Directions(), d.scope)
}

// inTx runs fn in one transaction of the backend, with the Direction store bound to it: an
// event and the rows that go with it are written together or not at all.
func (d *Service) inTx(ctx context.Context, fn func(storage.Tx, direction.Store) error) error {
	tx, err := d.st.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx, storage.BindDirection(tx.Directions(), d.scope)); err != nil {
		return err
	}
	return tx.Commit()
}
