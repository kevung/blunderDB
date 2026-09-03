/**
 * format.test.js
 *
 * Guards formatDate/formatDateTime/formatNumber/formatPercent/pluralCategory
 * (#213): every one of them must follow the active UI language, not a
 * hardcoded locale or the browser's own — that mismatch is exactly the bug
 * this utility replaces (see format.js's header comment).
 */

import { describe, test, expect, beforeEach } from 'vitest';

import { initLanguage, LOCALES } from '../i18n/index.js';
import { formatDate, formatDateTime, formatNumber, formatPercent, pluralCategory } from '../utils/format.js';

describe('formatDate / formatDateTime', () => {
    beforeEach(async () => {
        await initLanguage('en');
    });

    test.each(LOCALES)('%s formats a known date without throwing, and it is not empty', async (locale) => {
        await initLanguage(locale);
        const out = formatDate('2026-09-03T12:00:00Z');
        expect(out).not.toBe('');
        expect(out).toMatch(/2026/);
    });

    test('date order changes with the active language (en vs ja)', async () => {
        await initLanguage('en');
        const en = formatDate('2026-09-03T12:00:00Z');
        await initLanguage('ja');
        const ja = formatDate('2026-09-03T12:00:00Z');
        expect(en).not.toBe(ja);
        // en-US: month first; ja: year first (both ISO-adjacent orders differ).
        expect(en.indexOf('09')).toBeLessThan(en.indexOf('2026'));
        expect(ja.indexOf('2026')).toBeLessThan(ja.indexOf('09'));
    });

    test('formatDateTime appends a time component', async () => {
        await initLanguage('en');
        const dateOnly = formatDate('2026-09-03T14:30:00Z');
        const withTime = formatDateTime('2026-09-03T14:30:00Z');
        expect(withTime.length).toBeGreaterThan(dateOnly.length);
    });

    test('empty/invalid input returns the empty string', () => {
        expect(formatDate('')).toBe('');
        expect(formatDate(null)).toBe('');
        expect(formatDate('not-a-date')).toBe('');
        expect(formatDateTime(undefined)).toBe('');
    });
});

describe('formatNumber / formatPercent', () => {
    beforeEach(async () => {
        await initLanguage('en');
    });

    test('grouping separator changes with the active language (en vs de)', async () => {
        await initLanguage('en');
        const en = formatNumber(1234.5);
        await initLanguage('de');
        const de = formatNumber(1234.5);
        expect(en).not.toBe(de);
        expect(en).toContain(','); // en-US groups with a comma
        expect(de).toContain('.'); // de groups with a period
    });

    test('formatPercent renders a ratio as a percentage', async () => {
        await initLanguage('en');
        expect(formatPercent(0.1234, { minimumFractionDigits: 2 })).toBe('12.34%');
    });

    test('non-numeric input returns the empty string', () => {
        expect(formatNumber(null)).toBe('');
        expect(formatNumber(NaN)).toBe('');
        expect(formatPercent(undefined)).toBe('');
    });
});

describe('pluralCategory', () => {
    test('English only has one/other', async () => {
        await initLanguage('en');
        expect(pluralCategory(1)).toBe('one');
        expect(pluralCategory(0)).toBe('other');
        expect(pluralCategory(5)).toBe('other');
    });

    test('Russian has one/few/many categories English lacks (n===1 is not enough)', async () => {
        await initLanguage('ru');
        expect(pluralCategory(1)).toBe('one');
        expect(pluralCategory(2)).toBe('few');
        expect(pluralCategory(5)).toBe('many');
    });
});
