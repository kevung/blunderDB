import { describe, test, expect } from 'vitest';
import { compareValues, getSortValue, sortMatches, toDateInputValue, formatDate, formatDiceShort, fmtPR, fmtEquityError, fmtMwcLoss, fmtErrorsBlunders, MATCH_STAT_ROWS } from '../utils/matchTable.js';

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

describe('match stats cell formatters', () => {
    test('fmtPR shows 2 decimals when there are decisions, em-dash otherwise', () => {
        expect(fmtPR(5.456, 10)).toBe('5.46');
        expect(fmtPR(0, 0)).toBe('—');
        expect(fmtPR(3.1, 0)).toBe('—'); // no decisions → N/A regardless of value
    });

    test('fmtEquityError prefixes a minus and 3 decimals, em-dash at zero', () => {
        expect(fmtEquityError(0.1234)).toBe('-0.123');
        expect(fmtEquityError(0)).toBe('—');
    });

    test('fmtMwcLoss renders a negative percentage with 2 decimals, em-dash at zero', () => {
        expect(fmtMwcLoss(0.0512)).toBe('-5.12%');
        expect(fmtMwcLoss(0)).toBe('—');
    });

    test('fmtErrorsBlunders is "errors (blunders)"', () => {
        expect(fmtErrorsBlunders(7, 2)).toBe('7 (2)');
    });
});

describe('MATCH_STAT_ROWS', () => {
    const sample = {
        pr: 4.5,
        pr_checker: 3.2,
        pr_cube: 6.1,
        total_decisions: 100,
        checker_decisions: 80,
        double_decisions: 12,
        take_decisions: 8,
        total_errors: 20,
        total_blunders: 5,
        total_equity_error: 1.234,
        mwc_loss: 0.042,
        checker_errors: 12,
        checker_blunders: 3,
        checker_equity_error: 0.8,
        checker_mwc_loss: 0.03,
        double_errors: 4,
        double_blunders: 1,
        double_equity_error: 0.3,
        double_mwc_loss: 0.01,
        take_errors: 4,
        take_blunders: 1,
        take_equity_error: 0.13,
        take_mwc_loss: 0.011
    };

    test('every metric row has a label + fmt; sections have just a section key', () => {
        for (const row of MATCH_STAT_ROWS) {
            if (row.section) {
                expect(typeof row.section).toBe('string');
            } else {
                expect(typeof row.label).toBe('string');
                expect(typeof row.fmt).toBe('function');
                expect(typeof row.fmt(sample)).toBe('string');
            }
        }
    });

    test('the four section headers are present in order', () => {
        const sections = MATCH_STAT_ROWS.filter((r) => r.section).map((r) => r.section);
        expect(sections).toEqual(['match.performanceRating', 'match.totalErrors', 'match.checkerPlay', 'match.cubePlay']);
    });

    test('overall PR row formats the sample correctly', () => {
        const overall = MATCH_STAT_ROWS.find((r) => r.label === 'match.overallPr');
        expect(overall.fmt(sample)).toBe('4.50');
        expect(overall.valClass).toBe('pr-val');
        expect(overall.bullet).toBe(true);
    });
});
