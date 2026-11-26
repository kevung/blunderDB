package jobs

import (
	"context"
	"errors"
	"slices"
	"sync"
)

// Priority orders the engine's work: a lower value is more urgent.
type Priority int

const (
	// Visible is a computation whose result is on screen.
	Visible Priority = iota
	// Requested is a durable task the user launched.
	Requested
	// Background is a durable task nobody waits on at that moment.
	Background
)

var priorityNames = [...]string{Visible: "visible", Requested: "requested", Background: "background"}

func (p Priority) String() string {
	if p < Visible || p > Background {
		return "unknown"
	}
	return priorityNames[p]
}

// State is where a task stands.
type State string

const (
	Running   State = "running"
	Waiting   State = "waiting"
	Paused    State = "paused"
	Done      State = "done"
	Cancelled State = "cancelled"
	Failed    State = "error"
)

// Finished reports a state no task leaves.
func (s State) Finished() bool { return s == Done || s == Cancelled || s == Failed }

// Kind names what a task does; the frontend translates it.
type Kind string

const (
	GammonNetMissing Kind = "gammonnet-missing"
	GammonNetStale   Kind = "gammonnet-stale"
	GammonNetMatch   Kind = "gammonnet-match"
	Rollout          Kind = "rollout"
	RolloutBatch     Kind = "rollout-batch"
	Maintenance      Kind = "maintenance"
	MatchStatsFill   Kind = "match-stats-fill"
	Import           Kind = "import"
	Repair           Kind = "repair"
	Transfer         Kind = "transfer"
	BearoffTables    Kind = "bearoff"
	Sync             Kind = "sync"
)

// Spec describes a task at registration.
type Spec struct {
	Kind  Kind
	Label string
	// Library is the library group the task works on; 0 for none.
	Library  int
	Priority Priority
	// Exclusive tasks write their library one at a time, in the order of
	// its queue: such a task calls Turn before its work.
	Exclusive bool
	// Pausable tasks call Checkpoint between units of work.
	Pausable bool
	// Compute marks a task that runs the engine: a Background one yields
	// to it at its checkpoints. A Visible computation always outranks.
	Compute bool
	// Preemptible marks an exclusive task whose work resumes where it
	// stopped (a batch finds what is still missing): its checkpoint returns
	// ErrPreempted while a task of a higher priority waits for its library,
	// and its owner gives the library back with Requeue.
	Preemptible bool
	// Cancel stops the work; nil for a task that cannot be stopped.
	Cancel func()
}

// Info is a task as listed and as sent on jobs:changed.
type Info struct {
	ID       int64  `json:"id"`
	Kind     Kind   `json:"kind"`
	Label    string `json:"label"`
	Library  int    `json:"library"`
	Priority string `json:"priority"`
	State    State  `json:"state"`
	Done     int    `json:"done"`
	Total    int    `json:"total"`
	Error    string `json:"error,omitempty"`
	// Position is the rank (1 first) among its library's waiting tasks, 0
	// for a task not in the queue.
	Position int `json:"position"`
	// Yielded marks a running task held at its checkpoint by a more urgent
	// computation: it makes no progress until that one ends.
	Yielded    bool `json:"yielded"`
	Pausable   bool `json:"pausable"`
	Cancelable bool `json:"cancelable"`
}

// The events the registry sends to its listener.
const (
	// EventChanged carries the Info of a task registered or changed.
	EventChanged = "jobs:changed"
	// EventRemoved carries {"id": id} for a task no longer listed.
	EventRemoved = "jobs:removed"
)

var (
	ErrUnknown     = errors.New("jobs: no such task")
	ErrFinished    = errors.New("jobs: the task has ended")
	ErrNotPausable = errors.New("jobs: the task cannot be paused")
	ErrNotQueued   = errors.New("jobs: the task is not waiting in a queue")
	ErrRunning     = errors.New("jobs: a running task is cancelled, not removed")
	ErrNoCancel    = errors.New("jobs: the task cannot be cancelled")
	// ErrPreempted tells a preemptible task to give its library back.
	ErrPreempted = errors.New("jobs: a task of a higher priority waits for the library")
)

// keepFinished bounds the finished tasks still listed.
const keepFinished = 32

