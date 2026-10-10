/**
 * The end-of-import panel lists the batch's journal (file, outcome, error) and
 * offers to resume an import the user stopped; a line that gave a match opens it.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import FileImportProgressModal from '../components/FileImportProgressModal.svelte';
import { latestJournalEntries } from '../services/importJournal.js';

afterEach(cleanup);

const journal = [
    { path: '/c/a.xg', size: 10, outcome: 'new', matchId: 7 },
    { path: '/c/b.xg', size: 10, outcome: 'duplicate', matchId: 7 },
    { path: '/c/c.xg', size: 3, outcome: 'enriched', matchId: 8 },
    { path: '/c/d.xg', size: 3, outcome: 'error', error: 'not a match' }
];

function mount(extra = {}) {
    return render(FileImportProgressModal, {
        props: { visible: true, mode: 'completed', totalFiles: 4, results: { succeeded: 1, failed: 1, skipped: 1, errors: [] }, journal, onClose: vi.fn(), ...extra }
    });
}

describe('FileImportProgressModal journal', () => {
    test('lists each file with its outcome and error message', () => {
        const { getByTestId } = mount();
        const box = getByTestId('import-journal');
        expect(box.querySelectorAll('.journal-item')).toHaveLength(4);
        expect(must(box.querySelector('[data-outcome="error"]')).textContent).toContain('not a match');
        expect(box.textContent).toContain('d.xg');
    });

    test('a line with a match opens it; an error line offers nothing to open', async () => {
        const onOpenMatch = vi.fn();
        const { getByTestId } = mount({ onOpenMatch });
        const buttons = getByTestId('import-journal').querySelectorAll('button');
        expect(buttons).toHaveLength(3);
        await fireEvent.click(buttons[2]);
        expect(onOpenMatch).toHaveBeenCalledWith(8);
    });

    test('Resume shows only for an interrupted import and calls back', async () => {
        const first = mount({ interrupted: false, onResume: vi.fn() });
        expect(first.queryByTestId('import-resume')).toBeNull();
        cleanup();
        const onResume = vi.fn();
        const { getByTestId } = mount({ interrupted: true, onResume });
        await fireEvent.click(getByTestId('import-resume'));
        expect(onResume).toHaveBeenCalled();
    });

    test('the journal is windowed', () => {
        const many = Array.from({ length: 250 }, (_, i) => ({ path: `/c/${i}.xg`, outcome: 'new', matchId: i + 1 }));
        const { getByTestId } = mount({ journal: many });
        expect(getByTestId('import-journal').querySelectorAll('.journal-item')).toHaveLength(101);
        expect(getByTestId('more-journal')).toBeTruthy();
    });
});

describe('latestJournalEntries', () => {
    test('a file tried again shows its last outcome', () => {
        const out = latestJournalEntries([
            { path: '/a', outcome: 'error', error: 'x' },
            { path: '/b', outcome: 'new', matchId: 1 },
            { path: '/a', outcome: 'new', matchId: 2 }
        ]);
        expect(out.map((e) => [e.path, e.outcome])).toEqual([
            ['/b', 'new'],
            ['/a', 'new']
        ]);
    });
});
