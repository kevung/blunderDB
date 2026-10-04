/**
 * « Reprendre un import » : la fenêtre liste les lots que rien n'a terminé,
 * et ne demande que l'endroit des fichiers avant de relancer le lot choisi.
 */

import { describe, test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

const mocks = vi.hoisted(() => ({
    ListImportBatches: vi.fn(),
    pickFilesToResumeFromFolder: vi.fn(() => Promise.resolve(['/d/a.xg'])),
    pickFilesToResume: vi.fn(() => Promise.resolve(['/d/b.xg'])),
    resumeImportBatch: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({ ListImportBatches: mocks.ListImportBatches }));
vi.mock('../services/importService.js', () => ({
    pickFilesToResumeFromFolder: mocks.pickFilesToResumeFromFolder,
    pickFilesToResume: mocks.pickFilesToResume,
    resumeImportBatch: mocks.resumeImportBatch
}));

import { fileImportModeStore } from '../stores/importModalStore.js';
import ResumeBatchModal from '../components/ResumeBatchModal.svelte';

const batches = [
    { id: 3, startedAt: '2026-10-01T09:00:00Z', finishedAt: '', source: '/data/bmab' },
    { id: 2, startedAt: '2026-09-30T09:00:00Z', finishedAt: '2026-09-30T10:00:00Z', source: '/data/done' },
    { id: 1, startedAt: '2026-09-29T09:00:00Z', finishedAt: '', source: '5 files' }
];

beforeEach(() => {
    vi.clearAllMocks();
    mocks.ListImportBatches.mockResolvedValue(batches);
});
afterEach(() => {
    cleanup();
    fileImportModeStore.set('idle');
});

describe('ResumeBatchModal', () => {
    test('ne liste que les lots inachevés', async () => {
        const { findAllByTestId } = render(ResumeBatchModal, { props: { visible: true, onClose: vi.fn() } });
        const rows = await findAllByTestId('resume-batch-row');
        expect(rows).toHaveLength(2);
        expect(rows[0].textContent).toContain('/data/bmab');
        expect(rows[1].textContent).toContain('5 files');
    });

    test('dit qu’il n’y a rien à reprendre quand tous les lots sont finis', async () => {
        mocks.ListImportBatches.mockResolvedValue([batches[1]]);
        const { findByTestId } = render(ResumeBatchModal, { props: { visible: true, onClose: vi.fn() } });
        await findByTestId('resume-batch-empty');
    });

    test('choisir le dossier ferme la fenêtre et relance le lot de la ligne', async () => {
        const onClose = vi.fn();
        const { findAllByTestId, getAllByRole } = render(ResumeBatchModal, { props: { visible: true, onClose } });
        await findAllByTestId('resume-batch-row');
        const folderButtons = getAllByRole('button').filter((b) => b.textContent.includes('folder'));
        await fireEvent.click(folderButtons[1]);
        await waitFor(() => expect(onClose).toHaveBeenCalled());
        expect(mocks.resumeImportBatch).toHaveBeenCalledWith(1, ['/d/a.xg']);
    });

    test('choisir des fichiers passe par l’autre sélecteur', async () => {
        const { findAllByTestId, getAllByRole } = render(ResumeBatchModal, { props: { visible: true, onClose: vi.fn() } });
        await findAllByTestId('resume-batch-row');
        const fileButtons = getAllByRole('button').filter((b) => b.textContent.includes('files'));
        await fireEvent.click(fileButtons[0]);
        await waitFor(() => expect(mocks.resumeImportBatch).toHaveBeenCalledWith(3, ['/d/b.xg']));
    });

    test('un sélecteur annulé laisse la fenêtre ouverte', async () => {
        mocks.pickFilesToResumeFromFolder.mockResolvedValueOnce(null);
        const onClose = vi.fn();
        const { findAllByTestId, getAllByRole } = render(ResumeBatchModal, { props: { visible: true, onClose } });
        await findAllByTestId('resume-batch-row');
        await fireEvent.click(getAllByRole('button').filter((b) => b.textContent.includes('folder'))[0]);
        await waitFor(() => expect(mocks.pickFilesToResumeFromFolder).toHaveBeenCalled());
        expect(mocks.resumeImportBatch).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();
    });

    test('pendant un import en cours, aucun lot n’est proposé à la reprise', async () => {
        fileImportModeStore.set('importing');
        const { findByTestId, queryAllByTestId } = render(ResumeBatchModal, { props: { visible: true, onClose: vi.fn() } });
        await findByTestId('resume-batch-running');
        expect(queryAllByTestId('resume-batch-row')).toHaveLength(0);
        expect(mocks.resumeImportBatch).not.toHaveBeenCalled();
    });
});
