import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    IsPositionOnPile: vi.fn(),
    TogglePile: vi.fn()
}));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

import { IsPositionOnPile, TogglePile } from '../../wailsjs/go/database/Database.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { onPileStore, refreshPileState, togglePile } from '../services/pileService.js';

describe('pileService', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/x.db');
        positionStore.set({ ...emptyPosition(), id: 7 });
        onPileStore.set(false);
    });

    it('reads the marker of a stored position', async () => {
        IsPositionOnPile.mockResolvedValue(true);
        await refreshPileState();
        expect(IsPositionOnPile).toHaveBeenCalledWith(7);
        expect(get(onPileStore)).toBe(true);
    });

    it('does not ask the backend about a draft', async () => {
        positionStore.set(emptyPosition());
        onPileStore.set(true);
        await refreshPileState();
        expect(IsPositionOnPile).not.toHaveBeenCalled();
        expect(get(onPileStore)).toBe(false);
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
});
