/**
 * documentImport.test.js — l'assistant d'import de document, composant monté :
 * l'affichage de ce que Go rend (liste des refus, offre du repli, champs à
 * vérifier), la confirmation unique et l'import. Les règles de refus se testent
 * en Go (document.Refused).
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

const summary = (/** @type {boolean} */ confirmed) => ({
    title: 'Livre',
    entries: 2,
    sure: 1,
    toVerify: 1,
    uncertain: 0,
    complete: 1,
    defaults: { description: 'score 0-0', context: 'ctx', confirmed, direction: { description: 'to the right', confirmed } }
});
const review = (over = {}) => ({
    items: [
        { rank: 2, reference: 'p. 3, Problem 4', state: 'to-verify', checkers: 'sure', context: 'to-verify', complete: false, held: true, fallback: false, fields: ['cube', 'dice'], note: 'prose' }
    ],
    refused: [2],
    fallbackRanks: [],
    ...over
});

// Rank 1 is sure; rank 2 and 3 to verify — verified once VerifyDocumentEntry said so.
const sheet = (/** @type {string} */ releve) => {
    /** @type {number[]} */
    let verified = [];
    try {
        verified = JSON.parse(releve).verified ?? [];
    } catch {
        /* a relevé of the mock */
    }
    return [
        { rank: 1, page: 2, reference: 'p. 2, Problem 1', position: 'p1', state: 'sure', verified: false, fields: [], rect: { x: 0.1, y: 0.1, w: 0.5, h: 0.3 } },
        { rank: 2, page: 3, reference: 'p. 3, Problem 4', position: 'p2', state: 'to-verify', verified: verified.includes(2), fields: ['cube', 'dice'], rect: { x: 0.1, y: 0.5, w: 0.5, h: 0.3 } },
        { rank: 3, page: 3, reference: 'p. 3, Problem 5', position: 'p3', state: 'to-verify', verified: verified.includes(3), fields: ['checkers'] }
    ];
};

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    AnalyzeDocument: vi.fn(async () => '{"releve":1}'),
    SummarizeDocumentReleve: vi.fn(),
    ReviewDocumentReleve: vi.fn(),
    ConfirmDocumentReleve: vi.fn(async () => '{"releve":2}'),
    ImportDocumentReleve: vi.fn(async () => ({ collection: 'Livre', added: 2, repeated: 0, entries: 2, positions: 2, newPositions: 2, complete: 0, fallback: 1 })),
    GetAllCollections: vi.fn(async () => []),
    CancelImport: vi.fn(async () => {}),
    DocumentSheet: vi.fn(async (/** @type {string} */ r) => sheet(r)),
    LoadDocumentReleve: vi.fn(async () => ''),
    SaveDocumentReleve: vi.fn(async () => '/tmp/livre.pdf.releve.json'),
    VerifyDocumentEntry: vi.fn(async (/** @type {string} */ _p, /** @type {string} */ _r, /** @type {number} */ rank, /** @type {boolean} */ verified, /** @type {string} */ pos) =>
        JSON.stringify({ verified: verified ? [rank] : [], pos })
    ),
    DocumentDiagramImage: vi.fn(async () => 'data:image/png;base64,AAAA'),
    ParsePositionText: vi.fn(async () => ({ position: null }))
}));
const events = /** @type {Record<string, (p: any) => void>} */ ({});
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((/** @type {string} */ name, /** @type {(p: any) => void} */ fn) => {
        events[name] = fn;
        return () => {};
    })
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ OpenDocumentDialog: vi.fn(async () => '/tmp/livre.pdf'), ReadFileContent: vi.fn(), IsBlunderDBArchive: vi.fn(async () => false) }));

import * as db from '../../wailsjs/go/database/Database.js';
import * as app from '../../wailsjs/go/gui/App.js';
import { startDocumentImport, setChoice, cancelDocumentImport } from '../services/documentImportService.js';
import { documentImportStore, initialDocumentImport } from '../stores/documentImportStore.js';
import { activeModal, MODAL } from '../stores/uiStore.js';
import { importForegroundStore } from '../stores/groupActivityStore.js';
import DocumentImportModal from '../components/DocumentImportModal.svelte';

