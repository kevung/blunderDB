// Is the calibrated board still in view? A cut, a zoom or a camera move shows up as a
// drop of the similarity between the board rectified now and the board rectified at
// calibration; hysteresis keeps a noisy score from blinking "view lost".
//
// Only what the game never changes is compared: the frame along the two long edges and beyond
// the tray, and the empty strip between the point tips. Checkers, the bar and the trays are
// masked out, because a position far from the one at calibration is still the same view. A
// hand or an arm crossing the board does hide the compared zones, but only for a moment: the
// tracker declares the view lost only after the score has stayed low for several readings.

import { BOARD_RECT, FIELD_HEIGHT, FIELD_WIDTH, POINT_LENGTH, barZone } from './boardModel.js';

const BAR = barZone('near').rect;

/** Below this the board is lost; above FOUND it is back. Between, the state holds. */
export const PRESENCE_LOST = 0.5;
export const PRESENCE_FOUND = 0.7;
/** Share of the zone sampled in the reference that must be sampled now for a score to mean anything. */
const MIN_OVERLAP = 0.5;

/**
 * Width of frame read outside the board, in d. A board fills its picture tightly at times:
 * whatever of this band falls outside the picture is outside it in the reference too, and is
 * simply not compared.
 */
export const FRAME_BAND = 0.5;
/** The rectified zone the presence thumbnail covers: the board and a band of its frame. @type {import('./boardModel.js').Rect} */
export const PRESENCE_RECT = Object.freeze({ x0: BOARD_RECT.x0 - FRAME_BAND, y0: BOARD_RECT.y0 - FRAME_BAND, x1: BOARD_RECT.x1, y1: BOARD_RECT.y1 + FRAME_BAND });

/**
 * Whether the game can leave the model point (u, v) unchanged: true on the frame band (the
 * side beyond the clicked field is left out, since many boards carry a second tray there) and
 * in the strip between facing point tips, bar excluded, since bar checkers stack from its
 * middle.
 * @param {number} u
 * @param {number} v
 */
function isStatic(u, v) {
    if (u >= FIELD_WIDTH) return false;
    if (v < 0 || v >= FIELD_HEIGHT || u < BOARD_RECT.x0) return true;
    if (u < 0) return false; // tray
    if (u >= BAR.x0 && u < BAR.x1) return false;
    return v >= POINT_LENGTH && v < FIELD_HEIGHT - POINT_LENGTH;
}

/**
 * The pixels of a thumbnail of `rect` that presenceScore compares (1) or ignores (0), judged
 * at each pixel's centre as rectifyZone samples it.
 * @param {number} width
 * @param {number} height
 * @param {import('./boardModel.js').Rect} [rect]
 * @returns {Uint8Array}
 */
export function presenceMask(width, height, rect = PRESENCE_RECT) {
    const mask = new Uint8Array(width * height);
    const sx = (rect.x1 - rect.x0) / width;
    const sy = (rect.y1 - rect.y0) / height;
    for (let j = 0; j < height; j++) {
        const v = rect.y0 + (j + 0.5) * sy;
        for (let i = 0; i < width; i++) mask[j * width + i] = isStatic(rect.x0 + (i + 0.5) * sx, v) ? 1 : 0;
    }
    return mask;
}

/**
 * Similarity in [0, 1] of two rectified thumbnails of the same size: the zero-mean
 * normalised cross-correlation, clamped at 0. Subtracting the mean and dividing by the
 * spread makes it blind to exposure gain and offset, which a camera's auto-exposure
 * changes constantly; an anti-correlated or flat image scores 0.
 *
 * NaN pixels (outside the source image) are skipped on both sides, and so are the pixels
 * the mask zeroes.
 *
 * @param {{ width: number, height: number, data: ArrayLike<number> }} reference
 * @param {{ width: number, height: number, data: ArrayLike<number> }} current
 * @param {ArrayLike<number>} [mask] 1 where the pixel is compared; every pixel by default
 * @returns {number}
 */
export function presenceScore(reference, current, mask) {
    if (reference.width !== current.width || reference.height !== current.height) throw new RangeError('thumbnails of different sizes');
    const a = reference.data;
    const b = current.data;
    const n = a.length;
    if (mask && mask.length !== n) throw new RangeError('mask of a different size');
    let expected = 0;
    let count = 0;
    let sa = 0;
    let sb = 0;
    for (let i = 0; i < n; i++) {
        if ((mask && !mask[i]) || Number.isNaN(a[i])) continue;
        expected++;
        if (Number.isNaN(b[i])) continue;
        count++;
        sa += a[i];
        sb += b[i];
    }
    if (count < MIN_OVERLAP * expected || count < 2) return 0;
    const ma = sa / count;
    const mb = sb / count;
    let saa = 0;
    let sbb = 0;
    let sab = 0;
    for (let i = 0; i < n; i++) {
        if ((mask && !mask[i]) || Number.isNaN(a[i]) || Number.isNaN(b[i])) continue;
        const da = a[i] - ma;
        const db = b[i] - mb;
        saa += da * da;
        sbb += db * db;
        sab += da * db;
    }
    // A flat image has no structure to match: a black frame at a cut is not the board.
    const denom = Math.sqrt(saa * sbb);
    if (!(denom > 1e-9 * count)) return 0;
    return Math.min(1, Math.max(0, sab / denom));
}

/**
 * Next presence state from the current one and a new score. A non-finite score
 * carries no evidence and leaves the state unchanged.
 * @param {boolean} present
 * @param {number} score
 * @param {{ lost?: number, found?: number }} [thresholds]
 * @returns {boolean}
 */
export function nextPresence(present, score, { lost = PRESENCE_LOST, found = PRESENCE_FOUND } = {}) {
    if (!(lost < found)) throw new RangeError('lost threshold must be below found threshold');
    if (!Number.isFinite(score)) return present;
    if (present) return score >= lost;
    return score > found;
}

/**
 * Stateful wrapper around nextPresence. A calibration starts present: it was just
 * made on a visible board. The view is lost only after `hold` scores in a row below the lost
 * threshold, so that a hand passing over the board does not silence the analysis; it is found
 * again at the first score above the found threshold, since a board seen again is evidence
 * enough. A non-finite score neither extends nor breaks a run of low scores.
 * @param {{ lost?: number, found?: number, present?: boolean, hold?: number }} [options]
 */
export function createPresenceTracker({ lost = PRESENCE_LOST, found = PRESENCE_FOUND, present = true, hold = 1 } = {}) {
    if (!(Number.isInteger(hold) && hold >= 1)) throw new RangeError('hold must be a positive integer');
    let state = present;
    let low = 0;
    let lowSince = NaN;
    return {
        /**
         * @param {number} score
         * @param {number} [at] the instant of the reading, returned as `since` when the view is lost
         * @returns {{ present: boolean, changed: boolean, since: number }} `since`: the instant of
         *   the first low score of the run that lost the view, NaN otherwise
         */
        update(score, at = NaN) {
            if (!Number.isFinite(score)) return { present: state, changed: false, since: NaN };
            const before = state;
            if (state) {
                if (nextPresence(true, score, { lost, found })) low = 0;
                else if (++low === 1) lowSince = at;
                if (low >= hold) state = false;
            } else if ((state = nextPresence(false, score, { lost, found }))) low = 0;
            const changed = state !== before;
            return { present: state, changed, since: changed && !state ? lowSince : NaN };
        },
        get present() {
            return state;
        }
    };
}
