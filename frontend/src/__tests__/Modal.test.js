/**
 * Modal.test.js
 *
 * Modal.svelte is the overlay + box every modal is built on. These tests pin the
 * behaviour the thirteen modals used to each implement for themselves: the focus
 * lands inside on open and goes back where it was on close, Tab is trapped,
 * Escape closes through one handler that stops the event dead (the global
 * dispatcher must never see it), a click on the backdrop closes only when asked,
 * and the dialog is named by its title.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

import ModalFixture from './fixtures/ModalFixture.svelte';

let outside;

beforeEach(() => {
    outside = document.createElement('button');
    outside.id = 'outside';
    document.body.appendChild(outside);
    outside.focus();
});

afterEach(() => {
    cleanup();
    outside.remove();
});

function tab(target, shiftKey = false) {
    const event = new KeyboardEvent('keydown', { key: 'Tab', shiftKey, bubbles: true, cancelable: true });
    target.dispatchEvent(event);
    return event;
}

describe('Modal — focus', () => {
    test('is a dialog named by its title, and takes the focus on its first control', async () => {
        const { container } = render(ModalFixture, { props: { open: true } });
        await tick();

        const dialog = container.querySelector('[role="dialog"]');
        expect(dialog.getAttribute('aria-modal')).toBe('true');
        const title = container.querySelector('.modal-title');
        expect(dialog.getAttribute('aria-labelledby')).toBe(title.id);
        expect(title.textContent).toBe('A title');
        expect(document.activeElement).toBe(container.querySelector('#first'));
    });

    test('falls back to aria-label when there is no title', () => {
        const { container } = render(ModalFixture, { props: { open: true, withTitle: false, label: 'Plain' } });
        const dialog = container.querySelector('[role="dialog"]');
        expect(dialog.getAttribute('aria-label')).toBe('Plain');
        expect(dialog.hasAttribute('aria-labelledby')).toBe(false);
    });

    test('Tab wraps inside the box, the close cross last', async () => {
        const { container } = render(ModalFixture, { props: { open: true } });
        await tick();
        const first = container.querySelector('#first');
        const cross = container.querySelector('.modal-close');

        cross.focus();
        expect(tab(cross).defaultPrevented).toBe(true);
        expect(document.activeElement).toBe(first);

        expect(tab(first, true).defaultPrevented).toBe(true);
        expect(document.activeElement).toBe(cross);
    });

    test('the dialog itself takes the focus when it holds no control', () => {
        const { container } = render(ModalFixture, { props: { open: true, withControls: false, closeButton: false } });
        expect(document.activeElement).toBe(container.querySelector('[role="dialog"]'));
    });

    test('closing gives the focus back to what had it before', async () => {
        const { rerender } = render(ModalFixture, { props: { open: true } });
        expect(document.activeElement).not.toBe(outside);
        await rerender({ open: false });
        expect(document.activeElement).toBe(outside);
    });
});

describe('Modal — closing', () => {
    test('Escape calls onclose once and stops the event before window', async () => {
        const onclose = vi.fn();
        const seenByWindow = vi.fn();
        window.addEventListener('keydown', seenByWindow);
        const { container } = render(ModalFixture, { props: { open: true, onclose } });

        await fireEvent.keyDown(container.querySelector('#first'), { key: 'Escape' });
        expect(onclose).toHaveBeenCalledTimes(1);
        expect(seenByWindow).not.toHaveBeenCalled();
        window.removeEventListener('keydown', seenByWindow);
    });

    test('Escape does nothing when closeOnEscape is off', async () => {
        const onclose = vi.fn();
        const { container } = render(ModalFixture, { props: { open: true, onclose, closeOnEscape: false } });
        await fireEvent.keyDown(container.querySelector('[role="dialog"]'), { key: 'Escape' });
        expect(onclose).not.toHaveBeenCalled();
    });

    test('other keys reach the modal through onkeydown', async () => {
        const onkeydown = vi.fn();
        const { container } = render(ModalFixture, { props: { open: true, onkeydown } });
        await fireEvent.keyDown(container.querySelector('#first'), { key: 'Enter' });
        expect(onkeydown).toHaveBeenCalledTimes(1);
        expect(onkeydown.mock.calls[0][0].key).toBe('Enter');
    });

    test('the close cross calls onclose', async () => {
        const onclose = vi.fn();
        const { container } = render(ModalFixture, { props: { open: true, onclose } });
        await fireEvent.click(container.querySelector('.modal-close'));
        expect(onclose).toHaveBeenCalledTimes(1);
    });

    test('a click on the backdrop closes only with closeOnOverlay, never a click in the box', async () => {
        const onclose = vi.fn();
        const { container, rerender } = render(ModalFixture, { props: { open: true, onclose } });

        await fireEvent.click(container.querySelector('.modal-scroll'));
        expect(onclose).not.toHaveBeenCalled();

        await rerender({ open: true, onclose, closeOnOverlay: true });
        await fireEvent.click(container.querySelector('.modal-box'));
        await fireEvent.click(container.querySelector('#first'));
        expect(onclose).not.toHaveBeenCalled();
        await fireEvent.click(container.querySelector('.modal-scroll'));
        expect(onclose).toHaveBeenCalledTimes(1);
    });

    test('renders nothing while closed', () => {
        const { container } = render(ModalFixture, { props: { open: false } });
        expect(container.querySelector('[role="dialog"]')).toBeNull();
    });
});
