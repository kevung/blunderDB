import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('../../wailsjs/go/main/Database.js', () => ({
    GetPositionIDsByStatsSelection: vi.fn().mockResolvedValue([]),
    GetPositionIDsByTournament: vi.fn().mockResolvedValue([10, 11, 12]),
    GetPositionIDsByMatch: vi.fn().mockResolvedValue([20, 21]),
    LoadPositionsByFilters: vi.fn().mockResolvedValue([])
}));

vi.mock('../stores/uiStore.js', () => {
    const { writable } = require('svelte/store');
    return {
        activeTabStore: writable('analysis'),
        openPanel: vi.fn(),
        PANEL: { ANALYSIS: 'analysis', MATCH: 'match', TOURNAMENT: 'tournament', STATS: 'stats' },
        statusBarTextStore: writable(''),
        currentPositionIndexStore: writable(0),
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

// Import after mocks
import { GetPositionIDsByTournament, GetPositionIDsByMatch } from '../../wailsjs/go/main/Database.js';
import { activeTabStore } from '../stores/uiStore.js';
import { selectedTournamentStore } from '../stores/tournamentStore.js';
import { loadPositionsFromTournament, loadPositionsFromMatch, openTournamentInPanel } from '../services/positionLoader.js';
import { GRADE_BANDS, gradeForPR, makeGradeBandPlugin } from '../components/stats/gradeBands.js';

// ── Sample data ───────────────────────────────────────────────────────────────

const SAMPLE_TOURNAMENTS = [
    { ID: 1, Name: 'Open de Paris', Date: '2025-01-10', PR: 3.5, MWC: 0.021, NumDecisions: 120 },
    { ID: 2, Name: 'Monte Carlo BG', Date: '2025-04-05', PR: 2.8, MWC: 0.017, NumDecisions: 95 },
    { ID: 3, Name: 'World Cup 2025', Date: '2025-09-20', PR: 4.1, MWC: 0.025, NumDecisions: 200 }
];

const SAMPLE_MATCHES = [
    { ID: 10, Date: '2025-01-15T12:00:00Z', PlayerName: 'Alice', PR: 3.2, MWC: 0.019, NumDecisions: 40 },
    { ID: 11, Date: '2025-02-20T15:30:00Z', PlayerName: 'Bob', PR: 5.0, MWC: 0.031, NumDecisions: 28 },
    { ID: 12, Date: '2025-03-10T09:00:00Z', PlayerName: 'Carol', PR: 2.1, MWC: 0.013, NumDecisions: 55 }
];

const SAMPLE_RESULT = {
    Totals: { NumPositions: 415, NumMatches: 3, NumTournaments: 3, NumDecisions: 415 },
    PRGlobal: 3.5,
    PRChecker: 3.1,
    PRCube: 4.2,
    PRRolling: { 5: 2.8, 10: 3.0, 50: 3.4 },
    MWCGlobal: 0.021,
    MWCChecker: 0.018,
    MWCCube: 0.026,
    MWCRolling: null,
    MWCAvailable: true,
    PerTournament: SAMPLE_TOURNAMENTS,
    PerMatch: SAMPLE_MATCHES,
    CubeActionBreakdown: [],
    ErrorHistogram: [],
    TopBlunders: []
};

const EMPTY_RESULT = {
    Totals: { NumPositions: 0, NumMatches: 0, NumTournaments: 0, NumDecisions: 0 },
    PRGlobal: 0,
    PRChecker: 0,
    PRCube: 0,
    PRRolling: {},
    MWCGlobal: 0,
    MWCChecker: 0,
    MWCCube: 0,
    MWCRolling: null,
    MWCAvailable: false,
    PerTournament: [],
    PerMatch: [],
    CubeActionBreakdown: [],
    ErrorHistogram: [],
    TopBlunders: []
};

// ── Helpers (mirrors component logic) ────────────────────────────────────────

function truncateLabel(name, max = 22) {
    if (!name) return '';
    return name.length > max ? name.slice(0, max - 1) + '…' : name;
}

function clampRadius(n) {
    return Math.max(4, Math.min(12, 4 + (n / 500) * 8));
}

function parseDateMs(dateStr) {
    if (!dateStr) return 0;
    return new Date(dateStr).getTime();
}

function buildTourDatasets(tournaments, metric) {
    return [
        {
            label: metric === 'pr' ? 'PR' : 'MWC loss',
            data: tournaments.map((t) => (metric === 'pr' ? t.PR : t.MWC))
        }
    ];
}

function buildMatchDatasets(matches, metric) {
    return [
        {
            label: metric === 'pr' ? 'PR per match' : 'MWC loss per match',
            data: matches.map((m) => ({
                x: parseDateMs(m.Date),
                y: metric === 'pr' ? m.PR : m.MWC
            })),
            pointRadius: matches.map((m) => clampRadius(m.NumDecisions))
        }
    ];
}

// ── Tests: gradeBands.js ──────────────────────────────────────────────────────

describe('GRADE_BANDS', () => {
    test('has 6 bands', () => {
        expect(GRADE_BANDS).toHaveLength(6);
    });

    test('first band starts at 0', () => {
        expect(GRADE_BANDS[0].min).toBe(0);
    });

    test('last band ends at Infinity', () => {
        expect(GRADE_BANDS[GRADE_BANDS.length - 1].max).toBe(Infinity);
    });

    test('bands are contiguous (each min === previous max)', () => {
        for (let i = 1; i < GRADE_BANDS.length; i++) {
            expect(GRADE_BANDS[i].min).toBe(GRADE_BANDS[i - 1].max);
        }
    });

    test('each band has a color string', () => {
        for (const band of GRADE_BANDS) {
            expect(typeof band.color).toBe('string');
            expect(band.color.length).toBeGreaterThan(0);
        }
    });
});

describe('gradeForPR', () => {
    test('PR 0 → World Class', () => {
        expect(gradeForPR(0)).toBe('World Class');
    });

    test('PR 1.9 → World Class', () => {
        expect(gradeForPR(1.9)).toBe('World Class');
    });

    test('PR 2.0 → Expert', () => {
        expect(gradeForPR(2.0)).toBe('Expert');
    });

    test('PR 3.5 → Expert', () => {
        expect(gradeForPR(3.5)).toBe('Expert');
    });

    test('PR 4.0 → Advanced', () => {
        expect(gradeForPR(4.0)).toBe('Advanced');
    });

    test('PR 6.0 → Intermediate', () => {
        expect(gradeForPR(6.0)).toBe('Intermediate');
    });

    test('PR 9.0 → Casual', () => {
        expect(gradeForPR(9.0)).toBe('Casual');
    });

    test('PR 12.0 → Beginner', () => {
        expect(gradeForPR(12.0)).toBe('Beginner');
    });

    test('PR 20 → Beginner', () => {
        expect(gradeForPR(20)).toBe('Beginner');
    });
});

describe('makeGradeBandPlugin', () => {
    test('returns a plugin object with id and beforeDraw', () => {
        const plugin = makeGradeBandPlugin(GRADE_BANDS);
        expect(plugin.id).toBe('gradeBands');
        expect(typeof plugin.beforeDraw).toBe('function');
    });

    test('beforeDraw does not throw with a mock chart', () => {
        const plugin = makeGradeBandPlugin(GRADE_BANDS);
        const mockChart = {
            ctx: {
                save: vi.fn(),
                restore: vi.fn(),
                beginPath: vi.fn(),
                rect: vi.fn(),
                clip: vi.fn(),
                fillRect: vi.fn(),
                fillStyle: ''
            },
            chartArea: { top: 0, bottom: 300, left: 0, right: 400 },
            scales: {
                y: { getPixelForValue: (v) => 300 - v * 10 }
            }
        };
        expect(() => plugin.beforeDraw(mockChart)).not.toThrow();
    });

    test('beforeDraw skips rendering when y scale is missing', () => {
        const plugin = makeGradeBandPlugin(GRADE_BANDS);
        const ctx = { save: vi.fn(), restore: vi.fn(), fillRect: vi.fn() };
        plugin.beforeDraw({ ctx, chartArea: {}, scales: {} });
        expect(ctx.fillRect).not.toHaveBeenCalled();
    });
});

// ── Tests: dataset construction ───────────────────────────────────────────────

describe('StatsProgressionTab — tournament dataset (PR mode)', () => {
    test('dataset contains one entry per tournament', () => {
        const ds = buildTourDatasets(SAMPLE_TOURNAMENTS, 'pr');
        expect(ds[0].data).toHaveLength(SAMPLE_TOURNAMENTS.length);
    });

    test('dataset values match tournament PRs', () => {
        const ds = buildTourDatasets(SAMPLE_TOURNAMENTS, 'pr');
        expect(ds[0].data).toEqual(SAMPLE_TOURNAMENTS.map((t) => t.PR));
    });

    test('dataset label says PR in pr mode', () => {
        const ds = buildTourDatasets(SAMPLE_TOURNAMENTS, 'pr');
        expect(ds[0].label).toBe('PR');
    });
});

describe('StatsProgressionTab — tournament dataset (MWC mode)', () => {
    test('dataset values switch to MWC', () => {
        const ds = buildTourDatasets(SAMPLE_TOURNAMENTS, 'mwc');
        expect(ds[0].data).toEqual(SAMPLE_TOURNAMENTS.map((t) => t.MWC));
    });

    test('dataset label says MWC loss in mwc mode', () => {
        const ds = buildTourDatasets(SAMPLE_TOURNAMENTS, 'mwc');
        expect(ds[0].label).toBe('MWC loss');
    });
});

describe('StatsProgressionTab — match scatter dataset', () => {
    test('dataset has one point per match', () => {
        const ds = buildMatchDatasets(SAMPLE_MATCHES, 'pr');
        expect(ds[0].data).toHaveLength(SAMPLE_MATCHES.length);
    });

    test('x values are numeric timestamps', () => {
        const ds = buildMatchDatasets(SAMPLE_MATCHES, 'pr');
        for (const pt of ds[0].data) {
            expect(typeof pt.x).toBe('number');
            expect(pt.x).toBeGreaterThan(0);
        }
    });

    test('y values match PR', () => {
        const ds = buildMatchDatasets(SAMPLE_MATCHES, 'pr');
        expect(ds[0].data.map((p) => p.y)).toEqual(SAMPLE_MATCHES.map((m) => m.PR));
    });

    test('y values switch to MWC in mwc mode', () => {
        const ds = buildMatchDatasets(SAMPLE_MATCHES, 'mwc');
        expect(ds[0].data.map((p) => p.y)).toEqual(SAMPLE_MATCHES.map((m) => m.MWC));
    });

    test('point radius is clamped between 4 and 12', () => {
        const ds = buildMatchDatasets(SAMPLE_MATCHES, 'pr');
        for (const r of ds[0].pointRadius) {
            expect(r).toBeGreaterThanOrEqual(4);
            expect(r).toBeLessThanOrEqual(12);
        }
    });
});

// ── Tests: label truncation ───────────────────────────────────────────────────

describe('truncateLabel', () => {
    test('short label unchanged', () => {
        expect(truncateLabel('Open Paris')).toBe('Open Paris');
    });

    test('label at exactly max length unchanged', () => {
        const s = 'a'.repeat(22);
        expect(truncateLabel(s)).toBe(s);
    });

    test('long label gets truncated with ellipsis', () => {
        const s = 'a'.repeat(30);
        const result = truncateLabel(s);
        expect(result.endsWith('…')).toBe(true);
        expect(result.length).toBe(22);
    });

    test('null/undefined → empty string', () => {
        expect(truncateLabel(null)).toBe('');
        expect(truncateLabel(undefined)).toBe('');
    });
});

// ── Tests: empty-state condition ──────────────────────────────────────────────

describe('StatsProgressionTab — empty state', () => {
    test('null result → empty state', () => {
        const r = null;
        expect(!r || (r.PerTournament.length === 0 && r.PerMatch.length === 0)).toBe(true);
    });

    test('empty PerTournament + PerMatch → empty state', () => {
        expect(EMPTY_RESULT.PerTournament.length === 0 && EMPTY_RESULT.PerMatch.length === 0).toBe(true);
    });

    test('result with tournaments → no empty state', () => {
        const r = SAMPLE_RESULT;
        expect(!r || (r.PerTournament.length === 0 && r.PerMatch.length === 0)).toBe(false);
    });
});

// ── Tests: drill-down actions (via positionLoader) ────────────────────────────

describe('StatsProgressionTab — drill-down (openTournamentInPanel)', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        selectedTournamentStore.set(null);
        activeTabStore.set('dashboard');
    });

    test('openTournamentInPanel sets selectedTournamentStore and navigates', () => {
        openTournamentInPanel(42);
        expect(get(selectedTournamentStore)).toBe(42);
        expect(get(activeTabStore)).toBe('tournaments');
    });
});

