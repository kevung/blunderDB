import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Mock Wails Database binding
// Leaving MATCH is modeMachine's concern (countLinkLeavesMatch.test.js); here only the loading is exercised.
vi.mock('../services/modeMachine.js', () => ({ leaveStudiedModeForList: vi.fn(), forgetSubSearchOrigin: vi.fn() }));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetPositionIDsByStatsSelection: vi.fn(),
    GetPositionIDsByTournament: vi.fn(),
    GetPositionIDsByMatch: vi.fn(),
    LoadPositionIDsByFilters: vi.fn(),
    StatsTournamentIDs: vi.fn()
}));

// Mock uiStore to prevent side-effects
vi.mock('../stores/uiStore.js', () => {
    const { writable } = require('svelte/store');
    return {
        activeTabStore: writable('analysis'),
        openPanel: vi.fn(),
        PANEL: {
            ANALYSIS: 'analysis',
            MATCH: 'match',
            TOURNAMENT: 'tournament',
            STATS: 'stats'
        },
        statusBarTextStore: writable(''),
        matchOpenRequestStore: writable(null),
        currentPositionIndexStore: writable(0)
    };
});

// Mock tournamentStore
vi.mock('../stores/tournamentStore.js', () => {
    const { writable } = require('svelte/store');
    return {
        selectedTournamentStore: writable(null),
        tournamentOpenRequestStore: writable(null),
        tournamentIDsFilterStore: writable(/** @type {number[] | null} */ (null))
    };
});

// Mock databaseStore — loaded by default
vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return {
        databasePathStore: writable('/some/db.db')
    };
});

// Mock positionStore: a bare-bones stand-in for positionsStore (positionList.js
// in the real app) that just remembers the last id list it was given —
// loadPositionsFromSelection only ever calls setIds() on it now (D.8, #208).
vi.mock('../stores/positionStore.js', () => {
    const { writable } = require('svelte/store');
    const store = writable([]);
    return {
        positionsStore: {
            subscribe: store.subscribe,
            set: store.set,
            setIds: (ids) => store.set(ids)
        }
    };
});

// Mock statsStore (real module imports Wails bindings); expose only the filter.
vi.mock('../stores/statsStore.js', () => {
    const { writable } = require('svelte/store');
    return {
        statsFilterStore: writable({ decisionType: -1, playerName: '', tournamentIDs: [], dateFrom: '', dateTo: '', matchLength: [] }),
        statsResultFilterStore: writable(null),
        playerTableFilterStore: writable(null)
    };
});

import { GetPositionIDsByStatsSelection, GetPositionIDsByTournament, GetPositionIDsByMatch, LoadPositionIDsByFilters, StatsTournamentIDs } from '../../wailsjs/go/database/Database.js';

import {
    loadPositionsFromSelection,
    loadPositionsFromStatsSelection,
    loadWorstBlunders,
    loadPositionsFromTournament,
    loadPositionsFromMatch,
    openTournamentInPanel,
    openMatchInPanel,
    openTournamentListInPanel,
    openStatsTournaments,
    openStatsPlayerBlunders
} from '../services/positionLoader.js';

import { statsFilterStore, statsResultFilterStore, playerTableFilterStore } from '../stores/statsStore.js';

import { activeTabStore, matchOpenRequestStore } from '../stores/uiStore.js';
import { tournamentOpenRequestStore, tournamentIDsFilterStore } from '../stores/tournamentStore.js';
import { positionsStore } from '../stores/positionStore.js';

describe('loadPositionsFromStatsSelection', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        positionsStore.set([]);
        activeTabStore.set('something');
    });

    test('calls GetPositionIDsByStatsSelection with correct args', async () => {
        vi.mocked(GetPositionIDsByStatsSelection).mockResolvedValue([1, 2, 3]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([1, 2, 3]);

        const filter = { playerName: 'Bob', decisionType: -1 };
        const sel = { kind: 'checker' };
        await loadPositionsFromStatsSelection(filter, sel);

        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, sel);
    });

    test('loads position ids into positionsStore and switches to analysis tab', async () => {
        vi.mocked(GetPositionIDsByStatsSelection).mockResolvedValue([10, 20]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([10, 20]);

        await loadPositionsFromStatsSelection({}, { kind: 'all' });

        expect(get(positionsStore)).toEqual([10, 20]);
        expect(get(activeTabStore)).toBe('analysis');
    });
});

describe('loadWorstBlunders', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        positionsStore.set([]);
        activeTabStore.set('something');
    });

    test('requests the top_blunders selection under the current stats filter', async () => {
        const filter = { decisionType: 1, playerName: 'Alice', tournamentIDs: [], dateFrom: '', dateTo: '', matchLength: [] };
        statsFilterStore.set(filter);
        vi.mocked(GetPositionIDsByStatsSelection).mockResolvedValue([7, 8]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([7, 8]);

        await loadWorstBlunders();

        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'top_blunders' });
        expect(get(positionsStore)).toEqual([7, 8]);
        expect(get(activeTabStore)).toBe('analysis');
    });

    test('passes a positive count through as LastN', async () => {
        statsFilterStore.set({ decisionType: -1 });
        vi.mocked(GetPositionIDsByStatsSelection).mockResolvedValue([1]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([1]);

        await loadWorstBlunders(50);

        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith({ decisionType: -1 }, { Kind: 'top_blunders', LastN: 50 });
    });

    test('ignores a non-positive / non-integer count (backend default applies)', async () => {
        statsFilterStore.set({ decisionType: -1 });
        vi.mocked(GetPositionIDsByStatsSelection).mockResolvedValue([1]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([1]);

        await loadWorstBlunders(0);
        expect(GetPositionIDsByStatsSelection).toHaveBeenLastCalledWith({ decisionType: -1 }, { Kind: 'top_blunders' });
    });
});

