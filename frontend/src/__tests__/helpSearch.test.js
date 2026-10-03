/**
 * Search in the help: accent- and case-insensitive, occurrences counted in reading order,
 * the current one shown as the selection; "/" reaches the field without closing the help.
 */
import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database', () => ({ GetDatabaseVersion: vi.fn(() => Promise.resolve('2.0.0')) }));

import { findInHelp } from '../utils/helpSearch.js';
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
    await tick();
    const count = screen.getByTestId('help-search-count').textContent;
    expect(count).toMatch(/^0 \/ \d+$/);
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
    await tick();
    expect(screen.getByTestId('help-search-count').textContent).toBe('no result');
});
