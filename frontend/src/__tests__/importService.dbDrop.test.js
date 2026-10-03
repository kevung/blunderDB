import { describe, test, expect, vi, beforeEach } from 'vitest';

// Dropping a database on an open one asks in the user's language, and the answer
// is read back from the labels that were offered.

const ShowQuestionDialog = vi.fn();
const openDatabaseByPath = vi.fn();
const loadAllPositions = vi.fn();
const ImportXGMatch = vi.fn();
const ImportXGPPosition = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog,
    IsDirectory: vi.fn(),
    StartGammonNetBatch: vi.fn()
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

const { handleDbFileDrop } = await import('../services/importService.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { setLanguage } = await import('../i18n');
const { confirmModalStore, resolveConfirm } = await import('../services/confirmService.js');
const { get } = await import('svelte/store');

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/a.db');
});

describe('handleDbFileDrop', () => {
    test('the themed dialog is in the active language and Open opens the dropped file', async () => {
        await setLanguage('fr');
        const done = handleDbFileDrop('/x/autre.db');
        const message = await vi.waitFor(() => {
            const m = get(confirmModalStore);
            expect(m).not.toBeNull();
            return m;
        });
        expect(message.message).toContain('autre.db');
        expect(message.choices.map((c) => c.label)).toEqual(['Ouvrir', 'Fusionner']);
        expect(message.cancelLabel).toBe('Annuler');
        resolveConfirm('open');
        await done;
        expect(openDatabaseByPath).toHaveBeenCalledWith('/x/autre.db');
        await setLanguage('en');
    });

    test('dismissing does nothing', async () => {
        const done = handleDbFileDrop('/x/autre.db');
        await vi.waitFor(() => expect(get(confirmModalStore)).not.toBeNull());
        resolveConfirm(false);
        await done;
        expect(openDatabaseByPath).not.toHaveBeenCalled();
    });
});
