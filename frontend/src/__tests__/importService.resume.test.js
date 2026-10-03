import { describe, test, expect, vi } from 'vitest';

// A resumed batch reopens its id and sends the backend only the files its
// journal has not decided.

const ResumeImportBatch = vi.fn(async () => {});
const PendingImportFiles = vi.fn(async (files) => files.slice(1));
const BeginImportBatch = vi.fn(async () => 99);
const ImportFiles = vi.fn(async (paths) => ({
    succeeded: paths.length,
    skipped: 0,
    failed: 0,
    errors: [],
    hadMatches: true,
    lastPositionID: 0
}));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog: vi.fn(),
    IsDirectory: vi.fn(),
    StartGammonNetBatch: vi.fn(),
    ImportFiles
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
    AnalyzeImportDatabase: vi.fn(),
    CommitImportDatabase: vi.fn(),
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
    BeginImportBatch,
    FinishImportBatch: vi.fn(),
    ResumeImportBatch,
    PendingImportFiles,
    ImportReport: vi.fn(async () => null),
    RefreshSearchStatistics: vi.fn(),
    CountPositionsWithoutAnalysis: vi.fn(async () => 0)
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn(), EventsOn: vi.fn(() => () => {}) }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn(), openDatabaseByPath: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions: vi.fn() }));

const { resumeImportBatch } = await import('../services/importService.js');

describe('resumeImportBatch', () => {
    test('reopens the batch and imports only what the journal has not decided', async () => {
        await resumeImportBatch(12, ['/a.xg', '/b.xg']);
        expect(ResumeImportBatch).toHaveBeenCalledWith(12);
        expect(BeginImportBatch).not.toHaveBeenCalled();
        expect(ImportFiles).toHaveBeenCalledWith(['/b.xg']);
    });
});
