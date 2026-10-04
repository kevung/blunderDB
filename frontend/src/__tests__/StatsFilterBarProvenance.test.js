/**
 * The provenance fields of the Stats filter (engine, minimum depth) live in
 * Config.yaml beside the other fields: restored at mount, saved on change.
 */

import { describe, test, expect, vi } from 'vitest';
import { render, waitFor, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn().mockResolvedValue([{ Name: 'Ada', Count: 1 }]),
    GetAllTournaments: vi.fn().mockResolvedValue([]),
    GetStatsDateRange: vi.fn().mockResolvedValue(null)
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn().mockResolvedValue({ player_name: '', decision_type: null, analysis_engine: 'XG', min_analysis_depth: 3 }),
    SaveStatsFilter: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/x.db'), databaseLoadedStore: writable(true) };
});

import { SaveStatsFilter } from '../../wailsjs/go/main/Config.js';
import { statsFilterStore } from '../stores/statsStore.js';
import StatsFilterBar from '../components/stats/StatsFilterBar.svelte';

describe('StatsFilterBar provenance persistence', () => {
    test('restores and saves the engine and the minimum depth', async () => {
        const { container } = render(StatsFilterBar);
        await waitFor(() => expect(get(statsFilterStore).analysisEngine).toBe('XG'));
        expect(get(statsFilterStore).minAnalysisDepth).toBe(3);

        const depth = /** @type {HTMLInputElement} */ (container.querySelector('#fb-min-depth'));
        expect(depth.value).toBe('3');
        depth.value = '4';
        await fireEvent.change(depth);
        await waitFor(() => expect(SaveStatsFilter).toHaveBeenCalled(), { timeout: 2000 });
        expect(SaveStatsFilter).toHaveBeenLastCalledWith(expect.objectContaining({ analysis_engine: 'XG', min_analysis_depth: 4 }));
    });
});
