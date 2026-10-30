import { describe, it, expect } from 'vitest';
import { rectifyZone, sampleImage } from './rectify.js';
import { calibrateBoard } from './calibration.js';
import { pointZone, barZone, trayZone, FIELD_RECT } from './boardModel.js';
import { renderBoard, modelLuma, OBLIQUE_CLICKS, DARK, LIGHT, BAR, TRAY } from './synthetic.fixture.js';

const mean = (img) => {
    let s = 0;
    let n = 0;
    for (const v of img.data) if (!Number.isNaN(v)) ((s += v), n++);
    return s / n;
};

describe('rectifyZone', () => {
    const frame = renderBoard({ width: 320, height: 180, clicks: OBLIQUE_CLICKS, checkers: false });
    const cal = calibrateBoard(OBLIQUE_CLICKS);

    it('calibrates from the four clicks', () => {
        expect(cal.ok).toBe(true);
    });

    it('reads each zone back with its own shade through an oblique, mirrored view', () => {
        const shrink = (r) => ({ x0: r.x0 + 0.2, x1: r.x1 - 0.2, y0: r.y0 + 0.2, y1: r.y1 - 0.2 });
        for (const [zone, expected] of [
            [pointZone(1), DARK],
            [pointZone(2), LIGHT],
            [pointZone(12), LIGHT],
            [pointZone(13), DARK],
            [pointZone(24), LIGHT],
            [barZone('near'), BAR],
            [trayZone('far'), TRAY]
        ]) {
            const img = rectifyZone(frame, cal.modelToImage, shrink(zone.rect), 6, 30);
            expect(img.coverage).toBe(1);
            expect(Math.abs(mean(img) - expected), zone.id).toBeLessThan(3);
        }
    });

    it('matches the model pattern pixel by pixel away from edges', () => {
        const w = 130;
        const h = 130;
        const img = rectifyZone(frame, cal.modelToImage, FIELD_RECT, w, h);
        let agree = 0;
        for (let j = 0; j < h; j++) {
            for (let i = 0; i < w; i++) {
                const truth = modelLuma((i + 0.5) / 10, (j + 0.5) / 10, { checkers: false });
                if (Math.abs(img.data[j * w + i] - truth) < 2) agree++;
            }
        }
        // Only pixels whose bilinear footprint straddles a zone border may differ.
        expect(agree / (w * h)).toBeGreaterThan(0.85);
    });

    it('marks pixels outside the image as NaN and reports the coverage', () => {
        const img = rectifyZone(frame, cal.modelToImage, { x0: -40, y0: 0, x1: 13, y1: 13 }, 53, 13);
        expect(img.coverage).toBeGreaterThan(0.1);
        expect(img.coverage).toBeLessThan(0.9);
        expect(Number.isNaN(img.data[0])).toBe(true);
    });

    it('reads RGBA and single-channel images alike', () => {
        const rgba = renderBoard({ width: 320, height: 180, clicks: OBLIQUE_CLICKS, checkers: false, rgba: true });
        const a = rectifyZone(rgba, cal.modelToImage, FIELD_RECT, 20, 20);
        const b = rectifyZone(frame, cal.modelToImage, FIELD_RECT, 20, 20);
        a.data.forEach((v, i) => expect(Math.abs(v - b.data[i])).toBeLessThan(1));
        expect(sampleImage(frame, -1, 5)).toBeNaN();
    });

    it('refuses an empty zone or output', () => {
        expect(() => rectifyZone(frame, cal.modelToImage, { x0: 1, x1: 1, y0: 0, y1: 1 }, 4, 4)).toThrow(RangeError);
        expect(() => rectifyZone(frame, cal.modelToImage, FIELD_RECT, 0, 4)).toThrow(RangeError);
    });
});
