/**
 * ankiService.scoreCard.test.js — a card can ask about a score rather than
 * about a position (ADR-0042).
 *
 * The three things the service has to get right about such a card: it says
 * what kind it is, it reads the score out of the key rather than out of a
 * table of its own, and it touches NEITHER the board nor the position stores
 * — a score card leaves the board exactly as the user left it, where showing
 * the previous card's position would be the very lie showCard exists to stop.
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
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ dueCount: 1, totalCount: 36 })),
    GetAnkiDeckRetention: vi.fn(() => Promise.resolve(null)),
    GetAnkiDeckPositions: vi.fn(() => Promise.resolve([])),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetLinkedAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    SetAnkiCardSuspended: vi.fn(() => Promise.resolve()),
    BuryAnkiCard: vi.fn(() => Promise.resolve()),
    RemoveAnkiCard: vi.fn(() => Promise.resolve()),
    ResetAnkiDeck: vi.fn(() => Promise.resolve()),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([]))
}));
vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn(), perf: (_n, f) => f() } }));

import * as db from '../../wailsjs/go/database/Database.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { SOURCE_SCORES, isScoreCard, scoreCardAways, sourceLabel, showCard, createDeck, syncDeckCards, nextCramCard } from '../services/ankiService.js';

const scoreCard = (key) => ({ card: { id: 1, state: 0, kind: 'score', key, positionId: 0 }, position: { id: 0 } });
const positionCard = () => ({ card: { id: 2, state: 0, kind: 'position', key: '10', positionId: 10 }, position: { id: 10, board: [] } });

beforeEach(() => {
    vi.clearAllMocks();
    positionStore.set(null);
    positionsStore.set([]);
    currentPositionIndexStore.set(-1);
});

describe('a card that asks about a score', () => {
    test('is told apart by its kind, never by a missing position', () => {
        expect(isScoreCard(scoreCard('3:5'))).toBe(true);
        expect(isScoreCard(positionCard())).toBe(false);
        expect(isScoreCard(null)).toBe(false);
    });

    test('reads its two aways off the key, smaller first', () => {
        expect(scoreCardAways(scoreCard('3:5'))).toEqual([3, 5]);
        expect(scoreCardAways(scoreCard('5:3'))).toEqual([3, 5]);
        expect(scoreCardAways(scoreCard('2:2'))).toEqual([2, 2]);
    });

    test('reports an unreadable key rather than guessing a score', () => {
        expect(scoreCardAways(scoreCard(''))).toBeNull();
        expect(scoreCardAways(scoreCard('three:five'))).toBeNull();
        expect(scoreCardAways({ card: {} })).toBeNull();
    });

    test('leaves the board alone when it is shown', async () => {
        positionStore.set({ id: 99 });
        await showCard(scoreCard('3:5'));
        expect(get(positionStore)).toEqual({ id: 99 });
        expect(get(currentPositionIndexStore)).toBe(-1);
    });

    test('excludes no position from the next cram draw', async () => {
        await nextCramCard({ id: 7 }, scoreCard('3:5'));
        expect(db.GetRandomAnkiCard).toHaveBeenCalledWith(7, 0);
    });
});

describe('a deck of score sheets', () => {
    test('is created without a source of its own and filled by the backend', async () => {
        await createDeck({ name: 'Scores', sourceType: SOURCE_SCORES, sourceId: 0 });
        expect(db.CreateAnkiDeck).toHaveBeenCalledWith('Scores', '', SOURCE_SCORES, 0, '');
        // The 36 cards are stated by Sync, not by the panel: the user enters
        // no score (ADR-0042 rule 2).
        expect(db.SyncAnkiDeck).toHaveBeenCalledWith(42);
        expect(db.SyncAnkiDeckWithPositions).not.toHaveBeenCalled();
    });

    test('resyncs through the same route as any other deck', async () => {
        await syncDeckCards({ id: 3, sourceType: SOURCE_SCORES, sourceCommand: '' });
        expect(db.SyncAnkiDeck).toHaveBeenCalledWith(3);
    });

    test('names itself in the deck list', () => {
        expect(sourceLabel({ sourceType: SOURCE_SCORES })).toBe('Score sheets');
    });
});
