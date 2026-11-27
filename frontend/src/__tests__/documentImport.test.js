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

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    AnalyzeDocument: vi.fn(async () => '{"releve":1}'),
    SummarizeDocumentReleve: vi.fn(),
    ReviewDocumentReleve: vi.fn(),
    ConfirmDocumentReleve: vi.fn(async () => '{"releve":2}'),
    ImportDocumentReleve: vi.fn(async () => ({ collection: 'Livre', added: 2, repeated: 0, entries: 2, positions: 2, newPositions: 2, complete: 0, fallback: 1 })),
    GetAllCollections: vi.fn(async () => [])
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ OpenDocumentDialog: vi.fn(async () => '/tmp/livre.pdf'), ReadFileContent: vi.fn(), IsBlunderDBArchive: vi.fn(async () => false) }));

import * as db from '../../wailsjs/go/database/Database.js';
import { startDocumentImport, setChoice } from '../services/documentImportService.js';
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
        await waitFor(() => expect(getByTestId('document-import-review')).toBeTruthy());
        expect(getByTestId('document-import-review').textContent).toContain('p. 3, Problem 4');
        expect(getByTestId('document-import-review').textContent).toContain('cube, dice');
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
        await waitFor(() => expect(getByTestId('document-import-review')).toBeTruthy());
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
