// From the board homography to the calibration the Worker analyses: the two halves of the
// playing field as ROIs, and a witness off the board.
//
// The contract takes axis-aligned rectangles of the reduced frame, so each half is the
// bounding box of its four projected corners: under perspective it overlaps a little of the
// bar or of the frame, which the census sees as still background and costs nothing. The bar
// itself is left out — dice landing there are cocked and rethrown.

import { BAR_WIDTH, BOARD_RECT, FIELD_HEIGHT, FIELD_WIDTH } from './boardModel.js';
import { applyHomography } from './homography.js';

/** Witness size in reduced-frame pixels: big enough for a stable census, small enough to fit beside a board that fills the picture. */
export const WITNESS_MAX = Object.freeze({ w: 48, h: 24 });

const EDGE = 1e-6;

const HALF_WIDTH = (FIELD_WIDTH - BAR_WIDTH) / 2;

/** The two halves of the field in model units: home (columns 1–6) and outer (7–12). */
const HALVES = [
    { x0: 0, x1: HALF_WIDTH },
    { x0: HALF_WIDTH + BAR_WIDTH, x1: FIELD_WIDTH }
];

/**
 * @param {number[]} H model → source image
 * @param {{ x0: number, x1: number }} half
 * @param {number} sx
 * @param {number} sy
 */
function projectedBox(H, half, sx, sy) {
    const pts = [
        [half.x0, 0],
        [half.x1, 0],
        [half.x1, FIELD_HEIGHT],
        [half.x0, FIELD_HEIGHT]
    ].map((p) => {
        const [x, y] = applyHomography(H, /** @type {[number, number]} */ (p));
        return [x * sx, y * sy];
    });
    const xs = pts.map((p) => p[0]);
    const ys = pts.map((p) => p[1]);
    return { x0: Math.min(...xs), x1: Math.max(...xs), y0: Math.min(...ys), y1: Math.max(...ys) };
}

/**
 * @param {{ x0: number, x1: number, y0: number, y1: number }} b
 * @param {number} width
 * @param {number} height
 */
function clampRect(b, width, height) {
    // The tolerance keeps a corner that lands on a pixel edge there, whatever the float error.
    const x = Math.max(0, Math.floor(b.x0 + EDGE));
    const y = Math.max(0, Math.floor(b.y0 + EDGE));
    const x1 = Math.min(width, Math.ceil(b.x1 - EDGE));
    const y1 = Math.min(height, Math.ceil(b.y1 - EDGE));
    return { x, y, w: x1 - x, h: y1 - y };
}

/**
 * The largest witness, capped at `WITNESS_MAX`, in one of the four bands the board's bounding
 * box leaves free; `null` when the board fills the frame to within 3 pixels.
 * @param {{ x: number, y: number, w: number, h: number }} box
 * @param {number} width
 * @param {number} height
 */
function witnessBeside(box, width, height) {
    const bands = [
        { x: 0, y: 0, w: width, h: box.y },
        { x: 0, y: box.y + box.h, w: width, h: height - box.y - box.h },
        { x: 0, y: 0, w: box.x, h: height },
        { x: box.x + box.w, y: 0, w: width - box.x - box.w, h: height }
    ];
    let best = null;
    for (const b of bands) {
        const w = Math.min(b.w, WITNESS_MAX.w);
        const h = Math.min(b.h, WITNESS_MAX.h);
        if (w < 3 || h < 3) continue;
        if (!best || w * h > best.w * best.h) best = { x: b.x, y: b.y, w, h };
    }
    return best;
}

/**
 * The calibration of `contract.js` for a calibrated board.
 *
 * ROIs are named `left` and `right` by their place in the picture, which is the `side` the
 * dice events report: the home board's side is a fact of the camera, not of the player.
 *
 * @param {object} options
 * @param {number[]} options.modelToImage homography from `calibrateBoard`, in source pixels
 * @param {number} options.sourceWidth width of the image the clicks were taken on
 * @param {number} options.sourceHeight
 * @param {number} [options.width] reduced frame, 320 by default
 * @param {number} [options.height] reduced frame, 180 by default
 * @returns {{ ok: true, calibration: { width: number, height: number, rois: { name: string, x: number, y: number, w: number, h: number }[], witness: { x: number, y: number, w: number, h: number } } } | { ok: false, reason: 'outside' | 'no-witness' }}
 */
export function boardRois({ modelToImage, sourceWidth, sourceHeight, width = 320, height = 180 }) {
    const sx = width / sourceWidth;
    const sy = height / sourceHeight;
    const boxes = HALVES.map((h) => projectedBox(modelToImage, h, sx, sy));
    const rects = boxes.map((b) => clampRect(b, width, height));
    if (rects.some((r) => r.w < 3 || r.h < 3)) return { ok: false, reason: 'outside' };
    const [a, b] = boxes[0].x0 + boxes[0].x1 <= boxes[1].x0 + boxes[1].x1 ? [rects[0], rects[1]] : [rects[1], rects[0]];
    // The witness keeps clear of the trays too: checkers borne off move there.
    const union = clampRect(projectedBox(modelToImage, { x0: BOARD_RECT.x0, x1: BOARD_RECT.x1 }, sx, sy), width, height);
    const witness = witnessBeside(union, width, height);
    if (!witness) return { ok: false, reason: 'no-witness' };
    return {
        ok: true,
        calibration: {
            width,
            height,
            rois: [
                { name: 'left', ...a },
                { name: 'right', ...b }
            ],
            witness
        }
    };
}
