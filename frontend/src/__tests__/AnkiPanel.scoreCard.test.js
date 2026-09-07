/**
 * AnkiPanel.scoreCard.test.js — reviewing a card that asks about a score
 * (ADR-0042).
 *
 * The question is the score, shown where a position card shows its number;
 * the answer is the whole sheet, behind the same single mask as any other
 * card (ADR-0025). And it is the SAME sheet the Training tab draws — one
 * component, two hosts — with one difference the ADR is explicit about: no
 * ticking of faults here, because Anki schedules a memory and does not
 * measure a calculation.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateAnkiDeck: vi.fn(),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([])),
    UpdateAnkiDeck: vi.fn(),
    UpdateAnkiDeckParams: vi.fn(),
    DeleteAnkiDeck: vi.fn(),
    SyncAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeckWithPositions: vi.fn(() => Promise.resolve()),
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ newCount: 36, learningCount: 0, reviewCount: 0, totalCount: 36, dueCount: 36 })),
    GetAnkiDeckPositions: vi.fn(() => Promise.resolve([])),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    ResetAnkiDeck: vi.fn(),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import AnkiPanel from '../components/AnkiPanel.svelte';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiViewModeStore, ankiReviewActionStore, ankiPausedSessionStore, ankiAnswerShownStore } from '../stores/ankiStore.js';
import { analysisStore, selectedMoveStore, emptyAnalysis } from '../stores/analysisStore.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore } from '../stores/uiStore.js';

const DECK = { id: 1, name: 'Fiches de score', description: '', sourceType: 'scores', sourceId: 0, cardCount: 36, newCount: 36, dueCount: 36 };

function reviewingScore(key) {
    ankiDecksStore.set([DECK]);
    selectedAnkiDeckStore.set(DECK);
    ankiReviewCardStore.set({ card: { id: 5, state: 0, kind: 'score', key, positionId: 0 }, position: { id: 0 } });
    ankiViewModeStore.set('review');
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('search');
    positionsStore.set([]);
    positionStore.set(null);
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

describe('a review card that is a score', () => {
    test('names the score as the question and hides the sheet behind one mask', async () => {
        reviewingScore('3:5');
        const { container } = render(AnkiPanel);
        await settle();

        expect(container.querySelector('.review-position-id').textContent).toContain('3/5');
        expect(container.querySelector('.answer-masked')).not.toBeNull();
        expect(container.querySelector('.score-card')).toBeNull();
    });

    test('reveals the whole sheet on one gesture, and it cannot be ticked', async () => {
        reviewingScore('3:5');
        const { container } = render(AnkiPanel);
        await settle();

        await fireEvent.click(container.querySelector('.answer-masked'));
        await settle();

        const sheet = container.querySelector('.score-card');
        expect(sheet).not.toBeNull();
        const cells = sheet.querySelectorAll('button.number-cell');
        expect(cells.length).toBeGreaterThan(0);
        // Locked: revealed numbers, and not one of them clickable — the
        // faults are the Training tab's business (ADR-0042 rule 3).
        for (const cell of cells) expect(cell.disabled).toBe(true);
    });

    test('says so plainly when the key is not a score, rather than drawing an empty sheet', async () => {
        reviewingScore('nonsense');
        const { container } = render(AnkiPanel);
        await settle();

        expect(container.querySelector('.answer-absent')).not.toBeNull();
        expect(container.querySelector('.answer-masked')).toBeNull();
    });
});
