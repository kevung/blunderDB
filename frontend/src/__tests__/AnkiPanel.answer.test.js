/**
 * AnkiPanel.answer.test.js — the masked answer of a review card (ADR-0025).
 *
 * A card asks one question and gets one grade: the whole answer is hidden
 * behind one stand-in, revealed by one gesture, and grading stays available
 * throughout. A card whose position has no stored analysis says so plainly
 * instead of offering a mask that reveals nothing.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateAnkiDeck: vi.fn(),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([])),
    UpdateAnkiDeck: vi.fn(),
    UpdateAnkiDeckParams: vi.fn(),
    DeleteAnkiDeck: vi.fn(),
    SyncAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeckWithPositions: vi.fn(() => Promise.resolve()),
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ newCount: 1, learningCount: 0, reviewCount: 0, totalCount: 1, dueCount: 1 })),
    GetAnkiDeckPositions: vi.fn(() => Promise.resolve([{ id: 10 }])),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    ResetAnkiDeck: vi.fn(),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    LoadPositionsByFilters: vi.fn(() => Promise.resolve([])),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import AnkiPanel from '../components/AnkiPanel.svelte';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiViewModeStore, ankiReviewActionStore, ankiPausedSessionStore, ankiAnswerShownStore } from '../stores/ankiStore.js';
import { analysisStore, selectedMoveStore, emptyAnalysis } from '../stores/analysisStore.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore } from '../stores/uiStore.js';

const DECK = { id: 1, name: 'Alpha', description: '', sourceType: 'collection', sourceId: 7, cardCount: 1, newCount: 1, dueCount: 1 };

const CHECKER_ANALYSIS = {
    ...emptyAnalysis(),
    positionId: 10,
    analysisType: 'CheckerMove',
    checkerAnalysis: {
        moves: [
            { move: '13/7 8/7', equity: 0.512, error: 0, analysisDepth: '3-ply' },
            { move: '24/18 13/12', equity: 0.401, error: -0.111, analysisDepth: '3-ply' }
        ]
    },
    playedMoves: ['24/18 13/12']
};

const CUBE_ANALYSIS = {
    ...emptyAnalysis(),
    positionId: 10,
    analysisType: 'DoublingCube',
    doublingCubeAnalysis: {
        ...emptyAnalysis().doublingCubeAnalysis,
        playerWinChances: 72.5,
        opponentWinChances: 27.5,
        cubefulNoDoubleEquity: 0.62,
        cubefulDoubleTakeEquity: 0.78,
        cubefulDoublePassEquity: 1.0,
        bestCubeAction: 'Double, take'
    }
};

const POSITION = { id: 10, dice: [6, 1], player_on_roll: 0, cube: { value: 0, owner: -1 }, score: [-1, -1] };

function reviewing(analysis) {
    ankiDecksStore.set([DECK]);
    selectedAnkiDeckStore.set(DECK);
    ankiReviewCardStore.set({ card: { id: 5, state: 0 }, position: POSITION });
    ankiViewModeStore.set('review');
    positionStore.set(POSITION);
    analysisStore.set(analysis);
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('search');
    positionsStore.set([{ id: 10 }]);
    ankiReviewActionStore.set(null);
    ankiPausedSessionStore.set(null);
    ankiAnswerShownStore.set(false);
    selectedMoveStore.set(null);
    analysisStore.set(emptyAnalysis());
});

afterEach(() => {
    cleanup();
    databasePathStore.set('');
    ankiViewModeStore.set('list');
});

async function settle() {
    for (let i = 0; i < 4; i++) await tick();
}

describe('the answer of a review card', () => {
    test('starts masked, and the analysis is nowhere in the DOM', async () => {
        reviewing(CHECKER_ANALYSIS);
        const { container } = render(AnkiPanel);
        await settle();

        expect(container.querySelector('.answer-masked')).not.toBeNull();
        expect(container.querySelector('.checker-table')).toBeNull();
        expect(container.textContent).not.toContain('13/7');
    });

    test('clicking the stand-in reveals the whole analysis at once', async () => {
        reviewing(CHECKER_ANALYSIS);
        const { container } = render(AnkiPanel);
        await settle();

        await fireEvent.click(container.querySelector('.answer-masked'));
        await settle();

        expect(container.querySelector('.answer-masked')).toBeNull();
        expect(container.textContent).toContain('13/7');
        expect(container.textContent).toContain('24/18');
    });

    test('a cube card reveals the facts and the verdict', async () => {
        reviewing(CUBE_ANALYSIS);
        const { container } = render(AnkiPanel);
        await settle();

        expect(container.querySelector('.answer-masked')).not.toBeNull();
        await fireEvent.click(container.querySelector('.answer-masked'));
        await settle();

        expect(container.querySelectorAll('table').length).toBeGreaterThan(0);
        expect(container.textContent).toContain('72.5');
    });

    test('grading stays available while the answer is masked', async () => {
        reviewing(CHECKER_ANALYSIS);
        const { container } = render(AnkiPanel);
        await settle();

        expect(container.querySelector('.answer-masked')).not.toBeNull();
        const buttons = container.querySelectorAll('.btn-rating');
        expect(buttons).toHaveLength(4);
        for (const b of buttons) expect(b.disabled).toBe(false);
    });

    test('the grading strip sits above the answer, and only the answer scrolls', async () => {
        reviewing(CHECKER_ANALYSIS);
        const { container } = render(AnkiPanel);
        await settle();

        const body = container.querySelector('.review-body');
        const children = [...body.children];
        expect(children[0].classList.contains('review-strip')).toBe(true);
        expect(children[1].classList.contains('review-answer')).toBe(true);
    });

    test('a position without a stored analysis says so, unmasked', async () => {
        reviewing({ ...emptyAnalysis(), positionId: 10, analysisType: 'CheckerMove' });
        const { container } = render(AnkiPanel);
        await settle();

        expect(container.querySelector('.answer-masked')).toBeNull();
        expect(container.querySelector('.answer-absent')).not.toBeNull();
    });

    test('the reveal survives a remount — a tab round-trip does not re-hide it', async () => {
        reviewing(CHECKER_ANALYSIS);
        const first = render(AnkiPanel);
        await settle();
        await fireEvent.click(first.container.querySelector('.answer-masked'));
        await settle();
        expect(get(ankiAnswerShownStore)).toBe(true);
        cleanup();

        const { container } = render(AnkiPanel);
        await settle();
        expect(container.querySelector('.answer-masked')).toBeNull();
        expect(container.textContent).toContain('13/7');
    });

    test('clicking a revealed move shows it on the board, and leaving the review drops it', async () => {
        reviewing(CHECKER_ANALYSIS);
        ankiAnswerShownStore.set(true);
        const { container } = render(AnkiPanel);
        await settle();

        await fireEvent.click(container.querySelectorAll('.checker-table tbody tr')[0]);
        await settle();
        expect(get(selectedMoveStore)).toBe('13/7 8/7');

        await fireEvent.click(container.querySelector('.btn-back'));
        await settle();
        expect(get(selectedMoveStore)).toBeNull();
        expect(get(ankiAnswerShownStore)).toBe(false);
    });
});
