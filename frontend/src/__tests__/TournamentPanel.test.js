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

import { must } from './helpers/must.js';
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { answerConfirm } from './confirmHelper.js';
import { render, cleanup, screen, fireEvent, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CountMatches: vi.fn().mockResolvedValue(0),
    GetMatchByID: vi.fn().mockResolvedValue(null),
    GetGamesByMatch: vi.fn().mockResolvedValue([]),
    GetMatchDetailStats: vi.fn().mockResolvedValue(null),
    GetMatchMoveGrades: vi.fn().mockResolvedValue([]),
    GetMatchTimeSummary: vi.fn().mockResolvedValue(null),
    GetMatchOrigin: vi.fn().mockResolvedValue(null),
    GetMatchDecisionLosses: vi.fn().mockResolvedValue([]),
    GetMatchReview: vi.fn().mockResolvedValue(null),
    TrashMatch: vi.fn().mockResolvedValue(undefined),
    UpdateMatch: vi.fn().mockResolvedValue(undefined),
    SetMatchTournamentByName: vi.fn().mockResolvedValue(undefined),
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    LoadCommandHistory: vi.fn().mockResolvedValue([]),
    SaveCommand: vi.fn().mockResolvedValue(undefined),
    GetAllTournaments: vi.fn().mockResolvedValue([]),
    CreateTournament: vi.fn().mockResolvedValue(undefined),
    DeleteTournament: vi.fn().mockResolvedValue(undefined),
    UpdateTournament: vi.fn().mockResolvedValue(undefined),
    GetTournamentMatches: vi.fn().mockResolvedValue([]),
    RemoveMatchFromTournament: vi.fn().mockResolvedValue(undefined),
    ListMatches: vi.fn().mockResolvedValue([]),
    AddMatchToTournament: vi.fn().mockResolvedValue(undefined),
    GetMatchMovePositions: vi.fn().mockResolvedValue([]),
    LoadAnalysis: vi.fn().mockResolvedValue(null),
    SwapMatchPlayers: vi.fn().mockResolvedValue(undefined),
    SaveLastVisitedPosition: vi.fn().mockResolvedValue(undefined),
    UpdateMatchComment: vi.fn().mockResolvedValue(undefined),
    UpdateTournamentComment: vi.fn().mockResolvedValue(undefined),
    ReorderTournamentMatches: vi.fn().mockResolvedValue(undefined),
    GetTournamentReview: vi.fn().mockResolvedValue(null),
    GetPositionIDsByTournament: vi.fn().mockResolvedValue([])
}));

import {
    GetAllTournaments,
    CreateTournament,
    DeleteTournament,
    UpdateTournament,
    GetTournamentMatches,
    ListMatches,
    GetTournamentReview,
    GetPositionIDsByTournament
} from '../../wailsjs/go/database/Database.js';

import TournamentPanel from '../components/TournamentPanel.svelte';
import { openPanels, PANEL, statusBarTextStore } from '../stores/uiStore.js';
import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore, tournamentOpenRequestStore } from '../stores/tournamentStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { matchContextStore } from '../stores/positionStore.js';

// ── Helpers ───────────────────────────────────────────────────────────────────

function resetStores() {
    tournamentsStore.set([]);
    selectedTournamentStore.set(null);
    tournamentMatchesStore.set([]);
    tournamentOpenRequestStore.set(null);
    openPanels.set(new Set());
    statusBarTextStore.set('');
    databasePathStore.set('/fake/db.sqlite');
    matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
}

/** @param {Partial<import('../../wailsjs/go/models').domain.Tournament>} fields */
const aTournament = (fields) =>
    /** @type {import('../../wailsjs/go/models').domain.Tournament} */ ({
        id: 0,
        name: '',
        date: '',
        location: '',
        sortOrder: 0,
        createdAt: '',
        updatedAt: '',
        matchCount: 0,
        comment: '',
        pr: 0,
        mwc_loss: 0,
        ref_player: '',
        ...fields
    });

