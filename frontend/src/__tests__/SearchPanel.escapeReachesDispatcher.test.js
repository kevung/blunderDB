/**
 * SearchPanel.escapeReachesDispatcher.test.js
 *
 * #201 (D.1): while a field of the search form had the focus, the panel's
 * `document` keydown listener stopped EVERY key, so Escape never reached the
 * global dispatcher (App.svelte listens on `window`, one level up) and the
 * user was stuck in the field with no keyboard way out — the dispatcher is
 * what blurs the field on Escape.
 *
 * Locks the fix: Escape bubbles up to `window`; the bare keys still stay with
 * the field, and so does Tab until #204 settles its global meaning (today the
 * dispatcher would preventDefault it and break moving between fields).
 */

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
    field.focus();
    const reachedWindow = vi.fn();
    window.addEventListener('keydown', reachedWindow);
    return { field, reachedWindow, done: () => window.removeEventListener('keydown', reachedWindow) };
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
