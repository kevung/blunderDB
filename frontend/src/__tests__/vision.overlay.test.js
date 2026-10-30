import { describe, expect, it } from 'vitest';
import { dHashRgba, DRIFT_THRESHOLD, HOLD_SECONDS, createOverlayWatch, dHash, hammingDistance } from '../vision/overlay.js';

const W = 90;
const H = 80;
const ROI = { x: 0, y: 0, w: W, h: H };

/** @param {(x: number, y: number) => number} f */
const img = (f) => {
    const g = new Uint8Array(W * H);
    for (let y = 0; y < H; y++) for (let x = 0; x < W; x++) g[y * W + x] = f(x, y);
    return g;
};
const ramp = (x) => 20 + Math.floor((x * 200) / W);
const hashOf = (f) => dHash(img(f), W, ROI);

describe('overlay dHash', () => {
    it('is 64 bits and deterministic', () => {
        const h = hashOf(ramp);
        expect(h).toHaveLength(8);
        expect(hashOf(ramp)).toEqual(h);
    });
    it('is invariant to a gain', () => {
        expect(
            hammingDistance(
                hashOf(ramp),
                hashOf((x, y) => Math.floor(ramp(x, y) * 0.6))
            )
        ).toBe(0);
    });
    it('hamming distance counts differing bits', () => {
        expect(hammingDistance(new Uint8Array([0b1011, 0]), new Uint8Array([0, 0xff]))).toBe(3 + 8);
        expect(() => hammingDistance(new Uint8Array(8), new Uint8Array(7))).toThrow(RangeError);
    });
    it('a reversed ramp flips every bit', () => {
        expect(
            hammingDistance(
                hashOf(ramp),
                hashOf((x) => ramp(W - 1 - x))
            )
        ).toBe(64);
    });
    it('rejects a ROI outside the frame or too small', () => {
        expect(() => dHash(img(ramp), W, { x: 80, y: 0, w: 20, h: 20 })).toThrow(RangeError);
        expect(() => dHash(img(ramp), W, { x: 0, y: 0, w: 4, h: 4 })).toThrow(RangeError);
    });
});

describe('overlay dHash geometry', () => {
    // Deterministic texture: every cell differs, so any misplaced offset changes the hash.
    const noise = (x, y) => (x * 73 + y * 151 + ((x * y) % 7) * 37) % 256;
    const FW = 200;
    const FH = 120;
    const frame = new Uint8Array(FW * FH);
    for (let y = 0; y < FH; y++) for (let x = 0; x < FW; x++) frame[y * FW + x] = noise(x, y);
    const crop = (/** @type {number} */ x0, /** @type {number} */ y0, /** @type {number} */ w, /** @type {number} */ h) => {
        const c = new Uint8Array(w * h);
        for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) c[y * w + x] = frame[(y0 + y) * FW + x0 + x];
        return c;
    };
    it.each([
        [0, 0, 90, 80],
        [37, 23, 90, 80],
        [51, 5, 47, 61], // not divisible by 9×8
        [100, 60, 100, 60]
    ])('a ROI at (%i,%i) %i×%i hashes like its crop', (x, y, w, h) => {
        expect(dHash(frame, FW, { x, y, w, h })).toEqual(dHash(crop(x, y, w, h), w, { x: 0, y: 0, w, h }));
    });
    it('depends on the offset and on the row order', () => {
        const a = dHash(frame, FW, { x: 37, y: 23, w: 90, h: 80 });
        expect(a).not.toEqual(dHash(frame, FW, { x: 38, y: 23, w: 90, h: 80 }));
        expect(a).not.toEqual(dHash(frame, FW, { x: 37, y: 24, w: 90, h: 80 }));
        expect(new Set(a).size).toBeGreaterThan(1);
    });
    it('averages the whole cell, not one sample', () => {
        // 20×17: cell 0 spans columns 0-1 (100, 0 → mean 50), cell 1 columns 2-3 (60, 60).
        const w = 20;
        const h = 17;
        const g = new Uint8Array(w * h).fill(60);
        for (let y = 0; y < h; y++) g[y * w] = 100;
        for (let y = 0; y < h; y++) g[y * w + 1] = 0;
        expect(dHash(g, w, { x: 0, y: 0, w, h })[0] & 1).toBe(0);
        g.fill(60);
        for (let y = 0; y < h; y++) g[y * w] = 100; // cell 0 mean 80 > 60
        expect(dHash(g, w, { x: 0, y: 0, w, h })[0] & 1).toBe(1);
    });
    it('treats unequal cell widths by mean, not by sum', () => {
        expect([...dHash(new Uint8Array(20 * 17).fill(100), 20, { x: 0, y: 0, w: 20, h: 17 })]).toEqual(new Array(8).fill(0));
    });
    it('puts the top rows first', () => {
        const g = new Uint8Array(W * H);
        for (let y = 0; y < H; y++) for (let x = 0; x < W; x++) g[y * W + x] = y < H / 2 ? 255 - x : x;
        expect([...dHash(g, W, ROI)]).toEqual([255, 255, 255, 255, 0, 0, 0, 0]);
    });
    it('rejects a ROI exceeding the bottom of the frame', () => {
        expect(() => dHash(new Uint8Array(W * 20), W, { x: 0, y: 10, w: 20, h: 12 })).toThrow(RangeError);
        expect(() => dHash(new Uint8Array(W * 20), W, { x: 0, y: -1, w: 20, h: 12 })).toThrow(RangeError);
    });
    it('hashes RGBA like its gray conversion', () => {
        const rgba = new Uint8ClampedArray(W * H * 4);
        for (let y = 0; y < H; y++) for (let x = 0; x < W; x++) rgba.fill(ramp(W - 1 - x), (y * W + x) * 4, (y * W + x) * 4 + 3);
        expect(
            hammingDistance(
                dHashRgba(rgba, W, H, ROI),
                hashOf((x) => ramp(W - 1 - x))
            )
        ).toBeLessThanOrEqual(2);
        expect(dHashRgba(rgba, W, H, ROI)[0]).toBe(255);
        expect(dHashRgba(rgba, W, H, ROI)).toHaveLength(8);
    });
});

