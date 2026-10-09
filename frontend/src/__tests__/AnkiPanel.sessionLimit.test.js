/**
 * AnkiPanel.sessionLimit.test.js — the panel's side of the session limit
 * (ADR-0026 rules 3 and 4): the sitting stops at the limit and SAYS it stopped
 * there, the study button is inactive when the limit serves nothing, and cram
 * ignores the limit entirely.
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

describe('the study button', () => {
    test('is inactive on a deck limited to zero, cards due or not', async () => {
        ankiDecksStore.set([deckWith(0)]);
        selectedAnkiDeckStore.set(deckWith(0));
        const { container } = render(AnkiPanel);
        await settle();
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (must(container.querySelector('[data-testid="anki-study"]'))).disabled).toBe(true);
        // Cram is not bounded by the setting, so it stays available.
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (must(container.querySelector('[data-testid="anki-cram"]'))).disabled).toBe(false);
    });

    test('is active on a deck with a positive limit', async () => {
        ankiDecksStore.set([deckWith(2)]);
        selectedAnkiDeckStore.set(deckWith(2));
        const { container } = render(AnkiPanel);
        await settle();
        expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (must(container.querySelector('[data-testid="anki-study"]'))).disabled).toBe(false);
    });
});

describe('reaching the limit', () => {
    test('ends the sitting and says the limit ended it, not the queue', async () => {
        const deck = deckWith(1);
        ankiDecksStore.set([deck]);
        selectedAnkiDeckStore.set(deck);
        ankiReviewCardStore.set(CARD(1));
        ankiViewModeStore.set('review');
        // A card remains due: the queue is not what stops the session.
        vi.mocked(db.ReviewAnkiCard).mockResolvedValue(CARD(2));

        const { container } = render(AnkiPanel);
        await settle();
        await fireEvent.click(container.querySelectorAll('.btn-rating')[2]); // "Good"
        await settle();

        expect(get(ankiViewModeStore)).toBe('list');
        // The status bar holds a tMsg descriptor, so the KEY is what to assert:
        // it must be the limit's own message, never the ordinary completion one
        // that would claim the queue ran dry.
        const said = get(statusBarTextStore);
        expect(said.i18nKey).toBe('anki.sessionLimitReached');
        expect(said.i18nParams.count).toBe(1);
    });

    test('a session without a limit runs on while cards remain', async () => {
        const deck = deckWith(null);
        ankiDecksStore.set([deck]);
        selectedAnkiDeckStore.set(deck);
        ankiReviewCardStore.set(CARD(1));
        ankiViewModeStore.set('review');
        vi.mocked(db.ReviewAnkiCard).mockResolvedValue(CARD(2));

        const { container } = render(AnkiPanel);
        await settle();
        await fireEvent.click(container.querySelectorAll('.btn-rating')[2]);
        await settle();

        expect(get(ankiViewModeStore)).toBe('review');
    });
});

describe('grading faster than the board loads', () => {
    test('a grade given while the previous card is still loading counts both', async () => {
        const deck = deckWith(null);
        ankiDecksStore.set([deck]);
        selectedAnkiDeckStore.set(deck);
        ankiReviewCardStore.set(CARD(1));
        ankiViewModeStore.set('review');
        vi.mocked(db.ReviewAnkiCard).mockImplementation((/** @type {number} */ id) => Promise.resolve(id === 1 ? CARD(2) : null));
        // The second card is on screen, its board still loading.
        /** @type {(v?: unknown) => void} */
        let release = () => {};
        vi.mocked(showPosition).mockImplementationOnce(() => new Promise((resolve) => (release = resolve)));

        const { container } = render(AnkiPanel);
        await settle();
        await fireEvent.click(container.querySelectorAll('.btn-rating')[2]);
        await settle();
        expect(get(ankiReviewCardStore)?.card.id).toBe(2);
        await fireEvent.click(container.querySelectorAll('.btn-rating')[0]);
        await settle();
        release();
        await settle();

        const said = get(statusBarTextStore);
        expect(said.i18nKey).toBe('anki.reviewComplete');
        expect(said.i18nParams.count).toBe(2);
    });
});

describe('the setting in the deck settings view', () => {
    test('opens unchecked on a deck without a limit, and saves null', async () => {
        const deck = deckWith(null);
        ankiDecksStore.set([deck]);
        selectedAnkiDeckStore.set(deck);
        const { container } = render(AnkiPanel);
        await settle();
        // Through the gear: that is what loads the deck's own values into the
        // form. Setting the view mode by hand would test the initial state of
        // the component instead.
        await fireEvent.click(container.querySelector('[data-testid="anki-settings"]'));
        await settle();

        const checkboxes = container.querySelectorAll('.settings-row input[type="checkbox"]');
        const limited = checkboxes[checkboxes.length - 1];
        expect(/** @type {HTMLInputElement} */ (limited).checked).toBe(false);

        await fireEvent.click(container.querySelector('.settings-actions .btn-primary'));
        await settle();
        expect(db.UpdateAnkiDeckParams).toHaveBeenCalledWith(1, 0.9, 365, true, null);
    });

    test('opens checked on a limited deck and saves the number', async () => {
        const deck = deckWith(7);
        ankiDecksStore.set([deck]);
        selectedAnkiDeckStore.set(deck);
        const { container } = render(AnkiPanel);
        await settle();
        await fireEvent.click(container.querySelector('[data-testid="anki-settings"]'));
        await settle();

        const checkboxes = container.querySelectorAll('.settings-row input[type="checkbox"]');
        expect(/** @type {HTMLInputElement} */ (checkboxes[checkboxes.length - 1]).checked).toBe(true);

        await fireEvent.click(container.querySelector('.settings-actions .btn-primary'));
        await settle();
        expect(db.UpdateAnkiDeckParams).toHaveBeenCalledWith(1, 0.9, 365, true, 7);
    });
});
