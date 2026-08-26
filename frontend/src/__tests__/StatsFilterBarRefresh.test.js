/**
 * The Stats filter bar reads what the database contains — the player list, the
 * tournaments, the date bounds. It used to read them once, at mount, so any
 * mutation that happened while it was on screen left it describing a database
 * that no longer existed: merged names still offered in the dropdown, and a
 * selected player who had just been merged away still filtering every tab down
 * to nothing.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { render, waitFor } from '@testing-library/svelte';

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn(),
    GetAllTournaments: vi.fn().mockResolvedValue([]),
    GetStatsDateRange: vi.fn().mockResolvedValue({ DateFrom: '2025-01-01', DateTo: '2026-01-01' }),
    ComputeStats: vi.fn().mockResolvedValue({}),
    GetPlayerTable: vi.fn().mockResolvedValue([])
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn().mockResolvedValue(null),
    SaveStatsFilter: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../stores/uiStore.js', () => {
    const { writable } = require('svelte/store');
    return { dbMutationCounterStore: writable(0) };
});

vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/some/db.db'), databaseLoadedStore: writable(true) };
});

// Import after mocks
import { GetAllPlayerNames } from '../../wailsjs/go/database/Database.js';
import { GetStatsFilter } from '../../wailsjs/go/main/Config.js';
import { dbMutationCounterStore } from '../stores/uiStore.js';
import { statsFilterStore } from '../stores/statsStore.js';
import StatsFilterBar from '../components/stats/StatsFilterBar.svelte';

const BEFORE_MERGE = [
    { Name: 'K. Unger', Count: 3 },
    { Name: 'Kevin Unger', Count: 5 },
    { Name: 'Bob', Count: 4 }
];
const AFTER_MERGE = [
    { Name: 'Kevin Unger', Count: 8 },
    { Name: 'Bob', Count: 4 }
];

describe('StatsFilterBar refresh', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        dbMutationCounterStore.set(0);
        statsFilterStore.set({
            playerName: '',
            tournamentIDs: [],
            dateFrom: '',
            dateTo: '',
            decisionType: -1,
            matchLength: []
        });
        GetStatsFilter.mockResolvedValue(null);
        GetAllPlayerNames.mockResolvedValue(BEFORE_MERGE);
    });

    test('re-reads the player list when the database changes', async () => {
        const { container } = render(StatsFilterBar);
        await waitFor(() => {
            expect(container.querySelector('#fb-player')).toBeTruthy();
        });
        await waitFor(() => {
            expect(container.querySelectorAll('#fb-player option').length).toBe(BEFORE_MERGE.length + 1);
        });

        // A merge happened elsewhere in the app.
        GetAllPlayerNames.mockResolvedValue(AFTER_MERGE);
        dbMutationCounterStore.update((n) => n + 1);

        await waitFor(() => {
            const names = [...container.querySelectorAll('#fb-player option')].map((o) => o.value);
            expect(names).not.toContain('K. Unger');
            expect(names).toContain('Kevin Unger');
        });
    });

    test('drops a selected player who was merged away', async () => {
        // The user had picked the name that is about to disappear.
        GetStatsFilter.mockResolvedValue({ player_name: 'K. Unger' });

        render(StatsFilterBar);
        await waitFor(() => {
            expect(get(statsFilterStore).playerName).toBe('K. Unger');
        });

        GetAllPlayerNames.mockResolvedValue(AFTER_MERGE);
        dbMutationCounterStore.update((n) => n + 1);

        // Left in place, the filter would match nothing and every tab would
        // report empty statistics for a player the database no longer knows.
        await waitFor(() => {
            expect(get(statsFilterStore).playerName).toBe('');
        });
    });

    test('keeps a selected player who survived the merge', async () => {
        GetStatsFilter.mockResolvedValue({ player_name: 'Bob' });

        render(StatsFilterBar);
        await waitFor(() => {
            expect(get(statsFilterStore).playerName).toBe('Bob');
        });

        GetAllPlayerNames.mockResolvedValue(AFTER_MERGE);
        dbMutationCounterStore.update((n) => n + 1);

        await waitFor(() => {
            expect(GetAllPlayerNames).toHaveBeenCalledTimes(2);
        });
        expect(get(statsFilterStore).playerName).toBe('Bob');
    });
});
