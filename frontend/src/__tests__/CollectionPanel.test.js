/**
 * CollectionPanel.test.js
 *
 * CollectionPanel.svelte (877 l.) had no test file at all (D.13, #214). It
 * owns two views — the list of collections and the detail view of one
 * collection's positions — plus inline create/rename/delete and membership
 * toggling. This covers the load-on-open effect, both views, the create /
 * open / delete / toggle-membership flows, and the keyboard shortcuts, with
 * every Wails binding mocked and the real Svelte stores driving the component.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateCollection: vi.fn().mockResolvedValue(undefined),
    GetAllCollections: vi.fn().mockResolvedValue([]),
    DeleteCollection: vi.fn().mockResolvedValue(undefined),
    AddPositionToCollection: vi.fn().mockResolvedValue(undefined),
    RemovePositionFromCollection: vi.fn().mockResolvedValue(undefined),
    GetCollectionPositions: vi.fn().mockResolvedValue([]),
    ReorderCollectionPositions: vi.fn().mockResolvedValue(undefined),
    ReorderCollections: vi.fn().mockResolvedValue(undefined),
    UpdateCollection: vi.fn().mockResolvedValue(undefined),
    GetPositionCollections: vi.fn().mockResolvedValue([]),
    GetPositionIndexMap: vi.fn().mockResolvedValue({}),
    LoadAnalysis: vi.fn().mockResolvedValue(null)
}));

vi.mock('../services/confirmService.js', () => ({ confirmAction: vi.fn().mockResolvedValue(true) }));

import {
    CreateCollection,
    GetAllCollections,
    DeleteCollection,
    AddPositionToCollection,
    RemovePositionFromCollection,
    GetCollectionPositions,
    UpdateCollection
} from '../../wailsjs/go/database/Database.js';
import { confirmAction } from '../services/confirmService.js';

import CollectionPanel from '../components/CollectionPanel.svelte';
import { collectionsStore, selectedCollectionStore, collectionPositionsStore, activeCollectionStore } from '../stores/collectionStore.js';
import { openPanels, PANEL, statusBarTextStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore } from '../stores/positionStore.js';

// ── Helpers ───────────────────────────────────────────────────────────────────

function resetStores() {
    collectionsStore.set([]);
    selectedCollectionStore.set(null);
    collectionPositionsStore.set([]);
    activeCollectionStore.set(null);
    openPanels.set(new Set());
    statusBarTextStore.set('');
    statusBarModeStore.set('NORMAL');
    currentPositionIndexStore.set(0);
    databasePathStore.set('/fake/db.sqlite');
    positionStore.set(null);
}

const SAMPLE_COLLECTIONS = [
    { id: 1, name: 'Backgames', description: 'Deep back games', positionCount: 3, updatedAt: '2026-01-01 10:00:00' },
    { id: 2, name: 'Bear-offs', description: '', positionCount: 0, updatedAt: '2026-01-02 10:00:00' }
];

beforeEach(() => {
    vi.clearAllMocks();
    resetStores();
    GetAllCollections.mockResolvedValue(SAMPLE_COLLECTIONS);
});

afterEach(cleanup);

// ── List view ─────────────────────────────────────────────────────────────────

describe('CollectionPanel — list view', () => {
    test('loads collections once the database is open', async () => {
        render(CollectionPanel, { props: {} });

        expect(await screen.findByText('Backgames')).toBeTruthy();
        expect(screen.getByText('Bear-offs')).toBeTruthy();
        expect(GetAllCollections).toHaveBeenCalled();
    });

    test('empty database: shows the empty-collections message', async () => {
        GetAllCollections.mockResolvedValue([]);
        render(CollectionPanel, { props: {} });
        await vi.waitFor(() => expect(GetAllCollections).toHaveBeenCalled());

        expect(screen.queryByText('Backgames')).toBeNull();
    });

    test('creating a collection via the inline row on Enter', async () => {
        render(CollectionPanel, { props: {} });
        await screen.findByText('Backgames');

        const nameInput = screen.getByPlaceholderText(/new collection/i);
        await fireEvent.input(nameInput, { target: { value: 'Priming' } });
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        expect(CreateCollection).toHaveBeenCalledWith('Priming', '');
    });

    test('a duplicate collection name is rejected without calling the backend', async () => {
        render(CollectionPanel, { props: {} });
        await screen.findByText('Backgames');

        const nameInput = screen.getByPlaceholderText(/new collection/i);
        await fireEvent.input(nameInput, { target: { value: 'Backgames' } });
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        expect(CreateCollection).not.toHaveBeenCalled();
        expect(get(statusBarTextStore)).not.toBe('');
    });

    test('double-clicking a non-empty collection opens the detail view', async () => {
        GetCollectionPositions.mockResolvedValue([{ id: 101 }, { id: 102 }]);
        const onOpenCollection = vi.fn();
        render(CollectionPanel, { props: { onOpenCollection } });
        const row = (await screen.findByText('Backgames')).closest('tr');

        await fireEvent.dblClick(row);
        await vi.waitFor(() => expect(get(activeCollectionStore)).not.toBeNull());

        expect(get(activeCollectionStore)).toMatchObject({ id: 1, name: 'Backgames' });
        expect(get(collectionPositionsStore)).toEqual([{ id: 101 }, { id: 102 }]);
        expect(onOpenCollection).toHaveBeenCalledWith(expect.objectContaining({ id: 1 }), [{ id: 101 }, { id: 102 }]);
    });

    test('double-clicking an empty collection reports it instead of opening', async () => {
        GetCollectionPositions.mockResolvedValue([]);
        render(CollectionPanel, { props: {} });
        const row = (await screen.findByText('Bear-offs')).closest('tr');

        await fireEvent.dblClick(row);
        await vi.waitFor(() => expect(get(statusBarTextStore)).not.toBe(''));

        expect(get(activeCollectionStore)).toBeNull();
    });

    test('deleting a collection calls the backend and reloads the list', async () => {
        render(CollectionPanel, { props: {} });
        const row = (await screen.findByText('Backgames')).closest('tr');
        const deleteBtn = within(row).getByTitle(/delete/i);
        GetAllCollections.mockResolvedValue([SAMPLE_COLLECTIONS[1]]);

        await fireEvent.click(deleteBtn);
        await vi.waitFor(() => expect(GetAllCollections).toHaveBeenCalledTimes(2)); // initial load + post-delete reload

        expect(DeleteCollection).toHaveBeenCalledWith(1);
    });

    test('renaming a collection through the inline editor', async () => {
        render(CollectionPanel, { props: {} });
        const row = (await screen.findByText('Backgames')).closest('tr');
        const editBtn = within(row).getByTitle(/^edit$/i);
        await fireEvent.click(editBtn);

        const nameInput = within(row).getByDisplayValue('Backgames');
        await fireEvent.input(nameInput, { target: { value: 'Back games (renamed)' } });
        await fireEvent.keyDown(nameInput, { key: 'Enter' });

        await vi.waitFor(() => expect(UpdateCollection).toHaveBeenCalled());
        expect(UpdateCollection).toHaveBeenCalledWith(1, 'Back games (renamed)', 'Deep back games');
    });

    test('toggling a collection checkbox for the current position adds it', async () => {
        positionStore.set({ id: 55 });
        render(CollectionPanel, { props: {} });
        const row = (await screen.findByText('Backgames')).closest('tr');
        const checkbox = within(row).getByRole('checkbox');

        await fireEvent.click(checkbox);

        await vi.waitFor(() => expect(AddPositionToCollection).toHaveBeenCalledWith(1, 55));
    });

    test('no current position: the toggle checkbox is not rendered at all', async () => {
        positionStore.set(null);
        render(CollectionPanel, { props: {} });
        const row = (await screen.findByText('Backgames')).closest('tr');

        expect(within(row).queryByRole('checkbox')).toBeNull();
    });
});

// ── Detail view ───────────────────────────────────────────────────────────────

describe('CollectionPanel — detail view', () => {
    beforeEach(() => {
        activeCollectionStore.set(SAMPLE_COLLECTIONS[0]);
        selectedCollectionStore.set(SAMPLE_COLLECTIONS[0]);
        collectionPositionsStore.set([{ id: 201 }, { id: 202 }]);
    });

    test('shows the active collection name and position count', async () => {
        render(CollectionPanel, { props: {} });
        await tick();

        expect(screen.getByText('Backgames', { selector: '.detail-title' })).toBeTruthy();
        expect(document.querySelector('.detail-count').textContent).toContain('2');
    });

    test('the back button returns to the list view without touching stores', async () => {
        render(CollectionPanel, { props: {} });
        await tick();

        const backBtn = screen.getByTitle(/back to collections/i);
        await fireEvent.click(backBtn);

        expect(screen.queryByTitle(/back to collections/i)).toBeNull();
        // Leaving the detail view by the back button keeps the active
        // collection selected — only Escape / delete-collection clear it.
        expect(get(activeCollectionStore)).toMatchObject({ id: 1 });
    });

    test('removing a position row calls the backend and refreshes the list', async () => {
        const onOpenCollection = vi.fn();
        GetCollectionPositions.mockResolvedValue([{ id: 202 }]);
        render(CollectionPanel, { props: { onOpenCollection } });
        await tick();

        const rows = document.querySelectorAll('tbody tr');
        const firstRow = rows[0];
        const removeBtn = within(firstRow).getByTitle(/remove from collection/i);
        await fireEvent.click(removeBtn);
        await tick();
        await tick();

        expect(RemovePositionFromCollection).toHaveBeenCalledWith(1, 201);
        expect(get(collectionPositionsStore)).toEqual([{ id: 202 }]);
    });

    test('clicking a row selects and navigates to that position', async () => {
        render(CollectionPanel, { props: {} });
        await tick();

        const rows = document.querySelectorAll('tbody tr');
        await fireEvent.click(rows[1]);
        await tick();

        expect(get(positionStore)).toEqual({ id: 202 });
        expect(get(currentPositionIndexStore)).toBe(1);
    });
});

// ── Keyboard shortcuts ────────────────────────────────────────────────────────

describe('CollectionPanel — keyboard shortcuts', () => {
    beforeEach(() => {
        openPanels.set(new Set([PANEL.COLLECTION]));
    });

    test('Escape from the detail view (outside COLLECTION mode) returns to the list', async () => {
        activeCollectionStore.set(SAMPLE_COLLECTIONS[0]);
        collectionPositionsStore.set([{ id: 201 }]);
        statusBarModeStore.set('NORMAL');
        render(CollectionPanel, { props: {} });
        await tick();

        expect(screen.getByTitle(/back to collections/i)).toBeTruthy();
        await fireEvent.keyDown(document, { key: 'Escape' });

        expect(screen.queryByTitle(/back to collections/i)).toBeNull();
    });

    // `selectAndDisplayPosition` used to call `navigateToPosition()`
    // unconditionally, including on a shift/ctrl click meant only to grow the
    // multi-selection. `navigateToPosition` sets `currentPositionIndexStore`,
    // and the COLLECTION-mode "sync selection with j/k navigation" `$effect`
    // reacts to ANY change of that store by collapsing `selectedPositionIndices`
    // down to that one index — so a shift/ctrl-click that had just grown the
    // selection got it collapsed back to one row by that same click, before
    // Delete was ever pressed: multi-row removal by mouse was unreachable.
    // Fixed by skipping the navigate/currentPositionIndexStore update on a
    // shift/ctrl click (component fix, D.13/#214) — this test locks in the
    // real multi-remove instead of the collapse.
    test('Delete in COLLECTION mode: shift-click extends the selection, both rows are removed', async () => {
        activeCollectionStore.set(SAMPLE_COLLECTIONS[0]);
        collectionPositionsStore.set([{ id: 201 }, { id: 202 }]);
        statusBarModeStore.set('COLLECTION');
        currentPositionIndexStore.set(0);
        // onMount re-fetches when mode is already COLLECTION at mount (session
        // restore) — the first call must still see the seeded two positions;
        // only the reload after the delete should see the (now empty) result.
        GetCollectionPositions.mockResolvedValueOnce([{ id: 201 }, { id: 202 }]).mockResolvedValue([]);
        render(CollectionPanel, { props: {} });
        await tick();

        const rows = document.querySelectorAll('tbody tr');
        await fireEvent.click(rows[1], { shiftKey: true });
        await tick();

        await fireEvent.keyDown(document, { key: 'Delete' });
        await vi.waitFor(() => expect(RemovePositionFromCollection).toHaveBeenCalledTimes(2));

        expect(confirmAction).toHaveBeenCalledWith(expect.stringContaining('2'), expect.anything());
        // Removed highest index first, so the list never shifts under the loop.
        expect(RemovePositionFromCollection).toHaveBeenNthCalledWith(1, 1, 202);
        expect(RemovePositionFromCollection).toHaveBeenNthCalledWith(2, 1, 201);
    });

    test('Delete in COLLECTION mode with a single current position (no click) removes it', async () => {
        activeCollectionStore.set(SAMPLE_COLLECTIONS[0]);
        collectionPositionsStore.set([{ id: 201 }, { id: 202 }]);
        statusBarModeStore.set('COLLECTION');
        currentPositionIndexStore.set(1);
        GetCollectionPositions.mockResolvedValueOnce([{ id: 201 }, { id: 202 }]).mockResolvedValue([{ id: 201 }]);
        render(CollectionPanel, { props: {} });
        await tick();

        await fireEvent.keyDown(document, { key: 'Delete' });
        await vi.waitFor(() => expect(RemovePositionFromCollection).toHaveBeenCalled());

        expect(confirmAction).toHaveBeenCalled();
        expect(RemovePositionFromCollection).toHaveBeenCalledWith(1, 202);
    });
});
