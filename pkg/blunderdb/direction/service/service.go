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
// number. It runs in one transaction of the backend under two locks on what it names — the
// Direction, or its Rencontre when it plays in one, since a gesture may seat a match on a table
// the sisters share: a mutex in this process, and the backend's guard across processes
// (storage.GuardedBeginner: an advisory lock in PostgreSQL, the write lock in SQLite), so
// several serve daemons, the desktop and a `call` on one file are serialised alike. A version
// its caller stated (ExpectVersion) is compared inside that transaction, after the guard: what
// it read cannot change before the gesture commits. Reads take no lock, and the pages a gesture
// rewrites are written once its locks are released.
package service

import (
	"context"
	"errors"
	"fmt"
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
	// g is set on the service a gesture runs through, bound to its transaction (lockGesture).
	g *gestureTx
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
	// pageWarning is told of a page a gesture could not write (OnPageWarning).
	pageWarning func(PageWarning)
	// gestureMu guards locks, the process-side locks of the gestures; see lockGesture.
	gestureMu sync.Mutex
	locks     map[gestureLock]*sync.Mutex
}

// gestureLock names one process-side lock: a Direction's own, or a Rencontre's.
type gestureLock struct {
	scope string
	room  bool
	id    int64
}

// guardKey is the lock's name for the backend's guard (storage.GuardedBeginner).
func (k gestureLock) guardKey() string {
	if k.room {
		return fmt.Sprintf("direction|%s|rencontre|%d", k.scope, k.id)
	}
	return fmt.Sprintf("direction|%s|tournament|%d", k.scope, k.id)
}

// gestureMutex returns the process-side lock k names, made on first use.
func (m *Memory) gestureMutex(k gestureLock) *sync.Mutex {
	m.gestureMu.Lock()
	defer m.gestureMu.Unlock()
	if m.locks == nil {
		m.locks = map[gestureLock]*sync.Mutex{}
	}
	mu := m.locks[k]
	if mu == nil {
		mu = &sync.Mutex{}
		m.locks[k] = mu
	}
	return mu
}

// gestureTarget is what a gesture names, and so what its locks and its version are.
type gestureTarget struct {
	tournamentID int64 // the Direction, or 0 for a gesture on a room
	rencontreID  int64 // the room a gesture on a room names; or the room an attach joins
	own          bool  // the Direction's own lock as well as its room's: membership changes
}

// lockDirection opens a gesture on one Direction. A Direction that plays in a Rencontre is
// locked by its room: a gesture may seat a match on a table the sisters share, or write a
// sister's log in a swap, and its version is the room's.
func (d *Service) lockDirection(ctx context.Context, tournamentID int64) (*Service, func(*error), error) {
	g, end, _, err := d.lockGesture(ctx, gestureTarget{tournamentID: tournamentID}, true)
	return g, end, err
}

// lockTables is lockDirection that also says whether the Direction shares its tables.
func (d *Service) lockTables(ctx context.Context, tournamentID int64) (*Service, func(*error), bool, error) {
	return d.lockGesture(ctx, gestureTarget{tournamentID: tournamentID}, true)
}

// lockRoom opens a gesture on a room — the Rencontre rencontreID, or the room of the Direction
// tournamentID. Its end writes no page: a room gesture rewrites the pages it changed itself.
func (d *Service) lockRoom(ctx context.Context, tournamentID, rencontreID int64) (*Service, func(*error), error) {
	g, end, _, err := d.lockGesture(ctx, gestureTarget{tournamentID: tournamentID, rencontreID: rencontreID}, false)
	return g, end, err
}

// lockMembership opens a gesture that moves the Direction tournamentID in or out of a room
// (rencontreID, or the one it plays in): both the Direction's lock and the room's, so neither
// a gesture that read the old membership nor one that reads the new one runs alongside. The
// version checked is the room's when rencontreID is set, the Direction's otherwise.
func (d *Service) lockMembership(ctx context.Context, tournamentID, rencontreID int64) (*Service, func(*error), error) {
	g, end, _, err := d.lockGesture(ctx, gestureTarget{tournamentID: tournamentID, rencontreID: rencontreID, own: true}, false)
	return g, end, err
}

