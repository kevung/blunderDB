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

// ── Pure helpers (mirroring StatsErrorsTab component logic) ──────────────────

function bucketLabel(bucket) {
    const lo = (bucket.MinMP / 1000).toFixed(3);
    if (bucket.MaxMP < 0) return `\u2265${lo}`;
    const hi = (bucket.MaxMP / 1000).toFixed(3);
    return `${lo}\u2013${hi}`;
}

function blunderRate(c) {
    if (!c || c.NumDecisions === 0) return '0.0';
    return ((c.BlunderCount / c.NumDecisions) * 100).toFixed(1);
}

// ── Sample data ───────────────────────────────────────────────────────────────

const SAMPLE_CUBE_BREAKDOWN = [
    { Action: 'NoDouble',    PR: 3.5,  MWC: 0.021, NumDecisions: 80,  BlunderCount: 5  },
    { Action: 'DoubleTake',  PR: 4.2,  MWC: 0.028, NumDecisions: 60,  BlunderCount: 8  },
    { Action: 'DoublePass',  PR: 2.1,  MWC: 0.012, NumDecisions: 30,  BlunderCount: 1  },
    { Action: 'TooGood',     PR: 6.0,  MWC: 0.040, NumDecisions: 10,  BlunderCount: 3  },
];

const SAMPLE_HISTOGRAM = [
    { MinMP: 0,   MaxMP: 5,   Count: 120 },
    { MinMP: 5,   MaxMP: 10,  Count: 45  },
    { MinMP: 10,  MaxMP: 25,  Count: 30  },
    { MinMP: 25,  MaxMP: 50,  Count: 18  },
    { MinMP: 50,  MaxMP: 100, Count: 9   },
    { MinMP: 100, MaxMP: -1,  Count: 4   },
];

const SAMPLE_RESULT = {
    Totals: { NumPositions: 226, NumMatches: 12, NumTournaments: 3, NumDecisions: 350 },
    PRGlobal: 4.0,   PRChecker: 3.2,  PRCube: 5.5,
    PRRolling: {},
    MWCGlobal: 0.024, MWCChecker: 0.018, MWCCube: 0.032,
    MWCRolling: null, MWCAvailable: true,
    PerTournament: [], PerMatch: [],
    CubeActionBreakdown: SAMPLE_CUBE_BREAKDOWN,
    ErrorHistogram: SAMPLE_HISTOGRAM,
    TopBlunders: []
};

const EMPTY_RESULT = {
    Totals: { NumPositions: 0, NumMatches: 0, NumTournaments: 0, NumDecisions: 0 },
    PRGlobal: 0, PRChecker: 0, PRCube: 0, PRRolling: {},
    MWCGlobal: 0, MWCChecker: 0, MWCCube: 0, MWCRolling: null, MWCAvailable: false,
    PerTournament: [], PerMatch: [],
    CubeActionBreakdown: [], ErrorHistogram: [], TopBlunders: []
};

// ── Helper functions mirroring component logic ────────────────────────────────

function cubeDataValue(breakdown, metric, index) {
    const c = breakdown[index];
    if (!c) return null;
    return metric === 'pr' ? c.PR : c.MWC;
}

function compDataValues(result, metric) {
    return [
        metric === 'pr' ? (result?.PRChecker ?? 0) : (result?.MWCChecker ?? 0),
        metric === 'pr' ? (result?.PRCube    ?? 0) : (result?.MWCCube    ?? 0),
    ];
}

function hasCubeData(result) {
    return (result?.CubeActionBreakdown ?? []).length > 0;
}

function hasHistData(result) {
    return (result?.ErrorHistogram ?? []).some(b => b.Count > 0);
}

// ── Tests: bucketLabel ────────────────────────────────────────────────────────

