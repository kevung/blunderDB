/**
 * countLinkLeavesMatch.test.js
 *
 * A count of positions opens exactly those positions, whatever was on screen before. With a match
 * open, the board follows the match navigation and ignores the library index: a list loaded without
 * leaving MATCH was counted by the status bar while the board, the navigation and the Analysis panel
 * stayed on the match.
 *
 * Real stores, real positionLoader and modeMachine; the Wails bindings are mocked. App's navigation
 * effect is reproduced by `followIndex` through the same predicate (boardFollowsLibraryIndex).
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

const MATCH_POSITION_ID = 100;
const COUNTED = [11, 12, 13];

/** @param {number} id */
const positionOf = (id) => ({
    id,
    board: { points: [], bearoff: [15, 15] },
    cube: { owner: -1, value: 0 },
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0,
    has_jacoby: 0,
    has_beaver: 0,
    dice: [3, 1]
});

class FakeChart {
    destroy() {}
}
vi.mock('../components/stats/charts/chartjs.js', () => ({ loadChart: () => Promise.resolve(FakeChart) }));

vi.mock('../../wailsjs/go/database/Database.js', async () => {
    const { createDatabaseMock } = await import('../__mocks__/wails.js');
    const ids = () => Promise.resolve([11, 12, 13]);
    return createDatabaseMock({
        GetPositionIDsByStatsSelection: vi.fn(ids),
        GetPositionIDsByTournament: vi.fn(ids),
        GetPositionIDsByMatch: vi.fn(ids),
        LoadPositionIDsByFilters: vi.fn(async (/** @type {any} */ p) => String(p.restrictToPositionIDs).split(',').map(Number)),
        LoadPositionsByIDs: vi.fn(async (/** @type {number[]} */ list) => list.map((id) => ({ id, board: { points: [], bearoff: [15, 15] }, cube: { owner: -1, value: 0 }, score: [7, 7] }))),
        LoadPositionView: vi.fn(async (/** @type {number} */ id) => ({ analysis: { positionId: id, playedMove: `played ${id}` }, comment: '' })),
        GetStatsBreakdownPositionCounts: vi.fn(async () => ({ phase: { opening: { Positions: 3, Blunders: 1 } }, game_type: {}, tag: {}, score: {} })),
        SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
        CountAnkiDeckFilteredPositions: vi.fn(async () => 3),
        ListAnkiDeckFilteredPositionIDs: vi.fn(async (/** @type {number} */ _d, /** @type {string} */ _f, /** @type {number} */ offset, /** @type {number} */ limit) =>
            [11, 12, 13].slice(offset, offset + limit)
        )
    });
});

vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));

import { SaveLastVisitedPosition } from '../../wailsjs/go/database/Database.js';
import { statusBarModeStore, currentPositionIndexStore, activeTabStore } from '../stores/uiStore.js';
import { positionStore, positionsStore, matchContextStore, listedIds } from '../stores/positionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeCollectionStore } from '../stores/collectionStore.js';
import { boardFollowsLibraryIndex, showPosition } from '../services/positionService.js';
import { loadPositionsFromSelection } from '../services/positionLoader.js';
import { selectDeck } from '../services/ankiService.js';
import StatsBreakdownsTab from '../components/stats/StatsBreakdownsTab.svelte';
import StatsDashboardTab from '../components/stats/StatsDashboardTab.svelte';
import StatsProgressionTab from '../components/stats/StatsProgressionTab.svelte';

/** App's navigation effect: the board shows the indexed library position unless MATCH holds it. */
function followIndex() {
    return currentPositionIndexStore.subscribe((index) => {
        if (!boardFollowsLibraryIndex()) return;
        if (index < 0 || index >= get(positionsStore).length) return;
        positionsStore.getPosition(index).then((p) => p && showPosition(p));
    });
}

/** A match open on its third move, its position on the board and in the Analysis panel. */
async function openMatch() {
    const movePositions = [1, 2, MATCH_POSITION_ID].map((id, i) => ({ position: positionOf(id), move_number: i + 2, game_number: 1, checker_move: '13/9' }));
    matchContextStore.set(/** @type {any} */ ({ isMatchMode: true, matchID: 7, movePositions, currentIndex: 2, player1Name: 'A', player2Name: 'B' }));
    statusBarModeStore.set('MATCH');
    await showPosition(movePositions[2].position);
    expect(get(analysisStore).positionId).toBe(MATCH_POSITION_ID);
}

