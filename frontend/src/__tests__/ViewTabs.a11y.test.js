/**
 * ViewTabs.a11y.test.js
 *
 * A view tab was a bare <div> with onclick/ondblclick: no role, no focus, no
 * key (two Svelte a11y warnings, one of them silenced). It is now a
 * role="tab" inside a role="tablist", focusable, and Enter switches to it.
 * This locks the mouse gestures it already had — click switches, double-click
 * renames, Enter in the rename field commits without switching — and the one
 * key it gained. Only Enter: Space stays the global command-line key.
 */

import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

// Filled by the mock factory below; typed loosely, as a test double.
const { viewStoreMock } = vi.hoisted(() => ({ viewStoreMock: /** @type {any} */ ({}) }));

vi.mock('../stores/viewStore', async () => {
    const { writable } = await import('svelte/store');
    Object.assign(viewStoreMock, {
        views: writable([]),
        activeViewId: writable(1),
        switchTo: vi.fn(),
        addView: vi.fn(),
        closeView: vi.fn(),
        renameView: vi.fn()
    });
    return { viewStore: viewStoreMock };
});

import ViewTabs from '../components/ViewTabs.svelte';

/** @param {HTMLElement} container */
function tabs(container) {
    return /** @type {HTMLElement[]} */ ([...container.querySelectorAll('[role="tab"]')]);
}

beforeEach(() => {
    vi.clearAllMocks();
    viewStoreMock.views.set([
        { id: 1, name: '#1' },
        { id: 2, name: '#2' }
    ]);
    viewStoreMock.activeViewId.set(1);
});

afterEach(() => {
    cleanup();
});

describe('ViewTabs — roles', () => {
    test('the tabs sit in a tablist, aria-selected follows the active view', () => {
        const { container } = render(ViewTabs);
        expect(container.querySelector('[role="tablist"]')).not.toBeNull();
        const [first, second] = tabs(container);
        expect(first.getAttribute('aria-selected')).toBe('true');
        expect(second.getAttribute('aria-selected')).toBe('false');
        expect(second.tabIndex).toBe(0);
    });
});

describe('ViewTabs — gestures', () => {
    test('a click switches to the view', async () => {
        const { container } = render(ViewTabs);
        await fireEvent.click(tabs(container)[1]);
        expect(viewStoreMock.switchTo).toHaveBeenCalledWith(2);
    });

    test('Enter on a focused tab switches to the view', async () => {
        const { container } = render(ViewTabs);
        await fireEvent.keyDown(tabs(container)[1], { key: 'Enter' });
        expect(viewStoreMock.switchTo).toHaveBeenCalledWith(2);
    });

    test('Space on a tab is left to the global command line', async () => {
        const { container } = render(ViewTabs);
        await fireEvent.keyDown(tabs(container)[1], { key: ' ', code: 'Space' });
        expect(viewStoreMock.switchTo).not.toHaveBeenCalled();
    });

    test('a double-click renames, and Enter in the field commits without switching', async () => {
        const { container } = render(ViewTabs);
        await fireEvent.dblClick(tabs(container)[1]);
        await tick();
        const input = /** @type {HTMLInputElement} */ (container.querySelector('.rename-input'));
        expect(input).not.toBeNull();
        await fireEvent.input(input, { target: { value: 'Ouvertures' } });
        await fireEvent.keyDown(input, { key: 'Enter' });
        expect(viewStoreMock.renameView).toHaveBeenCalledWith(2, 'Ouvertures');
        expect(viewStoreMock.switchTo).not.toHaveBeenCalled();
    });

    test('the close button closes without switching', async () => {
        const { container } = render(ViewTabs);
        await fireEvent.click(/** @type {HTMLElement} */ (tabs(container)[1].querySelector('.close-btn')));
        expect(viewStoreMock.closeView).toHaveBeenCalledWith(2);
        expect(viewStoreMock.switchTo).not.toHaveBeenCalled();
    });
});
