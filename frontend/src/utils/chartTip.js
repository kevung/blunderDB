// Where a chart's tooltip goes so that it stays whole inside its frame: beside
// the pointer, turned to the other side when it would cross an edge, and pushed
// back inside when the frame is smaller than both sides allow.

/**
 * @param {{ x: number, y: number }} anchor the pointer, in the frame's pixels
 * @param {{ w: number, h: number }} tip the tooltip's size
 * @param {{ w: number, h: number }} frame the box the tooltip must stay in
 * @param {number} [gap] the distance kept from the pointer
 * @returns {{ left: number, top: number }}
 */
export function placeTip(anchor, tip, frame, gap = 10) {
    return { left: side(anchor.x, tip.w, frame.w, gap), top: side(anchor.y, tip.h, frame.h, gap) };
}

/**
 * One axis: after the anchor if it fits, else before it, else as far inside as
 * the frame allows (its start when the tip is larger than the frame).
 * @param {number} at @param {number} size @param {number} room @param {number} gap
 */
function side(at, size, room, gap) {
    if (at + gap + size <= room) return at + gap;
    if (at - gap - size >= 0) return at - gap - size;
    return Math.max(0, Math.min(room - size, at - size / 2));
}