// Registry holds every durable task of the process.
type Registry struct {
	mu   sync.Mutex
	seq  int64
	jobs []*Job // registration order
	// queue is the waiting exclusive tasks, every library mixed: a
	// library's queue is its tasks in this order.
	queue   []*Job
	visible int
	// wake is closed and replaced at every change: waiters select on it.
	wake chan struct{}

	// emitMu keeps the events in the order of the changes they report.
	emitMu sync.Mutex
	notify func(name string, payload any)
}

// New returns an empty registry; notify (nil for none) receives every event.
func New(notify func(name string, payload any)) *Registry {
	return &Registry{wake: make(chan struct{}), notify: notify}
}

// Job is the handle of one registered task.
type Job struct {
	r        *Registry
	id       int64
	spec     Spec
	state    State
	done     int
	total    int
	err      string
	paused   bool
	holding  bool // an exclusive task that has its turn
	stopping bool // Cancel was asked
	yielding bool // held at its checkpoint by a more urgent computation
	drop     bool // Remove was asked: unlisted once finished
	finished chan struct{}
}

type event struct {
	name    string
	payload any
}

// changedLocked wakes the waiters and returns the events for jobs.
func (r *Registry) changedLocked(jobs ...*Job) []event {
	close(r.wake)
	r.wake = make(chan struct{})
	evs := make([]event, 0, len(jobs))
	for _, j := range jobs {
		evs = append(evs, event{EventChanged, r.infoLocked(j)})
	}
	return evs
}

// unlockEmit releases mu and sends evs, in order with every other change.
func (r *Registry) unlockEmit(evs []event) {
	r.emitMu.Lock()
	r.mu.Unlock()
	defer r.emitMu.Unlock()
	if r.notify == nil {
		return
	}
	for _, e := range evs {
		r.notify(e.name, e.payload)
	}
}

// Start registers a task: an exclusive one enters its library's queue,
// before the tasks of a lower priority; any other runs at once. On a nil
// Registry the task runs alone, listed nowhere.
func (r *Registry) Start(s Spec) *Job {
	if r == nil {
		r = New(nil)
	}
	r.mu.Lock()
	r.seq++
	j := &Job{r: r, id: r.seq, spec: s, state: Running, finished: make(chan struct{})}
	r.jobs = append(r.jobs, j)
	if s.Exclusive {
		j.state = Waiting
		at := len(r.queue)
		for i, q := range r.queue {
			if q.spec.Priority > s.Priority {
				at = i
				break
			}
		}
		r.queue = slices.Insert(r.queue, at, j)
	}
	evs := r.changedLocked(r.queuedLocked(s.Library, j)...)
	r.unlockEmit(evs)
	return j
}

// queuedLocked is j followed by the waiting tasks of library, whose rank
// may have moved.
func (r *Registry) queuedLocked(library int, j *Job) []*Job {
	out := []*Job{j}
	for _, q := range r.queue {
		if q != j && q.spec.Library == library {
			out = append(out, q)
		}
	}
	return out
}

// Behind returns the kind of the task holding j's library while j waits for
// it, false when j would not wait on another task.
func (j *Job) Behind() (Kind, bool) {
	r := j.r
	r.mu.Lock()
	defer r.mu.Unlock()
	if !j.spec.Exclusive || j.holding {
		return "", false
	}
	for _, o := range r.jobs {
		if o.holding && o.spec.Library == j.spec.Library {
			return o.spec.Kind, true
		}
	}
	return "", false
}

// ID is the task's identifier in the registry.
func (j *Job) ID() int64 { return j.id }

// Ended is closed once the task has finished.
func (j *Job) Ended() <-chan struct{} { return j.finished }

// eligibleLocked reports whether j may take its library's turn: no task of
// the library holds it, and j is the first one not paused in its queue.
func (r *Registry) eligibleLocked(j *Job) bool {
	for _, o := range r.jobs {
		if o.holding && o.spec.Library == j.spec.Library {
			return false
		}
	}
	for _, q := range r.queue {
		if q.spec.Library != j.spec.Library || q.paused {
			continue
		}
		return q == j
	}
	return false
}

