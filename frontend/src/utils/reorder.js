// Reordering a list by one step (▲/▼ buttons) or by an arbitrary move (drag),
// with the new order persisted through a callback. Factored out of
// TournamentPanel (moveMatchUp/Down, handleMatchReorder) and CollectionPanel
// (moveCollectionUp/Down, movePositionUp/Down, the single-item branch of the
// drag handlers), which each re-derived the same splice-and-persist sequence.
//
// The (from, to) convention is the one of dragReorder.js: the item at `from`
// is taken out and reinserted at `to`, i.e. it ends up at index `to` of the
// result. moveUp/moveDown are the two one-step cases of the same move.

import { logger } from './logger.js';

/**
 * A copy of `list` with the item at `from` moved to `to`, or `null` when the
 * move is not possible (indices out of range, same index, no list).
 *
 * @template T
 * @param {T[] | null | undefined} list
 * @param {number} from
 * @param {number} to
 * @returns {T[] | null}
 */
export function moveItem(list, from, to) {
    if (!Array.isArray(list)) return null;
    const n = list.length;
    if (from < 0 || from >= n || to < 0 || to >= n || from === to) return null;
    const next = [...list];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    return next;
}

/** A copy of `list` with the item at `index` moved one step up, or `null` if it is first. */
export function moveUp(list, index) {
    return moveItem(list, index, index - 1);
}

/** A copy of `list` with the item at `index` moved one step down, or `null` if it is last. */
export function moveDown(list, index) {
    return moveItem(list, index, index + 1);
}

/**
 * Bind the moves above to a list held elsewhere (a store, a $state) and to a
 * persistence call. Each method applies the move locally first — the UI
 * answers at once — then persists; a persistence failure is logged and the
 * local order is kept, as the panels always did.
 *
 * @template T
 * @param {object} opts
 * @param {() => (T[] | null | undefined)} opts.get Current list; return null
 *   to disable reordering (e.g. no tournament selected).
 * @param {(next: T[], from: number, to: number) => void} opts.set Install the
 *   new order; `from`/`to` let the caller follow a selection.
 * @param {(next: T[]) => (void | Promise<void>)} opts.persist Save the new order.
 * @param {string} [opts.label] Named in the error log ("Error reordering <label>:").
 */
export function createReorder({ get, set, persist, label = 'items' }) {
    async function commit(next, from, to) {
        if (!next) return false;
        set(next, from, to);
        try {
            await persist(next);
        } catch (error) {
            logger.error(`Error reordering ${label}:`, error);
        }
        return true;
    }
    return {
        /** @returns {Promise<boolean>} whether a move happened */
        moveUp: (index) => commit(moveUp(get(), index), index, index - 1),
        moveDown: (index) => commit(moveDown(get(), index), index, index + 1),
        /** dragReorder's onReorder signature. */
        reorder: (from, to) => commit(moveItem(get(), from, to), from, to)
    };
}
