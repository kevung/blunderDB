// The match winning chances each decision cost (MatchDecisionLosses). A loss is
// a fraction, as Match.mwc_loss is; a decision the analysis does not price has
// a null loss and reads as unscored, never as zero.

/**
 * The decisions of the match in match order, each with its loss, and the running
 * total per player. Indexed like `movePositions`, so the chart shares the
 * time chart's decision axis. A Move absent from `losses` is unscored.
 *
 * @param {readonly { move_id: number, player_on_roll: number }[]} movePositions
 * @param {readonly { move_id: number, mwc_loss: number | null }[] | null | undefined} losses
 * @returns {{ items: { index: number, player: 0 | 1, loss: number | null, cum: number }[], totals: [number, number], scored: [number, number], any: boolean }}
 */
export function lossSeries(movePositions, losses) {
    const byMove = new Map((losses ?? []).map((d) => [d.move_id, d.mwc_loss]));
    /** @type {[number, number]} */
    const totals = [0, 0];
    /** @type {[number, number]} */
    const scored = [0, 0];
    const items = movePositions.map((mp, index) => {
        const player = mp.player_on_roll === 1 ? 1 : 0;
        const loss = byMove.get(mp.move_id) ?? null;
        if (loss !== null) {
            totals[player] += loss;
            scored[player]++;
        }
        return { index, player: /** @type {0 | 1} */ (player), loss, cum: totals[player] };
    });
    return { items, totals, scored, any: scored[0] + scored[1] > 0 };
}

/**
 * A loss as a percentage of the match, "1.23 %"; empty when unscored.
 *
 * @param {number | null | undefined} loss
 */
export function fmtLoss(loss) {
    if (loss === null || loss === undefined || !Number.isFinite(loss)) return '';
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
