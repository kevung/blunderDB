/**
 * MatchSheet.component.test.js
 *
 * The sheet of a match mounted on its own: it reads its match and registers nothing global
 * (no key scope, no shared list, no `#matchPanel`), and two sheets on screen share no DOM id.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';

const A = { id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 1, pr: 0, mwc_loss: 0 };
const B = { id: 8, player1_name: 'Carol', player2_name: 'Dave', match_length: 5, match_date: '2026-01-16', game_count: 1, pr: 0, mwc_loss: 0 };
const MOVES = [{ game_number: 1, move_number: 1, move_id: 1, position_id: 11, player_on_roll: 0, move_type: 'checker', checker_move: '8/5 6/5', position: { id: 11, dice: [3, 1] } }];

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() => Promise.resolve([])),
    CountMatches: vi.fn(() => Promise.resolve(0)),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    GetMatchByID: vi.fn((id) => Promise.resolve(id === 7 ? A : B)),
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
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SetMatchVideoSource: vi.fn(() => Promise.resolve()),
    RepairMatch: vi.fn(() => Promise.resolve(null)),
    TrashMatch: vi.fn(() => Promise.resolve())
}));
vi.mock('../services/keyDispatch.js', async (orig) => ({ ...(await orig()), registerKeys: vi.fn(() => () => {}) }));

import MatchSheet from '../components/MatchSheet.svelte';
import { registerKeys } from '../services/keyDispatch.js';
import { matchListStore } from '../stores/matchListStore.js';
import { matchContextStore } from '../stores/positionStore.js';
import { ListMatches, GetMatchMovePositions } from '../../wailsjs/go/database/Database.js';

beforeEach(() => {
    vi.clearAllMocks();
    matchListStore.set({ ...get(matchListStore), rows: [], total: 0, loaded: false });
    matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
});
afterEach(cleanup);

const settle = async () => {
    for (let i = 0; i < 6; i++) await new Promise((r) => setTimeout(r, 0));
};

describe('MatchSheet alone', () => {
    test('shows the match it is given: header, tabs, transcript', async () => {
        const { container } = render(MatchSheet, { props: { matchId: 7 } });
        await vi.waitFor(() => expect(container.querySelector('.detail-header')).not.toBeNull());
        expect(container.textContent).toContain('Alice');
        await vi.waitFor(() => expect(GetMatchMovePositions).toHaveBeenCalledWith(7));
        await vi.waitFor(() => expect(container.querySelector('details.game-section')).not.toBeNull());
        expect(container.querySelectorAll('[role="tab"]').length).toBeGreaterThanOrEqual(3);
    });

    test('shows nothing without a match, then follows the match id', async () => {
        const { container, rerender } = render(MatchSheet, { props: { matchId: null } });
        await settle();
        expect(container.querySelector('.detail-pane')).toBeNull();
        await rerender({ matchId: 8 });
        await vi.waitFor(() => expect(container.textContent).toContain('Carol'));
        await rerender({ matchId: 7 });
        await vi.waitFor(() => expect(container.textContent).toContain('Alice'));
    });

    test('registers no key scope, takes no panel id, leaves the shared list alone', async () => {
        const before = get(matchListStore);
        const { container } = render(MatchSheet, { props: { matchId: 7 } });
        await vi.waitFor(() => expect(container.querySelector('.detail-header')).not.toBeNull());
        await settle();
        expect(registerKeys).not.toHaveBeenCalled();
        expect(document.getElementById('matchPanel')).toBeNull();
        expect(ListMatches).not.toHaveBeenCalled();
        const after = get(matchListStore);
        expect([after.column, after.direction, after.text, after.rows]).toEqual([before.column, before.direction, before.text, before.rows]);
    });

    test('Escape and arrows on the sheet do not touch the match under review', async () => {
        const { container } = render(MatchSheet, { props: { matchId: 7 } });
        await vi.waitFor(() => expect(container.querySelector('details.game-section')).not.toBeNull());
        await fireEvent.keyDown(container.querySelector('[role="tablist"]'), { key: 'ArrowRight' });
        await fireEvent.keyDown(document.body, { key: 'Escape' });
        expect(get(matchContextStore).isMatchMode).toBe(false);
        expect(container.querySelector('.detail-header')).not.toBeNull();
    });

    test('two sheets on screen share no DOM id, and their tabs point at their own panels', async () => {
        const { container: one } = render(MatchSheet, { props: { matchId: 7 } });
        const { container: two } = render(MatchSheet, { props: { matchId: 8 } });
        await vi.waitFor(() => expect(one.querySelector('[role="tab"]')).not.toBeNull());
        await vi.waitFor(() => expect(two.querySelector('[role="tab"]')).not.toBeNull());
        const ids = [...document.querySelectorAll('[id]')].map((e) => e.id);
        expect(ids.length).toBeGreaterThan(10);
        expect(new Set(ids).size, 'every id is unique').toBe(ids.length);
        for (const tab of two.querySelectorAll('[role="tab"]')) {
            const panel = document.getElementById(tab.getAttribute('aria-controls'));
            expect(panel && two.contains(panel), 'the tab controls a panel of its own sheet').toBe(true);
            expect(panel.getAttribute('aria-labelledby')).toBe(tab.id);
        }
    });
});
