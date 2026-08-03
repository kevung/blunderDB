/**
 * exportService.completion.test.js
 *
 * Regression: after a successful export the dialog fell back to its "preparing" screen — a
 * spinner reading "counting positions to export" that never resolved, with no way out but
 * Cancel. handleExportCommit set the mode to 'completed' and then a `finally` block called
 * resetExportState(), which put it straight back to 'preparing'. handleExportClose already
 * resets when the user closes the dialog, so that reset was both redundant and destructive.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenExportDatabaseDialog: vi.fn(() => Promise.resolve('/tmp/export.db')),
    OpenExportMatDialog: vi.fn(() => Promise.resolve('')),
    ShowAlert: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ExportDatabase: vi.fn(() => Promise.resolve()),
    ExportMatchMAT: vi.fn(() => Promise.resolve()),
    SuggestMatFilename: vi.fn(() => Promise.resolve('m.mat')),
    GetAllMatches: vi.fn(() => Promise.resolve([])),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([]))
}));

import { exportDatabase, handleExportCommit, handleExportClose } from '../services/exportService.js';
import { exportModalModeStore, exportOptionsStore, resetExportState } from '../stores/exportModalStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { activeModal } from '../stores/uiStore.js';
import { ExportDatabase } from '../../wailsjs/go/database/Database.js';

beforeEach(() => {
    // The stores live for the module's lifetime; without this the options chosen by one
    // test leak into the next.
    resetExportState();
    databasePathStore.set('/tmp/source.db');
    positionsStore.set([{ id: 1 }, { id: 2 }, { id: 3 }]);
    vi.clearAllMocks();
});

afterEach(() => {
    activeModal.set(null);
    positionsStore.set([]);
});

describe('a successful export', () => {
    test('leaves the dialog on its completion screen', async () => {
        await exportDatabase();
        await handleExportCommit();

        expect(get(exportModalModeStore)).toBe('completed');
    });

    test('closing the dialog is what resets it', async () => {
        await exportDatabase();
        await handleExportCommit();
        handleExportClose();

        expect(get(exportModalModeStore)).toBe('preparing');
        expect(get(activeModal)).toBeNull();
    });

    test('sends identifiers rather than whole positions', async () => {
        await exportDatabase();
        await handleExportCommit();

        const sent = ExportDatabase.mock.calls[0][0];
        expect(sent.positionIDs).toEqual([1, 2, 3]);
        expect(sent.positions).toBeUndefined();
    });

    test('carries the watermark and password chosen in the dialog', async () => {
        await exportDatabase();
        exportOptionsStore.update((o) => ({
            ...o,
            watermarkEnabled: true,
            watermark: '  Cours de Jean  ',
            watermarkNote: 'Ne pas rediffuser',
            passwordEnabled: true,
            password: 's3cret'
        }));
        await handleExportCommit();

        const sent = ExportDatabase.mock.calls[0][0];
        expect(sent.watermark).toBe('Cours de Jean');
        expect(sent.watermarkNote).toBe('Ne pas rediffuser');
        expect(sent.password).toBe('s3cret');
    });

    test('sends nothing for a mechanism left unticked', async () => {
        await exportDatabase();
        exportOptionsStore.update((o) => ({
            ...o,
            watermarkEnabled: false,
            watermark: 'ignoré',
            passwordEnabled: false,
            password: 'ignoré'
        }));
        await handleExportCommit();

        const sent = ExportDatabase.mock.calls[0][0];
        expect(sent.watermark).toBe('');
        expect(sent.password).toBe('');
    });
});
