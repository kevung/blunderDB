/**
 * A Corpus row opens what it sums up: a match or a face-à-face opens matches in the Match panel,
 * a player's ranking line opens his matches, a window opens its decisions' positions. A count
 * inside a row opens the other half, and only that.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn(() => Promise.resolve([])),
    HeadToHead: vi.fn(() =>
        Promise.resolve({
            player_a: 'Alice',
            player_b: 'Bob',
            wins_a: 1,
            wins_b: 1,
            pr_a: 4.5,
            pr_b: 6.25,
            matches: [
                { id: 7, date: '2025-01-02', match_length: 7, outcome: 1, pr_a: 4, pr_b: 6 },
                { id: 8, date: '2025-02-02', match_length: 5, outcome: -1, pr_a: 5, pr_b: 6 }
            ]
        })
    ),
    PRByWindow: vi.fn(() => Promise.resolve([{ from: '2024-12', to: '2025-02', num_matches: 3, num_decisions: 420, pr: 5.12 }])),
    PlayerRanking: vi.fn(() => Promise.resolve([{ rank: 1, name: 'Alice', matches: 3, wins: 2, losses: 1, decisions: 600, pr: 4.4, pr_checker: 4.1, pr_cube: 6.2 }])),
    StatsMatchIDs: vi.fn(() => Promise.resolve([3, 4, 5]))
}));

vi.mock('../services/positionLoader.js', () => ({
    loadPositionsFromStatsSelection: vi.fn(() => Promise.resolve()),
    openMatchListInPanel: vi.fn(() => Promise.resolve())
}));

import { StatsMatchIDs } from '../../wailsjs/go/database/Database.js';
import { loadPositionsFromStatsSelection, openMatchListInPanel } from '../services/positionLoader.js';
import StatsCorpusTab from '../components/stats/StatsCorpusTab.svelte';

const filter = { playerName: '', tournamentIDs: [], dateFrom: '', dateTo: '', decisionType: -1, matchLength: [] };
const ALL = { Kind: 'all', OnlyWithError: false };

/** Render the tab and compute one section (1 h2h, 2 windows, 3 ranking). */
async function computed(/** @type {number} */ section, props = { filter }) {
    const r = render(StatsCorpusTab, { props });
    if (section === 1) {
        await fireEvent.input(r.getByLabelText('Player A'), { target: { value: 'Alice' } });
        await fireEvent.input(r.getByLabelText('Player B'), { target: { value: 'Bob' } });
    }
    await fireEvent.click(/** @type {HTMLElement} */ (r.container.querySelector(`section:nth-of-type(${section}) .corpus-controls button`)));
    return r;
}

beforeEach(() => vi.clearAllMocks());
afterEach(() => cleanup());

describe('Corpus › head-to-head', () => {
    test('the summary opens the matches of the face-à-face', async () => {
        const { findByTestId } = await computed(1);
        await fireEvent.click(await findByTestId('corpus-h2h-summary'));
        expect(openMatchListInPanel).toHaveBeenCalledWith([7, 8]);
    });

    test('a match row opens that match, by click or by Enter', async () => {
        const { findAllByTestId } = await computed(1);
        const rows = await findAllByTestId('corpus-h2h-row');
        await fireEvent.click(rows[1]);
        expect(openMatchListInPanel).toHaveBeenLastCalledWith([8]);
        await fireEvent.keyDown(rows[0], { key: 'Enter' });
        expect(openMatchListInPanel).toHaveBeenLastCalledWith([7]);
        expect(loadPositionsFromStatsSelection).not.toHaveBeenCalled();
    });
});

describe('Corpus › windows', () => {
    test('a window row opens the positions of its months, inside the filter', async () => {
        const { findByTestId } = await computed(2);
        await fireEvent.click(await findByTestId('corpus-window-row'));
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith({ ...filter, dateFrom: '2024-12-01', dateTo: '2025-02-28' }, ALL);
        expect(openMatchListInPanel).not.toHaveBeenCalled();
    });

    test('a narrower filter keeps its own bounds', async () => {
        const narrow = { ...filter, dateFrom: '2025-01-15', dateTo: '2025-02-10' };
        const { findByTestId } = await computed(2, { filter: narrow });
        await fireEvent.click(await findByTestId('corpus-window-row'));
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith({ ...narrow, dateFrom: '2025-01-15', dateTo: '2025-02-10' }, ALL);
    });

    test('its matches count opens the window matches, not its positions', async () => {
        const { findByTestId } = await computed(2);
        const row = await findByTestId('corpus-window-row');
        await fireEvent.click(/** @type {HTMLElement} */ (row.querySelector('.count-link')));
        expect(StatsMatchIDs).toHaveBeenCalledWith({ ...filter, dateFrom: '2024-12-01', dateTo: '2025-02-28' });
        await waitFor(() => expect(openMatchListInPanel).toHaveBeenCalledWith([3, 4, 5]));
        expect(loadPositionsFromStatsSelection).not.toHaveBeenCalled();
    });
});

describe('Corpus › ranking', () => {
    test('a player row opens his matches under the filter', async () => {
        const { findByTestId } = await computed(3);
        await fireEvent.click(await findByTestId('corpus-ranking-row'));
        expect(StatsMatchIDs).toHaveBeenCalledWith({ ...filter, playerName: 'Alice' });
        await waitFor(() => expect(openMatchListInPanel).toHaveBeenCalledWith([3, 4, 5]));
    });

    test('his decisions count opens his positions, not his matches', async () => {
        const { findByTestId } = await computed(3);
        const row = await findByTestId('corpus-ranking-row');
        await fireEvent.click(/** @type {HTMLElement} */ (row.querySelector('.count-link')));
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith({ ...filter, playerName: 'Alice' }, ALL);
        expect(StatsMatchIDs).not.toHaveBeenCalled();
    });
});
