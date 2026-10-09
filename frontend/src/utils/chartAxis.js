// The decision axis the match charts share: which decision a pointer is over,
// and how the arrow keys walk it.

/**
 * @param {MouseEvent} e a pointer event on the plot
 * @param {number} count decisions on the axis
 * @returns {number}
 */
export function indexAt(e, count) {
    const r = /** @type {HTMLElement} */ (e.currentTarget).getBoundingClientRect();
    const i = Math.floor(((e.clientX - r.left) / Math.max(1, r.width)) * count);
    return Math.min(count - 1, Math.max(0, i));
}

/**
 * The decision a key moves to: arrows step, Home and End jump to the ends.
 * `null` for any other key.
 *
 * @param {string} key
 * @param {number | null} current
 * @param {number} count
 * @returns {number | null}
 */
export function stepIndex(key, current, count) {
    if (count < 1) return null;
    if (key === 'Home') return 0;
    if (key === 'End') return count - 1;
    if (key !== 'ArrowRight' && key !== 'ArrowLeft') return null;
    const step = key === 'ArrowRight' ? 1 : -1;
    return Math.min(count - 1, Math.max(0, (current ?? (step > 0 ? -1 : count)) + step));
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
