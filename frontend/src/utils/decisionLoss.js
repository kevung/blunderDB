// The match winning chances each decision cost (MatchDecisionLosses). A loss is
// a fraction, as Match.mwc_loss is; a decision the analysis does not price has
// a null loss and reads as unscored, never as zero.

// The total difficulty (MWC) under which a match's loss-to-difficulty ratio
// is not given (ADR-0076, storage.DifficultyRatioFloor).
export const DIFFICULTY_RATIO_FLOOR = 0.005;

/**
 * @typedef {{ decisions: number, loss: number, difficulty: number, excess: number, ratio: number | null, avoidable: number }} DifficultySummary
 * One player's reading of ADR-0076 over the decisions carrying both a loss and
 * a difficulty, as storage.SummariseDifficulty adds them up.
 */

/**
 * The decisions of the match in match order, each with its loss, difficulty
 * and avoidable mark, the running total per player and the difficulty
 * summary. Indexed like `movePositions`, so the chart shares the time chart's
 * decision axis. A Move absent from `losses` is unscored.
 *
 * @param {readonly { move_id: number, player_on_roll: number }[]} movePositions
 * @param {readonly { move_id: number, mwc_loss: number | null, difficulty?: number | null, avoidable?: boolean }[] | null | undefined} losses
 * @returns {{ items: { index: number, player: 0 | 1, loss: number | null, difficulty: number | null, avoidable: boolean, cum: number }[], totals: [number, number], scored: [number, number], any: boolean, difficulty: [DifficultySummary, DifficultySummary], anyDifficulty: boolean }}
 */
export function lossSeries(movePositions, losses) {
    const byMove = new Map((losses ?? []).map((d) => [d.move_id, d]));
    /** @type {[number, number]} */
    const totals = [0, 0];
    /** @type {[number, number]} */
    const scored = [0, 0];
    /** @type {[DifficultySummary, DifficultySummary]} */
    const difficulty = /** @type {[DifficultySummary, DifficultySummary]} */ ([0, 1].map(() => ({ decisions: 0, loss: 0, difficulty: 0, excess: 0, ratio: null, avoidable: 0 })));
    const items = movePositions.map((mp, index) => {
        const player = mp.player_on_roll === 1 ? 1 : 0;
        const d = byMove.get(mp.move_id);
        const loss = d?.mwc_loss ?? null;
        const diff = loss === null ? null : (d?.difficulty ?? null);
        const avoidable = !!d?.avoidable;
        if (loss !== null) {
            totals[player] += loss;
            scored[player]++;
            if (diff !== null) {
                const s = difficulty[player];
                s.decisions++;
                s.loss += loss;
                s.difficulty += diff;
                if (avoidable) s.avoidable++;
            }
        }
        return { index, player: /** @type {0 | 1} */ (player), loss, difficulty: diff, avoidable, cum: totals[player] };
    });
    for (const s of difficulty) {
        s.excess = s.loss - s.difficulty;
        s.ratio = s.difficulty >= DIFFICULTY_RATIO_FLOOR ? s.loss / s.difficulty : null;
    }
    const any = scored[0] + scored[1] > 0;
    return { items, totals, scored, any, difficulty, anyDifficulty: difficulty[0].decisions + difficulty[1].decisions > 0 };
}

/**
 * A signed excess as a percentage, "+1.23 %" or "−0.40 %".
 *
 * @param {number} excess
 */
export function fmtExcess(excess) {
    if (!Number.isFinite(excess)) return '';
    const v = (excess * 100).toFixed(2);
    if (v === '0.00' || v === '-0.00') return '0.00 %';
    return (excess > 0 ? '+' : '−') + v.replace('-', '') + ' %';
}

/**
 * A loss as a percentage of the match, "1.23 %"; empty when unscored. A positive
 * loss too small to show at two decimals reads "<0.01 %", so it is never taken
 * for the exact "0" of a decision that cost nothing.
 *
 * @param {number | null | undefined} loss
 */
export function fmtLoss(loss) {
    if (loss === null || loss === undefined || !Number.isFinite(loss)) return '';
    if (loss > 0 && loss < 0.00005) return '<0.01 %';
    return (loss * 100).toFixed(2) + ' %';
}

/**
 * A round ceiling at or above `max`, so the axis ends on a figure a reader can
 * name (1, 2, 2.5, 5 times a power of ten). 1 for nothing to draw.
 *
 * @param {number} max
 */
export function niceCeil(max) {
    if (!(max > 0)) return 1;
    const exp = Math.pow(10, Math.floor(Math.log10(max)));
    for (const step of [1, 2, 2.5, 5, 10]) {
        if (step * exp >= max - 1e-12) return step * exp;
    }
    return 10 * exp;
}

/**
 * Rows ordered by the loss they cost, the unscored ones last whichever the
 * direction. Stable; `''` keeps the match order.
 *
 * @template {{ loss: number | null }} T
 * @param {readonly T[]} moves
 * @param {'' | 'asc' | 'desc'} direction
 * @returns {T[]}
 */
export function sortByLoss(moves, direction) {
    if (!direction) return [...moves];
    const sign = direction === 'asc' ? 1 : -1;
    return [...moves].sort((a, b) => {
        if (a.loss === null && b.loss === null) return 0;
        if (a.loss === null) return 1;
        if (b.loss === null) return -1;
        return sign * (a.loss - b.loss);
    });
}
