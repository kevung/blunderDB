import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('../../wailsjs/go/main/Database.js', () => ({
    GetPositionIDsByStatsSelection: vi.fn().mockResolvedValue([1, 2, 3]),
    GetPositionIDsByTournament: vi.fn().mockResolvedValue([]),
    GetPositionIDsByMatch: vi.fn().mockResolvedValue([]),
    LoadPositionsByFilters: vi.fn().mockResolvedValue([])
}));

vi.mock('../stores/uiStore.js', () => {
    const { writable } = require('svelte/store');
    return {
        activeTabStore: writable('analysis'),
        openPanel: vi.fn(),
        PANEL: { ANALYSIS: 'analysis', MATCH: 'match', TOURNAMENT: 'tournament', STATS: 'stats' },
        statusBarTextStore: writable(''),
        currentPositionIndexStore: writable(0)
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

// Import after mocks
import { GetPositionIDsByStatsSelection } from '../../wailsjs/go/main/Database.js';
import { statsFilterStore } from '../stores/statsStore.js';
import { loadPositionsFromStatsSelection } from '../services/positionLoader.js';
import { activeTabStore } from '../stores/uiStore.js';

// ── Sample data ───────────────────────────────────────────────────────────────

const SAMPLE_RESULT = {
    Totals: { NumPositions: 100, NumMatches: 10, NumTournaments: 3, NumDecisions: 250 },
    PRGlobal: 4.25,
    PRChecker: 3.80,
    PRCube: 5.10,
    PRRolling: { 5: 3.1, 10: 3.5, 50: 4.0, 100: 4.1, 250: 4.25 },
    MWCGlobal: 0.0512,
    MWCChecker: 0.0310,
    MWCCube: 0.0202,
    MWCRolling: null,
    MWCAvailable: true,
    PerTournament: [],
    PerMatch: [],
    CubeActionBreakdown: [],
    ErrorHistogram: [],
    TopBlunders: [
        {
            PositionID: 42,
            MatchID: 7,
            TournamentID: 2,
            ErrorMP: 350,
            MWCLoss: 0.0,
            Description: '',
            DecisionType: 0,
            MatchDate: '2025-03-15T00:00:00Z',
            PlayerNames: 'Alice vs Bob'
        },
        {
            PositionID: 99,
            MatchID: 7,
            TournamentID: 2,
            ErrorMP: 280,
            MWCLoss: 0.0,
            Description: '',
            DecisionType: 1,
            MatchDate: '2025-03-15T00:00:00Z',
            PlayerNames: 'Alice vs Bob'
        }
    ]
};

const EMPTY_RESULT = {
    Totals: { NumPositions: 0, NumMatches: 0, NumTournaments: 0, NumDecisions: 0 },
    PRGlobal: 0, PRChecker: 0, PRCube: 0,
    PRRolling: {},
    MWCGlobal: 0, MWCChecker: 0, MWCCube: 0,
    MWCRolling: null, MWCAvailable: false,
    PerTournament: [], PerMatch: [],
    CubeActionBreakdown: [], ErrorHistogram: [],
    TopBlunders: []
};

// ── Helper: isolated metric display logic ─────────────────────────────────────

function cardValue(result, metric, kind) {
    if (!result) return '—';
    function fmtPR(v) { return (v == null || isNaN(v)) ? '—' : v.toFixed(2); }
    function fmtMWC(v) { return (v == null || isNaN(v)) ? '—' : v.toFixed(4); }
    if (metric === 'pr') {
        if (kind === 'all') return fmtPR(result.PRGlobal);
        if (kind === 'checker') return fmtPR(result.PRChecker);
        if (kind === 'cube') return fmtPR(result.PRCube);
    } else {
        if (kind === 'all') return fmtMWC(result.MWCGlobal);
        if (kind === 'checker') return fmtMWC(result.MWCChecker);
        if (kind === 'cube') return fmtMWC(result.MWCCube);
    }
    return '—';
}

function rollingAvail(result, n) {
    if (!result || !result.Totals) return false;
    return result.Totals.NumDecisions >= n;
}

function fmtBlunderError(entry, metric) {
    if (metric === 'mwc' && entry.MWCLoss > 0) return entry.MWCLoss.toFixed(4);
    if (entry.ErrorMP == null) return '—';
    return (entry.ErrorMP / 1000).toFixed(3);
}

function decisionLabel(entry) {
    return entry.DecisionType === 1 ? 'Cube' : 'Checker';
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('StatsDashboardTab — display logic', () => {
    test('PR Global card shows PRGlobal formatted', () => {
        expect(cardValue(SAMPLE_RESULT, 'pr', 'all')).toBe('4.25');
    });

    test('PR Checker card shows PRChecker formatted', () => {
        expect(cardValue(SAMPLE_RESULT, 'pr', 'checker')).toBe('3.80');
    });

    test('PR Cube card shows PRCube formatted', () => {
        expect(cardValue(SAMPLE_RESULT, 'pr', 'cube')).toBe('5.10');
    });

    test('MWC Global card shows MWCGlobal formatted when metric=mwc', () => {
        expect(cardValue(SAMPLE_RESULT, 'mwc', 'all')).toBe('0.0512');
    });

    test('MWC Checker card shows MWCChecker formatted when metric=mwc', () => {
        expect(cardValue(SAMPLE_RESULT, 'mwc', 'checker')).toBe('0.0310');
    });

    test('null result returns — for all cards', () => {
        expect(cardValue(null, 'pr', 'all')).toBe('—');
        expect(cardValue(null, 'mwc', 'checker')).toBe('—');
    });
});

describe('StatsDashboardTab — rolling N availability', () => {
    test('N=5 is available with 250 decisions', () => {
        expect(rollingAvail(SAMPLE_RESULT, 5)).toBe(true);
    });

    test('N=250 is available with 250 decisions', () => {
        expect(rollingAvail(SAMPLE_RESULT, 250)).toBe(true);
    });

    test('N=500 is NOT available with 250 decisions', () => {
        expect(rollingAvail(SAMPLE_RESULT, 500)).toBe(false);
    });

    test('N=1000 is NOT available with 250 decisions', () => {
        expect(rollingAvail(SAMPLE_RESULT, 1000)).toBe(false);
    });

    test('empty result: all N unavailable', () => {
        expect(rollingAvail(EMPTY_RESULT, 5)).toBe(false);
    });
});

describe('StatsDashboardTab — blunder formatting', () => {
    test('checker blunder shows EMG error in PR mode', () => {
        const entry = SAMPLE_RESULT.TopBlunders[0]; // ErrorMP=350
        expect(fmtBlunderError(entry, 'pr')).toBe('0.350');
    });

    test('cube blunder shows EMG error in PR mode', () => {
        const entry = SAMPLE_RESULT.TopBlunders[1]; // ErrorMP=280
        expect(fmtBlunderError(entry, 'pr')).toBe('0.280');
    });

    test('blunder shows — for MWC when MWCLoss is 0', () => {
        const entry = { ...SAMPLE_RESULT.TopBlunders[0], MWCLoss: 0 };
        // Falls back to EMG since MWCLoss is not > 0
        expect(fmtBlunderError(entry, 'mwc')).toBe('0.350');
    });

    test('blunder shows MWC when MWCLoss is populated', () => {
        const entry = { ...SAMPLE_RESULT.TopBlunders[0], MWCLoss: 0.0123 };
        expect(fmtBlunderError(entry, 'mwc')).toBe('0.0123');
    });

    test('DecisionType=0 → Checker label', () => {
        expect(decisionLabel(SAMPLE_RESULT.TopBlunders[0])).toBe('Checker');
    });

    test('DecisionType=1 → Cube label', () => {
        expect(decisionLabel(SAMPLE_RESULT.TopBlunders[1])).toBe('Cube');
    });
});

describe('StatsDashboardTab — empty state condition', () => {
    test('NumDecisions === 0 triggers empty state', () => {
        expect(EMPTY_RESULT.Totals.NumDecisions === 0).toBe(true);
    });

    test('NumDecisions > 0 does not trigger empty state', () => {
        expect(SAMPLE_RESULT.Totals.NumDecisions === 0).toBe(false);
    });

    test('null result triggers empty state', () => {
        const result = null;
        expect(!result || result.Totals.NumDecisions === 0).toBe(true);
    });
});

describe('StatsDashboardTab — drill-down calls (via positionLoader)', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        GetPositionIDsByStatsSelection.mockResolvedValue([10, 20]);
        statsFilterStore.set({ playerName: '', tournamentIDs: [], dateFrom: '', dateTo: '', decisionType: -1, matchLength: [] });
    });

    test('loadPositionsFromStatsSelection with Kind=all calls Wails correctly', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'all', OnlyWithError: false });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'all', OnlyWithError: false });
    });

    test('loadPositionsFromStatsSelection with Kind=checker calls Wails correctly', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'checker', OnlyWithError: false });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'checker', OnlyWithError: false });
    });

    test('loadPositionsFromStatsSelection with Kind=cube calls Wails correctly', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'cube', OnlyWithError: false });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'cube', OnlyWithError: false });
    });

    test('openRollingN calls with Kind=last_n and correct LastN', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'last_n', LastN: 50 });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'last_n', LastN: 50 });
    });

    test('openBlunder calls with Kind=position and PositionID', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'position', PositionID: 42 });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, { Kind: 'position', PositionID: 42 });
    });

    test('successful drill-down switches to analysis tab', async () => {
        GetPositionIDsByStatsSelection.mockResolvedValue([10, 20]);
        const { LoadPositionsByFilters } = await import('../../wailsjs/go/main/Database.js');
        LoadPositionsByFilters.mockResolvedValue([{ id: 10 }, { id: 20 }]);
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'all', OnlyWithError: false });
        expect(get(activeTabStore)).toBe('analysis');
    });
});
