// The split screen (ADR-0101 §8): the board area cut in two, each pane holding one library group
// on a fixed side. The active group's pane is live (the one Board); the other is frozen, read by
// group number, and writes nothing. The front keeps one set of stores, photographed per group, so
// the toolbar, tabs, panels and status bar always speak for the active group.
import { writable } from 'svelte/store';

/**
 * @typedef {{ on: boolean, left: number | null, right: number | null, sameSide: 'left' | 'right' }} SplitState
 *   left / right: the group each side holds; a side never changes group on activation.
 *   sameSide: with one library on both sides, the side that is live.
 */

/** @type {import('svelte/store').Writable<SplitState>} */
export const splitStore = writable({ on: false, left: null, right: null, sameSide: 'left' });

/** A pane that refused to come live shakes: its side, and a counter that moves each time. */
export const splitShakeStore = writable({ side: /** @type {'left' | 'right' | null} */ (null), n: 0 });

/**
 * The frozen pane's position, which the swap of two views of one library reads; `refreeze` asks
 * the frozen pane to show another position of that library.
 */
export const splitFrozenStore = writable({
    positionId: /** @type {number | null} */ (null),
    refreeze: /** @type {{ positionId: number | null, n: number } | null} */ (null)
});
