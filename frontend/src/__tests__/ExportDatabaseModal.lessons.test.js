/**
 * ExportDatabaseModal.lessons.test.js
 *
 * The "Lessons" choice of the desktop export: the lessons are read when the dialog opens,
 * ticking the box selects them all, unticking one lesson leaves it out, and the chosen ids
 * reach the object the export service reads.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListLessons: vi.fn(async () => [
        { id: 4, name: 'Primes', stepCount: 3 },
        { id: 9, name: 'Holding games', stepCount: 1 }
    ])
}));

import ExportDatabaseModal from '../components/ExportDatabaseModal.svelte';

afterEach(cleanup);

describe('ExportDatabaseModal lessons', () => {
    test('ticking lessons selects them all; a lesson can be left out', async () => {
        const options = {
            includeMatches: false,
            matchIDs: [],
            includeTournaments: false,
            includeTournamentIDs: [],
            includeCollections: false,
            collectionIDs: [],
            includeLessons: false,
            lessonIDs: []
        };
        const onExport = vi.fn();
        const { container, getByText } = render(ExportDatabaseModal, { visible: true, exportOptions: options, onExport });
        const box = container.querySelector('#export-lessons');
        await waitFor(() => expect(/** @type {HTMLButtonElement | HTMLInputElement} */ (must(box)).disabled).toBe(false));

        await fireEvent.click(must(box));
        await tick();
        await waitFor(() => expect(getByText('Holding games')).toBeTruthy());

        await fireEvent.click(getByText('Holding games'));
        await tick();
        await fireEvent.click(must(container.querySelector('.btn-export')));
        expect(onExport).toHaveBeenCalled();
        expect(options.includeLessons).toBe(true);
        expect(options.lessonIDs).toEqual([4]);
    });
});
