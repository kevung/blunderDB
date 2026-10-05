import { describe, test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

const countPositions = vi.fn(() => Promise.resolve(12));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve(undefined)),
    GetDatabaseStatsEstimate: vi.fn(() => Promise.resolve({ position_count: 10, match_count: 1, blunder_count: 340, approximate: [] })),
    GetLibrarySettings: vi.fn(() => Promise.resolve({ errorThresholdMP: 80, blunderThresholdMP: 175 })),
    CountPositionsByFilters: (...a) => countPositions(...a),
    CountPositions: vi.fn(() => Promise.resolve(10)),
    ListPositionIDs: vi.fn(() => Promise.resolve([]))
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ CancelGammonNetBatch: vi.fn(() => Promise.resolve(undefined)) }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ EventsOn: vi.fn(() => () => {}) }));
const processCommand = vi.fn();
vi.mock('../commandProcessor.js', () => ({ processCommand: (...a) => processCommand(...a) }));

import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { refreshLibraryCounts } from '../stores/libraryCountsStore.js';
import StatusBar from '../components/StatusBar.svelte';

beforeEach(() => {
    processCommand.mockClear();
    countPositions.mockClear();
});
afterEach(() => {
    cleanup();
    positionsStore.setIds([]);
});

describe('StatusBar — lien des blunders', () => {
    test('le clic lance la recherche au seuil de la bibliothèque', async () => {
        databasePathStore.set('/tmp/x.db');
        render(StatusBar);
        await waitFor(() => screen.getByTitle(/175/));
        const link = screen.getByTestId('count-blunders-total');
        await fireEvent.click(link);
        expect(processCommand).toHaveBeenCalledWith('s E>175');
    });

    test("une liste qui n'est pas la bibliothèque montre ses blunders puis le total", async () => {
        databasePathStore.set('/tmp/x.db');
        positionsStore.setIds([3, 4, 5]);
        await refreshLibraryCounts();
        render(StatusBar);
        const link = await waitFor(() => screen.getByTestId('count-blunders'));
        await waitFor(() => expect(link.textContent).toContain('12 / 340'));
        expect(countPositions).toHaveBeenCalledWith({ moveErrorFilter: 'E>175', restrictToPositionIDs: '3,4,5' });
        expect(screen.getByTestId('count-blunders-list').getAttribute('title')).toMatch(/12.*175/);
        expect(screen.getByTestId('count-blunders-total').getAttribute('title')).toMatch(/340.*175/);
    });

    test('les deux chiffres lancent deux recherches distinctes', async () => {
        databasePathStore.set('/tmp/x.db');
        positionsStore.setIds([3, 4, 5]);
        await refreshLibraryCounts();
        render(StatusBar);
        const list = await waitFor(() => screen.getByTestId('count-blunders-list'));
        await fireEvent.click(list);
        expect(processCommand).toHaveBeenLastCalledWith('ss E>175');
        await fireEvent.click(screen.getByTestId('count-blunders-total'));
        expect(processCommand).toHaveBeenLastCalledWith('s E>175');
        expect(processCommand).toHaveBeenCalledTimes(2);
    });

    test('la bibliothèque entière ne montre que le total', async () => {
        databasePathStore.set('/tmp/x.db');
        const { openLibrary } = await import('../stores/positionStore.js');
        await openLibrary();
        await refreshLibraryCounts();
        render(StatusBar);
        const link = await waitFor(() => screen.getByTestId('count-blunders-total'));
        expect(link.textContent).toContain('340');
        expect(screen.queryByTestId('count-blunders-list')).toBeNull();
    });
});