describe('the assistant, mounted', () => {
    beforeEach(() => {
        documentImportStore.set(initialDocumentImport());
        /** @type {any} */ (db.SummarizeDocumentReleve).mockReset().mockResolvedValueOnce(summary(false)).mockResolvedValue(summary(true));
        // Go's answers: nothing offered before the defaults are confirmed; then the fallback takes rank 2.
        /** @type {any} */ (db.ReviewDocumentReleve)
            .mockReset()
            .mockResolvedValueOnce(review())
            .mockImplementation(async (/** @type {any} */ _r, /** @type {any} */ _complete, /** @type {any} */ fallback) =>
                review({ fallbackRanks: [2], refused: fallback ? [] : [2], items: [{ ...review().items[0], held: !fallback, fallback }] })
            );
    });
    afterEach(() => cleanup());

    test('what Go says is what is shown; confirmation is one gesture; the choices go back to Go', async () => {
        const { getByTestId, queryByTestId } = render(DocumentImportModal, { visible: true });
        await startDocumentImport();
        expect(get(activeModal)).toBe(MODAL.DOCUMENT_IMPORT);
        await waitFor(() => expect(getByTestId('document-sheet-reference').textContent).toBe('p. 3, Problem 4'));
        expect(getByTestId('document-sheet-fields').textContent).toContain('cube, dice');
        expect(db.SaveDocumentReleve).toHaveBeenCalledWith('/tmp/livre.pdf', '{"releve":1}');
        expect(/** @type {HTMLButtonElement} */ (getByTestId('document-import-run')).disabled).toBe(true);
        expect(/** @type {HTMLButtonElement} */ (getByTestId('document-import-fallback')).disabled).toBe(true);
        expect(get(importForegroundStore)).toBe(false);

        await fireEvent.click(getByTestId('document-import-confirm'));
        await waitFor(() => expect(queryByTestId('document-import-confirm')).toBeNull());
        expect(db.ConfirmDocumentReleve).toHaveBeenCalledWith('{"releve":1}');
        expect(db.ConfirmDocumentReleve).toHaveBeenCalledTimes(1);

        await waitFor(() => expect(/** @type {HTMLButtonElement} */ (getByTestId('document-import-fallback')).disabled).toBe(false));
        await fireEvent.click(getByTestId('document-import-fallback'));
        await waitFor(() => expect(/** @type {HTMLButtonElement} */ (getByTestId('document-import-run')).disabled).toBe(false));
        expect(db.ReviewDocumentReleve).toHaveBeenLastCalledWith('{"releve":2}', false, true);

        await fireEvent.click(getByTestId('document-import-complete'));
        await waitFor(() => expect(db.ReviewDocumentReleve).toHaveBeenLastCalledWith('{"releve":2}', true, true));

        await fireEvent.click(getByTestId('document-import-run'));
        await waitFor(() => expect(getByTestId('document-import-done')).toBeTruthy());
        expect(db.ImportDocumentReleve).toHaveBeenCalledWith('{"releve":2}', '', true, true, false);
        expect(get(importForegroundStore)).toBe(false);
    });

    test('no group switch while reading: the foreground flag is raised and Cancel is off', async () => {
        /** @type {(v: any) => void} */
        let release = () => {};
        /** @type {any} */ (db.AnalyzeDocument).mockImplementationOnce(() => new Promise((r) => (release = r)));
        const { getByTestId } = render(DocumentImportModal, { visible: true });
        const started = startDocumentImport();
        await waitFor(() => expect(getByTestId('document-import-reading')).toBeTruthy());
        expect(get(importForegroundStore)).toBe(true);
        release('{"releve":1}');
        await started;
        await waitFor(() => expect(getByTestId('document-import-run')).toBeTruthy());
        expect(get(importForegroundStore)).toBe(false);
    });

    test('a review answered after a newer choice is ignored', async () => {
        const { getByTestId } = render(DocumentImportModal, { visible: true });
        await startDocumentImport();
        await waitFor(() => expect(getByTestId('document-sheet')).toBeTruthy());
        /** @type {(v: any) => void} */
        let slow = () => {};
        /** @type {any} */ (db.ReviewDocumentReleve).mockReset();
        /** @type {any} */ (db.ReviewDocumentReleve).mockImplementationOnce(() => new Promise((r) => (slow = r))).mockResolvedValueOnce(review({ refused: [], items: [] }));
        const first = setChoice('acceptComplete', true);
        const second = setChoice('fallback', true);
        await second;
        expect(get(documentImportStore).review.refused).toEqual([]);
        slow(review({ refused: [2, 99] }));
        await first;
        expect(get(documentImportStore).review.refused).toEqual([]);
    });
});

