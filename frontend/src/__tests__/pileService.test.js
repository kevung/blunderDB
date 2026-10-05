import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    IsPositionOnPile: vi.fn(),
    TogglePile: vi.fn(),
    GetAllCollections: vi.fn()
}));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

import { IsPositionOnPile, TogglePile, GetAllCollections } from '../../wailsjs/go/database/Database.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { onPileStore, refreshPileState, togglePile, pileChangedStore } from '../services/pileService.js';

describe('pileService', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/x.db');
        positionStore.set({ ...emptyPosition(), id: 7 });
        onPileStore.set(false);
    });

    it('reads the marker of the position on the board', async () => {
        IsPositionOnPile.mockResolvedValue(true);
        await refreshPileState();
        expect(IsPositionOnPile).toHaveBeenCalledWith(get(positionStore));
        expect(get(onPileStore)).toBe(true);
    });

    it('keeps the marker of a draft already on the Pile when the board re-emits it', async () => {
        const draft = emptyPosition();
        positionStore.set(draft);
        TogglePile.mockResolvedValue({ onPile: true, brought: true });
        await togglePile();
        expect(get(onPileStore)).toBe(true);
        // The backend finds the draft by its hash, though its id is still 0.
        IsPositionOnPile.mockResolvedValue(true);
        positionStore.set({ ...draft });
        await refreshPileState();
        expect(IsPositionOnPile).toHaveBeenCalledWith(expect.objectContaining({ id: 0 }));
        expect(get(onPileStore)).toBe(true);
    });

    it('sends the displayed position to the backend and shows the answer', async () => {
        TogglePile.mockResolvedValue({ onPile: true, brought: false });
        await togglePile();
        expect(TogglePile).toHaveBeenCalledWith(get(positionStore));
        expect(get(onPileStore)).toBe(true);
    });

    it('does nothing without a database', async () => {
        databasePathStore.set('');
        await togglePile();
        expect(TogglePile).not.toHaveBeenCalled();
    });

    it('reloads the collections after a toggle, the Pile created on first use included', async () => {
        collectionsStore.set([]);
        const pile = { id: 3, name: 'Pile' };
        vi.mocked(GetAllCollections).mockResolvedValue(/** @type {any} */ ([pile]));
        vi.mocked(TogglePile).mockResolvedValue(/** @type {any} */ ({ onPile: true, brought: false }));
        const before = get(pileChangedStore);
        await togglePile();
        expect(GetAllCollections).toHaveBeenCalled();
        expect(get(collectionsStore)).toEqual([pile]);
        // The Collections panel listens to this signal to reload the Pile's rows when shown.
        expect(get(pileChangedStore)).toBe(before + 1);
    });
});
