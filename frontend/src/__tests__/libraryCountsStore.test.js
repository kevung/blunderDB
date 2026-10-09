import { must } from './helpers/must.js';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const estimate = vi.fn();
const exact = vi.fn();
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetLibrarySettings: () => Promise.resolve({ errorThresholdMP: 80, blunderThresholdMP: 150 }),
    GetDatabaseStatsEstimate: (/** @type {any[]} */ ...a) => estimate(...a),
    GetDatabaseStats: (/** @type {any[]} */ ...a) => exact(...a)
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
            blunderThresholdMP: 150,
            approximate: { positions: false, matches: false }
        });
    });

    it('marque « ≈ » une estimation et laisse les blunders inconnus', async () => {
        estimate.mockResolvedValue({ position_count: 5000000, match_count: 40, approximate: ['positions'] });
        await refreshLibraryCounts();
        const c = get(libraryCountsStore);
        expect(must(c).blunders).toBeNull();
        expect(formatCount(must(c).positions, must(c).approximate.positions)).toBe('≈ 5000000');
        expect(formatCount(must(c).blunders)).toBe('?');
        expect(formatCount(must(c).matches, must(c).approximate.matches)).toBe('40');
    });

    it('une réponse périmée n écrase pas une plus récente', async () => {
        let /** @type {((v: unknown) => void) | undefined} */ resolveOld;
        estimate.mockImplementationOnce(() => new Promise((r) => (resolveOld = r)));
        const old = refreshLibraryCounts();
        await vi.waitFor(() => expect(estimate).toHaveBeenCalledTimes(1));
        estimate.mockResolvedValueOnce({ position_count: 2064, match_count: 11, blunder_count: 87, approximate: [] });
        await refreshLibraryCounts();
        resolveOld?.({ position_count: 0, match_count: 0, blunder_count: 0, approximate: [] });
        await old;
        expect(get(libraryCountsStore)?.positions).toBe(2064);
    });
});
