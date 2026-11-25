package database

import "errors"

// The kinds of durable work a TaskTracker is told of.
const (
	TaskImport = "import"
	TaskRepair = "repair"
)

// TaskTracker lists a piece of durable work the desktop's frontend calls
// directly on the bound handle — an import of one file or of a library, a
// repair — in the desktop's registry of tasks (ADR-0115). It is told the
// work's kind, a label and the function that stops it (nil for none), and
// returns the function called once when the work ends, with its error.
type TaskTracker func(kind, label string, cancel func()) (ended func(err error))

// SetTaskTracker registers the desktop's tracker on d (nil removes it). A
// function, not a method: its callbacks are Go values the Wails binding of
// *Database could not carry.
func SetTaskTracker(d *Database, t TaskTracker) {
	if t == nil {
		d.tracker.Store(nil)
		return
	}
	d.tracker.Store(&t)
}

// errTaskPanicked ends a tracked piece of work that panicked.
var errTaskPanicked = errors.New("the task stopped on an internal error")

// tracked runs f as a piece of work the tracker lists, ending it whatever
// happens, a panic included (which goes on).
func tracked[T any](d *Database, kind, label string, cancel func(), f func() (T, error)) (out T, err error) {
	t := d.tracker.Load()
	if t == nil {
		return f()
	}
	ended := (*t)(kind, label, cancel)
	finished := false
	defer func() {
		if !finished {
			ended(errTaskPanicked)
		}
	}()
	out, err = f()
	finished = true
	ended(err)
	return out, err
}
