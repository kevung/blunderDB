// What the candidate moves table shows of the rollouts of a position: one entry per play, the
// rollout in progress first, then one of a board that is not stored, then the stored ones newest
// first. A rollout of the cube decision has no play row and is left out. Plays are matched by
// the board they leave (resolveBoard), never by their notation: two plays of the same shape may
// hit different checkers.
import { engineMarksHits, marksHits, playBoards } from './moveNotation.js';

/**
 * @typedef {{ move: string, equity: number, ci95: number, stdErr: number, jsd: number, games: number }} RolloutCandidate
 * @typedef {{ candidate: RolloutCandidate, record: any, live: boolean, boards: string[] }} RolloutEntry
 */

/**
 * @param {{ stored?: any[], unsaved?: any, live?: RolloutCandidate[], position?: any }} sources
 * @returns {Map<string, RolloutEntry>} keyed by the board each play leaves from position; a play
 *   that may leave several is keyed by its notation (notationKey), one that names no play is left
 *   out
 */
export function rolloutsByMove({ stored = [], unsaved = null, live = [], position = null } = {}) {
    /** @type {Map<string, RolloutEntry>} */
    const out = new Map();
    /** @param {RolloutCandidate[]} candidates @param {any} record @param {boolean} isLive */
    const add = (candidates, record, isLive) => {
        // A rollout in progress names its plays as the generator writes them, hits included; a
        // stored one as they were picked, possibly from an analysis that writes none.
        const marks = isLive || (engineMarksHits(record?.analysisEngine) ?? marksHits((candidates ?? []).map((c) => c.move)));
        for (const c of candidates ?? []) {
            const boards = playBoards(position, c.move, marks);
            const key = boards.length === 1 ? boards[0] : boards.length ? notationKey(c.move) : '';
            if (key && !out.has(key)) out.set(key, { candidate: c, record, live: isLive, boards });
        }
    };
    add(live, null, true);
    for (const r of [...(unsaved ? [unsaved] : []), ...(stored ?? [])]) {
        if (r?.kind === 'cube') continue;
        add(r?.candidates, r, false);
    }
    return out;
}

/** @param {string} move — the key of a play that may leave several boards: its parts, sorted. */
const notationKey = (move) =>
    'notation:' +
    String(move ?? '')
        .trim()
        .split(/\s+/)
        .sort()
        .join(' ');

/**
 * The rollout of one row of the analysis table: the entry leaving the board the row's play
 * leaves, none when that play leaves no single board. A rollout whose name may leave several
 * boards (one stored from an XG "6/1" that hits nothing, read leniently) is the row's when the
 * row is written exactly so and its board is one of them: the name came from that row.
 *
 * @param {Map<string, RolloutEntry> | null | undefined} rollouts
 * @param {string} move
 * @param {boolean} moveMarks whether the row's engine writes its hits (candidateMarksHits)
 * @param {any} position
 * @returns {RolloutEntry | undefined}
 */
export function rolloutFor(rollouts, move, moveMarks, position) {
    if (!rollouts || !rollouts.size || !move) return undefined;
    const boards = playBoards(position, move, moveMarks);
    if (boards.length !== 1) return undefined;
    const named = rollouts.get(notationKey(move));
    return rollouts.get(boards[0]) ?? (named?.boards.includes(boards[0]) ? named : undefined);
}

/**
 * The rollouts of the cube decision among them, newest first.
 * @param {any[]} [stored]
 * @param {any} [unsaved]
 */
export function cubeRollouts(stored = [], unsaved = null) {
    return [...(unsaved ? [unsaved] : []), ...(stored ?? [])].filter((r) => r?.kind === 'cube');
}
