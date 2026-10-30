import { describe, it, expect } from 'vitest';
import { presenceScore, nextPresence, createPresenceTracker } from './presence.js';
import { rectifyZone } from './rectify.js';
import { calibrateBoard } from './calibration.js';
import { BOARD_RECT } from './boardModel.js';
import { renderBoard, OBLIQUE_CLICKS, lcg } from './synthetic.fixture.js';

const cal = calibrateBoard(OBLIQUE_CLICKS);
const thumb = (frame) => rectifyZone(frame, cal.modelToImage, BOARD_RECT, 64, 48);
const reference = thumb(renderBoard({ width: 320, height: 180, clicks: OBLIQUE_CLICKS }));

describe('presenceScore', () => {
    it('stays near 1 under an exposure change', () => {
        const darker = renderBoard({ width: 320, height: 180, clicks: OBLIQUE_CLICKS, gain: 0.6, offset: 25 });
        expect(presenceScore(reference, thumb(darker))).toBeGreaterThan(0.98);
    });

    it('stays above the found threshold when the camera drifts a pixel or two', () => {
        const moved = OBLIQUE_CLICKS.map(([x, y]) => [x + 1.5, y - 1]);
        expect(presenceScore(reference, thumb(renderBoard({ width: 320, height: 180, clicks: moved })))).toBeGreaterThan(0.7);
    });

    it('falls below the lost threshold on another shot', () => {
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
        expect(t.update(0.4)).toEqual({ present: false, changed: true });
        expect(t.update(0.65)).toEqual({ present: false, changed: false });
        expect(t.update(0.8)).toEqual({ present: true, changed: true });
    });
});
