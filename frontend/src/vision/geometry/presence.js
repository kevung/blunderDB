// Is the calibrated board still in view? A cut, a zoom or a camera move shows up as a
// drop of the similarity between the board rectified now and the board rectified at
// calibration; hysteresis keeps a noisy score from blinking "view lost".

/** Below this the board is lost; above FOUND it is back. Between, the state holds. */
export const PRESENCE_LOST = 0.5;
export const PRESENCE_FOUND = 0.7;
/** Share of the zone that must be sampled on both sides for a score to mean anything. */
const MIN_OVERLAP = 0.5;

/**
 * Similarity in [0, 1] of two rectified thumbnails of the same size: the zero-mean
 * normalised cross-correlation, clamped at 0. Subtracting the mean and dividing by the
 * spread makes it blind to exposure gain and offset, which a camera's auto-exposure
 * changes constantly; an anti-correlated or flat image scores 0.
 *
 * NaN pixels (outside the source image) are skipped on both sides.
 *
 * @param {{ width: number, height: number, data: ArrayLike<number> }} reference
 * @param {{ width: number, height: number, data: ArrayLike<number> }} current
 * @returns {number}
 */
export function presenceScore(reference, current) {
    if (reference.width !== current.width || reference.height !== current.height) throw new RangeError('thumbnails of different sizes');
    const a = reference.data;
    const b = current.data;
    const n = a.length;
    let count = 0;
    let sa = 0;
    let sb = 0;
    for (let i = 0; i < n; i++) {
        if (Number.isNaN(a[i]) || Number.isNaN(b[i])) continue;
        count++;
        sa += a[i];
        sb += b[i];
    }
    if (count < MIN_OVERLAP * n || count < 2) return 0;
    const ma = sa / count;
    const mb = sb / count;
    let saa = 0;
    let sbb = 0;
    let sab = 0;
    for (let i = 0; i < n; i++) {
        if (Number.isNaN(a[i]) || Number.isNaN(b[i])) continue;
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
 * made on a visible board.
 * @param {{ lost?: number, found?: number, present?: boolean }} [options]
 */
export function createPresenceTracker({ lost = PRESENCE_LOST, found = PRESENCE_FOUND, present = true } = {}) {
    let state = present;
    return {
        /** @param {number} score @returns {{ present: boolean, changed: boolean }} */
        update(score) {
            const next = nextPresence(state, score, { lost, found });
            const changed = next !== state;
            state = next;
            return { present: state, changed };
        },
        get present() {
            return state;
        }
    };
}
