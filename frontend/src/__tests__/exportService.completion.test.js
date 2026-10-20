/**
 * exportService.completion.test.js
 *
 * The export dialog has two screens: the form, and the progress laid over it. There is no
 * completion screen — asking for a click to acknowledge success is friction the status bar
 * already covers, and it was the screen whose sharp change of size left WebKitGTK painting
 * a blank rectangle. A finished export therefore closes the dialog and reports in the
 * status bar.
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

import { exportDatabase, handleExportCommit } from '../services/exportService.js';
import { exportModalModeStore, exportOptionsStore, resetExportState } from '../stores/exportModalStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { activeModal, statusBarTextStore } from '../stores/uiStore.js';
import { ExportDatabase } from '../../wailsjs/go/database/Database.js';
import { OpenExportDatabaseDialog } from '../../wailsjs/go/gui/App.js';

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
    test('closes the dialog instead of asking for an acknowledgement', async () => {
        await exportDatabase();
        expect(get(activeModal)).toBe('exportDatabase');

        await handleExportCommit();

        expect(get(activeModal)).toBeNull();
        expect(get(exportModalModeStore)).toBe('metadata');
    });

    test('reports what happened in the status bar', async () => {
        await exportDatabase();
        await handleExportCommit();

        // Status messages are stored as tMsg descriptors and translated at render time.
        expect(get(statusBarTextStore)).toEqual({
            i18nKey: 'status.exportCompleted',
            i18nParams: { posCount: 3 }
        });
    });

    test('reports the count the dialog showed, else the scope count', async () => {
        await exportDatabase();
        await handleExportCommit({ positions: 2 });
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'status.exportCompleted', i18nParams: { posCount: 2 } });

        await exportDatabase();
        await handleExportCommit(null);
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'status.exportCompleted', i18nParams: { posCount: 3 } });
    });

    test('sends identifiers rather than whole positions', async () => {
        await exportDatabase();
        await handleExportCommit();

        const sent = vi.mocked(ExportDatabase).mock.calls[0][0];
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

        const sent = vi.mocked(ExportDatabase).mock.calls[0][0];
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

        const sent = vi.mocked(ExportDatabase).mock.calls[0][0];
        expect(sent.watermark).toBe('');
        expect(sent.password).toBe('');
    });

    test('an incremental export of the library goes to its folder without asking for a file', async () => {
        await exportDatabase();
        exportOptionsStore.update((o) => ({ ...o, wholeLibrary: true, incremental: true, syncFolder: '/sync/vers-portable' }));
        await handleExportCommit();

        expect(vi.mocked(OpenExportDatabaseDialog)).not.toHaveBeenCalled();
        const sent = vi.mocked(ExportDatabase).mock.calls[0][0];
        expect(sent.incremental).toBe(true);
        expect(sent.exportPath).toBe('/sync/vers-portable');
        // The folder is offered again the next time the dialog opens.
        expect(get(exportOptionsStore).syncFolder).toBe('/sync/vers-portable');
    });

    test('a list export is never incremental', async () => {
        await exportDatabase();
        exportOptionsStore.update((o) => ({ ...o, wholeLibrary: false, incremental: true, syncFolder: '/sync/x' }));
        await handleExportCommit();

        const sent = vi.mocked(ExportDatabase).mock.calls[0][0];
        expect(sent.incremental).toBe(false);
        expect(sent.exportPath).toBe('/tmp/export.db');
    });
});
