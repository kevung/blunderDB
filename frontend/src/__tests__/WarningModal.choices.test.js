/**
 * WarningModal.choices.test.js
 *
 * A confirmation with several answers: Enter answers with the focused button,
 * and the primary one holds the focus on open.
 */
import { must } from './helpers/must.js';
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import WarningModal from '../components/WarningModal.svelte';

const CHOICES = [
    { value: 'open', label: 'Open' },
    { value: 'merge', label: 'Merge', primary: true }
];

function mount() {
    const onChoose = vi.fn();
    const onClose = vi.fn();
    const { container } = render(WarningModal, { visible: true, mode: 'confirm', message: 'Drop', choices: CHOICES, onChoose, onClose });
    return { container, onChoose, onClose };
}

describe('WarningModal with choices — Enter', () => {
    afterEach(() => cleanup());

    test('answers with the primary choice, focused on open', async () => {
        const { container, onChoose } = mount();
        const primary = container.querySelector('[data-testid="choice-merge"]');
        await vi.waitFor(() => expect(document.activeElement).toBe(primary));
        await fireEvent.keyDown(document.activeElement, { key: 'Enter' });
        expect(onChoose).toHaveBeenCalledExactlyOnceWith('merge');
    });

    test('on Cancel, cancels', async () => {
        const { container, onChoose, onClose } = mount();
        const cancel = [...container.querySelectorAll('.modal-footer button')].find((b) => !b.dataset.testid);
        /** @type {HTMLElement} */ (must(cancel)).focus();
        await fireEvent.keyDown(cancel, { key: 'Enter' });
        expect(onClose).toHaveBeenCalledOnce();
        expect(onChoose).not.toHaveBeenCalled();
    });

    test('on another choice, answers with it', async () => {
        const { container, onChoose } = mount();
        const open = container.querySelector('[data-testid="choice-open"]');
        /** @type {HTMLElement} */ (must(open)).focus();
        await fireEvent.keyDown(open, { key: 'Enter' });
        expect(onChoose).toHaveBeenCalledExactlyOnceWith('open');
    });
});
