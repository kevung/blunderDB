// Where a chart's tooltip goes so that it stays whole inside its frame: beside
// the pointer, turned to the other side when it would cross an edge, and pushed
// back inside when the frame is smaller than both sides allow.

/**
 * The frame is the plot, `w` × `h`. A tooltip taller than the plot (a short
 * plot in a narrow panel) cannot stay inside it: it then stays inside the
 * visible part of the scrolling container, from `top` to `bottom` in the
 * plot's pixels (`top` negative when the container shows room above the plot).
 *
 * @param {{ x: number, y: number }} anchor the pointer, in the plot's pixels
 * @param {{ w: number, h: number }} tip the tooltip's size
 * @param {{ w: number, h: number, top?: number, bottom?: number }} frame
 * @param {number} [gap] the distance kept from the pointer
 * @returns {{ left: number, top: number }}
 */
export function placeTip(anchor, tip, frame, gap = 10) {
    const tall = tip.h > frame.h && frame.top !== undefined && frame.bottom !== undefined;
    const [lo, hi] = tall ? [/** @type {number} */ (frame.top), /** @type {number} */ (frame.bottom)] : [0, frame.h];
    return { left: side(anchor.x, tip.w, 0, frame.w, gap), top: side(anchor.y, tip.h, lo, hi, gap) };
}

/**
 * One axis: after the anchor if it fits, else before it, else as far inside
 * [lo, hi] as it allows (its start when the tip is larger than the room).
 * @param {number} at @param {number} size @param {number} lo @param {number} hi @param {number} gap
 */
function side(at, size, lo, hi, gap) {
    if (at + gap + size <= hi) return at + gap;
    if (at - gap - size >= lo) return at - gap - size;
    return Math.max(lo, Math.min(hi - size, at - size / 2));
}
