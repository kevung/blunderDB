import { describe, test, expect } from 'vitest';
import { must } from './helpers/must.js';
import { neilNumber, neilExact, neilEstimate, formatNeilNumber, metEquity } from '../services/neilNumbers.js';

describe('neilNumber', () => {
    test.each([
        [3, 10],
        [4, 9],
        [5, 8],
        [6, 7],
        [8, 6],
        [11, 5],
        [15, 4],
        [19, 3.5],
        [25, 3]
    ])('anchor %i → %d', (n, value) => expect(neilNumber(n)).toBe(value));

    test.each([
        [7, 6.5],
        [9, 6 - 1 / 3],
        [10, 5 + 1 / 3],
        [12, 4.75],
        [17, 3.75],
        [22, 3.25]
    ])('between two anchors, linear: %i → %d', (n, value) => expect(neilNumber(n)).toBeCloseTo(value, 10));

    test('outside 3..25 there is no number', () => {
        expect(neilNumber(2)).toBeNull();
        expect(neilNumber(26)).toBeNull();
    });
});

describe('formatNeilNumber', () => {
    test('every n from 3 to 25, exact whole + fraction', () => {
        /** @type {Record<number, string>} */
        const expected = {
            3: '10',
            4: '9',
            5: '8',
            6: '7',
            7: '6½',
            8: '6',
            9: '5⅔',
            10: '5⅓',
            11: '5',
            12: '4¾',
            13: '4½',
            14: '4¼',
            15: '4',
            16: '3⅞',
            17: '3¾',
            18: '3⅝',
            19: '3½',
            20: '3 5⁄12',
            21: '3⅓',
            22: '3¼',
            23: '3⅙',
            24: '3 1⁄12',
            25: '3'
        };
        for (let n = 3; n <= 25; n++) expect(formatNeilNumber(n)).toBe(expected[n]);
    });

    test('outside the table it reads empty', () => {
        expect(formatNeilNumber(2)).toBe('');
        expect(formatNeilNumber(26)).toBe('');
    });

    test('exact values agree with the float interpolation, anchors flagged', () => {
        for (let n = 3; n <= 25; n++) {
            const e = must(neilExact(n));
            expect(e.whole + e.num / e.den).toBeCloseTo(must(neilNumber(n)), 10);
            expect(e.anchor).toBe([3, 4, 5, 6, 8, 11, 15, 19, 25].includes(n));
        }
    });
});

describe('neilEstimate', () => {
    test('E = 50 + lead × Neil(trailer away), gap against the Kazaross-XG2 MET', () => {
        const cell = must(neilEstimate(4, 9));
        expect(cell.fallback).toBe(false);
        expect(cell.estimate).toBeCloseTo(50 + 5 * (6 - 1 / 3), 10);
        expect(cell.met).toBe(metEquity(4, 9));
        expect(cell.gap).toBeCloseTo(cell.estimate - must(metEquity(4, 9)), 10);
    });

    test('a leader at 1 or 2 away gets the MET, not the shortcut', () => {
        for (const leader of [1, 2]) {
            const cell = neilEstimate(leader, 6);
            expect(cell).toEqual({ estimate: metEquity(leader, 6), met: metEquity(leader, 6), gap: null, fallback: true });
        }
    });

    test('no estimate when the leader does not lead', () => {
        expect(neilEstimate(5, 5)).toBeNull();
        expect(neilEstimate(7, 4)).toBeNull();
    });

    test('the estimate never passes 100 %', () => {
        expect(must(neilEstimate(3, 25)).estimate).toBeLessThanOrEqual(100);
    });
});
