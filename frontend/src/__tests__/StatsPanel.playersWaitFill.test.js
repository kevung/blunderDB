/**
 * While the match statistics fill in the background, the Players tab waits for them with the
 * fill's progress instead of reading a table that would compute the rest of the library itself;
 * it reads the table once the fill ends.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    MaintenanceStatus: vi.fn(() => Promise.resolve([])),
    ComputeStats: vi.fn(() => Promise.resolve(null)),
    GetPlayerTable: vi.fn(() => Promise.resolve([])),
    GetAllPlayerNames: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    FillMatchStatsInBackground: vi.fn(() => Promise.resolve(false))
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn(() => Promise.resolve(null)),
    SaveStatsFilter: vi.fn(() => Promise.resolve())
}));

import { statsTabStore } from '../stores/statsStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { maintenanceStore, STATS_MATCHES } from '../stores/maintenanceStore.js';
import { GetPlayerTable } from '../../wailsjs/go/database/Database.js';
import StatsPanel from '../components/stats/StatsPanel.svelte';

describe('StatsPanel — Players while the match statistics fill', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        statsTabStore.set('players');
        maintenanceStore.set({ kind: STATS_MATCHES, done: 2, total: 5 });
    });
    afterEach(() => {
        cleanup();
        maintenanceStore.set(null);
    });

    test('waits with the progress, then reads the table once the fill ends', async () => {
        render(StatsPanel);
        await tick();
        expect(screen.getByTestId('players-wait-fill').textContent).toContain('2/5');
        expect(GetPlayerTable).not.toHaveBeenCalled();

        maintenanceStore.set(null);
        await tick();
        expect(screen.queryByTestId('players-wait-fill')).toBeNull();
        expect(GetPlayerTable).toHaveBeenCalledTimes(1);
    });
});
