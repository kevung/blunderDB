import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const MATCH = { id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2 };

const MOVES = [
    { move_id: 101, game_number: 1, move_number: 1, move_type: 'checker', player_on_roll: 0, position: { dice: [3, 1] }, checker_move: '8/5 6/5' },
    { move_id: 102, game_number: 1, move_number: 2, move_type: 'checker', player_on_roll: 1, position: { dice: [6, 5] }, checker_move: '24/13' },
    { move_id: 103, game_number: 1, move_number: 3, move_type: 'cube', player_on_roll: 0, position: { dice: [0, 0] }, cube_action: 'Double' },
    { move_id: 201, game_number: 2, move_number: 1, move_type: 'checker', player_on_roll: 0, position: { dice: [5, 2] }, checker_move: '13/8 13/11' },
    { move_id: 202, game_number: 2, move_number: 2, move_type: 'checker', player_on_roll: 1, position: { dice: [4, 4] }, checker_move: '24/20(2)' }
];

const GRADES = [
    { move_id: 101, error_mp: 0, grade: '' },
    { move_id: 102, error_mp: 152, grade: 'blunder' },
    { move_id: 103, error_mp: 61, grade: 'error' },
    { move_id: 202, error_mp: 230, grade: 'blunder' }
];

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() => Promise.resolve([MATCH])),
    CountMatches: vi.fn(() => Promise.resolve(1)),
    GetMatchByID: vi.fn(() => Promise.resolve(null)),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    TrashMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve(MOVES)),
    GetGamesByMatch: vi.fn(() =>
        Promise.resolve([
            { game_number: 1, initial_score: [0, 0], winner: 1, points_won: 1 },
            { game_number: 2, initial_score: [1, 0], winner: -1, points_won: 2 }
        ])
    ),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    GetMatchDecisionLosses: vi.fn(() =>
        Promise.resolve([
            { move_id: 101, game_number: 1, move_number: 1, player: 0, decision_type: 'checker', mwc_loss: 0.01, difficulty: 0.004, avoidable: false },
            { move_id: 102, game_number: 1, move_number: 2, player: 1, decision_type: 'checker', mwc_loss: 0.05, difficulty: 0.002, avoidable: false },
            { move_id: 103, game_number: 1, move_number: 3, player: 0, decision_type: 'cube', mwc_loss: 0.02, difficulty: 0.003, avoidable: false }
        ])
    ),
    GetMatchMoveGrades: vi.fn(() => Promise.resolve(GRADES)),
    GetMatchTimeSummary: vi.fn(() => Promise.resolve(null)),
    GetMatchOrigin: vi.fn(() => Promise.resolve(null)),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import { openPanels, PANEL } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { lastVisitedMatchStore, matchContextStore } from '../stores/positionStore.js';
import { ListMatches } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';
import { analysisChallengeStore, revealAnalysis } from '../stores/analysisChallengeStore.js';

async function openTranscript() {
    const view = render(MatchPanel);
    const { container } = view;
    await vi.waitFor(() => expect(ListMatches).toHaveBeenCalledTimes(2));
    await new Promise((r) => setTimeout(r, 0));
    for (let i = 0; i < 6; i++) await tick();
    if (!container.querySelector('tbody tr.selected')) {
        const cell = [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes('Alice'));
        await fireEvent.click(/** @type {Element} */ (cell));
    }
    await vi.waitFor(() => expect(container.querySelector('details.game-section')).not.toBeNull());
    for (let i = 0; i < 4; i++) await tick();
    return container;
}

/** @param {Element} container */
const rowsOf = (container) => [...container.querySelectorAll('details.game-section')[0].querySelectorAll('tr.transcript-row')];
/** @param {Element} row */
const marked = (row) => !!row.querySelector('.grade-mark') || row.classList.contains('graded-error') || row.classList.contains('graded-blunder');

describe('MatchPanel — the challenge hides the Transcript grades', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
        lastVisitedMatchStore.set({ matchID: 7, currentIndex: 0, gameNumber: 1 });
    });
    afterEach(() => {
        cleanup();
        analysisChallengeStore.set(false);
        lastVisitedMatchStore.set({ matchID: null, currentIndex: 0, gameNumber: 1 });
        matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
        openPanels.set(new Set());
    });

    /** @param {Element} container @param {number} i */
    async function goTo(container, i) {
        await fireEvent.click(rowsOf(container)[i]);
        await vi.waitFor(() => expect(rowsOf(container)[i].classList.contains('current-move')).toBe(true));
        await tick();
    }

    test('the difficulty column follows the same rule as the losses', async () => {
        const container = await openTranscript();
        const diff = (/** @type {number} */ i) => rowsOf(container)[i].querySelector('[data-testid="move-difficulty"]')?.textContent?.trim();
        expect(diff(1)).not.toBe('···');
        analysisChallengeStore.set(true);
        await tick();
        expect([0, 1, 2].map(diff)).toEqual(['···', '···', '···']);
        await goTo(container, 1);
        await vi.waitFor(() => expect(diff(1)).toBe('···'));
        revealAnalysis();
        await tick();
        expect(diff(1)).not.toBe('···');
        expect(diff(2)).toBe('···');
        await goTo(container, 2);
        analysisChallengeStore.set(false);
        await tick();
        expect(diff(2)).not.toBe('···');
    });

    test('on, every mark is hidden; the current move shows its own once its analysis is revealed', async () => {
        const container = await openTranscript();
        analysisChallengeStore.set(true);
        await tick();
        expect(rowsOf(container).some(marked)).toBe(false);

        await goTo(container, 1);
        expect(marked(rowsOf(container)[1])).toBe(false);
        revealAnalysis();
        await tick();
        expect(rowsOf(container)[1].querySelector('.grade-mark')?.textContent).toBe('??');
        expect(marked(rowsOf(container)[2]), 'another move stays hidden').toBe(false);
    });

    test('A, then B, then back to A: A is hidden again', async () => {
        const container = await openTranscript();
        analysisChallengeStore.set(true);
        await goTo(container, 1);
        revealAnalysis();
        await tick();
        expect(marked(rowsOf(container)[1])).toBe(true);
        await goTo(container, 2);
        await goTo(container, 1);
        expect(marked(rowsOf(container)[1])).toBe(false);
    });

    test('off again, everything shows', async () => {
        const container = await openTranscript();
        analysisChallengeStore.set(true);
        await tick();
        analysisChallengeStore.set(false);
        await tick();
        expect(marked(rowsOf(container)[1])).toBe(true);
        expect(marked(rowsOf(container)[2])).toBe(true);
    });
});