describe('the verification sheet', () => {
    beforeEach(() => {
        documentImportStore.set(initialDocumentImport());
        /** @type {any} */ (db.SummarizeDocumentReleve).mockReset().mockResolvedValue(summary(true));
        /** @type {any} */ (db.ReviewDocumentReleve).mockReset().mockResolvedValue(review());
        /** @type {any} */ (db.VerifyDocumentEntry).mockClear();
        /** @type {any} */ (db.AnalyzeDocument).mockClear();
    });
    afterEach(() => cleanup());

    test('the thumbnail beside the board; V verifies and moves on; E corrects; the counters filter', async () => {
        const { getByTestId, findByTestId } = render(DocumentImportModal, { visible: true });
        await startDocumentImport('/tmp/livre.pdf');
        await waitFor(() => expect(getByTestId('document-sheet-reference').textContent).toBe('p. 3, Problem 4'));
        expect(/** @type {HTMLImageElement} */ (await findByTestId('document-sheet-image')).src).toContain('data:image/png');
        expect(db.DocumentDiagramImage).toHaveBeenCalledWith('/tmp/livre.pdf', 3, 0.1, 0.5, 0.5, 0.3);
        expect(getByTestId('document-sheet-filter-to-verify').textContent).toContain('2');

        const box = getByTestId('document-sheet');
        await fireEvent.keyDown(box, { key: 'v' });
        expect(db.VerifyDocumentEntry).toHaveBeenLastCalledWith('/tmp/livre.pdf', '{"releve":1}', 2, true, '');
        await waitFor(() => expect(getByTestId('document-sheet-reference').textContent).toBe('p. 3, Problem 5'));
        expect(getByTestId('document-sheet-filter-verified').textContent).toContain('1');
        // The entry has no rectangle: no thumbnail, the board alone.
        expect(getByTestId('document-sheet-noimage')).toBeTruthy();

        await fireEvent.keyDown(box, { key: 'e' });
        const input = /** @type {HTMLInputElement} */ (await findByTestId('document-sheet-xgid'));
        await fireEvent.input(input, { target: { value: 'XGID=corrige' } });
        await fireEvent.keyDown(input, { key: 'Enter' });
        expect(db.VerifyDocumentEntry).toHaveBeenLastCalledWith('/tmp/livre.pdf', expect.any(String), 3, true, 'XGID=corrige');

        await fireEvent.click(getByTestId('document-sheet-filter-sure'));
        await waitFor(() => expect(getByTestId('document-sheet-reference').textContent).toBe('p. 2, Problem 1'));
    });

    test('a relevé kept beside the document is taken up again, not read anew', async () => {
        /** @type {any} */ (db.LoadDocumentReleve).mockResolvedValueOnce('{"verified":[2]}');
        const { getByTestId } = render(DocumentImportModal, { visible: true });
        await startDocumentImport('/tmp/livre.pdf');
        expect(db.AnalyzeDocument).not.toHaveBeenCalled();
        await waitFor(() => expect(getByTestId('document-sheet-reference').textContent).toBe('p. 3, Problem 5'));
    });
});

describe('reading progress and cancellation', () => {
    beforeEach(() => {
        documentImportStore.set(initialDocumentImport());
        /** @type {any} */ (db.SummarizeDocumentReleve).mockReset().mockResolvedValue(summary(true));
        /** @type {any} */ (db.ReviewDocumentReleve).mockReset().mockResolvedValue(review());
        /** @type {any} */ (app.OpenDocumentDialog).mockClear();
        /** @type {any} */ (db.CancelImport).mockClear();
    });
    afterEach(() => cleanup());

    test('a given path skips the file dialog', async () => {
        await startDocumentImport('/tmp/donne.pdf');
        expect(app.OpenDocumentDialog).not.toHaveBeenCalled();
        expect(db.AnalyzeDocument).toHaveBeenLastCalledWith('/tmp/donne.pdf', '', '');
    });

    test('the progress event is shown, and Cancel stops the reading', async () => {
        /** @type {(e: any) => void} */ let fail = () => {};
        /** @type {any} */ (db.AnalyzeDocument).mockImplementationOnce(() => new Promise((_, reject) => (fail = reject)));
        const { findByTestId, getByTestId } = render(DocumentImportModal, { visible: true });
        const started = startDocumentImport('/tmp/livre.pdf');
        await waitFor(() => expect(get(documentImportStore).phase).toBe('reading'));
        events['document-analyze:progress']({ page: 37, pages: 212, diagrams: 54 });
        expect((await findByTestId('document-import-progress')).textContent).toContain('37 / 212');
        await fireEvent.click(getByTestId('document-import-cancel'));
        expect(db.CancelImport).toHaveBeenCalled();
        fail(new Error('context canceled'));
        await started;
        expect(get(documentImportStore).phase).toBe('idle');
        expect(get(activeModal)).not.toBe(MODAL.DOCUMENT_IMPORT);
        cancelDocumentImport();
    });
});
