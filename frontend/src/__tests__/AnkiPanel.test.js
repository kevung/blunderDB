/**
 * AnkiPanel.test.js — a mounted AnkiPanel: the deck list renders from the
 * store, selecting a deck opens its detail, the view-mode store swaps in the
 * review and settings views, and the rating keys routed from App.svelte
 * reach the service.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateAnkiDeck: vi.fn(() => Promise.resolve(1)),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([])),
    UpdateAnkiDeck: vi.fn(() => Promise.resolve()),
    UpdateAnkiDeckParams: vi.fn(() => Promise.resolve()),
    DeleteAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeckWithPositions: vi.fn(() => Promise.resolve()),
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ newCount: 2, learningCount: 0, reviewCount: 1, totalCount: 3, dueCount: 2 })),
    GetAnkiDeckPositions: vi.fn(() => Promise.resolve([{ id: 10 }])),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    ResetAnkiDeck: vi.fn(() => Promise.resolve()),
    GetAllCollections: vi.fn(() => Promise.resolve([{ id: 7, name: 'Openings', positionCount: 3 }])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import * as db from '../../wailsjs/go/database/Database.js';
import AnkiPanel from '../components/AnkiPanel.svelte';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiDeckStatsStore, ankiViewModeStore, ankiReviewActionStore, ankiPausedSessionStore } from '../stores/ankiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore } from '../stores/uiStore.js';

const DECKS = [
    { id: 1, name: 'Alpha', description: 'first', sourceType: 'collection', sourceId: 7, cardCount: 3, newCount: 2, dueCount: 2 },
    { id: 2, name: 'Beta', description: '', sourceType: 'search', sourceCommand: JSON.stringify({ command: 's e>0.1', ids: [1] }), cardCount: 1, newCount: 0, dueCount: 0 }
];

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('search');
    ankiDecksStore.set([]);
    selectedAnkiDeckStore.set(null);
    ankiReviewCardStore.set(null);
    ankiDeckStatsStore.set(null);
    ankiViewModeStore.set('list');
    ankiReviewActionStore.set(null);
    ankiPausedSessionStore.set(null);
});

afterEach(() => {
    cleanup();
    databasePathStore.set('');
});

async function settle() {
    for (let i = 0; i < 4; i++) await tick();
}

describe('AnkiPanel', () => {
    test('lists the decks with their source and counts, or an empty state', async () => {
        const { container } = render(AnkiPanel);
        await settle();
        expect(container.querySelector('.empty-state')).not.toBeNull();

        db.GetAllAnkiDecks.mockResolvedValue(DECKS);
        ankiDecksStore.set(DECKS);
        await settle();
        const rows = container.querySelectorAll('tbody tr');
        expect(rows).toHaveLength(2);
        expect(rows[0].textContent).toContain('Alpha');
        expect(rows[0].querySelector('.source-cell').textContent).toBe('Openings');
        expect(rows[1].querySelector('.source-cell').textContent).toBe('s e>0.1');
        expect(container.querySelector('.empty-state')).toBeNull();
    });

    test('clicking a deck selects it and opens its detail with the stats', async () => {
        db.GetAllAnkiDecks.mockResolvedValue(DECKS);
        const { container } = render(AnkiPanel);
        await settle();

        await fireEvent.click(container.querySelectorAll('tbody tr')[0]);
        await settle();

        expect(get(selectedAnkiDeckStore)).toEqual(DECKS[0]);
        expect(db.GetAnkiDeckStats).toHaveBeenCalledWith(1);
        expect(container.querySelector('tbody tr').classList.contains('selected')).toBe(true);
        const detail = container.querySelector('.deck-detail');
        expect(detail).not.toBeNull();
        expect(detail.querySelectorAll('.stat-number')).toHaveLength(4);
        expect(detail.querySelector('.btn-study').disabled).toBe(false);
    });

    test('the study button is disabled with nothing due; cram only needs cards', async () => {
        ankiDecksStore.set(DECKS);
        selectedAnkiDeckStore.set(DECKS[1]);
        ankiDeckStatsStore.set({ newCount: 0, learningCount: 0, reviewCount: 1, totalCount: 1, dueCount: 0 });
        const { container } = render(AnkiPanel);
        await settle();
        expect(container.querySelector('.btn-study').disabled).toBe(true);
        expect(container.querySelector('.btn-cram').disabled).toBe(false);
    });

    test('the review view shows the current card and routes the rating keys to the service', async () => {
        ankiDecksStore.set(DECKS);
        selectedAnkiDeckStore.set(DECKS[0]);
        ankiReviewCardStore.set({ card: { id: 5, state: 0 }, position: { id: 10 } });
        ankiViewModeStore.set('review');
        const { container } = render(AnkiPanel);
        await settle();

        expect(container.querySelector('.view-title').textContent).toBe('Alpha');
        expect(container.querySelector('.card-state').textContent).toBe('New');
        expect(container.querySelectorAll('.btn-rating')).toHaveLength(4);

        ankiReviewActionStore.set(3);
        await settle();
        expect(db.ReviewAnkiCard).toHaveBeenCalledWith(5, 3);
        // No next card: the session ends and the list comes back.
        expect(get(ankiViewModeStore)).toBe('list');
    });

    test('"back" from the review view pauses the FSRS session on the deck', async () => {
        ankiDecksStore.set(DECKS);
        selectedAnkiDeckStore.set(DECKS[0]);
        ankiReviewCardStore.set({ card: { id: 5, state: 2 }, position: { id: 10 } });
        ankiViewModeStore.set('review');
        render(AnkiPanel);
        await settle();

        ankiReviewActionStore.set('back');
        await settle();
        expect(get(ankiViewModeStore)).toBe('list');
        expect(get(ankiPausedSessionStore)).toEqual({ deckId: 1, sessionCount: 0 });
    });

    test("the settings view opens on the deck's FSRS parameters and saves them", async () => {
        ankiDecksStore.set(DECKS);
        selectedAnkiDeckStore.set({ ...DECKS[0], requestRetention: 0.85, maximumInterval: 365, enableFuzz: false });
        ankiDeckStatsStore.set({ newCount: 2, learningCount: 0, reviewCount: 1, totalCount: 3, dueCount: 2 });
        const { container } = render(AnkiPanel);
        await settle();

        // The gear button is the first outlined button of the detail actions.
        await fireEvent.click(container.querySelector('.detail-actions .btn-outline'));
        await settle();
        expect(get(ankiViewModeStore)).toBe('settings');
        expect(container.querySelector('.settings-row input[type="number"]').value).toBe('0.85');

        await fireEvent.click(container.querySelector('.settings-actions .btn-primary'));
        await settle();
        expect(db.UpdateAnkiDeckParams).toHaveBeenCalledWith(1, 0.85, 365, false, null);
        expect(get(ankiViewModeStore)).toBe('list');
    });
});
