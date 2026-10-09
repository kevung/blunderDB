/**
 * Search in the help: accent- and case-insensitive, occurrences counted in reading order,
 * the current one shown as the selection; "/" reaches the field without closing the help.
 */
import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database', () => ({ GetDatabaseVersion: vi.fn(() => Promise.resolve('2.0.0')) }));

import { findInHelp, createHelpIndex } from '../utils/helpSearch.js';
import HelpModal from '../components/HelpModal.svelte';

afterEach(cleanup);

test('findInHelp ignores case and accents and walks every text node', () => {
    const root = document.createElement('div');
    root.innerHTML = '<p>Raccourcis clavier</p><ul><li>un raccourci</li><li>RACCOURCI</li></ul><p>rien</p>';
    const found = findInHelp(root, 'raccourci');
    expect(found).toHaveLength(3);
    expect(found[0].toString()).toBe('Raccourci');
    expect(findInHelp(root, 'ÉCOURC')).toHaveLength(0);
    expect(findInHelp(root, 'clavier')).toHaveLength(1);
    expect(findInHelp(root, '   ')).toEqual([]);
});

test('the search field counts the occurrences, Enter steps through them, "/" focuses the field', async () => {
    const onClose = vi.fn();
    render(HelpModal, { visible: true, onClose });
    await vi.waitFor(() => expect(screen.queryByTestId('help-loading')).toBeNull());

    const input = screen.getByTestId('help-search');
    await fireEvent.keyDown(document.activeElement || document.body, { key: '/' });
    await tick();
    expect(document.activeElement).toBe(input);

    await fireEvent.input(input, { target: { value: 'position' } });
    await vi.waitFor(() => expect(screen.getByTestId('help-search-count').textContent).toMatch(/^0 \/ [1-9]\d*$/));
    expect(window.getSelection().toString()).toBe('');

    await fireEvent.keyDown(input, { key: 'Enter' });
    await tick();
    expect(screen.getByTestId('help-search-count').textContent).toMatch(/^1 \/ \d+$/);
    await fireEvent.keyDown(input, { key: 'Enter' });
    await tick();
    expect(screen.getByTestId('help-search-count').textContent).toMatch(/^2 \/ \d+$/);
    expect(window.getSelection().toString().toLowerCase()).toBe('position');
    expect(onClose).not.toHaveBeenCalled();

    await fireEvent.input(input, { target: { value: 'zzzzqqq' } });
    await vi.waitFor(() => expect(screen.getByTestId('help-search-count').textContent).toBe('no result'));
});

test('typing alone refreshes count and highlights after a short debounce, no Enter needed', async () => {
    const store = new Map();
    vi.stubGlobal(
        'Highlight',
        class {
            constructor(...r) {
                this.ranges = r;
            }
        }
    );
    Object.defineProperty(CSS, 'highlights', { value: store, configurable: true, writable: true });
    render(HelpModal, { visible: true, onClose: vi.fn() });
    await vi.waitFor(() => expect(screen.queryByTestId('help-loading')).toBeNull());
    const input = screen.getByTestId('help-search');

    await fireEvent.input(input, { target: { value: 'pos' } });
    expect(store.has('help-search')).toBe(false); // debounced, not per synchronous event
    await vi.waitFor(() => expect(store.get('help-search')?.ranges.length).toBeGreaterThan(0));
    const shortCount = store.get('help-search').ranges.length;

    await fireEvent.input(input, { target: { value: 'POSITIONS' } }); // case-insensitive, narrower
    await vi.waitFor(() => expect(store.get('help-search')?.ranges.length).toBeLessThan(shortCount));

    await fireEvent.input(input, { target: { value: '' } });
    await vi.waitFor(() => expect(store.has('help-search')).toBe(false));
    expect(screen.queryByTestId('help-search-count')).toBeNull();
    vi.unstubAllGlobals();
    delete CSS.highlights;
});

test('Escape in the search field is left to the modal', async () => {
    const onClose = vi.fn();
    render(HelpModal, { visible: true, onClose });
    await vi.waitFor(() => expect(screen.queryByTestId('help-loading')).toBeNull());
    const input = screen.getByTestId('help-search');
    input.focus();
    await fireEvent.keyDown(input, { key: 'Escape' });
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled());
});

test('the index walks the text once; a query counts every hit but builds only `limit` ranges', () => {
    const root = document.createElement('div');
    root.innerHTML = '<p>aaa</p><p>Éa</p>';
    const walk = vi.spyOn(document, 'createTreeWalker');
    const range = vi.spyOn(document, 'createRange');
    const index = createHelpIndex(root);
    expect(walk).toHaveBeenCalledTimes(1);
    const first = index.search('a', 2);
    index.search('aa');
    index.search('é');
    expect(walk).toHaveBeenCalledTimes(1);
    expect(first.count).toBe(4);
    expect(first.ranges).toHaveLength(2);
    expect(index.search('a', 2).at(3).startContainer.nodeValue).toBe('Éa');
    walk.mockRestore();
    range.mockRestore();
});