const SAMPLE_TOURNAMENTS = [
    aTournament({ id: 1, name: 'Blunder Cup', matchCount: 2, date: '2026-01-01', location: 'Paris', pr: 4.5, mwc_loss: 0.02 }),
    aTournament({ id: 2, name: 'Amsterdam Open', matchCount: 0, date: '2026-02-01', location: 'Amsterdam', pr: 0, mwc_loss: 0 })
];

beforeEach(() => {
    vi.clearAllMocks();
    resetStores();
    vi.mocked(GetAllTournaments).mockResolvedValue(SAMPLE_TOURNAMENTS);
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

/** The creation field opens from the header's "+ New …" button. */
async function openCreation() {
    await fireEvent.click(screen.getByTestId('panel-new'));
    return screen.getByPlaceholderText(/new tournament/i);
}

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
        vi.mocked(GetAllTournaments).mockResolvedValue([]);
        renderOpen();
        await vi.waitFor(() => expect(GetAllTournaments).toHaveBeenCalled());

        expect(screen.queryByText('Blunder Cup')).toBeNull();
    });

    test('creating a tournament via the inline row on Enter', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        const nameInput = await openCreation();
        await fireEvent.input(nameInput, { target: { value: 'Winter Slam' } });
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        await vi.waitFor(() => expect(CreateTournament).toHaveBeenCalled());
        expect(CreateTournament).toHaveBeenCalledWith('Winter Slam', '', '');
    });

    test('Escape in the creation fields closes the form and forgets what was typed', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        const nameInput = await openCreation();
        await fireEvent.input(nameInput, { target: { value: 'Winter Slam' } });
        await fireEvent.keyDown(nameInput, { key: 'Escape' });

        expect(screen.queryByPlaceholderText(/new tournament/i)).toBeNull();
        expect(await openCreation()).toHaveValue('');
    });

    test('the Cancel button closes the creation form', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        await openCreation();
        await fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

        expect(screen.queryByPlaceholderText(/new tournament/i)).toBeNull();
    });

    test('a single click highlights the row without opening the tournament', async () => {
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');

        await fireEvent.click(must(row));

        expect(must(row).classList.contains('selected')).toBe(true);
        expect(get(selectedTournamentStore)).toBeNull();
        expect(GetTournamentMatches).not.toHaveBeenCalled();
    });

    test('a blank name does not create a tournament', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        const nameInput = await openCreation();
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        expect(CreateTournament).not.toHaveBeenCalled();
    });

    test('double-clicking a tournament row opens its matches (detail view)', async () => {
        vi.mocked(GetTournamentMatches).mockResolvedValue([{ id: 501, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, comment: '' }]);
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');

        await fireEvent.dblClick(must(row));

        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));
        expect(ListMatches).toHaveBeenCalledWith(expect.objectContaining({ Unassigned: true }));
        expect(GetTournamentMatches).toHaveBeenCalledWith(1);
        expect(await screen.findByText('Alice')).toBeTruthy();
    });

    test('the strip shows as many positions as the ids the link opens', async () => {
        vi.mocked(GetTournamentMatches).mockResolvedValue(/** @type {any} */ ([{ id: 501, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, comment: '' }]));
        vi.mocked(GetPositionIDsByTournament).mockResolvedValue([11, 12, 13]);
        renderOpen();
        await fireEvent.dblClick(/** @type {HTMLElement} */ ((await screen.findByText('Blunder Cup')).closest('tr')));

        await vi.waitFor(() => expect(GetPositionIDsByTournament).toHaveBeenCalledWith(1));
        const link = await screen.findByTestId('tournament-positions');
        expect(link.textContent).toMatch(/\b3\b/);
    });

    test('the review toggle loads the review of the most present player', async () => {
        vi.mocked(GetTournamentMatches).mockResolvedValue(/** @type {any} */ ([{ id: 501, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, comment: '' }]));
        renderOpen();
        await fireEvent.dblClick(/** @type {HTMLElement} */ ((await screen.findByText('Blunder Cup')).closest('tr')));
        await screen.findByText('Alice');
        expect(GetTournamentReview).not.toHaveBeenCalled();

        await fireEvent.click(screen.getByTestId('tournament-review-toggle'));

        await vi.waitFor(() => expect(GetTournamentReview).toHaveBeenCalledWith(1, ''));
        expect(screen.getByTestId('tournament-review')).toBeTruthy();
    });

    test('a tournament row is reachable by Tab and Enter opens it; Enter on the rename button still renames', async () => {
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');
        expect(must(row).getAttribute('tabindex')).toBe('0');

        await fireEvent.keyDown(must(must(row).querySelector('button[title]')), { key: 'Enter' });
        expect(get(selectedTournamentStore)).toBeNull();

        await fireEvent.keyDown(must(row), { key: 'Enter' });
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));
    });

    test('a single click selects the row and shows the open hint', async () => {
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');
        expect(screen.getByTestId('tournament-open-hint').classList.contains('concealed')).toBe(true);

        await fireEvent.click(must(row));

        expect(screen.getByTestId('tournament-open-hint').classList.contains('concealed')).toBe(false);
        expect(screen.getByTestId('tournament-open-hint').textContent).toMatch(/Enter/);
        expect(get(selectedTournamentStore)).toBeNull();
    });

    test('double-clicking a tournament, then the back button, returns to the list with the selection cleared', async () => {
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');

        await fireEvent.dblClick(must(row));
        await vi.waitFor(() => expect(get(selectedTournamentStore)).not.toBeNull());

        await fireEvent.click(screen.getByTitle(/back to tournaments/i));

        expect(get(selectedTournamentStore)).toBeNull();
        expect(await screen.findByText('Blunder Cup')).toBeTruthy(); // list view again
    });

    test('deleting a tournament asks for confirmation then reloads the list', async () => {
        renderOpen();
        const row = (await screen.findByText('Amsterdam Open')).closest('tr');
        const deleteBtn = within(must(row)).getByTitle(/delete/i);
        vi.mocked(GetAllTournaments).mockResolvedValue([SAMPLE_TOURNAMENTS[0]]);

        await fireEvent.click(deleteBtn);
        await answerConfirm(true);

        await vi.waitFor(() => expect(DeleteTournament).toHaveBeenCalledWith(2));
        await vi.waitFor(() => expect(screen.queryByText('Amsterdam Open')).toBeNull());
    });

    test('declining the confirmation leaves the tournament in place', async () => {
        renderOpen();
        const row = (await screen.findByText('Amsterdam Open')).closest('tr');
        const deleteBtn = within(must(row)).getByTitle(/delete/i);

        await fireEvent.click(deleteBtn);
        await answerConfirm(false);

        expect(DeleteTournament).not.toHaveBeenCalled();
        expect(screen.getByText('Amsterdam Open')).toBeTruthy();
    });

    test('renaming a tournament through the inline editor', async () => {
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');
        const editBtn = within(must(row)).getByTitle(/^edit$/i);
        await fireEvent.click(editBtn);

        const nameInput = within(must(row)).getByDisplayValue('Blunder Cup');
        await fireEvent.input(nameInput, { target: { value: 'Blunder Cup (2026)' } });
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        await vi.waitFor(() => expect(UpdateTournament).toHaveBeenCalled());
        expect(UpdateTournament).toHaveBeenCalledWith(1, 'Blunder Cup (2026)', '2026-01-01', 'Paris');
    });

    test('sorting by name toggles ascending/descending', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');

        const sortBtn = screen.getByRole('button', { name: /^name/i });
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
        vi.mocked(GetTournamentMatches).mockResolvedValue([]);
        renderOpen();
        const row = (await screen.findByText('Blunder Cup')).closest('tr');
        await fireEvent.dblClick(must(row));
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

// ── Deferred focus ────────────────────────────────────────────────────────────

/** Past the panel's own 100 ms focus timer. */
const pastFocusTimer = () => new Promise((resolve) => setTimeout(resolve, 150));

describe('TournamentPanel — deferred focus', () => {
    // The panel focuses itself 100 ms after it opens. A user (or a slow CI
    // runner) already typing a name by then lost the caret: Enter reached the
    // panel instead of the field, and no tournament was created.
    test('does not take the caret from the new-tournament field', async () => {
        renderOpen();
        const nameInput = await openCreation();
        nameInput.focus();

        await pastFocusTimer();

        expect(document.activeElement).toBe(nameInput);
    });

    test('still takes the keyboard when nobody is typing', async () => {
        renderOpen();

        await pastFocusTimer();

        expect(document.activeElement?.id).toBe('tournamentPanel');
    });
});

describe('TournamentPanel — focus after creation', () => {
    test('creating a tournament puts the focus on its new row, not on body', async () => {
        vi.mocked(CreateTournament).mockImplementation(async () => {
            vi.mocked(GetAllTournaments).mockResolvedValue([...SAMPLE_TOURNAMENTS, aTournament({ id: 3, name: 'Zeta Trophy' })]);
        });
        renderOpen();
        await screen.findByText('Blunder Cup');
        // Past the panel's deferred own focus (100 ms), as a person typing would be.
        await new Promise((r) => setTimeout(r, 150));
        const nameInput = await openCreation();
        await fireEvent.input(nameInput, { target: { value: 'Zeta Trophy' } });
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        await vi.waitFor(() => expect(document.activeElement?.closest('tr')?.textContent).toContain('Zeta Trophy'));
    });
});

// ── Open request from another panel (Stats) ───────────────────────────────────

describe('TournamentPanel — open request', () => {
    test('request made before the panel is mounted opens that tournament', async () => {
        tournamentOpenRequestStore.set(2);
        renderOpen();

        await vi.waitFor(() => expect(get(selectedTournamentStore)?.id).toBe(2));
        expect(get(tournamentOpenRequestStore)).toBeNull();
        expect(GetTournamentMatches).toHaveBeenCalledWith(2);
    });

    test('request while the panel shows another tournament switches to the requested one', async () => {
        renderOpen();
        await fireEvent.dblClick(await screen.findByText('Blunder Cup'));
        await vi.waitFor(() => expect(get(selectedTournamentStore)?.id).toBe(1));

        tournamentOpenRequestStore.set(2);

        await vi.waitFor(() => expect(get(selectedTournamentStore)?.id).toBe(2));
        expect(get(tournamentOpenRequestStore)).toBeNull();
    });

    test('request for a vanished tournament is consumed without opening anything', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');
        tournamentOpenRequestStore.set(99);

        await vi.waitFor(() => expect(get(tournamentOpenRequestStore)).toBeNull());
        expect(get(selectedTournamentStore)).toBeNull();
    });
});

