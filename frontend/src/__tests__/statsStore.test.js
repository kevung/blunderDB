import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Mock Wails Database binding
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ComputeStats: vi.fn(),
    ComputeStudyPlan: vi.fn(),
    ComputeStudyEffect: vi.fn(),
    ComputeDirectionalBiases: vi.fn()
}));

import { ComputeStats, ComputeStudyPlan, ComputeStudyEffect, ComputeDirectionalBiases } from '../../wailsjs/go/database/Database.js';
import {
    refreshStudyPlan,
    refreshStudyLoop,
    studyEffectStore,
    biasesStore,
    studyLoopLoadingStore,
    studyLoopErrorStore,
    studyPlanStore,
    statsFilterStore,
    statsResultStore,
    statsLoadingStore,
    statsErrorStore,
    statsMetricStore,
    refreshStats,
    statsResultFilterStore
} from '../stores/statsStore.js';

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

describe('refreshStudyPlan', () => {
    test('a slower, older reply does not overwrite the latest', async () => {
        let resolveOld;
        ComputeStudyPlan.mockImplementationOnce(() => new Promise((r) => (resolveOld = r)));
        ComputeStudyPlan.mockImplementationOnce(() => Promise.resolve({ tag: 'new' }));
        const first = refreshStudyPlan({ a: 1 }, 1);
        await refreshStudyPlan({ a: 2 }, 1);
        resolveOld({ tag: 'old' });
        await first;
        expect(get(studyPlanStore)).toEqual({ tag: 'new' });
    });
});

describe('refreshStudyLoop', () => {
    test('a slower, older reply does not overwrite the latest nor clear loading', async () => {
        let resolveOld;
        ComputeStudyEffect.mockImplementationOnce(() => new Promise((r) => (resolveOld = r)));
        ComputeStudyEffect.mockImplementationOnce(() => Promise.resolve({ tag: 'new' }));
        ComputeDirectionalBiases.mockImplementationOnce(() => Promise.resolve({ tag: 'old' }));
        ComputeDirectionalBiases.mockImplementationOnce(() => Promise.resolve({ tag: 'new' }));
        const first = refreshStudyLoop({ a: 1 }, 1);
        await refreshStudyLoop({ a: 2 }, 1);
        resolveOld({ tag: 'old' });
        await first;
        expect(get(studyEffectStore)).toEqual({ tag: 'new' });
        expect(get(biasesStore)).toEqual({ tag: 'new' });
        expect(get(studyLoopLoadingStore)).toBe(false);
        expect(get(studyLoopErrorStore)).toBeNull();
    });
});

describe('refreshStats() — an older reply landing last', () => {
    // Opening the panel computes the default filter, then the restored one;
    // the default, wider, can answer after the restored one.
    test('keeps the result and filter of the latest request', async () => {
        statsResultStore.set(null);
        vi.resetAllMocks();
        /** @type {(v: any) => void} */
        let resolveOld = () => {};
        /** @type {(v: any) => void} */
        let resolveNew = () => {};
        vi.mocked(ComputeStats)
            .mockImplementationOnce(() => new Promise((r) => (resolveOld = r)))
            .mockImplementationOnce(() => new Promise((r) => (resolveNew = r)));
        const oldFilter = { playerName: '' };
        const newFilter = { playerName: 'Alice' };
        const pOld = refreshStats(oldFilter, 'race');
        const pNew = refreshStats(newFilter, 'race');
        resolveNew({ who: 'new' });
        await pNew;
        expect(get(statsLoadingStore)).toBe(false);
        resolveOld({ who: 'old' });
        await pOld;
        expect(get(statsResultStore)).toEqual({ who: 'new' });
        expect(get(statsResultFilterStore)).toEqual(newFilter);
        expect(get(statsLoadingStore)).toBe(false);
    });
});
