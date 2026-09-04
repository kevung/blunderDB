/**
 * TournamentPanel.test.js
 *
 * TournamentPanel.svelte (916 l.) had no test file at all (D.13, #214). It
 * owns the tournament list and the detail view of one tournament's matches,
 * plus inline create/rename/delete and match reordering. This covers the
 * load-on-open effect (an `onChange` effect, D.10: it only fires when the
 * panel's visibility actually flips), both views, create/select/delete, and
 * the Escape/j-k keyboard shortcuts. Every Wails binding is mocked; the real
 * Svelte stores drive the component.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllTournaments: vi.fn().mockResolvedValue([]),
    CreateTournament: vi.fn().mockResolvedValue(undefined),
    DeleteTournament: vi.fn().mockResolvedValue(undefined),
    UpdateTournament: vi.fn().mockResolvedValue(undefined),
    GetTournamentMatches: vi.fn().mockResolvedValue([]),
    RemoveMatchFromTournament: vi.fn().mockResolvedValue(undefined),
    GetAllMatches: vi.fn().mockResolvedValue([]),
    AddMatchToTournament: vi.fn().mockResolvedValue(undefined),
    GetMatchMovePositions: vi.fn().mockResolvedValue([]),
    LoadAnalysis: vi.fn().mockResolvedValue(null),
    SwapMatchPlayers: vi.fn().mockResolvedValue(undefined),
    SaveLastVisitedPosition: vi.fn().mockResolvedValue(undefined),
    UpdateMatchComment: vi.fn().mockResolvedValue(undefined),
    UpdateTournamentComment: vi.fn().mockResolvedValue(undefined),
    ReorderTournamentMatches: vi.fn().mockResolvedValue(undefined)
}));

import { GetAllTournaments, CreateTournament, DeleteTournament, UpdateTournament, GetTournamentMatches, GetAllMatches } from '../../wailsjs/go/database/Database.js';

import TournamentPanel from '../components/TournamentPanel.svelte';
import { openPanels, PANEL, statusBarTextStore } from '../stores/uiStore.js';
import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore } from '../stores/tournamentStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { matchContextStore } from '../stores/positionStore.js';

// ── Helpers ───────────────────────────────────────────────────────────────────

function resetStores() {
    tournamentsStore.set([]);
    selectedTournamentStore.set(null);
    tournamentMatchesStore.set([]);
    openPanels.set(new Set());
    statusBarTextStore.set('');
    databasePathStore.set('/fake/db.sqlite');
    matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
}

const SAMPLE_TOURNAMENTS = [
    { id: 1, name: 'Blunder Cup', matchCount: 2, date: '2026-01-01', location: 'Paris', pr: 4.5, mwc_loss: 0.02 },
    { id: 2, name: 'Amsterdam Open', matchCount: 0, date: '2026-02-01', location: 'Amsterdam', pr: 0, mwc_loss: 0 }
];

beforeEach(() => {
    vi.clearAllMocks();
    resetStores();
    GetAllTournaments.mockResolvedValue(SAMPLE_TOURNAMENTS);
    vi.spyOn(window, 'confirm').mockReturnValue(true);
});

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

/** Render with the panel already open, which is what triggers the load effect. */
function renderOpen() {
    openPanels.set(new Set([PANEL.TOURNAMENT]));
    return render(TournamentPanel, { props: {} });
}

// ── List view ─────────────────────────────────────────────────────────────────

