// A synthetic table seen by a camera: a textured board, sensor noise, dice that tumble and
// settle, hands that come and go, checkers that move. A script says what is where at each
// instant of the media; `simulate` plays it as the live pipeline would — a 30 fps video, the
// rate gate following the detector's cadence, the describer — and returns the journal.

import { calibrateBoard } from '../../../vision/geometry/calibration.js';
import { boardRois } from '../../../vision/geometry/rois.js';
import { createDescriber, createDetector } from '../../../vision/descriptors.js';
import { createRateGate } from '../../../vision/sampler.js';
import { STATE_RATE } from '../../../vision/workerCore.js';
import { H, W, texture } from './frames.js';

/** Field corners in the picture, in click order: the home board is on the right. */
export const CLICKS = /** @type {[number, number][]} */ ([
    [280, 160],
    [40, 160],
    [40, 20],
    [280, 20]
]);

const board = calibrateBoard(CLICKS);
if (!board.ok) throw new Error('fixture board does not calibrate');
const rois = boardRois({ modelToImage: board.modelToImage, sourceWidth: W, sourceHeight: H });
if (!rois.ok) throw new Error('fixture board has no ROIs');
export const BOARD_CALIBRATION = rois.calibration;

/** @param {number} n */
function hash(n) {
    n = Math.imul(n ^ (n >>> 16), 0x45d9f3b);
    n = Math.imul(n ^ (n >>> 16), 0x45d9f3b);
    return (n ^ (n >>> 16)) >>> 0;
}

/**
 * @typedef {{ x: number, y: number, face?: number }} Die  top-left corner, 10×10 px
 * @typedef {{ x: number, y: number }} Hand  top-left corner, 44×36 px
 * @typedef {{ x: number, y: number }} Checker  centre, radius 6 px
 * @typedef {{ dice?: Die[], hands?: Hand[], checkers?: Checker[], cut?: boolean }} SceneState
 */

/**
 * @param {SceneState} s
 * @param {number} frame frame index of the video, seeds the sensor noise
 * @param {number} noise amplitude of the sensor noise, in grey levels
 */
export function render(s, frame, noise = 3) {
    const data = new Uint8ClampedArray(W * H * 4);
    const seed = s.cut ? 7 : 0;
    for (let y = 0; y < H; y++) {
        for (let x = 0; x < W; x++) {
            let v = s.cut ? texture(y, x, seed) : texture(x, y);
            for (const c of s.checkers ?? []) if ((x - c.x) ** 2 + (y - c.y) ** 2 < 36) v = 235 - ((x - c.x) ** 2 + (y - c.y) ** 2);
            for (const d of s.dice ?? []) {
                const dx = x - Math.round(d.x);
                const dy = y - Math.round(d.y);
                if (dx >= 0 && dx < 10 && dy >= 0 && dy < 10) v = (hash((d.face ?? 1) * 128 + dy * 10 + dx) & 3) === 0 ? 30 : 235;
            }
            for (const h of s.hands ?? []) {
                const dx = x - Math.round(h.x);
                const dy = y - Math.round(h.y);
                if (dx >= 0 && dx < 44 && dy >= 0 && dy < 36) v = 150 + dx + ((dx * 3 + dy * 5) % 11);
            }
            if (noise > 0) v += (hash(frame * 65536 + y * W + x) % (2 * noise + 1)) - noise;
            const i = (y * W + x) * 4;
            data[i] = data[i + 1] = data[i + 2] = Math.max(0, Math.min(255, v));
            data[i + 3] = 255;
        }
    }
    return data;
}

/**
 * Plays a script through the live pipeline.
 * @param {(t: number) => SceneState} script
 * @param {number} duration seconds of media
 * @param {{ fps?: number, noise?: number, thresholds?: object, start?: number, deliver?: number }} [options]
 *   `deliver` caps the analysed images per second of media, as a decoder that cannot keep up would
 */
export function simulate(script, duration, { fps = 30, noise = 3, thresholds, start = 0, deliver = Infinity } = {}) {
    const describer = createDescriber(BOARD_CALIBRATION);
    const detector = createDetector(/** @type {any} */ (thresholds));
    const gate = createRateGate(STATE_RATE);
    const records = [];
    const events = [];
    for (let f = Math.round(start * fps); f < duration * fps; f++) {
        const t = Math.round((f / fps) * 1e4) / 1e4;
        if (!gate.accept(t)) continue;
        const record = describer.describe({ mediaTime: t, data: render(script(t), f, noise) });
        records.push(record);
        events.push(...detector.push(record));
        const want = Math.min(detector.rate, deliver);
        if (want !== gate.rate) gate.setRate(want);
    }
    return { records, events, calibration: describer.calibration };
}

/** Linear interpolation of a path given as [t, x, y] knots, clamped at both ends. */
export function along(/** @type {[number, number, number][]} */ knots, /** @type {number} */ t) {
    if (t <= knots[0][0]) return { x: knots[0][1], y: knots[0][2] };
    for (let i = 1; i < knots.length; i++) {
        const [t1, x1, y1] = knots[i];
        const [t0, x0, y0] = knots[i - 1];
        if (t <= t1) {
            const a = (t - t0) / (t1 - t0);
            return { x: x0 + a * (x1 - x0), y: y0 + a * (y1 - y0) };
        }
    }
    const last = knots[knots.length - 1];
    return { x: last[1], y: last[2] };
}
