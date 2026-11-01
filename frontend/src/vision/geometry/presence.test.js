import { describe, it, expect } from 'vitest';
import { presenceScore, presenceMask, nextPresence, createPresenceTracker, PRESENCE_RECT } from './presence.js';
import { rectifyZone } from './rectify.js';
import { calibrateBoard } from './calibration.js';
import { BOARD_RECT } from './boardModel.js';
import { renderBoard, OBLIQUE_CLICKS, POSITIONS, lcg, stackCheckers } from './synthetic.fixture.js';

const calibration = calibrateBoard(OBLIQUE_CLICKS);
if (!calibration.ok) throw new Error(calibration.reason);
const cal = calibration;
const thumb = /** @param {import('./rectify.js').Image} frame */ (frame) => rectifyZone(frame, cal.modelToImage, BOARD_RECT, 64, 48);
const reference = thumb(renderBoard({ width: 320, height: 180, clicks: OBLIQUE_CLICKS }));

describe('presenceScore', () => {
    it('stays near 1 under an exposure change', () => {
        const darker = renderBoard({ width: 320, height: 180, clicks: OBLIQUE_CLICKS, gain: 0.6, offset: 25 });
        expect(presenceScore(reference, thumb(darker))).toBeGreaterThan(0.98);
    });

    it('stays above the found threshold when the camera drifts a pixel or two', () => {
        /** @type {[number, number][]} */
        const moved = OBLIQUE_CLICKS.map(([x, y]) => [x + 1.5, y - 1]);
        expect(presenceScore(reference, thumb(renderBoard({ width: 320, height: 180, clicks: moved })))).toBeGreaterThan(0.7);
    });

    it('falls below the lost threshold on another shot', () => {
        /** @type {[number, number][]} */
        const zoomed = [
            [300, 175],
            [-200, 190],
            [-120, -150],
            [280, -160]
        ];
        expect(presenceScore(reference, thumb(renderBoard({ width: 320, height: 180, clicks: zoomed })))).toBeLessThan(0.5);
        const rand = lcg(5);
        const noise = { width: 320, height: 180, data: Float32Array.from({ length: 320 * 180 }, () => 255 * rand()) };
        expect(presenceScore(reference, thumb(noise))).toBeLessThan(0.5);
        const black = { width: 320, height: 180, data: new Float32Array(320 * 180) };
        expect(presenceScore(reference, thumb(black))).toBe(0);
    });

    it('scores 0 when too little of the zone was sampled', () => {
        const holes = { ...reference, data: reference.data.map((v, i) => (i % 3 ? NaN : v)) };
        expect(presenceScore(reference, holes)).toBe(0);
        expect(() => presenceScore(reference, { width: 1, height: 1, data: [0] })).toThrow(RangeError);
    });
});

describe('presence on what the game never changes', () => {
    const W = 80;
    const H = 80;
    const mask = presenceMask(W, H);
    const felt = /** @param {object} o */ (o) =>
        rectifyZone(renderBoard({ width: 320, height: 180, clicks: OBLIQUE_CLICKS, points: [100, 130], background: 150, ...o }), cal.modelToImage, PRESENCE_RECT, W, H);
    const start = felt({ checkers: stackCheckers(POSITIONS.start) });

    it('keeps the frame band and the strip between the point tips, never a point, the bar or a tray', () => {
        const sx = (PRESENCE_RECT.x1 - PRESENCE_RECT.x0) / W;
        const sy = (PRESENCE_RECT.y1 - PRESENCE_RECT.y0) / H;
        const at = (/** @type {number} */ u, /** @type {number} */ v) => mask[Math.floor((v - PRESENCE_RECT.y0) / sy) * W + Math.floor((u - PRESENCE_RECT.x0) / sx)];
        expect(at(3.5, -0.3)).toBe(1); // frame, near edge
        expect(at(3.5, 13.3)).toBe(1); // frame, far edge
        expect(at(3.5, 6.5)).toBe(1); // between the tips
        expect(at(3.5, 2)).toBe(0); // a point
        expect(at(6.5, 6.5)).toBe(0); // the bar
        expect(at(-0.5, 3)).toBe(0); // a tray
    });

    it('a game going from the start to a bear-off scores as the start does', () => {
        for (const p of /** @type {const} */ (['middle', 'bearoff'])) {
            expect(presenceScore(start, felt({ checkers: stackCheckers(POSITIONS[p]) }), mask)).toBeGreaterThan(0.95);
        }
    });

    it('a camera moved by a few pixels falls below the lost threshold', () => {
        /** @type {[number, number][]} */
        const moved = OBLIQUE_CLICKS.map(([x, y]) => [x + 10, y - 5]);
        expect(presenceScore(start, felt({ clicks: moved, checkers: stackCheckers(POSITIONS.start) }), mask)).toBeLessThan(0.5);
    });

    it('a band of frame outside the picture is left out on both sides', () => {
        const holes = { ...start, data: start.data.map((v, i) => (i < W ? NaN : v)) };
        expect(presenceScore(holes, holes, mask)).toBeCloseTo(1, 6);
        expect(() => presenceScore(start, start, new Uint8Array(3))).toThrow(RangeError);
    });
});

describe('presence hysteresis', () => {
    it('is lost under 0.5 and found again only above 0.7', () => {
        const scores = [0.9, 0.6, 0.5, 0.49, 0.6, 0.7, 0.71, 0.55, 0.3];
        const states = [];
        let present = true;
        for (const s of scores) states.push((present = nextPresence(present, s)));
        expect(states).toEqual([true, true, true, false, false, false, true, true, false]);
    });

    it('ignores a missing score', () => {
        expect(nextPresence(true, NaN)).toBe(true);
        expect(nextPresence(false, NaN)).toBe(false);
        expect(() => nextPresence(true, 0.6, { lost: 0.7, found: 0.5 })).toThrow(RangeError);
    });

    it('reports transitions from the tracker', () => {
        const t = createPresenceTracker();
        expect(t.present).toBe(true);
        expect(t.update(0.4)).toMatchObject({ present: false, changed: true });
        expect(t.update(0.65)).toMatchObject({ present: false, changed: false });
        expect(t.update(0.8)).toMatchObject({ present: true, changed: true });
    });

    it('loses the view only after `hold` low scores in a row, dated at the first of them', () => {
        const t = createPresenceTracker({ hold: 3 });
        expect(t.update(0.2, 1).present).toBe(true);
        expect(t.update(0.2, 2).present).toBe(true);
        expect(t.update(0.9, 3).present).toBe(true); // the board seen again breaks the run
        t.update(0.2, 4);
        expect(t.update(NaN, 5).present).toBe(true); // no evidence, the run goes on
        t.update(0.2, 6);
        expect(t.update(0.2, 7)).toEqual({ present: false, changed: true, since: 4 });
        expect(t.update(0.8, 8)).toMatchObject({ present: true, changed: true });
        expect(() => createPresenceTracker({ hold: 0 })).toThrow(RangeError);
    });
});
