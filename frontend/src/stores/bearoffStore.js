import { writable } from 'svelte/store';

// A bearoff generation in flight, outside the modal so closing it loses nothing (ADR-0027: runs
// last minutes). { domain, done, total, startedAt, firstDone } or null. Remaining time is timed
// from the first callback (after the successor lists are built), not from the click, to be
// honest. Fed by ConfigModal's listeners on bearoff:progress/done/error.
export const bearoffProgressStore = writable(null);

// The last error a generation reported, cleared when a new one starts.
export const bearoffErrorStore = writable('');

// Remaining seconds measured so far, or null while too early to say.
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
