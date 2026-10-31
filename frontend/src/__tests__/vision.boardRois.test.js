import { describe, expect, it } from 'vitest';
import { boardRois, calibrateBoard, WITNESS_MAX } from '../vision/geometry/index.js';
import { normalizeCalibration } from '../vision/contract.js';

const clicksAt = (/** @type {number} */ s) =>
    /** @type {[number, number][]} */ ([
        [280 * s, 160 * s],
        [40 * s, 160 * s],
        [40 * s, 20 * s],
        [280 * s, 20 * s]
    ]);

describe('boardRois', () => {
    it('gives the two halves by their place in the picture and a witness off the board', () => {
        const b = calibrateBoard(clicksAt(6));
        if (!b.ok) throw new Error(b.reason);
        const res = boardRois({ modelToImage: b.modelToImage, sourceWidth: 1920, sourceHeight: 1080 });
        if (!res.ok) throw new Error(res.reason);
        const { rois, witness } = res.calibration;
        expect(rois.map((r) => r.name)).toEqual(['left', 'right']);
        // Home on the right: the outer board (columns 7–12) is the left ROI, the bar between.
        expect(rois[0]).toEqual({ name: 'left', x: 40, y: 20, w: 111, h: 140 });
        expect(rois[1]).toEqual({ name: 'right', x: 169, y: 20, w: 111, h: 140 });
        expect(witness.w).toBeLessThanOrEqual(WITNESS_MAX.w);
        // Clear of the field and of the trays beside the home board.
        const right = (/** @type {{ x: number, w: number }} */ r) => r.x + r.w;
        expect(witness.y + witness.h <= 20 || witness.y >= 160 || right(witness) <= 40 || witness.x >= 280 + 240 / 13 + 1).toBe(true);
        expect(() => normalizeCalibration(res.calibration)).not.toThrow();
    });

    it('takes a homography in fractions of the picture', () => {
        const b = calibrateBoard(clicksAt(1).map(([x, y]) => [x / 320, y / 180]));
        if (!b.ok) throw new Error(b.reason);
        const res = boardRois({ modelToImage: b.modelToImage, sourceWidth: 1, sourceHeight: 1 });
        expect(res.ok && res.calibration.rois[1]).toEqual({ name: 'right', x: 169, y: 20, w: 111, h: 140 });
    });

    it('refuses a board that fills the frame: no room for the witness', () => {
        const b = calibrateBoard([
            [320, 180],
            [0, 180],
            [0, 0],
            [320, 0]
        ]);
        if (!b.ok) throw new Error(b.reason);
        expect(boardRois({ modelToImage: b.modelToImage, sourceWidth: 320, sourceHeight: 180 })).toEqual({ ok: false, reason: 'no-witness' });
    });
});
