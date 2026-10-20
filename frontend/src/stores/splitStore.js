// The split screen (ADR-0101): the live group in one pane, a read-only viewer of a library group
// in the other, both the same width. The front keeps one set of stores, photographed per group,
// so only one group is live at a time; the viewer reads its group by number and writes nothing.
import { writable } from 'svelte/store';

/**
 * @typedef {{ on: boolean, group: number | null, side: 'left' | 'right' }} SplitState
 *   group: the group the viewer shows (null: the first other group, else the active one);
 *   side: where the viewer sits, so each library keeps its pane when the other is activated.
 */

/** @type {import('svelte/store').Writable<SplitState>} */
export const splitStore = writable({ on: false, group: null, side: 'right' });
