/**
 * positionList.test.js
 *
 * The browsed list is an id list plus a window cache: a miss loads the
 * window around the index in one loader call, sequential browsing prefetches
 * ahead once per half-window, the cache is bounded (LRU), and bulk reads go
 * around the cache in batches.
 */

import { describe, test, expect, vi } from 'vitest';
import { get } from 'svelte/store';
import { createPositionList } from '../stores/positionList.js';

const pos = (id) => ({ id, board: { bearoff: [id, id] } });

function makeList(options = {}, { known } = {}) {
    const loader = vi.fn(async (ids) => ids.filter((id) => !known || known.has(id)).map(pos));
    const list = createPositionList({ loader, ...options });
    return { list, loader };
}

const idsFrom = (from, to) => Array.from({ length: to - from + 1 }, (_, i) => from + i);

describe('list value and lookups', () => {
    test('starts empty and publishes { ids, length }', () => {
        const { list } = makeList();
        expect(get(list)).toEqual({ ids: [], length: 0 });
        expect(list.idAt(0)).toBeUndefined();
        expect(list.indexOf(1)).toBe(-1);
    });

    test('setIds publishes the ids; idAt / indexOf read them', () => {
        const { list } = makeList();
        list.setIds([30, 10, 20]);
        expect(get(list)).toEqual({ ids: [30, 10, 20], length: 3 });
        expect(list.idAt(1)).toBe(10);
        expect(list.idAt(3)).toBeUndefined();
        expect(list.idAt(-1)).toBeUndefined();
        expect(list.indexOf(20)).toBe(2);
        expect(list.indexOf(99)).toBe(-1);
    });

    test('set(positions) keeps the ids and seeds the cache, so no loader call is needed', async () => {
        const { list, loader } = makeList();
        list.set([pos(5), pos(6)]);
        expect(get(list).ids).toEqual([5, 6]);
        expect(list.peek(1)).toEqual(pos(6));
        await expect(list.getPosition(0)).resolves.toEqual(pos(5));
        expect(loader).not.toHaveBeenCalled();
    });

    test('set() seeds at most cacheSize positions, index 0 kept in preference', () => {
        const { list } = makeList({ cacheSize: 3 });
        list.set([pos(1), pos(2), pos(3), pos(4), pos(5)]);
        expect(get(list).length).toBe(5);
        expect(list.peek(0)).toEqual(pos(1));
        expect(list.peek(2)).toEqual(pos(3));
        expect(list.peek(3)).toBeUndefined();
        expect(list.cacheSize).toBe(3);
    });
});

describe('getPosition', () => {
    test('out of bounds resolves to null without calling the loader', async () => {
        const { list, loader } = makeList();
        await expect(list.getPosition(0)).resolves.toBeNull();
        list.setIds([1, 2]);
        await expect(list.getPosition(-1)).resolves.toBeNull();
        await expect(list.getPosition(2)).resolves.toBeNull();
        await expect(list.getPosition(1.5)).resolves.toBeNull();
        expect(loader).not.toHaveBeenCalled();
    });

    test('a miss loads the window around the index in one call', async () => {
        const { list, loader } = makeList({ windowSize: 50 });
        list.setIds(idsFrom(1, 1000));

        await expect(list.getPosition(500)).resolves.toEqual(pos(501));

        expect(loader).toHaveBeenCalledTimes(1);
        expect(loader.mock.calls[0][0]).toEqual(idsFrom(451, 551));
        expect(list.peek(450)).toEqual(pos(451));
        expect(list.peek(550)).toEqual(pos(551));
        expect(list.peek(551)).toBeUndefined();
    });

    test('the window is clamped to the list', async () => {
        const { list, loader } = makeList({ windowSize: 50 });
        list.setIds(idsFrom(1, 300));

        await list.getPosition(299);
        expect(loader.mock.calls[0][0]).toEqual(idsFrom(250, 300));

        await list.getPosition(0);
        expect(loader.mock.calls[1][0]).toEqual(idsFrom(1, 51));
    });

    test('sequential browsing costs one loader call per half-window, never one per step', async () => {
        const { list, loader } = makeList({ windowSize: 50 });
        list.setIds(idsFrom(1, 1000));

        await list.getPosition(0);
        expect(loader).toHaveBeenCalledTimes(1);

        for (let i = 1; i <= 25; i++) {
            await expect(list.getPosition(i)).resolves.toEqual(pos(i + 1));
        }
        expect(loader, 'steps inside the loaded window need nothing').toHaveBeenCalledTimes(1);

        for (let i = 26; i < 100; i++) {
            await expect(list.getPosition(i)).resolves.toEqual(pos(i + 1));
        }
        // Prefetches at 26, 52 and 78: each brings the next 50 ids ahead.
        expect(loader).toHaveBeenCalledTimes(4);
        expect(loader.mock.calls[1][0]).toEqual(idsFrom(52, 77));
        expect(loader.mock.calls[1][0].length).toBeLessThanOrEqual(50);
    });

    test('browsing backwards prefetches behind the index', async () => {
        const { list, loader } = makeList({ windowSize: 50 });
        list.setIds(idsFrom(1, 1000));

        await list.getPosition(999);
        expect(loader.mock.calls[0][0]).toEqual(idsFrom(950, 1000));
        for (let i = 998; i >= 974; i--) await list.getPosition(i);
        expect(loader).toHaveBeenCalledTimes(1);
        await list.getPosition(973);
        expect(loader).toHaveBeenCalledTimes(2);
        expect(loader.mock.calls[1][0]).toEqual(idsFrom(924, 949));
    });

    test('concurrent requests inside one window share its loader call', async () => {
        const { list, loader } = makeList({ windowSize: 5 });
        list.setIds(idsFrom(1, 100));

        const [a, b, c] = await Promise.all([list.getPosition(10), list.getPosition(11), list.getPosition(12)]);
        expect([a, b, c]).toEqual([pos(11), pos(12), pos(13)]);
        expect(loader).toHaveBeenCalledTimes(1);
        expect(loader.mock.calls[0][0]).toEqual(idsFrom(6, 16));
    });

    test('an id the loader does not return resolves to null, is not asked again, and the rest of the window is kept', async () => {
        const { list, loader } = makeList({ windowSize: 2 }, { known: new Set([1, 2, 4, 5]) });
        list.setIds([1, 2, 3, 4, 5]);

        await expect(list.getPosition(2)).resolves.toBeNull();
        await expect(list.getPosition(3)).resolves.toEqual(pos(4));
        await expect(list.getPosition(2)).resolves.toBeNull();
        expect(loader).toHaveBeenCalledTimes(1);

        // A reset forgets the absence too (the position may have come back).
        list.setIds([1, 2, 3, 4, 5], { reset: true });
        await list.getPosition(2);
        expect(loader).toHaveBeenCalledTimes(2);
    });

    test('a loader failure rejects and leaves the ids retryable', async () => {
        const loader = vi
            .fn()
            .mockRejectedValueOnce(new Error('bridge down'))
            .mockImplementation(async (ids) => ids.map(pos));
        const list = createPositionList({ loader, windowSize: 2 });
        list.setIds([1, 2, 3]);

        await expect(list.getPosition(1)).rejects.toThrow('bridge down');
        await expect(list.getPosition(1)).resolves.toEqual(pos(2));
        expect(loader).toHaveBeenCalledTimes(2);
    });
});

