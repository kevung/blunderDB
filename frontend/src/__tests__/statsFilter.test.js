/**
 * The Stats filter has one owner, statsStore: every tab, the bar and the links into Stats write
 * it through patchStatsFilter, which saves it to Config.yaml after a pause.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn(),
    SaveStatsFilter: vi.fn(() => Promise.resolve())
}));

import { GetStatsFilter, SaveStatsFilter } from '../../wailsjs/go/main/Config.js';
import {
    statsFilterStore,
    statsMetricStore,
    defaultStatsFilter,
    initStatsFilter,
    patchStatsFilter,
    resetStatsFilter,
    dropMissingPlayer,
    flushStatsFilterSave,
    _resetStatsFilterForTests
} from '../stores/statsStore.js';

beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    _resetStatsFilterForTests();
    vi.mocked(GetStatsFilter).mockResolvedValue(/** @type {any} */ (null));
});
afterEach(() => vi.useRealTimers());

describe('statsFilterStore', () => {
    test('the default filter carries every field, provenance included', () => {
        expect(get(statsFilterStore)).toEqual({ ...defaultStatsFilter });
        expect(defaultStatsFilter.analysisEngine).toBe('');
        expect(defaultStatsFilter.minAnalysisDepth).toBe(0);
    });

    test('a patch merges and is saved once, after the pause', async () => {
        await initStatsFilter();
        expect(patchStatsFilter({ playerName: 'Alice' })).toBe(true);
        patchStatsFilter({ matchLength: [7] });
        expect(get(statsFilterStore)).toMatchObject({ playerName: 'Alice', matchLength: [7] });
        expect(SaveStatsFilter).not.toHaveBeenCalled();
        vi.advanceTimersByTime(500);
        expect(SaveStatsFilter).toHaveBeenCalledTimes(1);
        expect(SaveStatsFilter).toHaveBeenCalledWith(expect.objectContaining({ player_name: 'Alice', match_length: [7], decision_type: null, metric: 'pr' }));
    });

    test('a range ending before it starts is refused and leaves the filter as it was', () => {
        patchStatsFilter({ dateFrom: '2025-03-01' });
        expect(patchStatsFilter({ dateTo: '2025-01-01' })).toBe(false);
        expect(get(statsFilterStore).dateTo).toBe('');
    });

    test('init restores the saved filter and metric once', async () => {
        vi.mocked(GetStatsFilter).mockResolvedValue(/** @type {any} */ ({ player_name: 'Bob', decision_type: 1, metric: 'mwc' }));
        await Promise.all([initStatsFilter(), initStatsFilter()]);
        expect(GetStatsFilter).toHaveBeenCalledTimes(1);
        expect(get(statsFilterStore)).toMatchObject({ playerName: 'Bob', decisionType: 1, analysisEngine: '' });
        expect(get(statsMetricStore)).toBe('mwc');
        // Restoring is not a change: nothing is written back.
        vi.advanceTimersByTime(500);
        expect(SaveStatsFilter).not.toHaveBeenCalled();
    });

    test('a choice made while the saved filter loads is applied over it', async () => {
        vi.mocked(GetStatsFilter).mockResolvedValue(/** @type {any} */ ({ player_name: 'Bob', decision_type: 1, match_length: [7] }));
        const init = initStatsFilter();
        patchStatsFilter({ playerName: 'Alice' });
        await init;
        expect(get(statsFilterStore)).toMatchObject({ playerName: 'Alice', decisionType: 1, matchLength: [7] });
    });

    test('a save still waiting is written at once when the window closes', async () => {
        await initStatsFilter();
        patchStatsFilter({ playerName: 'Alice' });
        window.dispatchEvent(new Event('beforeunload'));
        expect(SaveStatsFilter).toHaveBeenCalledTimes(1);
        expect(SaveStatsFilter).toHaveBeenCalledWith(expect.objectContaining({ player_name: 'Alice' }));
        vi.advanceTimersByTime(500);
        flushStatsFilterSave();
        expect(SaveStatsFilter).toHaveBeenCalledTimes(1);
    });

    test('a metric change is saved with the filter', async () => {
        await initStatsFilter();
        statsMetricStore.set('mwc7');
        vi.advanceTimersByTime(500);
        expect(SaveStatsFilter).toHaveBeenCalledWith(expect.objectContaining({ metric: 'mwc7' }));
    });

    test('reset keeps the player and clears the rest', () => {
        patchStatsFilter({ playerName: 'Alice', tournamentIDs: [3], analysisEngine: 'XG', minAnalysisDepth: 2 });
        resetStatsFilter();
        expect(get(statsFilterStore)).toEqual({ ...defaultStatsFilter, playerName: 'Alice' });
    });

    test('a player the database no longer holds is dropped', () => {
        patchStatsFilter({ playerName: 'K. Unger' });
        dropMissingPlayer(['Kevin Unger']);
        expect(get(statsFilterStore).playerName).toBe('');
        patchStatsFilter({ playerName: 'Kevin Unger' });
        dropMissingPlayer(['Kevin Unger']);
        expect(get(statsFilterStore).playerName).toBe('Kevin Unger');
    });
});
