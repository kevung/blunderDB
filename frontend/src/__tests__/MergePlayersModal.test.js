/**
 * MergePlayersModal.test.js
 *
 * The merge dialog was the one role="dialog" without the focus trap the other
 * modals use (now Modal.svelte's): Tab walked out of it into the match panel
 * behind. These tests pin the trap — focus lands inside on mount, Tab wraps at
 * the edges — and the dialog's contract with the binding: the names ticked in
 * the list and the canonical name typed in reach MergePlayers as-is, then the
 * parent is told to refresh and close.
 *
 * The selection is a SvelteSet mutated in place. It used to be re-assigned to a
 * copy on every click, which the template — tracking the original instance —
 * never saw: the merge button stayed disabled whatever was ticked. The payload
 * test fails against that version.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const MergePlayers = vi.fn(() => Promise.resolve());
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn(() =>
        Promise.resolve([
            { Name: 'Alice', Count: 3 },
            { Name: 'alice', Count: 1 },
            { Name: 'Bob', Count: 2 }
        ])
    ),
    MergePlayers: (...args) => MergePlayers(...args)
}));

import MergePlayersModal from '../components/MergePlayersModal.svelte';

let onClose;
let onMerged;
let outside;

async function mount() {
    const result = render(MergePlayersModal, { props: { onClose, onMerged } });
    // GetAllPlayerNames resolves on a microtask; two ticks let the list render.
    await tick();
    await tick();
    return result;
}

function tab(target, shiftKey = false) {
    const event = new KeyboardEvent('keydown', { key: 'Tab', shiftKey, bubbles: true, cancelable: true });
    target.dispatchEvent(event);
    return event;
}

beforeEach(() => {
    onClose = vi.fn();
    onMerged = vi.fn();
    MergePlayers.mockClear();
    // Something focusable outside the dialog — where Tab used to escape to.
    outside = document.createElement('button');
    outside.id = 'outside';
    document.body.appendChild(outside);
    outside.focus();
});

afterEach(() => {
    cleanup();
    outside.remove();
});

describe('MergePlayersModal — focus', () => {
    test('mounts as a dialog and takes the focus away from the page behind', async () => {
        const { container } = await mount();

        const dialog = container.querySelector('[role="dialog"]');
        expect(dialog).not.toBeNull();
        expect(dialog.getAttribute('aria-modal')).toBe('true');
        expect(dialog.contains(document.activeElement)).toBe(true);
    });

    test('Tab is trapped: it wraps from the last control to the first and back', async () => {
        const { container } = await mount();
        const dialog = container.querySelector('[role="dialog"]');

        // The filter field opens the dialog; the close cross, last in the DOM, ends it.
        const first = dialog.querySelector('.filter-input');
        const last = dialog.querySelector('.modal-close');
        expect(document.activeElement).toBe(first);

        last.focus();
        const forward = tab(last);
        expect(forward.defaultPrevented).toBe(true);
        expect(document.activeElement).toBe(first);

        const backward = tab(first, true);
        expect(backward.defaultPrevented).toBe(true);
        expect(document.activeElement).toBe(last);
        expect(dialog.contains(document.activeElement)).toBe(true);
    });

    test('closing gives the focus back to what had it before', async () => {
        const { unmount } = await mount();
        expect(document.activeElement).not.toBe(outside);
        unmount();
        expect(document.activeElement).toBe(outside);
    });
});

describe('MergePlayersModal — merging', () => {
    test('the ticked names and the canonical name reach the binding, then the parent refreshes and closes', async () => {
        const { container } = await mount();

        const rows = [...container.querySelectorAll('.player-row')];
        expect(rows.map((r) => r.querySelector('.player-name').textContent)).toEqual(['Alice', 'alice', 'Bob']);

        const mergeButton = container.querySelector('.btn-merge');
        expect(mergeButton.disabled).toBe(true);

        await fireEvent.click(rows[0]);
        await fireEvent.click(rows[1]);
        await tick();

        // The first ticked name is proposed as the one to keep.
        const canonical = container.querySelector('#canonical-input');
        expect(canonical.value).toBe('Alice');
        expect(mergeButton.disabled).toBe(false);
        expect(rows[0].classList.contains('selected')).toBe(true);
        expect(rows[1].classList.contains('selected')).toBe(true);
        expect(rows[2].classList.contains('selected')).toBe(false);

        await fireEvent.click(mergeButton);
        await tick();
        await tick();

        expect(MergePlayers).toHaveBeenCalledTimes(1);
        expect(MergePlayers).toHaveBeenCalledWith(['Alice', 'alice'], 'Alice');
        expect(onMerged).toHaveBeenCalledTimes(1);
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    test('unticking a name takes it out of the payload', async () => {
        const { container } = await mount();
        const rows = [...container.querySelectorAll('.player-row')];

        await fireEvent.click(rows[0]);
        await fireEvent.click(rows[1]);
        await fireEvent.click(rows[2]);
        await fireEvent.click(rows[1]); // untick 'alice'
        await tick();

        await fireEvent.click(container.querySelector('.btn-merge'));
        await tick();
        await tick();

        expect(MergePlayers).toHaveBeenCalledWith(['Alice', 'Bob'], 'Alice');
    });

    test('a canonical name typed by the user wins over the proposed one', async () => {
        const { container } = await mount();
        const rows = [...container.querySelectorAll('.player-row')];

        await fireEvent.click(rows[0]);
        await fireEvent.click(rows[1]);
        const canonical = container.querySelector('#canonical-input');
        await fireEvent.input(canonical, { target: { value: 'Alice Martin' } });
        await tick();

        await fireEvent.click(container.querySelector('.btn-merge'));
        await tick();
        await tick();

        expect(MergePlayers).toHaveBeenCalledWith(['Alice', 'alice'], 'Alice Martin');
    });

    test('Escape closes without merging', async () => {
        const { container } = await mount();
        const dialog = container.querySelector('[role="dialog"]');

        await fireEvent.keyDown(dialog, { key: 'Escape' });

        expect(onClose).toHaveBeenCalledTimes(1);
        expect(MergePlayers).not.toHaveBeenCalled();
        expect(onMerged).not.toHaveBeenCalled();
    });
});
