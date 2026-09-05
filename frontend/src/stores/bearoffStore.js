import { writable } from 'svelte/store';

// The state of a bearoff generation, kept outside the configuration modal so
// closing the modal does not lose it (ADR-0027, issue #308). A run for a wide
// domain is minutes long: the user is meant to close the dialog and carry on.
//
// { domain, done, total, startedAt, firstDone } while a run is in flight, null
// when idle. `startedAt`/`firstDone` are what turn the progress into a
// MEASURED remaining time rather than a second estimate: the first callback
// arrives after the successor lists are built, so timing from it — not from
// the click — is what makes the figure honest.
//
// Fed by ConfigModal.svelte's EventsOn listeners on
// bearoff:progress/done/error; internal/gui/bearoff.go knows nothing of a
// store, it only emits Wails events.
export const bearoffProgressStore = writable(null);

// The last error a generation reported, cleared when a new one starts.
export const bearoffErrorStore = writable('');

// remainingSeconds is what the progress has measured so far, or null while
// there is not enough of it to say anything. Exported rather than inlined so
// the arithmetic is testable without a component.
/**
 * @param {any} progress
 * @param {number} [now]
 * @returns {number|null}
 */
export function remainingSeconds(progress, now = Date.now()) {
    if (!progress || !progress.total || !progress.done) return null;
    const { done, total, startedAt, firstDone } = progress;
    // Measure from the first progress report, and against the work done
    // since: the run's fixed set-up is behind us by then, and counting it
    // would inflate every estimate that follows.
    const base = firstDone ?? 0;
    const advanced = done - base;
    // startedAt == null, not !startedAt: a timestamp of zero is a legitimate
    // value and a falsy check would throw the whole measurement away.
    if (advanced <= 0 || startedAt == null) return null;
    const elapsed = (now - startedAt) / 1000;
    if (elapsed <= 0) return null;
    return ((total - done) * elapsed) / advanced;
}
