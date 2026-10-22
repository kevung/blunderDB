// The lot, the protections and the copies to take back (ADR-0101, ADR-0102): the desktop's
// memory, never a library's. They outlive a switch of library group, since they name rows of
// several groups at once.
import { writable } from 'svelte/store';

/**
 * @typedef {'position' | 'match' | 'tournament' | 'collection' | 'lesson'} TransferKind
 * @typedef {{ group: number, kind: TransferKind, id: number, label: string, path?: string }} TransferItem
 *   a row of the library of group, named by its id there; in the lot, path is that library's
 *   file, since a group can open another one and the id would then name another row
 */

/** What the lot holds, in the order it was filled. @type {import('svelte/store').Writable<TransferItem[]>} */
export const transferBasketStore = writable(/** @type {TransferItem[]} */ ([]));

/** The groups protected from copies into them. */
export const lockedGroupsStore = writable(/** @type {Set<number>} */ (new Set()));

/**
 * @typedef {{ dst: number, receipt: number, name: string }} TransferUndo
 *   a position copied into the library of group dst, while it can be taken back: the receipt the
 *   backend keeps for it, and that library's name for the status bar
 */

/**
 * The session's copies that can be taken back, oldest first: the last one is the next to undo.
 * @type {import('svelte/store').Writable<TransferUndo[]>}
 */
export const transferHistoryStore = writable(/** @type {TransferUndo[]} */ ([]));

/**
 * A group opened or created another library in place: a copy into it is no longer one to take
 * back from there. @param {number} group
 */
export function forgetPaneCopyInto(group) {
    transferHistoryStore.update((h) => (h.some((u) => u.dst === group) ? h.filter((u) => u.dst !== group) : h));
}

/** A group closed: its rows leave the lot, its protection and the copies into it go with it. @param {number} id */
export function forgetTransferGroup(id) {
    forgetPaneCopyInto(id);
    transferBasketStore.update((items) => items.filter((i) => i.group !== id));
    lockedGroupsStore.update((s) => {
        if (!s.has(id)) return s;
        const next = new Set(s);
        next.delete(id);
        return next;
    });
}

const PROTECTED_KEY = 'blunderdb.protectedLibraries';

/** The files protected from copies, remembered from one session to the next. @returns {string[]} */
export function protectedPaths() {
    try {
        const saved = JSON.parse(localStorage.getItem(PROTECTED_KEY) ?? '[]');
        return Array.isArray(saved) ? saved.filter((p) => typeof p === 'string') : [];
    } catch {
        return [];
    }
}

/** Remembers whether the file at path is protected from copies. @param {string} path @param {boolean} on */
export function rememberProtection(path, on) {
    if (!path) return;
    const paths = protectedPaths().filter((p) => p !== path);
    if (on) paths.push(path);
    try {
        localStorage.setItem(PROTECTED_KEY, JSON.stringify(paths));
    } catch {
        // Not remembered: the protection still holds for this session.
    }
}
