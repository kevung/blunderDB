// The match winning chances each decision cost (MatchDecisionLosses). A loss is
// a fraction, as Match.mwc_loss is; a decision the analysis does not price has
// a null loss and reads as unscored, never as zero.

/**
 * @typedef {{ decisions: number, loss: number, difficulty: number, excess: number, ratio: number | null, avoidable: number }} DifficultySummary
 * One player's reading of ADR-0076 over the decisions carrying both a loss and
 * a difficulty: storage.DifficultySummary, served in the match review.
 */

/**
 * The two players' difficulty summaries as the match review serves them. The
 * excess and the ratio are Go's (storage.SummariseDifficulty), never added up
 * here, so the panel and `match --format summary` cannot disagree.
 *
 * @param {{ players?: { difficulty?: DifficultySummary }[] } | null | undefined} review
 * @returns {{ difficulty: [DifficultySummary | null, DifficultySummary | null], any: boolean }}
 */
export function servedDifficulty(review) {
    const d = /** @type {[DifficultySummary | null, DifficultySummary | null]} */ ([0, 1].map((p) => review?.players?.[p]?.difficulty ?? null));
    return { difficulty: d, any: (d[0]?.decisions ?? 0) + (d[1]?.decisions ?? 0) > 0 };
}

/**
 * The decisions of the match in match order, each with its loss, difficulty
 * and avoidable mark, and the running total per player. Indexed like
 * `movePositions`, so the chart shares the time chart's decision axis. A Move
 * absent from `losses` is unscored.
 *
 * @param {readonly { move_id: number, player_on_roll: number }[]} movePositions
 * @param {readonly { move_id: number, mwc_loss: number | null, difficulty?: number | null, avoidable?: boolean }[] | null | undefined} losses
 * @returns {{ items: { index: number, player: 0 | 1, loss: number | null, difficulty: number | null, avoidable: boolean, cum: number }[], totals: [number, number], scored: [number, number], any: boolean }}
 */
export function lossSeries(movePositions, losses) {
    const byMove = new Map((losses ?? []).map((d) => [d.move_id, d]));
    /** @type {[number, number]} */
    const totals = [0, 0];
    /** @type {[number, number]} */
    const scored = [0, 0];
    const items = movePositions.map((mp, index) => {
        const player = mp.player_on_roll === 1 ? 1 : 0;
        const d = byMove.get(mp.move_id);
        const loss = d?.mwc_loss ?? null;
        const diff = loss === null ? null : (d?.difficulty ?? null);
        const avoidable = !!d?.avoidable;
        if (loss !== null) {
            totals[player] += loss;
            scored[player]++;
        }
        return { index, player: /** @type {0 | 1} */ (player), loss, difficulty: diff, avoidable, cum: totals[player] };
    });
    const any = scored[0] + scored[1] > 0;
    return { items, totals, scored, any };
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

/**
 * Player 1's match winning chances along the decision axis of the charts
 * (MatchWinTrace): per Move of movePositions, its chances before and after
 * the decision, null where the trace has none (no analysis, an unscored play),
 * and whether the decision is an error. `any` is false at money, or when no
 * decision has a value: nothing is then drawn.
 *
 * @param {any[]} movePositions
 * @param {{ money?: boolean, decisions?: any[] } | null | undefined} trace
 */
export function winTraceSeries(movePositions, trace) {
    if (!trace || trace.money) return { items: [], any: false };
    const byMove = new Map((trace.decisions ?? []).map((d) => [d.move_id, d]));
    const items = movePositions.map((mp, index) => {
        const d = byMove.get(mp.move_id);
        return {
            index,
            player: /** @type {0 | 1} */ (d?.player === 1 ? 1 : 0),
            before: d?.mwc_before ?? null,
            after: d?.mwc_after ?? null,
            error: !!d?.error
        };
    });
    return { items, any: items.some((d) => d.before !== null) };
}

/**
 * The runs of the curve: consecutive decisions that have a value, each point
 * at its decision's left edge (before) and right edge (after, or before when
 * the play is unscored). A decision without value breaks the line.
 *
 * @param {{ index: number, before: number | null, after: number | null }[]} items
 * @returns {{ x: number, v: number }[][]} x in decision slots
 */
export function winTraceRuns(items) {
    /** @type {{ x: number, v: number }[][]} */
    const runs = [];
    /** @type {{ x: number, v: number }[] | null} */
    let run = null;
    for (const d of items) {
        if (d.before === null) {
            run = null;
            continue;
        }
        if (!run) {
            run = [];
            runs.push(run);
        }
        run.push({ x: d.index, v: d.before }, { x: d.index + 1, v: d.after ?? d.before });
    }
    return runs;
}
