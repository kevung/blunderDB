/**
 * MatchPanel.transcriptGrades.test.js
 *
 * #287 — the Transcript is coloured by gravity: each Move the backend grades
 * (GetMatchMoveGrades, at the library's thresholds) carries its mark in its
 * row, and each game's header counts its marks, so a collapsed game still says
 * where the blunders are.
 */

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
    GetAllMatches: vi.fn(() => Promise.resolve([MATCH])),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    DeleteMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve(MOVES)),
    GetGamesByMatch: vi.fn(() =>
        Promise.resolve([
            { game_number: 1, initial_score: [0, 0], winner: 0, points_won: 1 },
            { game_number: 2, initial_score: [1, 0], winner: 1, points_won: 2 }
        ])
    ),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    GetMatchMoveGrades: vi.fn(() => Promise.resolve(GRADES)),
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
import { libraryCountsStore } from '../stores/libraryCountsStore.js';
import { GetAllMatches, GetMatchMoveGrades } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';

async function openTranscript() {
    const view = render(MatchPanel);
    const { container } = view;
    await vi.waitFor(() => expect(GetAllMatches).toHaveBeenCalledTimes(2));
    await new Promise((r) => setTimeout(r, 0));
    for (let i = 0; i < 6; i++) await tick();
    if (!container.querySelector('tbody tr.selected')) {
        const cell = [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes('Alice'));
        await fireEvent.click(cell);
    }
    await vi.waitFor(() => expect(container.querySelector('details.game-section')).not.toBeNull());
    for (let i = 0; i < 4; i++) await tick();
    return container;
}

describe('MatchPanel — the Transcript carries the grade of every Move (#287)', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
        lastVisitedMatchStore.set({ matchID: 7, currentIndex: 0, gameNumber: 1 });
    });
    afterEach(() => {
        cleanup();
        lastVisitedMatchStore.set({ matchID: null, currentIndex: 0, gameNumber: 1 });
        matchContextStore.set({ isMatchMode: false, matchID: null, currentIndex: 0 });
        openPanels.set(new Set());
    });

    test('rows are marked by their grade, and only the graded ones', async () => {
        const container = await openTranscript();
        const rows = [...container.querySelectorAll('details.game-section')[0].querySelectorAll('tr.transcript-row')];
        expect(rows).toHaveLength(3);

        expect(rows[0].classList.contains('graded-error') || rows[0].classList.contains('graded-blunder'), 'a best play is unmarked').toBe(false);
        expect(rows[0].querySelector('.grade-mark')).toBeNull();

        expect(rows[1].classList.contains('graded-blunder')).toBe(true);
        expect(rows[1].querySelector('.grade-mark').textContent).toBe('??');
        expect(rows[1].querySelector('.grade-mark').getAttribute('title')).toContain('0.152');

        // A cube Move is graded like a checker Move.
        expect(rows[2].classList.contains('graded-error')).toBe(true);
        expect(rows[2].querySelector('.grade-mark').textContent).toBe('?');
    });

    test('each game header counts its marks, a collapsed game included', async () => {
        const container = await openTranscript();
        const sections = [...container.querySelectorAll('details.game-section')];
        expect(sections[1].open, 'game 2 is collapsed').toBe(false);

        const marks = (section) => [...section.querySelectorAll('summary .game-marks')].map((m) => m.textContent.trim());
        expect(marks(sections[0])).toEqual(['1 ??', '1 ?']);
        expect(marks(sections[1])).toEqual(['1 ??']);
    });

    test('a change of the library counter (thresholds moved, import landed) re-reads the grades', async () => {
        await openTranscript();
        const before = GetMatchMoveGrades.mock.calls.length;
        libraryCountsStore.set({ positions: 1, blunders: 2, matches: 1 });
        await vi.waitFor(() => expect(GetMatchMoveGrades.mock.calls.length).toBeGreaterThan(before));
        expect(GetMatchMoveGrades).toHaveBeenLastCalledWith(7);
    });
});