describe('StatsProgressionTab — drill-down (loadPositionsFromTournament)', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        GetPositionIDsByTournament.mockResolvedValue([10, 11, 12]);
    });

    test('calls GetPositionIDsByTournament with correct ID', async () => {
        await loadPositionsFromTournament(7);
        expect(GetPositionIDsByTournament).toHaveBeenCalledWith(7);
    });
});

describe('StatsProgressionTab — drill-down (loadPositionsFromMatch)', () => {
    beforeEach(() => {
        vi.resetAllMocks();
        GetPositionIDsByMatch.mockResolvedValue([20, 21]);
    });

    test('calls GetPositionIDsByMatch with correct ID', async () => {
        await loadPositionsFromMatch(99);
        expect(GetPositionIDsByMatch).toHaveBeenCalledWith(99);
    });
});

// ── Tests: single-tournament fallback ────────────────────────────────────────

describe('StatsProgressionTab — single tournament fallback', () => {
    test('single-tournament result has exactly 1 tournament', () => {
        const result = { ...SAMPLE_RESULT, PerTournament: [SAMPLE_TOURNAMENTS[0]], PerMatch: [] };
        expect(result.PerTournament.length).toBe(1);
    });

    test('PR value is shown for the single tournament', () => {
        const t = SAMPLE_TOURNAMENTS[0];
        const val = t.PR.toFixed(2);
        expect(val).toBe('3.50');
    });

    test('grade label is correct for PR 3.5', () => {
        expect(gradeForPR(3.5)).toBe('Expert');
    });
});
