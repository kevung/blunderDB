import { describe, test, expect, vi, beforeEach } from 'vitest';

// The document import is the one entry for any printed or written file: a
// sheet or an archive blunderDB produced imports by its codes, through the
// paper archive import, and its diagrams are never read.

const AnalyzeDocument = vi.fn(async () => '{"releve":1}');
const ImportPaperArchive = vi.fn(async () => ({ imported: [1], duplicates: 0, positions: 0, knownPositions: 0, missing: [], failed: [] }));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    AnalyzeDocument: (/** @type {any[]} */ ...a) => AnalyzeDocument(...a),
    ImportPaperArchive: (/** @type {any[]} */ ...a) => ImportPaperArchive(...a),
    ExportPaperArchive: vi.fn(),
    SummarizeDocumentReleve: vi.fn(),
    ReviewDocumentReleve: vi.fn(),
    ConfirmDocumentReleve: vi.fn(),
    ImportDocumentReleve: vi.fn(),
    GetAllCollections: vi.fn(async () => [])
}));
const OpenDocumentDialog = vi.fn();
/** @type {{ result: any }} */
const archiveTask = { result: null };
vi.mock('../services/longOpService.js', () => ({
    runLongOp: async (/** @type {string} */ _prefix, /** @type {() => Promise<number>} */ start) => {
        await start();
        return { status: 'done', result: archiveTask.result };
    }
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    // The import runs as a task; here it is the Database call it wraps.
    StartPaperArchiveImport: async (/** @type {any[]} */ ...a) => {
        archiveTask.result = await ImportPaperArchive(...a);
        return 1;
    },
    OpenDocumentDialog: (/** @type {any[]} */ ...a) => OpenDocumentDialog(...a),
    IsBlunderDBArchive: async (/** @type {string} */ p) => p.includes('feuille'),
    ReadFileContent: vi.fn(),
    OpenPaperArchiveDialog: vi.fn(),
    ExportPaperArchivePDF: vi.fn(),
    SaveBoardImageDialog: vi.fn(),
    SaveBoardSVG: vi.fn(),
    OpenLocalPage: vi.fn(),
    SheetPDFAvailable: vi.fn(async () => true),
    WriteSheetPDF: vi.fn()
}));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../stores/matchListStore.js', () => ({ matchListStore: { reload: vi.fn() } }));

const { startDocumentImport } = await import('../services/documentImportService.js');

beforeEach(() => vi.clearAllMocks());

describe("l'assistant d'import de document", () => {
    test('une feuille blunderDB choisie dans sa boîte part en archive papier', async () => {
        OpenDocumentDialog.mockResolvedValueOnce('/d/feuille.html');
        await startDocumentImport();
        expect(ImportPaperArchive).toHaveBeenCalledWith(['/d/feuille.html']);
        expect(AnalyzeDocument).not.toHaveBeenCalled();
    });

    test('un PDF blunderDB passé en argument aussi, sans boîte', async () => {
        await startDocumentImport('/d/feuille.pdf');
        expect(OpenDocumentDialog).not.toHaveBeenCalled();
        expect(ImportPaperArchive).toHaveBeenCalledWith(['/d/feuille.pdf']);
    });

    test('un livre de positions est lu par l’assistant', async () => {
        await startDocumentImport('/d/livre.pdf');
        expect(ImportPaperArchive).not.toHaveBeenCalled();
        expect(AnalyzeDocument).toHaveBeenCalled();
    });
});
