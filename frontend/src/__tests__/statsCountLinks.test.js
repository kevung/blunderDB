import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor, screen } from '@testing-library/svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetPositionIDsByStatsSelection: vi.fn(async () => [42]),
    GetPositionIDsByTournament: vi.fn(),
    GetPositionIDsByMatch: vi.fn(),
    LoadPositionIDsByFilters: vi.fn(async ({ restrictToPositionIDs }) => restrictToPositionIDs.split(',').map(Number)),
    StatsTournamentIDs: vi.fn(async () => [7])
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
import StatsPlayersTab from '../components/stats/StatsPlayersTab.svelte';
import { tournamentOpenRequestStore } from '../stores/tournamentStore.js';
import { statsFilterStore } from '../stores/statsStore.js';
import { activeTabStore } from '../stores/uiStore.js';
import { GetPositionIDsByStatsSelection, StatsTournamentIDs } from '../../wailsjs/go/database/Database.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

const result = /** @type {any} */ ({
    Totals: { NumPositions: 10, NumMatches: 2, NumTournaments: 1, NumDecisions: 20 },
    PRGlobal: 4,
    PRChecker: 4,
    PRCube: 4,
    PRRolling: {},
    MWCAvailable: false,
    PerTournament: [],
    TopBlunders: []
});

describe('Stats counts that open what they count', () => {
    test("the Dashboard's single tournament opens in the Tournaments panel", async () => {
        tournamentOpenRequestStore.set(null);
        render(StatsDashboardTab, { props: { result, metric: 'pr' } });
        const link = screen.getAllByTestId('count-link').find((b) => b.closest('.stats-totals'));
        await fireEvent.click(/** @type {HTMLElement} */ (link));
        await waitFor(() => expect(get(tournamentOpenRequestStore)).toBe(7));
        expect(StatsTournamentIDs).toHaveBeenCalled();
        expect(get(activeTabStore)).toBe('tournaments');
    });

    test("a Players row's blunders open its blunders without filtering on the player", async () => {
        statsFilterStore.update((f) => ({ ...f, playerName: '' }));
        const row = {
            name: 'Bob',
            matches: 1,
            wins: 1,
            losses: 0,
            decisions: 10,
            checker_decisions: 10,
            cube_decisions: 0,
            pr: 3,
            pr_checker: 3,
            pr_cube: 0,
            snowie_er: 3,
            blunders: 2,
            errors: 3,
            luck_known: false
        };
        render(StatsPlayersTab, { props: { rows: [row] } });
        await fireEvent.click(screen.getByRole('button', { name: '2' }));
        await waitFor(() => expect(GetPositionIDsByStatsSelection).toHaveBeenCalledWith(expect.objectContaining({ playerName: 'Bob', decisionType: -1 }), { Kind: 'blunders' }));
        expect(get(statsFilterStore).playerName).toBe('');
    });
});
