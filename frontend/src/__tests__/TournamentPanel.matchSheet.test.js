/**
 * TournamentPanel.matchSheet.test.js
 *
 * One click on a match of the selected tournament shows the Matches panel's own sheet
 * (header, summary, Transcript|Charts|Review|Details|Info|Stats tabs) under the table;
 * a double-click still enters match mode.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';

const T_MATCH = { id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2, pr: 0, mwc_loss: 0 };
const OTHER = { id: 8, player1_name: 'Carol', player2_name: 'Dave', match_length: 5, match_date: '2026-01-16', game_count: 1, pr: 0, mwc_loss: 0 };
const MOVES = [{ game_number: 1, move_number: 1, position_id: 11, player: 1, dice: '31', checker_move: '8/5 6/5' }];

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListDirections: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([{ id: 1, name: 'Blunder Cup', date: '', location: '', matchCount: 2, comment: '', pr: 0, mwc_loss: 0 }])),
    GetTournamentMatches: vi.fn(() => Promise.resolve([T_MATCH, OTHER])),
    GetPositionIDsByTournament: vi.fn(() => Promise.resolve([])),
    GetTournamentReview: vi.fn(() => Promise.resolve(null)),
    ListMatches: vi.fn(() => Promise.resolve([])),
    CountMatches: vi.fn(() => Promise.resolve(0)),
    GetMatchByID: vi.fn((id) => Promise.resolve(id === 7 ? T_MATCH : OTHER)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve(MOVES)),
    GetGamesByMatch: vi.fn(() => Promise.resolve([{ game_number: 1, initial_score: [0, 0], winner: 0, points_won: 1 }])),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    GetMatchMoveGrades: vi.fn(() => Promise.resolve([])),
    GetMatchTimeSummary: vi.fn(() => Promise.resolve(null)),
    GetMatchOrigin: vi.fn(() => Promise.resolve(null)),
    GetMatchDecisionLosses: vi.fn(() => Promise.resolve([])),
    GetMatchReview: vi.fn(() => Promise.resolve(null)),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    UpdateTournamentComment: vi.fn(() => Promise.resolve()),
    ReorderTournamentMatches: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    RemoveMatchFromTournament: vi.fn(() => Promise.resolve()),
    AddMatchToTournament: vi.fn(() => Promise.resolve()),
    UpdateTournament: vi.fn(() => Promise.resolve()),
    CreateTournament: vi.fn(() => Promise.resolve()),
    DeleteTournament: vi.fn(() => Promise.resolve()),
    TrashMatch: vi.fn(() => Promise.resolve())
}));

import TournamentPanel from '../components/TournamentPanel.svelte';
import { openPanels, PANEL } from '../stores/uiStore.js';
import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore, tournamentOpenRequestStore } from '../stores/tournamentStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { matchContextStore } from '../stores/positionStore.js';
import { matchListStore } from '../stores/matchListStore.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { answerConfirm } from './confirmHelper.js';
import { GetTournamentMatches, GetMatchMovePositions, TrashMatch, SwapMatchPlayers } from '../../wailsjs/go/database/Database.js';

beforeEach(() => {
    vi.clearAllMocks();
    tournamentsStore.set([]);
    selectedTournamentStore.set(null);
    tournamentMatchesStore.set([]);
    databasePathStore.set('/fake/db.sqlite');
    matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
    openPanels.set(new Set([PANEL.TOURNAMENT]));
    tournamentOpenRequestStore.set(1);
});
afterEach(cleanup);

async function openTournamentDetail() {
    render(TournamentPanel, { props: {} });
    return await screen.findByText('Bob');
}

describe('TournamentPanel — match sheet', () => {
    test('no sheet before a match is clicked', async () => {
        await openTournamentDetail();
        expect(screen.queryByTestId('match-tab-transcript')).toBeNull();
    });

    test('one click shows the Matches panel sheet with its tabs', async () => {
        const row = await openTournamentDetail();
        await fireEvent.click(row);

        for (const id of ['transcript', 'info', 'stats']) {
            expect(await screen.findByTestId(`match-tab-${id}`)).toBeTruthy();
        }
        expect(get(matchContextStore).isMatchMode).toBe(false);
    });

    test('clicking another match shows that match', async () => {
        const { GetMatchMovePositions } = await import('../../wailsjs/go/database/Database.js');
        const row = await openTournamentDetail();
        await fireEvent.click(row);
        await screen.findByTestId('match-tab-info');
        await fireEvent.click(screen.getByText('Dave'));
        await vi.waitFor(() => expect(GetMatchMovePositions).toHaveBeenCalledWith(8));
    });

    test('double-click enters match mode', async () => {
        const row = await openTournamentDetail();
        await fireEvent.dblClick(row);
        await vi.waitFor(() => expect(get(matchContextStore).isMatchMode).toBe(true));
    });
});

describe('TournamentPanel — hosted sheet', () => {
    test('mounting it touches nothing global: no shared sort, no list load, no key registration', async () => {
        const setSort = vi.spyOn(matchListStore, 'setSort');
        const row = await openTournamentDetail();
        await fireEvent.click(row);
        await screen.findByTestId('match-tab-info');

        expect(setSort).not.toHaveBeenCalled();
        // The Matches panel's own `v` handler is not registered: one `v` reaches one handler.
        const stop = vi.fn();
        const ev = new KeyboardEvent('keydown', { key: 'v', bubbles: true, cancelable: true });
        ev.stopPropagation = stop;
        document.body.dispatchEvent(ev);
        expect(stop).not.toHaveBeenCalled();
    });

    test('a second click on the same row closes the sheet', async () => {
        const row = await openTournamentDetail();
        await fireEvent.click(row);
        await screen.findByTestId('match-tab-info');
        await fireEvent.click(row);
        expect(screen.queryByTestId('match-tab-info')).toBeNull();
    });

    test('Escape closes the sheet first, then the tournament', async () => {
        const row = await openTournamentDetail();
        await fireEvent.click(row);
        await screen.findByTestId('match-tab-info');

        await fireEvent.keyDown(document.body, { key: 'Escape' });
        expect(screen.queryByTestId('match-tab-info')).toBeNull();
        expect(get(selectedTournamentStore)).not.toBeNull();

        await fireEvent.keyDown(document.body, { key: 'Escape' });
        expect(get(selectedTournamentStore)).toBeNull();
    });

    test('swapping the players reads the sheet again', async () => {
        const row = await openTournamentDetail();
        await fireEvent.click(row);
        await screen.findByTestId('match-tab-info');
        const before = vi.mocked(GetMatchMovePositions).mock.calls.length;

        await fireEvent.click(screen.getAllByTitle(/swap/i)[0]);

        await vi.waitFor(() => expect(SwapMatchPlayers).toHaveBeenCalledWith(7));
        await vi.waitFor(() => expect(vi.mocked(GetMatchMovePositions).mock.calls.length).toBeGreaterThan(before));
    });

    test('deleting the match from its sheet reloads the tournament and closes the sheet', async () => {
        const row = await openTournamentDetail();
        await fireEvent.click(row);
        await screen.findByTestId('match-tab-info');
        vi.mocked(GetTournamentMatches).mockResolvedValue(/** @type {any} */ ([OTHER]));

        await fireEvent.click(screen.getByTestId('match-more'));
        await fireEvent.click(await screen.findByText(/delete the match/i));
        await answerConfirm(true);

        await vi.waitFor(() => expect(TrashMatch).toHaveBeenCalledWith(7));
        await vi.waitFor(() => expect(screen.queryByText('Bob')).toBeNull());
        expect(screen.queryByTestId('match-tab-info')).toBeNull();
    });

    test('a requested tournament that no longer exists says so in the status bar', async () => {
        statusBarTextStore.set('');
        tournamentOpenRequestStore.set(99);
        render(TournamentPanel, { props: {} });
        await vi.waitFor(() => expect(JSON.stringify(get(statusBarTextStore))).toContain('tournament.gone'));
    });
});
