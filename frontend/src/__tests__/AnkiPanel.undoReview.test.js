/**
 * AnkiPanel.undoReview.test.js — « Annuler » takes back the last grade of the
 * sitting: it names the card just graded, shows it again, and says so when the
 * backend has nothing to undo.
 */
import { must } from './helpers/must.js';
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateAnkiDeck: vi.fn(),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([])),
    UpdateAnkiDeck: vi.fn(),
    UpdateAnkiDeckParams: vi.fn(() => Promise.resolve()),
    DeleteAnkiDeck: vi.fn(),
    SyncAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeckWithPositions: vi.fn(() => Promise.resolve()),
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ newCount: 5, learningCount: 0, reviewCount: 0, totalCount: 5, dueCount: 4 })),
    CountAnkiDeckPositions: vi.fn(() => Promise.resolve(2)),
    ListAnkiDeckPositionIDs: vi.fn((/** @type {number} */ _deck, offset = 0, limit = 0) => Promise.resolve([10, 11].slice(offset, limit > 0 ? offset + limit : undefined))),
    IndexOfAnkiDeckPosition: vi.fn((/** @type {number} */ _deck, /** @type {number} */ id) => Promise.resolve([10, 11].indexOf(id))),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    UndoAnkiReview: vi.fn(() => Promise.resolve(null)),
    GetLinkedAnkiCard: vi.fn(() => Promise.resolve(null)),
    ResetAnkiDeck: vi.fn(),
    GetAnkiDeckRetention: vi.fn(() => Promise.resolve({ sampleSize: 0, observedRetention: 0, targetRetention: 0.9 })),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));
vi.mock('../services/positionService.js', () => ({ showPosition: vi.fn(() => Promise.resolve()) }));

import * as db from '../../wailsjs/go/database/Database.js';
import { showPosition } from '../services/positionService.js';
import AnkiPanel from '../components/AnkiPanel.svelte';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiDeckStatsStore, ankiViewModeStore, ankiReviewActionStore, ankiPausedSessionStore } from '../stores/ankiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarTextStore } from '../stores/uiStore.js';

const POSITION = { id: 10, dice: [6, 1], cube: { value: 0, owner: -1 }, score: [-1, -1], player_on_roll: 0 };
const CARD = (id) => ({ card: { id, state: 0 }, position: { ...POSITION, id: 10 + id } });

const deckWith = (sessionLimit) => ({
    id: 1,
    name: 'Blunders',
    description: '',
    sourceType: 'collection',
    sourceId: 7,
    cardCount: 5,
    newCount: 5,
    dueCount: 4,
    requestRetention: 0.9,
    maximumInterval: 365,
    enableFuzz: true,
    sessionLimit
});

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('search');
    statusBarTextStore.set('');
    ankiReviewActionStore.set(null);
    ankiPausedSessionStore.set(null);
    ankiDeckStatsStore.set({ newCount: 5, learningCount: 0, reviewCount: 0, totalCount: 5, dueCount: 4 });
});

afterEach(() => {
    cleanup();
    databasePathStore.set('');
    ankiViewModeStore.set('list');
    selectedAnkiDeckStore.set(null);
    ankiReviewCardStore.set(null);
});

async function settle() {
    for (let i = 0; i < 5; i++) await tick();
}

async function reviewing() {
    const deck = deckWith(null);
    ankiDecksStore.set([deck]);
    selectedAnkiDeckStore.set(deck);
    ankiReviewCardStore.set(CARD(1));
    ankiViewModeStore.set('review');
    vi.mocked(db.ReviewAnkiCard).mockResolvedValue(CARD(2));
    const { container } = render(AnkiPanel);
    await settle();
    const undo = /** @type {HTMLButtonElement} */ (must(container.querySelector('[data-testid="anki-undo-review"]')));
    return { container, undo };
}

describe('undoing a grade', () => {
    test('is unavailable before any grade of the sitting', async () => {
        const { undo } = await reviewing();
        expect(undo.disabled).toBe(true);
    });

    test('names the card just graded and shows it again', async () => {
        const { container, undo } = await reviewing();
        await fireEvent.click(container.querySelectorAll('.btn-rating')[2]);
        await settle();
        expect(must(get(ankiReviewCardStore)).card.id).toBe(2);
        expect(undo.disabled).toBe(false);

        vi.mocked(db.UndoAnkiReview).mockResolvedValueOnce(CARD(1));
        await fireEvent.click(undo);
        await settle();

        expect(db.UndoAnkiReview).toHaveBeenCalledWith(1);
        expect(must(get(ankiReviewCardStore)).card.id).toBe(1);
        expect(showPosition).toHaveBeenLastCalledWith(expect.objectContaining({ id: 11 }));
        expect(get(statusBarTextStore).i18nKey).toBe('anki.reviewUndone');
        expect(undo.disabled).toBe(true);
    });

    test('the u key undoes too', async () => {
        const { container } = await reviewing();
        await fireEvent.click(container.querySelectorAll('.btn-rating')[2]);
        await settle();
        vi.mocked(db.UndoAnkiReview).mockResolvedValueOnce(CARD(1));
        ankiReviewActionStore.set('undo');
        await settle();
        expect(db.UndoAnkiReview).toHaveBeenCalledWith(1);
    });

    test('the named refusal says there is nothing to undo', async () => {
        const { container, undo } = await reviewing();
        await fireEvent.click(container.querySelectorAll('.btn-rating')[2]);
        await settle();
        vi.mocked(db.UndoAnkiReview).mockRejectedValueOnce('sqlite: undo review of anki card 1: storage: conflict: the card has no review to undo');
        await fireEvent.click(undo);
        await settle();
        expect(get(statusBarTextStore).i18nKey).toBe('anki.nothingToUndo');
        expect(must(get(ankiReviewCardStore)).card.id).toBe(2);
        expect(undo.disabled).toBe(true);
    });
});