/** What the user sees after the click: the counted list, its first position, its analysis. */
async function expectCountedListShown() {
    await waitFor(() => expect(get(analysisStore).positionId).toBe(COUNTED[0]));
    expect(get(statusBarModeStore)).toBe('NORMAL');
    expect(get(matchContextStore).isMatchMode).toBe(false);
    expect(get(activeCollectionStore)).toBeNull();
    expect(await listedIds()).toEqual(COUNTED);
    // The status bar's counter outside MATCH: index + 1 / length.
    expect(get(currentPositionIndexStore)).toBe(0);
    expect(get(positionsStore).length).toBe(COUNTED.length);
    expect(get(positionStore).id).toBe(COUNTED[0]);
    expect(get(analysisStore).playedMove).toBe(`played ${COUNTED[0]}`);
    expect(get(activeTabStore)).toBe('analysis');
}

const TOURNAMENT = { ID: 1, Name: 'Open', Date: '2025-01-10', PR: 3.5, MWC: 0.02, NumDecisions: 3 };
const iv = { available: true, low: 1, high: 2 };

/** Each Stats tab and the count it offers to click. */
const tabs = {
    breakdowns: async () => {
        render(StatsBreakdownsTab, { props: { result: { PerPhase: [{ Phase: 'opening', PR: 3, PRInterval: iv, NumDecisions: 3, BlunderCount: 1 }], PerGameType: [], PerTag: [], PerScore: [] } } });
        await fireEvent.click(await screen.findByTestId('breakdown-positions-phase-opening'));
    },
    dashboard: async () => {
        render(StatsDashboardTab, { props: { result: /** @type {any} */ ({ Totals: { NumDecisions: 3, PR: 3 }, TopBlunders: [], Rolling: [] }), metric: 'pr' } });
        const cards = /** @type {HTMLElement[]} */ (Array.from(document.querySelectorAll('button.stat-card')));
        expect(cards.length).toBeGreaterThan(0);
        await fireEvent.click(cards[0]);
    },
    progression: async () => {
        render(StatsProgressionTab, { props: { result: /** @type {any} */ ({ PerTournament: [TOURNAMENT], PerMatch: [] }), metric: 'pr' } });
        await fireEvent.click(await screen.findByText('Open positions'));
    }
};

/** @type {() => void} */
let unfollow = () => {};
beforeEach(async () => {
    databasePathStore.set('/some/db.db');
    statusBarModeStore.set('NORMAL');
    matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
    activeCollectionStore.set(null);
    activeTabStore.set('stats');
    positionsStore.setIds([1, 2, 3, 4]);
    currentPositionIndexStore.set(0);
    unfollow = followIndex();
});
afterEach(() => {
    unfollow();
    cleanup();
    vi.clearAllMocks();
});

describe('a count in the Stats panel opens the counted positions', () => {
    test.each(Object.keys(tabs))('%s, with a match open: the match is left for the counted list', async (tab) => {
        await openMatch();
        await tabs[/** @type {keyof typeof tabs} */ (tab)]();
        await expectCountedListShown();
        // The way back to the match (`m`) finds the move that was being studied.
        expect(SaveLastVisitedPosition).toHaveBeenCalledWith(7, 2);
    });

    test.each(Object.keys(tabs))('%s, from the library: the counted list replaces it', async (tab) => {
        await tabs[/** @type {keyof typeof tabs} */ (tab)]();
        await expectCountedListShown();
        expect(SaveLastVisitedPosition).not.toHaveBeenCalled();
    });
});

describe('every count that opens a list leaves the match', () => {
    test('a game mark, a study plan family or a tournament count (loadPositionsFromSelection)', async () => {
        await openMatch();
        await loadPositionsFromSelection(COUNTED);
        await expectCountedListShown();
    });

    test('a collection being browsed is left too', async () => {
        statusBarModeStore.set('COLLECTION');
        activeCollectionStore.set(/** @type {any} */ ({ id: 3, name: 'c' }));
        await loadPositionsFromSelection(COUNTED);
        await expectCountedListShown();
    });

    test('an Anki deck counter (due, new…)', async () => {
        await openMatch();
        await selectDeck({ id: 5, name: 'deck' }, 'due');
        await waitFor(() => expect(get(analysisStore).positionId).toBe(COUNTED[0]));
        expect(get(statusBarModeStore)).toBe('NORMAL');
        expect(get(matchContextStore).isMatchMode).toBe(false);
        expect(get(positionsStore).length).toBe(COUNTED.length);
        expect(get(currentPositionIndexStore)).toBe(0);
        expect(get(positionStore).id).toBe(COUNTED[0]);
    });
});
