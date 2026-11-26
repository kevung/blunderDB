import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// A paper archive page is a file the backend pipeline reads, dropped alone or
// collected from a folder; an HTML file that is not one is passed over by the
// backend, never refused. A scan image is not a file of the generic drop.

const ImportFiles = vi.fn();
const ImportXGMatch = vi.fn();
const ImportGnuBGMatch = vi.fn();
const IsDirectory = vi.fn(async () => false);

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    ImportFiles,
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog: vi.fn(),
    IsDirectory,
    QueueGammonNetBatch: vi.fn()
}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveIndividualPosition: vi.fn(),
    SaveAnalysis: vi.fn(),
    LoadComment: vi.fn(),
    SaveComment: vi.fn(),
    AnalyzeImportDatabase: vi.fn(),
    CommitImportDatabase: vi.fn(),
    CancelImport: vi.fn(),
    ImportXGMatch,
    ImportGnuBGMatch,
    ImportGnuBGMatchFromText: vi.fn(),
    ImportBGFMatch: vi.fn(),
    ImportBGFPosition: vi.fn(),
    ImportBGFPositionFromText: vi.fn(),
    ImportXGPPosition: vi.fn(),
    ParsePositionText: vi.fn(),
    BeginImportBatch: vi.fn(async () => 1),
    FinishImportBatch: vi.fn(),
    ImportReport: vi.fn(async () => null),
    ImportJournal: vi.fn(async () => []),
    RefreshSearchStatistics: vi.fn(),
    CountPositionsWithoutAnalysis: vi.fn(async () => 0)
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetAutoAnalyze: vi.fn(async () => false),
    GetGammonNetAnalysisPly: vi.fn(),
    GetGammonNetPruneK: vi.fn()
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn(), EventsOn: vi.fn(() => () => {}) }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions: vi.fn() }));

const { classifyDroppedFiles, importMultipleFiles } = await import('../services/importService.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { fileImportResultsStore } = await import('../stores/importModalStore.js');

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
});

describe("page d'archive papier", () => {
    test('glissée, elle est un fichier à importer ; une image reste une capture', async () => {
        const { importFiles, unsupported, captures } = await classifyDroppedFiles(['/d/archive.html', '/d/vieille.HTM', '/d/scan.png']);
        expect(importFiles).toEqual(['/d/archive.html', '/d/vieille.HTM']);
        expect(unsupported).toEqual([]);
        expect(captures).toEqual(['/d/scan.png']);
    });

    test('elle part au pipeline, jamais à un import fichier par fichier', async () => {
        // The backend passes over notes.html: one match, nothing refused.
        ImportFiles.mockResolvedValueOnce({
            succeeded: 1,
            skipped: 0,
            failed: 0,
            errors: [],
            hadMatches: true,
            lastPositionID: 0,
            cancelled: false
        });
        await importMultipleFiles(['/d/archive.html', '/d/notes.html']);
        expect(ImportFiles).toHaveBeenCalledTimes(1);
        expect(ImportFiles.mock.calls[0][0]).toEqual(['/d/archive.html', '/d/notes.html']);
        expect(ImportXGMatch).not.toHaveBeenCalled();
        expect(ImportGnuBGMatch).not.toHaveBeenCalled();
        const results = get(fileImportResultsStore);
        expect(results?.failed ?? 0).toBe(0);
    });
});
