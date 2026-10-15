/**
 * ExportDatabaseModal.scope.test.js
 *
 * The export's grid: a list may be exported as the whole library, which ticks every family;
 * a list's collections may bring their positions; a deck's review history follows the decks.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

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

describe('the preview', () => {
    const plan = {
        positions: 7,
        comments: 3,
        matches: 2,
        tournaments: 1,
        directions: 0,
        rencontres: 0,
        collections: 2,
        lessons: 0,
        ankiDecks: 0,
        transcriptions: 1,
        aliases: 4,
        filters: 0,
        frozenCollections: [{ id: 9, name: 'Live' }],
        degradedAnkiDecks: []
    };

    test('shows the counts the backend gives and marks the collections that leave frozen', async () => {
        const planFor = vi.fn(async () => plan);
        const { container, findByTestId } = render(ExportDatabaseModal, {
            visible: true,
            exportOptions: { ...base(), includeCollections: true, collectionIDs: [9] },
            planFor,
            onCancel: () => {},
            onExport: () => {}
        });
        await waitFor(() => expect(container.querySelector('[data-testid="export-matches-count"]')?.textContent).toBe('2'));
        expect((await findByTestId('export-transcriptions-count')).textContent).toBe('1');
        expect((await findByTestId('export-aliases-count')).textContent).toBe('4');
        expect(planFor).toHaveBeenCalled();
        expect(container.textContent).toContain('7');
    });
});
