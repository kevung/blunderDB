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

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/a.db');
});

describe('handleDbFileDrop', () => {
    test('the dialog is in the active language and "Open" opens the dropped file', async () => {
        await setLanguage('fr');
        ShowQuestionDialog.mockImplementation(async (_title, _msg, buttons) => buttons[0]);

        await handleDbFileDrop('/x/autre.db');

        const [title, message, buttons] = ShowQuestionDialog.mock.calls[0];
        expect(title).toBe('Base de données déjà ouverte');
        expect(message).toContain('autre.db');
        expect(buttons).toEqual(['Ouvrir', 'Fusionner', 'Annuler']);
        expect(openDatabaseByPath).toHaveBeenCalledWith('/x/autre.db');
        await setLanguage('en');
    });

    test('Cancel does nothing', async () => {
        ShowQuestionDialog.mockImplementation(async (_title, _msg, buttons) => buttons[2]);
        await handleDbFileDrop('/x/autre.db');
        expect(openDatabaseByPath).not.toHaveBeenCalled();
    });
});
