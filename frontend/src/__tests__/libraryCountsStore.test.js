import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const estimate = vi.fn();
const exact = vi.fn();
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetDatabaseStatsEstimate: (...a) => estimate(...a),
    GetDatabaseStats: (...a) => exact(...a)
}));

import { libraryCountsStore, refreshLibraryCounts, formatCount } from '../stores/libraryCountsStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

describe('compteur de bibliothèque', () => {
    beforeEach(() => {
        estimate.mockReset();
        exact.mockReset();
        databasePathStore.set('/tmp/x.db');
    });

    it("lit l'estimation, jamais le décompte exact", async () => {
        estimate.mockResolvedValue({ position_count: 12, match_count: 3, blunder_count: 4, approximate: [] });
        await refreshLibraryCounts();
        expect(exact).not.toHaveBeenCalled();
        expect(get(libraryCountsStore)).toEqual({
            positions: 12,
            blunders: 4,
            matches: 3,
            approximate: { positions: false, matches: false }
        });
    });

    it('marque « ≈ » une estimation et laisse les blunders inconnus', async () => {
        estimate.mockResolvedValue({ position_count: 5000000, match_count: 40, approximate: ['positions'] });
        await refreshLibraryCounts();
        const c = get(libraryCountsStore);
        expect(c.blunders).toBeNull();
        expect(formatCount(c.positions, c.approximate.positions)).toBe('≈ 5000000');
        expect(formatCount(c.blunders)).toBe('?');
        expect(formatCount(c.matches, c.approximate.matches)).toBe('40');
    });
});
