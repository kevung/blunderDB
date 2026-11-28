/**
 * MatchSelection.component.test.js
 *
 * Ctrl/Shift+click selects several matches of the list; the print and paper archive dialogs take
 * the selection, and refuse beyond 100 with a visible message and a disabled button.
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

const { MATCHES, printMatchSheet, exportPaperArchive } = vi.hoisted(() => ({
    MATCHES: [1, 2, 3, 4].map((id) => ({ id, player1_name: `A${id}`, player2_name: `B${id}`, match_length: 7, winner: 0 })),
    printMatchSheet: vi.fn(() => Promise.resolve('/x.html')),
    exportPaperArchive: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() => Promise.resolve(MATCHES)),
    CountMatches: vi.fn(() => Promise.resolve(4)),
    GetMatchByID: vi.fn(() => Promise.resolve(null)),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));
vi.mock('../services/matchSheetService.js', async (orig) => ({ ...(await orig()), printMatchSheet }));
vi.mock('../services/paperArchiveService.js', async (orig) => ({ ...(await orig()), exportPaperArchive }));

import { databasePathStore } from '../stores/databaseStore.js';
import MatchPanel from '../components/MatchPanel.svelte';
import MatchPrintModal from '../components/MatchPrintModal.svelte';
import PaperArchiveModal from '../components/PaperArchiveModal.svelte';

afterEach(() => {
    cleanup();
    databasePathStore.set('');
    vi.clearAllMocks();
});

async function rows() {
    databasePathStore.set('/tmp/x.db');
    const view = render(MatchPanel);
    await vi.waitFor(() => expect(view.container.querySelectorAll('tbody tr').length).toBeGreaterThanOrEqual(4));
    return { ...view, trs: () => [...view.container.querySelectorAll('tbody tr')] };
}

describe('match list selection', () => {
    test('Ctrl+click adds rows, Shift+click a range, a plain click clears', async () => {
        const { container, trs } = await rows();
        await fireEvent.click(trs()[0], { ctrlKey: true });
        await fireEvent.click(trs()[2], { ctrlKey: true });
        expect(container.querySelectorAll('tbody tr.selected').length).toBe(2);
        expect(container.querySelector('[data-testid="match-selection-chip"]')?.textContent).toContain('2');
        await fireEvent.click(trs()[3], { shiftKey: true });
        expect(container.querySelectorAll('tbody tr.selected').length).toBe(2);
        await fireEvent.click(trs()[1], { ctrlKey: true });
        await fireEvent.click(trs()[0], { ctrlKey: true, shiftKey: true });
        expect(container.querySelectorAll('tbody tr.selected').length).toBe(4);
        await fireEvent.click(trs()[0]);
        expect(container.querySelector('[data-testid="match-selection-chip"]')).toBeNull();
    });
});

describe('print dialog on a selection', () => {
    test('prints the selected ids in one call', async () => {
        const { getByTestId } = render(MatchPrintModal, { open: true, matchIds: [3, 5, 8] });
        expect(getByTestId('match-print-selection').textContent).toContain('3');
        await fireEvent.click(getByTestId('match-print-go'));
        await vi.waitFor(() => expect(printMatchSheet).toHaveBeenCalledTimes(1));
        expect(/** @type {any[][]} */ (printMatchSheet.mock.calls)[0][0]).toEqual([3, 5, 8]);
    });
    test('refuses beyond 100 with a message and a disabled button', () => {
        const { getByTestId } = render(MatchPrintModal, { open: true, matchIds: Array.from({ length: 101 }, (_, i) => i + 1) });
        expect(getByTestId('match-print-too-many').textContent).toContain('101');
        expect(/** @type {HTMLButtonElement} */ (getByTestId('match-print-go')).disabled).toBe(true);
    });
});

describe('paper archive dialog on a selection', () => {
    test('exports the selected ids', async () => {
        const { getByTestId } = render(PaperArchiveModal, { open: true, matchIds: [2, 4] });
        await fireEvent.click(getByTestId('paper-archive-export'));
        await vi.waitFor(() => expect(exportPaperArchive).toHaveBeenCalledTimes(1));
        expect(/** @type {any[][]} */ (exportPaperArchive.mock.calls)[0][0]).toEqual([2, 4]);
    });
    test('refuses beyond 100', () => {
        const { getByTestId } = render(PaperArchiveModal, { open: true, matchIds: Array.from({ length: 150 }, (_, i) => i + 1) });
        expect(getByTestId('paper-archive-too-many')).toBeTruthy();
        expect(/** @type {HTMLButtonElement} */ (getByTestId('paper-archive-export')).disabled).toBe(true);
    });
});
