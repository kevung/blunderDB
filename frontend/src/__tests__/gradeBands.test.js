/**
 * gradeBands.test.js
 *
 * These tests used to live inside `StatsProgressionTab.test.js` (D.13,
 * #214) even though they never touch that component — they exercise
 * `components/stats/gradeBands.js` directly, an independent module the
 * component only imports. Split out under its own honest name; see
 * `StatsProgressionTab.render.test.js` for the component itself.
 */

import { describe, test, expect, vi } from 'vitest';
import { GRADE_BANDS, gradeForPR, makeGradeBandPlugin } from '../components/stats/gradeBands.js';

describe('GRADE_BANDS', () => {
    test('has 6 bands', () => {
        expect(GRADE_BANDS).toHaveLength(6);
    });

    test('first band starts at 0', () => {
        expect(GRADE_BANDS[0].min).toBe(0);
    });

    test('last band ends at Infinity', () => {
        expect(GRADE_BANDS[GRADE_BANDS.length - 1].max).toBe(Infinity);
    });

    test('bands are contiguous (each min === previous max)', () => {
        for (let i = 1; i < GRADE_BANDS.length; i++) {
            expect(GRADE_BANDS[i].min).toBe(GRADE_BANDS[i - 1].max);
        }
    });

    test('each band has a color string', () => {
        for (const band of GRADE_BANDS) {
            expect(typeof band.color).toBe('string');
            expect(band.color.length).toBeGreaterThan(0);
        }
    });
});

describe('gradeForPR', () => {
    test('PR 0 → World Class', () => {
        expect(gradeForPR(0)).toBe('World Class');
    });

    test('PR 1.9 → World Class', () => {
        expect(gradeForPR(1.9)).toBe('World Class');
    });

    test('PR 2.0 → Expert', () => {
        expect(gradeForPR(2.0)).toBe('Expert');
    });

    test('PR 3.5 → Expert', () => {
        expect(gradeForPR(3.5)).toBe('Expert');
    });

    test('PR 4.0 → Advanced', () => {
        expect(gradeForPR(4.0)).toBe('Advanced');
    });

    test('PR 6.0 → Intermediate', () => {
        expect(gradeForPR(6.0)).toBe('Intermediate');
    });

    test('PR 9.0 → Casual', () => {
        expect(gradeForPR(9.0)).toBe('Casual');
    });

    test('PR 12.0 → Beginner', () => {
        expect(gradeForPR(12.0)).toBe('Beginner');
    });

    test('PR 20 → Beginner', () => {
        expect(gradeForPR(20)).toBe('Beginner');
    });
});

describe('makeGradeBandPlugin', () => {
    test('returns a plugin object with id and beforeDraw', () => {
        const plugin = makeGradeBandPlugin(GRADE_BANDS);
        expect(plugin.id).toBe('gradeBands');
        expect(typeof plugin.beforeDraw).toBe('function');
    });

    test('beforeDraw does not throw with a mock chart', () => {
        const plugin = makeGradeBandPlugin(GRADE_BANDS);
        const mockChart = {
            ctx: {
                save: vi.fn(),
                restore: vi.fn(),
                beginPath: vi.fn(),
                rect: vi.fn(),
                clip: vi.fn(),
                fillRect: vi.fn(),
                fillStyle: ''
            },
            chartArea: { top: 0, bottom: 300, left: 0, right: 400 },
            scales: {
                y: { getPixelForValue: (v) => 300 - v * 10 }
            }
        };
        expect(() => plugin.beforeDraw(mockChart)).not.toThrow();
    });

    test('beforeDraw skips rendering when y scale is missing', () => {
        const plugin = makeGradeBandPlugin(GRADE_BANDS);
        const ctx = { save: vi.fn(), restore: vi.fn(), fillRect: vi.fn() };
        plugin.beforeDraw({ ctx, chartArea: {}, scales: {} });
        expect(ctx.fillRect).not.toHaveBeenCalled();
    });
});
