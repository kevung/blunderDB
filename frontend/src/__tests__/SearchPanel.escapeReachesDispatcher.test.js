/**
 * SearchPanel.escapeReachesDispatcher.test.js
 *
 * While a field of the search form has the focus, the panel keeps the bare keys and Tab
 * (moving between fields) but lets Escape through to the global dispatcher, which blurs the
 * field: without it the user would be stuck in the field with no keyboard way out.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveSearchHistory: vi.fn(() => Promise.resolve()),
    LoadSearchHistory: vi.fn(() => Promise.resolve([])),
    DeleteSearchHistoryEntry: vi.fn(() => Promise.resolve()),
    LoadFilters: vi.fn(() => Promise.resolve([])),
    DeleteFilter: vi.fn(() => Promise.resolve()),
    LoadEditPosition: vi.fn(() => Promise.resolve(null)),
    LoadExcludePosition: vi.fn(() => Promise.resolve(null))
}));

import SearchPanel from '../components/SearchPanel.svelte';
import { registerKeys } from '../services/keyDispatch.js';
import { activeTabStore } from '../stores/uiStore.js';

afterEach(() => {
    cleanup();
    activeTabStore.set('matches');
});

async function mountWithFocusedField() {
    activeTabStore.set('search');
    const { container } = render(SearchPanel, { props: { onLoadPositionsByFilters: () => {}, onAddToFilterLibrary: () => {} } });
    await tick();
    const field = container.querySelector('.filter-checkbox input');
    expect(field).not.toBeNull();
    /** @type {HTMLElement} */ (must(field)).focus();
    const reachedWindow = vi.fn();
    const done = registerKeys('global', reachedWindow);
    return { field, reachedWindow, done };
}

describe('SearchPanel — keys from a focused field', () => {
    test('Escape reaches the global dispatcher on window', async () => {
        const { field, reachedWindow, done } = await mountWithFocusedField();
        await fireEvent.keyDown(field, { key: 'Escape' });
        expect(reachedWindow).toHaveBeenCalledTimes(1);
        expect(reachedWindow.mock.calls[0][0].key).toBe('Escape');
        done();
    });

    test('bare keys and Tab stay with the field', async () => {
        const { field, reachedWindow, done } = await mountWithFocusedField();
        await fireEvent.keyDown(field, { key: 'j' });
        await fireEvent.keyDown(field, { key: 'ArrowRight' });
        await fireEvent.keyDown(field, { key: 'Tab', code: 'Tab' });
        expect(reachedWindow).not.toHaveBeenCalled();
        done();
    });

    test('outside the search tab the panel does not intercept anything', async () => {
        const { field, reachedWindow, done } = await mountWithFocusedField();
        activeTabStore.set('analysis');
        await tick();
        await fireEvent.keyDown(field, { key: 'j' });
        expect(reachedWindow).toHaveBeenCalledTimes(1);
        done();
    });
});
