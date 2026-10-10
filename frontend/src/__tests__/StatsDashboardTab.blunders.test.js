import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetPositionIDsByStatsSelection: vi.fn(async () => [42]),
    GetPositionIDsByTournament: vi.fn(),
    GetPositionIDsByMatch: vi.fn(),
    LoadPositionIDsByFilters: vi.fn(async ({ restrictToPositionIDs }) => restrictToPositionIDs.split(',').map(Number))
}));

vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/some/db.db') };
});

vi.mock('../stores/positionStore.js', async (importOriginal) => {
    const { writable } = require('svelte/store');
    const store = writable(/** @type {number[]} */ ([]));
    return { ...(await importOriginal()), positionsStore: { subscribe: store.subscribe, set: store.set, setIds: (/** @type {number[]} */ ids) => store.set(ids) } };
});

import StatsDashboardTab from '../components/stats/StatsDashboardTab.svelte';
import { positionsStore } from '../stores/positionStore.js';
import { GetPositionIDsByStatsSelection } from '../../wailsjs/go/database/Database.js';

// The same position can be a top blunder in two matches: its id alone is not a row key.
const blunder = (/** @type {number} */ matchID) => ({ PositionID: 42, MatchID: matchID, TournamentID: 1, ErrorMP: 900, MWCLoss: 0.1, DecisionType: 0, MatchDate: '2025-01-01', PlayerNames: 'A vs B' });
const result = {
    Totals: { NumPositions: 10, NumMatches: 2, NumTournaments: 1, NumDecisions: 20 },
    PRGlobal: 4,
    PRChecker: 4,
    PRCube: 4,
    PRRolling: {},
    MWCAvailable: false,
    PerTournament: [],
    TopBlunders: [blunder(1), blunder(2)]
};

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

describe('StatsDashboardTab — top blunders', () => {
    test('every row is rendered, even when two share a position, and a click opens that position', async () => {
        render(StatsDashboardTab, { props: { result, metric: 'pr' } });
        const rows = document.querySelectorAll('.blunder-main');
        expect(rows).toHaveLength(2);
        await fireEvent.click(rows[1]);
        await waitFor(() => expect(get(positionsStore)).toEqual([42]));
        expect(GetPositionIDsByStatsSelection).toHaveBeenLastCalledWith(expect.anything(), { Kind: 'position', PositionID: 42 });
    });
});
