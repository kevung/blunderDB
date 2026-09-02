/**
 * reorder.test.js — the one-step and drag moves behind the ▲/▼ buttons and
 * the pointer drag of the tournament and collection tables.
 */
import { describe, test, expect, vi } from 'vitest';
import { moveItem, moveUp, moveDown, createReorder } from '../utils/reorder.js';

vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn() } }));
import { logger } from '../utils/logger.js';

describe('moveItem', () => {
    test('moves the item so it lands at `to`, like dragReorder (from, to)', () => {
        expect(moveItem(['a', 'b', 'c', 'd'], 0, 2)).toEqual(['b', 'c', 'a', 'd']);
        expect(moveItem(['a', 'b', 'c', 'd'], 3, 1)).toEqual(['a', 'd', 'b', 'c']);
    });

    test('returns a copy and leaves the input untouched', () => {
        const list = ['a', 'b'];
        const next = moveItem(list, 0, 1);
        expect(next).toEqual(['b', 'a']);
        expect(list).toEqual(['a', 'b']);
    });

    test('returns null for a no-op or an impossible move', () => {
        expect(moveItem(['a', 'b'], 1, 1)).toBeNull();
        expect(moveItem(['a', 'b'], -1, 0)).toBeNull();
        expect(moveItem(['a', 'b'], 0, 2)).toBeNull();
        expect(moveItem(null, 0, 1)).toBeNull();
        expect(moveItem([], 0, 0)).toBeNull();
    });
});

describe('moveUp / moveDown', () => {
    test('swap with the neighbour', () => {
        expect(moveUp(['a', 'b', 'c'], 2)).toEqual(['a', 'c', 'b']);
        expect(moveDown(['a', 'b', 'c'], 0)).toEqual(['b', 'a', 'c']);
    });

    test('are null at the edges', () => {
        expect(moveUp(['a', 'b'], 0)).toBeNull();
        expect(moveDown(['a', 'b'], 1)).toBeNull();
    });
});

describe('createReorder', () => {
    function setup(list) {
        const set = vi.fn((next) => (list = next));
        const persist = vi.fn(() => Promise.resolve());
        const order = createReorder({ get: () => list, set, persist, label: 'things' });
        return { order, set, persist, current: () => list };
    }

    test('moveUp installs the new order locally then persists it', async () => {
        const { order, set, persist, current } = setup(['a', 'b', 'c']);
        expect(await order.moveUp(1)).toBe(true);
        expect(current()).toEqual(['b', 'a', 'c']);
        expect(set).toHaveBeenCalledWith(['b', 'a', 'c'], 1, 0);
        expect(persist).toHaveBeenCalledWith(['b', 'a', 'c']);
    });

    test('moveDown and reorder hand set() the from/to pair so a selection can follow', async () => {
        const { order, set } = setup(['a', 'b', 'c']);
        await order.moveDown(0);
        expect(set).toHaveBeenLastCalledWith(['b', 'a', 'c'], 0, 1);
        await order.reorder(0, 2);
        expect(set).toHaveBeenLastCalledWith(['a', 'c', 'b'], 0, 2);
    });

    test('an impossible move neither sets nor persists', async () => {
        const { order, set, persist } = setup(['a', 'b']);
        expect(await order.moveUp(0)).toBe(false);
        expect(await order.moveDown(1)).toBe(false);
        expect(await order.reorder(1, 1)).toBe(false);
        expect(set).not.toHaveBeenCalled();
        expect(persist).not.toHaveBeenCalled();
    });

    test('get() returning null disables reordering (no tournament selected)', async () => {
        const set = vi.fn();
        const order = createReorder({ get: () => null, set, persist: vi.fn() });
        expect(await order.moveDown(0)).toBe(false);
        expect(set).not.toHaveBeenCalled();
    });

    test('a persistence failure is logged and the local order kept', async () => {
        let list = ['a', 'b'];
        const order = createReorder({
            get: () => list,
            set: (next) => (list = next),
            persist: () => Promise.reject(new Error('db down')),
            label: 'matches'
        });
        expect(await order.moveDown(0)).toBe(true);
        expect(list).toEqual(['b', 'a']);
        expect(logger.error).toHaveBeenCalledWith('Error reordering matches:', expect.any(Error));
    });
});
