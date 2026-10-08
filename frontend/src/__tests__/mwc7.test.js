import { describe, it, expect, beforeEach } from 'vitest';
import { language } from '../i18n/index.js';
import { fmtMwc7, fmtMwc7Interval, fmtMwc7Full, mwc7Tooltip, mwc7Percent, hasMwc7 } from '../utils/mwc7.js';

const base = { available: true, loss: 0.123, has_interval: true, low: 0.081, high: 0.165, elo: -170.4, elo_low: -200, elo_high: -140, elo_floored: false, matches: 12 };

beforeEach(() => language.set('en'));

describe('mwc7 formatting', () => {
    it('shows the loss in percent with one decimal', () => {
        expect(fmtMwc7(base)).toBe('12.3 %');
        expect(hasMwc7(base)).toBe(true);
        expect(mwc7Percent(base)).toBeCloseTo(12.3);
    });

    it('follows the locale decimal separator', () => {
        language.set('fr');
        expect(fmtMwc7(base)).toBe('12,3 %');
    });

    it('shows a dash, never a number, when unavailable', () => {
        const m = { ...base, available: false };
        expect(fmtMwc7(m)).toBe('—');
        expect(fmtMwc7Full(m)).toBe('—');
        expect(mwc7Percent(m)).toBeNull();
        expect(fmtMwc7(null)).toBe('—');
        expect(mwc7Tooltip(m)).toMatch(/money/i);
    });

    it('formats the interval only when there is one', () => {
        expect(fmtMwc7Interval(base)).toBe('[8.1–16.5]');
        expect(fmtMwc7Full(base)).toBe('12.3 % [8.1–16.5]');
        const noInterval = { ...base, has_interval: false };
        expect(fmtMwc7Interval(noInterval)).toBe('');
        expect(fmtMwc7Full(noInterval)).toBe('12.3 %');
    });

    it('states the Elo in the tooltip, as an upper bound when floored', () => {
        expect(mwc7Tooltip(base)).toContain('≈ -170 Elo');
        expect(mwc7Tooltip({ ...base, elo_floored: true })).toContain('≤ -170 Elo');
    });
});
