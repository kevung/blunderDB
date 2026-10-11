// Package maintenance is the library's state of health (ADR-0088): one line
// per operation that recomputes, repairs or reclaims what the library derives
// from its own data, each with a counter, an action and its last run.
//
// The package sits above storage and holds no SQL of its own: a line's count,
// its list of ids and its action call the same storage pass, so the three
// agree by construction. The CLI (`blunderdb maintenance`), the desktop
// wrapper and the daemon (`/v1/maintenance.*`) all go through it.
//
// Rules the package holds:
//
//   - A count declares its cost (Tier). Status computes only the free ones;
//     the others are asked for, one at a time, and honour the context.
//   - Nothing runs by itself: Run is the only writer, and it writes the
//     journal row of its kind only once a requested action has ended
//     (ADR-0007: opening or reading a library writes nothing).
//   - Run never takes a lock. Opts.Before is called around each unit of work
//     the storage pass commits; the desktop wrapper passes its own lock there,
//     since Database.mu is not reentrant, and the daemon passes nothing.
//   - One operation at a time per tenant is the caller's rule, not the
//     package's: the daemon shares its slot with the gammonNet sweeps and
//     the rollouts, the desktop wrapper (Database.acquireJob) with its
//     gammonNet batches and its stored rollouts.
package maintenance
