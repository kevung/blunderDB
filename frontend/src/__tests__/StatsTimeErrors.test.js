import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

const ROWS = [
    { player: 'Alice', bucket: 0, decisions: 4, scored: 4, mean_error_mp: 12.34, blunders: 1, positions: 3 },
    { player: 'Alice', bucket: 3, decisions: 2, scored: 0, mean_error_mp: 0, blunders: 0, positions: 0 }
];
const GetTimeErrors = vi.fn(() => Promise.resolve(ROWS));
vi.mock('../../wailsjs/go/database/Database.js', () => ({ GetTimeErrors: (/** @type {any[]} */ ...a) => GetTimeErrors(...a) }));
const load = vi.fn();
vi.mock('../services/positionLoader.js', () => ({ loadPositionsFromStatsSelection: (/** @type {any[]} */ ...a) => load(...a) }));

import StatsTimeErrors from '../components/stats/StatsTimeErrors.svelte';

describe('StatsTimeErrors', () => {
    afterEach(() => {
        cleanup();
        GetTimeErrors.mockClear();
    });

    test('one row per player and band; an unscored band reads empty, not zero', async () => {
        const { container } = render(StatsTimeErrors);
        await vi.waitFor(() => expect(container.querySelector('[data-testid="time-errors"]')).not.toBeNull());
        const rows = [...container.querySelectorAll('tbody tr')].map((r) => [...r.querySelectorAll('td')].map((c) => c.textContent));
        expect(rows[0].slice(2)).toEqual(['3', '12.3', '25.0 %']);
        expect(rows[1].slice(2)).toEqual(['0', '', '']);
    });

    test('a band count opens its positions, whatever the filter', async () => {
        const { container } = render(StatsTimeErrors);
        await vi.waitFor(() => expect(container.querySelector('[data-testid="time-band-0"]')).not.toBeNull());
        /** @type {HTMLElement} */ (container.querySelector('[data-testid="time-band-0"]')).click();
        expect(load).toHaveBeenCalledWith({}, { Kind: 'time_band', Player: 'Alice', TimeBucket: 0 });
    });

    test('absent when no decision has a time', async () => {
        GetTimeErrors.mockResolvedValueOnce([]);
        const { container } = render(StatsTimeErrors);
        await tick();
        await tick();
        expect(container.querySelector('[data-testid="time-errors"]')).toBeNull();
    });
});
