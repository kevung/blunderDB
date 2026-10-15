import { describe, test, expect, vi, beforeEach } from 'vitest';

// What the database import reports about matches and about what it set apart reaches the stores
// the modal reads.

const ShowQuestionDialog = vi.fn();
const openDatabaseByPath = vi.fn();
const loadAllPositions = vi.fn();
const ImportXGMatch = vi.fn();
const ImportXGPPosition = vi.fn();
const AnalyzeImportDatabase = vi.fn();
const CommitImportDatabase = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog,
    IsDirectory: vi.fn(),
    QueueGammonNetBatch: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetAutoAnalyze: vi.fn(async () => false),
    GetGammonNetAnalysisPly: vi.fn(),
    GetGammonNetPruneK: vi.fn(),
    SaveLanguage: vi.fn()
}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveIndividualPosition: vi.fn(),
    SaveAnalysis: vi.fn(),
    LoadComment: vi.fn(),
    SaveComment: vi.fn(),
    AnalyzeImportDatabase,
    CommitImportDatabase,
    CancelImport: vi.fn(),
    ImportXGMatch,
    ImportGnuBGMatch: vi.fn(),
    ImportGnuBGMatchFromText: vi.fn(),
    ImportBGFMatch: vi.fn(),
    ImportBGFPosition: vi.fn(),
    ImportBGFPositionFromText: vi.fn(),
    ImportXGPPosition,
    ParsePositionText: vi.fn(),
    BeginImportBatch: vi.fn(async () => 1),
    FinishImportBatch: vi.fn(),
    ImportReport: vi.fn(async () => null),
    RefreshSearchStatistics: vi.fn(),
    CountPositionsWithoutAnalysis: vi.fn(async () => 0)
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn(), openDatabaseByPath }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions }));

const { importDatabaseByPath, handleImportCommit } = await import('../services/importService.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { importAnalysisStore, importResultStore } = await import('../stores/importModalStore.js');
const { get } = await import('svelte/store');

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/a.db');
});

describe('the database import', () => {
    test('carries the matches of the analysis and of the commit to the stores', async () => {
        AnalyzeImportDatabase.mockResolvedValue({ toAdd: 1, toMerge: 0, toSkip: 0, total: 1, matches: 3, matchesKnown: 2 });
        CommitImportDatabase.mockResolvedValue({
            added: 1,
            merged: 0,
            skipped: 0,
            total: 1,
            matches: 3,
            matchesKnown: 2,
            livingCollectionsFrozen: ['A'],
            livingCollectionsOnManual: ['B'],
            unreadable: ['aliases']
        });
        await importDatabaseByPath('/x/src.db');
        expect(get(importAnalysisStore)).toMatchObject({ matches: 3, matchesKnown: 2 });
        await handleImportCommit();
        expect(get(importResultStore)).toMatchObject({
            matches: 3,
            matchesKnown: 2,
            livingCollectionsFrozen: ['A'],
            livingCollectionsOnManual: ['B'],
            unreadable: ['aliases']
        });
    });

    test('a result without those keys gives zeros and empty lists, never undefined', async () => {
        AnalyzeImportDatabase.mockResolvedValue({ toAdd: 1, toMerge: 0, toSkip: 0, total: 1 });
        CommitImportDatabase.mockResolvedValue({ added: 1, merged: 0, skipped: 0, total: 1, unreadable: null });
        await importDatabaseByPath('/x/src.db');
        await handleImportCommit();
        expect(get(importAnalysisStore)).toMatchObject({ matches: 0, matchesKnown: 0 });
        expect(get(importResultStore)).toMatchObject({ matchesKnown: 0, livingCollectionsFrozen: [], unreadable: [] });
    });
});
