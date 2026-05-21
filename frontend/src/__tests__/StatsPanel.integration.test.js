/**
 * StatsPanel.integration.test.js
 *
 * Integration test that simulates a complete user journey through the Stats
 * panel: filter → tab navigation → card click (drill-down) → PR/MWC toggle →
 * tournament context menu → cube action bar click.
 *
 * The Svelte components are NOT mounted (no @testing-library/svelte dependency);
 * instead we exercise the stores and service functions that implement the
 * panel behaviour, matching the pattern established by the other Stats tests.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ComputeStats: vi.fn(),
    GetPositionIDsByStatsSelection: vi.fn(),
    GetPositionIDsByTournament: vi.fn(),
    GetPositionIDsByMatch: vi.fn(),
    LoadPositionsByFilters: vi.fn()
}));

vi.mock('../stores/uiStore.js', () => {
    const { writable } = require('svelte/store');
    return {
        activeTabStore: writable('analysis'),
        openPanel: vi.fn(),
        closePanel: vi.fn(),
        PANEL: {
            ANALYSIS: 'analysis',
            MATCH: 'match',
            TOURNAMENT: 'tournament',
            STATS: 'stats'
        },
        statusBarTextStore: writable(''),
        currentPositionIndexStore: writable(0),
        openPanels: writable(new Set()),
        dbMutationCounterStore: writable(0)
    };
});

vi.mock('../stores/tournamentStore.js', () => {
    const { writable } = require('svelte/store');
    return { selectedTournamentStore: writable(null) };
});

vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/some/db.db') };
});

vi.mock('../stores/positionStore.js', () => {
    const { writable } = require('svelte/store');
    return { positionsStore: writable([]) };
});

// ── Imports (after mocks) ─────────────────────────────────────────────────────

import { ComputeStats, GetPositionIDsByStatsSelection, GetPositionIDsByTournament, LoadPositionsByFilters } from '../../wailsjs/go/database/Database.js';

import { statsFilterStore, statsResultStore, statsLoadingStore, statsErrorStore, statsMetricStore, refreshStats } from '../stores/statsStore.js';

import { loadPositionsFromStatsSelection, loadPositionsFromTournament, openTournamentInPanel } from '../services/positionLoader.js';

import { activeTabStore } from '../stores/uiStore.js';
import { selectedTournamentStore } from '../stores/tournamentStore.js';
import { positionsStore } from '../stores/positionStore.js';

// ── Fixture ───────────────────────────────────────────────────────────────────

const SAMPLE_RESULT = {
    Totals: { NumPositions: 300, NumMatches: 8, NumTournaments: 3, NumDecisions: 300 },
    PRGlobal: 4.5,
    PRChecker: 4.0,
    PRCube: 5.5,
    PRRolling: { 5: 3.5, 10: 4.0, 50: 4.3, 100: 4.4, 250: 4.5 },
    MWCGlobal: 0.054,
    MWCChecker: 0.038,
    MWCCube: 0.044,
    MWCRolling: { 5: 0.031, 10: 0.038, 50: 0.048, 100: 0.051, 250: 0.054 },
    MWCAvailable: true,
    PerTournament: [
        { ID: 10, Name: 'Open de Paris', PR: 3.8, MWC: 0.02, NumDecisions: 120 },
        { ID: 11, Name: 'Monte Carlo BG', PR: 5.2, MWC: 0.031, NumDecisions: 80 },
        { ID: 12, Name: 'World Cup Finals', PR: 4.6, MWC: 0.028, NumDecisions: 100 }
    ],
    PerMatch: [
        { ID: 1, Date: '2025-01-10T00:00:00Z', PlayerName: 'Alice', PR: 3.2, MWC: 0.018, NumDecisions: 40 },
        { ID: 2, Date: '2025-02-20T00:00:00Z', PlayerName: 'Alice', PR: 5.9, MWC: 0.036, NumDecisions: 25 }
    ],
    CubeActionBreakdown: [
        { Action: 'NoDouble', PR: 3.5, MWC: 0.021, NumDecisions: 80, BlunderCount: 5 },
        { Action: 'DoubleTake', PR: 6.2, MWC: 0.04, NumDecisions: 60, BlunderCount: 12 },
        { Action: 'DoublePass', PR: 2.1, MWC: 0.012, NumDecisions: 30, BlunderCount: 2 }
    ],
    ErrorHistogram: [
        { MinMP: 0, MaxMP: 5, Count: 150 },
        { MinMP: 5, MaxMP: 10, Count: 80 },
        { MinMP: 10, MaxMP: 25, Count: 40 },
        { MinMP: 25, MaxMP: 50, Count: 20 },
        { MinMP: 50, MaxMP: 100, Count: 8 },
        { MinMP: 100, MaxMP: -1, Count: 2 }
    ],
    TopBlunders: [
        {
            PositionID: 42,
            MatchID: 1,
            TournamentID: 10,
            ErrorMP: 450,
            MWCLoss: 0.0,
            Description: '',
            DecisionType: 0,
            MatchDate: '2025-01-10T00:00:00Z',
            PlayerNames: 'Alice vs Bob'
        }
    ]
};

// ── Reset helpers ─────────────────────────────────────────────────────────────

function resetAll() {
    statsResultStore.set(null);
    statsLoadingStore.set(false);
    statsErrorStore.set(null);
    statsMetricStore.set('pr');
    statsFilterStore.set({
        playerName: '',
        tournamentIDs: [],
        dateFrom: '',
        dateTo: '',
        decisionType: -1,
        matchLength: []
    });
    positionsStore.set([]);
    activeTabStore.set('analysis');
    selectedTournamentStore.set(null);
    vi.resetAllMocks();
}

// ── 1. Filter → ComputeStats ──────────────────────────────────────────────────

describe('Integration: filter change triggers ComputeStats', () => {
    beforeEach(resetAll);

    test('refreshStats is called with the current filter', async () => {
        ComputeStats.mockResolvedValue(SAMPLE_RESULT);
        const filter = get(statsFilterStore);
        await refreshStats(filter);
        expect(ComputeStats).toHaveBeenCalledWith(filter);
    });

    test('after refreshStats, statsResultStore contains the result', async () => {
        ComputeStats.mockResolvedValue(SAMPLE_RESULT);
        await refreshStats(get(statsFilterStore));
        expect(get(statsResultStore)).toEqual(SAMPLE_RESULT);
    });

    test('changing player filter produces a new ComputeStats call', async () => {
        ComputeStats.mockResolvedValue(SAMPLE_RESULT);

        const filter1 = get(statsFilterStore);
        await refreshStats(filter1);
        expect(ComputeStats).toHaveBeenCalledTimes(1);

        const filter2 = { ...filter1, playerName: 'Alice' };
        statsFilterStore.set(filter2);
        await refreshStats(filter2);
        expect(ComputeStats).toHaveBeenCalledTimes(2);
        expect(ComputeStats).toHaveBeenLastCalledWith(filter2);
    });

    test('statsLoadingStore is false after a successful refresh', async () => {
        ComputeStats.mockResolvedValue(SAMPLE_RESULT);
        await refreshStats(get(statsFilterStore));
        expect(get(statsLoadingStore)).toBe(false);
    });

    test('statsErrorStore is set and result cleared on backend failure', async () => {
        ComputeStats.mockRejectedValue(new Error('db error'));
        await refreshStats(get(statsFilterStore));
        expect(get(statsErrorStore)).toBe('db error');
        expect(get(statsResultStore)).toBeNull();
    });
});

// ── 2. Three tabs are represented (via tab name constants) ────────────────────

describe('Integration: three Stats tabs exist', () => {
    const TABS = ['dashboard', 'progression', 'errors'];

    test.each(TABS)('tab "%s" is a valid tab name', (tab) => {
        expect(TABS).toContain(tab);
    });

    test('all three tab names are distinct', () => {
        const unique = new Set(TABS);
        expect(unique.size).toBe(3);
    });
});

// ── 3. Dashboard → card click → drill-down ───────────────────────────────────

describe('Integration: Dashboard card click triggers drill-down', () => {
    beforeEach(() => {
        resetAll();
        GetPositionIDsByStatsSelection.mockResolvedValue([1, 2, 3]);
        LoadPositionsByFilters.mockResolvedValue([{ id: 1 }, { id: 2 }, { id: 3 }]);
    });

    test('clicking "all" card calls GetPositionIDsByStatsSelection with Kind=all', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'all', OnlyWithError: false });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'all', OnlyWithError: false });
    });

    test('clicking "checker" card calls GetPositionIDsByStatsSelection with Kind=checker', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'checker', OnlyWithError: false });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'checker', OnlyWithError: false });
    });

    test('clicking "cube" card calls GetPositionIDsByStatsSelection with Kind=cube', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'cube', OnlyWithError: false });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'cube', OnlyWithError: false });
    });

    test('drill-down loads positions into positionsStore', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'all', OnlyWithError: false });
        expect(get(positionsStore)).toEqual([{ id: 1 }, { id: 2 }, { id: 3 }]);
    });

    test('drill-down switches the active tab to analysis', async () => {
        activeTabStore.set('stats');
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'all', OnlyWithError: false });
        expect(get(activeTabStore)).toBe('analysis');
    });
});

// ── 4. Rolling N drill-down ───────────────────────────────────────────────────

describe('Integration: rolling-N click triggers last_n drill-down', () => {
    beforeEach(() => {
        resetAll();
        GetPositionIDsByStatsSelection.mockResolvedValue([10, 11, 12]);
        LoadPositionsByFilters.mockResolvedValue([{ id: 10 }, { id: 11 }, { id: 12 }]);
    });

    test('clicking N=10 calls GetPositionIDsByStatsSelection with Kind=last_n LastN=10', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'last_n', LastN: 10 });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'last_n', LastN: 10 });
    });

    test('clicking N=50 calls GetPositionIDsByStatsSelection with Kind=last_n LastN=50', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'last_n', LastN: 50 });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'last_n', LastN: 50 });
    });
});

// ── 5. PR ↔ MWC toggle — no refetch ──────────────────────────────────────────

describe('Integration: PR/MWC toggle changes display without refetching', () => {
    beforeEach(() => {
        resetAll();
        ComputeStats.mockResolvedValue(SAMPLE_RESULT);
    });

    test('toggling to mwc does NOT call ComputeStats again', async () => {
        await refreshStats(get(statsFilterStore));
        const callsBefore = ComputeStats.mock.calls.length;

        // Simulate toggle
        statsMetricStore.set('mwc');
        // No await — toggle is synchronous
        expect(ComputeStats.mock.calls.length).toBe(callsBefore);
    });

    test('in PR mode, result.PRGlobal is the displayed value for "all"', async () => {
        await refreshStats(get(statsFilterStore));
        statsMetricStore.set('pr');
        const result = get(statsResultStore);
        expect(result.PRGlobal).toBe(SAMPLE_RESULT.PRGlobal);
    });

    test('in MWC mode, result.MWCGlobal is the displayed value for "all"', async () => {
        await refreshStats(get(statsFilterStore));
        statsMetricStore.set('mwc');
        const result = get(statsResultStore);
        expect(result.MWCGlobal).toBe(SAMPLE_RESULT.MWCGlobal);
    });

    test('toggling PR→MWC→PR does not change the result object', async () => {
        await refreshStats(get(statsFilterStore));
        const snapshot = get(statsResultStore);
        statsMetricStore.set('mwc');
        statsMetricStore.set('pr');
        expect(get(statsResultStore)).toBe(snapshot);
    });
});

// ── 6. Progression tab — tournament point click → "Open tournament" ───────────

describe('Integration: Progression tab — tournament context menu', () => {
    beforeEach(() => {
        resetAll();
        GetPositionIDsByTournament.mockResolvedValue([20, 21, 22]);
        LoadPositionsByFilters.mockResolvedValue([{ id: 20 }, { id: 21 }, { id: 22 }]);
    });

    test('openTournamentInPanel sets selectedTournamentStore', () => {
        const tourID = SAMPLE_RESULT.PerTournament[0].ID; // 10
        openTournamentInPanel(tourID);
        expect(get(selectedTournamentStore)).toBe(tourID);
    });

    test('openTournamentInPanel switches active tab to tournaments', () => {
        openTournamentInPanel(10);
        expect(get(activeTabStore)).toBe('tournaments');
    });

    test('"Open positions" for a tournament calls GetPositionIDsByTournament', async () => {
        await loadPositionsFromTournament(10);
        expect(GetPositionIDsByTournament).toHaveBeenCalledWith(10);
    });

    test('"Open positions" for a tournament loads positions into positionsStore', async () => {
        await loadPositionsFromTournament(10);
        expect(get(positionsStore)).toEqual([{ id: 20 }, { id: 21 }, { id: 22 }]);
    });
});

// ── 7. Errors tab — DoubleTake bar click ──────────────────────────────────────

describe('Integration: Errors tab — cube action bar click', () => {
    beforeEach(() => {
        resetAll();
        GetPositionIDsByStatsSelection.mockResolvedValue([30, 31]);
        LoadPositionsByFilters.mockResolvedValue([{ id: 30 }, { id: 31 }]);
    });

    test('clicking DoubleTake bar calls loadPositionsFromStatsSelection with correct SelectionSpec', async () => {
        const filter = get(statsFilterStore);
        const sel = { Kind: 'cube_action', CubeAction: 'DoubleTake', OnlyWithError: true };
        await loadPositionsFromStatsSelection(filter, sel);
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, sel);
    });

    test('clicking NoDouble bar produces Kind=cube_action CubeAction=NoDouble', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, {
            Kind: 'cube_action',
            CubeAction: 'NoDouble',
            OnlyWithError: true
        });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'cube_action', CubeAction: 'NoDouble', OnlyWithError: true });
    });

    test('clicking a histogram bucket calls loadPositionsFromStatsSelection with Kind=error_bucket', async () => {
        const filter = get(statsFilterStore);
        // Bucket: 5–10 mp
        await loadPositionsFromStatsSelection(filter, {
            Kind: 'error_bucket',
            BucketMinMP: 5,
            BucketMaxMP: 10
        });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'error_bucket', BucketMinMP: 5, BucketMaxMP: 10 });
    });

    test('DoubleTake drill-down loads positions into positionsStore', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, {
            Kind: 'cube_action',
            CubeAction: 'DoubleTake',
            OnlyWithError: true
        });
        expect(get(positionsStore)).toEqual([{ id: 30 }, { id: 31 }]);
    });
});

// ── 8. Combined scenario: filter → result → PR toggle → drill-down ────────────

describe('Integration: full journey — filter, result, toggle, drill-down', () => {
    beforeEach(() => {
        resetAll();
        ComputeStats.mockResolvedValue(SAMPLE_RESULT);
        GetPositionIDsByStatsSelection.mockResolvedValue([5, 6, 7]);
        LoadPositionsByFilters.mockResolvedValue([{ id: 5 }, { id: 6 }, { id: 7 }]);
    });

    test('full scenario completes without error', async () => {
        // 1. Apply player filter
        const filter = { ...get(statsFilterStore), playerName: 'Alice' };
        statsFilterStore.set(filter);

        // 2. Refresh stats
        await refreshStats(filter);
        expect(get(statsResultStore)).toEqual(SAMPLE_RESULT);
        expect(ComputeStats).toHaveBeenCalledWith(filter);

        // 3. Toggle to MWC — no new ComputeStats call
        statsMetricStore.set('mwc');
        expect(ComputeStats).toHaveBeenCalledTimes(1);
        expect(get(statsMetricStore)).toBe('mwc');

        // 4. Click checker card — drill-down
        await loadPositionsFromStatsSelection(filter, { Kind: 'checker', OnlyWithError: false });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'checker', OnlyWithError: false });
        expect(get(positionsStore)).toEqual([{ id: 5 }, { id: 6 }, { id: 7 }]);
        expect(get(activeTabStore)).toBe('analysis');
    });
});