describe('loadPositionsFromTournament', () => {
    test('calls GetPositionIDsByTournament with tournamentID', async () => {
        vi.mocked(GetPositionIDsByTournament).mockResolvedValue([5, 6]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([5, 6]);

        await loadPositionsFromTournament(42);
        expect(GetPositionIDsByTournament).toHaveBeenCalledWith(42);
    });
});

describe('loadPositionsFromMatch', () => {
    test('calls GetPositionIDsByMatch with matchID', async () => {
        vi.mocked(GetPositionIDsByMatch).mockResolvedValue([7, 8]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([7, 8]);

        await loadPositionsFromMatch(99);
        expect(GetPositionIDsByMatch).toHaveBeenCalledWith(99);
    });
});

describe('loadPositionsFromSelection', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        positionsStore.set([]);
        activeTabStore.set('analysis');
    });

    test('sets statusBarTextStore and skips fetch when ids is empty', async () => {
        await loadPositionsFromSelection([]);
        expect(LoadPositionIDsByFilters).not.toHaveBeenCalled();
    });

    test('passes comma-separated IDs as restrictToPositionIDs', async () => {
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([1, 2, 3]);
        await loadPositionsFromSelection([1, 2, 3]);
        expect(LoadPositionIDsByFilters).toHaveBeenCalledWith(expect.objectContaining({ restrictToPositionIDs: '1,2,3' }));
    });
});

describe('openTournamentInPanel', () => {
    test('requests the tournament and switches to tournaments tab', () => {
        openTournamentInPanel(7);
        expect(get(tournamentOpenRequestStore)).toBe(7);
        expect(get(activeTabStore)).toBe('tournaments');
    });
});

describe('openMatchInPanel', () => {
    test('switches to matches tab', () => {
        activeTabStore.set('analysis');
        openMatchInPanel(12);
        expect(get(matchOpenRequestStore)).toBe(12);
        expect(get(activeTabStore)).toBe('matches');
    });
});

describe('openTournamentListInPanel', () => {
    beforeEach(() => {
        tournamentOpenRequestStore.set(null);
        tournamentIDsFilterStore.set(null);
        activeTabStore.set('stats');
    });

    test('a single tournament opens instead of a one-row list, dropping an earlier restriction', () => {
        tournamentIDsFilterStore.set([3, 9]);
        openTournamentListInPanel([5]);
        expect(get(tournamentOpenRequestStore)).toBe(5);
        expect(get(tournamentIDsFilterStore)).toBeNull();
        expect(get(activeTabStore)).toBe('tournaments');
    });

    test('several tournaments restrict the list', () => {
        openTournamentListInPanel([3, 9]);
        expect(get(tournamentIDsFilterStore)).toEqual([3, 9]);
        expect(get(tournamentOpenRequestStore)).toBeNull();
        expect(get(activeTabStore)).toBe('tournaments');
    });

    test('none leaves the panel where it is', () => {
        openTournamentListInPanel([]);
        expect(get(activeTabStore)).toBe('stats');
    });
});

describe('Stats counts', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        statsResultFilterStore.set(null);
    });

    test('the tournament count asks under the filter that produced the result', async () => {
        const shown = { decisionType: -1, playerName: 'Alice', tournamentIDs: [], dateFrom: '2025-01-01', dateTo: '', matchLength: [] };
        statsResultFilterStore.set(shown);
        statsFilterStore.set({ ...shown, dateFrom: '2026-01-01' });
        vi.mocked(StatsTournamentIDs).mockResolvedValue([4, 6]);

        await openStatsTournaments();

        expect(StatsTournamentIDs).toHaveBeenCalledWith(shown);
        expect(get(tournamentIDsFilterStore)).toEqual([4, 6]);
    });

    test("a Players row's blunders are the blunders selection of that player, every decision type", async () => {
        statsFilterStore.set(/** @type {any} */ ({ decisionType: 0, playerName: '', tournamentIDs: [2], dateFrom: '', dateTo: '', matchLength: [] }));
        vi.mocked(GetPositionIDsByStatsSelection).mockResolvedValue([1]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([1]);

        await openStatsPlayerBlunders('Bob');

        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(expect.objectContaining({ playerName: 'Bob', decisionType: -1, tournamentIDs: [2] }), { Kind: 'blunders' });
    });

    test('the blunders open under the filter the players table was read with', async () => {
        playerTableFilterStore.set(/** @type {any} */ ({ decisionType: -1, playerName: '', tournamentIDs: [8], dateFrom: '', dateTo: '', matchLength: [] }));
        statsFilterStore.set(/** @type {any} */ ({ decisionType: -1, playerName: '', tournamentIDs: [9], dateFrom: '', dateTo: '', matchLength: [] }));
        vi.mocked(GetPositionIDsByStatsSelection).mockResolvedValue([1]);
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValue([1]);

        await openStatsPlayerBlunders('Bob');

        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(expect.objectContaining({ playerName: 'Bob', tournamentIDs: [8] }), { Kind: 'blunders' });
        playerTableFilterStore.set(null);
    });
});
