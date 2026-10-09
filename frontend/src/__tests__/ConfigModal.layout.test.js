/**
 * ConfigModal.svelte : les neuf onglets se montent en colonne (aria-orientation) et chacun
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
    test('nine tabs in a vertical tablist, each selectable', async () => {
        const { container } = render(ConfigModal, { visible: true, onClose: () => {} });
        const list = container.ownerDocument.querySelector('[role="tablist"]');
        expect(list.getAttribute('aria-orientation')).toBe('vertical');
        const tabs = [...list.querySelectorAll('[role="tab"]')];
        expect(tabs).toHaveLength(9);
        expect(tabs[0].getAttribute('aria-selected')).toBe('true');
        await fireEvent.click(tabs[7]);
        expect(tabs[7].getAttribute('aria-selected')).toBe('true');
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

describe('ConfigModal scroll', () => {
    test('each tab keeps its own scroll offset; a tab never seen starts at the top', async () => {
        const store = new WeakMap();
        const desc = Object.getOwnPropertyDescriptor(Element.prototype, 'scrollTop');
        Object.defineProperty(Element.prototype, 'scrollTop', {
            configurable: true,
            get() {
                return store.get(this) ?? 0;
            },
            set(v) {
                store.set(this, v);
            }
        });
        try {
            const { container } = render(ConfigModal, { visible: true, onClose: () => {} });
            const doc = container.ownerDocument;
            const tabs = [...doc.querySelectorAll('[role="tab"]')];
            const body = doc.querySelector('.tab-body');
            body.scrollTop = 120;
            await fireEvent.click(tabs[1]);
            await Promise.resolve();
            expect(body.scrollTop).toBe(0);
            body.scrollTop = 40;
            await fireEvent.click(tabs[0]);
            await Promise.resolve();
            expect(body.scrollTop).toBe(120);
            await fireEvent.click(tabs[1]);
            await Promise.resolve();
            expect(body.scrollTop).toBe(40);
        } finally {
            if (desc) Object.defineProperty(Element.prototype, 'scrollTop', desc);
            else delete Element.prototype.scrollTop;
        }
    });
});
