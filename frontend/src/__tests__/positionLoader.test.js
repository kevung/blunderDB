import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Mock Wails Database binding
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetPositionIDsByStatsSelection: vi.fn(),
    GetPositionIDsByTournament: vi.fn(),
    GetPositionIDsByMatch: vi.fn(),
    LoadPositionIDsByFilters: vi.fn()
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
        currentPositionIndexStore: writable(0)
    };
});

// Mock tournamentStore
vi.mock('../stores/tournamentStore.js', () => {
    const { writable } = require('svelte/store');
    return {
        selectedTournamentStore: writable(null)
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
        statsFilterStore: writable({ decisionType: -1, playerName: '', tournamentIDs: [], dateFrom: '', dateTo: '', matchLength: [] })
    };
});

import { GetPositionIDsByStatsSelection, GetPositionIDsByTournament, GetPositionIDsByMatch, LoadPositionIDsByFilters } from '../../wailsjs/go/database/Database.js';

import {
    loadPositionsFromSelection,
    loadPositionsFromStatsSelection,
    loadWorstBlunders,
    loadPositionsFromTournament,
    loadPositionsFromMatch,
    openTournamentInPanel,
    openMatchInPanel
} from '../services/positionLoader.js';

import { statsFilterStore } from '../stores/statsStore.js';

import { activeTabStore } from '../stores/uiStore.js';
import { selectedTournamentStore } from '../stores/tournamentStore.js';
import { positionsStore } from '../stores/positionStore.js';

describe('loadPositionsFromStatsSelection', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        positionsStore.set([]);
        activeTabStore.set('something');
    });

    test('calls GetPositionIDsByStatsSelection with correct args', async () => {
        GetPositionIDsByStatsSelection.mockResolvedValue([1, 2, 3]);
        LoadPositionIDsByFilters.mockResolvedValue([1, 2, 3]);

        const filter = { playerName: 'Bob', decisionType: -1 };
        const sel = { kind: 'checker' };
        await loadPositionsFromStatsSelection(filter, sel);

        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, sel);
    });

    test('loads position ids into positionsStore and switches to analysis tab', async () => {
        GetPositionIDsByStatsSelection.mockResolvedValue([10, 20]);
        LoadPositionIDsByFilters.mockResolvedValue([10, 20]);

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
        GetPositionIDsByStatsSelection.mockResolvedValue([7, 8]);
        LoadPositionIDsByFilters.mockResolvedValue([7, 8]);

        await loadWorstBlunders();

        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'top_blunders' });
        expect(get(positionsStore)).toEqual([7, 8]);
        expect(get(activeTabStore)).toBe('analysis');
    });

    test('passes a positive count through as LastN', async () => {
        statsFilterStore.set({ decisionType: -1 });
        GetPositionIDsByStatsSelection.mockResolvedValue([1]);
        LoadPositionIDsByFilters.mockResolvedValue([1]);

        await loadWorstBlunders(50);

        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith({ decisionType: -1 }, { Kind: 'top_blunders', LastN: 50 });
    });

    test('ignores a non-positive / non-integer count (backend default applies)', async () => {
        statsFilterStore.set({ decisionType: -1 });
        GetPositionIDsByStatsSelection.mockResolvedValue([1]);
        LoadPositionIDsByFilters.mockResolvedValue([1]);

        await loadWorstBlunders(0);
        expect(GetPositionIDsByStatsSelection).toHaveBeenLastCalledWith({ decisionType: -1 }, { Kind: 'top_blunders' });
    });
});

describe('loadPositionsFromTournament', () => {
    test('calls GetPositionIDsByTournament with tournamentID', async () => {
        GetPositionIDsByTournament.mockResolvedValue([5, 6]);
        LoadPositionIDsByFilters.mockResolvedValue([5, 6]);

        await loadPositionsFromTournament(42);
        expect(GetPositionIDsByTournament).toHaveBeenCalledWith(42);
    });
});

describe('loadPositionsFromMatch', () => {
    test('calls GetPositionIDsByMatch with matchID', async () => {
        GetPositionIDsByMatch.mockResolvedValue([7, 8]);
        LoadPositionIDsByFilters.mockResolvedValue([7, 8]);

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
        LoadPositionIDsByFilters.mockResolvedValue([1, 2, 3]);
        await loadPositionsFromSelection([1, 2, 3]);
        expect(LoadPositionIDsByFilters).toHaveBeenCalledWith(expect.objectContaining({ restrictToPositionIDs: '1,2,3' }));
    });
});

describe('openTournamentInPanel', () => {
    test('sets selectedTournamentStore and switches to tournaments tab', () => {
        openTournamentInPanel(7);
        expect(get(selectedTournamentStore)).toBe(7);
        expect(get(activeTabStore)).toBe('tournaments');
    });
});

describe('openMatchInPanel', () => {
    test('switches to matches tab', () => {
        activeTabStore.set('analysis');
        openMatchInPanel(12);
        expect(get(activeTabStore)).toBe('matches');
    });
});
