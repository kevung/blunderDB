/**
 * exportService.scope.test.js
 *
 * What the export sends depends on what is listed. The whole library is the daemon's export
 * (ExportOptions.wholeLibrary): every family, a family picked whole sent as "all" (no ids).
 * A filtered list offers, and sends, only the matches its positions were played in.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

let library = false;

vi.mock('../stores/positionStore.js', async (importOriginal) => ({
    ...(await importOriginal()),
    browsingLibrary: () => library,
    listedIds: () => Promise.resolve([11, 12])
}));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenExportDatabaseDialog: vi.fn(() => Promise.resolve('/tmp/export.db')),
    OpenExportMatDialog: vi.fn(() => Promise.resolve('')),
    ShowAlert: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ExportDatabase: vi.fn(() => Promise.resolve()),
    CollectionCoverage: vi.fn(() => Promise.resolve({})),
    LoadMetadata: vi.fn(() => Promise.resolve({})),
    ExportMatchMAT: vi.fn(() => Promise.resolve()),
    SuggestMatFilename: vi.fn(() => Promise.resolve('m.mat')),
    GetAllMatches: vi.fn(() => Promise.resolve([{ id: 1 }, { id: 2 }, { id: 3 }])),
    GetAllCollections: vi.fn(() => Promise.resolve([{ id: 7 }, { id: 8 }])),
    GetAllTournaments: vi.fn(() => Promise.resolve([{ id: 4 }])),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([{ id: 5 }])),
    CountPositions: vi.fn(() => Promise.resolve(40)),
    ListLessons: vi.fn(() => Promise.resolve([{ id: 9 }])),
    MatchIDsReachedBy: vi.fn(() => Promise.resolve([2]))
}));

import { exportDatabase, handleExportCommit, setExportScope, scopeDefaults } from '../services/exportService.js';
import { exportMatchesStore, exportOptionsStore, exportPositionCountStore, exportScopeStore, resetExportState } from '../stores/exportModalStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { activeModal } from '../stores/uiStore.js';
import { ExportDatabase, MatchIDsReachedBy } from '../../wailsjs/go/database/Database.js';
import { OpenExportDatabaseDialog } from '../../wailsjs/go/gui/App.js';

/** The dialog's picks once it has auto-selected every item of the ticked families. */
function pickEverything() {
    exportOptionsStore.update((/** @type {any} */ o) => ({
        ...o,
        matchIDs: get(exportMatchesStore).map((/** @type {any} */ m) => m.id),
        collectionIDs: o.includeCollections ? [7, 8] : [],
        includeTournamentIDs: o.includeTournaments ? [4] : [],
        lessonIDs: o.includeLessons ? [9] : [],
        deckIDs: o.includeAnkiDecks ? [5] : []
    }));
}

beforeEach(() => {
    resetExportState();
    databasePathStore.set('/tmp/source.db');
    positionsStore.set([{ id: 11 }, { id: 12 }]);
    vi.clearAllMocks();
});

afterEach(() => {
    activeModal.set(null);
    positionsStore.set([]);
});

describe('the whole library', () => {
    test('is sent as the whole library, every family whole', async () => {
        library = true;
        await exportDatabase();
        pickEverything();
        await handleExportCommit();

        const sent = vi.mocked(ExportDatabase).mock.calls[0][0];
        expect(sent).toMatchObject({
            wholeLibrary: true,
            positionIDs: [],
            includeMatches: true,
            includeTranscriptions: true,
            includeAliases: true,
            matchIDs: [],
            includeCollections: true,
            collectionIDs: [],
            includeTournaments: true,
            tournamentIDs: [],
            includeLessons: true,
            lessonIDs: [],
            includeFilterLibrary: true,
            includeAnkiDecks: true,
            deckIDs: [],
            includeAnkiHistory: true
        });
        expect(MatchIDsReachedBy).not.toHaveBeenCalled();
    });

    test('keeps a narrowed pick as ids, and an emptied one as none', async () => {
        library = true;
        await exportDatabase();
        pickEverything();
        exportOptionsStore.update((/** @type {any} */ o) => ({ ...o, collectionIDs: [8], lessonIDs: [] }));
        await handleExportCommit();

        const sent = vi.mocked(ExportDatabase).mock.calls[0][0];
        expect(sent).toMatchObject({ includeCollections: true, collectionIDs: [8], includeLessons: false, lessonIDs: [] });
    });
});

