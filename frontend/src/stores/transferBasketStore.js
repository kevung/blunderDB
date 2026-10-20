// The transfer basket and the group locks (ADR-0101): the desktop's memory, never a library's.
// Both outlive a switch of library group, since they name rows of several groups at once.
import { writable } from 'svelte/store';

/**
 * @typedef {'position' | 'match' | 'collection' | 'lesson'} TransferKind
 * @typedef {{ group: number, kind: TransferKind, id: number, label: string, path?: string }} TransferItem
 *   a row of the library of group, named by its id there; in the basket, path is that library's
 *   file, since a group can open another one and the id would then name another row
 */

/** What the basket holds, in the order it was filled. @type {import('svelte/store').Writable<TransferItem[]>} */
export const transferBasketStore = writable(/** @type {TransferItem[]} */ ([]));

/** The groups locked against transfers into them. */
export const lockedGroupsStore = writable(/** @type {Set<number>} */ (new Set()));

/** A group closed: its rows leave the basket, its lock goes with it. @param {number} id */
export function forgetTransferGroup(id) {
    transferBasketStore.update((items) => items.filter((i) => i.group !== id));
    lockedGroupsStore.update((s) => {
        if (!s.has(id)) return s;
        const next = new Set(s);
        next.delete(id);
        return next;
    });
}
