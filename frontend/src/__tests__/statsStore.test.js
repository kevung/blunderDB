import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Mock Wails Database binding
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ComputeStats: vi.fn()
}));

import { ComputeStats } from '../../wailsjs/go/database/Database.js';
import { statsFilterStore, statsResultStore, statsLoadingStore, statsErrorStore, statsMetricStore, refreshStats } from '../stores/statsStore.js';

describe('statsStore — initial state', () => {
    test('statsResultStore starts null', () => {
        expect(get(statsResultStore)).toBeNull();
    });

    test('statsLoadingStore starts false', () => {
        expect(get(statsLoadingStore)).toBe(false);
    });

    test('statsErrorStore starts null', () => {
        expect(get(statsErrorStore)).toBeNull();
    });

    test('statsMetricStore starts pr', () => {
        expect(get(statsMetricStore)).toBe('pr');
    });

    test('statsFilterStore has sensible defaults', () => {
        const f = get(statsFilterStore);
        expect(f.playerName).toBe('');
        expect(f.decisionType).toBe(-1);
        expect(f.tournamentIDs).toEqual([]);
        expect(f.matchLength).toEqual([]);
    });
});

describe('refreshStats()', () => {
    const fakeResult = { prGlobal: 3.14, totals: { numDecisions: 42 } };

    beforeEach(() => {
        statsResultStore.set(null);
        statsLoadingStore.set(false);
        statsErrorStore.set(null);
        vi.resetAllMocks();
    });

    test('sets loading=true then false, populates result on success', async () => {
        let resolveCall;
        ComputeStats.mockReturnValue(
            new Promise((res) => {
                resolveCall = res;
            })
        );

        const promise = refreshStats({ playerName: '', decisionType: -1 });
        expect(get(statsLoadingStore)).toBe(true);

        resolveCall(fakeResult);
        await promise;

        expect(get(statsLoadingStore)).toBe(false);
        expect(get(statsResultStore)).toEqual(fakeResult);
        expect(get(statsErrorStore)).toBeNull();
    });

    test('sets error store and clears result on failure', async () => {
        ComputeStats.mockRejectedValue(new Error('backend error'));

        await refreshStats({});

        expect(get(statsLoadingStore)).toBe(false);
        expect(get(statsResultStore)).toBeNull();
        expect(get(statsErrorStore)).toBe('backend error');
    });

    test('calls ComputeStats with the provided filter', async () => {
        ComputeStats.mockResolvedValue(fakeResult);
        const filter = { playerName: 'Alice', decisionType: 0 };
        await refreshStats(filter);
        expect(ComputeStats).toHaveBeenCalledWith(filter);
    });
});
