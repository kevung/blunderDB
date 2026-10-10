/**
 * matchListStore.test.js — the shared match list: first page and total from the
 * database, the filter and the sort passed down as SQL options, further pages
 * appended, one row refreshed in place.
 */
import { describe, test, expect, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(),
    CountMatches: vi.fn(),
    GetMatchByID: vi.fn()
}));

import { createMatchList, sqlSortKey } from '../stores/matchListStore.js';

const rows = (from, /** @type {number} */ n) => Array.from({ length: n }, (_, i) => ({ id: from + i, player1_name: `P${from + i}`, player2_name: 'Q', pr: 0, mwc_loss: 0 }));

function make(total = 1200, pageSize = 500) {
    const list = vi.fn(async ({ Offset, Limit }) => rows(Offset + 1, Limit === 0 ? total : Math.min(Limit, total - Offset)));
    const count = vi.fn(async () => total);
    const getByID = vi.fn(async (id) => ({ id, player1_name: 'Renamed', pr: 99 }));
    return { list, count, getByID, store: createMatchList({ list, count, getByID, pageSize }) };
}

describe('sqlSortKey', () => {
    test('maps a column and direction to the storage key', () => {
        expect(sqlSortKey(null, 'asc')).toBe('');
        expect(sqlSortKey('date', 'desc')).toBe('date');
        expect(sqlSortKey('date', 'asc')).toBe('date_asc');
        expect(sqlSortKey('player1', 'desc')).toBe('player1_desc');
        expect(sqlSortKey('pr', 'asc')).toBeNull();
    });
});

describe('createMatchList', () => {
    test('reload fetches the first page and the total, not the whole list', async () => {
        const { store, list } = make();
        await store.reload();
        const s = get(store);
        expect(s.rows).toHaveLength(500);
        expect(s.total).toBe(1200);
        expect(list).toHaveBeenCalledWith({ Text: '', Sort: '', Limit: 500, Offset: 0 });
    });

    test('loadMore appends the next page until the total is reached', async () => {
        const { store } = make();
        await store.reload();
        await store.loadMore();
        await store.loadMore();
        expect(get(store).rows).toHaveLength(1200);
        expect(get(store).rows[1199].id).toBe(1200);
        await store.loadMore();
        expect(get(store).rows).toHaveLength(1200);
    });

    test('the filter and the sort go to the database and reload from the start', async () => {
        const { store, list, count } = make();
        await store.reload();
        await store.setText('alice');
        expect(list).toHaveBeenLastCalledWith({ Text: 'alice', Sort: '', Limit: 500, Offset: 0 });
        expect(count).toHaveBeenLastCalledWith(expect.objectContaining({ Text: 'alice' }));
        await store.setSort('length', 'desc');
        expect(list).toHaveBeenLastCalledWith({ Text: 'alice', Sort: 'length_desc', Limit: 500, Offset: 0 });
    });

    test('a computed column is ordered here, over the whole filtered list', async () => {
        const { store, list } = make(10);
        await store.setSort('pr', 'asc');
        expect(list).toHaveBeenLastCalledWith({ Text: '', Sort: '', Limit: 0, Offset: 0 });
        expect(get(store).rows).toHaveLength(10);
    });

    test('refreshRow replaces one row and keeps the others and the badges', async () => {
        const { store } = make(3);
        await store.reload();
        const before = get(store).rows;
        await store.refreshRow(2);
        const after = get(store).rows;
        expect(after[1].player1_name).toBe('Renamed');
        expect(after[1].pr).toBe(0);
        expect(after[0]).toBe(before[0]);
        expect(after[2]).toBe(before[2]);
    });

    test('an older answer never overwrites a newer one', async () => {
        let release;
        const slow = new Promise((resolve) => (release = resolve));
        const list = vi
            .fn()
            .mockImplementationOnce(async () => (await slow, rows(1, 2)))
            .mockImplementation(async () => rows(100, 1));
        const store = createMatchList({ list, count: async () => 1, getByID: vi.fn(), pageSize: 10 });
        const first = store.reload();
        await store.setText('x');
        release();
        await first;
        expect(get(store).rows.map((r) => r.id)).toEqual([100]);
    });
});