describe('TournamentPanel — list view', () => {
    test('opening the panel loads the tournament list', async () => {
        renderOpen();

        expect(await screen.findByText('Blunder Cup')).toBeTruthy();
        expect(screen.getByText('Amsterdam Open')).toBeTruthy();
        expect(GetAllTournaments).toHaveBeenCalled();
    });

    test('never opened: no load, list stays empty', async () => {
        render(TournamentPanel, { props: {} }); // openPanels starts empty — no visibility change
        await tick();

        expect(GetAllTournaments).not.toHaveBeenCalled();
        expect(screen.queryByText('Blunder Cup')).toBeNull();
    });

    test('empty tournament list shows the empty-state message', async () => {
        GetAllTournaments.mockResolvedValue([]);
        renderOpen();
        await vi.waitFor(() => expect(GetAllTournaments).toHaveBeenCalled());

        expect(screen.queryByText('Blunder Cup')).toBeNull();
    });

    test('creating a tournament via the inline row on Enter', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        const nameInput = screen.getByPlaceholderText(/new tournament/i);
        await fireEvent.input(nameInput, { target: { value: 'Winter Slam' } });
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        await vi.waitFor(() => expect(CreateTournament).toHaveBeenCalled());
        expect(CreateTournament).toHaveBeenCalledWith('Winter Slam', '', '');
    });

    test('a blank name does not create a tournament', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        const nameInput = screen.getByPlaceholderText(/new tournament/i);
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        expect(CreateTournament).not.toHaveBeenCalled();
    });

    test('clicking a tournament row opens its matches (detail view)', async () => {
        GetTournamentMatches.mockResolvedValue([{ id: 501, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, comment: '' }]);
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');

        await fireEvent.click(row);

        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));
        expect(GetAllMatches).toHaveBeenCalled();
        expect(GetTournamentMatches).toHaveBeenCalledWith(1);
        expect(await screen.findByText('Alice')).toBeTruthy();
    });

    test('clicking a tournament, then the back button, returns to the list with the selection cleared', async () => {
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');

        await fireEvent.click(row);
        await vi.waitFor(() => expect(get(selectedTournamentStore)).not.toBeNull());

        await fireEvent.click(screen.getByTitle(/back to tournaments/i));

        expect(get(selectedTournamentStore)).toBeNull();
        expect(await screen.findByText('Blunder Cup')).toBeTruthy(); // list view again
    });

    test('deleting a tournament asks for confirmation then reloads the list', async () => {
        renderOpen();
        const row = (await screen.findByText('Amsterdam Open')).closest('tr');
        const deleteBtn = within(row).getByTitle(/delete/i);
        GetAllTournaments.mockResolvedValue([SAMPLE_TOURNAMENTS[0]]);

        await fireEvent.click(deleteBtn);

        await vi.waitFor(() => expect(DeleteTournament).toHaveBeenCalledWith(2));
        expect(window.confirm).toHaveBeenCalled();
        await vi.waitFor(() => expect(screen.queryByText('Amsterdam Open')).toBeNull());
    });

    test('declining the confirmation leaves the tournament in place', async () => {
        window.confirm.mockReturnValue(false);
        renderOpen();
        const row = (await screen.findByText('Amsterdam Open')).closest('tr');
        const deleteBtn = within(row).getByTitle(/delete/i);

        await fireEvent.click(deleteBtn);

        expect(DeleteTournament).not.toHaveBeenCalled();
        expect(screen.getByText('Amsterdam Open')).toBeTruthy();
    });

    test('renaming a tournament through the inline editor', async () => {
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');
        const editBtn = within(row).getByTitle(/^edit$/i);
        await fireEvent.click(editBtn);

        const nameInput = within(row).getByDisplayValue('Blunder Cup');
        await fireEvent.input(nameInput, { target: { value: 'Blunder Cup (2026)' } });
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        await vi.waitFor(() => expect(UpdateTournament).toHaveBeenCalled());
        expect(UpdateTournament).toHaveBeenCalledWith(1, 'Blunder Cup (2026)', '2026-01-01', 'Paris');
    });

    test('sorting by name toggles ascending/descending', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        const sortBtn = screen.getByRole('button', { name: /name/i });
        await fireEvent.click(sortBtn);
        await tick();

        const namesAsc = [...document.querySelectorAll('tbody tr td:first-child')].map((td) => td.textContent);
        expect(namesAsc).toEqual(['Amsterdam Open', 'Blunder Cup']);

        await fireEvent.click(sortBtn);
        await tick();
        const namesDesc = [...document.querySelectorAll('tbody tr td:first-child')].map((td) => td.textContent);
        expect(namesDesc).toEqual(['Blunder Cup', 'Amsterdam Open']);
    });
});

// ── Keyboard shortcuts ────────────────────────────────────────────────────────

describe('TournamentPanel — keyboard shortcuts', () => {
    test('Escape from the detail view returns to the list', async () => {
        GetTournamentMatches.mockResolvedValue([]);
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');
        await fireEvent.click(row);
        await vi.waitFor(() => expect(get(selectedTournamentStore)).not.toBeNull());

        await fireEvent.keyDown(document, { key: 'Escape' });

        expect(get(selectedTournamentStore)).toBeNull();
    });

    test('j/k walk the tournament list and select the next/previous row', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        await fireEvent.keyDown(document, { key: 'j' });
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));

        // Deselect (toggle) before walking again from a clean slate.
        selectedTournamentStore.set(null);
        tournamentMatchesStore.set([]);
        await tick();

        await fireEvent.keyDown(document, { key: 'j' });
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));
        await fireEvent.keyDown(document, { key: 'j' });
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 2 }));
    });
});
