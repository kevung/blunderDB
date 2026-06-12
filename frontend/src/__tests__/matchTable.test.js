import { describe, test, expect } from 'vitest';
import { compareValues, getSortValue, sortMatches, toDateInputValue, formatDate, formatDiceShort } from '../utils/matchTable.js';

describe('compareValues', () => {
    test('nulls sort last regardless of order', () => {
        expect(compareValues(null, null)).toBe(0);
        expect(compareValues(null, 5)).toBe(1);
        expect(compareValues(5, null)).toBe(-1);
        expect(compareValues(undefined, 'a')).toBe(1);
    });

    test('strings compare case-insensitively', () => {
        expect(compareValues('apple', 'Banana')).toBeLessThan(0);
        expect(compareValues('Zebra', 'apple')).toBeGreaterThan(0);
        expect(compareValues('Same', 'same')).toBe(0);
    });

    test('numbers compare numerically', () => {
        expect(compareValues(2, 10)).toBeLessThan(0);
        expect(compareValues(10, 2)).toBeGreaterThan(0);
        expect(compareValues(3, 3)).toBe(0);
    });
});

describe('getSortValue', () => {
    const match = {
        player1_name: 'Alice',
        player2_name: 'Bob',
        match_date: '2026-01-02',
        match_length: 7,
        tournament_name: 'Worlds',
        event: 'EventX',
        pr: 4.2,
        mwc_loss: 1.5
    };

    test.each([
        ['player1', 'Alice'],
        ['player2', 'Bob'],
        ['date', '2026-01-02'],
        ['length', 7],
        ['tournament', 'Worlds'],
        ['pr', 4.2],
        ['mwc', 1.5]
    ])('column %s → %s', (column, expected) => {
        expect(getSortValue(match, column)).toBe(expected);
    });

    test('tournament falls back to event when tournament_name is absent', () => {
        expect(getSortValue({ event: 'EventX' }, 'tournament')).toBe('EventX');
    });

    test('missing fields use safe defaults', () => {
        expect(getSortValue({}, 'player1')).toBe('');
        expect(getSortValue({}, 'length')).toBe(0);
        expect(getSortValue({}, 'pr')).toBe(0);
        expect(getSortValue({}, 'unknown')).toBe('');
    });
});

describe('sortMatches', () => {
    const matches = [
        { id: 1, player1_name: 'Charlie', match_length: 5 },
        { id: 2, player1_name: 'alice', match_length: 11 },
        { id: 3, player1_name: 'Bob', match_length: 7 }
    ];

    test('returns the original array (not a copy) when no column is given', () => {
        expect(sortMatches(matches, '', 'asc')).toBe(matches);
    });

    test('sorts ascending by string column, case-insensitive', () => {
        const ids = sortMatches(matches, 'player1', 'asc').map((m) => m.id);
        expect(ids).toEqual([2, 3, 1]); // alice, Bob, Charlie
    });

    test('sorts descending by numeric column', () => {
        const ids = sortMatches(matches, 'length', 'desc').map((m) => m.id);
        expect(ids).toEqual([2, 3, 1]); // 11, 7, 5
    });

    test('does not mutate the input array', () => {
        const before = matches.map((m) => m.id);
        sortMatches(matches, 'player1', 'asc');
        expect(matches.map((m) => m.id)).toEqual(before);
    });
});

describe('toDateInputValue', () => {
    test('valid date → yyyy-mm-dd', () => {
        expect(toDateInputValue('2026-03-15T10:00:00Z')).toBe('2026-03-15');
    });

    test('empty or invalid → empty string', () => {
        expect(toDateInputValue('')).toBe('');
        expect(toDateInputValue(null)).toBe('');
        expect(toDateInputValue('not-a-date')).toBe('');
    });
});

describe('formatDate', () => {
    test('valid date → yyyy/mm/dd with zero-padding', () => {
        // No trailing Z → parsed as local time, so the local getDate() is
        // timezone-independent for this assertion.
        expect(formatDate('2026-03-05T12:00:00')).toBe('2026/03/05');
    });

    test('empty or invalid → dash', () => {
        expect(formatDate('')).toBe('-');
        expect(formatDate(null)).toBe('-');
        expect(formatDate('garbage')).toBe('-');
    });
});

describe('formatDiceShort', () => {
    test('joins the two dice', () => {
        expect(formatDiceShort([3, 1])).toBe('31');
        expect(formatDiceShort([6, 6])).toBe('66');
    });

    test('empty when there are no dice', () => {
        expect(formatDiceShort(null)).toBe('');
        expect(formatDiceShort([0, 0])).toBe('');
        expect(formatDiceShort(undefined)).toBe('');
    });
});
