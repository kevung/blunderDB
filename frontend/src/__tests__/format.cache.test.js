/**
 * Formatters are cached per language: the cached text must equal a direct Intl call, and a
 * language change must reach the next call instead of reusing the previous locale's formatter.
 */

import { describe, test, expect, beforeEach } from 'vitest';

import { initLanguage } from '../i18n/index.js';
import { formatDate, formatDateTime, formatNumber, formatPercent, pluralCategory } from '../utils/format.js';

const WHEN = new Date(2026, 8, 3, 14, 5);

describe('cached Intl formatters', () => {
    beforeEach(async () => {
        await initLanguage('en');
    });

    test('a repeated call gives the same text as a direct Intl call', () => {
        for (let i = 0; i < 2; i++) {
            expect(formatDate(WHEN)).toBe(new Intl.DateTimeFormat('en', { year: 'numeric', month: '2-digit', day: '2-digit' }).format(WHEN));
            expect(formatDateTime(WHEN)).toBe(new Intl.DateTimeFormat('en', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(WHEN));
            expect(formatNumber(1234567.891, { maximumFractionDigits: 1 })).toBe(new Intl.NumberFormat('en', { maximumFractionDigits: 1 }).format(1234567.891));
            expect(formatPercent(0.256)).toBe(new Intl.NumberFormat('en', { style: 'percent' }).format(0.256));
        }
    });

    test('different options are not served by the same formatter', () => {
        expect(formatNumber(1.5, { minimumFractionDigits: 3 })).toBe('1.500');
        expect(formatNumber(1.5, { maximumFractionDigits: 0 })).toBe('2');
        expect(formatNumber(1.5)).toBe('1.5');
    });

    test('a language change reaches the next call', async () => {
        const enNumber = formatNumber(1234567.5);
        const enDate = formatDateTime(WHEN);
        await initLanguage('de');
        expect(formatNumber(1234567.5)).toBe(new Intl.NumberFormat('de').format(1234567.5));
        expect(formatNumber(1234567.5)).not.toBe(enNumber);
        expect(formatDateTime(WHEN)).toBe(new Intl.DateTimeFormat('de', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(WHEN));
        expect(formatDateTime(WHEN)).not.toBe(enDate);
        await initLanguage('en');
        expect(formatNumber(1234567.5)).toBe(enNumber);
    });

    test('plural rules follow the language', async () => {
        await initLanguage('ru');
        expect(pluralCategory(3)).toBe('few');
        await initLanguage('en');
        expect(pluralCategory(3)).toBe('other');
    });
});
