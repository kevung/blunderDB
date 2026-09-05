import { describe, test, expect } from 'vitest';
import { remainingSeconds } from '../stores/bearoffStore.js';

// The remaining time shown during a generation is MEASURED, not a second
// estimate: the whole point of the Bearoff tab's progress line is that the
// figure stops being about the developer's machine once the run has started.
describe('remainingSeconds', () => {
    test('says nothing before there is anything to measure', () => {
        expect(remainingSeconds(null)).toBe(null);
        expect(remainingSeconds({ done: 0, total: 100, startedAt: 1000, firstDone: 0 }, 2000)).toBe(null);
        expect(remainingSeconds({ done: 10, total: 0, startedAt: 1000, firstDone: 0 }, 2000)).toBe(null);
    });

    test('extrapolates from the work done since the first report', () => {
        // A quarter done in 10 s: three quarters left, so 30 s.
        const progress = { done: 250, total: 1000, startedAt: 0, firstDone: 0 };
        expect(remainingSeconds(progress, 10_000)).toBeCloseTo(30, 5);
    });

    test('ignores the set-up that happened before the first report', () => {
        // The successor lists are built before any progress arrives: counting
        // that time against the pairs done since would inflate every figure.
        const progress = { done: 600, total: 1000, startedAt: 0, firstDone: 100 };
        // 500 pairs in 10 s → 400 left → 8 s.
        expect(remainingSeconds(progress, 10_000)).toBeCloseTo(8, 5);
    });

    test('counts down between two reports', () => {
        const progress = { done: 500, total: 1000, startedAt: 0, firstDone: 0 };
        const early = remainingSeconds(progress, 10_000);
        const later = remainingSeconds(progress, 20_000);
        expect(later).toBeGreaterThan(early);
    });
});