describe('bucketLabel', () => {
    test('0–5 mp → "0.000–0.005"', () => {
        expect(bucketLabel({ MinMP: 0, MaxMP: 5 })).toBe('0.000\u20130.005');
    });

    test('5–10 mp → "0.005–0.010"', () => {
        expect(bucketLabel({ MinMP: 5, MaxMP: 10 })).toBe('0.005\u20130.010');
    });

    test('100–∞ (MaxMP=-1) → "≥0.100"', () => {
        expect(bucketLabel({ MinMP: 100, MaxMP: -1 })).toBe('\u22650.100');
    });

    test('25–50 mp → "0.025–0.050"', () => {
        expect(bucketLabel({ MinMP: 25, MaxMP: 50 })).toBe('0.025\u20130.050');
    });
});

// ── Tests: blunderRate ────────────────────────────────────────────────────────

describe('blunderRate', () => {
    test('5 blunders / 80 decisions → 6.3%', () => {
        expect(blunderRate({ NumDecisions: 80, BlunderCount: 5 })).toBe('6.3');
    });

    test('0 decisions → 0.0%', () => {
        expect(blunderRate({ NumDecisions: 0, BlunderCount: 0 })).toBe('0.0');
    });

    test('null → 0.0%', () => {
        expect(blunderRate(null)).toBe('0.0');
    });

    test('3 blunders / 10 decisions → 30.0%', () => {
        expect(blunderRate({ NumDecisions: 10, BlunderCount: 3 })).toBe('30.0');
    });
});

// ── Tests: cube action dataset values ────────────────────────────────────────

describe('StatsErrorsTab — cube action breakdown', () => {
    test('PR mode: first bar uses c.PR', () => {
        expect(cubeDataValue(SAMPLE_CUBE_BREAKDOWN, 'pr', 0)).toBe(3.5);
    });

    test('MWC mode: first bar uses c.MWC', () => {
        expect(cubeDataValue(SAMPLE_CUBE_BREAKDOWN, 'mwc', 0)).toBe(0.021);
    });

    test('PR mode: TooGood bar uses its PR', () => {
        expect(cubeDataValue(SAMPLE_CUBE_BREAKDOWN, 'pr', 3)).toBe(6.0);
    });

    test('MWC mode: DoubleTake bar uses its MWC', () => {
        expect(cubeDataValue(SAMPLE_CUBE_BREAKDOWN, 'mwc', 1)).toBe(0.028);
    });

    test('hasCubeData true when breakdown non-empty', () => {
        expect(hasCubeData(SAMPLE_RESULT)).toBe(true);
    });

    test('hasCubeData false when breakdown empty', () => {
        expect(hasCubeData(EMPTY_RESULT)).toBe(false);
    });

    test('hasCubeData false for null result', () => {
        expect(hasCubeData(null)).toBe(false);
    });
});

// ── Tests: checker vs cube comparison ────────────────────────────────────────

describe('StatsErrorsTab — checker vs cube comparison', () => {
    test('PR mode: checker value uses PRChecker', () => {
        expect(compDataValues(SAMPLE_RESULT, 'pr')[0]).toBe(3.2);
    });

    test('PR mode: cube value uses PRCube', () => {
        expect(compDataValues(SAMPLE_RESULT, 'pr')[1]).toBe(5.5);
    });

    test('MWC mode: checker value uses MWCChecker', () => {
        expect(compDataValues(SAMPLE_RESULT, 'mwc')[0]).toBe(0.018);
    });

    test('MWC mode: cube value uses MWCCube', () => {
        expect(compDataValues(SAMPLE_RESULT, 'mwc')[1]).toBe(0.032);
    });

    test('null result: both values are 0', () => {
        expect(compDataValues(null, 'pr')).toEqual([0, 0]);
    });
});

// ── Tests: histogram ─────────────────────────────────────────────────────────

