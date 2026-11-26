// The gestures that lead into the roll and score maps: from a candidate play, from a command,
// down and up the trail. The board follows through sendPositionToEval.
import { get } from 'svelte/store';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { positionStore } from '../stores/positionStore.js';
import { evalViewStore, rollTrailStore, EVAL_VIEW } from '../stores/mapStore.js';
import { sendPositionToEval } from './modeMachine.js';
import { positionAfterPlay } from '../utils/moveNotation.js';

/** The signature the Eval panel reads a position by, as sendPositionToEval stores it.
 * @param {any} position */
export function seedSignature(position) {
    const seed = JSON.parse(JSON.stringify(position));
    seed.id = 0;
    return JSON.stringify(seed);
}

/** The position the trail shows: the last play's, the root when none.
 * @param {{ root: any, steps: any[] }} trail */
export function trailPosition(trail) {
    return trail.steps.length ? trail.steps[trail.steps.length - 1].after : trail.root;
}

/** The roll map of what follows a play: the plays the other side may answer with.
 * @param {any} position with its dice
 * @param {string} move
 * @returns {boolean} false when the play leaves no single board
 */
export function openRollReplies(position, move) {
    const [die1, die2] = position?.dice ?? [0, 0];
    const after = positionAfterPlay(position, die1, die2, move);
    if (!after) return false;
    const root = JSON.parse(JSON.stringify(position));
    rollTrailStore.set({ root, steps: [{ die1, die2, move, before: root, after }] });
    evalViewStore.set(EVAL_VIEW.ROLLS);
    sendPositionToEval(after);
    return true;
}

/** Descend: play move on the roll die1-die2 of the trail's position.
 * @param {number} die1 @param {number} die2 @param {string} move */
export function descend(die1, die2, move) {
    const trail = get(rollTrailStore);
    const before = trail.steps.length ? trailPosition(trail) : get(positionStore);
    const after = positionAfterPlay({ ...before, dice: [die1, die2] }, die1, die2, move);
    if (!after) return false;
    const root = trail.root ?? JSON.parse(JSON.stringify(before));
    rollTrailStore.set({ root, steps: [...trail.steps, { die1, die2, move, before: { ...before, dice: [die1, die2] }, after }] });
    sendPositionToEval(after);
    return true;
}

/** Climb back to the n-th link of the trail (0 is the start).
 * @param {number} n */
export function climbTo(n) {
    const trail = get(rollTrailStore);
    if (n >= trail.steps.length) return;
    const steps = trail.steps.slice(0, n);
    rollTrailStore.set({ root: trail.root, steps });
    sendPositionToEval(steps.length ? steps[steps.length - 1].after : trail.root);
}

/** Open a view of the Eval panel on the position on the board.
 * @param {string} view */
export function openEvalView(view) {
    rollTrailStore.set({ root: null, steps: [] });
    evalViewStore.set(view);
    if (get(statusBarModeStore) === 'EVAL') return;
    const shown = get(positionStore);
    if (shown?.board) sendPositionToEval(shown);
    else activeTabStore.set('eval');
}
