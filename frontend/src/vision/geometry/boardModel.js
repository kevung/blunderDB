// Board model in checker diameters d, never in absolute sizes: published board
// dimensions contradict each other, while every board is drawn around its checkers.
//
// The frame is the bottom player's. Origin at the outer corner of his 1-point,
// u along the near edge towards his 12-point, v across the board towards the far edge.
// The four calibration clicks are the corners of the playing field in this order:
// 1-point, 12-point, 13-point, 24-point. Whether the home board appears on the left or
// on the right of the image is carried by the homography (a mirror), not by the model.

/** Width of a point and of the bar, in d. */
export const POINT_WIDTH = 1;
/** Length of a point: five checkers in a row. */
export const POINT_LENGTH = 5;
/** Space between the tips of facing points. */
export const GAP = 3;
export const BAR_WIDTH = 1;
/** Bear-off trays lie beside the home board, outside the clicked field. */
export const TRAY_WIDTH = 1;

export const FIELD_WIDTH = 12 * POINT_WIDTH + BAR_WIDTH;
export const FIELD_HEIGHT = 2 * POINT_LENGTH + GAP;

/**
 * Model coordinates of the four calibration clicks, in click order.
 * @type {[number, number][]}
 */
export const CALIBRATION_CORNERS = [
    [0, 0],
    [FIELD_WIDTH, 0],
    [FIELD_WIDTH, FIELD_HEIGHT],
    [0, FIELD_HEIGHT]
];

/**
 * @typedef {{ x0: number, y0: number, x1: number, y1: number }} Rect model-space, x0 < x1, y0 < y1
 * @typedef {{ id: string, kind: 'point' | 'bar' | 'tray', rect: Rect, base: [number, number], dir: [number, number] }} Zone
 *   `base` is the middle of the edge checkers stack from; `dir` the unit stacking direction.
 */

/** Left edge of the column holding a point, counting columns 0..11 from the 1-point side. */
function columnX(col) {
    return col * POINT_WIDTH + (col >= 6 ? BAR_WIDTH : 0);
}

/**
 * Zone of a point in the bottom player's numbering, 1..24.
 * @param {number} n
 * @returns {Zone}
 */
export function pointZone(n) {
    if (!Number.isInteger(n) || n < 1 || n > 24) throw new RangeError(`point ${n} outside 1..24`);
    const near = n <= 12;
    const x0 = columnX(near ? n - 1 : 24 - n);
    const x1 = x0 + POINT_WIDTH;
    const cx = x0 + POINT_WIDTH / 2;
    return near
        ? { id: `point:${n}`, kind: 'point', rect: { x0, y0: 0, x1, y1: POINT_LENGTH }, base: [cx, 0], dir: [0, 1] }
        : { id: `point:${n}`, kind: 'point', rect: { x0, y0: FIELD_HEIGHT - POINT_LENGTH, x1, y1: FIELD_HEIGHT }, base: [cx, FIELD_HEIGHT], dir: [0, -1] };
}

/**
 * Bar halves and trays are named by the side of the board they lie on, not by the
 * player who owns them: which player's checkers sit where is a house rule, not geometry.
 * @param {'near' | 'far'} side
 * @returns {Zone}
 */
export function barZone(side) {
    const x0 = columnX(6) - BAR_WIDTH;
    const x1 = x0 + BAR_WIDTH;
    const cx = x0 + BAR_WIDTH / 2;
    const mid = FIELD_HEIGHT / 2;
    if (side === 'near') return { id: 'bar:near', kind: 'bar', rect: { x0, y0: 0, x1, y1: mid }, base: [cx, mid], dir: [0, -1] };
    if (side === 'far') return { id: 'bar:far', kind: 'bar', rect: { x0, y0: mid, x1, y1: FIELD_HEIGHT }, base: [cx, mid], dir: [0, 1] };
    throw new RangeError(`bar side ${side}`);
}

/**
 * @param {'near' | 'far'} side
 * @returns {Zone}
 */
export function trayZone(side) {
    const x0 = -TRAY_WIDTH;
    const cx = x0 + TRAY_WIDTH / 2;
    const mid = FIELD_HEIGHT / 2;
    if (side === 'near') return { id: 'tray:near', kind: 'tray', rect: { x0, y0: 0, x1: 0, y1: mid }, base: [cx, 0], dir: [0, 1] };
    if (side === 'far') return { id: 'tray:far', kind: 'tray', rect: { x0, y0: mid, x1: 0, y1: FIELD_HEIGHT }, base: [cx, FIELD_HEIGHT], dir: [0, -1] };
    throw new RangeError(`tray side ${side}`);
}

/**
 * The 28 zones: 24 points, two bar halves, two trays.
 * @returns {Zone[]}
 */
export function boardZones() {
    const zones = [];
    for (let n = 1; n <= 24; n++) zones.push(pointZone(n));
    zones.push(barZone('near'), barZone('far'), trayZone('near'), trayZone('far'));
    return zones;
}

/** Whole rectified frame: field and trays. @type {Rect} */
export const BOARD_RECT = { x0: -TRAY_WIDTH, y0: 0, x1: FIELD_WIDTH, y1: FIELD_HEIGHT };
/** The clicked field alone. @type {Rect} */
export const FIELD_RECT = { x0: 0, y0: 0, x1: FIELD_WIDTH, y1: FIELD_HEIGHT };

/**
 * Centre of the k-th checker (0-based) of a stack: (k + 0.5)·d from the base edge.
 * Beyond five the layout of a pile is not settled, so callers measure an occupied
 * length rather than ask for fixed slots past the point's length.
 * @param {Zone} zone
 * @param {number} k
 * @returns {[number, number]}
 */
export function checkerCenter(zone, k) {
    const t = (k + 0.5) * POINT_WIDTH;
    return [zone.base[0] + zone.dir[0] * t, zone.base[1] + zone.dir[1] * t];
}

/**
 * Checker centres of the starting position, for the ghost checkers that check the
 * calibration's orientation. `near` are the bottom player's.
 * @returns {{ side: 'near' | 'far', center: [number, number] }[]}
 */
export function initialPositionGhosts() {
    const start = [
        [24, 2],
        [13, 5],
        [8, 3],
        [6, 5]
    ];
    const ghosts = [];
    for (const [n, count] of start) {
        for (let k = 0; k < count; k++) {
            ghosts.push({ side: 'near', center: checkerCenter(pointZone(n), k) });
            ghosts.push({ side: 'far', center: checkerCenter(pointZone(25 - n), k) });
        }
    }
    return ghosts;
}
