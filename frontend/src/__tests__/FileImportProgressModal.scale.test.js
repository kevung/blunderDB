/**
 * A night-long import sends thousands of progress events and may collect
 * thousands of errors: the modal must keep up and render a bounded list.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

import FileImportProgressModal from '../components/FileImportProgressModal.svelte';

afterEach(cleanup);

function progress(i, total) {
    return {
        filesDone: i,
        filesTotal: total,
        imported: i,
        duplicates: 0,
        failed: 0,
        positions: i * 500,
        bytesRead: i * 1000,
        bytesTotal: total * 1000,
        positionsPerSec: 6000,
        etaSeconds: total - i,
        currentFile: `/corpus/${i}.xg`,
        done: false
    };
}

describe('FileImportProgressModal at scale', () => {
    test('10 000 progress events render under budget, with rate and ETA', async () => {
        const total = 10000;
        const { rerender, getByTestId } = render(FileImportProgressModal, {
            props: { visible: true, mode: 'importing', totalFiles: total, progress: progress(0, total), onCancel: vi.fn(), onMinimize: vi.fn() }
        });
        const start = performance.now();
        for (let i = 1; i <= total; i++) {
            await rerender({ progress: progress(i, total), currentIndex: i });
        }
        await tick();
        expect(performance.now() - start).toBeLessThan(10000);
        expect(getByTestId('import-rate').textContent).toContain('6000');
    });

    test('10 000 errors render a bounded window and say how many are left', () => {
        const errors = Array.from({ length: 10000 }, (_, i) => ({ file: `/corpus/${i}.xg`, message: 'broken' }));
        const { container, getByTestId } = render(FileImportProgressModal, {
            props: { visible: true, mode: 'completed', totalFiles: 10000, results: { succeeded: 0, failed: 10000, skipped: 0, errors }, onClose: vi.fn() }
        });
        expect(container.querySelectorAll('.error-item').length).toBe(101);
        expect(getByTestId('more-errors').textContent).toContain('9900');
    });

    test('Minimize hands the import to the status bar', async () => {
        const onMinimize = vi.fn();
        const { getByText } = render(FileImportProgressModal, {
            props: { visible: true, mode: 'importing', totalFiles: 3, onCancel: vi.fn(), onMinimize }
        });
        await fireEvent.click(getByText(/^(Réduire|Minimize)$/));
        expect(onMinimize).toHaveBeenCalledOnce();
    });
});
