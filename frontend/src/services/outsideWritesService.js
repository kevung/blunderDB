// The writes another process (the CLI, a serve daemon on the same file) makes to an open
// library: the backend tells them from the desktop's own (PRAGMA data_version and a count of its
// own commits), and they reach the panels through libraryWritten like any job's.
import { LibrariesWrittenElsewhere } from '../../wailsjs/go/gui/App.js';
import { libraryWritten } from '../stores/groupActivityStore.js';
import { logger } from '../utils/logger.js';

/** How often the libraries are asked while the window is visible; never while it is hidden. */
export const OUTSIDE_WRITES_POLL_MS = 2000;
/**
 * The longest a group written from outside waits for its read: a long CLI import writes at
 * every poll, and reading every panel again each time would be a read without end.
 */
export const OUTSIDE_WRITES_MAX_WAIT_MS = 10000;

/** The groups written from outside and not read yet, with when the first write was seen. */
const pending = new Map();
/** @type {ReturnType<typeof setInterval> | null} */
let timer = null;
let asking = false;
let started = false;

/**
 * Asks the backend once. A group written is read again once its writes have stopped for a poll,
 * or after OUTSIDE_WRITES_MAX_WAIT_MS of writes; with now, at once (the window comes back).
 * @param {{ now?: boolean }} [opts]
 */
export async function checkOutsideWrites({ now = false } = {}) {
    if (asking) return;
    asking = true;
    /** @type {number[]} */
    let written = [];
    try {
        written = (await LibrariesWrittenElsewhere()) ?? [];
    } catch (error) {
        logger.debug('Asking for outside writes failed:', error);
    } finally {
        asking = false;
    }
    const at = Date.now();
    const writing = new Set(written);
    for (const group of writing) if (!pending.has(group)) pending.set(group, at);
    for (const [group, since] of [...pending]) {
        if (now || !writing.has(group) || at - since >= OUTSIDE_WRITES_MAX_WAIT_MS) {
            pending.delete(group);
            libraryWritten(group);
        }
    }
}

function visible() {
    return typeof document === 'undefined' || document.visibilityState !== 'hidden';
}

function follow() {
    if (visible()) {
        if (!timer) timer = setInterval(() => void checkOutsideWrites(), OUTSIDE_WRITES_POLL_MS);
        void checkOutsideWrites({ now: true });
    } else if (timer) {
        clearInterval(timer);
        timer = null;
    }
}

function onFocus() {
    void checkOutsideWrites({ now: true });
}

/** Starts asking while the window is visible, and at once when it comes back or takes focus. */
export function initOutsideWrites() {
    if (started) return;
    started = true;
    document.addEventListener('visibilitychange', follow);
    window.addEventListener('focus', onFocus);
    follow();
}

/** Stops asking; what was pending is dropped. */
export function stopOutsideWrites() {
    pending.clear();
    if (!started) return;
    started = false;
    document.removeEventListener('visibilitychange', follow);
    window.removeEventListener('focus', onFocus);
    if (timer) clearInterval(timer);
    timer = null;
}
