import { describe, test, expect } from 'vitest';
import { VIDEO_RATES, stepRate, rateKeyDirection } from '../utils/videoRate.js';

describe('videoRate', () => {
    test('sixteen quarter steps from 0.25× to 4×', () => {
        expect(VIDEO_RATES).toHaveLength(16);
        expect(VIDEO_RATES[0]).toBe(0.25);
        expect(VIDEO_RATES.at(-1)).toBe(4);
        expect(VIDEO_RATES).toContain(1);
        expect(VIDEO_RATES).toContain(3.75);
    });

    test('one step each way, and the ends hold rather than wrap', () => {
        expect(stepRate(1, 1)).toBe(1.25);
        expect(stepRate(1, -1)).toBe(0.75);
        expect(stepRate(4, 1)).toBe(4);
        expect(stepRate(0.25, -1)).toBe(0.25);
    });

    test('only the speeds the source accepts', () => {
        const youtube = [0.25, 0.5, 0.75, 1, 1.25, 1.5, 1.75, 2];
        expect(stepRate(2, 1, youtube)).toBe(2);
        expect(stepRate(1.75, 1, youtube)).toBe(2);
        expect(stepRate(1, 1, [0.5, 1, 2])).toBe(2);
    });

    test('[ slows, ] speeds up, typed with AltGr as well', () => {
        const key = (/** @type {string} */ k, extra = {}) => /** @type {KeyboardEvent} */ (/** @type {any} */ ({ key: k, metaKey: false, ...extra }));
        expect(rateKeyDirection(key('['))).toBe(-1);
        expect(rateKeyDirection(key(']'))).toBe(1);
        expect(rateKeyDirection(key(']', { altKey: true }))).toBe(1);
        expect(rateKeyDirection(key('[', { altKey: true, ctrlKey: true }))).toBe(-1);
        expect(rateKeyDirection(key('[', { metaKey: true }))).toBe(0);
        expect(rateKeyDirection(key('v'))).toBe(0);
    });
});
