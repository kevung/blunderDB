// Package service runs the Direction of a Tournament and the Rencontre it plays in (ADR-0047,
// ADR-0056) on the storage contract, so the desktop, the CLI and the serve daemon share one
// implementation over SQLite and PostgreSQL alike (ADR-0057 rule 2).
//
// The direction package holds the rules and the engine; this one reads and writes what they
// need through storage.Stores and shapes the views the callers show. Every write is one
// transaction of the backend, and a gesture that writes several Directions runs in one
// storage.Tx.
//
// # Concurrent gestures
//
// A gesture reads a Direction's log, decides, and appends the next event at the next sequence
// number. The backends hold no global lock, so the service serialises the gestures itself, in
// its Memory: one at a time per (scope, Direction), and a gesture that may write several
// Directions (a room, a shared configuration) or put a match on a table of a shared room alone
// in its scope. Reads take no lock, and the pages a gesture rewrites are written once its lock
// is released.
//
// That serialisation is per process. Between processes — several serve daemons over one
// PostgreSQL — the primary key of direction_event (tournament, seq) is the guard: the second
// of two simultaneous gestures fails on it, and the caller gets that conflict as an error
// (storage.ErrConflict), never a silently merged log. Repeating the gesture reads the new log.
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
	// gestureMu guards the two lock tables below; see lockDirection.
	gestureMu  sync.Mutex
	scopeLocks map[string]*sync.RWMutex
	dirLocks   map[forecastKey]*sync.Mutex
}

// gestureLocks returns the scope's room lock and the Direction's own lock, made on first use.
func (m *Memory) gestureLocks(scope string, tournamentID int64) (*sync.RWMutex, *sync.Mutex) {
	m.gestureMu.Lock()
	defer m.gestureMu.Unlock()
	if m.scopeLocks == nil {
		m.scopeLocks = map[string]*sync.RWMutex{}
		m.dirLocks = map[forecastKey]*sync.Mutex{}
	}
	room := m.scopeLocks[scope]
	if room == nil {
		room = &sync.RWMutex{}
		m.scopeLocks[scope] = room
	}
	if tournamentID == 0 {
		return room, nil
	}
	k := forecastKey{scope, tournamentID}
	own := m.dirLocks[k]
	if own == nil {
		own = &sync.Mutex{}
		m.dirLocks[k] = own
	}
	return room, own
}

// ownLock serialises the gestures of one Direction: a gesture reads the log, decides, and
// appends at the next sequence number, so two at once would claim the same number. The
// returned func releases it. Gestures of other Directions run alongside.
func (d *Service) ownLock(tournamentID int64) func() {
	room, own := d.gestureLocks(d.scope, tournamentID)
	room.RLock()
	own.Lock()
	return func() {
		own.Unlock()
		room.RUnlock()
	}
}

// roomLock serialises a gesture that may write several Directions of the scope — a room, a
// configuration a room shares — against every other gesture of the scope.
func (d *Service) roomLock() func() {
	room, _ := d.gestureLocks(d.scope, 0)
	room.Lock()
	return room.Unlock
}

// tablesLock takes the lock a gesture that puts a match on a table needs. In a Rencontre it is
// the room's: a table free of the sisters must stay free until the match is written on it, and
// a swap may write a sister's log. Membership is read under the room's lock, so it cannot change
// between that reading and the choice; outside any Rencontre the gesture falls back to its own
// Direction's lock, and checks again that no attach slipped in meanwhile. shared says which lock
// is held.
func (d *Service) tablesLock(ctx context.Context, tournamentID int64) (release func(), shared bool) {
	for {
		release = d.roomLock()
		if rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID); err != nil || rid != 0 {
			return release, true
		}
		release()
		release = d.ownLock(tournamentID)
		if rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID); err == nil && rid == 0 {
			return release, false
		}
		release()
	}
}

// lockDirection opens a gesture on one Direction: its lock, then the caller's version checked
// under it (ExpectVersion). The returned func releases the lock and rewrites the Direction's
// display page, so every caller gets its page alike and a slow folder holds no gesture.
//
// A caller that states a version holds the token of the whole room when the Direction plays in
// a Rencontre (DirectionVersion), and a sister's gesture moves it: the check then takes the
// room's lock, or two gestures on two sisters could both pass under one token.
func (d *Service) lockDirection(ctx context.Context, tournamentID int64) (func(), error) {
	if _, ok := expectedVersion(ctx); ok {
		release, _, err := d.lockTables(ctx, tournamentID)
		return release, err
	}
	unlock := d.ownLock(tournamentID)
	return func() {
		unlock()
		d.writePages(ctx, tournamentID)
	}, nil
}

// lockTables is tablesLock for a gesture: the caller's version is checked under the lock, and
// the release rewrites the Direction's display page.
func (d *Service) lockTables(ctx context.Context, tournamentID int64) (release func(), shared bool, err error) {
	unlock, shared := d.tablesLock(ctx, tournamentID)
	if err := d.checkVersion(ctx, tournamentID, 0); err != nil {
		unlock()
		return nil, false, err
	}
	return func() {
		d.recordVersion(ctx, tournamentID, 0)
		unlock()
		d.writePages(ctx, tournamentID)
	}, shared, nil
}

// lockRoom is roomLock for a gesture on a room: the caller's version of what it names — a
// Direction, or a Rencontre when rencontreID is set — is checked under the lock. Its release
// writes nothing: a room gesture rewrites the pages it changed itself.
func (d *Service) lockRoom(ctx context.Context, tournamentID, rencontreID int64) (func(), error) {
	unlock := d.roomLock()
	if err := d.checkVersion(ctx, tournamentID, rencontreID); err != nil {
		unlock()
		return nil, err
	}
	return func() {
		d.recordVersion(ctx, tournamentID, rencontreID)
		unlock()
	}, nil
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
