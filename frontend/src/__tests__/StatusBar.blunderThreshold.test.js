import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve(undefined)),
    GetDatabaseStatsEstimate: vi.fn(() => Promise.resolve({ position_count: 10, match_count: 1, blunder_count: 2, approximate: [] })),
    GetLibrarySettings: vi.fn(() => Promise.resolve({ errorThresholdMP: 80, blunderThresholdMP: 175 }))
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ CancelGammonNetBatch: vi.fn(() => Promise.resolve(undefined)) }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ EventsOn: vi.fn(() => () => {}) }));

import { commandTextStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import StatusBar from '../components/StatusBar.svelte';

afterEach(cleanup);

describe('StatusBar — lien des blunders', () => {
    test('la recherche préparée porte le seuil de la bibliothèque', async () => {
        databasePathStore.set('/tmp/x.db');
        render(StatusBar);
        const link = await waitFor(() => screen.getByTitle(/175/));
        await fireEvent.click(link);
        expect(get(commandTextStore)).toBe('s E>175');
    });
});
