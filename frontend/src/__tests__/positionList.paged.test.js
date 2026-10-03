/**
 * positionList.paged.test.js
 *
 * A paged list (the library) holds its length and the id pages it was asked for, never the whole
 * id list: a library of a million positions is browsed — next, previous, jump to N, first, last,
 * find a position by id — through a source of windows, under a bound on the ids held.
 */

import { describe, test, expect, vi } from 'vitest';
import { get } from 'svelte/store';
import { createPositionList, indexInList, listLength } from '../stores/positionList.js';

const MILLION = 1_000_000;

// A generated library: ids are odd (gaps, as after deletions), computed rather than stored so the
// test itself holds no million-entry array.
const idOf = (i) => 2 * i + 1;

function makeSource(total = MILLION) {
    return {
        count: vi.fn(async () => total),
        window: vi.fn(async (offset, limit) => {
            const out = [];
            for (let i = offset; i < Math.min(total, offset + limit); i++) out.push(idOf(i));
            return out;
        }),
        indexOf: vi.fn(async (id) => (id % 2 === 1 && (id - 1) / 2 < total ? (id - 1) / 2 : -1))
    };
}

const pos = (id) => ({ id });

function makePaged(options = {}) {
    const loader = vi.fn(async (ids) => ids.map(pos));
    const list = createPositionList({ loader, idPageSize: 1000, idPages: 8, ...options });
    return { list, loader };
}

describe('paged list', () => {
    test('setSource publishes the length, not the ids', async () => {
        const { list } = makePaged();
        const source = makeSource();
        await expect(list.setSource(source)).resolves.toBe(MILLION);
        expect(get(list)).toEqual({ ids: null, length: MILLION, paged: true });
        expect(source.count).toHaveBeenCalledTimes(1);
        expect(source.window).not.toHaveBeenCalled();
        expect(list.isPaged()).toBe(true);
        expect(list.isSource(source)).toBe(true);
    });

    test('the last position loads one id page and one window of positions', async () => {
        const { list, loader } = makePaged();
        const source = makeSource();
        await list.setSource(source);
        const last = MILLION - 1;
        await expect(list.getPosition(last)).resolves.toEqual({ id: idOf(last) });
        expect(list.idAt(last)).toBe(idOf(last));
        expect(loader).toHaveBeenCalledTimes(1);
        // The window around the last index spans at most two id pages.
        expect(source.window.mock.calls.length).toBeLessThanOrEqual(2);
        expect(list.heldIds).toBeLessThanOrEqual(2000);
    });

    test('browsing forward reads id pages as it goes, bounded', async () => {
        const { list } = makePaged();
        const source = makeSource();
        await list.setSource(source);
        for (let i = 0; i < 5000; i++) {
            const p = await list.getPosition(i);
            expect(p?.id).toBe(idOf(i));
        }
        // 5 000 positions over pages of 1 000: a handful of page reads, never the list.
        expect(source.window.mock.calls.length).toBeLessThanOrEqual(8);
        expect(list.heldIds).toBeLessThanOrEqual(8 * 1000);
    });

    test('random jumps across a million ids stay under a time and memory budget', async () => {
        const { list } = makePaged();
        await list.setSource(makeSource());
        const started = performance.now();
        let seed = 7;
        for (let n = 0; n < 500; n++) {
            seed = (seed * 48271) % 2147483647;
            const i = seed % MILLION;
            const p = await list.getPosition(i);
            expect(p?.id).toBe(idOf(i));
        }
        expect(performance.now() - started).toBeLessThan(2000);
        expect(list.heldIds).toBeLessThanOrEqual(8 * 1000);
        expect(list.cacheSize).toBeLessThanOrEqual(512);
    });

    test('findIndex asks the source for an id no loaded page holds', async () => {
        const { list } = makePaged();
        const source = makeSource();
        await list.setSource(source);
        expect(list.indexOf(idOf(777_777))).toBe(-1);
        await expect(list.findIndex(idOf(777_777))).resolves.toBe(777_777);
        expect(source.window).not.toHaveBeenCalled();
        await expect(list.findIndex(4)).resolves.toBe(-1);

        await list.getPosition(10);
        source.indexOf.mockClear();
        await expect(list.findIndex(idOf(12))).resolves.toBe(12);
        expect(source.indexOf, 'a loaded page answers without the source').not.toHaveBeenCalled();
    });

    test('resolveIdAt fetches the page of an index not yet loaded', async () => {
        const { list } = makePaged();
        await list.setSource(makeSource());
        expect(list.idAt(123_456)).toBeUndefined();
        await expect(list.resolveIdAt(123_456)).resolves.toBe(idOf(123_456));
        expect(list.idAt(123_456)).toBe(idOf(123_456));
        await expect(list.resolveIdAt(MILLION)).resolves.toBeUndefined();
    });

    test('a page fetched for a replaced list is dropped', async () => {
        const { list } = makePaged();
        const source = makeSource();
        /** @type {(v: number[]) => void} */
        let release = () => {};
        source.window.mockImplementationOnce(() => new Promise((resolve) => (release = resolve)));
        await list.setSource(source);
        const pending = list.ensureIds(0, 0);
        await vi.waitFor(() => expect(source.window).toHaveBeenCalled());
        list.setIds([42, 43]);
        release([1, 3, 5]);
        await pending;
        expect(list.idAt(0)).toBe(42);
        expect(get(list)).toEqual({ ids: [42, 43], length: 2, paged: false });
    });

    test('bulk reads walk the source in batches without evicting the browsed pages', async () => {
        const { list } = makePaged({ batchSize: 500 });
        const source = makeSource(3000);
        await list.setSource(source);
        await list.getPosition(0);
        const all = await list.getAllPositions();
        expect(all.map((p) => p.id)).toEqual(Array.from({ length: 3000 }, (_, i) => idOf(i)));
        expect(list.idAt(0)).toBe(1);
    });

    test('snapshotList / restoreList put a paged list back without reading it', async () => {
        const { list } = makePaged();
        const source = makeSource();
        await list.setSource(source);
        const snap = list.snapshotList();
        expect(listLength(snap)).toBe(MILLION);
        await expect(indexInList(snap, idOf(5))).resolves.toBe(5);

        list.setIds([9, 8]);
        expect(listLength(list.snapshotList())).toBe(2);
        await expect(indexInList(list.snapshotList(), 8)).resolves.toBe(1);

        list.restoreList(snap);
        expect(get(list)).toEqual({ ids: null, length: MILLION, paged: true });
        expect(source.count).toHaveBeenCalledTimes(1);
    });

    test('recount re-reads the length of the library', async () => {
        const { list } = makePaged();
        const source = makeSource(10);
        await list.setSource(source);
        source.count.mockResolvedValueOnce(11);
        await expect(list.recount()).resolves.toBe(11);
        expect(get(list).length).toBe(11);
    });
});

