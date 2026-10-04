/**
 * MatchPanel.budget.test.js — a library of fifty thousand matches costs a bounded window of
 * rows, and the keyboard still walks it.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() =>
        Promise.resolve(Array.from({ length: 50000 }, (_, i) => ({ id: i + 1, player1_name: `P${i + 1}`, player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2 })))
    ),
    CountMatches: vi.fn(() => Promise.resolve(50000)),
    GetMatchByID: vi.fn(() => Promise.resolve(null)),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    DeleteMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    GetGamesByMatch: vi.fn(() => Promise.resolve([])),
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
import { ListMatches } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';

describe('MatchPanel — fifty thousand matches', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
    });
    afterEach(cleanup);

    test('only a window of rows is in the DOM, and j selects the first one', async () => {
        const started = performance.now();
        const { container } = render(MatchPanel);
        await vi.waitFor(() => expect(ListMatches).toHaveBeenCalledTimes(2));
        await new Promise((resolve) => setTimeout(resolve, 0));
        for (let i = 0; i < 4; i++) await tick();

        const rows = container.querySelectorAll('tbody tr:not(.spacer)').length;
        expect(rows).toBeGreaterThan(0);
        expect(rows).toBeLessThan(300);
        expect(performance.now() - started, 'mount and first paint of 50 000 matches').toBeLessThan(5000);

        await fireEvent.keyDown(document, { key: 'j' });
        await vi.waitFor(() => expect(container.querySelector('tbody tr.selected')).not.toBeNull());
    });
});
