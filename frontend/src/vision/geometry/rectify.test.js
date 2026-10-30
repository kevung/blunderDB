import { describe, it, expect } from 'vitest';
import { rectifyZone, sampleImage } from './rectify.js';
import { applyHomography } from './homography.js';
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
            expect(Math.abs(mean(img) - expected), zone.id).toBeLessThan(1e-6);
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
        expect(agree / (w * h)).toBeGreaterThan(0.91);
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

describe('sampling convention', () => {
    // Bilinear interpolation reproduces an affine field exactly, so any shift of the
    // half-pixel convention or of the sample point shows up as a plain number.
    const W = 40;
    const H = 30;
    const f = (x, y) => 3 * x + 5 * y + 7;
    const ramp = { width: W, height: H, data: Float64Array.from({ length: W * H }, (_, k) => f((k % W) + 0.5, Math.floor(k / W) + 0.5)) };

    it('puts pixel centres at integer + 0.5 and interpolates exactly between them', () => {
        for (const [x, y] of [
            [0.5, 0.5],
            [1.25, 7.8],
            [20.5, 3.1],
            [17.9, 28.6],
            [39.5, 29.5]
        ]) {
            expect(Math.abs(sampleImage(ramp, x, y) - f(x, y))).toBeLessThan(1e-9);
        }
    });

    it('clamps the half-pixel border to the outermost centres', () => {
        expect(Math.abs(sampleImage(ramp, 0.2, 10.3) - f(0.5, 10.3))).toBeLessThan(1e-9);
        expect(Math.abs(sampleImage(ramp, 12.7, 29.9) - f(12.7, 29.5))).toBeLessThan(1e-9);
    });

    it('samples each output pixel at its centre through the homography', () => {
        const Hm = [2.1, 0.3, 4, -0.2, 1.8, 3, 0.004, -0.003, 1];
        const rect = { x0: 1, y0: 2, x1: 13, y1: 11 };
        const w = 24;
        const h = 18;
        const img = rectifyZone(ramp, Hm, rect, w, h);
        expect(img.coverage).toBe(1);
        for (let j = 0; j < h; j++) {
            for (let i = 0; i < w; i++) {
                const u = rect.x0 + ((i + 0.5) * (rect.x1 - rect.x0)) / w;
                const v = rect.y0 + ((j + 0.5) * (rect.y1 - rect.y0)) / h;
                const [x, y] = applyHomography(Hm, [u, v]);
                expect(Math.abs(img.data[j * w + i] - f(x, y))).toBeLessThan(1e-3);
            }
        }
    });
});
