// The Eval panel's view and the trail of the roll map (the plays that lead from the position the
// map started on to the one it shows). The board follows the trail through sendPositionToEval; a
// change the trail did not make resets it.
import { writable } from 'svelte/store';

export const EVAL_VIEW = Object.freeze({ DECISION: 'decision', SCORES: 'scores', ROLLS: 'rolls' });

/** @type {import('svelte/store').Writable<string>} */
export const evalViewStore = writable(EVAL_VIEW.DECISION);

/**
 * root: the position the trail leaves from. steps: each play, with the position it was played
 * from (dice set) and the one it leads to.
 * @typedef {{ die1: number, die2: number, move: string, before: any, after: any }} TrailStep
 * @type {import('svelte/store').Writable<{ root: any, steps: TrailStep[] }>}
 */
export const rollTrailStore = writable({ root: null, steps: [] });

/** The score map's match length for this session, null while the configured default holds. */
/** @type {import('svelte/store').Writable<number | null>} */
export const mapMatchLengthStore = writable(null);

/**
 * What the board previews while a roll of the map is selected and not yet played: that roll's
 * dice on the trail's position, and the play drawn as arrows. Display only: the position the
 * board holds stays the one before the roll, so nothing describes a board that was not played.
 * @type {import('svelte/store').Writable<{ dice: [number, number], move: string } | null>}
 */
export const rollPreviewStore = writable(null);
