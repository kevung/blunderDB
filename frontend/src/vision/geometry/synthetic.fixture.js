// Synthetic board images for the geometry tests: a pattern defined in model units,
// rendered through a known homography so every expectation is computable exactly.

import { BOARD_RECT, FIELD_HEIGHT, POINT_LENGTH, initialPositionGhosts } from './boardModel.js';
import { computeHomography, applyHomography, invertHomography } from './homography.js';
import { CALIBRATION_CORNERS } from './boardModel.js';

export const BACKGROUND = 220;
export const FIELD = 120;
export const DARK = 40;
export const LIGHT = 180;
export const BAR = 70;
export const TRAY = 100;
export const CHECKER = 250;

/** Luminance of the model at (u, v), in d. */
export function modelLuma(u, v, { checkers = true } = {}) {
    const { x0, x1, y0, y1 } = BOARD_RECT;
    if (u < x0 || u >= x1 || v < y0 || v >= y1) return BACKGROUND;
    if (checkers) {
        for (const g of initialPositionGhosts()) {
            if (Math.hypot(u - g.center[0], v - g.center[1]) < 0.45) return CHECKER;
        }
    }
    if (u < 0) return TRAY;
    if (u >= 6 && u < 7) return BAR;
    const col = u < 6 ? Math.floor(u) : Math.floor(u) - 1;
    const inPoint = v < POINT_LENGTH || v >= FIELD_HEIGHT - POINT_LENGTH;
    if (!inPoint) return FIELD;
    const top = v >= FIELD_HEIGHT - POINT_LENGTH;
    return (col + (top ? 1 : 0)) % 2 === 0 ? DARK : LIGHT;
}

/**
 * Render the board seen through the camera whose field corners land on `clicks`.
 * @param {{ width: number, height: number, clicks: [number, number][], gain?: number, offset?: number, rgba?: boolean, checkers?: boolean }} o
 */
export function renderBoard({ width, height, clicks, gain = 1, offset = 0, rgba = false, checkers = true }) {
    const res = computeHomography(CALIBRATION_CORNERS, clicks);
    if (!res.ok) throw new Error(res.reason);
    const toModel = invertHomography(res.H);
    const channels = rgba ? 4 : 1;
    const data = rgba ? new Uint8ClampedArray(width * height * 4) : new Float32Array(width * height);
    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            const [u, v] = applyHomography(toModel, [x + 0.5, y + 0.5]);
            const l = Math.min(255, Math.max(0, modelLuma(u, v, { checkers }) * gain + offset));
            const o = (y * width + x) * channels;
            if (rgba) {
                data[o] = data[o + 1] = data[o + 2] = l;
                data[o + 3] = 255;
            } else data[o] = l;
        }
    }
    return { width, height, data, H: res.H };
}

/** A perspective view of the board in a 320×180 frame, home on the right. */
export const OBLIQUE_CLICKS = [
    [262, 160],
    [52, 166],
    [88, 28],
    [236, 24]
];

/** Deterministic pseudo-random numbers in [0, 1). */
export function lcg(seed) {
    let s = seed >>> 0;
    return () => {
        s = (Math.imul(s, 1664525) + 1013904223) >>> 0;
        return s / 2 ** 32;
    };
}
