/**
 * ankiService.setAside.test.js — the three gestures that take a card out of a
 * session without grading it (G.14, #242): suspend, bury, remove.
 *
 * What they must NOT do is tell the scheduler anything. A card set aside is
 * not a card answered, and routing them through reviewCard — the obvious
 * shortcut, since it already advances — would record a grade for a card the
 * user explicitly declined to grade.
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
    GetAnkiDeckPositions: vi.fn(() => Promise.resolve([])),
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
vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn(), perf: (_n, f) => f() } }));

import * as db from '../../wailsjs/go/database/Database.js';
import { suspendCard, buryCard, removeCard } from '../services/ankiService.js';
import { ankiReviewCardStore } from '../stores/ankiStore.js';

const deck = { id: 7, name: 'deck' };
const card = { card: { id: 42 }, position: { id: 100 } };

describe('setting a card aside', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        ankiReviewCardStore.set(card);
        db.GetNextAnkiCard.mockResolvedValue(null);
        db.GetRandomAnkiCard.mockResolvedValue(null);
    });

    test('suspend calls the right backend and never grades', async () => {
        await suspendCard(card, deck);
        expect(db.SetAnkiCardSuspended).toHaveBeenCalledWith(42, true);
        expect(db.ReviewAnkiCard).not.toHaveBeenCalled();
    });

    test('bury calls the right backend and never grades', async () => {
        await buryCard(card, deck);
        expect(db.BuryAnkiCard).toHaveBeenCalledWith(42);
        expect(db.ReviewAnkiCard).not.toHaveBeenCalled();
    });

    test('remove calls the right backend and never grades', async () => {
        await removeCard(card, deck);
        expect(db.RemoveAnkiCard).toHaveBeenCalledWith(42);
        expect(db.ReviewAnkiCard).not.toHaveBeenCalled();
    });

    test('the session moves on to the next due card', async () => {
        const next = { card: { id: 43 }, position: { id: 101 } };
        db.GetNextAnkiCard.mockResolvedValue(next);
        const got = await buryCard(card, deck);
        expect(db.GetNextAnkiCard).toHaveBeenCalledWith(7);
        expect(got).toBe(next);
        expect(get(ankiReviewCardStore)).toBe(next);
    });

    test('an empty queue ends the session rather than leaving the card up', async () => {
        const got = await suspendCard(card, deck);
        expect(got).toBe(null);
        expect(get(ankiReviewCardStore)).toBe(null);
    });

    test('cramming draws at random, never the card just set aside', async () => {
        const next = { card: { id: 44 }, position: { id: 102 } };
        db.GetRandomAnkiCard.mockResolvedValue(next);
        await removeCard(card, deck, { cram: true });
        expect(db.GetRandomAnkiCard).toHaveBeenCalledWith(7, 100);
        expect(db.GetNextAnkiCard).not.toHaveBeenCalled();
    });
});