describe('overlay watch', () => {
    const a = hashOf(ramp);
    const b = hashOf((x) => ramp(W - 1 - x));

    it('holds its thresholds', () => {
        // Valeurs provisoires, à régler sur les journaux V1.1 : ce test force à les rouvrir exprès.
        expect(DRIFT_THRESHOLD).toBe(10);
        expect(HOLD_SECONDS).toBe(1);
    });
    it('is silent on a stable overlay and the first sample', () => {
        const w = createOverlayWatch();
        for (let t = 0; t < 5; t += 0.5) expect(w.push(t, a)).toBeNull();
    });
    it('ignores a change that does not last', () => {
        const w = createOverlayWatch();
        w.push(0, a);
        expect(w.push(1, b)).toBeNull();
        expect(w.push(1.5, a)).toBeNull();
        expect(w.push(3, a)).toBeNull();
    });
    it('reports a held change once, with its distance', () => {
        const w = createOverlayWatch();
        w.push(0, a);
        expect(w.push(1, b)).toBeNull();
        expect(w.push(2, b)).toBeNull(); // exactly HOLD_SECONDS: not yet
        const ev = w.push(2.5, b);
        expect(ev).toMatchObject({ type: 'overlay.changed', mediaTime: 1, payload: { distance: 64 } });
        // One event per change, even long after: the new look is the reference.
        for (let t = 3; t < 10; t++) expect(w.push(t, b)).toBeNull();
    });
    it('forgets a candidate that fell back under the threshold', () => {
        const w = createOverlayWatch();
        w.push(0, a);
        w.push(1, b);
        w.push(1.5, a); // back: candidate forgotten
        expect(w.push(2.5, b)).toBeNull(); // new candidate from 2.5, not 1
        expect(w.push(3.4, b)).toBeNull();
        expect(w.push(3.6, b)).not.toBeNull();
    });
    it('restarts the hold while the overlay is still moving', () => {
        const w = createOverlayWatch();
        const c = hashOf((x) => (x < W / 2 ? 20 + x * 2 : 220 - x));
        expect(hammingDistance(b, c)).toBeGreaterThan(DRIFT_THRESHOLD);
        w.push(0, a);
        w.push(1, b);
        expect(w.push(1.8, c)).toBeNull(); // different candidate: restart at 1.8
        expect(w.push(2.5, c)).toBeNull(); // would fire if still counted from 1
        const ev = w.push(3, c);
        expect(ev).toMatchObject({ mediaTime: 1.8 });
    });
    it('a distance at the threshold is not a change', () => {
        const near = a.slice();
        near[0] ^= 0xff;
        near[1] ^= 0b11; // 10 bits
        expect(hammingDistance(a, near)).toBe(DRIFT_THRESHOLD);
        const w = createOverlayWatch();
        w.push(0, a);
        for (let t = 1; t < 6; t++) expect(w.push(t, near)).toBeNull();
    });
});
