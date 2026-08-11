/**
 * ImportProgressModals.escape.test.js
 *
 * fiche-09: Escape did nothing in ImportProgressModal/FileImportProgressModal,
 * even once the terminal state was reached and the "Fermer" button was on
 * screen. Fixed by closing on Escape only in the terminal state(s) — while an
 * import is actually running, Escape must not silently abandon it (Annuler is
 * the deliberate action for that).
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import ImportProgressModal from '../components/ImportProgressModal.svelte';
import FileImportProgressModal from '../components/FileImportProgressModal.svelte';

afterEach(cleanup);

describe('ImportProgressModal — Escape', () => {
    test('does nothing while analyzing', async () => {
        const onClose = vi.fn();
        const { container } = render(ImportProgressModal, {
            props: { visible: true, mode: 'analyzing', onCancel: vi.fn(), onCommit: vi.fn(), onClose }
        });
        await fireEvent.keyDown(container.querySelector('.modal-overlay'), { key: 'Escape' });
        expect(onClose).not.toHaveBeenCalled();
    });

    test('does nothing while committing', async () => {
        const onClose = vi.fn();
        const { container } = render(ImportProgressModal, {
            props: { visible: true, mode: 'committing', onCancel: vi.fn(), onCommit: vi.fn(), onClose }
        });
        await fireEvent.keyDown(container.querySelector('.modal-overlay'), { key: 'Escape' });
        expect(onClose).not.toHaveBeenCalled();
    });

    test('does nothing in the preview step when there is something to import (Annuler/Importer are the choices)', async () => {
        const onClose = vi.fn();
        const { container } = render(ImportProgressModal, {
            props: {
                visible: true,
                mode: 'preview',
                analysis: { toAdd: 3, toMerge: 0, toSkip: 0, total: 3, importPath: '' },
                onCancel: vi.fn(),
                onCommit: vi.fn(),
                onClose
            }
        });
        await fireEvent.keyDown(container.querySelector('.modal-overlay'), { key: 'Escape' });
        expect(onClose).not.toHaveBeenCalled();
    });

    test('closes in the preview step when there is nothing to import (its own "Fermer" button)', async () => {
        const onClose = vi.fn();
        const { container } = render(ImportProgressModal, {
            props: {
                visible: true,
                mode: 'preview',
                analysis: { toAdd: 0, toMerge: 0, toSkip: 2, total: 2, importPath: '' },
                onCancel: vi.fn(),
                onCommit: vi.fn(),
                onClose
            }
        });
        await fireEvent.keyDown(container.querySelector('.modal-overlay'), { key: 'Escape' });
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    test('closes once completed', async () => {
        const onClose = vi.fn();
        const { container } = render(ImportProgressModal, {
            props: { visible: true, mode: 'completed', onCancel: vi.fn(), onCommit: vi.fn(), onClose }
        });
        await fireEvent.keyDown(container.querySelector('.modal-overlay'), { key: 'Escape' });
        expect(onClose).toHaveBeenCalledTimes(1);
    });
});

describe('FileImportProgressModal — Escape', () => {
    test('does nothing while importing', async () => {
        const onClose = vi.fn();
        const { container } = render(FileImportProgressModal, {
            props: { visible: true, mode: 'importing', totalFiles: 5, currentIndex: 2, onCancel: vi.fn(), onClose }
        });
        await fireEvent.keyDown(container.querySelector('.modal-overlay'), { key: 'Escape' });
        expect(onClose).not.toHaveBeenCalled();
    });

    test('closes once completed', async () => {
        const onClose = vi.fn();
        const { container } = render(FileImportProgressModal, {
            props: {
                visible: true,
                mode: 'completed',
                results: { succeeded: 2, failed: 0, skipped: 0, errors: [] },
                onCancel: vi.fn(),
                onClose
            }
        });
        await fireEvent.keyDown(container.querySelector('.modal-overlay'), { key: 'Escape' });
        expect(onClose).toHaveBeenCalledTimes(1);
    });
});
