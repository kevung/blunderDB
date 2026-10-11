/**
 * columnView.test.js — the per-column view of a table (ADR-0090): numeric filter syntax,
 * text filters, unmeasured values last in both directions, ties broken by the text column.
 */
import { describe, test, expect } from 'vitest';
import { parseNumericFilter, applyColumnView, filterIsInvalid } from '../utils/columnView.js';

/** @param {string} text @param {number} v */
const keeps = (text, v) => {
    const f = parseNumericFilter(text);
    if (!f || 'error' in f) throw new Error(`unreadable ${text}`);
    return f.test(v);
};

describe('parseNumericFilter', () => {
    test('reads comparisons, ranges and a bare number', () => {
        expect(keeps('>=500', 500)).toBe(true);
        expect(keeps('≥500', 499)).toBe(false);
        expect(keeps('>500', 500)).toBe(false);
        expect(keeps('<2.5', 2.4)).toBe(true);
        expect(keeps('<2,5', 2.5)).toBe(false);
        expect(keeps('<=2', 2)).toBe(true);
        expect(keeps('≤ 2', 2.1)).toBe(false);
        expect(keeps('3-7', 3)).toBe(true);
        expect(keeps('3-7', 7)).toBe(true);
        expect(keeps('7 - 3', 8)).toBe(false);
        expect(keeps('5', 5)).toBe(true);
        expect(keeps('5', 6)).toBe(false);
        expect(keeps('-2', -2)).toBe(true);
    });
    test('blank is no filter; anything else unreadable is an error', () => {
        expect(parseNumericFilter('')).toBeNull();
        expect(parseNumericFilter('   ')).toBeNull();
        expect(parseNumericFilter('abc')).toEqual({ error: true });
        expect(parseNumericFilter('>=')).toEqual({ error: true });
        expect(filterIsInvalid({ key: 'n', filter: 'number' }, '>>3')).toBe(true);
        expect(filterIsInvalid({ key: 'n', filter: 'number' }, '>3')).toBe(false);
        expect(filterIsInvalid({ key: 'n', filter: 'text' }, '>>3')).toBe(false);
    });
});

const ROWS = [
    { name: 'Cid', pr: 4.2, n: 600 },
    { name: 'ann', pr: null, n: 10 },
    { name: 'Bob', pr: 4.2, n: 900 },
    { name: 'Dee', pr: 2.1, n: 300 }
];
const COLUMNS = [
    { key: 'name', filter: /** @type {const} */ ('text') },
    { key: 'pr', filter: /** @type {const} */ ('number') },
    { key: 'n', value: (/** @type {any} */ r) => r.n, filter: /** @type {const} */ ('number') }
];
const names = (/** @type {any[]} */ rows) => rows.map((r) => r.name);

describe('applyColumnView', () => {
    test('unmeasured rows sink in both directions; ties fall back to the name', () => {
        expect(names(applyColumnView(ROWS, COLUMNS, { sort: { column: 'pr', direction: 'asc' } }))).toEqual(['Dee', 'Bob', 'Cid', 'ann']);
        expect(names(applyColumnView(ROWS, COLUMNS, { sort: { column: 'pr', direction: 'desc' } }))).toEqual(['Bob', 'Cid', 'Dee', 'ann']);
    });
    test('text sorts case-insensitively', () => {
        expect(names(applyColumnView(ROWS, COLUMNS, { sort: { column: 'name', direction: 'asc' } }))).toEqual(['ann', 'Bob', 'Cid', 'Dee']);
    });
    test('filters combine; a numeric filter drops unmeasured rows; an unreadable one filters nothing', () => {
        expect(names(applyColumnView(ROWS, COLUMNS, { filters: { n: '>=500' } }))).toEqual(['Cid', 'Bob']);
        expect(names(applyColumnView(ROWS, COLUMNS, { filters: { n: '>=500', name: 'B' } }))).toEqual(['Bob']);
        expect(names(applyColumnView(ROWS, COLUMNS, { filters: { pr: '<5' } }))).toEqual(['Cid', 'Bob', 'Dee']);
        expect(names(applyColumnView(ROWS, COLUMNS, { filters: { pr: 'oops' } }))).toEqual(names(ROWS));
    });
    test('no sort keeps the given order and never mutates the input', () => {
        const before = [...ROWS];
        applyColumnView(ROWS, COLUMNS, { sort: { column: 'n', direction: 'desc' } });
        expect(ROWS).toEqual(before);
        expect(applyColumnView(ROWS, COLUMNS, {})).toEqual(ROWS);
    });
});

describe('a bare number', () => {
    test('matches at the precision typed, not by float equality', () => {
        expect(keeps('4.2', 4.2)).toBe(true);
        expect(keeps('4.2', 4.1 + 0.1)).toBe(true);
        expect(keeps('4.2', 4.24)).toBe(true);
        expect(keeps('4.2', 4.26)).toBe(false);
        expect(keeps('4.20', 4.204)).toBe(true);
        expect(keeps('4.20', 4.206)).toBe(false);
    });
});
