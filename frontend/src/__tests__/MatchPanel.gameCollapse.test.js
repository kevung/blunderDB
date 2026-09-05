/**
 * MatchPanel.gameCollapse.test.js
 *
 * Reported: with a match selected, game 1 of the transcript cannot be
 * collapsed — clicking its header reopens it at once — while the later games
 * collapse normally.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

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
import { GetAllMatches } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';

async function settle() {
    await vi.waitFor(() => expect(GetAllMatches).toHaveBeenCalledTimes(2));
    await new Promise((r) => setTimeout(r, 0));
    for (let i = 0; i < 6; i++) await tick();
}

describe('MatchPanel — every game of the transcript collapses, game 1 included', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
        // The match was last left on its very first move: that is what makes
        // game 1 the one seeded open.
        lastVisitedMatchStore.set({ matchID: 7, currentIndex: 0, gameNumber: 1 });
    });
    afterEach(() => {
        cleanup();
        lastVisitedMatchStore.set({ matchID: null, currentIndex: 0, gameNumber: 1 });
        matchContextStore.set({ isMatchMode: false, matchID: null, currentIndex: 0 });
        openPanels.set(new Set());
    });

    test('collapsing game 1 keeps it collapsed', async () => {
        const { container } = render(MatchPanel);
        await settle();

        // lastVisitedMatch already selects this match on open; clicking it again
        // would toggle the selection off.
        if (!container.querySelector('tbody tr.selected')) {
            const cell = [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes('Alice'));
            await fireEvent.click(cell);
        }
        await vi.waitFor(() => expect(container.querySelector('.detail-tabs')).not.toBeNull());
        for (let i = 0; i < 4; i++) await tick();

        // The transcript is one of the detail tabs; make sure it is the one showing.
        const transcriptTab = [...container.querySelectorAll('.detail-tab')].find((b) => !b.classList.contains('export-mat-btn') && !b.classList.contains('enter-match-btn'));
        if (!transcriptTab.classList.contains('active')) {
            await fireEvent.click(transcriptTab);
        }
        await vi.waitFor(() => expect(container.querySelector('details.game-section')).not.toBeNull());
        for (let i = 0; i < 4; i++) await tick();

        const sections = () => [...container.querySelectorAll('details.game-section')];
        expect(sections().length, 'both games are listed').toBe(2);
        const first = sections()[0];
        expect(first.open, 'game 1 is the one seeded open').toBe(true);

        // Collapse it the way the browser does: flip `open`, then fire toggle.
        first.open = false;
        await fireEvent(first, new Event('toggle'));
        for (let i = 0; i < 4; i++) await tick();

        expect(sections()[0].open, 'game 1 must stay collapsed').toBe(false);
    });

    // The real report: the user is reviewing the match. The transcript follows
    // the move being reviewed, and the move being reviewed is in game 1.
    test('collapsing game 1 keeps it collapsed while reviewing the match', async () => {
        const { container } = render(MatchPanel);
        await settle();

        if (!container.querySelector('tbody tr.selected')) {
            const cell = [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes('Alice'));
            await fireEvent.click(cell);
        }
        await vi.waitFor(() => expect(container.querySelector('.detail-tabs')).not.toBeNull());
        for (let i = 0; i < 4; i++) await tick();

        // Enter match mode on the first move — game 1.
        matchContextStore.set({ isMatchMode: true, matchID: 7, currentIndex: 0 });
        await vi.waitFor(() => expect(container.querySelector('details.game-section')).not.toBeNull());
        for (let i = 0; i < 4; i++) await tick();

        const sections = () => [...container.querySelectorAll('details.game-section')];
        const first = sections()[0];
        expect(first.open, 'game 1 holds the move under review').toBe(true);

        first.open = false;
        await fireEvent(first, new Event('toggle'));
        for (let i = 0; i < 6; i++) await tick();

        expect(sections()[0].open, 'game 1 must stay collapsed even under review').toBe(false);

        // And the later game, which holds no reviewed move, collapses too.
        const second = sections()[1];
        second.open = true;
        await fireEvent(second, new Event('toggle'));
        for (let i = 0; i < 4; i++) await tick();
        second.open = false;
        await fireEvent(second, new Event('toggle'));
        for (let i = 0; i < 4; i++) await tick();
        expect(sections()[1].open, 'game 2 collapses, as it always did').toBe(false);
    });

    // The behaviour the reopening exists for must survive: stepping from game
    // 1 into game 2 opens game 2, even if the user had collapsed it.
    test('crossing into another game still opens it', async () => {
        const { container } = render(MatchPanel);
        await settle();

        if (!container.querySelector('tbody tr.selected')) {
            const cell = [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes('Alice'));
            await fireEvent.click(cell);
        }
        await vi.waitFor(() => expect(container.querySelector('.detail-tabs')).not.toBeNull());
        for (let i = 0; i < 4; i++) await tick();

        matchContextStore.set({ isMatchMode: true, matchID: 7, currentIndex: 0 });
        await vi.waitFor(() => expect(container.querySelector('details.game-section')).not.toBeNull());
        for (let i = 0; i < 4; i++) await tick();

        const sections = () => [...container.querySelectorAll('details.game-section')];
        expect(sections()[1].open, 'game 2 starts collapsed').toBe(false);

        // Step forward into game 2 (moves 0-1 are game 1, 2-3 are game 2).
        matchContextStore.set({ isMatchMode: true, matchID: 7, currentIndex: 2 });
        for (let i = 0; i < 6; i++) await tick();

        expect(sections()[1].open, 'stepping into game 2 opens it').toBe(true);
    });
});
