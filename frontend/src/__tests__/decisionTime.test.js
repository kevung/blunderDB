import { describe, test, expect } from 'vitest';
import { fmtDuration, fmtMean, moveTotalMS, sortByDuration, timeBars } from '../utils/decisionTime.js';

describe('decision times', () => {
    test('an unknown duration is empty, never zero', () => {
        expect(fmtDuration(null)).toBe('');
        expect(fmtDuration(undefined)).toBe('');
        expect(fmtDuration(0)).toBe('0.0 s');
        expect(fmtMean(0, 0)).toBe('');
        expect(moveTotalMS({})).toBeNull();
        expect(moveTotalMS({ cube_decision_ms: 2000 })).toBe(2000);
    });

    test('formats seconds and minutes', () => {
        expect(fmtDuration(5200)).toBe('5.2 s');
        expect(fmtDuration(65000)).toBe('1:05');
    });

    test('sorts with the unknown last in both directions', () => {
        const moves = [{ mp: {} }, { mp: { decision_ms: 3000 } }, { mp: { decision_ms: 9000 } }];
        expect(sortByDuration(moves, 'desc').map((m) => m.mp.decision_ms ?? null)).toEqual([9000, 3000, null]);
        expect(sortByDuration(moves, 'asc').map((m) => m.mp.decision_ms ?? null)).toEqual([3000, 9000, null]);
        expect(sortByDuration(moves, '')).toEqual(moves);
    });

    test('the chart has a bar only for a known time', () => {
        expect(timeBars([{ player_on_roll: 0, decision_ms: 1000 }, { player_on_roll: 1 }, { player_on_roll: 1, decision_ms: 2000 }])).toEqual([
            { index: 0, player: 0, ms: 1000 },
            { index: 2, player: 1, ms: 2000 }
        ]);
    });
});
