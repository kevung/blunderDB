import { describe, test, expect } from 'vitest';
import { narrowToDays, narrowToMonths, narrowToPeriod } from '../utils/statsWindow.js';

describe('statsWindow', () => {
    test('a span is intersected with the filter dates, never widened', () => {
        expect(narrowToDays({ dateFrom: '', dateTo: '' }, '2025-01-01', '2025-01-31')).toEqual({ dateFrom: '2025-01-01', dateTo: '2025-01-31' });
        expect(narrowToDays({ dateFrom: '2025-01-10', dateTo: '2025-01-20' }, '2025-01-01', '2025-01-31')).toEqual({ dateFrom: '2025-01-10', dateTo: '2025-01-20' });
    });

    test('a window of months ends on the last day of its last month', () => {
        expect(narrowToMonths({}, { from: '2024-12', to: '2025-02' })).toEqual({ dateFrom: '2024-12-01', dateTo: '2025-02-28' });
    });

    test('a week runs Monday to Sunday, a month to its last day', () => {
        expect(narrowToPeriod({}, '2025-12-29', 'week')).toEqual({ dateFrom: '2025-12-29', dateTo: '2026-01-04' });
        expect(narrowToPeriod({}, '2024-02-01', 'month')).toEqual({ dateFrom: '2024-02-01', dateTo: '2024-02-29' });
    });
});
