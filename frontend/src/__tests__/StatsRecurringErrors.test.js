import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';

vi.mock('../services/positionLoader.js', () => ({
    loadPositionsFromSelection: vi.fn()
}));

import StatsRecurringErrors from '../components/stats/StatsRecurringErrors.svelte';
import { loadPositionsFromSelection } from '../services/positionLoader.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

const data = {
    NumDecisions: 10,
    ThresholdMP: 50,
    Groups: [
        { GameType: 'holding', Kind: 'checker', Theme: 'blots', Count: 3, SumErrorMP: 400, PRCost: 20, PositionIDs: [7, 3] },
        { GameType: 'race', Kind: 'cube', Theme: 'offer_missed', Count: 1, SumErrorMP: 120, PRCost: 6, PositionIDs: [9] }
    ]
};

describe('StatsRecurringErrors', () => {
    test('lists the groups in the order the backend ranked them, with the PR cost', () => {
        render(StatsRecurringErrors, { props: { data } });
        const rows = screen.getAllByRole('row').slice(1);
        expect(rows).toHaveLength(2);
        expect(rows[0].textContent).toContain('20.00');
        expect(rows[1].textContent).toContain('6.00');
    });

    test('a click on a group opens its positions', async () => {
        render(StatsRecurringErrors, { props: { data } });
        const buttons = screen.getAllByRole('button');
        await fireEvent.click(buttons[1]);
        expect(loadPositionsFromSelection).toHaveBeenCalledWith([9]);
    });

    test('says so when the filter has no error', () => {
        render(StatsRecurringErrors, { props: { data: { NumDecisions: 4, Groups: [] } } });
        expect(screen.queryAllByRole('row')).toHaveLength(0);
    });
});
