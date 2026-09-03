/**
 * ankiService.showCard.test.js — a review card is displayed through
 * showPosition, the one function that loads the position's analysis.
 *
 * showCard used to set positionStore by hand, so analysisStore kept the
 * analysis of whatever position was browsed last. The Analysis tab stays
 * reachable during a review (the review key guard lets Ctrl combos through),
 * and it showed ANOTHER position's numbers — the reason ADR-0025 rule 1 makes
 * the stored analysis the answer only once this path is fixed.
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
    GetAnkiDeckPositions: vi.fn(() => Promise.resolve([{ id: 10 }, { id: 11 }])),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    ResetAnkiDeck: vi.fn(),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([]))
}));
vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn(), perf: (_n, f) => f() } }));

import * as db from '../../wailsjs/go/database/Database.js';
import { showPosition } from '../services/positionService.js';
import { showCard, startSession, reviewCard } from '../services/ankiService.js';
import { ankiAnswerShownStore } from '../stores/ankiStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';

const card = (id) => ({ card: { id: id * 100, state: 0 }, position: { id } });

beforeEach(() => {
    vi.clearAllMocks();
    positionsStore.set([{ id: 10 }, { id: 11 }]);
    currentPositionIndexStore.set(0);
    ankiAnswerShownStore.set(false);
    selectedMoveStore.set(null);
});

describe('showCard', () => {
    test('displays the card through showPosition, so its analysis is loaded', async () => {
        await showCard(card(11));
        expect(showPosition).toHaveBeenCalledTimes(1);
        expect(showPosition).toHaveBeenCalledWith(expect.objectContaining({ id: 11 }));
    });

    test('points the status bar at the card position', async () => {
        await showCard(card(11));
        expect(get(currentPositionIndexStore)).toBe(1);
    });

    test('a position outside the loaded list leaves the index alone', async () => {
        currentPositionIndexStore.set(1);
        await showCard(card(99));
        expect(get(currentPositionIndexStore)).toBe(1);
    });
});

describe('what resets the reveal', () => {
    // ADR-0025 rule 5: a change of QUESTION re-hides the answer, a change of
    // VIEW does not. AnkiPanel calls showCard again every time the Anki tab
    // regains focus — so putting the reset inside showCard erased the answer
    // of someone who had merely gone to look at the Eval panel.
    test('re-showing the SAME card leaves the answer revealed', async () => {
        ankiAnswerShownStore.set(true);
        selectedMoveStore.set('13/7 8/7');

        await showCard(card(10));

        expect(get(ankiAnswerShownStore)).toBe(true);
        expect(get(selectedMoveStore)).toBe('13/7 8/7');
    });

    test('the next card of a session hides its answer', async () => {
        ankiAnswerShownStore.set(true);
        selectedMoveStore.set('13/7 8/7');
        db.ReviewAnkiCard.mockResolvedValueOnce(card(11));

        await reviewCard(card(10), 3);

        expect(get(ankiAnswerShownStore)).toBe(false);
        expect(get(selectedMoveStore)).toBeNull();
    });

    test('starting a session hides the first card answer', async () => {
        ankiAnswerShownStore.set(true);
        db.GetNextAnkiCard.mockResolvedValueOnce(card(10));

        await startSession({ id: 1 });

        expect(get(ankiAnswerShownStore)).toBe(false);
    });
});

describe('session walking', () => {
    test('the first card of a session is displayed through showPosition', async () => {
        db.GetNextAnkiCard.mockResolvedValueOnce(card(10));
        await startSession({ id: 1 });
        expect(showPosition).toHaveBeenCalledWith(expect.objectContaining({ id: 10 }));
    });

    test('grading a card displays the next one before resolving', async () => {
        db.ReviewAnkiCard.mockResolvedValueOnce(card(11));
        const next = await reviewCard(card(10), 3);
        expect(next.position.id).toBe(11);
        expect(showPosition).toHaveBeenCalledWith(expect.objectContaining({ id: 11 }));
    });

    test('the end of a session displays nothing', async () => {
        db.ReviewAnkiCard.mockResolvedValueOnce(null);
        const next = await reviewCard(card(10), 3);
        expect(next).toBeNull();
        expect(showPosition).not.toHaveBeenCalled();
    });
});
