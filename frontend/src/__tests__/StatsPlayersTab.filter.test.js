/**
 * Picking a player in the Players table puts them on the shared filter: the bar shows the name
 * and the choice is saved. The filter's player is marked in the full table, and the row's decision
 * count opens that player's positions.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn(() =>
        Promise.resolve([
            { Name: 'Alice', Count: 3 },
            { Name: 'Bob', Count: 2 }
        ])
    ),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    GetStatsAnalysisEngines: vi.fn(() => Promise.resolve([])),
    GetStatsDateRange: vi.fn(() => Promise.resolve(null))
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn(() => Promise.resolve(null)),
    SaveStatsFilter: vi.fn(() => Promise.resolve()),
    GetTableSorts: vi.fn(() => Promise.resolve({})),
    SaveTableSort: vi.fn(() => Promise.resolve())
}));
vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/db'), databaseLoadedStore: writable(true) };
});
vi.mock('../services/positionLoader.js', () => ({
    loadPositionsFromStatsSelection: vi.fn(() => Promise.resolve()),
    openMatchListInPanel: vi.fn(() => Promise.resolve()),
    openStatsPlayerBlunders: vi.fn(() => Promise.resolve())
}));

import { SaveStatsFilter } from '../../wailsjs/go/main/Config.js';
import { loadPositionsFromStatsSelection, openStatsPlayerBlunders } from '../services/positionLoader.js';
import { statsFilterStore, playerTableFilterStore, patchStatsFilter, _resetStatsFilterForTests } from '../stores/statsStore.js';
import StatsFilterBar from '../components/stats/StatsFilterBar.svelte';
import StatsPlayersTab from '../components/stats/StatsPlayersTab.svelte';

const row = (/** @type {string} */ name, /** @type {number} */ pr) => ({
    name,
    matches: 3,
    wins: 2,
    losses: 1,
    decisions: 120,
    checker_decisions: 100,
    cube_decisions: 20,
    pr,
    pr_checker: pr,
    pr_cube: pr,
    snowie_er: pr,
    blunders: 4,
    luck_known: false
});
const ROWS = [row('Alice', 4), row('Bob', 6)];

beforeEach(() => {
    vi.clearAllMocks();
    _resetStatsFilterForTests();
});
afterEach(() => cleanup());

describe('Players › shared filter', () => {
    test('a row click shows the player in the bar and saves the choice', async () => {
        const bar = render(StatsFilterBar);
        const input = /** @type {HTMLInputElement} */ (await waitFor(() => bar.container.querySelector('#fb-player')));
        const onSelectPlayer = vi.fn();
        const { getByText } = render(StatsPlayersTab, { props: { rows: ROWS, onSelectPlayer } });

        await fireEvent.click(getByText('Bob'));

        expect(get(statsFilterStore).playerName).toBe('Bob');
        await waitFor(() => expect(input.value).toBe('Bob'));
        expect(onSelectPlayer).toHaveBeenCalledWith('Bob');
        await waitFor(() => expect(SaveStatsFilter).toHaveBeenCalledWith(expect.objectContaining({ player_name: 'Bob' })), { timeout: 2000 });
    });

    test('a refused date puts the field back to the applied one', async () => {
        const { container } = render(StatsFilterBar);
        const from = /** @type {HTMLInputElement} */ (await waitFor(() => container.querySelector('#fb-date-from')));
        const to = /** @type {HTMLInputElement} */ (container.querySelector('#fb-date-to'));
        await fireEvent.change(from, { target: { value: '2025-03-01' } });
        await fireEvent.change(to, { target: { value: '2025-01-01' } });
        expect(get(statsFilterStore).dateTo).toBe('');
        expect(to.value).toBe('');
        expect(from.value).toBe('2025-03-01');
    });

    test("the filter's player is marked in the full table", async () => {
        patchStatsFilter({ playerName: 'Bob' });
        const { container } = render(StatsPlayersTab, { props: { rows: ROWS } });
        const marked = [...container.querySelectorAll('tbody tr.selected')];
        expect(marked).toHaveLength(1);
        expect(marked[0].textContent).toContain('Bob');
        expect(container.querySelectorAll('tbody tr')).toHaveLength(2);
    });

    test("a row's decisions open that player's positions, under the table's filter, not the dashboard", async () => {
        patchStatsFilter({ dateFrom: '2025-01-01', decisionType: 1, analysisEngine: 'XG', minAnalysisDepth: 2 });
        const onSelectPlayer = vi.fn();
        const { container } = render(StatsPlayersTab, { props: { rows: ROWS, onSelectPlayer } });
        const row = [...container.querySelectorAll('tbody tr')].find((tr) => tr.textContent?.includes('Alice'));
        // Matches also count those without an analysed decision: no match list reproduces them.
        const links = /** @type {HTMLElement[]} */ ([...(row?.querySelectorAll('[data-testid="count-link"]') ?? [])]);
        expect(links.map((l) => l.textContent)).toEqual(['120', '4']);

        await fireEvent.click(links[0]);
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(
            expect.objectContaining({ playerName: 'Alice', dateFrom: '2025-01-01', decisionType: -1, analysisEngine: 'XG', minAnalysisDepth: 2 }),
            { Kind: 'all', OnlyWithError: false }
        );
        expect(onSelectPlayer).not.toHaveBeenCalled();
        expect(get(statsFilterStore).playerName).toBe('');
    });
});

describe('Players › counts read with the table', () => {
    afterEach(() => playerTableFilterStore.set(null));

    test('decisions and blunders open under the filter the table was read with, not the bar', async () => {
        playerTableFilterStore.set(/** @type {any} */ ({ decisionType: -1, playerName: '', tournamentIDs: [8], dateFrom: '', dateTo: '', matchLength: [] }));
        patchStatsFilter({ tournamentIDs: [9] });
        const { container } = render(StatsPlayersTab, { props: { rows: ROWS } });
        const row = [...container.querySelectorAll('tbody tr')].find((tr) => tr.textContent?.includes('Alice'));
        const [decisions, blunders] = /** @type {HTMLElement[]} */ ([...(row?.querySelectorAll('[data-testid="count-link"]') ?? [])]);

        await fireEvent.click(decisions);
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(expect.objectContaining({ playerName: 'Alice', tournamentIDs: [8] }), { Kind: 'all', OnlyWithError: false });
        await fireEvent.click(blunders);
        expect(openStatsPlayerBlunders).toHaveBeenCalledWith('Alice');
    });
});
