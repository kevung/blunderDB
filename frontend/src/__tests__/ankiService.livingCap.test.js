/**
 * ankiService.livingCap.test.js — a deck fed by a living collection is
 * resynchronised under a declared ceiling; when the ceiling cuts the query's
 * result short the status bar says so, and stays silent otherwise.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../services/positionService.js', () => ({
    showPosition: vi.fn(() => Promise.resolve())
}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateAnkiDeck: vi.fn(),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([])),
    UpdateAnkiDeckParams: vi.fn(),
    DeleteAnkiDeck: vi.fn(),
    SyncAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeckWithPositions: vi.fn(() => Promise.resolve()),
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ dueCount: 1, totalCount: 2 })),
    CountAnkiDeckPositions: vi.fn(() => Promise.resolve(0)),
    ListAnkiDeckPositionIDs: vi.fn((/** @type {number} */ _deck, offset = 0, limit = 0) => Promise.resolve([].slice(offset, limit > 0 ? offset + limit : undefined))),
    IndexOfAnkiDeckPosition: vi.fn((/** @type {number} */ _deck, /** @type {number} */ id) => Promise.resolve(/** @type {number[]} */ ([]).indexOf(id))),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    SetAnkiCardSuspended: vi.fn(() => Promise.resolve()),
    BuryAnkiCard: vi.fn(() => Promise.resolve()),
    RemoveAnkiCard: vi.fn(() => Promise.resolve()),
    ResetAnkiDeck: vi.fn(),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([]))
}));
vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn(), perf: (/** @type {string} */ _n, /** @type {() => unknown} */ f) => f() } }));

import * as db from '../../wailsjs/go/database/Database.js';
import { syncDeckCards } from '../services/ankiService.js';
import { statusBarTextStore } from '../stores/uiStore.js';

const deck = { id: 3, name: 'Gaffes', sourceType: 'collection', sourceCommand: '' };

describe('a deck fed by a living collection', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        statusBarTextStore.set('');
    });

    test('a truncated evaluation is reported', async () => {
        /** @type {import('vitest').Mock} */ (db.SyncAnkiDeck).mockResolvedValueOnce({ deckId: 3, source: { total: 9000, cap: 5000, truncated: true, positionIds: [] } });
        await syncDeckCards(deck);
        expect(db.SyncAnkiDeck).toHaveBeenCalledWith(3);
        const msg = /** @type {any} */ (get(statusBarTextStore));
        expect(msg).toBeTruthy();
        expect(msg.i18nKey).toBe('collection.livingCapped');
        expect(msg.i18nParams).toMatchObject({ name: 'Gaffes', total: 9000, count: 5000 });
    });

    test('a whole evaluation says nothing', async () => {
        /** @type {import('vitest').Mock} */ (db.SyncAnkiDeck).mockResolvedValueOnce({ deckId: 3, source: { total: 12, cap: 5000, truncated: false, positionIds: [] } });
        await syncDeckCards(deck);
        expect(get(statusBarTextStore)).toBe('');
    });
});
