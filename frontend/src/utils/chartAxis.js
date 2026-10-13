// The decision axis the match charts share: which decision a pointer is over,
// and how the arrow keys walk it.

/**
 * The decisions a chart answers to, in match order: those of the players
 * shown.
 *
 * @param {readonly { player_on_roll: number }[]} movePositions
 * @param {readonly [boolean, boolean]} shown
 * @returns {number[]}
 */
export function visibleIndices(movePositions, shown) {
    /** @type {number[]} */
    const out = [];
    movePositions.forEach((mp, i) => {
        if (shown[mp.player_on_roll === 1 ? 1 : 0]) out.push(i);
    });
    return out;
}

/**
 * The visible decision closest to `i`; the earlier one on a tie.
 *
 * @param {number} i
 * @param {readonly number[]} visible ascending
 * @returns {number | null}
 */
export function nearestVisible(i, visible) {
    let best = null;
    for (const v of visible) {
        if (best === null || Math.abs(v - i) < Math.abs(best - i)) best = v;
    }
    return best;
}

/**
 * The decision under a pointer. With `visible`, a decision of a hidden player
 * gives the closest visible one.
 *
 * @param {MouseEvent} e a pointer event on the plot
 * @param {number} count decisions on the axis
 * @param {readonly number[]} [visible]
 * @returns {number}
 */
export function indexAt(e, count, visible) {
    const r = /** @type {HTMLElement} */ (e.currentTarget).getBoundingClientRect();
    const i = Math.floor(((e.clientX - r.left) / Math.max(1, r.width)) * count);
    const clamped = Math.min(count - 1, Math.max(0, i));
    return visible ? (nearestVisible(clamped, visible) ?? clamped) : clamped;
}

/**
 * The decision a key moves to: arrows step, Home and End jump to the ends.
 * `null` for any other key. With `visible`, only those decisions are walked.
 *
 * @param {string} key
 * @param {number | null} current
 * @param {number} count
 * @param {readonly number[]} [visible] ascending
 * @returns {number | null}
 */
export function stepIndex(key, current, count, visible) {
    if (visible) {
        if (visible.length < 1) return null;
        if (key === 'Home') return visible[0];
        if (key === 'End') return visible[visible.length - 1];
        if (key !== 'ArrowRight' && key !== 'ArrowLeft') return null;
        if (current === null) return key === 'ArrowRight' ? visible[0] : visible[visible.length - 1];
        const next = key === 'ArrowRight' ? visible.find((v) => v > current) : visible.findLast((v) => v < current);
        if (next !== undefined) return next;
        return key === 'ArrowRight' ? visible[visible.length - 1] : visible[0];
    }
    if (count < 1) return null;
    if (key === 'Home') return 0;
    if (key === 'End') return count - 1;
    if (key !== 'ArrowRight' && key !== 'ArrowLeft') return null;
    const step = key === 'ArrowRight' ? 1 : -1;
    return Math.min(count - 1, Math.max(0, (current ?? (step > 0 ? -1 : count)) + step));
}

/**
 * Toggles a player in the legend; the last one shown cannot be hidden.
 *
 * @param {readonly [boolean, boolean]} shown
 * @param {0 | 1} p
 * @returns {[boolean, boolean]}
 */
export function togglePlayer(shown, p) {
    if (shown[p] && !shown[1 - p]) return [shown[0], shown[1]];
    return p === 0 ? [!shown[0], shown[1]] : [shown[0], !shown[1]];
}

/**
 * The games along the decision axis: where each starts and ends, so a chart
 * can mark the boundaries and name each stretch. `movePositions` must be in
 * match order.
 *
 * @param {readonly { game_number: number }[]} movePositions
 * @returns {{ game: number, from: number, to: number }[]} `to` is exclusive
 */
export function gameSpans(movePositions) {
    /** @type {{ game: number, from: number, to: number }[]} */
    const spans = [];
    movePositions.forEach((mp, i) => {
        const last = spans[spans.length - 1];
        if (last && last.game === mp.game_number) last.to = i + 1;
        else spans.push({ game: mp.game_number, from: i, to: i + 1 });
    });
    return spans;
}
