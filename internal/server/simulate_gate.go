package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// simulateMaxBodyBytes caps the body of directions.simulate well under the default cap: a
// format and 512 entries are a few kilobytes, and its two free-form members are decoded.
const simulateMaxBodyBytes = 1 << 20

// smallBodyPaths are the routes whose body is capped under the server's default
// (limitBody): what they decode is sized by the request.
var smallBodyPaths = map[string]int64{"/v1/directions.simulate": simulateMaxBodyBytes}

// simulateGate bounds the simulations computed at once in this process: one is pure
// computing for up to service.SimulateBudget, and a client repeating it must not take the
// processors the gestures need. The slots are the process's, every tenant together, so they
// are shared fairly: a tenant that holds one takes another only while no other tenant waits,
// and a tenant that holds none waits a short while for one rather than being turned away.
type simulateGate struct {
	mu       sync.Mutex
	slots    int
	maxWait  int
	held     map[string]int
	total    int
	waiting  int
	released chan struct{}
}

func newSimulateGate(slots, maxWait int) *simulateGate {
	return &simulateGate{slots: slots, maxWait: maxWait, held: map[string]int{}, released: make(chan struct{})}
}

var simulations = newSimulateGate(2, 8)

// acquire takes a slot for tenant, or answers storage.ErrBusy. The release it returns is
// called once.
func (g *simulateGate) acquire(ctx context.Context, tenant string) (func(), error) {
	g.mu.Lock()
	queued := false
	defer func() {
		if queued {
			g.waiting--
		}
		g.mu.Unlock()
	}()
	// A waiter outlasts every simulation that was running when it arrived.
	deadline := time.NewTimer(service.SimulateBudget + time.Second)
	defer deadline.Stop()
	for {
		if g.total < g.slots && (g.held[tenant] == 0 || g.waiting == 0) {
			g.held[tenant]++
			g.total++
			return func() { g.release(tenant) }, nil
		}
		if g.held[tenant] > 0 {
			return nil, fmt.Errorf("%w: this tenant already runs a simulation", storage.ErrBusy)
		}
		if !queued {
			if g.waiting >= g.maxWait {
				return nil, fmt.Errorf("%w: too many simulations are waiting", storage.ErrBusy)
			}
			queued = true
			g.waiting++
		}
		released := g.released
		g.mu.Unlock()
		select {
		case <-released:
			g.mu.Lock()
		case <-ctx.Done():
			g.mu.Lock()
			return nil, ctx.Err()
		case <-deadline.C:
			g.mu.Lock()
			return nil, fmt.Errorf("%w: no simulation ended in time", storage.ErrBusy)
		}
	}
}

func (g *simulateGate) release(tenant string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.held[tenant]--; g.held[tenant] == 0 {
		delete(g.held, tenant)
	}
	g.total--
	close(g.released)
	g.released = make(chan struct{})
}

// running is the number of slots taken and of tenants waiting for one.
func (g *simulateGate) running() (total, waiting int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.total, g.waiting
}
