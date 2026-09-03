/**
 * autofocus.test.js — #205
 *
 * A Svelte action replacing the HTML `autofocus` attribute (a11y_autofocus):
 * every call site mounts its element fresh through an `{#if editing}` block,
 * so the action only needs to call `.focus()` once, on mount.
 */
import { describe, test, expect } from 'vitest';
import { autofocus } from '../utils/autofocus.js';

describe('autofocus action', () => {
    test('focuses the node it is applied to', () => {
        const input = document.createElement('input');
        document.body.appendChild(input);
        expect(document.activeElement).not.toBe(input);

        autofocus(input);

        expect(document.activeElement).toBe(input);
        input.remove();
    });
});
