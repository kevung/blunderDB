import { describe, test, expect, vi, beforeEach } from 'vitest';

// A .db that appears in the watched folder is another machine's incremental
// export: it is merged without a preview, and a file that adds nothing counts
// as skipped.

const CommitImportDatabase = vi.fn();
const AnalyzeImportDatabase = vi.fn();
const loadAllPositions = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog: vi.fn(),
    IsDirectory: vi.fn(),
    QueueGammonNetBatch: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetAutoAnalyze: vi.fn(async () => false),
    GetGammonNetAnalysisPly: vi.fn(),
    GetGammonNetPruneK: vi.fn()
}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveIndividualPosition: vi.fn(),
    SaveAnalysis: vi.fn(),
    LoadComment: vi.fn(),
    SaveComment: vi.fn(),
    AnalyzeImportDatabase,
    CommitImportDatabase,
    CancelImport: vi.fn(),
    ImportXGMatch: vi.fn(),
    ImportGnuBGMatch: vi.fn(),
    ImportGnuBGMatchFromText: vi.fn(),
    ImportBGFMatch: vi.fn(),
    ImportOGXMMatch: vi.fn(),
    ImportBGFPosition: vi.fn(),
    ImportBGFPositionFromText: vi.fn(),
    ImportXGPPosition: vi.fn(),
    ParsePositionText: vi.fn(),
    BeginImportBatch: vi.fn(async () => 1),
    FinishImportBatch: vi.fn(),
    ImportReport: vi.fn(async () => null),
    RefreshSearchStatistics: vi.fn(),
    CountPositionsWithoutAnalysis: vi.fn(async () => 0)
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn(), EventsOn: vi.fn(() => () => {}) }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions }));

const { importWatchedFiles } = await import('../services/importService.js');

beforeEach(() => {
    vi.clearAllMocks();
});

describe('importWatchedFiles with databases', () => {
    test('merges each .db without a preview and counts what added nothing as skipped', async () => {
        CommitImportDatabase.mockResolvedValueOnce({ added: 12, matches: 1 }).mockResolvedValueOnce({ added: 0, matches: 0 });
        const r = await importWatchedFiles(['/sync/a1.db', '/sync/a2.DB']);
        expect(CommitImportDatabase.mock.calls.map((c) => c[0])).toEqual(['/sync/a1.db', '/sync/a2.DB']);
        expect(AnalyzeImportDatabase).not.toHaveBeenCalled();
        expect(r).toEqual({ succeeded: 1, skipped: 1, failed: 0 });
    });

    test('a database that cannot be merged is a failure, not a stop', async () => {
        CommitImportDatabase.mockRejectedValueOnce(new Error('not a database')).mockResolvedValueOnce({ added: 3 });
        const r = await importWatchedFiles(['/sync/bad.db', '/sync/good.db']);
        expect(r).toEqual({ succeeded: 1, skipped: 0, failed: 1 });
    });
});
