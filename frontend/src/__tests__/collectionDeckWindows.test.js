/**
 * collectionDeckWindows.test.js — a collection or an Anki deck of a million positions is opened by
 * its count, a first window and ranks: no call asks the backend for the whole list, and the
 * browsed list never holds it.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const N = 1_000_000;
/** @param {number} offset @param {number} limit */
const windowOf = (offset, limit) => Array.from({ length: limit > 0 ? Math.min(limit, N - offset) : N - offset }, (_, i) => offset + i + 1);

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CountCollectionPositions: vi.fn(() => Promise.resolve(N)),
    ListCollectionPositionIDs: vi.fn((/** @type {number} */ _id, offset = 0, limit = 0) => Promise.resolve(windowOf(offset, limit))),
    IndexOfCollectionPosition: vi.fn((/** @type {number} */ _id, /** @type {number} */ position) => Promise.resolve(position - 1)),
    CountAnkiDeckPositions: vi.fn(() => Promise.resolve(N)),
    ListAnkiDeckPositionIDs: vi.fn((/** @type {number} */ _id, offset = 0, limit = 0) => Promise.resolve(windowOf(offset, limit))),
    IndexOfAnkiDeckPosition: vi.fn((/** @type {number} */ _id, /** @type {number} */ position) => Promise.resolve(position - 1)),
    CountAnkiDeckFilteredPositions: vi.fn(() => Promise.resolve(2)),
    ListAnkiDeckFilteredPositionIDs: vi.fn(() => Promise.resolve([4, 9])),
    IndexOfAnkiDeckFilteredPosition: vi.fn(() => Promise.resolve(1)),
    LoadPositionsByIDs: vi.fn((/** @type {number[]} */ ids) => Promise.resolve(ids.map((id) => ({ id, board: {} })))),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ dueCount: 0, totalCount: N })),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([])),
    GetAllCollections: vi.fn(() => Promise.resolve([]))
}));

import * as db from '../../wailsjs/go/database/Database.js';
import { positionsStore, positionStore, collectionSource, deckSource } from '../stores/positionStore.js';
import { handleOpenCollection } from '../services/modeMachine.js';
import { selectDeck } from '../services/ankiService.js';
import { statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore.js';

beforeEach(() => {
    vi.clearAllMocks();
    positionsStore.set([]);
    positionStore.set(null);
    statusBarModeStore.set('NORMAL');
    currentPositionIndexStore.set(-1);
});

/** No call may ask for a window unbounded on both sides, nor one as large as the list. */
const boundedCalls = (/** @type {import('vitest').Mock} */ list) => list.mock.calls.every(([, , limit]) => limit > 0 && limit < 10_000);

describe('a collection of a million positions', () => {
    test('opens on its count and first window, the list holds only its length', async () => {
        await handleOpenCollection({ id: 7, name: 'Big' }, collectionSource(7));

        expect(get(statusBarModeStore)).toBe('COLLECTION');
        expect(get(positionsStore)).toMatchObject({ ids: null, length: N, paged: true });
        expect(get(positionStore).id).toBe(1);
        expect(db.CountCollectionPositions).toHaveBeenCalledWith(7);
        expect(db.ListCollectionPositionIDs).toHaveBeenCalled();
        expect(boundedCalls(/** @type {any} */ (db.ListCollectionPositionIDs))).toBe(true);
    });

    test('reads the rank of a position from the backend, not from a held list', async () => {
        await handleOpenCollection({ id: 7, name: 'Big' }, collectionSource(7));
        expect(await positionsStore.findIndex(654_321)).toBe(654_320);
        expect(db.IndexOfCollectionPosition).toHaveBeenCalledWith(7, 654_321);
    });

    test('an empty collection is refused without touching the list', async () => {
        vi.mocked(db.CountCollectionPositions).mockResolvedValueOnce(0);
        await handleOpenCollection({ id: 8, name: 'Empty' }, collectionSource(8));
        expect(get(statusBarModeStore)).not.toBe('COLLECTION');
        expect(db.ListCollectionPositionIDs).not.toHaveBeenCalled();
    });
});

describe('an Anki deck of a million positions', () => {
    test('selecting it counts and windows its positions, never loads them', async () => {
        await selectDeck({ id: 3 });

        expect(get(positionsStore)).toMatchObject({ ids: null, length: N, paged: true });
        expect(db.CountAnkiDeckPositions).toHaveBeenCalledWith(3);
        expect(boundedCalls(/** @type {any} */ (db.ListAnkiDeckPositionIDs))).toBe(true);
        expect(await positionsStore.findIndex(10)).toBe(9);
        expect(db.IndexOfAnkiDeckPosition).toHaveBeenCalledWith(3, 10);
    });

    test('the deck source reads its three answers from the backend', async () => {
        const source = deckSource(3);
        expect(await source.count()).toBe(N);
        expect(await source.window(5, 2)).toEqual([6, 7]);
        expect(await source.indexOf(8)).toBe(7);
    });

    test('a filtered deck source lists the cards one counter counts', async () => {
        const source = deckSource(3, 'due');
        expect(await source.count()).toBe(2);
        expect(await source.window(0, 0)).toEqual([4, 9]);
        expect(await source.indexOf(9)).toBe(1);
        expect(db.CountAnkiDeckFilteredPositions).toHaveBeenCalledWith(3, 'due');
        expect(db.ListAnkiDeckFilteredPositionIDs).toHaveBeenCalledWith(3, 'due', 0, 0);
        expect(db.IndexOfAnkiDeckFilteredPosition).toHaveBeenCalledWith(3, 'due', 9);
    });
});
