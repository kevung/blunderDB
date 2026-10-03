/**
 * MatchPanel.confirmDelete.test.js
 *
 * Deleting a match asks through the themed dialog (confirmAction), and only a
 * confirmed answer reaches the backend.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { answerConfirm } from './confirmHelper.js';
import { confirmModalStore, resolveConfirm } from '../services/confirmService.js';

const MATCH = { id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2 };
const MOVES = [];

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
            { game_number: 1, initial_score: [0, 0], winner: 1, points_won: 1 },
            { game_number: 2, initial_score: [1, 0], winner: -1, points_won: 2 }
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

import { openPanels, PANEL } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { GetAllMatches, DeleteMatch, GetMatchMovePositions } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';

describe('MatchPanel — deleting a match', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
    });
    afterEach(() => {
        cleanup();
        openPanels.set(new Set());
    });

    test('is confirmed through the dialog; refusing deletes nothing', async () => {
        const { container } = render(MatchPanel);
        await vi.waitFor(() => expect(GetAllMatches).toHaveBeenCalled());
        const del = await vi.waitFor(() => {
            const b = container.querySelector('button.icon-btn.delete');
            expect(b).not.toBeNull();
            return b;
        });

        await fireEvent.click(del);
        const message = await answerConfirm(false);
        expect(message).toContain('Alice');
        expect(DeleteMatch).not.toHaveBeenCalled();

        await fireEvent.click(del);
        await answerConfirm(true);
        await vi.waitFor(() => expect(DeleteMatch).toHaveBeenCalledWith(7));
    });

    test('Enter on the confirmation does not also open the match', async () => {
        const { container } = render(MatchPanel);
        await vi.waitFor(() => expect(GetAllMatches).toHaveBeenCalled());
        const row = await vi.waitFor(() => {
            const r = container.querySelector('tbody tr');
            expect(r).not.toBeNull();
            return r;
        });
        await fireEvent.click(row);
        const del = container.querySelector('button.icon-btn.delete');
        await fireEvent.click(del);
        await vi.waitFor(() => expect(get(confirmModalStore)).not.toBeNull());
        const loadsBefore = GetMatchMovePositions.mock.calls.length;
        await fireEvent.keyDown(document.body, { key: 'Enter' });
        await new Promise((r) => setTimeout(r, 50));
        expect(GetMatchMovePositions.mock.calls.length).toBe(loadsBefore);
        resolveConfirm(false);
    });
});
