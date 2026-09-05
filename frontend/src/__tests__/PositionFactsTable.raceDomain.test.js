import { describe, test, expect, beforeEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import PositionFactsTable from '../components/PositionFactsTable.svelte';

const epc = { epc: 42.5, pipCount: 40, wastage: 2.5, meanRolls: 5.2, stdDev: 1.1 };

// The race block names the one-sided table it read when that table is wider
// than the home board (ADR-0027 §9). At six points the label would be noise;
// beyond it, it is the only sign that a side with a chequer outside its home
// board was answered at all.
describe('PositionFactsTable — the race domain', () => {
    beforeEach(cleanup);

    test('says nothing at six points', () => {
        render(PositionFactsTable, { bottomEPC: epc, topEPC: epc, bottomPoints: 6, topPoints: 6 });
        expect(screen.queryByText(/^OS-\d\d$/)).toBeNull();
    });

    test('names the table when a side was answered from a wider one', () => {
        render(PositionFactsTable, { bottomEPC: epc, topEPC: epc, bottomPoints: 8, topPoints: 6 });
        expect(screen.getByText('OS-08')).toBeTruthy();
    });

    test('names the widest of the two', () => {
        render(PositionFactsTable, { bottomEPC: epc, topEPC: epc, bottomPoints: 7, topPoints: 10 });
        expect(screen.getByText('OS-10')).toBeTruthy();
    });

    test('says nothing when the width is unknown', () => {
        render(PositionFactsTable, { bottomEPC: epc, topEPC: epc });
        expect(screen.queryByText(/^OS-\d\d$/)).toBeNull();
    });
});
