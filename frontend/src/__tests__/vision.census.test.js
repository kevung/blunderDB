import { describe, expect, it } from 'vitest';
import { censusDistance, censusRoi, toGray } from '../vision/census.js';
import { createDescriber, createDetector, replay } from '../vision/descriptors.js';
import { CALIBRATION, H, makeFrame, scene, texture, W } from './fixtures/vision/frames.js';

const ROI = CALIBRATION.rois[0];
const census = (/** @type {Uint8ClampedArray} */ data) => censusRoi(toGray(data, W, H), W, ROI);

describe('census descriptor', () => {
    it('is invariant to a global gain (auto-exposure)', () => {
        const a = census(makeFrame((x, y) => texture(x, y)));
        // Gain below clipping; rounding to 8 bits may flip a bit sitting on the tau margin.
        const b = census(makeFrame((x, y) => texture(x, y), 1.2));
        expect(censusDistance(a.bits, b.bits)).toBeLessThan(0.002);
        expect(b.mean).toBeGreaterThan(a.mean);
    });

    it('ignores sensor grain on a flat field below tau', () => {
        const a = census(makeFrame(() => 120));
        const b = census(makeFrame((x, y) => 120 + ((x + y) % 2)));
        expect(censusDistance(a.bits, b.bits)).toBe(0);
    });

    it('sees a change of structure inside the ROI', () => {
        const a = census(scene());
        const b = census(scene([ROI]));
        expect(censusDistance(a.bits, b.bits)).toBeGreaterThan(0.2);
        expect(census(scene()).bits).toEqual(a.bits);
    });

    it('measures the distance as a fraction of bits', () => {
        expect(censusDistance(new Uint8Array([0, 0]), new Uint8Array([255, 0]))).toBe(0.5);
        expect(() => censusDistance(new Uint8Array(1), new Uint8Array(2))).toThrow(RangeError);
    });
});

describe('describer and detector', () => {
    it('records mean, distance to previous and to background per ROI', () => {
        const d = createDescriber(CALIBRATION);
        const r0 = d.describe({ mediaTime: 0, data: scene() });
        expect(r0).toEqual({ t: 0, u: 0, w: null, r: { diceLeft: [expect.any(Number), null, null], diceRight: [expect.any(Number), null, null] } });
        const r1 = d.describe({ mediaTime: 0.5, data: scene([ROI]) });
        expect(r1.u).toBe(0);
        expect(r1.w).toBe(0);
        expect(r1.r.diceLeft[1]).toBeGreaterThan(0.2);
        expect(r1.r.diceLeft[2]).toBe(r1.r.diceLeft[1]);
        expect(r1.r.diceRight[1]).toBe(0);
        const r2 = d.describe({ mediaTime: 1, data: scene([ROI]) });
        expect(r2.r.diceLeft[1]).toBe(0); // still against the previous frame
        expect(r2.r.diceLeft[2]).toBeGreaterThan(0.2); // still away from the background
        expect(JSON.parse(JSON.stringify(r2))).toEqual(r2);
    });

    it('marks the frame uncertain and resets backgrounds when the witness jumps', () => {
        const d = createDescriber(CALIBRATION);
        d.describe({ mediaTime: 0, data: scene() });
        const cut = d.describe({ mediaTime: 0.5, data: scene([CALIBRATION.witness, ROI], 3) });
        expect(cut.u).toBe(1);
        expect(cut.w).toBeGreaterThan(0.6);
        expect(cut.r.diceLeft[2]).toBeNull(); // the background is this frame from now on
        const after = d.describe({ mediaTime: 1, data: scene([CALIBRATION.witness, ROI], 3) });
        expect(after.u).toBe(0);
        expect(after.r.diceLeft[2]).toBe(0);
    });

    it('emits one frame.uncertain per episode, at its first frame', () => {
        const records = [0, 1, 1, 0, 1].map((u, i) => ({ t: i / 2, u, w: u ? 0.8 : 0.01, r: {} }));
        const events = replay(/** @type {any} */ (records), createDetector());
        expect(events.map((e) => [e.type, e.mediaTime, e.state])).toEqual([
            ['frame.uncertain', 0.5, 'uncertain'],
            ['frame.uncertain', 2, 'uncertain']
        ]);
        expect(events[0].payload).toEqual({ witness: 0.8 });
    });

    it('refuses a frame of another size than the calibration', () => {
        const d = createDescriber(CALIBRATION);
        expect(() => d.describe({ mediaTime: 0, width: 160, height: 90, data: new Uint8ClampedArray(160 * 90 * 4) })).toThrow(RangeError);
    });
});
