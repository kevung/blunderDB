/**
 * MatchPanel.detailPaneReserved.test.js
 *
 * #201 (D.1): the detail pane used to appear on the first click on a match,
 * narrowing the list from 100% to 45% (`has-detail`) — the clicked row moved
 * under the cursor before the second click of a double-click. The pane's
 * width is now reserved: it is always mounted (a hint when nothing is
 * selected) and the list's class list — hence its CSS width — does not
 * depend on the selection.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllMatches: vi.fn(() => Promise.resolve([{ id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2 }])),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    DeleteMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    GetGamesByMatch: vi.fn(() => Promise.resolve([])),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import { openPanels, PANEL } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { GetAllMatches } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';

/**
 * The list is loaded twice at mount (onMount, then the visibility effect) and
 * the second load clears the selection when it resolves: wait for both before
 * touching a row.
 */
async function settle() {
    await vi.waitFor(() => expect(GetAllMatches).toHaveBeenCalledTimes(2));
    await new Promise((resolve) => setTimeout(resolve, 0));
    for (let i = 0; i < 4; i++) await tick();
}

describe('MatchPanel — the detail pane keeps its width whether or not a match is selected', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
    });
    afterEach(cleanup);

    test('a hint stands in for the detail before any selection, and the list does not change class on selection', async () => {
        const { container } = render(MatchPanel);
        await settle();

        const list = container.querySelector('.match-list-pane');
        const detail = () => container.querySelector('.detail-pane');
        expect(list).not.toBeNull();
        expect(detail(), 'the pane is mounted before any selection').not.toBeNull();
        expect(detail().textContent).toContain('Select a match');
        expect(detail().querySelector('.detail-header')).toBeNull();
        const classesBefore = list.className;

        const cell = () => [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes('Alice'));
        expect(cell()).toBeTruthy();
        await fireEvent.click(cell());
        await vi.waitFor(() => expect(container.querySelector('tbody tr.selected')).not.toBeNull());

        expect(list.className, 'the list keeps its class list, hence its width').toBe(classesBefore);
        expect(detail().querySelector('.detail-header'), 'the selected match fills the pane').not.toBeNull();
        expect(detail().textContent).toContain('Alice');

        // Deselecting brings the hint back, still without touching the list.
        await fireEvent.click(cell());
        await vi.waitFor(() => expect(container.querySelector('tbody tr.selected')).toBeNull());
        expect(list.className).toBe(classesBefore);
        expect(detail().textContent).toContain('Select a match');
        expect(detail().querySelector('.detail-header')).toBeNull();
    });
});