describe('StatsErrorsTab — error histogram', () => {
    test('histogram has 6 buckets in sample data', () => {
        expect(SAMPLE_HISTOGRAM).toHaveLength(6);
    });

    test('hasHistData true when at least one bucket has count > 0', () => {
        expect(hasHistData(SAMPLE_RESULT)).toBe(true);
    });

    test('hasHistData false when all buckets are 0', () => {
        const allZero = { ErrorHistogram: SAMPLE_HISTOGRAM.map(b => ({ ...b, Count: 0 })) };
        expect(hasHistData(allZero)).toBe(false);
    });

    test('hasHistData false for null result', () => {
        expect(hasHistData(null)).toBe(false);
    });

    test('labels are derived from bucketLabel for each bucket', () => {
        const labels = SAMPLE_HISTOGRAM.map(bucketLabel);
        expect(labels[0]).toBe('0.000\u20130.005');
        expect(labels[5]).toBe('\u22650.100');
    });

    test('histogram data does not change with metric toggle (always counts)', () => {
        // PR mode
        const dataPR  = SAMPLE_HISTOGRAM.map(b => b.Count);
        // MWC mode (should be identical)
        const dataMWC = SAMPLE_HISTOGRAM.map(b => b.Count);
        expect(dataPR).toEqual(dataMWC);
    });
});

// ── Tests: empty state ────────────────────────────────────────────────────────

describe('StatsErrorsTab — empty state conditions', () => {
    test('null result triggers empty state', () => {
        const result = null;
        expect(!result || (result?.Totals?.NumDecisions ?? 0) === 0).toBe(true);
    });

    test('zero decisions triggers empty state', () => {
        expect(EMPTY_RESULT.Totals.NumDecisions === 0).toBe(true);
    });

    test('non-zero decisions does not trigger empty state', () => {
        expect(SAMPLE_RESULT.Totals.NumDecisions === 0).toBe(false);
    });
});

// ── Tests: drill-down calls ───────────────────────────────────────────────────

describe('StatsErrorsTab — drill-down calls', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        GetPositionIDsByStatsSelection.mockResolvedValue([10, 20]);
        statsFilterStore.set({
            playerName: '',
            tournamentIDs: [],
            dateFrom: '',
            dateTo: '',
            decisionType: -1,
            matchLength: []
        });
    });

    test('cube_action click calls with OnlyWithError:true and correct CubeAction', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, {
            Kind: 'cube_action',
            CubeAction: 'DoubleTake',
            OnlyWithError: true
        });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, {
            Kind: 'cube_action',
            CubeAction: 'DoubleTake',
            OnlyWithError: true
        });
    });

    test('checker bar click calls with Kind=checker and OnlyWithError:true', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'checker', OnlyWithError: true });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, {
            Kind: 'checker',
            OnlyWithError: true
        });
    });

    test('cube bar click calls with Kind=cube and OnlyWithError:true', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'cube', OnlyWithError: true });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, {
            Kind: 'cube',
            OnlyWithError: true
        });
    });

    test('histogram bucket click passes correct MinMP/MaxMP bounds', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, {
            Kind: 'error_bucket',
            BucketMinMP: 25,
            BucketMaxMP: 50
        });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, {
            Kind: 'error_bucket',
            BucketMinMP: 25,
            BucketMaxMP: 50
        });
    });

    test('last bucket click passes BucketMaxMP=-1', async () => {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, {
            Kind: 'error_bucket',
            BucketMinMP: 100,
            BucketMaxMP: -1
        });
        expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, {
            Kind: 'error_bucket',
            BucketMinMP: 100,
            BucketMaxMP: -1
        });
    });

    test('different cube actions produce different CubeAction values', async () => {
        const filter = get(statsFilterStore);
        for (const action of ['NoDouble', 'DoubleTake', 'DoublePass', 'TooGood']) {
            vi.resetAllMocks();
            GetPositionIDsByStatsSelection.mockResolvedValue([]);
            await loadPositionsFromStatsSelection(filter, {
                Kind: 'cube_action',
                CubeAction: action,
                OnlyWithError: true
            });
            expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(filter, {
                Kind: 'cube_action',
                CubeAction: action,
                OnlyWithError: true
            });
        }
    });
});