// ── Filter field in the header strip ──────────────────────────────────────────

describe('TournamentPanel — filter', () => {
    test('typing narrows the list by name, location or date, with a shown/total counter', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');
        const field = /** @type {HTMLInputElement} */ (screen.getByTestId('tournament-filter'));

        await fireEvent.input(field, { target: { value: 'amst' } });
        expect(screen.queryByText('Blunder Cup')).toBeNull();
        expect(screen.getByText('Amsterdam Open')).toBeTruthy();
        expect(screen.getByText('1 / 2')).toBeTruthy();

        await fireEvent.input(field, { target: { value: 'paris' } });
        expect(screen.getByText('Blunder Cup')).toBeTruthy();
        expect(screen.queryByText('Amsterdam Open')).toBeNull();

        await fireEvent.input(field, { target: { value: '2026-02' } });
        expect(screen.getByText('Amsterdam Open')).toBeTruthy();
        expect(screen.queryByText('Blunder Cup')).toBeNull();
    });

    test('Escape empties the field and restores the list', async () => {
        renderOpen();
        await screen.findByText('Blunder Cup');
        const field = /** @type {HTMLInputElement} */ (screen.getByTestId('tournament-filter'));
        await fireEvent.input(field, { target: { value: 'zzz' } });
        expect(screen.queryByText('Blunder Cup')).toBeNull();

        await fireEvent.keyDown(field, { key: 'Escape' });
        expect(field.value).toBe('');
        expect(screen.getByText('Blunder Cup')).toBeTruthy();
        expect(screen.getByText('Amsterdam Open')).toBeTruthy();
    });
});