// Turn waits until the task holds its library: at once for a task that is
// not exclusive. It returns ctx's error, or context.Canceled once the task
// was cancelled, without the turn.
func (j *Job) Turn(ctx context.Context) error {
	r := j.r
	for {
		r.mu.Lock()
		if j.stopping {
			r.mu.Unlock()
			return context.Canceled
		}
		if !j.spec.Exclusive || j.holding {
			r.mu.Unlock()
			return nil
		}
		if r.eligibleLocked(j) {
			j.holding = true
			j.state = Running
			r.queue = slices.DeleteFunc(r.queue, func(q *Job) bool { return q == j })
			r.unlockEmit(r.changedLocked(r.queuedLocked(j.spec.Library, j)...))
			return nil
		}
		w := r.wake
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w:
		}
	}
}

// outrankedLocked reports a computation more urgent than p running.
func (r *Registry) outrankedLocked(j *Job) bool {
	if !j.spec.Compute {
		return false
	}
	if r.visible > 0 && j.spec.Priority > Visible {
		return true
	}
	if j.spec.Priority < Background {
		return false
	}
	for _, o := range r.jobs {
		if o != j && o.spec.Compute && o.spec.Priority < j.spec.Priority && o.state == Running && !o.paused {
			return true
		}
	}
	return false
}

// Checkpoint returns once the task may go on: not paused, and outranked by
// no computation. It returns ctx's error, or context.Canceled once the task
// was cancelled.
func (j *Job) Checkpoint(ctx context.Context) error {
	r := j.r
	for {
		r.mu.Lock()
		if j.stopping {
			return r.leaveCheckpointLocked(j, context.Canceled)
		}
		if r.preemptedLocked(j) {
			return r.leaveCheckpointLocked(j, ErrPreempted)
		}
		outranked := r.outrankedLocked(j)
		if !j.paused && !outranked {
			return r.leaveCheckpointLocked(j, nil)
		}
		// The listing shows why a running task makes no progress.
		if yielding := !j.paused && outranked; yielding != j.yielding {
			j.yielding = yielding
			r.unlockEmit(r.changedLocked(j))
			continue
		}
		w := r.wake
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			r.mu.Lock()
			return r.leaveCheckpointLocked(j, ctx.Err())
		case <-w:
		}
	}
}

// leaveCheckpointLocked releases mu and returns err, reporting first that
// the task no longer yields.
func (r *Registry) leaveCheckpointLocked(j *Job, err error) error {
	if !j.yielding {
		r.mu.Unlock()
		return err
	}
	j.yielding = false
	r.unlockEmit(r.changedLocked(j))
	return err
}

// preemptedLocked reports a preemptible task holding its library while a
// task of a higher priority, not paused, waits for it.
func (r *Registry) preemptedLocked(j *Job) bool {
	if !j.spec.Preemptible || !j.holding {
		return false
	}
	for _, q := range r.queue {
		if q.spec.Library == j.spec.Library && !q.paused && !q.stopping && q.spec.Priority < j.spec.Priority {
			return true
		}
	}
	return false
}

// Requeue gives the library back and puts the task in its queue again,
// first among the tasks of its priority: its owner calls Turn to resume.
func (j *Job) Requeue() {
	r := j.r
	r.mu.Lock()
	if !j.holding || j.state.Finished() {
		r.mu.Unlock()
		return
	}
	j.holding = false
	j.state = Waiting
	at := len(r.queue)
	for i, q := range r.queue {
		if q.spec.Priority >= j.spec.Priority {
			at = i
			break
		}
	}
	r.queue = slices.Insert(r.queue, at, j)
	r.unlockEmit(r.changedLocked(r.queuedLocked(j.spec.Library, j)...))
}

// Yield is Checkpoint for a loop that has no error to return: it blocks
// until the task may go on or ctx is done.
func (j *Job) Yield(ctx context.Context) { _ = j.Checkpoint(ctx) }

// Progress records done of total.
func (j *Job) Progress(done, total int) {
	r := j.r
	r.mu.Lock()
	if j.state.Finished() || (j.done == done && j.total == total) {
		r.mu.Unlock()
		return
	}
	j.done, j.total = done, total
	r.unlockEmit([]event{{EventChanged, r.infoLocked(j)}})
}

