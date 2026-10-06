/**
 * MatchPanel.origin.test.js
 *
 * A match played here carries its origin above the Transcript: one line, and
 * the revealed seed with its fingerprint; an imported match carries none.
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

const SEED = '00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff';
const FINGERPRINT = 'f0e1d2c3b4a5968778695a4b3c2d1e0ff0e1d2c3b4a5968778695a4b3c2d1e0f';
const ORIGIN = {
    match_id: 7,
    start: '',
    dice_seed: SEED,
    fingerprint: FINGERPRINT,
    stopped_early: true,
    over_time: 2,
    bot_level: 'expert',
    bot_engine: 'v0.9.0',
    cadence: '{"name":"rapid-3+12","reserve":180,"delay":12}',
    cadence_settings: { name: 'rapid-3+12', reserve: 180, delay: 12 }
};

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
    GetMatchOrigin: vi.fn(() => Promise.resolve(ORIGIN)),
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
import { ListMatches, GetMatchOrigin, GetMatchMovePositions, GetGamesByMatch } from '../../wailsjs/go/database/Database.js';
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

describe('MatchPanel — the origin of a match played here', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(GetMatchOrigin).mockImplementation(() => Promise.resolve(ORIGIN));
        vi.mocked(GetMatchMovePositions).mockImplementation(() => Promise.resolve(MOVES));
        vi.mocked(GetGamesByMatch).mockImplementation(() => Promise.resolve([]));
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

    test('one line says it was played here, opened onto the revealed seed and its fingerprint', async () => {
        const container = await openTranscript();
        expect(GetMatchOrigin).toHaveBeenCalledWith(7);
        const line = container.querySelector('[data-testid="match-origin"]');
        expect(line).not.toBeNull();
        const summary = line.querySelector('summary').textContent;
        expect(summary).toContain('rapid-3+12');
        expect(summary).toContain('Bob');
        expect(summary).toContain('v0.9.0');
        expect(line.querySelector('[data-testid="origin-stopped"]')).not.toBeNull();
        expect(line.querySelector('[data-testid="origin-seed"]').textContent).toBe(SEED);
        expect(line.querySelector('[data-testid="origin-fingerprint"]').textContent).toBe(FINGERPRINT);
    });

    test('a match lost on time says so, not that it was stopped', async () => {
        vi.mocked(GetMatchOrigin).mockImplementation(() => Promise.resolve({ ...ORIGIN, lost_on_time: true, cadence_settings: { ...ORIGIN.cadence_settings, timeOut: 'lose_match' } }));
        const container = await openTranscript();
        const line = container.querySelector('[data-testid="match-origin"]');
        expect(line.querySelector('[data-testid="origin-lost-on-time"]').textContent).toContain('Bob');
        expect(line.querySelector('[data-testid="origin-stopped"]')).toBeNull();
    });

    test('a match played here without a single move still shows its origin', async () => {
        vi.mocked(GetMatchMovePositions).mockImplementation(() => Promise.resolve([]));
        vi.mocked(GetGamesByMatch).mockImplementation(() => Promise.resolve([]));
        const { container } = render(MatchPanel);
        await vi.waitFor(() => expect(container.querySelector('[data-testid="match-origin"]')).not.toBeNull());
    });

    test('a match not played here draws no origin', async () => {
        vi.mocked(GetMatchOrigin).mockImplementation(() => Promise.resolve(null));
        const container = await openTranscript();
        expect(GetMatchOrigin).toHaveBeenCalledWith(7);
        expect(container.querySelector('[data-testid="match-origin"]')).toBeNull();
    });
});