// lockGesture serialises a gesture against every other one on what it names, in this process
// (a mutex) and across processes (the backend's guard, held by the gesture's transaction), then
// checks the caller's version, if it stated one (ExpectVersion), inside that transaction: what
// the comparison read cannot change before the gesture's writes commit.
//
// It returns the service bound to that transaction — the gesture reads and writes through it
// — and the end the caller defers with its error: nil commits, anything else rolls back, so a
// gesture writes all of its rows or none. A conflict on the log's sequence (a writer that took
// no guard) is a stale version for a caller that stated one. withPages rewrites the
// Direction's pages once the locks are released.
func (d *Service) lockGesture(ctx context.Context, t gestureTarget, withPages bool) (*Service, func(*error), bool, error) {
	if d.g != nil {
		// Already inside a gesture: its transaction and its locks hold.
		return d, func(*error) {}, false, nil
	}
	for {
		room, err := d.roomOf(ctx, d.st, t)
		if err != nil {
			return nil, nil, false, err
		}
		keys := d.lockKeys(t, room)
		for _, k := range keys {
			d.gestureMutex(k).Lock()
		}
		unlock := func() {
			for i := len(keys) - 1; i >= 0; i-- {
				d.gestureMutex(keys[i]).Unlock()
			}
		}
		g, tx, err := d.beginGesture(ctx, keys)
		if err != nil {
			unlock()
			return nil, nil, false, err
		}
		// Another process may have moved the Direction between the reading and the guard.
		if now, err := d.roomOf(ctx, g.st, t); err != nil || now != room {
			_ = tx.Rollback()
			unlock()
			if err != nil {
				return nil, nil, false, err
			}
			continue
		}
		if err := g.checkVersion(ctx, t.tournamentID, t.rencontreID); err != nil {
			_ = tx.Rollback()
			unlock()
			return nil, nil, false, err
		}
		end := func(errp *error) {
			if *errp == nil && g.g.failed {
				*errp = errGestureAborted
			}
			if *errp == nil {
				g.recordVersion(ctx, t.tournamentID, t.rencontreID)
				*errp = tx.Commit()
			}
			if *errp != nil {
				_ = tx.Rollback()
				if _, ok := expectedVersion(ctx); ok && errors.Is(*errp, storage.ErrConflict) {
					*errp = ErrStale
				}
			}
			unlock()
			if *errp == nil && withPages && t.tournamentID != 0 {
				d.writePages(context.WithoutCancel(ctx), t.tournamentID)
			}
		}
		return g, end, room != 0, nil
	}
}

// errGestureAborted reports a gesture whose nested transaction rolled back while the gesture
// itself returned no error: nothing of it is committed.
var errGestureAborted = errors.New("direction: the gesture was rolled back")

// roomOf is the room a gesture's locks follow: the Rencontre it names, or the one its Direction
// plays in (0 when none).
func (d *Service) roomOf(ctx context.Context, st storage.Stores, t gestureTarget) (int64, error) {
	if t.tournamentID == 0 || (t.rencontreID != 0 && t.own) {
		return t.rencontreID, nil
	}
	return st.Rencontres().Of(ctx, d.scope, t.tournamentID)
}

// lockKeys are the locks a gesture takes, the Direction's before the room's: the order every
// gesture that takes both follows.
func (d *Service) lockKeys(t gestureTarget, room int64) []gestureLock {
	var keys []gestureLock
	if t.tournamentID != 0 && (t.own || room == 0) {
		keys = append(keys, gestureLock{scope: d.scope, id: t.tournamentID})
	}
	if room != 0 {
		keys = append(keys, gestureLock{scope: d.scope, room: true, id: room})
	}
	return keys
}

// gestureTx is the transaction a gesture runs in, shared by the transactions it nests.
type gestureTx struct {
	failed bool
}

// beginGesture opens the gesture's transaction under the backend's guard and returns the
// service bound to it.
func (d *Service) beginGesture(ctx context.Context, keys []gestureLock) (*Service, storage.Tx, error) {
	var tx storage.Tx
	var err error
	if gb, ok := d.st.(storage.GuardedBeginner); ok {
		names := make([]string, len(keys))
		for i, k := range keys {
			names[i] = k.guardKey()
		}
		tx, err = gb.BeginGuardedTx(ctx, names...)
	} else {
		tx, err = d.st.BeginTx(ctx)
	}
	if err != nil {
		return nil, nil, err
	}
	g := &gestureTx{}
	return &Service{st: &txStorage{Tx: tx, g: g}, scope: d.scope, Memory: d.Memory, g: g}, tx, nil
}

// txStorage is a gesture's transaction seen as a Storage, so the service's code reads and
// writes through it unchanged. A transaction it begins is the gesture's own: committing it
// waits for the gesture's end, and rolling it back fails the whole gesture.
type txStorage struct {
	storage.Tx
	g *gestureTx
}

func (t *txStorage) BeginTx(context.Context) (storage.Tx, error) {
	return &nestedTx{Tx: t.Tx, g: t.g}, nil
}

func (t *txStorage) Close() error { return nil }

func (t *txStorage) Version(ctx context.Context) (string, error) {
	return t.Metadata().Version(ctx, "")
}

func (t *txStorage) Migrate(context.Context) error {
	return errors.New("direction: no migration inside a gesture")
}

// nestedTx is a transaction begun inside a gesture's.
type nestedTx struct {
	storage.Tx
	g    *gestureTx
	done bool
}

func (n *nestedTx) Commit() error {
	n.done = true
	return nil
}

func (n *nestedTx) Rollback() error {
	if !n.done {
		n.done = true
		n.g.failed = true
	}
	return nil
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
