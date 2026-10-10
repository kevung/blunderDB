/** The Match panel's list restricted to the matches another view opened (a Stats corpus row). */
import { describe, test, expect, vi } from 'vitest';
import { get } from 'svelte/store';
import { createMatchList } from '../stores/matchListStore.js';

describe('matchListStore › setIDs', () => {
    test('lists and counts exactly those matches, then all of them again', async () => {
        const list = vi.fn(async () => [{ id: 3 }, { id: 5 }]);
        const count = vi.fn(async () => 2);
        const store = createMatchList({ list, count, getByID: vi.fn() });
        await store.setIDs([3, 5]);
        expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ IDs: [3, 5] }));
        expect(count).toHaveBeenLastCalledWith(expect.objectContaining({ IDs: [3, 5] }));
        expect(get(store).ids).toEqual([3, 5]);
        expect(get(store).total).toBe(2);
        await store.setIDs(null);
        expect(/** @type {any} */ (list.mock.lastCall)?.[0]).not.toHaveProperty('IDs');
        expect(get(store).ids).toBeNull();
    });
});