describe('a filtered list', () => {
    test('offers and sends only the matches its positions reach', async () => {
        library = false;
        await exportDatabase();
        expect(MatchIDsReachedBy).toHaveBeenCalledWith([11, 12]);
        expect(get(exportMatchesStore).map((/** @type {any} */ m) => m.id)).toEqual([2]);
        pickEverything();
        await handleExportCommit();

        const sent = vi.mocked(ExportDatabase).mock.calls[0][0];
        expect(sent).toMatchObject({ wholeLibrary: false, positionIDs: [11, 12], includeMatches: true, includeTranscriptions: true, matchIDs: [2], includeAnkiDecks: false });
    });

    test('sends the drafts only with their matches, and not once unticked', async () => {
        library = false;
        await exportDatabase();
        pickEverything();
        exportOptionsStore.update((/** @type {any} */ o) => ({ ...o, includeTranscriptions: false }));
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({ includeMatches: true, includeTranscriptions: false });

        vi.clearAllMocks();
        await exportDatabase();
        pickEverything();
        exportOptionsStore.update((/** @type {any} */ o) => ({ ...o, includeMatches: false, matchIDs: [] }));
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({ includeMatches: false, includeTranscriptions: false });
    });

    test('sends the aliases only with their matches, and not once unticked', async () => {
        library = false;
        await exportDatabase();
        pickEverything();
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({ includeMatches: true, includeAliases: true });

        vi.clearAllMocks();
        await exportDatabase();
        pickEverything();
        exportOptionsStore.update((/** @type {any} */ o) => ({ ...o, includeAliases: false }));
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({ includeAliases: false });

        vi.clearAllMocks();
        await exportDatabase();
        pickEverything();
        exportOptionsStore.update((/** @type {any} */ o) => ({ ...o, includeMatches: false, matchIDs: [] }));
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({ includeMatches: false, includeAliases: false });
    });
});

describe('the file', () => {
    test('is asked for once the form is confirmed, and dismissing it keeps the form', async () => {
        await exportDatabase();
        expect(OpenExportDatabaseDialog).not.toHaveBeenCalled();
        vi.mocked(OpenExportDatabaseDialog).mockResolvedValueOnce('');
        await handleExportCommit();
        expect(ExportDatabase).not.toHaveBeenCalled();
        expect(get(activeModal)).not.toBeNull();
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({ exportPath: '/tmp/export.db' });
    });
});

describe('a list exported as the whole library', () => {
    test('offers every match and counts every position', async () => {
        await exportDatabase();
        expect(get(exportScopeStore).chooseable).toBe(true);
        expect(get(exportPositionCountStore)).toBe(2);
        await setExportScope(true);
        expect(get(exportMatchesStore)).toHaveLength(3);
        expect(get(exportPositionCountStore)).toBe(40);
        exportOptionsStore.update((/** @type {any} */ o) => ({ ...o, ...scopeDefaults(true) }));
        pickEverything();
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({ wholeLibrary: true, positionIDs: [], matchIDs: [] });
    });
});

describe('a list', () => {
    test("brings its collections' positions and its decks' history only when asked", async () => {
        await exportDatabase();
        exportOptionsStore.update((/** @type {any} */ o) => ({ ...o, includeCollections: true, collectionIDs: [7], includeAnkiDecks: true, deckIDs: [5] }));
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({
            includeCollectionPositions: false,
            includeAnkiDecks: true,
            deckIDs: [5],
            includeAnkiHistory: false
        });

        await exportDatabase();
        exportOptionsStore.update((/** @type {any} */ o) => ({
            ...o,
            includeCollections: true,
            collectionIDs: [7],
            includeCollectionPositions: true,
            includeAnkiDecks: true,
            deckIDs: [5],
            includeAnkiHistory: true
        }));
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[1][0]).toMatchObject({ includeCollectionPositions: true, includeAnkiHistory: true });
    });

    test('sends the drafts of the matches its tournaments bring', async () => {
        await exportDatabase();
        exportOptionsStore.update((/** @type {any} */ o) => ({ ...o, matchIDs: [], includeTournaments: true, includeTournamentIDs: [4] }));
        await handleExportCommit();
        expect(vi.mocked(ExportDatabase).mock.calls[0][0]).toMatchObject({ includeMatches: false, includeTranscriptions: true, includeAliases: true });
    });
});
