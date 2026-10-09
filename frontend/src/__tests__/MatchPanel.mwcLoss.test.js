// @ts-nocheck -- DOM queries on a rendered panel, as the neighbouring MatchPanel tests
/**
 * MatchPanel.mwcLoss.test.js
 *
 * The Transcript shows the match winning chances each decision cost, laid out
 * as the decision times are (a column, a sort header, absent when nothing is
 * scored), and the loss and time charts jump to a decision in the Transcript.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

const MATCH = { id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2 };

const MOVES = [
    { move_id: 101, game_number: 1, move_number: 1, move_type: 'checker', player_on_roll: 0, position: { dice: [3, 1] }, checker_move: '8/5 6/5', decision_ms: 5200, cube_decision_ms: 1000 },
    { move_id: 102, game_number: 1, move_number: 2, move_type: 'checker', player_on_roll: 1, position: { dice: [6, 5] }, checker_move: '24/13' },
    { move_id: 103, game_number: 1, move_number: 3, move_type: 'cube', player_on_roll: 0, position: { dice: [0, 0] }, cube_action: 'Double', decision_ms: 12000 },
    { move_id: 201, game_number: 2, move_number: 1, move_type: 'checker', player_on_roll: 0, position: { dice: [5, 2] }, checker_move: '13/8 13/11' },
    { move_id: 202, game_number: 2, move_number: 2, move_type: 'checker', player_on_roll: 1, position: { dice: [4, 4] }, checker_move: '24/20(2)' }
];

const LOSSES = [
    { move_id: 101, game_number: 1, move_number: 1, player: 0, decision_type: 'checker', mwc_loss: 0.0123, difficulty: 0.004, avoidable: false },
    { move_id: 102, game_number: 1, move_number: 2, player: 1, decision_type: 'checker', mwc_loss: null, difficulty: null, avoidable: false },
    { move_id: 103, game_number: 1, move_number: 3, player: 0, decision_type: 'cube', mwc_loss: 0, difficulty: 0, avoidable: false },
    { move_id: 201, game_number: 2, move_number: 1, player: 0, decision_type: 'checker', mwc_loss: 0.05, difficulty: 0.001, avoidable: true },
    { move_id: 202, game_number: 2, move_number: 2, player: 1, decision_type: 'checker', mwc_loss: 0.002, difficulty: 0.003, avoidable: false }
];

const SUMMARY = {
    has_cadence: true,
    players: [
        { total_ms: 18200, checker_count: 1, checker_total_ms: 5200, cube_count: 2, cube_total_ms: 13000, unknown: 0, over_time: true },
        { total_ms: 0, checker_count: 0, checker_total_ms: 0, cube_count: 0, cube_total_ms: 0, unknown: 2, over_time: false }
    ]
};

// The match review as Go serves it: the difficulty summary is
// storage.SummariseDifficulty's over LOSSES, which the panel shows as is.
const reviewPlayer = (difficulty) => ({
    pr: 0,
    pr_interval: { available: false, low: 0, high: 0, units: 1 },
    decisions: 0,
    mwc7: { available: false },
    mwc_loss: 0,
    to_review: [],
    pace: { hasty: 0, deliberate: 0, unknown: 0, hasty_loss: 0, deliberate_loss: 0 },
    luck: { available: false, rolls: 0, rolls_measured: 0 },
    difficulty
});
const REVIEW = {
    match_id: 1,
    players: [
        reviewPlayer({ decisions: 3, loss: 0.0623, difficulty: 0.005, excess: 0.0573, ratio: 12.46, avoidable: 1 }),
        reviewPlayer({ decisions: 1, loss: 0.002, difficulty: 0.003, excess: -0.001, ratio: null, avoidable: 0 })
    ]
};

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
    GetMatchMoveGrades: vi.fn(() => Promise.resolve([])),
    GetMatchDecisionLosses: vi.fn(() => Promise.resolve(LOSSES)),
    GetMatchReview: vi.fn(() => Promise.resolve(REVIEW)),
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
import { ListMatches, GetMatchDecisionLosses, LoadAnalysis } from '../../wailsjs/go/database/Database.js';
import { openTab } from './matchTabHelper.js';
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

describe('MatchPanel — the Transcript carries the MWC loss of every decision', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
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

    const losses = (container) => [...container.querySelectorAll('details.game-section')[0].querySelectorAll('[data-testid="move-loss"]')].map((c) => c.textContent.trim());

    test('a loss reads in percent, zero as 0, an unscored decision as a dash', async () => {
        const container = await openTranscript();
        expect(losses(container)).toEqual(['1.23 %', '—', '0']);
    });

    test('the MWC header orders the rows by loss, the unscored ones last', async () => {
        const container = await openTranscript();
        const header = container.querySelector('[data-testid="loss-sort"]');
        await fireEvent.click(header);
        expect(losses(container)).toEqual(['1.23 %', '0', '—']);
        await fireEvent.click(header);
        expect(losses(container)).toEqual(['0', '1.23 %', '—']);
        await fireEvent.click(header);
        expect(losses(container)).toEqual(['1.23 %', '—', '0']);
    });

    test('a match with nothing scored has no loss column, no loss chart', async () => {
        vi.mocked(GetMatchDecisionLosses).mockResolvedValueOnce(LOSSES.map((d) => ({ ...d, mwc_loss: null })));
        const container = await openTranscript();
        expect(container.querySelector('[data-testid="move-loss"]')).toBeNull();
        expect(container.querySelector('[data-testid="match-losses"]')).toBeNull();
    });

    test('the chart header table ends on the per-player total, the sum of the column', async () => {
        const container = await openTranscript();
        expect(container.querySelector('[data-testid="loss-total-0"]').textContent).toBe('6.23 %');
        expect(container.querySelector('[data-testid="loss-total-1"]').textContent).toBe('0.20 %');
    });

    test('a difficulty reads beside the loss, an avoidable error is marked in the Transcript and on the chart', async () => {
        const container = await openTranscript();
        const game1 = container.querySelectorAll('details.game-section')[0];
        expect([...game1.querySelectorAll('[data-testid="move-difficulty"]')].map((c) => c.textContent.trim())).toEqual(['0.40 %', '—', '0']);
        expect(game1.querySelector('[data-testid="move-avoidable"]')).toBeNull();
        const game2 = container.querySelectorAll('details.game-section')[1];
        game2.open = true;
        await fireEvent(game2, new Event('toggle'));
        for (let i = 0; i < 4; i++) await tick();
        expect(game2.querySelectorAll('[data-testid="move-avoidable"]').length).toBe(1);
        expect(container.querySelectorAll('[data-testid="avoidable-dot"]').length).toBe(1);
        expect(container.querySelectorAll('[data-testid="loss-plot-per"] line.difficulty').length).toBe(3);
    });

    test('the header table gives each player the difficulty, the excess, the ratio and the avoidable errors the review serves', async () => {
        const container = await openTranscript();
        const cell = (id) => container.querySelector(`[data-testid="${id}"]`).textContent;
        expect(cell('difficulty-total-0')).toBe('0.50 %');
        expect(cell('excess-0')).toBe('+5.73 %');
        expect(cell('ratio-0')).toBe('12.46');
        expect(cell('avoidable-0')).toBe('1');
        expect(cell('excess-1')).toBe('−0.10 %');
        expect(cell('ratio-1')).toBe('—'); // 0.3 % of difficulty: under the floor
    });

    test('the header sits above a tab bar; the transcript is shown, the other tabs wait mounted', async () => {
        const container = await openTranscript();
        const header = container.querySelector('[data-testid="match-detail-header"]');
        const tablist = container.querySelector('[role="tablist"]');
        expect(header.compareDocumentPosition(tablist) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
        // The review scored no decision: there is nothing to revisit, so no such tab.
        const tabs = [...tablist.querySelectorAll('[role="tab"]')].map((b) => b.getAttribute('data-testid'));
        expect(tabs).toEqual(['match-tab-transcript', 'match-tab-charts', 'match-tab-details', 'match-tab-info', 'match-tab-stats']);
        const panel = (/** @type {string} */ id) => container.querySelector(`#match-tabpanel-${id}`);
        expect(panel('transcript').hidden).toBe(false);
        expect(panel('charts').hidden).toBe(true);
        expect(panel('charts').querySelector('[data-testid="match-losses"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="match-losses"] table')).toBeNull();
    });

    test('a tab shown is remembered for the next match', async () => {
        const container = await openTranscript();
        await openTab(container, 'details');
        expect(container.querySelector('#match-tabpanel-details').hidden).toBe(false);
        expect(container.querySelector('#match-tabpanel-transcript').hidden).toBe(true);
        expect(container.querySelector('[data-testid="match-tab-details"]').getAttribute('aria-selected')).toBe('true');
        expect(localStorage.getItem('blunderdb.matchTab')).toBe('details');
    });

    test('the arrow keys walk the tabs and never reach the match', async () => {
        const container = await openTranscript();
        const before = get(matchContextStore).currentIndex;
        const tab = container.querySelector('[data-testid="match-tab-transcript"]');
        await fireEvent.keyDown(tab, { key: 'ArrowRight' });
        expect(container.querySelector('#match-tabpanel-charts').hidden).toBe(false);
        await fireEvent.keyDown(tab, { key: 'End' });
        expect(container.querySelector('#match-tabpanel-stats').hidden).toBe(false);
        expect(get(matchContextStore).currentIndex).toBe(before);
    });

    test('Enter on a chart jumps to the decision: the review opens it and the row is marked', async () => {
        const container = await openTranscript();
        const plot = container.querySelector('[data-testid="loss-plot-per"]');
        plot.focus();
        await fireEvent.keyDown(plot, { key: 'ArrowRight' });
        await fireEvent.keyDown(plot, { key: 'ArrowRight' });
        await fireEvent.keyDown(plot, { key: 'Enter' });
        await vi.waitFor(() => expect(LoadAnalysis).toHaveBeenCalled());
        for (let i = 0; i < 4; i++) await tick();
        expect(get(matchContextStore).currentIndex).toBe(1);
        expect(container.querySelector('tr.current-move').getAttribute('data-move-idx')).toBe('1');
    });

    test('a jump into the game the review is in reopens it when the user folded it', async () => {
        const container = await openTranscript();
        const plot = container.querySelector('[data-testid="loss-plot-per"]');
        await fireEvent.keyDown(plot, { key: 'Home' });
        await fireEvent.keyDown(plot, { key: 'Enter' });
        await vi.waitFor(() => expect(container.querySelector('tr.current-move')).not.toBeNull());
        const game1 = container.querySelectorAll('details.game-section')[0];
        game1.open = false;
        await fireEvent(game1, new Event('toggle'));
        for (let i = 0; i < 4; i++) await tick();
        expect(game1.querySelector('[data-move-idx="2"]')).toBeNull();
        await fireEvent.keyDown(plot, { key: 'ArrowRight' });
        await fireEvent.keyDown(plot, { key: 'ArrowRight' });
        await fireEvent.keyDown(plot, { key: 'Enter' });
        await vi.waitFor(() => expect(get(matchContextStore).currentIndex).toBe(2));
        for (let i = 0; i < 4; i++) await tick();
        expect(game1.open).toBe(true);
        expect(game1.querySelector('[data-move-idx="2"]')).not.toBeNull();
    });

    test('the time chart jumps alike', async () => {
        const container = await openTranscript();
        const plot = container.querySelector('[data-testid="times-plot"]');
        await fireEvent.keyDown(plot, { key: 'End' });
        await fireEvent.keyDown(plot, { key: 'Enter' });
        await vi.waitFor(() => expect(get(matchContextStore).currentIndex).toBe(4));
    });

    test('hovering a decision on one chart marks it on the other and in the Transcript, cleared on leave', async () => {
        const container = await openTranscript();
        const times = container.querySelector('[data-testid="times-plot"]');
        const lossPlot = container.querySelector('[data-testid="loss-plot-per"]');
        await fireEvent.keyDown(times, { key: 'End' });
        expect(times.querySelectorAll('rect.hot').length).toBe(0); // the last decision has no time
        expect(lossPlot.querySelector('rect.hot')).not.toBeNull();
        expect(lossPlot.querySelector('line.cross')).not.toBeNull();
        await fireEvent.keyDown(lossPlot, { key: 'Home' });
        expect(times.querySelector('rect.hot')).not.toBeNull();
        expect(container.querySelector('tr.linked').getAttribute('data-move-idx')).toBe('0');
        await fireEvent.mouseLeave(lossPlot);
        expect(container.querySelector('tr.linked')).toBeNull();
        expect(times.querySelector('line.cross')).toBeNull();
    });
});
