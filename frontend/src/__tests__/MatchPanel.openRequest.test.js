/**
 * MatchPanel.openRequest.test.js
 *
 * The command palette (#287) opens a match by asking the match panel for it:
 * the panel, once its list is loaded, opens it as a double-click on its row
 * would — its own moves loaded first, never those of the match last shown.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { get } from 'svelte/store';

const MATCH = { id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2 };

// Two games, two moves each.
const MOVES = [
    { game_number: 1, move_number: 1, position_id: 11, player: 1, dice: '31', checker_move: '8/5 6/5' },
    { game_number: 1, move_number: 2, position_id: 12, player: 2, dice: '65', checker_move: '24/13' },
    { game_number: 2, move_number: 1, position_id: 21, player: 1, dice: '52', checker_move: '13/8 13/11' },
    { game_number: 2, move_number: 2, position_id: 22, player: 2, dice: '44', checker_move: '24/20(2)' }
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
    GetMatchMoveGrades: vi.fn(() => Promise.resolve([])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import { openPanels, PANEL, matchOpenRequestStore, activeTabStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { lastVisitedMatchStore, matchContextStore } from '../stores/positionStore.js';
import { GetMatchMovePositions, SaveLastVisitedPosition } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';

describe('MatchPanel — a match requested from the command palette', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
        lastVisitedMatchStore.set({ matchID: null, currentIndex: 0, gameNumber: 1 });
    });
    afterEach(() => {
        cleanup();
        matchOpenRequestStore.set(null);
        matchContextStore.set({ isMatchMode: false, matchID: null, currentIndex: 0 });
        openPanels.set(new Set());
    });

    test('opens it, on its own moves, and consumes the request', async () => {
        matchOpenRequestStore.set(7);
        render(MatchPanel);
        await vi.waitFor(() => expect(SaveLastVisitedPosition).toHaveBeenCalled());
        expect(GetMatchMovePositions).toHaveBeenCalledWith(7);
        expect(SaveLastVisitedPosition).toHaveBeenCalledWith(7, 0);
        expect(get(matchOpenRequestStore)).toBeNull();
        expect(get(activeTabStore)).toBe('analysis');
    });

    test('a match gone since the palette listed it is only reported', async () => {
        matchOpenRequestStore.set(999);
        render(MatchPanel);
        await vi.waitFor(() => expect(get(matchOpenRequestStore)).toBeNull());
        expect(SaveLastVisitedPosition).not.toHaveBeenCalled();
    });
});
