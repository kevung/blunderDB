// Synthetic board images for the geometry tests: a pattern defined in model units,
// rendered through a known homography so every expectation is computable exactly.

import { BOARD_RECT, FIELD_HEIGHT, POINT_LENGTH, barZone, checkerCenter, pointZone, trayZone } from './boardModel.js';
import { computeHomography, applyHomography, invertHomography } from './homography.js';
import { CALIBRATION_CORNERS } from './boardModel.js';

export const BACKGROUND = 220;
export const FIELD = 120;
export const DARK = 40;
export const LIGHT = 180;
export const BAR = 70;
export const TRAY = 100;
export const CHECKER = 250;
export const CHECKER_DARK = 15;

/**
 * @typedef {[number | string, number, 'light' | 'dark']} Stack where (point 1..24, or
 *   'bar:near' | 'bar:far' | 'tray:near' | 'tray:far'), how many, which colour
 * @typedef {[number, number, number]} Checker centre u, v and luminance
 */

/**
 * @param {Stack[]} stacks
 * @returns {Checker[]}
 */
export function stackCheckers(stacks) {
    return stacks.flatMap(([where, count, colour]) => {
        const zone =
            typeof where === 'number'
                ? pointZone(where)
                : where.startsWith('bar:')
                  ? barZone(/** @type {'near' | 'far'} */ (where.slice(4)))
                  : trayZone(/** @type {'near' | 'far'} */ (where.slice(5)));
        const luma = colour === 'dark' ? CHECKER_DARK : CHECKER;
        return Array.from({ length: count }, (_, k) => /** @type {Checker} */ ([...checkerCenter(zone, k), luma]));
    });
}

/** A few stacks, enough structure for the presence score to lock onto. */
const CHECKERS = [
    [24, 2],
    [13, 5],
    [8, 3],
    [6, 5],
    [1, 2],
    [12, 5],
    [17, 3],
    [19, 5]
].flatMap(([n, count]) => Array.from({ length: count }, (_, k) => /** @type {Checker} */ ([...checkerCenter(pointZone(n), k), CHECKER])));

/**
 * Positions along a game, fifteen checkers a side: the start, a middle game with checkers on
 * the bar, and a bear-off with most of them in the trays.
 * @type {Record<'start' | 'middle' | 'bearoff', Stack[]>}
 */
export const POSITIONS = {
    start: [
        [24, 2, 'light'],
        [13, 5, 'light'],
        [8, 3, 'light'],
        [6, 5, 'light'],
        [1, 2, 'dark'],
        [12, 5, 'dark'],
        [17, 3, 'dark'],
        [19, 5, 'dark']
    ],
    middle: [
        [20, 3, 'light'],
        [11, 2, 'light'],
        [9, 2, 'light'],
        [7, 2, 'light'],
        [5, 3, 'light'],
        [4, 2, 'light'],
        ['bar:near', 1, 'light'],
        [23, 2, 'dark'],
        [21, 2, 'dark'],
        [16, 3, 'dark'],
        [14, 2, 'dark'],
        [10, 2, 'dark'],
        [3, 2, 'dark'],
        ['bar:far', 2, 'dark']
    ],
    bearoff: [
        [6, 1, 'light'],
        [5, 2, 'light'],
        [3, 3, 'light'],
        [2, 1, 'light'],
        ['tray:near', 8, 'light'],
        [19, 2, 'dark'],
        [20, 2, 'dark'],
        [22, 1, 'dark'],
        [24, 1, 'dark'],
        ['tray:far', 9, 'dark']
    ]
};

/**
 * Luminance of the model at (u, v), in d.
 * @param {number} u
 * @param {number} v
 * @param {{ checkers?: boolean | Checker[], points?: [number, number], background?: number }} [options]
 *   `checkers`: true for the default stacks, or the checkers to draw; `points`: luminance of the
 *   dark and light points (a felt board's points are much closer to each other than its
 *   checkers are); `background`: luminance of the frame and of all around the board
 */
export function modelLuma(u, v, { checkers = true, points = [DARK, LIGHT], background = BACKGROUND } = {}) {
    const { x0, x1, y0, y1 } = BOARD_RECT;
    if (u < x0 || u >= x1 || v < y0 || v >= y1) return background;
    if (checkers) {
        for (const c of checkers === true ? CHECKERS : checkers) {
            if (Math.hypot(u - c[0], v - c[1]) < 0.45) return c[2];
        }
    }
    if (u < 0) return TRAY;
    if (u >= 6 && u < 7) return BAR;
    const col = u < 6 ? Math.floor(u) : Math.floor(u) - 1;
    const inPoint = v < POINT_LENGTH || v >= FIELD_HEIGHT - POINT_LENGTH;
    if (!inPoint) return FIELD;
    const top = v >= FIELD_HEIGHT - POINT_LENGTH;
    return points[(col + (top ? 1 : 0)) % 2];
}

/**
 * Render the board seen through the camera whose field corners land on `clicks`.
 * `occluder`, in image pixels, paints an opaque block over the picture: a hand or an arm.
 * @param {{ width: number, height: number, clicks: [number, number][], gain?: number, offset?: number, rgba?: boolean, checkers?: boolean | Checker[], points?: [number, number], background?: number, occluder?: { x: number, y: number, w: number, h: number, luma: number } }} o
 */
export function renderBoard({ width, height, clicks, gain = 1, offset = 0, rgba = false, checkers = true, points, background, occluder }) {
    const res = computeHomography(CALIBRATION_CORNERS, clicks);
    if (!res.ok) throw new Error(res.reason);
    const toModel = invertHomography(res.H);
    if (!toModel) throw new Error('singular');
    const channels = rgba ? 4 : 1;
    const data = rgba ? new Uint8ClampedArray(width * height * 4) : new Float32Array(width * height);
    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            const [u, v] = applyHomography(toModel, [x + 0.5, y + 0.5]);
            const hidden = occluder && x >= occluder.x && x < occluder.x + occluder.w && y >= occluder.y && y < occluder.y + occluder.h;
            const l = Math.min(255, Math.max(0, (hidden ? occluder.luma : modelLuma(u, v, { checkers, points, background })) * gain + offset));
            const o = (y * width + x) * channels;
            if (rgba) {
                data[o] = data[o + 1] = data[o + 2] = l;
                data[o + 3] = 255;
            } else data[o] = l;
        }
    }
    return { width, height, data, H: res.H };
}

/**
 * A perspective view of the board in a 320×180 frame, home on the right.
 * @type {[number, number][]}
 */
export const OBLIQUE_CLICKS = [
    [262, 160],
    [52, 166],
    [88, 28],
    [236, 24]
];

/**
 * Deterministic pseudo-random numbers in [0, 1).
 * @param {number} seed
 */
export function lcg(seed) {
    let s = seed >>> 0;
    return () => {
        s = (Math.imul(s, 1664525) + 1013904223) >>> 0;
        return s / 2 ** 32;
    };
}
