import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ComputeStats: vi.fn().mockResolvedValue({}),
    GetPlayerTable: vi.fn()
}));

vi.mock('../stores/uiStore.js', () => {
    const { writable } = require('svelte/store');
    return { dbMutationCounterStore: writable(0) };
});

vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/some/db.db') };
});

// Import after mocks
import { GetPlayerTable } from '../../wailsjs/go/database/Database.js';
import { refreshPlayerTable, playerTableStore, playerTableErrorStore } from '../stores/statsStore.js';

const ROWS = [
    { name: 'Alice', matches: 2, wins: 1, losses: 1, decisions: 40, pr: 4.2, luck_known: true, luck_rate_mp: 12, luck_rolls: 100 },
    { name: 'Bob', matches: 1, wins: 0, losses: 1, decisions: 20, pr: 6.8, luck_known: false, luck_rate_mp: 0, luck_rolls: 0 }
];

const baseFilter = {
    playerName: '',
    tournamentIDs: [],
    dateFrom: '',
    dateTo: '',
    decisionType: -1,
    matchLength: []
};

describe('refreshPlayerTable', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        playerTableStore.set(null);
        playerTableErrorStore.set(null);
        GetPlayerTable.mockResolvedValue(ROWS);
    });

    test('fetches the table and stores the rows', async () => {
        await refreshPlayerTable(baseFilter, 'key-1');
        expect(GetPlayerTable).toHaveBeenCalledTimes(1);
        expect(get(playerTableStore)).toEqual(ROWS);
    });

    test('does not refetch when only the player selection changed', async () => {
        await refreshPlayerTable(baseFilter, 'key-2');
        expect(GetPlayerTable).toHaveBeenCalledTimes(1);

        // The backend ignores playerName and decisionType for this table, so
        // picking a player in another tab must not trigger an identical fetch.
        await refreshPlayerTable({ ...baseFilter, playerName: 'Alice', decisionType: 0 }, 'key-2');
        expect(GetPlayerTable).toHaveBeenCalledTimes(1);
    });

    test('refetches when a filter the table does depend on changes', async () => {
        await refreshPlayerTable(baseFilter, 'key-3');
        await refreshPlayerTable({ ...baseFilter, dateFrom: '2025-01-01' }, 'key-3');
        expect(GetPlayerTable).toHaveBeenCalledTimes(2);
    });

    test('refetches when the database changed', async () => {
        await refreshPlayerTable(baseFilter, 'key-4');
        await refreshPlayerTable(baseFilter, 'key-5');
        expect(GetPlayerTable).toHaveBeenCalledTimes(2);
    });

    test('surfaces an error and allows a retry', async () => {
        GetPlayerTable.mockRejectedValueOnce(new Error('boom'));
        await refreshPlayerTable(baseFilter, 'key-6');
        expect(get(playerTableErrorStore)).toBe('boom');
        expect(get(playerTableStore)).toBeNull();

        // A failed fetch must not be cached, or the panel would stay empty
        // until something else changed.
        GetPlayerTable.mockResolvedValue(ROWS);
        await refreshPlayerTable(baseFilter, 'key-6');
        expect(get(playerTableStore)).toEqual(ROWS);
    });

    test('an empty database yields an empty table, not a null one', async () => {
        GetPlayerTable.mockResolvedValue(null);
        await refreshPlayerTable(baseFilter, 'key-7');
        expect(get(playerTableStore)).toEqual([]);
    });
});
