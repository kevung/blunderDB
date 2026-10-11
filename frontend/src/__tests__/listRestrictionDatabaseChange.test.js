import { describe, test, expect, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(async () => []),
    CountMatches: vi.fn(async () => 0),
    GetMatchByID: vi.fn(async () => null)
}));

import { matchListStore } from '../stores/matchListStore.js';
import { tournamentIDsFilterStore } from '../stores/tournamentStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

// A restriction opened from Stats names ids of one database: they mean other rows, or none, in the next.
describe('list restrictions and the open database', () => {
    test('opening another database clears the match and tournament restrictions', async () => {
        databasePathStore.set('/a.db');
        await matchListStore.setIDs([1, 2]);
        tournamentIDsFilterStore.set([3, 4]);

        databasePathStore.set('/b.db');

        expect(get(matchListStore).ids).toBeNull();
        expect(get(tournamentIDsFilterStore)).toBeNull();
    });

    test('the same database set again keeps them', async () => {
        databasePathStore.set('/a.db');
        await matchListStore.setIDs([1, 2]);
        tournamentIDsFilterStore.set([3, 4]);

        databasePathStore.set('/a.db');

        expect(get(matchListStore).ids).toEqual([1, 2]);
        expect(get(tournamentIDsFilterStore)).toEqual([3, 4]);
    });
});
