/**
 * MatchPanel.transcriptTimes.test.js
 *
 * The Transcript shows the time of each decision, the cube decision apart from
 * the checker play, empty when unknown; orders the rows by it; and sums it per
 * player above the games, with the turns past the Cadence's reserve.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const MATCH = { id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2 };

const MOVES = [
    { move_id: 101, game_number: 1, move_number: 1, move_type: 'checker', player_on_roll: 0, position: { dice: [3, 1] }, checker_move: '8/5 6/5', decision_ms: 5200, cube_decision_ms: 1000 },
    { move_id: 102, game_number: 1, move_number: 2, move_type: 'checker', player_on_roll: 1, position: { dice: [6, 5] }, checker_move: '24/13' },
    { move_id: 103, game_number: 1, move_number: 3, move_type: 'cube', player_on_roll: 0, position: { dice: [0, 0] }, cube_action: 'Double', decision_ms: 12000 },
    { move_id: 201, game_number: 2, move_number: 1, move_type: 'checker', player_on_roll: 0, position: { dice: [5, 2] }, checker_move: '13/8 13/11' },
    { move_id: 202, game_number: 2, move_number: 2, move_type: 'checker', player_on_roll: 1, position: { dice: [4, 4] }, checker_move: '24/20(2)' }
];

const SUMMARY = {
    has_cadence: true,
    players: [
        { total_ms: 18200, checker_count: 1, checker_total_ms: 5200, cube_count: 2, cube_total_ms: 13000, unknown: 0, over_time: true },
        { total_ms: 0, checker_count: 0, checker_total_ms: 0, cube_count: 0, cube_total_ms: 0, unknown: 2, over_time: false }
    ]
};

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() => Promise.resolve([MATCH])),
    CountMatches: vi.fn(() => Promise.resolve(1)),
    GetMatchByID: vi.fn(() => Promise.resolve(null)),
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
    GetMatchTimeSummary: vi.fn(() => Promise.resolve(SUMMARY)),
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
import { ListMatches, GetMatchOrigin } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';

async function openTranscript() {
    const view = render(MatchPanel);
    const { container } = view;
    await vi.waitFor(() => expect(ListMatches).toHaveBeenCalledTimes(2));
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

describe('MatchPanel — the Transcript carries the time of every decision', () => {
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

    const col = (container, name, game = 0) => [...container.querySelectorAll('details.game-section')[game].querySelectorAll(`[data-testid="move-time-${name}"]`)].map((c) => c.textContent.trim());

    test('cube, play and clock are three columns, a time not taken reads as a dash', async () => {
        const container = await openTranscript();
        expect(col(container, 'cube')).toEqual(['1.0 s', '—', '12.0 s']);
        expect(col(container, 'play')).toEqual(['5.2 s', '—', '—']);
        expect(col(container, 'clock')).toEqual(['0:06', '0:00', '0:18']);
    });

    test('the Play header orders the rows by time, the unknown ones last, the clock keeps the match order', async () => {
        const container = await openTranscript();
        const header = container.querySelector('button.time-sort');
        await fireEvent.click(header);
        expect(col(container, 'cube')).toEqual(['12.0 s', '1.0 s', '—']);
        expect(col(container, 'clock')).toEqual(['0:18', '0:06', '0:00']);
        await fireEvent.click(header);
        expect(col(container, 'cube')).toEqual(['1.0 s', '12.0 s', '—']);
        await fireEvent.click(header);
        expect(col(container, 'cube')).toEqual(['1.0 s', '—', '12.0 s']);
    });

    test('under a Cadence the clock column shows the reserve left, and the metadata state the cadence and the bank', async () => {
        vi.mocked(GetMatchOrigin).mockResolvedValueOnce({
            dice_seed: 'ab',
            cadence_settings: { name: 'rapid-2+12', reserve: 120, delay: 12, timeOut: 'lose_match' },
            clock: { start: [120000, 120000], remaining: [117000, 120000, 100000, 0, 0] }
        });
        const container = await openTranscript();
        expect(col(container, 'clock')).toEqual(['1:57', '2:00', '1:40']);
        expect(container.querySelector('[data-testid="header-cadence"]').textContent).toContain('2:00');
        await fireEvent.click([...container.querySelectorAll('button')].find((b) => b.classList.contains('detail-tab') && b.textContent.trim() === 'Info'));
        const row = container.querySelector('[data-testid="meta-cadence"]');
        expect(row.textContent).toContain('rapid-2+12');
        expect(container.querySelector('[data-testid="meta-bank"]').textContent).toContain('2:00 each');
    });

    test('the summary marks the player whose reserve ran out, and leaves an unknown mean empty', async () => {
        const container = await openTranscript();
        const summary = container.querySelector('[data-testid="match-times"]');
        expect(summary).not.toBeNull();
        expect(summary.querySelector('[data-testid="overrun-0"]').textContent).toBe('●');
        const bob = [...summary.querySelectorAll('tbody tr')][1];
        expect([...bob.querySelectorAll('td')].slice(1, 4).map((c) => c.textContent)).toEqual(['', '', '']);
    });
});
