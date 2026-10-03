/**
 * Home / End on a tab move the tab focus and stop there: they must not also reach the global
 * handler, where they jump to the first / last position.
 */
import { test, expect, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import TabbedPanel from '../components/TabbedPanel.svelte';

afterEach(cleanup);

test('Home and End on a tab do not bubble', async () => {
    const noop = () => {};
    const { container } = render(TabbedPanel, { props: { onLoadPositionsByFilters: noop, onCloseAnalysis: noop, onCloseComment: noop, onOpenCollection: noop, onAddToFilterLibrary: noop } });
    let reached = 0;
    const spy = () => reached++;
    document.addEventListener('keydown', spy);
    const tab = /** @type {HTMLElement} */ (container.querySelector('[role="tab"]'));
    await fireEvent.keyDown(tab, { key: 'End' });
    await fireEvent.keyDown(tab, { key: 'Home' });
    document.removeEventListener('keydown', spy);
    expect(reached).toBe(0);
});