// Relabel changes what the task is called, once known.
func (j *Job) Relabel(label string) {
	r := j.r
	r.mu.Lock()
	j.spec.Label = label
	r.unlockEmit([]event{{EventChanged, r.infoLocked(j)}})
}

// Finish ends the task: done when err is nil, cancelled when it was
// cancelled or err is a cancellation, in error otherwise. A second call
// does nothing.
func (j *Job) Finish(err error) {
	r := j.r
	r.mu.Lock()
	if j.state.Finished() {
		r.mu.Unlock()
		return
	}
	switch {
	case err == nil && !j.stopping:
		j.state = Done
	case j.stopping || errors.Is(err, context.Canceled):
		j.state = Cancelled
	default:
		j.state = Failed
		j.err = err.Error()
	}
	j.holding, j.paused = false, false
	r.queue = slices.DeleteFunc(r.queue, func(q *Job) bool { return q == j })
	close(j.finished)
	evs := r.changedLocked(r.queuedLocked(j.spec.Library, j)...)
	if j.drop {
		evs = append(evs, r.unlistLocked(j))
	}
	evs = append(evs, r.trimLocked()...)
	r.unlockEmit(evs)
}

func (r *Registry) unlistLocked(j *Job) event {
	r.jobs = slices.DeleteFunc(r.jobs, func(o *Job) bool { return o == j })
	return event{EventRemoved, map[string]int64{"id": j.id}}
}

// trimLocked unlists the oldest finished tasks past keepFinished.
func (r *Registry) trimLocked() []event {
	n := 0
	for _, j := range r.jobs {
		if j.state.Finished() {
			n++
		}
	}
	var evs []event
	for _, j := range slices.Clone(r.jobs) {
		if n <= keepFinished {
			break
		}
		if j.state.Finished() {
			evs = append(evs, r.unlistLocked(j))
			n--
		}
	}
	return evs
}

func (r *Registry) infoLocked(j *Job) Info {
	in := Info{
		ID: j.id, Kind: j.spec.Kind, Label: j.spec.Label, Library: j.spec.Library,
		Priority: j.spec.Priority.String(), State: j.state, Done: j.done, Total: j.total, Error: j.err,
		Pausable: j.spec.Pausable, Cancelable: j.spec.Cancel != nil,
		Yielded: j.yielding && j.state == Running && !j.paused,
	}
	if j.paused && !j.state.Finished() {
		in.State = Paused
	}
	rank := 0
	for _, q := range r.queue {
		if q.spec.Library == j.spec.Library {
			rank++
			if q == j {
				in.Position = rank
				break
			}
		}
	}
	return in
}

// List returns every listed task in registration order.
func (r *Registry) List() []Info {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Info, 0, len(r.jobs))
	for _, j := range r.jobs {
		out = append(out, r.infoLocked(j))
	}
	return out
}

func (r *Registry) findLocked(id int64) (*Job, error) {
	if r == nil {
		return nil, ErrUnknown
	}
	for _, j := range r.jobs {
		if j.id == id {
			return j, nil
		}
	}
	return nil, ErrUnknown
}

// Cancel stops a task: its Turn and Checkpoint return at once, and its
// Spec.Cancel is called. It ends when its owner calls Finish.
func (r *Registry) Cancel(id int64) error {
	if r == nil {
		return ErrUnknown
	}
	r.mu.Lock()
	j, err := r.findLocked(id)
	if err == nil && j.state.Finished() {
		err = ErrFinished
	}
	if err == nil && j.spec.Cancel == nil && !(j.spec.Exclusive && !j.holding) {
		err = ErrNoCancel
	}
	if err != nil {
		r.mu.Unlock()
		return err
	}
	stop := r.stopLocked(j)
	evs := r.changedLocked(j)
	r.unlockEmit(evs)
	if stop != nil {
		stop()
	}
	return nil
}

// stopLocked marks j cancelled and returns its Spec.Cancel, called by the
// caller once mu is released (it may call back into the registry).
func (r *Registry) stopLocked(j *Job) func() {
	if j.stopping {
		return nil
	}
	j.stopping = true
	j.paused = false
	return j.spec.Cancel
}

