/**
 * TournamentPanel.selection.test.js
 *
 * In a tournament, Ctrl/Shift+click selects several matches, and « Imprimer le tournoi… » opens the
 * print dialog on every match of the tournament.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';

const M = (/** @type {number} */ id, /** @type {string} */ a, /** @type {string} */ b) => ({
    id,
    player1_name: a,
    player2_name: b,
    match_length: 7,
    match_date: '2026-01-15',
    game_count: 1,
    pr: 0,
    mwc_loss: 0
});
const MATCHES = [M(7, 'Alice', 'Bob'), M(8, 'Carol', 'Dave'), M(9, 'Erin', 'Finn')];

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListDirections: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([{ id: 1, name: 'Blunder Cup', date: '', location: '', matchCount: 3, comment: '', pr: 0, mwc_loss: 0 }])),
    GetTournamentMatches: vi.fn(() => Promise.resolve(MATCHES)),
    GetPositionIDsByTournament: vi.fn(() => Promise.resolve([])),
    GetTournamentReview: vi.fn(() => Promise.resolve(null)),
    ListMatches: vi.fn(() => Promise.resolve([])),
    CountMatches: vi.fn(() => Promise.resolve(0)),
    GetMatchByID: vi.fn(() => Promise.resolve(null))
}));

import TournamentPanel from '../components/TournamentPanel.svelte';
import { activeTabStore } from '../stores/uiStore.js';
import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore, tournamentOpenRequestStore } from '../stores/tournamentStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

beforeEach(() => {
    tournamentsStore.set([]);
    selectedTournamentStore.set(null);
    tournamentMatchesStore.set([]);
    databasePathStore.set('/fake/db.sqlite');
    activeTabStore.set('tournaments');
    tournamentOpenRequestStore.set(1);
});
afterEach(cleanup);

describe('TournamentPanel selection', () => {
    test('Ctrl+click and Shift+click select several matches', async () => {
        const { container } = render(TournamentPanel, { props: {} });
        const first = await screen.findByText('Bob');
        await fireEvent.click(/** @type {Element} */ (first.closest('tr')), { ctrlKey: true });
        await fireEvent.click(/** @type {Element} */ (screen.getByText('Finn').closest('tr')), { shiftKey: true });
        expect(container.querySelectorAll('tbody tr.selected').length).toBe(3);
        expect(screen.getByTestId('tournament-selection-chip').textContent).toContain('3');
    });

    test('Print the tournament opens the dialog on every match', async () => {
        render(TournamentPanel, { props: {} });
        await screen.findByText('Bob');
        await fireEvent.click(screen.getByTestId('tournament-print'));
        expect((await screen.findByTestId('match-print-selection')).textContent).toContain('3');
    });
});
