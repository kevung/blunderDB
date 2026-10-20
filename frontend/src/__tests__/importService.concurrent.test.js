import { describe, test, expect, vi } from 'vitest';
import { get } from 'svelte/store';

// One file import at a time. An import running in the background in group A while B is on
// screen keeps its own group: a second import is refused, and A's end analyses A, badges A with
// A's counts.

const ResumeImportBatch = vi.fn(async () => {});
const PendingImportFiles = vi.fn(async (files) => files.slice(1));
const BeginImportBatch = vi.fn(async () => 99);
/** @type {(v: any) => void} */
let finishA = () => {};
const ImportFiles = vi.fn(
    (paths) =>
        new Promise((resolve) => {
            finishA = () => resolve({ succeeded: paths.length, skipped: 0, failed: 0, errors: [], hadMatches: true, lastPositionID: 0, group: 1, batchClosed: true, report: null, journal: [] });
        })
);
const QueueGammonNetBatchIn = vi.fn();
const CountPositionsWithoutAnalysis = vi.fn(async () => 5);

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog: vi.fn(),
    IsDirectory: vi.fn(),
    QueueGammonNetBatch: vi.fn(),
    QueueGammonNetBatchIn,
    CancelLibraryImport: vi.fn(async () => {}),
    ImportFiles
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetAutoAnalyze: vi.fn(async () => true),
    GetGammonNetAnalysisPly: vi.fn(async () => 2),
    GetGammonNetPruneK: vi.fn(async () => 0),
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
    CountPositionsWithoutAnalysis
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn(), EventsOn: vi.fn(() => () => {}) }));
const setStatusBarMessage = vi.fn();
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: (/** @type {any} */ m) => setStatusBarMessage(m), openDatabaseByPath: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions: vi.fn() }));

const { importMultipleFiles } = await import('../services/importService.js');
const { activeGroupId, groupIdsStore, groupActivityStore } = await import('../stores/groupActivityStore.js');
const { databasePathStore } = await import('../stores/databaseStore.js');

describe('one file import at a time, attached to its group', () => {
    test('a second import is refused; the first ends in its own group', async () => {
        databasePathStore.set('/libs/a.db');
        groupIdsStore.set([1, 2]);
        activeGroupId.set(1);
        const first = importMultipleFiles(['/a.xg']);
        await vi.waitFor(() => expect(ImportFiles).toHaveBeenCalledTimes(1));

        // B comes on screen while A's import runs in the background.
        activeGroupId.set(2);
        expect(await importMultipleFiles(['/b.xg'])).toBeNull();
        expect(ImportFiles).toHaveBeenCalledTimes(1);
        expect(setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'status.importBusy', i18nParams: null });

        finishA();
        await first;
        // A's end analyses A, never B: no backlog count on the library on screen.
        expect(QueueGammonNetBatchIn).toHaveBeenCalledWith(1, 2, 0, 0);
        expect(CountPositionsWithoutAnalysis).not.toHaveBeenCalled();
        expect(get(groupActivityStore).get(1)).toEqual({
            state: 'done',
            job: 'import',
            message: { i18nKey: 'status.importDone', i18nParams: { succeeded: 1, skipped: 0, failed: 0 } }
        });

        // Once it is over, the next import goes through.
        const next = importMultipleFiles(['/b.xg']);
        await vi.waitFor(() => expect(ImportFiles).toHaveBeenCalledTimes(2));
        finishA();
        await next;
    });
});