describe('a paged list opened on its first page', () => {
    test('is as long as the page until the count settles it, the page read once', async () => {
        const { list } = makePaged();
        const source = makeSource();
        const first = await source.window(0, list.firstPageSize());

        expect(list.adoptFirstPage(source, first)).toEqual({ length: 1000, exact: false });
        expect(get(list).length).toBe(1000);

        list.resolveLength(source, MILLION);
        expect(get(list).length).toBe(MILLION);
        expect(await list.getPosition(5)).toEqual(pos(idOf(5)));
        expect(source.window).toHaveBeenCalledTimes(1);
    });

    test('a short first page is the whole list', async () => {
        const { list } = makePaged();
        const source = makeSource(3);
        const first = await source.window(0, list.firstPageSize());
        expect(list.adoptFirstPage(source, first)).toEqual({ length: 3, exact: true });
    });

    test('a length settled for another list is ignored', async () => {
        const { list } = makePaged();
        const source = makeSource();
        list.adoptFirstPage(source, await source.window(0, 1000));
        list.adoptFirstPage(makeSource(), await source.window(0, 1000));
        list.resolveLength(source, 5);
        expect(get(list).length).toBe(1000);
    });
    test('a snapshot taken while counting comes back at the settled length', async () => {
        const { list } = makePaged();
        const source = makeSource();
        list.adoptFirstPage(source, await source.window(0, 1000));
        const taken = list.snapshotList();
        list.setIds([5, 6]);
        list.resolveLength(source, MILLION);
        expect(listLength(taken)).toBe(MILLION);
        list.restoreList(taken);
        expect(get(list).length).toBe(MILLION);
    });
});
