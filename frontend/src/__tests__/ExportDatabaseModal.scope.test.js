/**
 * ExportDatabaseModal.scope.test.js
 *
 * The export's grid: a list may be exported as the whole library, which ticks every family;
 * a list's collections may bring their positions; a deck's review history follows the decks.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListLessons: vi.fn(async () => [])
}));

import ExportDatabaseModal from '../components/ExportDatabaseModal.svelte';

afterEach(cleanup);

const base = () => ({
    wholeLibrary: false,
    includeMatches: true,
    matchIDs: [],
    includeTournaments: false,
    includeTournamentIDs: [],
    includeCollections: false,
    collectionIDs: [],
    includeLessons: false,
    lessonIDs: [],
    includeAnkiDecks: false,
    deckIDs: [],
    includeAnkiHistory: false
});

/** @param {HTMLElement} c @param {string} sel */
const box = (c, sel) => /** @type {HTMLInputElement} */ (must(c.querySelector(sel)));

describe('the scope', () => {
    test('is offered to a list only, and the library ticks every family', async () => {
        const onScope = vi.fn();
        const decks = [{ id: 5, name: 'Openings', cardCount: 12 }];
        const { container } = render(ExportDatabaseModal, { visible: true, exportOptions: base(), decks, chooseScope: true, onScope, onCancel: () => {}, onExport: () => {} });
        expect(box(container, '#export-scope-list').checked).toBe(true);
        await fireEvent.click(box(container, '#export-scope-library'));
        expect(onScope).toHaveBeenCalledWith(true);
        for (const id of ['#export-decks', '#export-anki-history', '#export-filter-library']) {
            expect(box(container, id).checked).toBe(true);
        }
        // The library holds every position: no collection needs to bring its own.
        expect(container.querySelector('#export-collection-positions')).toBeNull();
        cleanup();
        const library = render(ExportDatabaseModal, { visible: true, exportOptions: base(), onCancel: () => {}, onExport: () => {} });
        expect(library.container.querySelector('#export-scope-list')).toBeNull();
    });
});

describe('a list', () => {
    test("offers its collections' positions once collections are ticked", async () => {
        const { container } = render(ExportDatabaseModal, { visible: true, exportOptions: base(), onCancel: () => {}, onExport: () => {} });
        expect(box(container, '#export-collection-positions').disabled).toBe(true);
    });

    test('offers the review history once decks are ticked, unticked', async () => {
        const decks = [{ id: 5, name: 'Openings', cardCount: 12 }];
        const { container } = render(ExportDatabaseModal, { visible: true, exportOptions: base(), decks, onCancel: () => {}, onExport: () => {} });
        expect(box(container, '#export-anki-history').disabled).toBe(true);
        await fireEvent.click(box(container, '#export-decks'));
        expect(box(container, '#export-anki-history').disabled).toBe(false);
        expect(box(container, '#export-anki-history').checked).toBe(false);
    });
});

describe('the counts', () => {
    test("open what they count: the scope's positions, the matches offered", async () => {
        const onOpenPositions = vi.fn();
        const onOpenMatches = vi.fn();
        const matches = [
            { id: 3, player1_name: 'A', player2_name: 'B', game_count: 1 },
            { id: 8, player1_name: 'C', player2_name: 'D', game_count: 2 }
        ];
        const { getByTestId } = render(ExportDatabaseModal, {
            visible: true,
            exportOptions: { ...base(), wholeLibrary: true },
            positionCount: 40,
            matches,
            onOpenPositions,
            onOpenMatches,
            onCancel: () => {},
            onExport: () => {}
        });
        await fireEvent.click(getByTestId('count-link'));
        expect(onOpenPositions).toHaveBeenCalledWith(true);
        await fireEvent.click(getByTestId('export-matches-count'));
        expect(onOpenMatches).toHaveBeenCalledWith([3, 8]);
    });
});
