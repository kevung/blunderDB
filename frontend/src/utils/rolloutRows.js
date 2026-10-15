// What the candidate moves table shows of the rollouts of a position: one entry per play, the
// rollout in progress first, then one of a board that is not stored, then the stored ones newest
// first. A rollout of the cube decision has no play row and is left out.
import { engineMarksHits, marksHits, playKey, samePlay, sameSteps } from './moveNotation.js';

/**
 * @typedef {{ move: string, equity: number, ci95: number, stdErr: number, jsd: number, games: number }} RolloutCandidate
 * @typedef {{ candidate: RolloutCandidate, record: any, live: boolean, marks: boolean }} RolloutEntry
 */

/**
 * @param {{ stored?: any[], unsaved?: any, live?: RolloutCandidate[] }} sources
 * @returns {Map<string, RolloutEntry>} keyed by playKey
 */
export function rolloutsByMove({ stored = [], unsaved = null, live = [] } = {}) {
    /** @type {Map<string, RolloutEntry>} */
    const out = new Map();
    /** @param {RolloutCandidate[]} candidates @param {any} record @param {boolean} isLive */
    const add = (candidates, record, isLive) => {
        // A rollout names its plays as the generator writes them, hits included; a record whose
        // engine is not known says so by its own names.
        const marks = isLive || (engineMarksHits(record?.analysisEngine) ?? marksHits((candidates ?? []).map((c) => c.move)));
        for (const c of candidates ?? []) {
            const key = playKey(c.move);
            if (key && !out.has(key)) out.set(key, { candidate: c, record, live: isLive, marks });
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
 * The rollout of one row of the analysis table: by its playKey, else, when one side's dialect
 * never writes its hits, by the single entry of the same shape (samePlay), or of the same steps
 * — two would be a guess.
 *
 * @param {Map<string, RolloutEntry> | null | undefined} rollouts
 * @param {string} move
 * @param {boolean} moveMarks whether the row's engine writes its hits (candidateMarksHits)
 * @returns {RolloutEntry | undefined}
 */
export function rolloutFor(rollouts, move, moveMarks) {
    if (!rollouts || !rollouts.size || !move) return undefined;
    const exact = rollouts.get(playKey(move));
    if (exact) return exact;
    /** @type {RolloutEntry[]} */
    const same = [];
    /** @type {RolloutEntry[]} */
    const steps = [];
    for (const entry of rollouts.values()) {
        if (!samePlay(move, moveMarks, entry.candidate.move, entry.marks)) continue;
        same.push(entry);
        if (sameSteps(move, entry.candidate.move)) steps.push(entry);
    }
    if (same.length === 1) return same[0];
    // The steps as written tell apart what the shape cannot.
    return steps.length === 1 ? steps[0] : undefined;
}

/**
 * The rollouts of the cube decision among them, newest first.
 * @param {any[]} [stored]
 * @param {any} [unsaved]
 */
export function cubeRollouts(stored = [], unsaved = null) {
    return [...(unsaved ? [unsaved] : []), ...(stored ?? [])].filter((r) => r?.kind === 'cube');
}
