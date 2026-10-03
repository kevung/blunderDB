/**
 * ConfigModal.svelte : les sept onglets se montent en colonne (aria-orientation) et chacun
 * ouvre son corps. Le débordement lui-même est géométrique (e2e) ; ce test tient le montage.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: () => () => {},
    EventsOff: () => {},
    WindowSetTitle: () => {},
    Quit: () => {},
    ClipboardGetText: () => Promise.resolve('')
}));
function goStub() {
    return new Proxy(() => Promise.resolve(null), { get: (_t, key) => (key === 'then' ? undefined : goStub()) });
}

import ConfigModal from '../components/ConfigModal.svelte';
import { get } from 'svelte/store';
import { pageStepStore, PAGE_STEPS, PAGE_STEP_DEFAULT } from '../stores/uiStore.js';

beforeEach(() => {
    window.go = goStub();
    Element.prototype.animate ??= () => ({ cancel() {}, finished: Promise.resolve(), onfinish: null });
});
afterEach(() => cleanup());

describe('ConfigModal tabs', () => {
    test('seven tabs in a vertical tablist, each selectable', async () => {
        const { container } = render(ConfigModal, { visible: true, onClose: () => {} });
        const list = container.ownerDocument.querySelector('[role="tablist"]');
        expect(list.getAttribute('aria-orientation')).toBe('vertical');
        const tabs = [...list.querySelectorAll('[role="tab"]')];
        expect(tabs).toHaveLength(7);
        expect(tabs[0].getAttribute('aria-selected')).toBe('true');
        await fireEvent.click(tabs[6]);
        expect(tabs[6].getAttribute('aria-selected')).toBe('true');
        expect(tabs[0].getAttribute('aria-selected')).toBe('false');
    });

    test('the Interface tab offers every PageUp / PageDown step, 10 % of the list included', async () => {
        const { container } = render(ConfigModal, { visible: true, onClose: () => {} });
        const select = /** @type {HTMLSelectElement} */ (container.ownerDocument.getElementById('config-page-step'));
        expect([...select.options].map((o) => o.value)).toEqual([...PAGE_STEPS]);
        expect(select.value).toBe(PAGE_STEP_DEFAULT);
        await fireEvent.change(select, { target: { value: '10%' } });
        expect(get(pageStepStore)).toBe('10%');
        pageStepStore.set(PAGE_STEP_DEFAULT);
    });
});
