import { describe, test, expect } from 'vitest';
import { nextSort } from '../utils/tableSort.js';

describe('nextSort', () => {
    test('switching to a different column uses defaultDir (asc by default)', () => {
        expect(nextSort(null, 'asc', 'date')).toEqual({ column: 'date', direction: 'asc' });
        expect(nextSort('player1', 'desc', 'date')).toEqual({ column: 'date', direction: 'asc' });
    });

    test('switching column honours an explicit defaultDir', () => {
        expect(nextSort(null, 'asc', 'equity', { defaultDir: 'desc' })).toEqual({ column: 'equity', direction: 'desc' });
    });

    test('same column ascending → descending', () => {
        expect(nextSort('date', 'asc', 'date', { tristate: true })).toEqual({ column: 'date', direction: 'desc' });
        expect(nextSort('equity', 'asc', 'equity')).toEqual({ column: 'equity', direction: 'desc' });
    });

    test('tristate: same column descending → unsorted', () => {
        expect(nextSort('date', 'desc', 'date', { tristate: true })).toEqual({ column: null, direction: 'asc' });
    });

    test('two-state (no tristate): same column descending → ascending', () => {
        expect(nextSort('equity', 'desc', 'equity')).toEqual({ column: 'equity', direction: 'asc' });
    });
});