// Pause stops a task at its next checkpoint, or keeps a waiting task from
// taking its turn.
func (r *Registry) Pause(id int64) error {
	return r.setPaused(id, true)
}

// Resume lets a paused task go on.
func (r *Registry) Resume(id int64) error {
	return r.setPaused(id, false)
}

func (r *Registry) setPaused(id int64, paused bool) error {
	if r == nil {
		return ErrUnknown
	}
	r.mu.Lock()
	j, err := r.findLocked(id)
	switch {
	case err != nil:
	case j.state.Finished():
		err = ErrFinished
	case !j.spec.Pausable && !(j.spec.Exclusive && !j.holding):
		err = ErrNotPausable
	}
	if err != nil {
		r.mu.Unlock()
		return err
	}
	if j.paused == paused || j.stopping {
		r.mu.Unlock()
		return nil
	}
	j.paused = paused
	r.unlockEmit(r.changedLocked(j))
	return nil
}

// Move puts a waiting task at index (0 first) of its library's queue.
func (r *Registry) Move(id int64, index int) error {
	if r == nil {
		return ErrUnknown
	}
	r.mu.Lock()
	j, err := r.findLocked(id)
	if err == nil && !slices.Contains(r.queue, j) {
		err = ErrNotQueued
	}
	if err != nil {
		r.mu.Unlock()
		return err
	}
	var slots []int
	var lib []*Job
	for i, q := range r.queue {
		if q.spec.Library == j.spec.Library {
			slots = append(slots, i)
			if q != j {
				lib = append(lib, q)
			}
		}
	}
	index = max(0, min(index, len(lib)))
	lib = slices.Insert(lib, index, j)
	for k, at := range slots {
		r.queue[at] = lib[k]
	}
	r.unlockEmit(r.changedLocked(lib...))
	return nil
}

// Remove unlists a finished task, or cancels a waiting one and unlists it
// once it has ended. A task running is refused: it is cancelled.
func (r *Registry) Remove(id int64) error {
	if r == nil {
		return ErrUnknown
	}
	r.mu.Lock()
	j, err := r.findLocked(id)
	if err != nil {
		r.mu.Unlock()
		return err
	}
	if j.state.Finished() {
		r.unlockEmit([]event{r.unlistLocked(j)})
		return nil
	}
	if !slices.Contains(r.queue, j) {
		r.mu.Unlock()
		return ErrRunning
	}
	j.drop = true
	stop := r.stopLocked(j)
	r.unlockEmit(r.changedLocked(j))
	if stop != nil {
		stop()
	}
	return nil
}

// Stop cancels every unfinished task of library (0: every task bound to a
// library) and waits for them until ctx is done. It returns the tasks still
// running then. A task bound to no library, or one running that cannot be
// cancelled, is left alone: waiting for it could only spend the grace.
func (r *Registry) Stop(ctx context.Context, library int) []Info {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	var stops []func()
	var waits []*Job
	var changed []*Job
	for _, j := range r.jobs {
		if j.state.Finished() || j.spec.Library == 0 || (library != 0 && j.spec.Library != library) {
			continue
		}
		if j.spec.Cancel == nil && !slices.Contains(r.queue, j) {
			continue
		}
		if s := r.stopLocked(j); s != nil {
			stops = append(stops, s)
		}
		waits = append(waits, j)
		changed = append(changed, j)
	}
	r.unlockEmit(r.changedLocked(changed...))
	for _, s := range stops {
		s()
	}
	var late []Info
	for _, j := range waits {
		select {
		case <-j.finished:
		case <-ctx.Done():
		}
		r.mu.Lock()
		if !j.state.Finished() {
			late = append(late, r.infoLocked(j))
		}
		r.mu.Unlock()
	}
	return late
}

// EnterVisible marks a computation whose result is on screen as running
// until the returned function is called (once; later calls do nothing).
// Lower priorities yield to it at their checkpoints.
func (r *Registry) EnterVisible() (exit func()) {
	if r == nil {
		return func() {}
	}
	r.mu.Lock()
	r.visible++
	r.unlockEmit(r.changedLocked())
	var once sync.Once
	return func() {
		once.Do(func() {
			r.mu.Lock()
			r.visible--
			r.unlockEmit(r.changedLocked())
		})
	}
}
