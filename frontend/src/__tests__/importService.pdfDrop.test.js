import { describe, test, expect, vi, beforeEach } from 'vitest';

// A dropped PDF that blunderDB printed goes to the paper archive import,
// whose counts the status bar shows; another PDF, a position book, to the
// document import. Neither goes to the match pipeline.

const ImportFiles = vi.fn();
const ImportXGMatch = vi.fn();
const ImportGnuBGMatch = vi.fn();
const IsDirectory = vi.fn(async () => false);

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    ImportFiles,
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: (/** @type {any[]} */ ...a) => OpenPositionFilesDialog(...a),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog: vi.fn(),
    IsDirectory,
    QueueGammonNetBatch: vi.fn(),
    OpenPaperArchiveDialog: vi.fn(),
    ExportPaperArchivePDF: vi.fn(),
    SaveBoardImageDialog: vi.fn(),
    OpenLocalPage: vi.fn(),
    SaveBoardSVG: vi.fn(),
    SheetPDFAvailable: vi.fn(async () => true),
    IsBlunderDBArchive: (/** @type {any[]} */ ...a) => IsBlunderDBArchive(...a),
    WriteSheetPDF: vi.fn()
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
    ImportPaperArchive: (/** @type {any[]} */ ...a) => ImportPaperArchive(...a),
    ExportPaperArchive: vi.fn(),
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
const setStatusBarMessage = vi.fn();
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: (/** @type {any[]} */ ...a) => setStatusBarMessage(...a) }));
const ImportPaperArchive = vi.fn();
const IsBlunderDBArchive = vi.fn(async (/** @type {string} */ p) => !p.includes('livre'));
const startDocumentImport = vi.fn();
const OpenPositionFilesDialog = vi.fn();
vi.mock('../services/documentImportService.js', () => ({ startDocumentImport: (/** @type {any[]} */ ...a) => startDocumentImport(...a) }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions: vi.fn() }));

vi.mock('../stores/matchListStore.js', () => ({ matchListStore: { reload: vi.fn() } }));
vi.mock('../i18n', async (orig) => ({ ...(await orig()), tMsg: (/** @type {string} */ k, /** @type {any} */ p) => ({ k, p }) }));

const { classifyDroppedFiles, handleFileDrop, importPosition } = await import('../services/importService.js');
const { databasePathStore } = await import('../stores/databaseStore.js');

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
});

describe('PDF déposé', () => {
    test('classé à part, ni capture ni fichier du pipeline', async () => {
        const out = await classifyDroppedFiles(['/d/feuille.PDF', '/d/m.xg', '/d/x.zip']);
        expect(out.archives).toEqual(['/d/feuille.PDF']);
        expect(out.importFiles).toEqual(['/d/m.xg']);
        expect(out.unsupported).toEqual(['/d/x.zip']);
    });

    test("part à l'import d'archive papier, ses compteurs à la barre d'état", async () => {
        ImportPaperArchive.mockResolvedValueOnce({ imported: [4], duplicates: 0, positions: 2, knownPositions: 1, missing: [], failed: [] });
        await handleFileDrop(0, 0, ['/d/archive.pdf', '/d/liste.pdf']);
        expect(ImportPaperArchive).toHaveBeenCalledWith(['/d/archive.pdf', '/d/liste.pdf']);
        expect(ImportFiles).not.toHaveBeenCalled();
        const last = setStatusBarMessage.mock.calls.at(-1)?.[0];
        expect(last).toEqual({ k: 'paperArchive.imported', p: expect.objectContaining({ imported: 1, positions: 2, knownPositions: 1 }) });
    });

    test("un PDF sans charge utile blunderDB part à l'assistant de document", async () => {
        await handleFileDrop(0, 0, ['/d/livre.pdf']);
        expect(ImportPaperArchive).not.toHaveBeenCalled();
        expect(startDocumentImport).toHaveBeenCalledWith('/d/livre.pdf');
    });

    test('Ctrl-I : chaque PDF suit sa route, le reste son import', async () => {
        OpenPositionFilesDialog.mockResolvedValueOnce(['/d/feuille.pdf', '/d/livre.pdf']);
        ImportPaperArchive.mockResolvedValueOnce({ imported: [], duplicates: 0, positions: 3, knownPositions: 0, missing: [], failed: [] });
        await importPosition();
        expect(ImportPaperArchive).toHaveBeenCalledWith(['/d/feuille.pdf']);
        expect(startDocumentImport).toHaveBeenCalledWith('/d/livre.pdf');
        expect(ImportFiles).not.toHaveBeenCalled();
    });

    test('sans base ouverte, rien ne part', async () => {
        databasePathStore.set('');
        await handleFileDrop(0, 0, ['/d/archive.pdf']);
        expect(ImportPaperArchive).not.toHaveBeenCalled();
        expect(setStatusBarMessage.mock.calls.at(-1)?.[0]).toEqual({ k: 'status.noDbOpenedBeforeImport', p: undefined });
    });
});
