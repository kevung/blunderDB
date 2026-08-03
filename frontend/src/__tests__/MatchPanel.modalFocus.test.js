/**
 * MatchPanel.modalFocus.test.js
 *
 * Regression: text fields in the export dialog could not be clicked into. The caret appeared
 * and vanished at once, while Tab still reached the field — which is what made the symptom so
 * hard to place.
 *
 * The cause was outside the dialog entirely. MatchPanel installs a click listener on
 * `document` for the whole life of the panel, to drop focus when the user clicks away from
 * it. Nothing scoped it to the panel's own context, so it also fired for clicks inside a
 * modal: the field took focus on mousedown, and the click that followed blurred it.
 *
 * SearchHistoryPanel carries the same listener and the same guard.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllMatches: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    DeleteMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    GetGamesByMatch: vi.fn(() => Promise.resolve([])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import MatchPanel from '../components/MatchPanel.svelte';
import { openPanels, PANEL, activeModal, MODAL } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

let field;

beforeEach(() => {
    databasePathStore.set('/tmp/test.db');
    activeModal.set(null);
    openPanels.set(new Set([PANEL.MATCHES]));
    // Stands in for a text field of a dialog: outside the match panel, like every modal.
    field = document.createElement('input');
    field.id = 'a-field-in-a-dialog';
    document.body.appendChild(field);
});

afterEach(() => {
    cleanup();
    field.remove();
    activeModal.set(null);
    openPanels.set(new Set());
});

describe('MatchPanel click-away handling', () => {
    test('a click inside an open modal does not steal the focus', async () => {
        render(MatchPanel);
        await tick();

        activeModal.set(MODAL.EXPORT_DATABASE);
        await tick();

        field.focus();
        expect(document.activeElement).toBe(field);

        await fireEvent.click(field);
        await tick();

        expect(document.activeElement).toBe(field);
    });

    test('with no modal open, clicking away from the panel still drops focus', async () => {
        render(MatchPanel);
        await tick();

        field.focus();
        expect(document.activeElement).toBe(field);

        await fireEvent.click(field);
        await tick();

        // The panel's own behaviour is unchanged: this is what the listener is for.
        expect(document.activeElement).not.toBe(field);
    });
});
