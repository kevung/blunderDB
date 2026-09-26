/**
 * What a stored record says was PLAYED, as predicates the move and cube tables highlight with.
 * `playedMoves`/`playedCubeActions` are every play seen in every match reaching the position (right
 * while browsing); `playedMove`/`playedCubeAction` is the one play of the match being walked, the
 * only one highlighted in MATCH mode. Shared with the Anki review (ADR-0025 rule 6).
 */
import { normalizeCubeAction } from './cubeAction.js';

/**
 * Normalize a move string for comparison by sorting individual moves:
 * "5/2 5/4" and "5/4 5/2" are the same move written in a different order.
 */
export function normalizeMoveString(move) {
    if (!move) return '';
    return move.split(' ').sort().join(' ');
}

/**
 * @param {object} analysis stored record (analysisStore's shape)
 * @param {{ matchMode?: boolean }} opts
 * @returns {(move: {move?: string}) => boolean}
 */
export function playedMovePredicate(analysis, { matchMode = false } = {}) {
    return (move) => {
        if (!move?.move) return false;
        const normalized = normalizeMoveString(move.move);

        if (matchMode) {
            if (!analysis?.playedMove) return false;
            return normalizeMoveString(analysis.playedMove) === normalized;
        }

        if (analysis?.playedMoves?.length > 0) {
            if (analysis.playedMoves.some((played) => normalizeMoveString(played) === normalized)) return true;
        }
        // Fallback to the deprecated single field for records written before it
        // was pluralised.
        if (analysis?.playedMove) return normalizeMoveString(analysis.playedMove) === normalized;
        return false;
    };
}

/**
 * @param {object} analysis stored record (analysisStore's shape)
 * @param {{ matchMode?: boolean }} opts
 * @returns {(action: string) => boolean}
 */
export function playedCubeActionPredicate(analysis, { matchMode = false } = {}) {
    return (action) => {
        const actionParts = normalizeCubeAction(action);

        if (matchMode) {
            if (!analysis?.playedCubeAction) return false;
            const playedParts = normalizeCubeAction(analysis.playedCubeAction);
            return actionParts.every((a) => playedParts.includes(a));
        }

        const allPlayedParts = new Set();
        for (const played of analysis?.playedCubeActions ?? []) {
            for (const part of normalizeCubeAction(played)) allPlayedParts.add(part);
        }
        if (allPlayedParts.size === 0 && analysis?.playedCubeAction) {
            for (const part of normalizeCubeAction(analysis.playedCubeAction)) allPlayedParts.add(part);
        }
        if (allPlayedParts.size === 0) return false;
        return actionParts.every((a) => allPlayedParts.has(a));
    };
}
