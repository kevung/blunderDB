/**
 * documentImport.test.js — l'assistant d'import de document : les règles de ce
 * que l'import refuse, et le composant monté, de la confirmation à l'import.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

const summary = (over = {}) => ({
    title: 'Livre',
    entries: 3,
    sure: 1,
    toVerify: 2,
    uncertain: 0,
    complete: 1,
    defaults: { description: 'score 0-0', context: 'ctx', confirmed: false, direction: { description: 'to the right', confirmed: false } },
    unverified: [2, 3],
    unverifiedComplete: [3],
    fallback: [3],
    fallbackComplete: [3],
    ...over
});
const confirmed = summary({ defaults: { description: 'score 0-0', context: 'ctx', confirmed: true, direction: { description: 'to the right', confirmed: true } } });

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    AnalyzeDocument: vi.fn(async () => '{"releve":1}'),
    SummarizeDocumentReleve: vi.fn(),
    ReviewDocumentReleve: vi.fn(async () => [{ rank: 3, reference: 'p. 3, Problem 4', state: 'to-verify', note: 'context not read' }]),
    ConfirmDocumentReleve: vi.fn(async () => '{"releve":2}'),
    ImportDocumentReleve: vi.fn(async () => ({ collection: 'Livre', added: 3, repeated: 0, entries: 3, positions: 3, newPositions: 3, complete: 1, fallback: 0 })),
    GetAllCollections: vi.fn(async () => [])
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ OpenDocumentDialog: vi.fn(async () => '/tmp/livre.pdf'), ReadFileContent: vi.fn() }));

import * as db from '../../wailsjs/go/database/Database.js';
import { refusedRanks, pendingConfirmations, startDocumentImport } from '../services/documentImportService.js';
import { documentImportStore, initialDocumentImport } from '../stores/documentImportStore.js';
import { activeModal, MODAL } from '../stores/uiStore.js';
import DocumentImportModal from '../components/DocumentImportModal.svelte';

describe('refusedRanks', () => {
    test('accepting complete entries and the fallback lifts what each covers', () => {
        const s = summary();
        expect(refusedRanks(s, { acceptComplete: false, fallback: false })).toEqual([2, 3]);
        expect(refusedRanks(s, { acceptComplete: true, fallback: false })).toEqual([3]);
        expect(refusedRanks(s, { acceptComplete: true, fallback: true })).toEqual([]);
        expect(refusedRanks(s, { acceptComplete: false, fallback: true })).toEqual([2]);
        expect(refusedRanks(null, { acceptComplete: false, fallback: false })).toEqual([]);
    });
    test('pending confirmations name what the document still asks', () => {
        expect(pendingConfirmations(summary())).toEqual({ defaults: true, direction: true });
        expect(pendingConfirmations(confirmed)).toEqual({ defaults: false, direction: false });
        expect(pendingConfirmations(summary({ defaults: undefined }))).toEqual({ defaults: false, direction: false });
    });
});

describe('the assistant, mounted', () => {
    beforeEach(() => {
        documentImportStore.set(initialDocumentImport());
        db.SummarizeDocumentReleve.mockReset().mockResolvedValueOnce(summary()).mockResolvedValue(confirmed);
    });
    afterEach(() => cleanup());

    test('read, confirm once, accept complete, import', async () => {
        const { getByTestId, queryByTestId } = render(DocumentImportModal, { visible: true });
        await startDocumentImport();
        expect(get(activeModal)).toBe(MODAL.DOCUMENT_IMPORT);
        await waitFor(() => expect(getByTestId('document-import-review')).toBeTruthy());
        expect(getByTestId('document-import-review').textContent).toContain('p. 3, Problem 4');
        expect(getByTestId('document-import-run').disabled).toBe(true);
        expect(getByTestId('document-import-fallback').disabled).toBe(true);

        await fireEvent.click(getByTestId('document-import-confirm'));
        await waitFor(() => expect(queryByTestId('document-import-confirm')).toBeNull());
        expect(db.ConfirmDocumentReleve).toHaveBeenCalledWith('{"releve":1}', true, true);

        await fireEvent.click(getByTestId('document-import-fallback'));
        await fireEvent.click(getByTestId('document-import-complete'));
        await waitFor(() => expect(getByTestId('document-import-run').disabled).toBe(false));
        await fireEvent.click(getByTestId('document-import-run'));
        await waitFor(() => expect(getByTestId('document-import-done')).toBeTruthy());
        expect(db.ImportDocumentReleve).toHaveBeenCalledWith('{"releve":2}', '', true, true, false);
    });
});
