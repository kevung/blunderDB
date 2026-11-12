package server

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
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

// simulationUnavailable is a 503 of directions.simulate that is not a lost write: reason
// tells the host what to do — "simulation_busy", retry as is after retryAfter seconds;
// "too_heavy", ask for less.
type simulationUnavailable struct {
	reason, message, retryAfter string
}

func (e *simulationUnavailable) Error() string { return e.message }

func (e *simulationUnavailable) ErrorDetails() map[string]any {
	return map[string]any{"reason": e.reason}
}

func simulationBusy(why string) error {
	return &simulationUnavailable{reason: "simulation_busy", retryAfter: busyRetryAfter,
		message: "simulation busy: " + why + "; retry shortly"}
}

// clientGone marks a request whose caller left before the answer: no failure of the
// daemon, and nobody to read an envelope.
type clientGone struct{ cause error }

func (e clientGone) Error() string { return "the client closed the request: " + e.cause.Error() }
func (e clientGone) Unwrap() error { return e.cause }

// statusClientClosedRequest is the status a log shows for a request its caller abandoned.
const statusClientClosedRequest = 499

// simulate answers directions.simulate through the gate.
func simulate(ctx context.Context, scope string, req service.SimulateRequest) (*service.Simulation, error) {
	release, err := simulations.acquire(ctx, scope)
	if err != nil {
		return nil, err
	}
	defer release()
	sim, err := service.Simulate(ctx, req)
	switch {
	case errors.Is(err, service.ErrSimulationTooHeavy):
		return nil, &simulationUnavailable{reason: "too_heavy", message: err.Error()}
	case errors.Is(err, context.Canceled):
		return nil, clientGone{err}
	}
	return sim, gestureError(err)
}

// acquire takes a slot for tenant, or answers a simulationUnavailable: busy. The wait is
// not part of service.SimulateBudget. The release it returns is called once.
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
			return nil, simulationBusy("this tenant already runs a simulation")
		}
		if !queued {
			if g.waiting >= g.maxWait {
				return nil, simulationBusy("too many simulations are waiting")
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
			return nil, clientGone{ctx.Err()}
		case <-deadline.C:
			g.mu.Lock()
			return nil, simulationBusy("no simulation ended in time")
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
