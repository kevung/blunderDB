/**
 * ConfigModal.svelte : les huit onglets se montent en colonne (aria-orientation) et chacun
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

beforeEach(() => {
    window.go = goStub();
    Element.prototype.animate ??= () => ({ cancel() {}, finished: Promise.resolve(), onfinish: null });
});
afterEach(() => cleanup());

describe('ConfigModal tabs', () => {
    test('eight tabs in a vertical tablist, each selectable', async () => {
        const { container } = render(ConfigModal, { visible: true, onClose: () => {} });
        const list = container.ownerDocument.querySelector('[role="tablist"]');
        expect(list.getAttribute('aria-orientation')).toBe('vertical');
        const tabs = [...list.querySelectorAll('[role="tab"]')];
        expect(tabs).toHaveLength(8);
        expect(tabs[0].getAttribute('aria-selected')).toBe('true');
        await fireEvent.click(tabs[6]);
        expect(tabs[6].getAttribute('aria-selected')).toBe('true');
        expect(tabs[0].getAttribute('aria-selected')).toBe('false');
    });
});
