/**
 * focusTrap.js is a plain Svelte action (no component): it focuses the first
 * focusable element inside the node on mount, wraps Tab/Shift+Tab at the
 * edges, and restores focus to whatever was focused before on destroy. These
 * tests exercise it directly against a jsdom DOM tree, no component render
 * needed.
 */
import { describe, test, expect, afterEach } from 'vitest';
import { trapFocus } from '../utils/focusTrap.js';

function keydown(target, key, opts = {}) {
    const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...opts });
    target.dispatchEvent(event);
    return event;
}

let container;

afterEach(() => {
    if (container) {
        container.remove();
        container = null;
    }
});

describe('trapFocus', () => {
    test('focuses the first focusable element on mount', () => {
        container = document.createElement('div');
        container.innerHTML = `
            <span>not focusable</span>
            <button id="first">First</button>
            <button id="second">Second</button>
        `;
        document.body.appendChild(container);

        trapFocus(container);

        expect(document.activeElement.id).toBe('first');
    });

    test('does nothing on mount when there is no focusable element', () => {
        container = document.createElement('div');
        container.innerHTML = `<span>nothing focusable here</span>`;
        document.body.appendChild(container);

        const previousActive = document.activeElement;
        trapFocus(container);

        expect(document.activeElement).toBe(previousActive);
    });

    test('Tab on the last element wraps focus to the first', () => {
        container = document.createElement('div');
        container.innerHTML = `
            <button id="first">First</button>
            <button id="middle">Middle</button>
            <button id="last">Last</button>
        `;
        document.body.appendChild(container);
        trapFocus(container);

        document.getElementById('last').focus();
        const event = keydown(container, 'Tab');

        expect(document.activeElement.id).toBe('first');
        expect(event.defaultPrevented).toBe(true);
    });

    test('Shift+Tab on the first element wraps focus to the last', () => {
        container = document.createElement('div');
        container.innerHTML = `
            <button id="first">First</button>
            <button id="middle">Middle</button>
            <button id="last">Last</button>
        `;
        document.body.appendChild(container);
        trapFocus(container);

        document.getElementById('first').focus();
        const event = keydown(container, 'Tab', { shiftKey: true });

        expect(document.activeElement.id).toBe('last');
        expect(event.defaultPrevented).toBe(true);
    });

    test('Tab in the middle of the trap is left alone (no wrap)', () => {
        container = document.createElement('div');
        container.innerHTML = `
            <button id="first">First</button>
            <button id="middle">Middle</button>
            <button id="last">Last</button>
        `;
        document.body.appendChild(container);
        trapFocus(container);

        document.getElementById('middle').focus();
        const event = keydown(container, 'Tab');

        // Only the browser's native focus advancement would move focus here
        // (jsdom doesn't implement that); what matters is the handler does
        // NOT hijack this key.
        expect(event.defaultPrevented).toBe(false);
    });

    test('non-Tab keys are ignored', () => {
        container = document.createElement('div');
        container.innerHTML = `<button id="first">First</button><button id="last">Last</button>`;
        document.body.appendChild(container);
        trapFocus(container);

        document.getElementById('last').focus();
        const event = keydown(container, 'Enter');

        expect(event.defaultPrevented).toBe(false);
        expect(document.activeElement.id).toBe('last');
    });

    test('disabled inputs and negative tabindex are excluded from the focusable set', () => {
        container = document.createElement('div');
        container.innerHTML = `
            <button id="first">First</button>
            <input id="disabled-input" disabled />
            <div id="not-focusable" tabindex="-1">skip me</div>
            <button id="last">Last</button>
        `;
        document.body.appendChild(container);
        trapFocus(container);

        document.getElementById('last').focus();
        keydown(container, 'Tab');

        expect(document.activeElement.id).toBe('first');
    });

    test('destroy restores focus to the previously focused element', () => {
        const outsideButton = document.createElement('button');
        outsideButton.id = 'outside';
        document.body.appendChild(outsideButton);
        outsideButton.focus();
        expect(document.activeElement.id).toBe('outside');

        container = document.createElement('div');
        container.innerHTML = `<button id="inside">Inside</button>`;
        document.body.appendChild(container);

        const action = trapFocus(container);
        expect(document.activeElement.id).toBe('inside');

        action.destroy();
        expect(document.activeElement.id).toBe('outside');

        outsideButton.remove();
    });

    test('destroy removes the keydown listener', () => {
        container = document.createElement('div');
        container.innerHTML = `<button id="first">First</button><button id="last">Last</button>`;
        document.body.appendChild(container);

        const action = trapFocus(container);
        action.destroy();

        document.getElementById('last').focus();
        const event = keydown(container, 'Tab');

        // After destroy, the trap must no longer intercept Tab.
        expect(event.defaultPrevented).toBe(false);
    });
});
