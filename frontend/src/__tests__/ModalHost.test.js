/**
 * ModalHost.svelte owns the application-level modals App.svelte used to list
 * inline. Each modal keeps its own Escape / focus-trap handling; this test
 * mounts the host alone and, for every modal id, checks that opening it shows
 * one dialog and that Escape closes it — the wiring the move could have
 * broken (a lost onClose, a table id not matching its MODAL key).
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { tick } from 'svelte';

// ConfigModal subscribes to a Wails event at init and the other modals call
// bound Go methods on mount; the generated bindings resolve `window.go` /
// `window.runtime` at call time, so stub both rather than each module.
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: () => () => {},
    EventsOff: () => {},
    WindowSetTitle: () => {},
    Quit: () => {},
    ClipboardGetText: () => Promise.resolve('')
}));
// window.go.<ns>.<struct>.<Method>() → Promise<null>, whatever the depth.
function goStub() {
    return new Proxy(() => Promise.resolve(null), { get: (_t, key) => (key === 'then' ? undefined : goStub()) });
}

import ModalHost from '../components/ModalHost.svelte';
import { MODAL_TABLES } from '../components/modalTables.js';
import { activeModal, MODAL, openModal } from '../stores/uiStore.js';
import { confirmModalStore, confirmAction } from '../services/confirmService.js';

beforeEach(() => {
    window.go = goStub();
    // HelpModal fades in; jsdom has no Web Animations API.
    Element.prototype.animate ??= () => {
        const anim = { cancel() {}, finished: Promise.resolve(), onfinish: null };
        setTimeout(() => anim.onfinish?.(), 0);
        return anim;
    };
    activeModal.set(null);
});

afterEach(() => {
    cleanup();
    activeModal.set(null);
    confirmModalStore.set(null);
});

function dialogs(container) {
    return container.querySelectorAll('[role="dialog"]');
}

/** Send Escape where the modal listens: a focused field, else the overlay (bubbles to window). */
async function pressEscape(container) {
    const overlay = container.querySelector('[role="dialog"]');
    const focused = document.activeElement;
    const target = focused && overlay.contains(focused) ? focused : overlay.querySelector('input, button') || overlay;
    await fireEvent.keyDown(target, { key: 'Escape' });
    await tick();
}

// COMMAND is the status-bar command line, not a dialog; every other id opens
// one modal of the host.
const DIALOG_MODALS = Object.values(MODAL).filter((id) => id !== MODAL.COMMAND);

describe('ModalHost', () => {
    test('nothing is shown while no modal is active', () => {
        const { container } = render(ModalHost);
        expect(dialogs(container).length).toBe(0);
    });

    test.each(DIALOG_MODALS)('%s opens one dialog and Escape closes it', async (id) => {
        const { container } = render(ModalHost);
        openModal(id);
        await tick();
        expect(dialogs(container).length, `${id} should show exactly one dialog`).toBe(1);

        await pressEscape(container);
        expect(get(activeModal), `${id} should close on Escape`).toBeNull();
        // HelpModal fades out, so the element may outlive the state by a frame.
        await waitFor(() => expect(dialogs(container).length).toBe(0));
    });

    test('the confirm dialog is driven by confirmModalStore, not activeModal', async () => {
        const { container } = render(ModalHost);
        const answer = confirmAction('Sure?', { confirmLabel: 'Yes', cancelLabel: 'No' });
        await tick();
        expect(get(activeModal)).toBeNull();
        expect(dialogs(container).length).toBe(1);
        await pressEscape(container);
        expect(dialogs(container).length).toBe(0);
        await expect(answer).resolves.toBe(false);
    });

    test('every reference table renders the expected number of columns', async () => {
        const { container } = render(ModalHost);
        for (const [id, tables] of Object.entries(MODAL_TABLES)) {
            openModal(id);
            await tick();
            const sections = container.querySelectorAll('[role="dialog"] table');
            expect(sections.length, id).toBe(tables.length);
            sections.forEach((table, i) => {
                const headers = table.querySelectorAll('thead th');
                expect(headers.length, `${id} table ${i}`).toBe(tables[i].colCount + 1);
                expect(headers[1].textContent).toBe(String(tables[i].colOffset));
            });
            activeModal.set(null);
            await tick();
        }
    });
});
