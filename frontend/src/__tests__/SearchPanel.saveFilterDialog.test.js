/**
 * SearchPanel.saveFilterDialog.test.js
 *
 * fiche-09: the "save this search as a filter" mini-dialog (search history →
 * bookmark icon) had no role="dialog", no focus trap, no Escape, no
 * autofocus — plain divs styled to look like a dialog but invisible to
 * assistive tech and unusable from the keyboard. This locks the fix: the
 * dialog now gets role="dialog" + aria-modal, use:trapFocus (which also
 * autofocuses the name field), and Escape closes it.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

import SearchPanel from '../components/SearchPanel.svelte';
import { searchHistoryStore } from '../stores/searchHistoryStore.js';
import { filterLibraryStore } from '../stores/filterLibraryStore.js';

afterEach(() => {
    cleanup();
    searchHistoryStore.set([]);
    filterLibraryStore.set([]);
});

function noop() {}

async function openSaveDialog() {
    searchHistoryStore.set([{ timestamp: 1700000000000, command: 't"blunder"', position: null, excludePosition: null }]);
    const utils = render(SearchPanel, {
        props: { onLoadPositionsByFilters: noop, onAddToFilterLibrary: vi.fn().mockResolvedValue(undefined) }
    });
    await tick();

    // Switch to the "history" sub-tab, where the bookmark action lives.
    const subTabButtons = utils.container.querySelectorAll('.sub-tab-btn');
    await fireEvent.click(subTabButtons[1]);
    await tick();

    const bookmarkBtn = utils.container.querySelector('.action-btn');
    expect(bookmarkBtn).not.toBeNull();
    await fireEvent.click(bookmarkBtn);
    await tick();

    return utils;
}

describe('SearchPanel — save-filter dialog accessibility', () => {
    test('opens as a labelled dialog with the name field auto-focused', async () => {
        const { container } = await openSaveDialog();

        const dialog = container.querySelector('[role="dialog"]');
        expect(dialog).not.toBeNull();
        expect(dialog.getAttribute('aria-modal')).toBe('true');
        expect(dialog.hasAttribute('aria-label')).toBe(true);

        const input = container.querySelector('#filterNameInput');
        expect(input).not.toBeNull();
        expect(document.activeElement).toBe(input);
    });

    test('Escape closes the dialog', async () => {
        const { container } = await openSaveDialog();

        const dialog = container.querySelector('[role="dialog"]');
        await fireEvent.keyDown(dialog, { key: 'Escape' });
        await tick();

        expect(container.querySelector('[role="dialog"]')).toBeNull();
    });
});
