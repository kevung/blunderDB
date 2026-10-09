import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

// A fake backend: each breakdown row holds a list of positions, and the count
// endpoint reports their lengths — as the Go side computes both from one read.
/** @type {Record<string, Record<string, {all: number[], blunders: number[]}>>} */
const rows = {
    phase: { opening: { all: [1, 2, 3], blunders: [2] } },
    game_type: { holding: { all: [4, 5], blunders: /** @type {number[]} */ ([]) } },
    tag: { '#timing': { all: [6], blunders: [6] } },
    score: { money: { all: [7, 8], blunders: [8] }, '3-5': { all: [9, 10, 11, 12], blunders: [9, 12] } }
};

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetStatsBreakdownPositionCounts: vi.fn(async () =>
        Object.fromEntries(Object.entries(rows).map(([dim, keys]) => [dim, Object.fromEntries(Object.entries(keys).map(([k, r]) => [k, { Positions: r.all.length, Blunders: r.blunders.length }]))]))
    ),
    GetPositionIDsByStatsSelection: vi.fn(async (_filter, sel) => {
        const r = rows[sel.Breakdown]?.[sel.BreakdownKey];
        if (sel.Kind !== 'breakdown' || !r) return [];
        return sel.OnlyBlunders ? r.blunders : r.all;
    }),
    LoadPositionIDsByFilters: vi.fn(async ({ restrictToPositionIDs }) => restrictToPositionIDs.split(',').map(Number)),
    GetPositionIDsByTournament: vi.fn(),
    GetPositionIDsByMatch: vi.fn()
}));

vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/some/db.db') };
});

vi.mock('../stores/positionStore.js', () => {
    const { writable } = require('svelte/store');
    const store = writable(/** @type {number[]} */ ([]));
    return { positionsStore: { subscribe: store.subscribe, set: store.set, setIds: (/** @type {number[]} */ ids) => store.set(ids) } };
});

import StatsBreakdownsTab from '../components/stats/StatsBreakdownsTab.svelte';
import { positionsStore } from '../stores/positionStore.js';
import { statsFilterStore, statsResultFilterStore } from '../stores/statsStore.js';
import { GetPositionIDsByStatsSelection, GetStatsBreakdownPositionCounts } from '../../wailsjs/go/database/Database.js';

const iv = { available: true, low: 1, high: 2 };
const result = {
    PerPhase: [{ Phase: 'opening', PR: 3, PRInterval: iv, NumDecisions: 5, BlunderCount: 1 }],
    PerGameType: [{ GameType: 'holding', PR: 4, PRInterval: iv, NumDecisions: 2, BlunderCount: 0 }],
    PerTag: [{ Tag: '#timing', PR: 6, PRInterval: iv, NumDecisions: 1, BlunderCount: 1 }],
    PerScore: [
        { Money: true, MoverAway: 0, OpponentAway: 0, PR: 2, PRInterval: iv, NumDecisions: 3, BlunderCount: 1 },
        { Money: false, MoverAway: 3, OpponentAway: 5, PR: 5, PRInterval: iv, NumDecisions: 6, BlunderCount: 2 }
    ]
};

beforeEach(() => {
    positionsStore.set([]);
    statsResultFilterStore.set(null);
});
afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

describe('StatsBreakdownsTab — clickable counts', () => {
    const cases = [
        ['phase', 'opening'],
        ['game_type', 'holding'],
        ['tag', '#timing'],
        ['score', 'money'],
        ['score', '3-5']
    ];

    test.each(cases)('%s %s: the figure shown is the number of positions loaded', async (dim, key) => {
        render(StatsBreakdownsTab, { props: { result } });
        for (const kind of ['positions', 'blunders']) {
            const n = kind === 'blunders' ? rows[dim][key].blunders.length : rows[dim][key].all.length;
            if (n === 0) {
                // Nothing to open: the zero is shown, not offered as a link.
                await waitFor(() => expect(screen.queryByTestId(`breakdown-positions-${dim}-${key}`)).not.toBeNull());
                expect(screen.queryByTestId(`breakdown-${kind}-${dim}-${key}`)).toBeNull();
                continue;
            }
            const link = await screen.findByTestId(`breakdown-${kind}-${dim}-${key}`);
            const shown = Number(link.textContent);
            await fireEvent.click(link);
            await waitFor(() => expect(get(positionsStore)).toHaveLength(shown));
            expect(GetPositionIDsByStatsSelection).toHaveBeenLastCalledWith(expect.anything(), {
                Kind: 'breakdown',
                Breakdown: dim,
                BreakdownKey: key,
                OnlyBlunders: kind === 'blunders'
            });
        }
    });

    test('the decision count stays beside the position count, as the PR denominator', async () => {
        render(StatsBreakdownsTab, { props: { result } });
        const link = await screen.findByTestId('breakdown-positions-phase-opening');
        const row = /** @type {HTMLElement} */ (link.closest('tr'));
        expect(link.textContent).toBe('3');
        expect(row.textContent).toContain('5');
    });
});

describe('StatsBreakdownsTab — counts that cannot be read', () => {
    test('a failed count call shows on the cell instead of a wait', async () => {
        vi.mocked(GetStatsBreakdownPositionCounts).mockRejectedValueOnce(new Error('boom'));
        render(StatsBreakdownsTab, { props: { result } });
        const cell = await screen.findByTestId('breakdown-error-positions-phase-opening');
        expect(cell.getAttribute('title')).toContain('boom');
        expect(document.querySelector('.pending')).toBeNull();
    });

    test('a row the counts do not know shows, instead of a wait', async () => {
        vi.mocked(GetStatsBreakdownPositionCounts).mockResolvedValueOnce(/** @type {any} */ ({ phase: {}, game_type: {}, tag: {}, score: {} }));
        render(StatsBreakdownsTab, { props: { result } });
        await screen.findByTestId('breakdown-error-positions-phase-opening');
        expect(document.querySelector('.pending')).toBeNull();
    });

    test('counts and opens under the filter that produced the result, not the bar', async () => {
        const produced = { playerName: '', decisionType: -1 };
        statsResultFilterStore.set(produced);
        statsFilterStore.set({ playerName: 'Alice', decisionType: -1 });
        render(StatsBreakdownsTab, { props: { result } });
        const link = await screen.findByTestId('breakdown-positions-phase-opening');
        expect(GetStatsBreakdownPositionCounts).toHaveBeenLastCalledWith(produced);
        await fireEvent.click(link);
        await waitFor(() => expect(GetPositionIDsByStatsSelection).toHaveBeenLastCalledWith(produced, expect.objectContaining({ BreakdownKey: 'opening' })));
    });
});
