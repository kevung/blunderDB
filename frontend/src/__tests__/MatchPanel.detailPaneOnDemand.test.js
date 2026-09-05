/**
 * MatchPanel.detailPaneOnDemand.test.js
 *
 * The detail pane exists only while a match is selected. #201 (D.1) had
 * reserved its width instead — always mounted, a "select a match" hint
 * standing in — so that the first click could not move the clicked row out
 * from under the cursor before the second click of a double-click. That cure
 * cost 55% of the panel to display one sentence and left the match list too
 * narrow to read, so the reservation is gone: the list spans the full width
 * until a selection gives the pane something to show.
 *
 * What this test pins is the pair: no pane and a full-width list before any
 * selection, pane and `has-detail` list after one, and back again when the
 * selection is dropped.
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

describe('MatchPanel — the detail pane appears only for a selected match', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
    });
    afterEach(cleanup);

    test('no pane and a full-width list until a match is selected', async () => {
        const { container } = render(MatchPanel);
        await settle();

        const list = container.querySelector('.match-list-pane');
        const detail = () => container.querySelector('.detail-pane');
        expect(list).not.toBeNull();
        expect(detail(), 'nothing is selected, so there is no pane to show').toBeNull();
        expect(list.classList.contains('has-detail'), 'the list spans the full width').toBe(false);

        const cell = () => [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes('Alice'));
        expect(cell()).toBeTruthy();
        await fireEvent.click(cell());
        await vi.waitFor(() => expect(container.querySelector('tbody tr.selected')).not.toBeNull());

        expect(detail(), 'the selection opens the pane').not.toBeNull();
        expect(detail().querySelector('.detail-header')).not.toBeNull();
        expect(detail().textContent).toContain('Alice');
        expect(list.classList.contains('has-detail'), 'the list makes room for it').toBe(true);

        // Dropping the selection closes the pane and gives the width back.
        await fireEvent.click(cell());
        await vi.waitFor(() => expect(container.querySelector('tbody tr.selected')).toBeNull());
        expect(detail()).toBeNull();
        expect(list.classList.contains('has-detail')).toBe(false);
    });
});
