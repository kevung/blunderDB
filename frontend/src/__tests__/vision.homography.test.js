// Guard of ADR-0109 rule 3: the calibration homography is a Hartley-normalised DLT
// and a non-convex quadrilateral of clicks is refused.
import { describe, it, expect } from 'vitest';
import { computeHomography, applyHomography, calibrateBoard, CALIBRATION_CORNERS } from '../vision/geometry/index.js';

describe('vision homography (ADR-0109)', () => {
    it('recovers a camera whose pixel origin lies on the vanishing line, which only a normalised DLT can', () => {
        const src = [
            [900, 900],
            [1100, 920],
            [1080, 1100],
            [920, 1080]
        ];
        const Htrue = [1, 0.1, -800, -0.1, 1, -700, 0.0005, 0.0004, 0];
        const res = computeHomography(
            src,
            src.map((p) => applyHomography(Htrue, p))
        );
        expect(res.ok).toBe(true);
        const [x, y] = applyHomography(res.H, [1000, 1000]);
        const [xt, yt] = applyHomography(Htrue, [1000, 1000]);
        expect(Math.hypot(x - xt, y - yt)).toBeLessThan(1e-6);
    });

    it('calibrates a board from four convex clicks and maps the corners onto them', () => {
        const clicks = [
            [262, 160],
            [52, 166],
            [88, 28],
            [236, 24]
        ];
        const cal = calibrateBoard(clicks);
        expect(cal.ok).toBe(true);
        CALIBRATION_CORNERS.forEach((c, i) => {
            const [x, y] = applyHomography(cal.modelToImage, c);
            expect(x).toBeCloseTo(clicks[i][0], 9);
            expect(y).toBeCloseTo(clicks[i][1], 9);
        });
    });

    it('refuses a non-convex quadrilateral of clicks', () => {
        const dart = [
            [10, 10],
            [100, 10],
            [40, 30],
            [10, 80]
        ];
        const bowTie = [
            [10, 10],
            [95, 80],
            [100, 12],
            [8, 75]
        ];
        expect(calibrateBoard(dart)).toEqual({ ok: false, reason: 'non-convex' });
        expect(calibrateBoard(bowTie)).toEqual({ ok: false, reason: 'non-convex' });
    });
});
