import { describe, test, expect, vi } from 'vitest';

// Where the build renders PDFs, a sheet is saved as a PDF and opened in the
// viewer; the HTML page is never written.

const SaveBoardImageDialog = vi.fn(async (/** @type {any[]} */ ..._a) => '/tmp/feuille.pdf');
const SaveBoardSVG = vi.fn();
const WriteSheetPDF = vi.fn(async (/** @type {any[]} */ ..._a) => undefined);
const OpenLocalPage = vi.fn(async (/** @type {any[]} */ ..._a) => undefined);
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    SaveBoardImageDialog: (/** @type {any[]} */ ...a) => SaveBoardImageDialog(...a),
    SaveBoardSVG: (/** @type {any[]} */ ...a) => SaveBoardSVG(...a),
    WriteSheetPDF: (/** @type {any[]} */ ...a) => WriteSheetPDF(...a),
    OpenLocalPage: (/** @type {any[]} */ ...a) => OpenLocalPage(...a),
    SheetPDFAvailable: async () => true
}));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

const { saveAndOpenPage, pdfName } = await import('../services/printService.js');

describe('impression en PDF', () => {
    test('le nom proposé passe en .pdf', () => {
        expect(pdfName('blunderdb-match-3.html')).toBe('blunderdb-match-3.pdf');
        expect(pdfName('a.HTM')).toBe('a.pdf');
    });

    test('le PDF est rendu, enregistré puis ouvert', async () => {
        const path = await saveAndOpenPage('<html></html>', 'feuille.html', { saved: 's', failed: 'f' });
        expect(path).toBe('/tmp/feuille.pdf');
        expect(SaveBoardImageDialog).toHaveBeenCalledWith('pdf', 'feuille.pdf');
        expect(WriteSheetPDF).toHaveBeenCalledWith('/tmp/feuille.pdf', '<html></html>');
        expect(SaveBoardSVG).not.toHaveBeenCalled();
        expect(OpenLocalPage).toHaveBeenCalledWith('/tmp/feuille.pdf');
    });
});
