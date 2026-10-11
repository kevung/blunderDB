/**
 * Picking a player in the Players table puts them on the shared filter: the bar shows the name
 * and the choice is saved. The filter's player is marked in the full table, and the row's counts
 * open that player's matches or decisions.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', async () => {
    const { createDatabaseMock } = await import('../__mocks__/wails.js');
    return createDatabaseMock({
        GetAllPlayerNames: vi.fn(() =>
            Promise.resolve([
                { Name: 'Alice', Count: 3 },
                { Name: 'Bob', Count: 2 }
            ])
        ),
        GetAllTournaments: vi.fn(() => Promise.resolve([])),
        GetStatsAnalysisEngines: vi.fn(() => Promise.resolve([])),
        GetStatsDateRange: vi.fn(() => Promise.resolve(null)),
        StatsMatchIDs: vi.fn(() => Promise.resolve([5, 6]))
    });
});
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn(() => Promise.resolve(null)),
    SaveStatsFilter: vi.fn(() => Promise.resolve())
}));
vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/db'), databaseLoadedStore: writable(true) };
});
vi.mock('../services/positionLoader.js', () => ({
    loadPositionsFromStatsSelection: vi.fn(() => Promise.resolve()),
    openMatchListInPanel: vi.fn(() => Promise.resolve())
}));

import { SaveStatsFilter } from '../../wailsjs/go/main/Config.js';
import { StatsMatchIDs } from '../../wailsjs/go/database/Database.js';
import { loadPositionsFromStatsSelection, openMatchListInPanel } from '../services/positionLoader.js';
import { statsFilterStore, patchStatsFilter, _resetStatsFilterForTests } from '../stores/statsStore.js';
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

    test("the filter's player is marked in the full table", async () => {
        patchStatsFilter({ playerName: 'Bob' });
        const { container } = render(StatsPlayersTab, { props: { rows: ROWS } });
        const marked = [...container.querySelectorAll('tbody tr.selected')];
        expect(marked).toHaveLength(1);
        expect(marked[0].textContent).toContain('Bob');
        expect(container.querySelectorAll('tbody tr')).toHaveLength(2);
    });

    test("a row's counts open that player's matches and decisions, not the dashboard", async () => {
        patchStatsFilter({ dateFrom: '2025-01-01', decisionType: 1 });
        const onSelectPlayer = vi.fn();
        const { container } = render(StatsPlayersTab, { props: { rows: ROWS, onSelectPlayer } });
        const links = /** @type {HTMLElement[]} */ ([...container.querySelectorAll('tbody tr')[0].querySelectorAll('[data-testid="count-link"]')]);
        expect(links.map((l) => l.textContent)).toEqual(['3', '2–1', '120']);

        await fireEvent.click(links[0]);
        expect(StatsMatchIDs).toHaveBeenCalledWith(expect.objectContaining({ playerName: 'Alice', dateFrom: '2025-01-01', decisionType: -1 }));
        await waitFor(() => expect(openMatchListInPanel).toHaveBeenCalledWith([5, 6]));

        await fireEvent.click(links[2]);
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(expect.objectContaining({ playerName: 'Alice', decisionType: -1 }), { Kind: 'all', OnlyWithError: false });
        expect(onSelectPlayer).not.toHaveBeenCalled();
        expect(get(statsFilterStore).playerName).toBe('');
    });
});