describe('cache bounds', () => {
    test('evicts the least recently used positions beyond cacheSize and reloads them on demand', async () => {
        const { list, loader } = makeList({ windowSize: 2, cacheSize: 10 });
        list.setIds(idsFrom(1, 1000));

        await list.getPosition(0); // ids 1..3
        for (let i = 100; i <= 900; i += 100) await list.getPosition(i); // 5 ids each
        expect(list.cacheSize).toBeLessThanOrEqual(10);
        expect(list.peek(0), 'the oldest window was evicted').toBeUndefined();
        expect(list.peek(900), 'the newest window is kept').toEqual(pos(901));

        const calls = loader.mock.calls.length;
        await expect(list.getPosition(0)).resolves.toEqual(pos(1));
        expect(loader).toHaveBeenCalledTimes(calls + 1);
    });

    test('setIds keeps the cache by default and drops it with reset', async () => {
        const { list, loader } = makeList({ windowSize: 2 });
        list.setIds([1, 2, 3]);
        await list.getPosition(0);
        expect(loader).toHaveBeenCalledTimes(1);

        list.setIds([3, 2, 1]);
        expect(list.peek(2)).toEqual(pos(1));
        await list.getPosition(2);
        expect(loader, 'a reordered list reuses the cached positions').toHaveBeenCalledTimes(1);

        list.setIds([1, 2, 3], { reset: true });
        expect(list.peek(0)).toBeUndefined();
        await list.getPosition(0);
        expect(loader).toHaveBeenCalledTimes(2);
    });

    test('upsert and invalidate edit single entries', async () => {
        const { list } = makeList();
        list.set([pos(1), pos(2)]);
        list.upsert({ id: 2, edited: true });
        expect(list.peek(1)).toEqual({ id: 2, edited: true });
        list.invalidate(2);
        expect(list.peek(1)).toBeUndefined();
        expect(list.peek(0)).toEqual(pos(1));
        list.invalidate();
        expect(list.peek(0)).toBeUndefined();
    });
});

describe('getPositions / getAllPositions', () => {
    test('returns the range in list order, loading the missing ids in batches, without touching the cache', async () => {
        const { list, loader } = makeList({ batchSize: 3 });
        list.setIds([8, 7, 6, 5, 4, 3, 2, 1]);
        list.upsert(pos(7));

        const got = await list.getAllPositions();
        expect(got.map((p) => p.id)).toEqual([8, 7, 6, 5, 4, 3, 2, 1]);
        expect(loader).toHaveBeenCalledTimes(3);
        expect(loader.mock.calls.map((c) => c[0])).toEqual([[8, 6, 5], [4, 3, 2], [1]]);
        expect(list.cacheSize, 'bulk reads do not evict the browsing window').toBe(1);
    });

    test('clamps the range and skips ids the loader does not return', async () => {
        const { list } = makeList({}, { known: new Set([1, 3]) });
        list.setIds([1, 2, 3]);
        const got = await list.getPositions(-5, 50);
        expect(got.map((p) => p.id)).toEqual([1, 3]);
        await expect(list.getPositions(2, 1)).resolves.toEqual([]);
    });
});
