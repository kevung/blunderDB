/**
 * ankiService.test.js — the deck logic behind AnkiPanel: parsing a deck's
 * stored search, syncing decks with their source, and walking a review or
 * cram session, with the Wails bindings mocked.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateAnkiDeck: vi.fn(() => Promise.resolve(42)),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([])),
    UpdateAnkiDeckParams: vi.fn(() => Promise.resolve()),
    DeleteAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeckWithPositions: vi.fn(() => Promise.resolve()),
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ dueCount: 1, totalCount: 3 })),
    GetAnkiDeckPositions: vi.fn(() => Promise.resolve([{ id: 10 }, { id: 11 }])),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    ResetAnkiDeck: vi.fn(() => Promise.resolve()),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    LoadPositionsByFilters: vi.fn(() => Promise.resolve([]))
}));
vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn(), perf: (_n, f) => f() } }));

import * as db from '../../wailsjs/go/database/Database.js';
import { logger } from '../utils/logger.js';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiDeckStatsStore, ankiViewModeStore } from '../stores/ankiStore.js';
import { positionsStore, positionStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import {
    parseSourceCommand,
    buildSearchSource,
    mergeIds,
    stateLabel,
    sourceLabel,
    resumedSessionCount,
    canStudy,
    canCram,
    resolveSearchDeckIds,
    syncDeckCards,
    syncAllDecksAndReload,
    createDeck,
    deleteDeck,
    selectDeck,
    saveDeckParams,
    startSession,
    reviewCard,
    nextCramCard
} from '../services/ankiService.js';

const card = (id, posId) => ({ card: { id, state: 0 }, position: { id: posId, board: [] } });

beforeEach(() => {
    vi.clearAllMocks();
    ankiDecksStore.set([]);
    selectedAnkiDeckStore.set(null);
    ankiReviewCardStore.set(null);
    ankiDeckStatsStore.set(null);
    ankiViewModeStore.set('list');
    positionsStore.set([]);
    positionStore.set(null);
    currentPositionIndexStore.set(-1);
});

describe('pure helpers', () => {
    test('parseSourceCommand reads the JSON document and the legacy comma list', () => {
        expect(parseSourceCommand(JSON.stringify({ command: 's e>0.1', position: '{"cube":1}', ids: [1, 2] }))).toEqual({ command: 's e>0.1', position: { cube: 1 }, ids: [1, 2] });
        expect(parseSourceCommand(JSON.stringify({ ids: [5] }))).toEqual({ command: null, position: {}, ids: [5] });
        expect(parseSourceCommand('3, 4,x,5')).toEqual({ command: null, position: {}, ids: [3, 4, 5] });
        expect(parseSourceCommand('')).toEqual({ command: null, position: {}, ids: [] });
    });

    test('buildSearchSource keeps the command and board when there was a search, the ids otherwise', () => {
        expect(JSON.parse(buildSearchSource({ command: 's', position: '{}' }, [1]))).toEqual({ command: 's', position: '{}', ids: [1] });
        expect(JSON.parse(buildSearchSource(null, [1, 2]))).toEqual({ ids: [1, 2] });
    });

    test('mergeIds keeps search order, appends stored ids, drops duplicates', () => {
        expect(mergeIds([3, 1], [1, 2, 3])).toEqual([3, 1, 2]);
    });

    test('stateLabel names the four FSRS states', () => {
        expect([0, 1, 2, 3, 9].map(stateLabel)).toEqual(['New', 'Learning', 'Review', 'Relearning', '?']);
    });

    test('sourceLabel names the collection, else the stored command, else "Search"', () => {
        const colls = [{ id: 7, name: 'Openings' }];
        expect(sourceLabel({ sourceType: 'collection', sourceId: 7 }, colls)).toBe('Openings');
        expect(sourceLabel({ sourceType: 'collection', sourceId: 8 }, colls)).toBe('Collection #8');
        expect(sourceLabel({ sourceType: 'search', sourceCommand: JSON.stringify({ command: 's b>0.05' }) })).toBe('s b>0.05');
        expect(sourceLabel({ sourceType: 'search', sourceCommand: '1,2' })).toBe('Search');
        expect(sourceLabel({ sourceType: 'search', sourceCommand: '' })).toBe('Search');
    });

    test('resumedSessionCount resumes only a paused FSRS session on the same deck', () => {
        const deck = { id: 1 };
        expect(resumedSessionCount({ deckId: 1, sessionCount: 4 }, deck)).toBe(4);
        expect(resumedSessionCount({ deckId: 2, sessionCount: 4 }, deck)).toBe(0);
        expect(resumedSessionCount({ deckId: 1, sessionCount: 4 }, deck, true)).toBe(0);
        expect(resumedSessionCount(null, deck)).toBe(0);
    });

    test('canStudy needs due cards, canCram only needs cards', () => {
        expect(canStudy({ dueCount: 0, totalCount: 3 })).toBe(false);
        expect(canCram({ dueCount: 0, totalCount: 3 })).toBe(true);
        expect(canStudy({ dueCount: 2, totalCount: 3 })).toBe(true);
        expect(canCram({ dueCount: 0, totalCount: 0 })).toBe(false);
        expect(canStudy(null)).toBe(false);
    });
});

describe('resolveSearchDeckIds', () => {
    test('returns the stored ids when no command was stored', async () => {
        expect(await resolveSearchDeckIds(JSON.stringify({ ids: [4, 5] }))).toEqual([4, 5]);
        expect(db.LoadPositionsByFilters).not.toHaveBeenCalled();
    });

    test('re-runs a bare `s` on the stored board and merges the stored ids in', async () => {
        db.LoadPositionsByFilters.mockResolvedValueOnce([{ id: 9 }, { id: 4 }]);
        const ids = await resolveSearchDeckIds(JSON.stringify({ command: 's', position: '{"cube":1}', ids: [4, 5] }));
        expect(db.LoadPositionsByFilters).toHaveBeenCalledTimes(1);
        expect(ids).toEqual([9, 4, 5]);
    });

    test('a filtered command goes through the command parser', async () => {
        db.LoadPositionsByFilters.mockResolvedValueOnce([{ id: 1 }]);
        const ids = await resolveSearchDeckIds(JSON.stringify({ command: 's e>0.1', position: '{}', ids: [] }));
        expect(ids).toEqual([1]);
        const payload = db.LoadPositionsByFilters.mock.calls[0][0];
        expect(payload).toBeTruthy();
    });

    test('a failing search is logged and yields no ids', async () => {
        db.LoadPositionsByFilters.mockRejectedValueOnce(new Error('boom'));
        expect(await resolveSearchDeckIds(JSON.stringify({ command: 's', position: '{}', ids: [1] }))).toEqual([]);
        expect(logger.error).toHaveBeenCalled();
    });
});

describe('syncing', () => {
    test('a collection deck syncs from its collection', async () => {
        await syncDeckCards({ id: 3, sourceType: 'collection' });
        expect(db.SyncAnkiDeck).toHaveBeenCalledWith(3);
        expect(db.SyncAnkiDeckWithPositions).not.toHaveBeenCalled();
    });

    test('a search deck syncs from its resolved ids, and is left alone when they are empty', async () => {
        await syncDeckCards({ id: 3, sourceType: 'search', sourceCommand: JSON.stringify({ ids: [1, 2] }) });
        expect(db.SyncAnkiDeckWithPositions).toHaveBeenCalledWith(3, [1, 2]);

        await syncDeckCards({ id: 4, sourceType: 'search', sourceCommand: JSON.stringify({ ids: [] }) });
        expect(db.SyncAnkiDeckWithPositions).toHaveBeenCalledTimes(1);
        expect(db.SyncAnkiDeck).not.toHaveBeenCalled();
    });

    test('syncAllDecksAndReload syncs every deck, survives one failing, then reloads the store', async () => {
        const decks = [
            { id: 1, name: 'a', sourceType: 'collection' },
            { id: 2, name: 'b', sourceType: 'collection' }
        ];
        db.GetAllAnkiDecks.mockResolvedValue(decks);
        db.SyncAnkiDeck.mockImplementation((id) => (id === 1 ? Promise.reject(new Error('x')) : Promise.resolve()));
        await syncAllDecksAndReload();
        expect(db.SyncAnkiDeck).toHaveBeenCalledTimes(2);
        expect(logger.error).toHaveBeenCalledWith('Error syncing deck "a":', expect.any(Error));
        expect(get(ankiDecksStore)).toEqual(decks);
    });
});

describe('deck lifecycle', () => {
    test('createDeck from a collection creates, syncs, reloads and returns the id', async () => {
        const id = await createDeck({ name: 'N', sourceType: 'collection', sourceId: 7 });
        expect(id).toBe(42);
        expect(db.CreateAnkiDeck).toHaveBeenCalledWith('N', '', 'collection', 7, '');
        expect(db.SyncAnkiDeck).toHaveBeenCalledWith(42);
        expect(db.GetAllAnkiDecks).toHaveBeenCalled();
    });

    test('createDeck from a search stores the search and syncs the matched ids', async () => {
        db.LoadPositionsByFilters.mockResolvedValueOnce([{ id: 8 }]);
        await createDeck({ name: 'S', sourceType: 'search', sourceId: 99, lastSearch: { command: 's', position: '{}' }, positionIds: [8, 9] });
        const [, , type, sourceId, sourceCommand] = db.CreateAnkiDeck.mock.calls[0];
        expect(type).toBe('search');
        expect(sourceId).toBe(0);
        expect(JSON.parse(sourceCommand)).toEqual({ command: 's', position: '{}', ids: [8, 9] });
        expect(db.SyncAnkiDeckWithPositions).toHaveBeenCalledWith(42, [8, 9]);
    });

    test('deleteDeck clears the selection only when the deleted deck was selected', async () => {
        selectedAnkiDeckStore.set({ id: 5 });
        ankiViewModeStore.set('review');
        await deleteDeck(6);
        expect(get(selectedAnkiDeckStore)).toEqual({ id: 5 });
        expect(get(ankiViewModeStore)).toBe('review');
        await deleteDeck(5);
        expect(get(selectedAnkiDeckStore)).toBeNull();
        expect(get(ankiViewModeStore)).toBe('list');
    });

    test('selectDeck loads the stats and makes the deck positions the counted list', async () => {
        await selectDeck({ id: 5 });
        expect(get(selectedAnkiDeckStore)).toEqual({ id: 5 });
        expect(get(ankiDeckStatsStore)).toEqual({ dueCount: 1, totalCount: 3 });
        expect(get(positionsStore).ids).toEqual([10, 11]);
        expect(get(currentPositionIndexStore)).toBe(0);
    });

    test('saveDeckParams updates, reloads and refreshes the selected deck object', async () => {
        selectedAnkiDeckStore.set({ id: 5, requestRetention: 0.9 });
        db.GetAllAnkiDecks.mockResolvedValueOnce([{ id: 5, requestRetention: 0.95 }]);
        await saveDeckParams(5, { requestRetention: 0.95, maximumInterval: 100, enableFuzz: false });
        expect(db.UpdateAnkiDeckParams).toHaveBeenCalledWith(5, 0.95, 100, false, null);
        expect(get(selectedAnkiDeckStore)).toEqual({ id: 5, requestRetention: 0.95 });
    });
});

describe('sessions', () => {
    const deck = { id: 5, sourceType: 'collection' };

    test('startSession syncs, loads positions, draws the next due card and shows it', async () => {
        db.GetNextAnkiCard.mockResolvedValueOnce(card(1, 11));
        const first = await startSession(deck);
        expect(db.SyncAnkiDeck).toHaveBeenCalledWith(5);
        expect(first.card.id).toBe(1);
        expect(get(ankiReviewCardStore)).toEqual(first);
        expect(get(positionStore)).toEqual({ id: 11, board: [] });
        expect(get(positionStore)).not.toBe(first.position); // a copy: the board edits its own object
        expect(get(currentPositionIndexStore)).toBe(1);
    });

    test('startSession returns null when nothing is due, and draws randomly when cramming', async () => {
        expect(await startSession(deck)).toBeNull();
        expect(db.GetRandomAnkiCard).not.toHaveBeenCalled();

        db.GetRandomAnkiCard.mockResolvedValueOnce(card(2, 10));
        const c = await startSession(deck, { cram: true });
        expect(db.GetRandomAnkiCard).toHaveBeenCalledWith(5, 0);
        expect(c.card.id).toBe(2);
    });

    test('reviewCard grades the card and shows the next one; the session ends on null', async () => {
        db.ReviewAnkiCard.mockResolvedValueOnce(card(2, 10));
        const next = await reviewCard(card(1, 11), 3);
        expect(db.ReviewAnkiCard).toHaveBeenCalledWith(1, 3);
        expect(next.card.id).toBe(2);
        expect(get(ankiReviewCardStore)).toEqual(next);

        expect(await reviewCard(next, 4)).toBeNull();
        expect(get(ankiReviewCardStore)).toBeNull();
    });

    test('nextCramCard never schedules and skips the card just shown', async () => {
        db.GetRandomAnkiCard.mockResolvedValueOnce(card(3, 10));
        const next = await nextCramCard(deck, card(1, 11));
        expect(db.GetRandomAnkiCard).toHaveBeenCalledWith(5, 11);
        expect(db.ReviewAnkiCard).not.toHaveBeenCalled();
        expect(next.card.id).toBe(3);
    });
});
