// What the candidate moves table shows of the rollouts of a position: one entry per play, the
// rollout in progress first, then one of a board that is not stored, then the stored ones newest
// first. A rollout of the cube decision has no play row and is left out.
import { canonicalMove } from './moveNotation.js';

/**
 * @typedef {{ move: string, equity: number, ci95: number, stdErr: number, jsd: number, games: number }} RolloutCandidate
 * @typedef {{ candidate: RolloutCandidate, record: any, live: boolean }} RolloutEntry
 */

/**
 * @param {{ stored?: any[], unsaved?: any, live?: RolloutCandidate[] }} sources
 * @returns {Map<string, RolloutEntry>} keyed by canonicalMove
 */
export function rolloutsByMove({ stored = [], unsaved = null, live = [] } = {}) {
    /** @type {Map<string, RolloutEntry>} */
    const out = new Map();
    /** @param {RolloutCandidate[]} candidates @param {any} record @param {boolean} isLive */
    const add = (candidates, record, isLive) => {
        for (const c of candidates ?? []) {
            const key = canonicalMove(c.move);
            if (key && !out.has(key)) out.set(key, { candidate: c, record, live: isLive });
        }
    };
    add(live, null, true);
    for (const r of [...(unsaved ? [unsaved] : []), ...(stored ?? [])]) {
        if (r?.kind === 'cube') continue;
        add(r?.candidates, r, false);
    }
    return out;
}

/**
 * The rollouts of the cube decision among them, newest first.
 * @param {any[]} [stored]
 * @param {any} [unsaved]
 */
export function cubeRollouts(stored = [], unsaved = null) {
    return [...(unsaved ? [unsaved] : []), ...(stored ?? [])].filter((r) => r?.kind === 'cube');
}
